package repository

import (
	"context"

	"github.com/location-microservice/internal/domain"
)

// BoundaryRepository определяет методы для работы с административными границами
type BoundaryRepository interface {
	// GetByID возвращает административную границу по ID
	GetByID(ctx context.Context, id string) (*domain.AdminBoundary, error)

	// SearchByText выполняет текстовый поиск по названиям границ с поддержкой языков и фильтрации
	SearchByText(ctx context.Context, query string, lang string, adminLevels []int, limit int) ([]*domain.AdminBoundary, error)

	// ReverseGeocode возвращает адрес по координатам
	ReverseGeocode(ctx context.Context, lat, lon float64) (*domain.Address, error)

	// GetTile генерирует MVT тайл для заданных координат
	GetTile(ctx context.Context, z, x, y int) ([]byte, error)

	// GetByPoint возвращает административные границы для точки (reverse geocoding)
	GetByPoint(ctx context.Context, lat, lon float64) ([]*domain.AdminBoundary, error)

	// Search выполняет текстовый поиск по названиям границ
	Search(ctx context.Context, query string, limit int) ([]*domain.AdminBoundary, error)

	// GetChildren возвращает дочерние границы для родительской
	GetChildren(ctx context.Context, parentID string) ([]*domain.AdminBoundary, error)

	// GetByAdminLevel возвращает границы определенного уровня
	GetByAdminLevel(ctx context.Context, level int, limit int) ([]*domain.AdminBoundary, error)
}
