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

type TransportUseCase struct {
	transportRepo repository.TransportRepository
	logger        *zap.Logger
}

func NewTransportUseCase(
	transportRepo repository.TransportRepository,
	logger *zap.Logger,
) *TransportUseCase {
	return &TransportUseCase{
		transportRepo: transportRepo,
		logger:        logger,
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
