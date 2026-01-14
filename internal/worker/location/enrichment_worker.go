package location

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
	"github.com/location-microservice/internal/worker"
	"go.uber.org/zap"
)

const (
	maxBatchSize    = 20                      // максимум сообщений за раз
	emptyQueueSleep = 100 * time.Millisecond // пауза если очередь пуста
)

// LocationEnrichmentWorker обрабатывает события обогащения локаций
type LocationEnrichmentWorker struct {
	*worker.BaseWorker
	streamRepo         repository.StreamRepository
	enrichedLocationUC *usecase.EnrichedLocationUseCase
	consumerName       string
	maxRetries         int
}

// NewLocationEnrichmentWorker создает новый LocationEnrichmentWorker
func NewLocationEnrichmentWorker(
	streamRepo repository.StreamRepository,
	enrichedLocationUC *usecase.EnrichedLocationUseCase,
	consumerGroup string,
	maxRetries int,
	logger *zap.Logger,
) *LocationEnrichmentWorker {
	hostname, _ := os.Hostname()
	consumerName := fmt.Sprintf("%s-%d", hostname, os.Getpid())

	return &LocationEnrichmentWorker{
		BaseWorker:         worker.NewBaseWorker("location-enrichment", consumerGroup, logger),
		streamRepo:         streamRepo,
		enrichedLocationUC: enrichedLocationUC,
		consumerName:       consumerName,
		maxRetries:         maxRetries,
	}
}

// Start запускает воркер
func (w *LocationEnrichmentWorker) Start(ctx context.Context) error {
	logger := w.Logger()
	logger.Info("Starting LocationEnrichmentWorker (batch mode)",
		zap.String("consumer_group", w.ConsumerGroup()),
		zap.String("consumer_name", w.consumerName),
		zap.Int("max_batch_size", maxBatchSize))

	// Создаем consumer group
	if err := w.streamRepo.CreateConsumerGroup(ctx, domain.StreamLocationEnrich, w.ConsumerGroup()); err != nil {
		logger.Error("Failed to create consumer group", zap.Error(err))
		return fmt.Errorf("failed to create consumer group: %w", err)
	}

	// Основной цикл обработки
	for {
		select {
		case <-w.StopChan():
			logger.Info("Worker stopped")
			return nil

		case <-ctx.Done():
			logger.Info("Context cancelled")
			return ctx.Err()

		default:
			// Обрабатываем batch сообщений
			processed, err := w.processBatch(ctx)
			if err != nil {
				logger.Error("Failed to process batch", zap.Error(err))
				time.Sleep(time.Second) // пауза при ошибке
				continue
			}

			// Если ничего не обработали - короткая пауза
			if processed == 0 {
				time.Sleep(emptyQueueSleep)
			}
		}
	}
}

// processBatch читает и обрабатывает batch сообщений
// Возвращает количество обработанных сообщений
func (w *LocationEnrichmentWorker) processBatch(ctx context.Context) (int, error) {
	logger := w.Logger()

	// 1. Читаем до 20 сообщений (неблокирующий режим)
	messages, err := w.streamRepo.ConsumeBatch(
		ctx,
		domain.StreamLocationEnrich,
		w.ConsumerGroup(),
		w.consumerName,
		maxBatchSize,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to consume batch: %w", err)
	}

	if len(messages) == 0 {
		return 0, nil // очередь пуста
	}

	logger.Info("Processing batch",
		zap.Int("message_count", len(messages)))

	// 2. Парсим события
	events := make([]*domain.LocationEnrichEvent, 0, len(messages))
	messageIDs := make([]string, 0, len(messages))

	for _, msg := range messages {
		event, err := w.parseMessage(msg)
		if err != nil {
			logger.Warn("Failed to parse message, skipping",
				zap.String("message_id", msg.ID),
				zap.Error(err))
			// ACK битое сообщение чтобы не застревало
			_ = w.streamRepo.AckMessage(ctx, domain.StreamLocationEnrich, w.ConsumerGroup(), msg.ID)
			continue
		}

		events = append(events, event)
		messageIDs = append(messageIDs, msg.ID)
	}

	if len(events) == 0 {
		return len(messages), nil // все сообщения были битые
	}

	// 3. Конвертируем в batch request
	locations := make([]dto.LocationInput, len(events))
	for i, event := range events {
		locations[i] = dto.LocationInput{
			Index:        i,
			Country:      event.Country,
			Region:       event.Region,
			Province:     event.Province,
			City:         event.City,
			District:     event.District,
			Neighborhood: event.Neighborhood,
			Latitude:     event.Latitude,
			Longitude:    event.Longitude,
			IsVisible:    event.IsVisible,
		}
	}

	req := dto.EnrichLocationBatchRequest{
		Locations: locations,
	}

	// 4. Вызываем batch обогащение
	resp, err := w.enrichedLocationUC.EnrichLocationBatch(ctx, req)
	if err != nil {
		logger.Error("EnrichLocationBatch failed", zap.Error(err))
		return 0, fmt.Errorf("enrichment failed: %w", err)
	}

	// 5. Публикуем результаты в stream:location:done
	for i, result := range resp.Results {
		if i >= len(events) {
			break
		}
		event := events[i]

		doneEvent := w.buildDoneEvent(event.PropertyID, result)

		if err := w.streamRepo.PublishToStream(ctx, domain.StreamLocationDone, doneEvent); err != nil {
			logger.Error("Failed to publish done event",
				zap.String("property_id", event.PropertyID.String()),
				zap.Error(err))
			// Продолжаем с остальными
		}
	}

	// 6. ACK всех обработанных сообщений
	if err := w.streamRepo.AckMessages(ctx, domain.StreamLocationEnrich, w.ConsumerGroup(), messageIDs); err != nil {
		logger.Error("Failed to ack messages", zap.Error(err))
		// Не критично - сообщения будут переобработаны
	}

	logger.Info("Batch processed successfully",
		zap.Int("processed", len(events)),
		zap.Int("success", resp.Meta.SuccessCount),
		zap.Int("errors", resp.Meta.ErrorCount))

	return len(messages), nil
}

// parseMessage парсит сообщение из стрима в LocationEnrichEvent
func (w *LocationEnrichmentWorker) parseMessage(msg domain.StreamMessage) (*domain.LocationEnrichEvent, error) {
	data, ok := msg.Data["data"].(string)
	if !ok {
		return nil, fmt.Errorf("missing or invalid 'data' field")
	}

	var event domain.LocationEnrichEvent
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return nil, fmt.Errorf("failed to unmarshal event: %w", err)
	}

	return &event, nil
}

// buildDoneEvent создает LocationDoneEvent из результата обогащения
func (w *LocationEnrichmentWorker) buildDoneEvent(
	propertyID uuid.UUID,
	result dto.EnrichedLocationResult,
) *domain.LocationDoneEvent {
	doneEvent := &domain.LocationDoneEvent{
		PropertyID: propertyID,
		Error:      result.Error,
	}

	if result.EnrichedLocation != nil {
		doneEvent.EnrichedLocation = w.convertEnrichedLocation(result.EnrichedLocation)
	}

	if len(result.NearestTransport) > 0 {
		doneEvent.NearestTransport = w.convertNearestStations(result.NearestTransport)
	}

	return doneEvent
}

// convertEnrichedLocation конвертирует DTO в domain
func (w *LocationEnrichmentWorker) convertEnrichedLocation(dto *dto.EnrichedLocationDTO) *domain.EnrichedLocation {
	if dto == nil {
		return nil
	}

	result := &domain.EnrichedLocation{
		IsAddressVisible: dto.IsAddressVisible,
	}

	if dto.Country != nil {
		result.Country = &domain.BoundaryInfo{
			ID:             dto.Country.ID,
			Name:           dto.Country.Name,
			TranslateNames: dto.Country.TranslateNames,
		}
	}
	if dto.Region != nil {
		result.Region = &domain.BoundaryInfo{
			ID:             dto.Region.ID,
			Name:           dto.Region.Name,
			TranslateNames: dto.Region.TranslateNames,
		}
	}
	if dto.Province != nil {
		result.Province = &domain.BoundaryInfo{
			ID:             dto.Province.ID,
			Name:           dto.Province.Name,
			TranslateNames: dto.Province.TranslateNames,
		}
	}
	if dto.City != nil {
		result.City = &domain.BoundaryInfo{
			ID:             dto.City.ID,
			Name:           dto.City.Name,
			TranslateNames: dto.City.TranslateNames,
		}
	}
	if dto.District != nil {
		result.District = &domain.BoundaryInfo{
			ID:             dto.District.ID,
			Name:           dto.District.Name,
			TranslateNames: dto.District.TranslateNames,
		}
	}
	if dto.Neighborhood != nil {
		result.Neighborhood = &domain.BoundaryInfo{
			ID:             dto.Neighborhood.ID,
			Name:           dto.Neighborhood.Name,
			TranslateNames: dto.Neighborhood.TranslateNames,
		}
	}

	return result
}

// convertNearestStations конвертирует DTO станций в domain
func (w *LocationEnrichmentWorker) convertNearestStations(stations []dto.PriorityTransportStation) []domain.NearestStation {
	result := make([]domain.NearestStation, len(stations))
	for i, s := range stations {
		result[i] = domain.NearestStation{
			StationID:       s.StationID,
			Name:            s.Name,
			Type:            s.Type,
			Lat:             s.Lat,
			Lon:             s.Lon,
			Distance:        s.LinearDistance,
			WalkingDistance: &s.WalkingDistance,
			WalkingDuration: &s.WalkingTime,
		}

		if len(s.Lines) > 0 {
			result[i].Lines = make([]domain.TransportLineInfo, len(s.Lines))
			for j, l := range s.Lines {
				result[i].Lines[j] = domain.TransportLineInfo{
					ID:    l.ID,
					Name:  l.Name,
					Ref:   l.Ref,
					Type:  l.Type,
					Color: l.Color,
				}
			}
		}
	}
	return result
}
