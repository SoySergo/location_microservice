package usecase

import (
	"context"
	"sync"

	"github.com/location-microservice/internal/pkg/errors"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

const (
	// DefaultTransportRadius is the default search radius for transport stations in meters
	DefaultTransportRadius = 1500 // 1.5 km
	// DefaultTransportLimit is the default number of transport stations per point
	DefaultTransportLimit = 5
)

// Ensure EnrichedLocationUseCase implements BatchLocationEnricher interface
var _ BatchLocationEnricher = (*EnrichedLocationUseCase)(nil)

// EnrichedLocationUseCase - usecase для полного обогащения локаций
type EnrichedLocationUseCase struct {
	searchUC    *SearchUseCase    // для DetectLocationBatch
	transportUC *TransportUseCase // для GetNearestTransportByPriorityBatch
	logger      *zap.Logger
}

// NewEnrichedLocationUseCase создает новый EnrichedLocationUseCase
func NewEnrichedLocationUseCase(
	searchUC *SearchUseCase,
	transportUC *TransportUseCase,
	logger *zap.Logger,
) *EnrichedLocationUseCase {
	return &EnrichedLocationUseCase{
		searchUC:    searchUC,
		transportUC: transportUC,
		logger:      logger,
	}
}

// EnrichLocationBatch обогащает пачку локаций параллельно.
//
// Архитектура параллельной обработки:
// - Горутина 1: DetectLocationBatch для ВСЕХ локаций → возвращает enriched location с ID границ
// - Горутина 2: GetNearestTransportByPriorityBatch только для IsVisible=true → возвращает транспорт
// - Обе горутины работают параллельно с sync.WaitGroup
// - Результаты объединяются после завершения обеих горутин
//
// Поведение при ошибках:
// - Ошибка DetectLocationBatch прерывает весь процесс (критическая)
// - Ошибка транспорта не прерывает обработку, результаты возвращаются без транспорта (graceful degradation)
//
// Разделение локаций:
// - Visible: IsVisible=true И есть координаты (Latitude, Longitude) → получают транспорт
// - Non-visible: остальные → обогащаются только по границам
//
// Возвращает EnrichLocationBatchResponse с полными метаданными о процессе обработки.
func (uc *EnrichedLocationUseCase) EnrichLocationBatch(
	ctx context.Context,
	req dto.EnrichLocationBatchRequest,
) (*dto.EnrichLocationBatchResponse, error) {
	if len(req.Locations) == 0 {
		return nil, errors.ErrInvalidRequest
	}

	uc.logger.Info("EnrichLocationBatch started",
		zap.Int("total_locations", len(req.Locations)))

	// Разделяем локации: visible (с координатами) и остальные
	var visibleLocations []dto.LocationInput
	var visibleIndices []int // индексы visible локаций для маппинга результатов

	for i, loc := range req.Locations {
		if loc.IsVisible != nil && *loc.IsVisible &&
			loc.Latitude != nil && loc.Longitude != nil {
			visibleLocations = append(visibleLocations, loc)
			visibleIndices = append(visibleIndices, i)
		}
	}

	// Структуры для результатов горутин
	var wg sync.WaitGroup
	var detectResult *dto.DetectLocationBatchResponse
	var transportResult *dto.PriorityTransportBatchResponse
	var detectErr, transportErr error

	// Горутина 1: DetectLocationBatch для ВСЕХ локаций
	wg.Add(1)
	go func() {
		defer wg.Done()
		detectReq := dto.DetectLocationBatchRequest{
			Locations: req.Locations,
		}
		detectResult, detectErr = uc.searchUC.DetectLocationBatch(ctx, detectReq)
	}()

	// Горутина 2: GetNearestTransportByPriorityBatch только для visible
	if len(visibleLocations) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			// Формируем точки для поиска транспорта
			points := make([]dto.PriorityTransportPoint, len(visibleLocations))
			for i, loc := range visibleLocations {
				points[i] = dto.PriorityTransportPoint{
					Lat: *loc.Latitude,
					Lon: *loc.Longitude,
				}
			}

			transportReq := dto.PriorityTransportBatchRequest{
				Points: points,
				Radius: DefaultTransportRadius,
				Limit:  DefaultTransportLimit,
			}
			transportResult, transportErr = uc.transportUC.GetNearestTransportByPriorityBatch(ctx, transportReq)
		}()
	}

	// Ждём завершения обеих горутин
	wg.Wait()

	// Обрабатываем ошибки
	if detectErr != nil {
		uc.logger.Error("DetectLocationBatch failed", zap.Error(detectErr))
		return nil, detectErr
	}

	// Объединяем результаты
	results := make([]dto.EnrichedLocationResult, len(req.Locations))

	// Копируем результаты детекции
	for i, dr := range detectResult.Results {
		results[i] = dto.EnrichedLocationResult{
			Index:            dr.Index,
			EnrichedLocation: dr.EnrichedLocation,
			Error:            dr.Error,
		}
	}

	// Добавляем транспорт к visible локациям
	if transportResult != nil && transportErr == nil {
		for i, tr := range transportResult.Results {
			if i < len(visibleIndices) {
				originalIdx := visibleIndices[i]
				results[originalIdx].NearestTransport = tr.Stations
			}
		}
	} else if transportErr != nil {
		uc.logger.Warn("GetNearestTransportByPriorityBatch failed, continuing without transport",
			zap.Error(transportErr))
	}

	// Подсчёт статистики
	successCount := 0
	errorCount := 0
	for _, r := range results {
		if r.Error == "" {
			successCount++
		} else {
			errorCount++
		}
	}

	uc.logger.Info("EnrichLocationBatch completed",
		zap.Int("total", len(req.Locations)),
		zap.Int("success", successCount),
		zap.Int("errors", errorCount),
		zap.Int("with_transport", len(visibleLocations)))

	return &dto.EnrichLocationBatchResponse{
		Results: results,
		Meta: dto.EnrichLocationBatchMeta{
			TotalLocations: len(req.Locations),
			SuccessCount:   successCount,
			ErrorCount:     errorCount,
			WithTransport:  len(visibleLocations),
		},
	}, nil
}
