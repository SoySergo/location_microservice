package repository

import (
	"context"
	"time"
)

// CacheRepository определяет методы для работы с кешем
type CacheRepository interface {
	// Get получает значение из кеша по ключу
	Get(ctx context.Context, key string) ([]byte, error)

	// Set сохраняет значение в кеше с TTL
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete удаляет значение из кеша
	Delete(ctx context.Context, key string) error

	// Exists проверяет существование ключа
	Exists(ctx context.Context, key string) (bool, error)

	// GetTile получает тайл из кеша
	GetTile(ctx context.Context, z, x, y int) ([]byte, error)

	// SetTile сохраняет тайл в кеше
	SetTile(ctx context.Context, z, x, y int, data []byte, ttl time.Duration) error
}
