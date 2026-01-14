package usecase

import (
	"context"
	"sync"

	"github.com/location-microservice/internal/pkg/errors"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

// Ensure EnrichedLocationUseCase implements BatchLocationEnricher interface
var _ BatchLocationEnricher = (*EnrichedLocationUseCase)(nil)

// EnrichedLocationUseCase - usecase для полного обогащения локаций
type EnrichedLocationUseCase struct {
	enrichmentDebugUC *EnrichmentDebugUseCase // для EnrichLocationBatch и GetNearestTransportByPriorityBatch
	logger            *zap.Logger
}

// NewEnrichedLocationUseCase создает новый EnrichedLocationUseCase
func NewEnrichedLocationUseCase(
	enrichmentDebugUC *EnrichmentDebugUseCase,
	logger *zap.Logger,
) *EnrichedLocationUseCase {
	return &EnrichedLocationUseCase{
		enrichmentDebugUC: enrichmentDebugUC,
		logger:            logger,
	}
}

// EnrichLocationBatch обогащает пачку локаций параллельно.
//
// Архитектура параллельной обработки:
// - Горутина 1: EnrichLocationBatch (из EnrichmentDebugUseCase) для ВСЕХ локаций → возвращает enriched location с ID границ
// - Горутина 2: GetNearestTransportByPriorityBatch (из EnrichmentDebugUseCase) только для IsVisible=true → возвращает транспорт
// - Обе горутины работают параллельно с sync.WaitGroup
// - Результаты объединяются после завершения обеих горутин
//
// Поведение при ошибках:
// - Ошибка EnrichLocationBatch прерывает весь процесс (критическая)
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
	var enrichResult *dto.EnrichmentDebugLocationBatchResponse
	var transportResult *dto.PriorityTransportBatchResponse
	var enrichErr, transportErr error

	// Горутина 1: EnrichLocationBatch для ВСЕХ локаций
	wg.Add(1)
	go func() {
		defer wg.Done()
		enrichReq := dto.EnrichmentDebugLocationBatchRequest{
			Locations: req.Locations,
		}
		enrichResult, enrichErr = uc.enrichmentDebugUC.EnrichLocationBatch(ctx, enrichReq)
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
				Radius: 1500, // 1.5 km по умолчанию
				Limit:  5,    // 5 станций на точку
			}
			transportResult, transportErr = uc.enrichmentDebugUC.GetNearestTransportByPriorityBatch(ctx, transportReq)
		}()
	}

	// Ждём завершения обеих горутин
	wg.Wait()

	// Обрабатываем ошибки
	if enrichErr != nil {
		uc.logger.Error("EnrichLocationBatch failed", zap.Error(enrichErr))
		return nil, enrichErr
	}

	// Объединяем результаты
	results := make([]dto.EnrichedLocationResult, len(req.Locations))

	// Копируем результаты обогащения
	for i, er := range enrichResult.Results {
		results[i] = dto.EnrichedLocationResult{
			Index:            er.Index,
			EnrichedLocation: er.EnrichedLocation,
			Error:            er.Error,
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
