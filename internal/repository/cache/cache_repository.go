package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type cacheRepository struct {
	client *redis.Client
	logger *zap.Logger
}

func NewCacheRepository(redis *Redis) repository.CacheRepository {
	return &cacheRepository{
		client: redis.Client(),
		logger: redis.logger,
	}
}

func (r *cacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	val, err := r.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, nil // Cache miss
	}
	if err != nil {
		r.logger.Error("Failed to get from cache", zap.String("key", key), zap.Error(err))
		return nil, fmt.Errorf("cache get error: %w", err)
	}

	r.logger.Debug("Cache hit", zap.String("key", key))
	return val, nil
}

func (r *cacheRepository) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	err := r.client.Set(ctx, key, value, ttl).Err()
	if err != nil {
		r.logger.Error("Failed to set cache", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("cache set error: %w", err)
	}

	r.logger.Debug("Cache set", zap.String("key", key), zap.Duration("ttl", ttl))
	return nil
}

func (r *cacheRepository) Delete(ctx context.Context, key string) error {
	err := r.client.Del(ctx, key).Err()
	if err != nil {
		r.logger.Error("Failed to delete from cache", zap.String("key", key), zap.Error(err))
		return fmt.Errorf("cache delete error: %w", err)
	}

	r.logger.Debug("Cache deleted", zap.String("key", key))
	return nil
}

func (r *cacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	val, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		r.logger.Error("Failed to check cache existence", zap.String("key", key), zap.Error(err))
		return false, fmt.Errorf("cache exists error: %w", err)
	}

	return val > 0, nil
}

func (r *cacheRepository) GetTile(ctx context.Context, z, x, y int) ([]byte, error) {
	key := fmt.Sprintf("tile:%d:%d:%d", z, x, y)
	return r.Get(ctx, key)
}

func (r *cacheRepository) SetTile(ctx context.Context, z, x, y int, data []byte, ttl time.Duration) error {
	key := fmt.Sprintf("tile:%d:%d:%d", z, x, y)
	return r.Set(ctx, key, data, ttl)
}

// GetStats получает статистику из кеша
func (r *cacheRepository) GetStats(ctx context.Context) (*domain.Statistics, error) {
	key := "stats:current"
	data, err := r.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, nil // Cache miss
	}

	var stats domain.Statistics
	if err := json.Unmarshal(data, &stats); err != nil {
		r.logger.Error("Failed to unmarshal stats from cache", zap.Error(err))
		return nil, fmt.Errorf("unmarshal stats: %w", err)
	}

	return &stats, nil
}

// SetStats сохраняет статистику в кеше
func (r *cacheRepository) SetStats(ctx context.Context, stats *domain.Statistics, ttl time.Duration) error {
	key := "stats:current"
	data, err := json.Marshal(stats)
	if err != nil {
		r.logger.Error("Failed to marshal stats", zap.Error(err))
		return fmt.Errorf("marshal stats: %w", err)
	}

	return r.Set(ctx, key, data, ttl)
}
