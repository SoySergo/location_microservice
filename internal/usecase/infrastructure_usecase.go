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
			return uc.buildResultWithoutWalking(transportStations, lat, lon, ctx), nil
		}

		// Обогащаем транспорт информацией о линиях
		enrichedTransport := uc.enrichTransportWithLines(ctx, result.Transport)

		// Возвращаем успешный результат
		return &domain.InfrastructureResult{
			Transport: enrichedTransport,
		}, nil
	}
}

// enrichTransportWithLines добавляет информацию о линиях к транспортным станциям
func (uc *InfrastructureUseCase) enrichTransportWithLines(ctx context.Context, transport []domain.TransportWithDistance) []domain.TransportWithDistance {
	for i := range transport {
		lines, err := uc.transportRepo.GetLinesByStationID(ctx, transport[i].StationID)
		if err != nil || len(lines) == 0 {
			continue
		}

		lineInfos := make([]domain.TransportLineInfo, 0, len(lines))
		for _, line := range lines {
			lineInfos = append(lineInfos, domain.TransportLineInfo{
				ID:    line.ID,
				Name:  line.Name,
				Ref:   line.Ref,
				Color: line.Color,
			})
		}
		transport[i].Lines = lineInfos
	}
	return transport
}

// buildResultWithoutWalking строит результат без пешеходных расстояний
func (uc *InfrastructureUseCase) buildResultWithoutWalking(
	stations []*domain.TransportStation,
	lat, lon float64,
	ctx context.Context,
) *domain.InfrastructureResult {
	transport := make([]domain.TransportWithDistance, 0, len(stations))
	for _, station := range stations {
		// Используем дистанцию из БД если есть, иначе вычисляем
		var linearDist float64
		if station.Distance != nil {
			linearDist = *station.Distance
		} else {
			linearDist = uc.calculateDistance(lat, lon, station.Lat, station.Lon)
		}

		// Получаем информацию о линиях
		var lineInfos []domain.TransportLineInfo
		if lines, err := uc.transportRepo.GetLinesByStationID(ctx, station.ID); err == nil && len(lines) > 0 {
			lineInfos = make([]domain.TransportLineInfo, 0, len(lines))
			for _, line := range lines {
				lineInfos = append(lineInfos, domain.TransportLineInfo{
					ID:    line.ID,
					Name:  line.Name,
					Ref:   line.Ref,
					Color: line.Color,
				})
			}
		}

		transport = append(transport, domain.TransportWithDistance{
			StationID:      station.ID,
			Name:           station.Name,
			Type:           station.Type,
			Lat:            station.Lat,
			Lon:            station.Lon,
			Lines:          lineInfos,
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
