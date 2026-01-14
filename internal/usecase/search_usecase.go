package usecase

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/pkg/errors"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/usecase/dto"
)

// SearchUseCase - use case для поиска и геокодирования
type SearchUseCase struct {
	boundaryRepo repository.BoundaryRepository
	cacheRepo    repository.CacheRepository
	logger       *zap.Logger
	cacheTTL     time.Duration
}

// NewSearchUseCase - создание нового SearchUseCase
func NewSearchUseCase(
	boundaryRepo repository.BoundaryRepository,
	cacheRepo repository.CacheRepository,
	logger *zap.Logger,
	cacheTTL time.Duration,
) *SearchUseCase {
	return &SearchUseCase{
		boundaryRepo: boundaryRepo,
		cacheRepo:    cacheRepo,
		logger:       logger,
		cacheTTL:     cacheTTL,
	}
}

// Search - поиск границ по текстовому запросу
func (uc *SearchUseCase) Search(ctx context.Context, req dto.SearchRequest) (*dto.SearchResponse, error) {
	// Установка значений по умолчанию
	if req.Limit == 0 {
		req.Limit = 10
	}

	// Поиск границ
	boundaries, err := uc.boundaryRepo.SearchByText(
		ctx,
		req.Query,
		req.Language,
		req.AdminLevels,
		req.Limit,
	)
	if err != nil {
		uc.logger.Error("Failed to search boundaries", zap.Error(err))
		return nil, err
	}

	// Преобразование в response
	results := make([]dto.SearchResult, 0, len(boundaries))
	for _, b := range boundaries {
		// Convert to DTO with string ID
		results = append(results, dto.ConvertSearchResult(b))
	}

	return &dto.SearchResponse{
		Results: results,
		Total:   len(results),
	}, nil
}

// ReverseGeocode - обратное геокодирование координат
func (uc *SearchUseCase) ReverseGeocode(ctx context.Context, req dto.ReverseGeocodeRequest) (*dto.ReverseGeocodeResponse, error) {
	// Валидация координат
	if !utils.ValidateCoordinates(req.Lat, req.Lon) {
		return nil, errors.ErrInvalidCoordinates
	}

	// Получение адреса
	addr, err := uc.boundaryRepo.ReverseGeocode(ctx, req.Lat, req.Lon)
	if err != nil {
		uc.logger.Error("Failed to reverse geocode", zap.Error(err))
		return nil, err
	}

	return &dto.ReverseGeocodeResponse{
		Address: *addr,
	}, nil
}

// BatchReverseGeocode - пакетное обратное геокодирование
func (uc *SearchUseCase) BatchReverseGeocode(
	ctx context.Context,
	req dto.BatchReverseGeocodeRequest,
) (*dto.BatchReverseGeocodeResponse, error) {
	addresses := make([]domain.Address, len(req.Points))

	for i, point := range req.Points {
		if !utils.ValidateCoordinates(point.Lat, point.Lon) {
			return nil, errors.ErrInvalidCoordinates.WithDetails(map[string]interface{}{
				"point_index": i,
			})
		}

		addr, err := uc.boundaryRepo.ReverseGeocode(ctx, point.Lat, point.Lon)
		if err != nil {
			// Логируем ошибку, но продолжаем с пустым адресом
			uc.logger.Warn("Failed to geocode point", zap.Int("index", i), zap.Error(err))
			addresses[i] = domain.Address{}
			continue
		}

		addresses[i] = *addr
	}

	return &dto.BatchReverseGeocodeResponse{
		Addresses: addresses,
	}, nil
}

// DetectLocationBatch обогащает пачку локаций эффективно (2 параллельных запроса в БД)
// Логика:
// 1. Делим локации на 2 группы: visible (есть координаты) и name-based (нет координат или не visible)
// 2. Запускаем параллельно 2 запроса в БД:
//   - GetByPointBatch для visible локаций (reverse geocoding по координатам)
//   - SearchByTextBatch для name-based локаций (поиск по названиям)
//
// 3. Объединяем результаты и возвращаем
func (uc *SearchUseCase) DetectLocationBatch(
	ctx context.Context,
	req dto.DetectLocationBatchRequest,
) (*dto.DetectLocationBatchResponse, error) {
	if len(req.Locations) == 0 {
		return nil, errors.ErrInvalidRequest
	}

	uc.logger.Info("DetectLocationBatch started",
		zap.Int("total_locations", len(req.Locations)))

	// Шаг 1: Разделяем локации на 2 группы
	var visibleLocations []dto.LocationInput   // с координатами и visible=true
	var nameBasedLocations []dto.LocationInput // без координат или visible=false

	for _, loc := range req.Locations {
		hasCoords := loc.Latitude != nil && loc.Longitude != nil
		isVisible := loc.IsVisible != nil && *loc.IsVisible

		// Если есть координаты И visible=true -> reverse geocoding
		// Иначе -> поиск по названиям
		if hasCoords && isVisible {
			visibleLocations = append(visibleLocations, loc)
		} else {
			nameBasedLocations = append(nameBasedLocations, loc)
		}
	}

	uc.logger.Debug("Locations split",
		zap.Int("visible_count", len(visibleLocations)),
		zap.Int("name_based_count", len(nameBasedLocations)))

	// Шаг 2: Запускаем параллельно 2 запроса к БД
	type batchResult struct {
		visibleResults   map[int][]*domain.AdminBoundary
		nameBasedResults []domain.BoundarySearchResult
		visibleErr       error
		nameBasedErr     error
	}

	resultChan := make(chan batchResult, 1)

	go func() {
		var wg sync.WaitGroup
		var result batchResult

		// Горутина для visible локаций (reverse geocoding по координатам)
		if len(visibleLocations) > 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()

				points := make([]domain.LatLon, len(visibleLocations))
				for i, loc := range visibleLocations {
					points[i] = domain.LatLon{Lat: *loc.Latitude, Lon: *loc.Longitude}
				}

				boundariesByPoint, err := uc.boundaryRepo.GetByPointBatch(ctx, points)
				if err != nil {
					uc.logger.Error("GetByPointBatch failed", zap.Error(err))
					result.visibleErr = err
					return
				}
				result.visibleResults = boundariesByPoint
			}()
		}

		// Горутина для name-based локаций (поиск по названиям)
		if len(nameBasedLocations) > 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Собираем все запросы на поиск
				var searchRequests []domain.BoundarySearchRequest

				for _, loc := range nameBasedLocations {
					// Добавляем запросы для каждого уровня иерархии, который есть
					// Приоритет: от детального к общему
					if loc.Neighborhood != nil && *loc.Neighborhood != "" {
						searchRequests = append(searchRequests, domain.BoundarySearchRequest{
							Index:      loc.Index*100 + 10, // уникальный ID
							Name:       *loc.Neighborhood,
							AdminLevel: 10,
						})
					}
					if loc.District != nil && *loc.District != "" {
						searchRequests = append(searchRequests, domain.BoundarySearchRequest{
							Index:      loc.Index*100 + 9,
							Name:       *loc.District,
							AdminLevel: 9,
						})
					}
					if loc.City != nil && *loc.City != "" {
						searchRequests = append(searchRequests, domain.BoundarySearchRequest{
							Index:      loc.Index*100 + 8,
							Name:       *loc.City,
							AdminLevel: 8,
						})
					}
					if loc.Province != nil && *loc.Province != "" {
						searchRequests = append(searchRequests, domain.BoundarySearchRequest{
							Index:      loc.Index*100 + 6,
							Name:       *loc.Province,
							AdminLevel: 6,
						})
					}
					if loc.Region != nil && *loc.Region != "" {
						searchRequests = append(searchRequests, domain.BoundarySearchRequest{
							Index:      loc.Index*100 + 4,
							Name:       *loc.Region,
							AdminLevel: 4,
						})
					}
					// Всегда добавляем страну
					searchRequests = append(searchRequests, domain.BoundarySearchRequest{
						Index:      loc.Index*100 + 2,
						Name:       loc.Country,
						AdminLevel: 2,
					})
				}

				if len(searchRequests) > 0 {
					results, err := uc.boundaryRepo.SearchByTextBatch(ctx, searchRequests)
					if err != nil {
						uc.logger.Error("SearchByTextBatch failed", zap.Error(err))
						result.nameBasedErr = err
						return
					}
					result.nameBasedResults = results
				}
			}()
		}

		wg.Wait()
		resultChan <- result
	}()

	// Ждём результаты
	batchRes := <-resultChan

	// Шаг 3: Собираем результаты
	// Создаём маппинг loc.Index -> позиция в слайсе results
	indexToPos := make(map[int]int)
	for i, loc := range req.Locations {
		indexToPos[loc.Index] = i
	}

	results := make([]dto.LocationDetectionResult, len(req.Locations))
	// Инициализируем результаты
	for i, loc := range req.Locations {
		results[i] = dto.LocationDetectionResult{
			Index: loc.Index,
		}
	}

	successCount := 0
	errorCount := 0
	dbQueriesCount := 0

	if len(visibleLocations) > 0 {
		dbQueriesCount++
	}
	if len(nameBasedLocations) > 0 {
		dbQueriesCount++
	}

	// Обрабатываем visible локации (reverse geocoding)
	if batchRes.visibleErr != nil {
		for _, loc := range visibleLocations {
			if pos, ok := indexToPos[loc.Index]; ok {
				results[pos] = dto.LocationDetectionResult{
					Index: loc.Index,
					Error: "failed to resolve by coordinates: " + batchRes.visibleErr.Error(),
				}
				errorCount++
			}
		}
	} else {
		// Обрабатываем visible локации по их позиции в visibleLocations slice
		for i, loc := range visibleLocations {
			pos, ok := indexToPos[loc.Index]
			if !ok {
				continue
			}

			boundaries := batchRes.visibleResults[i]
			if len(boundaries) == 0 {
				results[pos] = dto.LocationDetectionResult{
					Index: loc.Index,
					Error: "no boundaries found for coordinates",
				}
				errorCount++
				continue
			}

			enriched := uc.boundariesToEnrichedLocation(boundaries)
			results[pos] = dto.LocationDetectionResult{
				Index:            loc.Index,
				EnrichedLocation: enriched,
			}
			successCount++
		}
	}

	// Обрабатываем name-based локации
	if batchRes.nameBasedErr != nil {
		for _, loc := range nameBasedLocations {
			if pos, ok := indexToPos[loc.Index]; ok {
				results[pos] = dto.LocationDetectionResult{
					Index: loc.Index,
					Error: "failed to resolve by name: " + batchRes.nameBasedErr.Error(),
				}
				errorCount++
			}
		}
	} else {
		// Группируем результаты поиска по исходному индексу локации
		searchResultsByLocIdx := make(map[int][]domain.BoundarySearchResult)
		for _, sr := range batchRes.nameBasedResults {
			locIdx := sr.Index / 100 // восстанавливаем исходный индекс
			searchResultsByLocIdx[locIdx] = append(searchResultsByLocIdx[locIdx], sr)
		}

		for _, loc := range nameBasedLocations {
			pos, ok := indexToPos[loc.Index]
			if !ok {
				continue
			}

			searchResults := searchResultsByLocIdx[loc.Index]

			// Собираем найденные границы
			var foundBoundaries []*domain.AdminBoundary
			for _, sr := range searchResults {
				if sr.Found && sr.Boundary != nil {
					foundBoundaries = append(foundBoundaries, sr.Boundary)
				}
			}

			if len(foundBoundaries) == 0 {
				results[pos] = dto.LocationDetectionResult{
					Index: loc.Index,
					Error: "no boundaries found by name",
				}
				errorCount++
				continue
			}

			enriched := uc.boundariesToEnrichedLocation(foundBoundaries)
			results[pos] = dto.LocationDetectionResult{
				Index:            loc.Index,
				EnrichedLocation: enriched,
			}
			successCount++
		}
	}

	uc.logger.Info("DetectLocationBatch completed",
		zap.Int("total", len(req.Locations)),
		zap.Int("success", successCount),
		zap.Int("errors", errorCount),
		zap.Int("visible_processed", len(visibleLocations)),
		zap.Int("name_based_processed", len(nameBasedLocations)),
		zap.Int("db_queries", dbQueriesCount))

	return &dto.DetectLocationBatchResponse{
		Results: results,
		Meta: dto.LocationBatchMeta{
			TotalLocations:   len(req.Locations),
			SuccessCount:     successCount,
			ErrorCount:       errorCount,
			VisibleCount:     len(visibleLocations),
			NameResolveCount: len(nameBasedLocations),
			DBQueriesCount:   dbQueriesCount,
		},
	}, nil
}

// boundariesToEnrichedLocation преобразует слайс AdminBoundary в EnrichedLocationDTO
func (uc *SearchUseCase) boundariesToEnrichedLocation(boundaries []*domain.AdminBoundary) *dto.EnrichedLocationDTO {
	result := &dto.EnrichedLocationDTO{}

	for _, b := range boundaries {
		info := &dto.BoundaryInfoDTO{
			ID:             b.ID,
			Name:           b.Name,
			TranslateNames: make(map[string]string),
		}

		// Собираем переводы
		if b.NameEn != "" {
			info.TranslateNames["en"] = b.NameEn
		}
		if b.NameEs != "" {
			info.TranslateNames["es"] = b.NameEs
		}
		if b.NameCa != "" {
			info.TranslateNames["ca"] = b.NameCa
		}
		if b.NameRu != "" {
			info.TranslateNames["ru"] = b.NameRu
		}
		if b.NameUk != "" {
			info.TranslateNames["uk"] = b.NameUk
		}
		if b.NameFr != "" {
			info.TranslateNames["fr"] = b.NameFr
		}
		if b.NamePt != "" {
			info.TranslateNames["pt"] = b.NamePt
		}
		if b.NameIt != "" {
			info.TranslateNames["it"] = b.NameIt
		}
		if b.NameDe != "" {
			info.TranslateNames["de"] = b.NameDe
		}

		if len(info.TranslateNames) == 0 {
			info.TranslateNames = nil
		}

		switch b.AdminLevel {
		case 2:
			result.Country = info
		case 4:
			result.Region = info
		case 6:
			result.Province = info
		case 8:
			result.City = info
		case 9:
			result.District = info
		case 10:
			result.Neighborhood = info
		}
	}

	return result
}
