package location

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/google/uuid"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/worker"
	"go.uber.org/zap"
)

// LocationEnrichmentWorker обрабатывает события обогащения локаций
type LocationEnrichmentWorker struct {
	*worker.BaseWorker
	streamRepo    repository.StreamRepository
	enrichmentUC  *usecase.EnrichmentUseCase
	consumerName  string
	maxRetries    int
}

// NewLocationEnrichmentWorker создает новый LocationEnrichmentWorker
func NewLocationEnrichmentWorker(
	streamRepo repository.StreamRepository,
	enrichmentUC *usecase.EnrichmentUseCase,
	consumerGroup string,
	maxRetries int,
	logger *zap.Logger,
) *LocationEnrichmentWorker {
	// Генерируем уникальное имя consumer'а (используем hostname + PID)
	hostname, _ := os.Hostname()
	consumerName := fmt.Sprintf("%s-%d", hostname, os.Getpid())

	return &LocationEnrichmentWorker{
		BaseWorker:   worker.NewBaseWorker("location-enrichment", consumerGroup, logger),
		streamRepo:   streamRepo,
		enrichmentUC: enrichmentUC,
		consumerName: consumerName,
		maxRetries:   maxRetries,
	}
}

// Start запускает воркер
func (w *LocationEnrichmentWorker) Start(ctx context.Context) error {
	logger := w.Logger()
	logger.Info("Starting LocationEnrichmentWorker",
		zap.String("consumer_group", w.ConsumerGroup()),
		zap.String("consumer_name", w.consumerName))

	// Создаем consumer group, если его нет
	if err := w.streamRepo.CreateConsumerGroup(ctx, domain.StreamLocationEnrich, w.ConsumerGroup()); err != nil {
		logger.Error("Failed to create consumer group", zap.Error(err))
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	// Подписываемся на стрим
	msgChan, err := w.streamRepo.ConsumeStream(
		ctx,
		domain.StreamLocationEnrich,
		w.ConsumerGroup(),
		w.consumerName,
	)
	if err != nil {
		logger.Error("Failed to consume stream", zap.Error(err))
		return fmt.Errorf("failed to consume stream: %w", err)
	}

	// Обрабатываем сообщения
	for {
		select {
		case <-w.StopChan():
			logger.Info("Worker stopped")
			return nil

		case <-ctx.Done():
			logger.Info("Context cancelled")
			return ctx.Err()

		case msg, ok := <-msgChan:
			if !ok {
				logger.Warn("Message channel closed")
				return fmt.Errorf("message channel closed")
			}

			// Обрабатываем сообщение
			if err := w.processMessage(ctx, msg); err != nil {
				logger.Error("Failed to process message",
					zap.String("message_id", msg.ID),
					zap.Error(err))
				// Не останавливаем воркер при ошибке обработки - продолжаем работать
				continue
			}

			// Подтверждаем обработку
			if err := w.streamRepo.AckMessage(ctx, domain.StreamLocationEnrich, w.ConsumerGroup(), msg.ID); err != nil {
				logger.Error("Failed to acknowledge message",
					zap.String("message_id", msg.ID),
					zap.Error(err))
			}
		}
	}
}

// processMessage обрабатывает одно сообщение
func (w *LocationEnrichmentWorker) processMessage(ctx context.Context, msg domain.StreamMessage) error {
	logger := w.Logger()

	// Десериализуем событие
	var event domain.LocationEnrichEvent
	if err := json.Unmarshal([]byte(msg.Data), &event); err != nil {
		logger.Error("Failed to unmarshal event",
			zap.String("message_id", msg.ID),
			zap.String("raw_data", msg.Data),
			zap.Error(err))
		// Не публикуем событие с ошибкой для неизвестного property_id
		// Сообщение будет ACK'нуто и пропущено (dead letter pattern)
		return nil
	}

	logger.Info("Processing location enrichment",
		zap.String("property_id", event.PropertyID.String()),
		zap.String("country", event.Country))

	// Обогащаем локацию
	result, err := w.enrichmentUC.EnrichLocation(ctx, &event)
	if err != nil {
		logger.Error("Failed to enrich location",
			zap.String("property_id", event.PropertyID.String()),
			zap.Error(err))
		// Публикуем ответ с ошибкой
		w.publishError(ctx, event.PropertyID, fmt.Sprintf("failed to enrich location: %v", err))
		return nil // Не возвращаем ошибку, чтобы ACK сообщение
	}

	// Публикуем результат
	if err := w.streamRepo.PublishToStream(ctx, domain.StreamLocationDone, result); err != nil {
		logger.Error("Failed to publish result",
			zap.String("property_id", event.PropertyID.String()),
			zap.Error(err))
		return fmt.Errorf("failed to publish result: %w", err)
	}

	logger.Info("Location enriched successfully",
		zap.String("property_id", event.PropertyID.String()),
		zap.Bool("has_enriched_location", result.EnrichedLocation != nil),
		zap.Int("nearest_transport_count", len(result.NearestTransport)))

	return nil
}

// publishError публикует событие с ошибкой
func (w *LocationEnrichmentWorker) publishError(ctx context.Context, propertyID uuid.UUID, errorMsg string) {
	logger := w.Logger()

	result := domain.LocationDoneEvent{
		PropertyID: propertyID,
		Error:      errorMsg,
	}

	if err := w.streamRepo.PublishToStream(ctx, domain.StreamLocationDone, &result); err != nil {
		logger.Error("Failed to publish error",
			zap.Error(err))
	}
}
