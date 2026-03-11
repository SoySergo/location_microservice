package usecase

import (
	"context"

	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/pkg/errors"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

const (
	// defaultPropertyLocationRadius — радиус поиска по умолчанию (метры)
	defaultPropertyLocationRadius = 1000
	// defaultPropertyTransportLimit — кол-во ближайших станций по умолчанию
	defaultPropertyTransportLimit = 5
	// environmentRadiusKm — радиус для проверки окружения в километрах
	environmentRadiusKm = 1.0
)

// PropertyLocationUseCase — агрегирующий usecase для получения данных локации объекта недвижимости.
// Объединяет данные о транспорте, POI и окружающей среде.
type PropertyLocationUseCase struct {
	transportUC     *TransportUseCase
	poiRepo         repository.POIRepository
	environmentRepo repository.EnvironmentRepository
	logger          *zap.Logger
}

// NewPropertyLocationUseCase создает новый PropertyLocationUseCase
func NewPropertyLocationUseCase(
	transportUC *TransportUseCase,
	poiRepo repository.POIRepository,
	environmentRepo repository.EnvironmentRepository,
	logger *zap.Logger,
) *PropertyLocationUseCase {
	return &PropertyLocationUseCase{
		transportUC:     transportUC,
		poiRepo:         poiRepo,
		environmentRepo: environmentRepo,
		logger:          logger,
	}
}

// GetPropertyLocationData возвращает агрегированные данные локации объекта.
// Параллельно запрашивает ближайший транспорт, количество POI по категориям и наличие объектов окружения.
func (uc *PropertyLocationUseCase) GetPropertyLocationData(
	ctx context.Context,
	req dto.PropertyLocationRequest,
) (*dto.PropertyLocationResponse, error) {
	if !utils.ValidateCoordinates(req.Lat, req.Lon) {
		return nil, errors.ErrInvalidCoordinates
	}

	radius := req.Radius
	if radius == 0 {
		radius = defaultPropertyLocationRadius
	}

	g, ctx := errgroup.WithContext(ctx)

	var transportStations []dto.PriorityTransportStation
	var poiCounts map[string]int
	var envSummary dto.EnvironmentSummary

	// Горутина 1: ближайший транспорт с приоритетом
	g.Go(func() error {
		transportReq := dto.PriorityTransportRequest{
			Lat:    req.Lat,
			Lon:    req.Lon,
			Radius: float64(radius),
			Limit:  defaultPropertyTransportLimit,
		}
		result, err := uc.transportUC.GetNearestTransportByPriority(ctx, transportReq)
		if err != nil {
			uc.logger.Warn("Failed to get priority transport for property location", zap.Error(err))
			return nil // не прерываем остальные запросы
		}
		transportStations = result.Stations
		return nil
	})

	// Горутина 2: количество POI по категориям
	g.Go(func() error {
		counts, err := uc.poiRepo.CountByCategories(ctx, req.Lat, req.Lon, radius)
		if err != nil {
			uc.logger.Warn("Failed to count POI by categories", zap.Error(err))
			return nil
		}
		poiCounts = counts
		return nil
	})

	// Горутина 3: наличие объектов окружения
	g.Go(func() error {
		radiusKm := float64(radius) / 1000.0

		greenSpaces, err := uc.environmentRepo.GetGreenSpacesNearby(ctx, req.Lat, req.Lon, radiusKm)
		if err != nil {
			uc.logger.Warn("Failed to check green spaces", zap.Error(err))
		}
		envSummary.GreenSpacesNearby = len(greenSpaces) > 0

		waterBodies, err := uc.environmentRepo.GetWaterBodiesNearby(ctx, req.Lat, req.Lon, radiusKm)
		if err != nil {
			uc.logger.Warn("Failed to check water bodies", zap.Error(err))
		}
		envSummary.WaterNearby = len(waterBodies) > 0

		beaches, err := uc.environmentRepo.GetBeachesNearby(ctx, req.Lat, req.Lon, radiusKm)
		if err != nil {
			uc.logger.Warn("Failed to check beaches", zap.Error(err))
		}
		envSummary.BeachNearby = len(beaches) > 0

		return nil
	})

	if err := g.Wait(); err != nil {
		return nil, err
	}

	if transportStations == nil {
		transportStations = []dto.PriorityTransportStation{}
	}
	if poiCounts == nil {
		poiCounts = make(map[string]int)
	}

	return &dto.PropertyLocationResponse{
		NearestTransport: transportStations,
		POISummary:       poiCounts,
		Environment:      envSummary,
	}, nil
}
