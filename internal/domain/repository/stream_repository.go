package repository

import (
	"context"

	"github.com/location-microservice/internal/domain"
)

// StreamRepository - интерфейс для работы с Redis Streams
type StreamRepository interface {
	// ConsumeStream читает сообщения из стрима
	ConsumeStream(ctx context.Context, stream, group, consumer string) (<-chan domain.StreamMessage, error)

	// AckMessage подтверждает обработку сообщения
	AckMessage(ctx context.Context, stream, group, messageID string) error

	// CreateConsumerGroup создаёт consumer group
	CreateConsumerGroup(ctx context.Context, stream, group string) error

	// PublishToStream публикует сообщение в стрим
	PublishToStream(ctx context.Context, stream string, data interface{}) error
}
