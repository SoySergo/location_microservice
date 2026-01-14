package usecase

import (
	"context"
	"math"
	"sort"
	"sync"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/pkg/errors"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

// EnrichmentDebugUseCase - usecase для тестирования/дебага логики обогащения
type EnrichmentDebugUseCase struct {
	transportRepo   repository.TransportRepository
	enrichmentUC    *EnrichmentUseCase
	logger          *zap.Logger
	defaultRadius   float64 // в метрах
	walkingSpeedMps float64 // скорость ходьбы в м/с (по умолчанию ~5 км/ч = 1.39 м/с)
}

// NewEnrichmentDebugUseCase создает новый EnrichmentDebugUseCase
func NewEnrichmentDebugUseCase(
	transportRepo repository.TransportRepository,
	enrichmentUC *EnrichmentUseCase,
	logger *zap.Logger,
) *EnrichmentDebugUseCase {
	return &EnrichmentDebugUseCase{
		transportRepo:   transportRepo,
		enrichmentUC:    enrichmentUC,
		logger:          logger,
		defaultRadius:   1500, // 1.5 km по умолчанию
		walkingSpeedMps: 1.39, // ~5 km/h
	}
}

// GetNearestTransportEnriched возвращает ближайшие станции транспорта
// в формате, который возвращает воркер обогащения
func (uc *EnrichmentDebugUseCase) GetNearestTransportEnriched(
	ctx context.Context,
	req dto.EnrichmentDebugTransportRequest,
) (*dto.EnrichmentDebugTransportResponse, error) {
	// Validate coordinates
	if req.Lat < -90 || req.Lat > 90 || req.Lon < -180 || req.Lon > 180 {
		return nil, errors.ErrInvalidCoordinates
	}

	// Set defaults
	radius := req.MaxDistance
	if radius == 0 {
		radius = uc.defaultRadius
	}

	limit := req.Limit
	if limit == 0 {
		limit = 10
	}

	// Определяем типы транспорта для поиска
	transportTypes := req.Types
	if len(transportTypes) == 0 {
		transportTypes = []string{"metro", "train", "tram", "bus"}
	}

	uc.logger.Info("GetNearestTransportEnriched",
		zap.Float64("lat", req.Lat),
		zap.Float64("lon", req.Lon),
		zap.Float64("radius", radius),
		zap.Strings("types", transportTypes),
		zap.Int("limit", limit))

	// Получаем станции с группировкой (убираем дубликаты выходов)
	priorities := make([]domain.TransportPriority, len(transportTypes))
	for i, t := range transportTypes {
		priorities[i] = domain.TransportPriority{
			Type:  t,
			Limit: limit,
		}
	}

	uc.logger.Debug("Fetching nearest stations with priorities", zap.Any("priorities", priorities))
	stations, err := uc.transportRepo.GetNearestStationsGrouped(
		ctx, req.Lat, req.Lon, priorities, radius,
	)
	if err != nil {
		uc.logger.Error("Failed to get nearest stations", zap.Error(err))
		return nil, err
	}
	uc.logger.Debug("Fetched stations", zap.Int("count", len(stations)))

	// Преобразуем в формат ответа с линиями и расстояниями
	result := make([]dto.EnrichedTransportStation, 0, len(stations))
	seenStations := make(map[string]bool) // для дополнительной дедупликации по имени

	uc.logger.Debug("Enriching stations with distances and lines")
	for _, station := range stations {
		// Дополнительная дедупликация по нормализованному имени
		normalizedName := normalizeStationName(station.Name)
		if seenStations[normalizedName] {
			continue
		}
		seenStations[normalizedName] = true

		// Вычисляем расстояние
		var distance float64
		if station.Distance != nil {
			distance = *station.Distance
		} else {
			distance = haversineDistance(req.Lat, req.Lon, station.Lat, station.Lon)
		}

		// Вычисляем примерное время пешком (манхэттенское расстояние + 20% на реальный маршрут)
		walkingDistance := distance * 1.2                        // примерная корректировка
		walkingTime := walkingDistance / uc.walkingSpeedMps / 60 // в минутах

		uc.logger.Debug("Processing station",
			zap.Int64("station_id", station.ID),
			zap.String("name", station.Name),
			zap.Float64("linear_distance", distance),
			zap.Float64("walking_distance", walkingDistance),
			zap.Float64("walking_time_min", walkingTime),
		)
		// Получаем линии для станции
		lines, err := uc.transportRepo.GetLinesByStationID(ctx, station.ID)

		uc.logger.Debug("Fetched lines for station",
			zap.Int64("station_id", station.ID),
			zap.Int("line_count", len(lines)),
		)

		// Преобразуем линии в формат ответа
		var lineInfos []dto.TransportLineInfoEnriched
		if err == nil && len(lines) > 0 {
			// Дедупликация линий по ref
			seenRefs := make(map[string]bool)
			lineInfos = make([]dto.TransportLineInfoEnriched, 0, len(lines))
			for _, line := range lines {
				// Пропускаем дубликаты линий (например L3 туда и обратно)
				if line.Ref != "" && seenRefs[line.Ref] {
					continue
				}
				if line.Ref != "" {
					seenRefs[line.Ref] = true
				}

				lineInfos = append(lineInfos, dto.TransportLineInfoEnriched{
					ID:    line.ID,
					Name:  line.Name,
					Ref:   line.Ref,
					Type:  line.Type,
					Color: line.Color,
				})
			}
		}

		uc.logger.Debug("Adding station to result",
			zap.Int64("station_id", station.ID),
			zap.String("name", station.Name),
			zap.Int("line_count", len(lineInfos)),
		)

		result = append(result, dto.EnrichedTransportStation{
			StationID:       station.ID,
			Name:            station.Name,
			Type:            station.Type,
			Lat:             station.Lat,
			Lon:             station.Lon,
			LinearDistance:  math.Round(distance*100) / 100,
			WalkingDistance: math.Round(walkingDistance*100) / 100,
			WalkingTime:     math.Round(walkingTime*10) / 10, // округляем до 0.1 минуты
			Lines:           lineInfos,
		})
	}

	uc.logger.Debug("Sorting and limiting results", zap.Int("total_stations", len(result)))
	// Сортируем по расстоянию
	sort.Slice(result, func(i, j int) bool {
		return result[i].LinearDistance < result[j].LinearDistance
	})

	uc.logger.Debug("Applying limit to results", zap.Int("limit", limit))

	// Обрезаем до лимита
	if len(result) > limit {
		result = result[:limit]
	}

	uc.logger.Info("GetNearestTransportEnriched completed", zap.Int("returned_stations", len(result)))

	return &dto.EnrichmentDebugTransportResponse{
		Stations: result,
		Meta: dto.EnrichmentDebugMeta{
			TotalFound:  len(result),
			SearchPoint: dto.Point{Lat: req.Lat, Lon: req.Lon},
			RadiusM:     radius,
			Types:       transportTypes,
		},
	}, nil
}

// haversineDistance вычисляет расстояние между двумя точками в метрах
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
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

// normalizeStationName нормализует название станции для дедупликации
func normalizeStationName(name string) string {
	// Простая нормализация - можно расширить
	result := ""
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= 'а' && r <= 'я') || (r >= 'А' && r <= 'Я') ||
			(r >= '0' && r <= '9') {
			result += string(r)
		}
	}
	return result
}

// GetNearestTransportEnrichedBatch возвращает ближайшие станции транспорта
// для пачки координат одним эффективным запросом к БД
func (uc *EnrichmentDebugUseCase) GetNearestTransportEnrichedBatch(
	ctx context.Context,
	req dto.EnrichmentDebugTransportBatchRequest,
) (*dto.EnrichmentDebugTransportBatchResponse, error) {
	// Validate request
	if len(req.Points) == 0 {
		return nil, errors.ErrInvalidRequest
	}

	for _, p := range req.Points {
		if p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 {
			return nil, errors.ErrInvalidCoordinates
		}
	}

	// Set defaults
	maxDistance := req.MaxDistance
	if maxDistance == 0 {
		maxDistance = uc.defaultRadius
	}

	// Преобразуем DTO в domain request
	domainPoints := make([]domain.TransportSearchPoint, len(req.Points))
	for i, p := range req.Points {
		types := p.Types
		if len(types) == 0 {
			types = []string{"metro"}
		}
		limit := p.Limit
		if limit == 0 {
			limit = 3
		}
		domainPoints[i] = domain.TransportSearchPoint{
			Lat:   p.Lat,
			Lon:   p.Lon,
			Types: types,
			Limit: limit,
		}
	}

	domainReq := domain.BatchTransportRequest{
		Points:      domainPoints,
		MaxDistance: maxDistance,
	}

	uc.logger.Info("GetNearestTransportEnrichedBatch",
		zap.Int("points_count", len(req.Points)),
		zap.Float64("max_distance", maxDistance))

	// Шаг 1: Получаем станции (без линий)
	stations, err := uc.transportRepo.GetNearestStationsBatch(ctx, domainReq)
	if err != nil {
		uc.logger.Error("Failed to get batch stations", zap.Error(err))
		return nil, err
	}

	uc.logger.Debug("Fetched batch stations", zap.Int("count", len(stations)))

	// Шаг 2: Собираем уникальные ID станций для запроса линий
	stationIDSet := make(map[int64]bool)
	var stationIDs []int64
	for _, s := range stations {
		if !stationIDSet[s.StationID] {
			stationIDs = append(stationIDs, s.StationID)
			stationIDSet[s.StationID] = true
		}
	}

	// Шаг 3: Получаем линии для всех станций одним запросом
	var linesMap map[int64][]domain.TransportLineInfo
	if len(stationIDs) > 0 {
		linesMap, err = uc.transportRepo.GetLinesByStationIDsBatch(ctx, stationIDs)
		if err != nil {
			uc.logger.Warn("Failed to get lines for batch stations, continuing without lines", zap.Error(err))
			linesMap = make(map[int64][]domain.TransportLineInfo)
		}
		uc.logger.Debug("Fetched lines for stations", zap.Int("stations_with_lines", len(linesMap)))
	}

	// Группируем результаты по point_idx и вычисляем расстояния в слое usecase
	resultsByPoint := make(map[int][]dto.EnrichedTransportStation)
	totalStations := 0

	for _, station := range stations {
		// Находим исходную точку для расчета расстояния
		if station.PointIdx >= len(req.Points) {
			continue
		}
		searchPoint := req.Points[station.PointIdx]

		// Вычисляем расстояния (distance уже пришла из БД)
		linearDistance := station.Distance
		walkingDistance := linearDistance * 1.2                  // примерная корректировка
		walkingTime := walkingDistance / uc.walkingSpeedMps / 60 // в минутах

		// Преобразуем линии (получаем из linesMap по station ID)
		var lineInfos []dto.TransportLineInfoEnriched
		if lines, ok := linesMap[station.StationID]; ok {
			for _, line := range lines {
				lineInfos = append(lineInfos, dto.TransportLineInfoEnriched{
					ID:    line.ID,
					Name:  line.Name,
					Ref:   line.Ref,
					Type:  line.Type,
					Color: line.Color,
				})
			}
		}

		enrichedStation := dto.EnrichedTransportStation{
			StationID:       station.StationID,
			Name:            station.Name,
			Type:            station.Type,
			Lat:             station.Lat,
			Lon:             station.Lon,
			LinearDistance:  math.Round(linearDistance*100) / 100,
			WalkingDistance: math.Round(walkingDistance*100) / 100,
			WalkingTime:     math.Round(walkingTime*10) / 10,
			Lines:           lineInfos,
		}

		resultsByPoint[station.PointIdx] = append(resultsByPoint[station.PointIdx], enrichedStation)
		totalStations++

		uc.logger.Debug("Processing batch station",
			zap.Int("point_idx", station.PointIdx),
			zap.Int64("station_id", station.StationID),
			zap.String("name", station.Name),
			zap.Float64("lat", searchPoint.Lat),
			zap.Float64("lon", searchPoint.Lon),
		)
	}

	// Формируем ответ с сортировкой по расстоянию для каждой точки
	results := make([]dto.PointTransportResult, len(req.Points))
	for i, p := range req.Points {
		stationsForPoint := resultsByPoint[i]

		// Сортируем станции по расстоянию
		sort.Slice(stationsForPoint, func(a, b int) bool {
			return stationsForPoint[a].LinearDistance < stationsForPoint[b].LinearDistance
		})

		results[i] = dto.PointTransportResult{
			PointIndex:  i,
			SearchPoint: dto.Point{Lat: p.Lat, Lon: p.Lon},
			Stations:    stationsForPoint,
		}
	}

	uc.logger.Info("GetNearestTransportEnrichedBatch completed",
		zap.Int("total_points", len(req.Points)),
		zap.Int("total_stations", totalStations))

	return &dto.EnrichmentDebugTransportBatchResponse{
		Results: results,
		Meta: dto.EnrichmentDebugBatchMeta{
			TotalPoints:   len(req.Points),
			TotalStations: totalStations,
			RadiusM:       maxDistance,
		},
	}, nil
}

// EnrichLocation тестирует полное обогащение локации через EnrichmentUseCase
func (uc *EnrichmentDebugUseCase) EnrichLocation(
	ctx context.Context,
	req dto.EnrichmentDebugLocationRequest,
) (*dto.EnrichmentDebugLocationResponse, error) {
	// Преобразуем DTO в domain event
	event := &domain.LocationEnrichEvent{
		Country:      req.Country,
		Region:       req.Region,
		Province:     req.Province,
		City:         req.City,
		District:     req.District,
		Neighborhood: req.Neighborhood,
		Street:       req.Street,
		HouseNumber:  req.HouseNumber,
		PostalCode:   req.PostalCode,
		Latitude:     req.Latitude,
		Longitude:    req.Longitude,
	}

	uc.logger.Info("EnrichLocation debug request",
		zap.String("country", req.Country),
		zap.Stringp("city", req.City),
		zap.Float64p("lat", req.Latitude),
		zap.Float64p("lon", req.Longitude))

	// Вызываем основной usecase обогащения
	result, err := uc.enrichmentUC.EnrichLocation(ctx, event)
	if err != nil {
		uc.logger.Error("EnrichLocation failed",
			zap.String("country", req.Country),
			zap.Error(err))
		return nil, err
	}

	// Преобразуем результат в DTO
	response := &dto.EnrichmentDebugLocationResponse{
		Error: result.Error,
	}

	if result.EnrichedLocation != nil {
		response.EnrichedLocation = &dto.EnrichedLocationDTO{
			IsAddressVisible: result.EnrichedLocation.IsAddressVisible,
		}

		if result.EnrichedLocation.Country != nil {
			response.EnrichedLocation.Country = &dto.BoundaryInfoDTO{
				ID:             result.EnrichedLocation.Country.ID,
				Name:           result.EnrichedLocation.Country.Name,
				TranslateNames: result.EnrichedLocation.Country.TranslateNames,
			}
		}
		if result.EnrichedLocation.Region != nil {
			response.EnrichedLocation.Region = &dto.BoundaryInfoDTO{
				ID:             result.EnrichedLocation.Region.ID,
				Name:           result.EnrichedLocation.Region.Name,
				TranslateNames: result.EnrichedLocation.Region.TranslateNames,
			}
		}
		if result.EnrichedLocation.Province != nil {
			response.EnrichedLocation.Province = &dto.BoundaryInfoDTO{
				ID:             result.EnrichedLocation.Province.ID,
				Name:           result.EnrichedLocation.Province.Name,
				TranslateNames: result.EnrichedLocation.Province.TranslateNames,
			}
		}
		if result.EnrichedLocation.City != nil {
			response.EnrichedLocation.City = &dto.BoundaryInfoDTO{
				ID:             result.EnrichedLocation.City.ID,
				Name:           result.EnrichedLocation.City.Name,
				TranslateNames: result.EnrichedLocation.City.TranslateNames,
			}
		}
		if result.EnrichedLocation.District != nil {
			response.EnrichedLocation.District = &dto.BoundaryInfoDTO{
				ID:             result.EnrichedLocation.District.ID,
				Name:           result.EnrichedLocation.District.Name,
				TranslateNames: result.EnrichedLocation.District.TranslateNames,
			}
		}
		if result.EnrichedLocation.Neighborhood != nil {
			response.EnrichedLocation.Neighborhood = &dto.BoundaryInfoDTO{
				ID:             result.EnrichedLocation.Neighborhood.ID,
				Name:           result.EnrichedLocation.Neighborhood.Name,
				TranslateNames: result.EnrichedLocation.Neighborhood.TranslateNames,
			}
		}
	}

	if len(result.NearestTransport) > 0 {
		response.NearestTransport = make([]dto.NearestStationDTO, 0, len(result.NearestTransport))
		for _, station := range result.NearestTransport {
			stationDTO := dto.NearestStationDTO{
				StationID: station.StationID,
				Name:      station.Name,
				Type:      station.Type,
				Lat:       station.Lat,
				Lon:       station.Lon,
				Distance:  math.Round(station.Distance*100) / 100,
			}

			if len(station.Lines) > 0 {
				stationDTO.Lines = make([]dto.TransportLineInfoEnriched, 0, len(station.Lines))
				for _, line := range station.Lines {
					stationDTO.Lines = append(stationDTO.Lines, dto.TransportLineInfoEnriched{
						ID:    line.ID,
						Name:  line.Name,
						Ref:   line.Ref,
						Type:  line.Type,
						Color: line.Color,
					})
				}
			}

			response.NearestTransport = append(response.NearestTransport, stationDTO)
		}
	}

	uc.logger.Info("EnrichLocation debug completed",
		zap.Bool("has_enriched_location", response.EnrichedLocation != nil),
		zap.Int("nearest_transport_count", len(response.NearestTransport)),
		zap.String("error", response.Error))

	return response, nil
}

// EnrichLocationBatch обогащает пачку локаций эффективно (2 параллельных запроса в БД)
// Логика:
// 1. Делим локации на 2 группы: visible (есть координаты) и name-based (нет координат или не visible)
// 2. Запускаем параллельно 2 запроса в БД:
//   - GetByPointBatch для visible локаций (reverse geocoding по координатам)
//   - SearchByTextBatch для name-based локаций (поиск по названиям)
//
// 3. Объединяем результаты и возвращаем
func (uc *EnrichmentDebugUseCase) EnrichLocationBatch(
	ctx context.Context,
	req dto.EnrichmentDebugLocationBatchRequest,
) (*dto.EnrichmentDebugLocationBatchResponse, error) {
	if len(req.Locations) == 0 {
		return nil, errors.ErrInvalidRequest
	}

	uc.logger.Info("EnrichLocationBatch started",
		zap.Int("total_locations", len(req.Locations)))

	// Шаг 1: Разделяем локации на 2 группы
	var visibleLocations []dto.LocationInput   // с координатами и visible=true
	var nameBasedLocations []dto.LocationInput // без координат или visible=false

	for _, loc := range req.Locations {
		hasCoords := loc.Latitude != nil && loc.Longitude != nil
		isVisible := loc.IsVisible != nil && *loc.IsVisible

		// Если есть координаты И visible=true -> reverse geocoding
		// Иначе -> поиск по названиям
		if hasCoords && isVisible {
			visibleLocations = append(visibleLocations, loc)
		} else {
			nameBasedLocations = append(nameBasedLocations, loc)
		}
	}

	uc.logger.Debug("Locations split",
		zap.Int("visible_count", len(visibleLocations)),
		zap.Int("name_based_count", len(nameBasedLocations)))

	// Шаг 2: Запускаем параллельно 2 запроса к БД
	type batchResult struct {
		visibleResults   map[int][]*domain.AdminBoundary
		nameBasedResults []domain.BoundarySearchResult
		visibleErr       error
		nameBasedErr     error
	}

	resultChan := make(chan batchResult, 1)

	go func() {
		var wg sync.WaitGroup
		var result batchResult

		// Горутина для visible локаций (reverse geocoding по координатам)
		if len(visibleLocations) > 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()

				points := make([]domain.LatLon, len(visibleLocations))
				for i, loc := range visibleLocations {
					points[i] = domain.LatLon{Lat: *loc.Latitude, Lon: *loc.Longitude}
				}

				boundariesByPoint, err := uc.enrichmentUC.boundaryRepo.GetByPointBatch(ctx, points)
				if err != nil {
					uc.logger.Error("GetByPointBatch failed", zap.Error(err))
					result.visibleErr = err
					return
				}
				result.visibleResults = boundariesByPoint
			}()
		}

		// Горутина для name-based локаций (поиск по названиям)
		if len(nameBasedLocations) > 0 {
			wg.Add(1)
			go func() {
				defer wg.Done()

				// Собираем все запросы на поиск
				var searchRequests []domain.BoundarySearchRequest

				for _, loc := range nameBasedLocations {
					// Добавляем запросы для каждого уровня иерархии, который есть
					// Приоритет: от детального к общему
					if loc.Neighborhood != nil && *loc.Neighborhood != "" {
						searchRequests = append(searchRequests, domain.BoundarySearchRequest{
							Index:      loc.Index*100 + 10, // уникальный ID
							Name:       *loc.Neighborhood,
							AdminLevel: 10,
						})
					}
					if loc.District != nil && *loc.District != "" {
						searchRequests = append(searchRequests, domain.BoundarySearchRequest{
							Index:      loc.Index*100 + 9,
							Name:       *loc.District,
							AdminLevel: 9,
						})
					}
					if loc.City != nil && *loc.City != "" {
						searchRequests = append(searchRequests, domain.BoundarySearchRequest{
							Index:      loc.Index*100 + 8,
							Name:       *loc.City,
							AdminLevel: 8,
						})
					}
					if loc.Province != nil && *loc.Province != "" {
						searchRequests = append(searchRequests, domain.BoundarySearchRequest{
							Index:      loc.Index*100 + 6,
							Name:       *loc.Province,
							AdminLevel: 6,
						})
					}
					if loc.Region != nil && *loc.Region != "" {
						searchRequests = append(searchRequests, domain.BoundarySearchRequest{
							Index:      loc.Index*100 + 4,
							Name:       *loc.Region,
							AdminLevel: 4,
						})
					}
					// Всегда добавляем страну
					searchRequests = append(searchRequests, domain.BoundarySearchRequest{
						Index:      loc.Index*100 + 2,
						Name:       loc.Country,
						AdminLevel: 2,
					})
				}

				if len(searchRequests) > 0 {
					results, err := uc.enrichmentUC.boundaryRepo.SearchByTextBatch(ctx, searchRequests)
					if err != nil {
						uc.logger.Error("SearchByTextBatch failed", zap.Error(err))
						result.nameBasedErr = err
						return
					}
					result.nameBasedResults = results
				}
			}()
		}

		wg.Wait()
		resultChan <- result
	}()

	// Ждём результаты
	batchRes := <-resultChan

	// Шаг 3: Собираем результаты
	// Создаём маппинг loc.Index -> позиция в слайсе results
	indexToPos := make(map[int]int)
	for i, loc := range req.Locations {
		indexToPos[loc.Index] = i
	}

	results := make([]dto.LocationEnrichmentResult, len(req.Locations))
	// Инициализируем результаты
	for i, loc := range req.Locations {
		results[i] = dto.LocationEnrichmentResult{
			Index: loc.Index,
		}
	}

	successCount := 0
	errorCount := 0
	dbQueriesCount := 0

	if len(visibleLocations) > 0 {
		dbQueriesCount++
	}
	if len(nameBasedLocations) > 0 {
		dbQueriesCount++
	}

	// Обрабатываем visible локации (reverse geocoding)
	if batchRes.visibleErr != nil {
		for _, loc := range visibleLocations {
			if pos, ok := indexToPos[loc.Index]; ok {
				results[pos] = dto.LocationEnrichmentResult{
					Index: loc.Index,
					Error: "failed to resolve by coordinates: " + batchRes.visibleErr.Error(),
				}
				errorCount++
			}
		}
	} else {
		// Обрабатываем visible локации по их позиции в visibleLocations slice
		for i, loc := range visibleLocations {
			pos, ok := indexToPos[loc.Index]
			if !ok {
				continue
			}

			boundaries := batchRes.visibleResults[i]
			if len(boundaries) == 0 {
				results[pos] = dto.LocationEnrichmentResult{
					Index: loc.Index,
					Error: "no boundaries found for coordinates",
				}
				errorCount++
				continue
			}

			enriched := uc.boundariesToEnrichedLocation(boundaries)
			results[pos] = dto.LocationEnrichmentResult{
				Index:            loc.Index,
				EnrichedLocation: enriched,
			}
			successCount++
		}
	}

	// Обрабатываем name-based локации
	if batchRes.nameBasedErr != nil {
		for _, loc := range nameBasedLocations {
			if pos, ok := indexToPos[loc.Index]; ok {
				results[pos] = dto.LocationEnrichmentResult{
					Index: loc.Index,
					Error: "failed to resolve by name: " + batchRes.nameBasedErr.Error(),
				}
				errorCount++
			}
		}
	} else {
		// Группируем результаты поиска по исходному индексу локации
		searchResultsByLocIdx := make(map[int][]domain.BoundarySearchResult)
		for _, sr := range batchRes.nameBasedResults {
			locIdx := sr.Index / 100 // восстанавливаем исходный индекс
			searchResultsByLocIdx[locIdx] = append(searchResultsByLocIdx[locIdx], sr)
		}

		for _, loc := range nameBasedLocations {
			pos, ok := indexToPos[loc.Index]
			if !ok {
				continue
			}

			searchResults := searchResultsByLocIdx[loc.Index]

			// Собираем найденные границы
			var foundBoundaries []*domain.AdminBoundary
			for _, sr := range searchResults {
				if sr.Found && sr.Boundary != nil {
					foundBoundaries = append(foundBoundaries, sr.Boundary)
				}
			}

			if len(foundBoundaries) == 0 {
				results[pos] = dto.LocationEnrichmentResult{
					Index: loc.Index,
					Error: "no boundaries found by name",
				}
				errorCount++
				continue
			}

			enriched := uc.boundariesToEnrichedLocation(foundBoundaries)
			results[pos] = dto.LocationEnrichmentResult{
				Index:            loc.Index,
				EnrichedLocation: enriched,
			}
			successCount++
		}
	}

	uc.logger.Info("EnrichLocationBatch completed",
		zap.Int("total", len(req.Locations)),
		zap.Int("success", successCount),
		zap.Int("errors", errorCount),
		zap.Int("visible_processed", len(visibleLocations)),
		zap.Int("name_based_processed", len(nameBasedLocations)),
		zap.Int("db_queries", dbQueriesCount))

	return &dto.EnrichmentDebugLocationBatchResponse{
		Results: results,
		Meta: dto.LocationBatchMeta{
			TotalLocations:   len(req.Locations),
			SuccessCount:     successCount,
			ErrorCount:       errorCount,
			VisibleCount:     len(visibleLocations),
			NameResolveCount: len(nameBasedLocations),
			DBQueriesCount:   dbQueriesCount,
		},
	}, nil
}

// boundariesToEnrichedLocation преобразует слайс AdminBoundary в EnrichedLocationDTO
func (uc *EnrichmentDebugUseCase) boundariesToEnrichedLocation(boundaries []*domain.AdminBoundary) *dto.EnrichedLocationDTO {
	result := &dto.EnrichedLocationDTO{}

	for _, b := range boundaries {
		info := &dto.BoundaryInfoDTO{
			ID:             b.ID,
			Name:           b.Name,
			TranslateNames: make(map[string]string),
		}

		// Собираем переводы
		if b.NameEn != "" {
			info.TranslateNames["en"] = b.NameEn
		}
		if b.NameEs != "" {
			info.TranslateNames["es"] = b.NameEs
		}
		if b.NameCa != "" {
			info.TranslateNames["ca"] = b.NameCa
		}
		if b.NameRu != "" {
			info.TranslateNames["ru"] = b.NameRu
		}
		if b.NameUk != "" {
			info.TranslateNames["uk"] = b.NameUk
		}
		if b.NameFr != "" {
			info.TranslateNames["fr"] = b.NameFr
		}
		if b.NamePt != "" {
			info.TranslateNames["pt"] = b.NamePt
		}
		if b.NameIt != "" {
			info.TranslateNames["it"] = b.NameIt
		}
		if b.NameDe != "" {
			info.TranslateNames["de"] = b.NameDe
		}

		if len(info.TranslateNames) == 0 {
			info.TranslateNames = nil
		}

		switch b.AdminLevel {
		case 2:
			result.Country = info
		case 4:
			result.Region = info
		case 6:
			result.Province = info
		case 8:
			result.City = info
		case 9:
			result.District = info
		case 10:
			result.Neighborhood = info
		}
	}

	return result
}

// ========== Priority Transport Methods ==========

// GetNearestTransportByPriority возвращает ближайший транспорт с приоритетом.
// Логика приоритизации: если в радиусе есть metro/train - возвращаем только их,
// иначе возвращаем bus/tram.
// Включает информацию о линиях (L2, L4, номера автобусов) и их цветах.
func (uc *EnrichmentDebugUseCase) GetNearestTransportByPriority(
	ctx context.Context,
	req dto.PriorityTransportRequest,
) (*dto.PriorityTransportResponse, error) {
	// Validate coordinates
	if req.Lat < -90 || req.Lat > 90 || req.Lon < -180 || req.Lon > 180 {
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
func (uc *EnrichmentDebugUseCase) GetNearestTransportByPriorityBatch(
	ctx context.Context,
	req dto.PriorityTransportBatchRequest,
) (*dto.PriorityTransportBatchResponse, error) {
	// Validate request
	if len(req.Points) == 0 {
		return nil, errors.ErrInvalidRequest
	}

	for _, p := range req.Points {
		if p.Lat < -90 || p.Lat > 90 || p.Lon < -180 || p.Lon > 180 {
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
