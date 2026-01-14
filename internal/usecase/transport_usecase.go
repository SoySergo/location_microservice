package usecase

import (
	"context"
	"math"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/pkg/errors"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

type TransportUseCase struct {
	transportRepo   repository.TransportRepository
	logger          *zap.Logger
	defaultRadius   float64 // 1500 m by default
	walkingSpeedMps float64 // 1.39 m/s = ~5 km/h
}

func NewTransportUseCase(
	transportRepo repository.TransportRepository,
	logger *zap.Logger,
) *TransportUseCase {
	return &TransportUseCase{
		transportRepo:   transportRepo,
		logger:          logger,
		defaultRadius:   1500, // 1.5 km
		walkingSpeedMps: 1.39, // ~5 km/h
	}
}

func (uc *TransportUseCase) GetNearestStations(
	ctx context.Context,
	req dto.NearestTransportRequest,
) (*dto.NearestTransportResponse, error) {
	// Validate coordinates
	if !utils.ValidateCoordinates(req.Lat, req.Lon) {
		return nil, errors.ErrInvalidCoordinates
	}

	// Set default max distance if not provided
	if req.MaxDistance == 0 {
		req.MaxDistance = 5000 // 5km default
	}

	// Get nearest stations
	stations, err := uc.transportRepo.GetNearestStations(
		ctx,
		req.Lat,
		req.Lon,
		req.Types,
		req.MaxDistance,
		5, // limit to 5 stations
	)
	if err != nil {
		uc.logger.Error("Failed to get nearest stations", zap.Error(err))
		return nil, err
	}

	// Build response with lines
	result := make([]dto.TransportStationWithLines, 0, len(stations))
	for _, station := range stations {
		// Get lines for this station
		var transportLines []*domain.TransportLine
		if len(station.LineIDs) > 0 {
			lines, err := uc.transportRepo.GetLinesByIDs(ctx, station.LineIDs)
			if err != nil {
				uc.logger.Warn("Failed to get lines for station", zap.Int64("station_id", station.ID))
			} else {
				transportLines = lines
			}
		}

		// Calculate distance
		distance := utils.HaversineDistance(req.Lat, req.Lon, station.Lat, station.Lon) * 1000 // to meters

		// Convert to DTO with string IDs
		result = append(result, dto.ConvertTransportStation(station, transportLines, distance))
	}

	return &dto.NearestTransportResponse{
		Stations: result,
	}, nil
}

// BatchGetNearestStations - пакетный поиск ближайших станций для нескольких точек
func (uc *TransportUseCase) BatchGetNearestStations(
	ctx context.Context,
	req dto.BatchNearestTransportRequest,
) (*dto.BatchNearestTransportResponse, error) {
	// Валидация входных данных
	for i, point := range req.Points {
		if !utils.ValidateCoordinates(point.Lat, point.Lon) {
			uc.logger.Warn("Invalid coordinates in batch request",
				zap.Int("point_index", i),
				zap.Float64("lat", point.Lat),
				zap.Float64("lon", point.Lon))
			return nil, errors.ErrInvalidCoordinates
		}
	}

	// Установка дефолтной дистанции если не указана
	maxDistance := req.MaxDistance
	if maxDistance == 0 {
		maxDistance = 5000 // 5km по умолчанию
	}

	// Структура для хранения результатов
	type indexedResult struct {
		index    int
		stations []dto.TransportStationWithLines
		err      error
	}

	// Канал для результатов
	resultsChan := make(chan indexedResult, len(req.Points))

	// Параллельная обработка каждой точки
	for i, point := range req.Points {
		go func(idx int, pt dto.Point) {
			// Получение ближайших станций
			stations, err := uc.transportRepo.GetNearestStations(
				ctx,
				pt.Lat,
				pt.Lon,
				req.Types,
				maxDistance,
				5, // лимит на 5 станций
			)
			if err != nil {
				uc.logger.Error("Failed to get nearest stations in batch",
					zap.Int("point_index", idx),
					zap.Error(err))
				resultsChan <- indexedResult{index: idx, err: err}
				return
			}

			// Формирование результата с линиями
			result := make([]dto.TransportStationWithLines, 0, len(stations))
			for _, station := range stations {
				// Получение линий для станции
				var transportLines []*domain.TransportLine
				if len(station.LineIDs) > 0 {
					lines, err := uc.transportRepo.GetLinesByIDs(ctx, station.LineIDs)
					if err != nil {
						uc.logger.Warn("Failed to get lines for station in batch",
							zap.Int64("station_id", station.ID),
							zap.Int("point_index", idx))
					} else {
						transportLines = lines
					}
				}

				// Расчет расстояния
				distance := utils.HaversineDistance(pt.Lat, pt.Lon, station.Lat, station.Lon) * 1000 // в метры

				// Convert to DTO with string IDs
				result = append(result, dto.ConvertTransportStation(station, transportLines, distance))
			}

			resultsChan <- indexedResult{index: idx, stations: result}
		}(i, point)
	}

	// Сбор результатов
	results := make([][]dto.TransportStationWithLines, len(req.Points))
	for i := 0; i < len(req.Points); i++ {
		res := <-resultsChan
		if res.err != nil {
			// Возвращаем пустой массив для точек с ошибкой
			results[res.index] = []dto.TransportStationWithLines{}
		} else {
			results[res.index] = res.stations
		}
	}
	close(resultsChan)

	return &dto.BatchNearestTransportResponse{
		Results: results,
	}, nil
}

// GetTransportTileByTypes возвращает MVT тайл с транспортом с фильтрацией по типам
func (uc *TransportUseCase) GetTransportTileByTypes(ctx context.Context, z, x, y int, types []string) ([]byte, error) {
	// Валидация zoom level
	if z < 0 || z > 18 {
		return nil, errors.ErrInvalidZoom
	}

	// Валидация типов транспорта
	if len(types) > 0 {
		for _, t := range types {
			if !domain.IsValidTransportType(t) {
				uc.logger.Warn("Invalid transport type", zap.String("type", t))
				return nil, errors.ErrInvalidTransportType
			}
		}
	}

	// Получаем тайл из репозитория
	tile, err := uc.transportRepo.GetTransportTileByTypes(ctx, z, x, y, types)
	if err != nil {
		uc.logger.Error("Failed to get transport tile by types",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Strings("types", types),
			zap.Error(err))
		return nil, err
	}

	return tile, nil
}

// GetLinesByStationID возвращает линии для станции (для hover логики)
func (uc *TransportUseCase) GetLinesByStationID(ctx context.Context, stationID int64) ([]*domain.TransportLine, error) {
	lines, err := uc.transportRepo.GetLinesByStationID(ctx, stationID)
	if err != nil {
		uc.logger.Error("Failed to get lines by station ID",
			zap.Int64("station_id", stationID),
			zap.Error(err))
		return nil, err
	}

	return lines, nil
}

// GetNearestTransportByPriority возвращает ближайший транспорт с приоритетом по типу и расстоянию.
// Приоритет: metro/train -> bus/tram (если нет высокоприоритетного в радиусе).
func (uc *TransportUseCase) GetNearestTransportByPriority(
	ctx context.Context,
	req dto.PriorityTransportRequest,
) (*dto.PriorityTransportResponse, error) {
	// Validate coordinates
	if !utils.ValidateCoordinates(req.Lat, req.Lon) {
		return nil, errors.ErrInvalidCoordinates
	}

	// Set defaults
	radius := req.Radius
	if radius == 0 {
		radius = uc.defaultRadius
	}

	limit := req.Limit
	if limit == 0 {
		limit = 5
	}

	uc.logger.Info("GetNearestTransportByPriority",
		zap.Float64("lat", req.Lat),
		zap.Float64("lon", req.Lon),
		zap.Float64("radius", radius),
		zap.Int("limit", limit))

	// Получаем станции с приоритетом
	stations, err := uc.transportRepo.GetNearestTransportByPriority(ctx, req.Lat, req.Lon, radius, limit)
	if err != nil {
		uc.logger.Error("Failed to get priority transport", zap.Error(err))
		return nil, err
	}

	// Определяем тип приоритета
	hasHighPriority := false
	priorityType := "bus/tram"
	for _, s := range stations {
		if s.Type == "metro" || s.Type == "train" {
			hasHighPriority = true
			priorityType = "metro/train"
			break
		}
	}

	// Преобразуем в DTO с расчётом времени ходьбы
	walkingSpeedKmH := uc.walkingSpeedMps * 3.6 // м/с -> км/ч
	result := make([]dto.PriorityTransportStation, 0, len(stations))

	for _, s := range stations {
		// Примерное время пешком (манхэттенское расстояние + 20%)
		walkingDistance := s.Distance * 1.2
		walkingTime := walkingDistance / uc.walkingSpeedMps / 60 // в минутах

		// Преобразуем линии
		lines := make([]dto.TransportLineInfoEnriched, 0, len(s.Lines))
		for _, line := range s.Lines {
			lines = append(lines, dto.TransportLineInfoEnriched{
				ID:    line.ID,
				Name:  line.Name,
				Ref:   line.Ref,
				Type:  line.Type,
				Color: line.Color,
			})
		}

		result = append(result, dto.PriorityTransportStation{
			StationID:       s.StationID,
			Name:            s.Name,
			NameEn:          s.NameEn,
			Type:            s.Type,
			Lat:             s.Lat,
			Lon:             s.Lon,
			LinearDistance:  math.Round(s.Distance*100) / 100,
			WalkingDistance: math.Round(walkingDistance*100) / 100,
			WalkingTime:     math.Round(walkingTime*10) / 10,
			Lines:           lines,
		})
	}

	return &dto.PriorityTransportResponse{
		Stations: result,
		Meta: dto.PriorityTransportMeta{
			TotalFound:      len(result),
			SearchPoint:     dto.Point{Lat: req.Lat, Lon: req.Lon},
			RadiusM:         radius,
			HasHighPriority: hasHighPriority,
			PriorityType:    priorityType,
			WalkingSpeedKmH: walkingSpeedKmH,
		},
	}, nil
}

// GetNearestTransportByPriorityBatch возвращает ближайший транспорт с приоритетом
// для множества точек одним эффективным запросом к БД.
func (uc *TransportUseCase) GetNearestTransportByPriorityBatch(
	ctx context.Context,
	req dto.PriorityTransportBatchRequest,
) (*dto.PriorityTransportBatchResponse, error) {
	// Validate request
	if len(req.Points) == 0 {
		return nil, errors.ErrInvalidRequest
	}

	for _, p := range req.Points {
		if !utils.ValidateCoordinates(p.Lat, p.Lon) {
			return nil, errors.ErrInvalidCoordinates
		}
	}

	// Set defaults
	radius := req.Radius
	if radius == 0 {
		radius = uc.defaultRadius
	}

	limit := req.Limit
	if limit == 0 {
		limit = 3
	}

	uc.logger.Info("GetNearestTransportByPriorityBatch",
		zap.Int("points_count", len(req.Points)),
		zap.Float64("radius", radius),
		zap.Int("limit_per_point", limit))

	// Преобразуем DTO в domain
	domainPoints := make([]domain.TransportSearchPoint, len(req.Points))
	for i, p := range req.Points {
		domainPoints[i] = domain.TransportSearchPoint{
			Lat:   p.Lat,
			Lon:   p.Lon,
			Limit: limit,
		}
	}

	// Получаем станции одним запросом
	batchResults, err := uc.transportRepo.GetNearestTransportByPriorityBatch(ctx, domainPoints, radius, limit)
	if err != nil {
		uc.logger.Error("Failed to get batch priority transport", zap.Error(err))
		return nil, err
	}

	// Преобразуем в DTO с расчётом времени ходьбы
	walkingSpeedKmH := uc.walkingSpeedMps * 3.6
	results := make([]dto.PriorityTransportPointResult, len(batchResults))
	totalStations := 0

	for i, br := range batchResults {
		stations := make([]dto.PriorityTransportStation, 0, len(br.Stations))

		for _, s := range br.Stations {
			walkingDistance := s.Distance * 1.2
			walkingTime := walkingDistance / uc.walkingSpeedMps / 60

			lines := make([]dto.TransportLineInfoEnriched, 0, len(s.Lines))
			for _, line := range s.Lines {
				lines = append(lines, dto.TransportLineInfoEnriched{
					ID:    line.ID,
					Name:  line.Name,
					Ref:   line.Ref,
					Type:  line.Type,
					Color: line.Color,
				})
			}

			stations = append(stations, dto.PriorityTransportStation{
				StationID:       s.StationID,
				Name:            s.Name,
				NameEn:          s.NameEn,
				Type:            s.Type,
				Lat:             s.Lat,
				Lon:             s.Lon,
				LinearDistance:  math.Round(s.Distance*100) / 100,
				WalkingDistance: math.Round(walkingDistance*100) / 100,
				WalkingTime:     math.Round(walkingTime*10) / 10,
				Lines:           lines,
			})
		}

		results[i] = dto.PriorityTransportPointResult{
			PointIndex:  br.PointIndex,
			SearchPoint: dto.Point{Lat: br.SearchPoint.Lat, Lon: br.SearchPoint.Lon},
			Stations:    stations,
		}
		totalStations += len(stations)
	}

	return &dto.PriorityTransportBatchResponse{
		Results: results,
		Meta: dto.PriorityTransportBatchMeta{
			TotalPoints:     len(req.Points),
			TotalStations:   totalStations,
			RadiusM:         radius,
			WalkingSpeedKmH: walkingSpeedKmH,
		},
	}, nil
}
