package usecase

import (
	"context"
	"sync"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/pkg/errors"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

const (
	// DefaultTransportRadius - радиус поиска транспорта по умолчанию (метры)
	DefaultTransportRadius = 1500 // 1.5 km
	// DefaultTransportLimit - максимальное количество станций на точку
	DefaultTransportLimit = 5
)

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

// EnrichLocationBatch обогащает пачку локаций параллельно
// 1. Горутина 1: DetectLocationBatch для всех локаций → возвращает адреса с ID
// 2. Горутина 2: GetNearestTransportByPriorityBatch для IsVisible=true → возвращает транспорт
// 3. Ждём завершения обеих горутин
// 4. Объединяем и возвращаем результаты
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

// EnrichLocation реализует интерфейс LocationEnricher для совместимости с worker
func (uc *EnrichedLocationUseCase) EnrichLocation(
	ctx context.Context,
	event *domain.LocationEnrichEvent,
) (*domain.LocationDoneEvent, error) {
	// Конвертируем event в batch request с одним элементом
	input := dto.LocationInput{
		Index:        0,
		Country:      event.Country,
		Region:       event.Region,
		Province:     event.Province,
		City:         event.City,
		District:     event.District,
		Neighborhood: event.Neighborhood,
		Latitude:     event.Latitude,
		Longitude:    event.Longitude,
		IsVisible:    event.IsVisible,
	}

	req := dto.EnrichLocationBatchRequest{
		Locations: []dto.LocationInput{input},
	}

	resp, err := uc.EnrichLocationBatch(ctx, req)
	if err != nil {
		return &domain.LocationDoneEvent{
			PropertyID: event.PropertyID,
			Error:      err.Error(),
		}, nil
	}

	if len(resp.Results) == 0 {
		return &domain.LocationDoneEvent{
			PropertyID: event.PropertyID,
			Error:      "no results returned",
		}, nil
	}

	result := resp.Results[0]

	// Конвертируем результат в domain.LocationDoneEvent
	doneEvent := &domain.LocationDoneEvent{
		PropertyID: event.PropertyID,
		Error:      result.Error,
	}

	if result.EnrichedLocation != nil {
		doneEvent.EnrichedLocation = convertToEnrichedLocation(result.EnrichedLocation)
	}

	if len(result.NearestTransport) > 0 {
		doneEvent.NearestTransport = convertToNearestStations(result.NearestTransport)
	}

	return doneEvent, nil
}

// Вспомогательные функции конвертации
func convertToEnrichedLocation(dto *dto.EnrichedLocationDTO) *domain.EnrichedLocation {
	if dto == nil {
		return nil
	}

	result := &domain.EnrichedLocation{
		IsAddressVisible: dto.IsAddressVisible,
	}

	if dto.Country != nil {
		result.Country = &domain.BoundaryInfo{
			ID:             dto.Country.ID,
			Name:           dto.Country.Name,
			TranslateNames: dto.Country.TranslateNames,
		}
	}
	if dto.Region != nil {
		result.Region = &domain.BoundaryInfo{
			ID:             dto.Region.ID,
			Name:           dto.Region.Name,
			TranslateNames: dto.Region.TranslateNames,
		}
	}
	if dto.Province != nil {
		result.Province = &domain.BoundaryInfo{
			ID:             dto.Province.ID,
			Name:           dto.Province.Name,
			TranslateNames: dto.Province.TranslateNames,
		}
	}
	if dto.City != nil {
		result.City = &domain.BoundaryInfo{
			ID:             dto.City.ID,
			Name:           dto.City.Name,
			TranslateNames: dto.City.TranslateNames,
		}
	}
	if dto.District != nil {
		result.District = &domain.BoundaryInfo{
			ID:             dto.District.ID,
			Name:           dto.District.Name,
			TranslateNames: dto.District.TranslateNames,
		}
	}
	if dto.Neighborhood != nil {
		result.Neighborhood = &domain.BoundaryInfo{
			ID:             dto.Neighborhood.ID,
			Name:           dto.Neighborhood.Name,
			TranslateNames: dto.Neighborhood.TranslateNames,
		}
	}

	return result
}

func convertToNearestStations(stations []dto.PriorityTransportStation) []domain.NearestStation {
	result := make([]domain.NearestStation, len(stations))
	for i, s := range stations {
		result[i] = domain.NearestStation{
			StationID:       s.StationID,
			Name:            s.Name,
			Type:            s.Type,
			Lat:             s.Lat,
			Lon:             s.Lon,
			Distance:        s.LinearDistance,
			WalkingDistance: &s.WalkingDistance,
			WalkingDuration: &s.WalkingTime,
		}

		if len(s.Lines) > 0 {
			result[i].Lines = make([]domain.TransportLineInfo, len(s.Lines))
			for j, l := range s.Lines {
				result[i].Lines[j] = domain.TransportLineInfo{
					ID:    l.ID,
					Name:  l.Name,
					Ref:   l.Ref,
					Type:  l.Type,
					Color: l.Color,
				}
			}
		}
	}
	return result
}

// Compile-time check that EnrichedLocationUseCase implements LocationEnricher
var _ LocationEnricher = (*EnrichedLocationUseCase)(nil)
