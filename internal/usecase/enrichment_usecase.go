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
	isVisible := uc.isAddressVisible(event)
	enrichedLocation.IsAddressVisible = &isVisible

	result.EnrichedLocation = enrichedLocation

	// Поиск ближайшего транспорта, если есть координаты
	if event.Latitude != nil && event.Longitude != nil {
		nearestTransport, err := uc.findNearestTransport(ctx, *event.Latitude, *event.Longitude)
		if err != nil {
			uc.logger.Warn("Failed to find nearest transport",
				zap.String("property_id", event.PropertyID.String()),
				zap.Error(err))
			// Не считаем это критичной ошибкой
		} else {
			result.NearestTransport = nearestTransport
		}
	}

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
		return nil, fmt.Errorf("boundary not found for level %d: %w", adminLevel, err)
	}

	// Получаем всю иерархию через parent_id
	return uc.resolveLocationHierarchy(ctx, boundary.ID)
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
		switch boundary.AdminLevel {
		case 2:
			result.CountryID = &boundary.ID
		case 4:
			result.RegionID = &boundary.ID
		case 6:
			result.ProvinceID = &boundary.ID
		case 8:
			result.CityID = &boundary.ID
		case 9:
			result.DistrictID = &boundary.ID
		case 10:
			result.NeighborhoodID = &boundary.ID
		}
	}

	// Проверяем, что хотя бы страна найдена
	if result.CountryID == nil {
		return nil, errors.ErrLocationNotFound
	}

	return result, nil
}

// resolveLocationHierarchy получает всю иерархию локации через parent_id
func (uc *EnrichmentUseCase) resolveLocationHierarchy(ctx context.Context, boundaryID int64) (*domain.EnrichedLocation, error) {
	result := &domain.EnrichedLocation{}
	currentID := boundaryID

	// Проходим по цепочке parent_id вверх по иерархии
	for currentID != 0 {
		boundary, err := uc.boundaryRepo.GetByID(ctx, currentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get boundary %d: %w", currentID, err)
		}

		// Заполняем соответствующее поле в зависимости от admin_level
		switch boundary.AdminLevel {
		case 2:
			result.CountryID = &boundary.ID
		case 4:
			result.RegionID = &boundary.ID
		case 6:
			result.ProvinceID = &boundary.ID
		case 8:
			result.CityID = &boundary.ID
		case 9:
			result.DistrictID = &boundary.ID
		case 10:
			result.NeighborhoodID = &boundary.ID
		}

		// Переходим к родительской границе
		if boundary.ParentID != nil {
			currentID = *boundary.ParentID
		} else {
			break
		}
	}

	// Проверяем, что хотя бы страна найдена
	if result.CountryID == nil {
		return nil, fmt.Errorf("country not found in hierarchy")
	}

	return result, nil
}

// findBoundaryByName ищет границу по названию с учетом всех языковых полей
func (uc *EnrichmentUseCase) findBoundaryByName(ctx context.Context, name string, adminLevel int) (*domain.AdminBoundary, error) {
	// Поиск по всем языковым полям
	boundaries, err := uc.boundaryRepo.SearchByText(ctx, name, "", []int{adminLevel}, 1)
	if err != nil {
		return nil, fmt.Errorf("search failed: %w", err)
	}

	if len(boundaries) == 0 {
		return nil, errors.ErrLocationNotFound
	}

	return boundaries[0], nil
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

		result = append(result, domain.NearestStation{
			StationID: station.ID,
			Name:      station.Name,
			Type:      station.Type,
			Distance:  distance,
			LineIDs:   station.LineIDs,
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
