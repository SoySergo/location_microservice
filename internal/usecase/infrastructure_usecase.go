package usecase

import (
	"context"
	"fmt"
	"math"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"go.uber.org/zap"
)

// InfrastructureUseCase - use case для работы с инфраструктурой
type InfrastructureUseCase struct {
	infraRepo  repository.InfrastructureRepository
	mapboxRepo repository.MapboxRepository
	logger     *zap.Logger
	maxMetro   int
	maxTrain   int
	maxTram    int
	maxBus     int
	poiRadius  float64
}

// NewInfrastructureUseCase создает новый InfrastructureUseCase
func NewInfrastructureUseCase(
	infraRepo repository.InfrastructureRepository,
	mapboxRepo repository.MapboxRepository,
	logger *zap.Logger,
	maxMetro, maxTrain, maxTram, maxBus int,
	poiRadius float64,
) *InfrastructureUseCase {
	return &InfrastructureUseCase{
		infraRepo:  infraRepo,
		mapboxRepo: mapboxRepo,
		logger:     logger,
		maxMetro:   maxMetro,
		maxTrain:   maxTrain,
		maxTram:    maxTram,
		maxBus:     maxBus,
		poiRadius:  poiRadius,
	}
}

// GetInfrastructure получает инфраструктуру для локации
func (uc *InfrastructureUseCase) GetInfrastructure(
	ctx context.Context,
	lat, lon float64,
	transportRadius float64,
) (*domain.InfrastructureResult, error) {
	// 1. Получаем транспорт с приоритетами
	transportPriorities := []domain.TransportPriority{
		{Type: "metro", Limit: uc.maxMetro},
		{Type: "train", Limit: uc.maxTrain},
		{Type: "tram", Limit: uc.maxTram},
		{Type: "bus", Limit: uc.maxBus},
	}

	transportStations, err := uc.infraRepo.GetNearestTransportGrouped(
		ctx, lat, lon, transportPriorities, transportRadius,
	)
	if err != nil {
		uc.logger.Error("Failed to get transport", zap.Error(err))
		return nil, fmt.Errorf("failed to get transport: %w", err)
	}

	// 2. Получаем POI с учетом категорий
	poiCategories := []domain.POICategoryConfig{
		{Category: "shop", Subcategory: "supermarket", Limit: 3},
		{Category: "shop", Subcategory: "convenience", Limit: 2},
		{Category: "amenity", Subcategory: "pharmacy", Limit: 2},
		{Category: "amenity", Subcategory: "hospital", Limit: 2},
		{Category: "amenity", Subcategory: "school", Limit: 3},
		{Category: "amenity", Subcategory: "kindergarten", Limit: 2},
		{Category: "leisure", Subcategory: "park", Limit: 2},
		{Category: "leisure", Subcategory: "playground", Limit: 1},
	}

	pois, err := uc.infraRepo.GetNearestPOIs(ctx, lat, lon, poiCategories, uc.poiRadius)
	if err != nil {
		uc.logger.Error("Failed to get POIs", zap.Error(err))
		// Не критичная ошибка, продолжаем
		pois = []*domain.POI{}
	}

	// 3. Балансируем точки для Mapbox (до 24 destinations + 1 source = 25 total)
	maxMapboxDestinations := 24
	transportCount := len(transportStations)
	poisCount := len(pois)

	// Приоритет транспорту: до 9 станций
	maxTransportForMatrix := 9
	if transportCount > maxTransportForMatrix {
		transportCount = maxTransportForMatrix
	}

	// Остаток для POI
	maxPOIsForMatrix := maxMapboxDestinations - transportCount
	if poisCount > maxPOIsForMatrix {
		poisCount = maxPOIsForMatrix
	}

	uc.logger.Debug("Balancing points for Mapbox",
		zap.Int("transport_count", transportCount),
		zap.Int("pois_count", poisCount),
		zap.Int("total_destinations", transportCount+poisCount))

	// 4. Вычисляем пешеходные расстояния через Mapbox
	var result domain.InfrastructureResult

	if transportCount+poisCount > 0 {
		// Создаем origin (адрес объекта)
		origins := []domain.Coordinate{{Lat: lat, Lon: lon}}

		// Создаем destinations (транспорт + POI)
		var destinations []domain.Coordinate
		var destinationIDs []string // для мапинга результатов

		// Добавляем транспорт
		for i := 0; i < transportCount; i++ {
			station := transportStations[i]
			destinations = append(destinations, domain.Coordinate{
				Lat: station.Lat,
				Lon: station.Lon,
			})
			destinationIDs = append(destinationIDs, fmt.Sprintf("transport_%d", station.ID))
		}

		// Добавляем POI
		for i := 0; i < poisCount; i++ {
			poi := pois[i]
			destinations = append(destinations, domain.Coordinate{
				Lat: poi.Lat,
				Lon: poi.Lon,
			})
			destinationIDs = append(destinationIDs, fmt.Sprintf("poi_%d", poi.ID))
		}

		// Вызываем Mapbox Matrix API
		matrixResp, err := uc.mapboxRepo.GetWalkingMatrix(ctx, origins, destinations)
		if err != nil {
			uc.logger.Error("Failed to get walking matrix", zap.Error(err))
			// Не критичная ошибка, возвращаем данные без пешеходных расстояний
		} else {
			// Обрабатываем результаты
			result.WalkingDistances = make(map[string]float64)

			if len(matrixResp.Distances) > 0 && len(matrixResp.Distances[0]) == len(destinations) {
				for i, destID := range destinationIDs {
					distance := matrixResp.Distances[0][i]
					duration := matrixResp.Durations[0][i]
					result.WalkingDistances[destID] = distance

					// Обновляем транспорт с пешеходными расстояниями
					if i < transportCount {
						station := transportStations[i]
						linearDist := uc.calculateDistance(lat, lon, station.Lat, station.Lon)
						result.Transport = append(result.Transport, domain.TransportWithDistance{
							StationID:       station.ID,
							Name:            station.Name,
							Type:            station.Type,
							Lat:             station.Lat,
							Lon:             station.Lon,
							LineIDs:         station.LineIDs,
							LinearDistance:  linearDist,
							WalkingDistance: &distance,
							WalkingDuration: &duration,
						})
					}

					// Обновляем POI с пешеходными расстояниями
					if i >= transportCount && i-transportCount < poisCount {
						poi := pois[i-transportCount]
						linearDist := uc.calculateDistance(lat, lon, poi.Lat, poi.Lon)
						result.POIs = append(result.POIs, domain.POIWithDistance{
							ID:              poi.ID,
							Name:            poi.Name,
							Category:        poi.Category,
							Subcategory:     poi.Subcategory,
							Lat:             poi.Lat,
							Lon:             poi.Lon,
							LinearDistance:  linearDist,
							WalkingDistance: &distance,
							WalkingDuration: &duration,
						})
					}
				}
			}
		}
	}

	// Если Mapbox не сработал, добавляем без пешеходных расстояний
	if len(result.Transport) == 0 && len(transportStations) > 0 {
		for _, station := range transportStations {
			linearDist := uc.calculateDistance(lat, lon, station.Lat, station.Lon)
			result.Transport = append(result.Transport, domain.TransportWithDistance{
				StationID:      station.ID,
				Name:           station.Name,
				Type:           station.Type,
				Lat:            station.Lat,
				Lon:            station.Lon,
				LineIDs:        station.LineIDs,
				LinearDistance: linearDist,
			})
		}
	}

	if len(result.POIs) == 0 && len(pois) > 0 {
		for _, poi := range pois {
			linearDist := uc.calculateDistance(lat, lon, poi.Lat, poi.Lon)
			result.POIs = append(result.POIs, domain.POIWithDistance{
				ID:             poi.ID,
				Name:           poi.Name,
				Category:       poi.Category,
				Subcategory:    poi.Subcategory,
				Lat:            poi.Lat,
				Lon:            poi.Lon,
				LinearDistance: linearDist,
			})
		}
	}

	return &result, nil
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
