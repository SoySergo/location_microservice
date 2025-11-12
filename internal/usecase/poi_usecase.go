package usecase

import (
	"context"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/pkg/errors"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

type POIUseCase struct {
	poiRepo repository.POIRepository
	logger  *zap.Logger
}

func NewPOIUseCase(
	poiRepo repository.POIRepository,
	logger *zap.Logger,
) *POIUseCase {
	return &POIUseCase{
		poiRepo: poiRepo,
		logger:  logger,
	}
}

func (uc *POIUseCase) SearchByRadius(
	ctx context.Context,
	req dto.RadiusPOIRequest,
) (*dto.RadiusPOIResponse, error) {
	// Validate coordinates
	if !utils.ValidateCoordinates(req.Lat, req.Lon) {
		return nil, errors.ErrInvalidCoordinates
	}

	// Validate radius
	if !utils.ValidateRadius(req.RadiusKm) {
		return nil, errors.ErrInvalidRadius
	}

	// Set default limit
	if req.Limit == 0 {
		req.Limit = 100
	}

	// Search POIs
	pois, err := uc.poiRepo.GetNearby(
		ctx,
		req.Lat,
		req.Lon,
		req.RadiusKm,
		req.Categories,
	)
	if err != nil {
		uc.logger.Error("Failed to search POIs by radius", zap.Error(err))
		return nil, err
	}

	// Apply limit
	if len(pois) > req.Limit {
		pois = pois[:req.Limit]
	}

	// Build response
	result := make([]dto.POISimple, 0, len(pois))
	for _, poi := range pois {
		distance := utils.HaversineDistance(req.Lat, req.Lon, poi.Lat, poi.Lon) * 1000 // to meters

		result = append(result, dto.POISimple{
			ID:          poi.ID,
			Name:        poi.Name,
			Category:    poi.Category,
			Subcategory: poi.Subcategory,
			Lat:         poi.Lat,
			Lon:         poi.Lon,
			Distance:    distance,
		})
	}

	return &dto.RadiusPOIResponse{
		POIs:  result,
		Total: len(result),
	}, nil
}

func (uc *POIUseCase) GetCategories(ctx context.Context, lang string) ([]*domain.POICategory, error) {
	categories, err := uc.poiRepo.GetCategories(ctx)
	if err != nil {
		uc.logger.Error("Failed to get POI categories", zap.Error(err))
		return nil, err
	}

	return categories, nil
}

func (uc *POIUseCase) GetSubcategories(ctx context.Context, categoryID string, lang string) ([]*domain.POISubcategory, error) {
	subcategories, err := uc.poiRepo.GetSubcategories(ctx, categoryID)
	if err != nil {
		uc.logger.Error("Failed to get POI subcategories", zap.Error(err))
		return nil, err
	}

	return subcategories, nil
}
