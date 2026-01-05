package redis

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

type streamRepository struct {
	client *redis.Client
	logger *zap.Logger
}

// NewStreamRepository создает новый экземпляр StreamRepository
func NewStreamRepository(client *redis.Client, logger *zap.Logger) repository.StreamRepository {
	return &streamRepository{
		client: client,
		logger: logger,
	}
}

// CreateConsumerGroup создаёт consumer group для стрима
func (r *streamRepository) CreateConsumerGroup(ctx context.Context, stream, group string) error {
	// Пытаемся создать consumer group, начиная с ID "$" (новые сообщения)
	// MKSTREAM автоматически создаст стрим, если он не существует
	err := r.client.XGroupCreateMkStream(ctx, stream, group, "$").Err()
	if err != nil {
		// Игнорируем ошибку BUSYGROUP - группа уже существует
		if err.Error() == "BUSYGROUP Consumer Group name already exists" {
			r.logger.Debug("Consumer group already exists",
				zap.String("stream", stream),
				zap.String("group", group))
			return nil
		}
		r.logger.Error("Failed to create consumer group",
			zap.String("stream", stream),
			zap.String("group", group),
			zap.Error(err))
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	r.logger.Info("Consumer group created successfully",
		zap.String("stream", stream),
		zap.String("group", group))
	return nil
}

// ConsumeStream читает сообщения из стрима с использованием consumer group
func (r *streamRepository) ConsumeStream(ctx context.Context, stream, group, consumer string) (<-chan domain.StreamMessage, error) {
	msgChan := make(chan domain.StreamMessage, 10)

	go func() {
		defer close(msgChan)

		// Начинаем читать с непрочитанных сообщений (">")
		lastID := ">"

		for {
			select {
			case <-ctx.Done():
				r.logger.Info("Stream consumer stopped",
					zap.String("stream", stream),
					zap.String("consumer", consumer))
				return
			default:
				// XReadGroup блокирует на 1 секунду, ожидая новых сообщений
				result, err := r.client.XReadGroup(ctx, &redis.XReadGroupArgs{
					Group:    group,
					Consumer: consumer,
					Streams:  []string{stream, lastID},
					Count:    10,
					Block:    1 * time.Second,
				}).Result()

				if err != nil {
					if err == redis.Nil {
						// Нет новых сообщений - продолжаем ждать
						continue
					}
					if ctx.Err() != nil {
						// Контекст был отменён
						return
					}
					r.logger.Error("Failed to read from stream",
						zap.String("stream", stream),
						zap.Error(err))
					time.Sleep(time.Second)
					continue
				}

				// Обрабатываем полученные сообщения
				for _, stream := range result {
					for _, msg := range stream.Messages {
						// Извлекаем JSON данные из поля "data"
						data, ok := msg.Values["data"].(string)
						if !ok {
							r.logger.Warn("Message does not contain 'data' field",
								zap.String("message_id", msg.ID))
							continue
						}

						select {
						case msgChan <- domain.StreamMessage{
							ID:   msg.ID,
							Data: data,
						}:
							r.logger.Debug("Message sent to channel",
								zap.String("message_id", msg.ID))
						case <-ctx.Done():
							return
						}
					}
				}
			}
		}
	}()

	return msgChan, nil
}

// AckMessage подтверждает обработку сообщения
func (r *streamRepository) AckMessage(ctx context.Context, stream, group, messageID string) error {
	err := r.client.XAck(ctx, stream, group, messageID).Err()
	if err != nil {
		r.logger.Error("Failed to acknowledge message",
			zap.String("stream", stream),
			zap.String("group", group),
			zap.String("message_id", messageID),
			zap.Error(err))
		return fmt.Errorf("failed to acknowledge message: %w", err)
	}

	r.logger.Debug("Message acknowledged",
		zap.String("message_id", messageID))
	return nil
}

// PublishToStream публикует сообщение в стрим
func (r *streamRepository) PublishToStream(ctx context.Context, stream string, data interface{}) error {
	// Сериализуем данные в JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		r.logger.Error("Failed to marshal data",
			zap.String("stream", stream),
			zap.Error(err))
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Публикуем в стрим
	result, err := r.client.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		Values: map[string]interface{}{
			"data": string(jsonData),
		},
	}).Result()

	if err != nil {
		r.logger.Error("Failed to publish to stream",
			zap.String("stream", stream),
			zap.Error(err))
		return fmt.Errorf("failed to publish to stream: %w", err)
	}

	r.logger.Debug("Message published to stream",
		zap.String("stream", stream),
		zap.String("message_id", result))
	return nil
}
