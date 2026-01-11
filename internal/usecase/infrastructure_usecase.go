package usecase

import (
	"context"
	"fmt"
	"math"

	"github.com/google/uuid"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"go.uber.org/zap"
)

// InfrastructureUseCase - use case для работы с инфраструктурой
type InfrastructureUseCase struct {
	transportRepo  repository.TransportRepository
	batchScheduler *MapboxBatchScheduler
	logger         *zap.Logger
	maxMetro       int
	maxTrain       int
}

// NewInfrastructureUseCase создает новый InfrastructureUseCase
func NewInfrastructureUseCase(
	transportRepo repository.TransportRepository,
	batchScheduler *MapboxBatchScheduler,
	logger *zap.Logger,
	maxMetro, maxTrain int,
) *InfrastructureUseCase {
	return &InfrastructureUseCase{
		transportRepo:  transportRepo,
		batchScheduler: batchScheduler,
		logger:         logger,
		maxMetro:       maxMetro,
		maxTrain:       maxTrain,
	}
}

// GetInfrastructure получает инфраструктуру для локации через batch scheduler
func (uc *InfrastructureUseCase) GetInfrastructure(
	ctx context.Context,
	propertyID uuid.UUID,
	lat, lon float64,
	transportRadius float64,
) (*domain.InfrastructureResult, error) {
	// 1. Получаем транспорт с приоритетами (только метро и поезд)
	transportPriorities := []domain.TransportPriority{
		{Type: "metro", Limit: uc.maxMetro},
		{Type: "train", Limit: uc.maxTrain},
	}

	transportStations, err := uc.transportRepo.GetNearestStationsGrouped(
		ctx, lat, lon, transportPriorities, transportRadius,
	)
	if err != nil {
		uc.logger.Error("Failed to get transport", zap.Error(err))
		return nil, fmt.Errorf("failed to get transport: %w", err)
	}

	// 2. Отправляем запрос в batch scheduler
	resultChan := make(chan *MapboxBatchResult, 1)
	batchReq := &MapboxBatchRequest{
		PropertyID: propertyID,
		Lat:        lat,
		Lon:        lon,
		Stations:   transportStations,
		ResultChan: resultChan,
	}

	uc.batchScheduler.ScheduleRequest(batchReq)

	// 3. Ждем результат
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result := <-resultChan:
		if result.Error != nil {
			uc.logger.Warn("Mapbox batch request failed",
				zap.String("property_id", propertyID.String()),
				zap.Error(result.Error))
			// Возвращаем результат без walking distances
			return uc.buildResultWithoutWalking(transportStations, lat, lon), nil
		}

		// Возвращаем успешный результат
		return &domain.InfrastructureResult{
			Transport: result.Transport,
		}, nil
	}
}

// buildResultWithoutWalking строит результат без пешеходных расстояний
func (uc *InfrastructureUseCase) buildResultWithoutWalking(
	stations []*domain.TransportStation,
	lat, lon float64,
) *domain.InfrastructureResult {
	transport := make([]domain.TransportWithDistance, 0, len(stations))
	for _, station := range stations {
		linearDist := uc.calculateDistance(lat, lon, station.Lat, station.Lon)
		transport = append(transport, domain.TransportWithDistance{
			StationID:      station.ID,
			Name:           station.Name,
			Type:           station.Type,
			Lat:            station.Lat,
			Lon:            station.Lon,
			LineIDs:        station.LineIDs,
			LinearDistance: linearDist,
		})
	}
	return &domain.InfrastructureResult{
		Transport: transport,
	}
}

// calculateDistance вычисляет расстояние между двумя точками в метрах (формула Haversine)
func (uc *InfrastructureUseCase) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0 // meters

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
