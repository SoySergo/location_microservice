package repository

import (
	"context"

	"github.com/location-microservice/internal/domain"
)

// BoundaryRepository определяет методы для работы с административными границами
type BoundaryRepository interface {
	// GetByID возвращает административную границу по ID
	GetByID(ctx context.Context, id int64) (*domain.AdminBoundary, error)

	// SearchByText выполняет текстовый поиск по названиям границ с поддержкой языков и фильтрации
	SearchByText(ctx context.Context, query string, lang string, adminLevels []int, limit int) ([]*domain.AdminBoundary, error)

	// SearchByTextBatch выполняет батчевый текстовый поиск для нескольких запросов одним SQL
	SearchByTextBatch(ctx context.Context, requests []domain.BoundarySearchRequest) ([]domain.BoundarySearchResult, error)

	// ReverseGeocode возвращает адрес по координатам
	ReverseGeocode(ctx context.Context, lat, lon float64) (*domain.Address, error)

	// ReverseGeocodeBatch возвращает адреса для нескольких точек одним запросом
	ReverseGeocodeBatch(ctx context.Context, points []domain.LatLon) ([]*domain.Address, error)

	// GetTile генерирует MVT тайл для заданных координат
	GetTile(ctx context.Context, z, x, y int) ([]byte, error)

	// GetByPoint возвращает административные границы для точки (reverse geocoding)
	GetByPoint(ctx context.Context, lat, lon float64) ([]*domain.AdminBoundary, error)

	// GetByPointBatch возвращает административные границы для нескольких точек одним запросом
	// Возвращает map[point_idx] -> []*AdminBoundary с полными данными о границах
	GetByPointBatch(ctx context.Context, points []domain.LatLon) (map[int][]*domain.AdminBoundary, error)

	// Search выполняет текстовый поиск по названиям границ
	Search(ctx context.Context, query string, limit int) ([]*domain.AdminBoundary, error)

	// GetChildren возвращает дочерние границы для родительской
	GetChildren(ctx context.Context, parentID int64) ([]*domain.AdminBoundary, error)

	// GetByAdminLevel возвращает границы определенного уровня
	GetByAdminLevel(ctx context.Context, level int, limit int) ([]*domain.AdminBoundary, error)

	// GetBoundariesInRadius возвращает границы в радиусе от точки (для использования в коде)
	GetBoundariesInRadius(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.AdminBoundary, error)

	// GetBoundariesRadiusTile генерирует MVT тайл с границами в радиусе от точки
	GetBoundariesRadiusTile(ctx context.Context, lat, lon, radiusKm float64) ([]byte, error)
}
