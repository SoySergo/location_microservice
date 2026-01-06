package repository

import (
	"context"

	"github.com/location-microservice/internal/domain"
)

// POIRepository определяет методы для работы с точками интереса
type POIRepository interface {
	// GetByID возвращает POI по ID
	GetByID(ctx context.Context, id int64) (*domain.POI, error)

	// GetNearby возвращает POI в радиусе от точки
	GetNearby(ctx context.Context, lat, lon float64, radiusKm float64, categories []string) ([]*domain.POI, error)

	// Search выполняет текстовый поиск POI
	Search(ctx context.Context, query string, categories []string, limit int) ([]*domain.POI, error)

	// GetByCategory возвращает POI определенной категории
	GetByCategory(ctx context.Context, category string, limit int) ([]*domain.POI, error)

	// GetCategories возвращает все категории POI
	GetCategories(ctx context.Context) ([]*domain.POICategory, error)

	// GetSubcategories возвращает подкатегории для категории
	GetSubcategories(ctx context.Context, categoryID int64) ([]*domain.POISubcategory, error)

	// GetPOITile генерирует MVT тайл с POI для заданных координат тайла
	GetPOITile(ctx context.Context, z, x, y int, categories []string) ([]byte, error)

	// GetPOIRadiusTile генерирует MVT тайл с POI в радиусе от точки
	GetPOIRadiusTile(ctx context.Context, lat, lon, radiusKm float64, categories []string) ([]byte, error)

	// GetPOIByBoundaryTile генерирует MVT тайл с POI внутри административной границы
	GetPOIByBoundaryTile(ctx context.Context, boundaryID int64, categories []string) ([]byte, error)

	// GetPOITileByCategories генерирует MVT тайл с POI по координатам тайла с фильтрацией по категориям и подкатегориям
	GetPOITileByCategories(ctx context.Context, z, x, y int, categories, subcategories []string) ([]byte, error)
}
