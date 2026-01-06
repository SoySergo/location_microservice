package usecase

import (
	"context"

	"github.com/location-microservice/internal/domain"
	"go.uber.org/zap"
)

// EnrichmentUseCaseExtended - расширенный use case для обогащения с инфраструктурой
type EnrichmentUseCaseExtended struct {
	*EnrichmentUseCase
	infraUseCase    *InfrastructureUseCase
	transportRadius float64
	logger          *zap.Logger
}

// NewEnrichmentUseCaseExtended создает новый EnrichmentUseCaseExtended
func NewEnrichmentUseCaseExtended(
	baseUseCase *EnrichmentUseCase,
	infraUseCase *InfrastructureUseCase,
	transportRadius float64,
	logger *zap.Logger,
) *EnrichmentUseCaseExtended {
	return &EnrichmentUseCaseExtended{
		EnrichmentUseCase: baseUseCase,
		infraUseCase:      infraUseCase,
		transportRadius:   transportRadius,
		logger:            logger,
	}
}

// EnrichLocationExtended обогащает локацию с инфраструктурой
func (uc *EnrichmentUseCaseExtended) EnrichLocationExtended(
	ctx context.Context,
	event *domain.LocationEnrichEvent,
) (*domain.LocationDoneEventExtended, error) {
	// Сначала выполняем базовое обогащение
	baseResult, err := uc.EnrichmentUseCase.EnrichLocation(ctx, event)
	if err != nil {
		return nil, err
	}

	// Создаем расширенный результат
	result := &domain.LocationDoneEventExtended{
		PropertyID:       baseResult.PropertyID,
		EnrichedLocation: baseResult.EnrichedLocation,
		NearestTransport: baseResult.NearestTransport,
		Error:            baseResult.Error,
	}

	// Проверяем, есть ли полный адрес и координаты
	if event.HasStreetAddress() && event.Latitude != nil && event.Longitude != nil {
		uc.logger.Debug("Property has full street address, fetching infrastructure",
			zap.String("property_id", event.PropertyID.String()))

		// Получаем инфраструктуру
		infrastructure, err := uc.infraUseCase.GetInfrastructure(
			ctx,
			event.PropertyID,
			*event.Latitude,
			*event.Longitude,
			uc.transportRadius,
		)
		if err != nil {
			uc.logger.Warn("Failed to get infrastructure",
				zap.String("property_id", event.PropertyID.String()),
				zap.Error(err))
			// Не считаем критичной ошибкой
		} else {
			result.Infrastructure = infrastructure
		}
	} else {
		uc.logger.Debug("Property has no full street address, skipping infrastructure",
			zap.String("property_id", event.PropertyID.String()),
			zap.Bool("has_street_address", event.HasStreetAddress()),
			zap.Bool("has_coordinates", event.Latitude != nil && event.Longitude != nil))
	}

	return result, nil
}
