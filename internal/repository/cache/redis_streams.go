package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/location-microservice/internal/config"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// NewRedisStreams creates a dedicated Redis client for working with streams.
// It connects to the Redis instance specified in cfg and returns a *redis.Client
// that can be used for stream operations. Returns an error if the connection fails.
func NewRedisStreams(cfg *config.RedisStreamsConfig, logger *zap.Logger) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis streams: %w", err)
	}

	logger.Info("Redis Streams connected",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
	)

	return client, nil
}
