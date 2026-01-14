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
	enrichmentUC *EnrichmentUseCase // для DetectLocationBatch
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

// SetEnrichmentUseCase устанавливает EnrichmentUseCase для DetectLocationBatch
// Используется для избежания циклических зависимостей
func (uc *SearchUseCase) SetEnrichmentUseCase(enrichmentUC *EnrichmentUseCase) {
	uc.enrichmentUC = enrichmentUC
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

// buildSearchRequestIndex creates a unique index for batch search requests
// by combining location index and admin level
func buildSearchRequestIndex(locationIndex, adminLevel int) int {
	return locationIndex*100 + adminLevel
}

// extractLocationIndex extracts the original location index from a search request index
func extractLocationIndex(searchRequestIndex int) int {
	return searchRequestIndex / 100
}

// DetectLocationBatch обнаруживает/обогащает локации без транспорта
// Использует parallel batch queries для эффективной обработки
func (uc *SearchUseCase) DetectLocationBatch(
	ctx context.Context,
	req dto.DetectLocationBatchRequest,
) (*dto.DetectLocationBatchResponse, error) {
	if len(req.Locations) == 0 {
		return nil, errors.ErrInvalidRequest
	}

	// Validate that all locations have a Country
	for i, loc := range req.Locations {
		if loc.Country == "" {
			return nil, errors.ErrInvalidRequest.WithDetails(map[string]interface{}{
				"location_index": i,
				"error":          "country is required",
			})
		}
	}

	uc.logger.Info("DetectLocationBatch started",
		zap.Int("total_locations", len(req.Locations)))

	// Разделяем локации на 2 группы: visible (с координатами) и name-based (без координат или не visible)
	var visibleLocations []dto.LocationInput
	var nameBasedLocations []dto.LocationInput

	for _, loc := range req.Locations {
		hasCoords := loc.Latitude != nil && loc.Longitude != nil
		isVisible := loc.IsVisible != nil && *loc.IsVisible

		if hasCoords && isVisible {
			visibleLocations = append(visibleLocations, loc)
		} else {
			nameBasedLocations = append(nameBasedLocations, loc)
		}
	}

	uc.logger.Debug("Locations split",
		zap.Int("visible_count", len(visibleLocations)),
		zap.Int("name_based_count", len(nameBasedLocations)))

	// Структура для хранения результатов обеих горутин
	type batchResult struct {
		visibleResults   map[int][]*domain.AdminBoundary
		nameBasedResults []domain.BoundarySearchResult
		visibleErr       error
		nameBasedErr     error
	}

	var wg sync.WaitGroup
	var result batchResult

	// Горутина 1: GetByPointBatch для visible локаций (reverse geocoding)
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

	// Горутина 2: SearchByTextBatch для name-based локаций
	if len(nameBasedLocations) > 0 {
		wg.Add(1)
		go func() {
			defer wg.Done()

			var searchRequests []domain.BoundarySearchRequest

			for _, loc := range nameBasedLocations {
				// Добавляем запросы для каждого уровня иерархии
				if loc.Neighborhood != nil && *loc.Neighborhood != "" {
					searchRequests = append(searchRequests, domain.BoundarySearchRequest{
						Index:      buildSearchRequestIndex(loc.Index, 10),
						Name:       *loc.Neighborhood,
						AdminLevel: 10,
					})
				}
				if loc.District != nil && *loc.District != "" {
					searchRequests = append(searchRequests, domain.BoundarySearchRequest{
						Index:      buildSearchRequestIndex(loc.Index, 9),
						Name:       *loc.District,
						AdminLevel: 9,
					})
				}
				if loc.City != nil && *loc.City != "" {
					searchRequests = append(searchRequests, domain.BoundarySearchRequest{
						Index:      buildSearchRequestIndex(loc.Index, 8),
						Name:       *loc.City,
						AdminLevel: 8,
					})
				}
				if loc.Province != nil && *loc.Province != "" {
					searchRequests = append(searchRequests, domain.BoundarySearchRequest{
						Index:      buildSearchRequestIndex(loc.Index, 6),
						Name:       *loc.Province,
						AdminLevel: 6,
					})
				}
				if loc.Region != nil && *loc.Region != "" {
					searchRequests = append(searchRequests, domain.BoundarySearchRequest{
						Index:      buildSearchRequestIndex(loc.Index, 4),
						Name:       *loc.Region,
						AdminLevel: 4,
					})
				}
				// Всегда добавляем страну
				searchRequests = append(searchRequests, domain.BoundarySearchRequest{
					Index:      buildSearchRequestIndex(loc.Index, 2),
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

	// Ждём завершения обеих горутин
	wg.Wait()

	// Собираем результаты
	indexToPos := make(map[int]int)
	for i, loc := range req.Locations {
		indexToPos[loc.Index] = i
	}

	results := make([]dto.DetectLocationResult, len(req.Locations))
	for i, loc := range req.Locations {
		results[i] = dto.DetectLocationResult{
			Index: loc.Index,
		}
	}

	// Обрабатываем visible локации
	if result.visibleErr != nil {
		for _, loc := range visibleLocations {
			if pos, ok := indexToPos[loc.Index]; ok {
				results[pos] = dto.DetectLocationResult{
					Index: loc.Index,
					Error: "failed to resolve by coordinates: " + result.visibleErr.Error(),
				}
			}
		}
	} else {
		for i, loc := range visibleLocations {
			pos, ok := indexToPos[loc.Index]
			if !ok {
				continue
			}

			boundaries := result.visibleResults[i]
			if len(boundaries) == 0 {
				results[pos] = dto.DetectLocationResult{
					Index: loc.Index,
					Error: "no boundaries found for coordinates",
				}
				continue
			}

			enriched := uc.boundariesToEnrichedLocation(boundaries)
			results[pos] = dto.DetectLocationResult{
				Index:            loc.Index,
				EnrichedLocation: enriched,
			}
		}
	}

	// Обрабатываем name-based локации
	if result.nameBasedErr != nil {
		for _, loc := range nameBasedLocations {
			if pos, ok := indexToPos[loc.Index]; ok {
				results[pos] = dto.DetectLocationResult{
					Index: loc.Index,
					Error: "failed to resolve by name: " + result.nameBasedErr.Error(),
				}
			}
		}
	} else {
		searchResultsByLocIdx := make(map[int][]domain.BoundarySearchResult)
		for _, sr := range result.nameBasedResults {
			locIdx := extractLocationIndex(sr.Index)
			searchResultsByLocIdx[locIdx] = append(searchResultsByLocIdx[locIdx], sr)
		}

		for _, loc := range nameBasedLocations {
			pos, ok := indexToPos[loc.Index]
			if !ok {
				continue
			}

			searchResults := searchResultsByLocIdx[loc.Index]
			var foundBoundaries []*domain.AdminBoundary
			for _, sr := range searchResults {
				if sr.Found && sr.Boundary != nil {
					foundBoundaries = append(foundBoundaries, sr.Boundary)
				}
			}

			if len(foundBoundaries) == 0 {
				results[pos] = dto.DetectLocationResult{
					Index: loc.Index,
					Error: "no boundaries found by name",
				}
				continue
			}

			enriched := uc.boundariesToEnrichedLocation(foundBoundaries)
			results[pos] = dto.DetectLocationResult{
				Index:            loc.Index,
				EnrichedLocation: enriched,
			}
		}
	}

	uc.logger.Info("DetectLocationBatch completed",
		zap.Int("total", len(req.Locations)))

	return &dto.DetectLocationBatchResponse{
		Results: results,
	}, nil
}

// boundariesToEnrichedLocation преобразует boundaries в EnrichedLocationDTO
func (uc *SearchUseCase) boundariesToEnrichedLocation(boundaries []*domain.AdminBoundary) *dto.EnrichedLocationDTO {
	enriched := &dto.EnrichedLocationDTO{}

	for _, b := range boundaries {
		bInfo := &dto.BoundaryInfoDTO{
			ID:             b.ID,
			Name:           b.Name,
			TranslateNames: make(map[string]string),
		}

		// Собираем переводы
		if b.NameEn != "" {
			bInfo.TranslateNames["en"] = b.NameEn
		}
		if b.NameEs != "" {
			bInfo.TranslateNames["es"] = b.NameEs
		}
		if b.NameCa != "" {
			bInfo.TranslateNames["ca"] = b.NameCa
		}
		if b.NameRu != "" {
			bInfo.TranslateNames["ru"] = b.NameRu
		}
		if b.NameUk != "" {
			bInfo.TranslateNames["uk"] = b.NameUk
		}
		if b.NameFr != "" {
			bInfo.TranslateNames["fr"] = b.NameFr
		}
		if b.NamePt != "" {
			bInfo.TranslateNames["pt"] = b.NamePt
		}
		if b.NameIt != "" {
			bInfo.TranslateNames["it"] = b.NameIt
		}
		if b.NameDe != "" {
			bInfo.TranslateNames["de"] = b.NameDe
		}

		if len(bInfo.TranslateNames) == 0 {
			bInfo.TranslateNames = nil
		}

		switch b.AdminLevel {
		case 2:
			enriched.Country = bInfo
		case 4:
			enriched.Region = bInfo
		case 6:
			enriched.Province = bInfo
		case 8:
			enriched.City = bInfo
		case 9:
			enriched.District = bInfo
		case 10:
			enriched.Neighborhood = bInfo
		}
	}

	return enriched
}
