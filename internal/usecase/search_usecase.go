package usecase

import (
	"context"
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
		results = append(results, dto.SearchResult{
			ID:         b.ID,
			Name:       b.Name,
			Type:       b.Type,
			AdminLevel: b.AdminLevel,
			CenterPoint: domain.Point{
				Lat: b.CenterLat,
				Lon: b.CenterLon,
			},
			AreaSqKm: b.AreaSqKm,
		})
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
