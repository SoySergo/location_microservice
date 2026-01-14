package usecase

import (
	"context"
	"fmt"
	"math"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/pkg/errors"
	"go.uber.org/zap"
)

// EnrichmentUseCase - use case для обогащения локаций
type EnrichmentUseCase struct {
	boundaryRepo    repository.BoundaryRepository
	transportRepo   repository.TransportRepository
	logger          *zap.Logger
	transportTypes  []string
	transportRadius float64
}

// NewEnrichmentUseCase создает новый EnrichmentUseCase
func NewEnrichmentUseCase(
	boundaryRepo repository.BoundaryRepository,
	transportRepo repository.TransportRepository,
	logger *zap.Logger,
	transportTypes []string,
	transportRadius float64,
) *EnrichmentUseCase {
	return &EnrichmentUseCase{
		boundaryRepo:    boundaryRepo,
		transportRepo:   transportRepo,
		logger:          logger,
		transportTypes:  transportTypes,
		transportRadius: transportRadius,
	}
}

// EnrichLocation обогащает локацию из события
func (uc *EnrichmentUseCase) EnrichLocation(ctx context.Context, event *domain.LocationEnrichEvent) (*domain.LocationDoneEvent, error) {
	result := &domain.LocationDoneEvent{
		PropertyID: event.PropertyID,
	}

	// Попытка резолвить локацию
	enrichedLocation, err := uc.resolveLocation(ctx, event)
	if err != nil {
		uc.logger.Error("Failed to resolve location",
			zap.String("property_id", event.PropertyID.String()),
			zap.Error(err))
		result.Error = fmt.Sprintf("failed to resolve location: %v", err)
		return result, nil // Возвращаем результат с ошибкой, но не прерываем обработку
	}

	// Определяем видимость адреса
	// isVisible := uc.isAddressVisible(event)
	// enrichedLocation.IsAddressVisible = &isVisible

	result.EnrichedLocation = enrichedLocation

	// Покачто скрываем
	// Поиск ближайшего транспорта, если есть координаты
	// if event.Latitude != nil && event.Longitude != nil {
	// 	nearestTransport, err := uc.findNearestTransport(ctx, *event.Latitude, *event.Longitude)
	// 	if err != nil {
	// 		uc.logger.Warn("Failed to find nearest transport",
	// 			zap.String("property_id", event.PropertyID.String()),
	// 			zap.Error(err))
	// 		// Не считаем это критичной ошибкой
	// 	} else {
	// 		result.NearestTransport = nearestTransport
	// 	}
	// }

	return result, nil
}

// resolveLocation резолвит локацию из события
func (uc *EnrichmentUseCase) resolveLocation(ctx context.Context, event *domain.LocationEnrichEvent) (*domain.EnrichedLocation, error) {
	// Стратегия 1: Поиск от самого детального уровня к общему
	if event.Neighborhood != nil && *event.Neighborhood != "" {
		return uc.resolveFromLevel(ctx, *event.Neighborhood, 10, event)
	}

	if event.District != nil && *event.District != "" {
		return uc.resolveFromLevel(ctx, *event.District, 9, event)
	}

	if event.City != nil && *event.City != "" {
		return uc.resolveFromLevel(ctx, *event.City, 8, event)
	}

	if event.Province != nil && *event.Province != "" {
		return uc.resolveFromLevel(ctx, *event.Province, 6, event)
	}

	if event.Region != nil && *event.Region != "" {
		return uc.resolveFromLevel(ctx, *event.Region, 4, event)
	}

	// Стратегия 2: Reverse geocoding по координатам
	if event.Latitude != nil && event.Longitude != nil {
		return uc.resolveFromCoordinates(ctx, *event.Latitude, *event.Longitude)
	}

	// Стратегия 3: Поиск только страны
	return uc.resolveFromLevel(ctx, event.Country, 2, event)
}

// resolveFromLevel резолвит локацию начиная с определенного уровня
func (uc *EnrichmentUseCase) resolveFromLevel(ctx context.Context, name string, adminLevel int, event *domain.LocationEnrichEvent) (*domain.EnrichedLocation, error) {
	// Ищем границу по названию
	boundary, err := uc.findBoundaryByName(ctx, name, adminLevel)
	if err != nil {
		uc.logger.Error("Failed to find boundary by name",
			zap.String("name", name),
			zap.Int("admin_level", adminLevel),
			zap.Error(err))
		return nil, fmt.Errorf("boundary not found for level %d: %w", adminLevel, err)
	}

	uc.logger.Debug("Found boundary",
		zap.String("name", name),
		zap.Int("admin_level", adminLevel),
		zap.Int64("boundary_id", boundary.ID),
		zap.String("boundary_name", boundary.Name))

	// Получаем всю иерархию через parent_id
	result, err := uc.resolveLocationHierarchy(ctx, boundary.ID)
	if err != nil {
		// Если иерархия неполная, пробуем fallback стратегии
		uc.logger.Debug("Hierarchy incomplete, trying fallback strategies",
			zap.String("name", name),
			zap.Int("admin_level", adminLevel))

		return uc.resolveWithFallback(ctx, boundary, event)
	}

	return result, nil
}

// resolveFromCoordinates резолвит локацию по координатам
func (uc *EnrichmentUseCase) resolveFromCoordinates(ctx context.Context, lat, lon float64) (*domain.EnrichedLocation, error) {
	// Получаем границы для точки
	boundaries, err := uc.boundaryRepo.GetByPoint(ctx, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("failed to get boundaries by point: %w", err)
	}

	if len(boundaries) == 0 {
		return nil, errors.ErrLocationNotFound
	}

	// Создаем результат из всех найденных границ
	result := &domain.EnrichedLocation{}
	for _, boundary := range boundaries {
		info := uc.boundaryToInfo(boundary)
		switch boundary.AdminLevel {
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

	// Проверяем, что найден хотя бы один уровень иерархии
	// (не обязательно страна, т.к. OSM данные могут быть неполными)
	if result.Country == nil && result.Region == nil && result.Province == nil && result.City == nil {
		return nil, errors.ErrLocationNotFound
	}

	return result, nil
}

// boundaryToInfo преобразует AdminBoundary в BoundaryInfo
func (uc *EnrichmentUseCase) boundaryToInfo(boundary *domain.AdminBoundary) *domain.BoundaryInfo {
	info := &domain.BoundaryInfo{
		ID:             boundary.ID,
		Name:           boundary.Name,
		TranslateNames: make(map[string]string),
	}

	// Собираем все переводы
	if boundary.NameEn != "" {
		info.TranslateNames["en"] = boundary.NameEn
	}
	if boundary.NameEs != "" {
		info.TranslateNames["es"] = boundary.NameEs
	}
	if boundary.NameCa != "" {
		info.TranslateNames["ca"] = boundary.NameCa
	}
	if boundary.NameRu != "" {
		info.TranslateNames["ru"] = boundary.NameRu
	}
	if boundary.NameUk != "" {
		info.TranslateNames["uk"] = boundary.NameUk
	}
	if boundary.NameFr != "" {
		info.TranslateNames["fr"] = boundary.NameFr
	}
	if boundary.NamePt != "" {
		info.TranslateNames["pt"] = boundary.NamePt
	}
	if boundary.NameIt != "" {
		info.TranslateNames["it"] = boundary.NameIt
	}
	if boundary.NameDe != "" {
		info.TranslateNames["de"] = boundary.NameDe
	}

	// Если нет переводов, не возвращаем пустую map
	if len(info.TranslateNames) == 0 {
		info.TranslateNames = nil
	}

	return info
}

// resolveLocationHierarchy получает всю иерархию локации через parent_id
func (uc *EnrichmentUseCase) resolveLocationHierarchy(ctx context.Context, boundaryID int64) (*domain.EnrichedLocation, error) {
	result := &domain.EnrichedLocation{}
	currentID := boundaryID

	uc.logger.Debug("Starting hierarchy resolution", zap.Int64("starting_boundary_id", boundaryID))

	// Проходим по цепочке parent_id вверх по иерархии
	for currentID != 0 {
		boundary, err := uc.boundaryRepo.GetByID(ctx, currentID)
		if err != nil {
			uc.logger.Error("Failed to get boundary by ID",
				zap.Int64("boundary_id", currentID),
				zap.Error(err))
			return nil, fmt.Errorf("failed to get boundary %d: %w", currentID, err)
		}

		uc.logger.Debug("Processing boundary in hierarchy",
			zap.Int64("boundary_id", boundary.ID),
			zap.String("name", boundary.Name),
			zap.Int("admin_level", boundary.AdminLevel),
			zap.Int64p("parent_id", boundary.ParentID))

		// Заполняем соответствующее поле в зависимости от admin_level
		info := uc.boundaryToInfo(boundary)
		switch boundary.AdminLevel {
		case 2:
			result.Country = info
			uc.logger.Debug("Set Country", zap.Int64("country_id", boundary.ID))
		case 4:
			result.Region = info
			uc.logger.Debug("Set Region", zap.Int64("region_id", boundary.ID))
		case 6:
			result.Province = info
			uc.logger.Debug("Set Province", zap.Int64("province_id", boundary.ID))
		case 8:
			result.City = info
			uc.logger.Debug("Set City", zap.Int64("city_id", boundary.ID))
		case 9:
			result.District = info
			uc.logger.Debug("Set District", zap.Int64("district_id", boundary.ID))
		case 10:
			result.Neighborhood = info
			uc.logger.Debug("Set Neighborhood", zap.Int64("neighborhood_id", boundary.ID))
		default:
			uc.logger.Debug("Skipping unknown admin_level", zap.Int("admin_level", boundary.AdminLevel))
		}

		// Переходим к родительской границе
		if boundary.ParentID != nil {
			currentID = *boundary.ParentID
		} else {
			uc.logger.Debug("Reached top of hierarchy (no parent)")
			break
		}
	}

	// Проверяем, что найден хотя бы один уровень иерархии
	// (не обязательно страна, т.к. OSM данные могут быть неполными)
	if result.Country == nil && result.Region == nil && result.Province == nil && result.City == nil {
		uc.logger.Debug("Hierarchy incomplete, will try fallback",
			zap.Int64("starting_boundary_id", boundaryID))
		return nil, fmt.Errorf("no valid hierarchy found")
	}

	uc.logger.Debug("Hierarchy resolution completed successfully")

	return result, nil
}

// findBoundaryByName ищет границу по названию с учетом всех языковых полей
func (uc *EnrichmentUseCase) findBoundaryByName(ctx context.Context, name string, adminLevel int) (*domain.AdminBoundary, error) {
	uc.logger.Debug("Searching boundary by name",
		zap.String("name", name),
		zap.Int("admin_level", adminLevel))

	// Поиск по всем языковым полям
	boundaries, err := uc.boundaryRepo.SearchByText(ctx, name, "", []int{adminLevel}, 1)
	if err != nil {
		uc.logger.Error("SearchByText failed",
			zap.String("name", name),
			zap.Int("admin_level", adminLevel),
			zap.Error(err))
		return nil, fmt.Errorf("search failed: %w", err)
	}

	uc.logger.Debug("SearchByText result",
		zap.String("name", name),
		zap.Int("admin_level", adminLevel),
		zap.Int("found_count", len(boundaries)))

	if len(boundaries) == 0 {
		uc.logger.Debug("No boundaries found",
			zap.String("name", name),
			zap.Int("admin_level", adminLevel))
		return nil, errors.ErrLocationNotFound
	}

	uc.logger.Debug("Found boundary",
		zap.String("search_name", name),
		zap.Int64("boundary_id", boundaries[0].ID),
		zap.String("boundary_name", boundaries[0].Name),
		zap.Int("boundary_admin_level", boundaries[0].AdminLevel))

	return boundaries[0], nil
}

// resolveWithFallback пытается восстановить иерархию через альтернативные методы
func (uc *EnrichmentUseCase) resolveWithFallback(ctx context.Context, boundary *domain.AdminBoundary, event *domain.LocationEnrichEvent) (*domain.EnrichedLocation, error) {
	result := &domain.EnrichedLocation{}

	// Сохраняем найденную границу
	info := uc.boundaryToInfo(boundary)
	switch boundary.AdminLevel {
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

	uc.logger.Debug("Fallback: saved initial boundary",
		zap.Int("admin_level", boundary.AdminLevel),
		zap.Int64("boundary_id", boundary.ID))

	// Стратегия 1: Попробовать найти границы по координатам события
	if event.Latitude != nil && event.Longitude != nil {
		uc.logger.Debug("Fallback: trying coordinates from event",
			zap.Float64("lat", *event.Latitude),
			zap.Float64("lon", *event.Longitude))

		coordResult, err := uc.resolveFromCoordinates(ctx, *event.Latitude, *event.Longitude)
		if err == nil {
			// Объединяем результаты: берем недостающие уровни из координат
			if result.Country == nil && coordResult.Country != nil {
				result.Country = coordResult.Country
			}
			if result.Region == nil && coordResult.Region != nil {
				result.Region = coordResult.Region
			}
			if result.Province == nil && coordResult.Province != nil {
				result.Province = coordResult.Province
			}
			if result.City == nil && coordResult.City != nil {
				result.City = coordResult.City
			}
			if result.District == nil && coordResult.District != nil {
				result.District = coordResult.District
			}
			if result.Neighborhood == nil && coordResult.Neighborhood != nil {
				result.Neighborhood = coordResult.Neighborhood
			}

			uc.logger.Debug("Fallback: merged with coordinate results")
		} else {
			uc.logger.Debug("Fallback: coordinates resolution returned no extra data")
		}
	}

	// Стратегия 2: Если все еще нет страны, попробовать найти её напрямую по имени из события
	if result.Country == nil && event.Country != "" {
		uc.logger.Debug("Fallback: trying to find country by name",
			zap.String("country", event.Country))

		countryBoundary, err := uc.findBoundaryByName(ctx, event.Country, 2)
		if err == nil {
			result.Country = uc.boundaryToInfo(countryBoundary)
			uc.logger.Debug("Fallback: found country",
				zap.Int64("country_id", countryBoundary.ID),
				zap.String("country_name", countryBoundary.Name))
		} else {
			uc.logger.Debug("Fallback: country not in OSM data (expected for partial imports)")
		}
	}

	// Проверяем финальный результат - достаточно иметь хотя бы один уровень иерархии
	if result.Country == nil && result.Region == nil && result.Province == nil && result.City == nil {
		return nil, fmt.Errorf("failed to resolve any hierarchy level even with fallback strategies")
	}

	uc.logger.Debug("Fallback resolution successful")

	return result, nil
}

// findNearestTransport находит ближайшие станции транспорта
func (uc *EnrichmentUseCase) findNearestTransport(ctx context.Context, lat, lon float64) ([]domain.NearestStation, error) {
	stations, err := uc.transportRepo.GetNearestStations(
		ctx,
		lat,
		lon,
		uc.transportTypes,
		uc.transportRadius,
		10, // максимум 10 станций
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get nearest stations: %w", err)
	}

	// Преобразуем в формат NearestStation и вычисляем точное расстояние
	result := make([]domain.NearestStation, 0, len(stations))
	for _, station := range stations {
		// Вычисляем расстояние от координат свойства до станции
		distance := uc.calculateDistance(lat, lon, station.Lat, station.Lon)

		// Получаем линии для станции
		lines, err := uc.transportRepo.GetLinesByStationID(ctx, station.ID)
		var lineInfos []domain.TransportLineInfo
		if err == nil && len(lines) > 0 {
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

		result = append(result, domain.NearestStation{
			StationID: station.ID,
			Name:      station.Name,
			Type:      station.Type,
			Lat:       station.Lat,
			Lon:       station.Lon,
			Distance:  distance,
			Lines:     lineInfos,
		})
	}

	return result, nil
}

// isAddressVisible определяет, является ли адрес видимым (точным)
func (uc *EnrichmentUseCase) isAddressVisible(event *domain.LocationEnrichEvent) bool {
	// Адрес считается точным, если есть улица и номер дома ИЛИ точные координаты
	hasStreetAddress := event.Street != nil && *event.Street != "" &&
		event.HouseNumber != nil && *event.HouseNumber != ""
	hasCoordinates := event.Latitude != nil && event.Longitude != nil

	return hasStreetAddress || hasCoordinates
}

// calculateDistance вычисляет расстояние между двумя точками в метрах (формула Haversine)
func (uc *EnrichmentUseCase) calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
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
