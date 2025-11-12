package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/location-microservice/internal/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Redis struct {
	client *redis.Client
	logger *zap.Logger
}

func NewRedis(cfg *config.RedisConfig, logger *zap.Logger) (*Redis, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	logger.Info("Redis connected",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
	)

	return &Redis{
		client: client,
		logger: logger,
	}, nil
}

func (r *Redis) Close() error {
	r.logger.Info("Closing Redis connection")
	return r.client.Close()
}

func (r *Redis) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func (r *Redis) Client() *redis.Client {
	return r.client
}
