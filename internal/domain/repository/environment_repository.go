package repository

import (
	"context"

	"github.com/location-microservice/internal/domain"
)

// EnvironmentRepository определяет методы для работы с экологическими объектами
type EnvironmentRepository interface {
	// GetGreenSpacesNearby возвращает зеленые зоны в радиусе
	GetGreenSpacesNearby(ctx context.Context, lat, lon float64, radiusKm float64) ([]*domain.GreenSpace, error)

	// GetWaterBodiesNearby возвращает водные объекты в радиусе
	GetWaterBodiesNearby(ctx context.Context, lat, lon float64, radiusKm float64) ([]*domain.WaterBody, error)

	// GetBeachesNearby возвращает пляжи в радиусе
	GetBeachesNearby(ctx context.Context, lat, lon float64, radiusKm float64) ([]*domain.Beach, error)

	// GetNoiseSourcesNearby возвращает источники шума в радиусе
	GetNoiseSourcesNearby(ctx context.Context, lat, lon float64, radiusKm float64) ([]*domain.NoiseSource, error)

	// GetTouristZonesNearby возвращает туристические зоны в радиусе
	GetTouristZonesNearby(ctx context.Context, lat, lon float64, radiusKm float64) ([]*domain.TouristZone, error)

	// GetGreenSpaceByID возвращает зеленую зону по ID
	GetGreenSpaceByID(ctx context.Context, id string) (*domain.GreenSpace, error)

	// GetBeachByID возвращает пляж по ID
	GetBeachByID(ctx context.Context, id string) (*domain.Beach, error)

	// GetTouristZoneByID возвращает туристическую зону по ID
	GetTouristZoneByID(ctx context.Context, id string) (*domain.TouristZone, error)

	// GetGreenSpacesTile возвращает MVT tile с зелеными зонами
	GetGreenSpacesTile(ctx context.Context, z, x, y int) ([]byte, error)

	// GetWaterTile возвращает MVT tile с водными объектами
	GetWaterTile(ctx context.Context, z, x, y int) ([]byte, error)

	// GetBeachesTile возвращает MVT tile с пляжами
	GetBeachesTile(ctx context.Context, z, x, y int) ([]byte, error)

	// GetNoiseSourcesTile возвращает MVT tile с источниками шума
	GetNoiseSourcesTile(ctx context.Context, z, x, y int) ([]byte, error)

	// GetTouristZonesTile возвращает MVT tile с туристическими зонами
	GetTouristZonesTile(ctx context.Context, z, x, y int) ([]byte, error)

	// GetEnvironmentRadiusTile генерирует MVT тайл со всеми экологическими объектами в радиусе от точки
	GetEnvironmentRadiusTile(ctx context.Context, lat, lon, radiusKm float64) ([]byte, error)
}
