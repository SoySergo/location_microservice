package usecase

import (
	"context"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/pkg/errors"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

const (
	// defaultNearbyRadiusKm — радиус поиска по умолчанию (км)
	defaultNearbyRadiusKm = 1.0
	// defaultNearbyLimit — лимит результатов по умолчанию
	defaultNearbyLimit = 20
	// defaultTransportNearbyRadiusM — радиус поиска транспорта по умолчанию (метры)
	defaultTransportNearbyRadiusM = 1500
	// defaultTransportNearbyLimit — лимит станций транспорта по умолчанию
	defaultTransportNearbyLimit = 10
)

// NearbyUseCase — usecase для получения данных поблизости по категории.
// Для "transport" делегирует в TransportUseCase, для остальных — в POIUseCase.
type NearbyUseCase struct {
	transportUC *TransportUseCase
	poiUC       *POIUseCase
	logger      *zap.Logger
}

// NewNearbyUseCase создает новый NearbyUseCase
func NewNearbyUseCase(
	transportUC *TransportUseCase,
	poiUC *POIUseCase,
	logger *zap.Logger,
) *NearbyUseCase {
	return &NearbyUseCase{
		transportUC: transportUC,
		poiUC:       poiUC,
		logger:      logger,
	}
}

// GetNearbyTransport возвращает станции транспорта с приоритетом (metro/train → tram → bus)
func (uc *NearbyUseCase) GetNearbyTransport(
	ctx context.Context,
	lat, lon float64,
	radiusM float64,
	limit int,
) (*dto.PriorityTransportResponse, error) {
	if !utils.ValidateCoordinates(lat, lon) {
		return nil, errors.ErrInvalidCoordinates
	}

	if radiusM == 0 {
		radiusM = defaultTransportNearbyRadiusM
	}
	if limit == 0 {
		limit = defaultTransportNearbyLimit
	}

	req := dto.PriorityTransportRequest{
		Lat:    lat,
		Lon:    lon,
		Radius: radiusM,
		Limit:  limit,
	}

	return uc.transportUC.GetNearestTransportByPriority(ctx, req)
}

// GetNearbyPOI возвращает POI поблизости по фронтенд-категории
func (uc *NearbyUseCase) GetNearbyPOI(
	ctx context.Context,
	category string,
	lat, lon float64,
	radiusKm float64,
	limit int,
) (*dto.NearbyPOIResponse, error) {
	if !utils.ValidateCoordinates(lat, lon) {
		return nil, errors.ErrInvalidCoordinates
	}

	osmCategories := domain.GetOSMCategories(category)
	if osmCategories == nil {
		return nil, errors.ErrInvalidRequest
	}

	if radiusKm == 0 {
		radiusKm = defaultNearbyRadiusKm
	}
	if limit == 0 {
		limit = defaultNearbyLimit
	}

	req := dto.RadiusPOIRequest{
		Lat:        lat,
		Lon:        lon,
		RadiusKm:   radiusKm,
		Categories: osmCategories,
		Limit:      limit,
	}

	result, err := uc.poiUC.SearchByRadius(ctx, req)
	if err != nil {
		return nil, err
	}

	return &dto.NearbyPOIResponse{
		Category: category,
		Items:    result.POIs,
		Total:    result.Total,
	}, nil
}
