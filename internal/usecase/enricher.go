package usecase

import (
	"context"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/usecase/dto"
)

// LocationEnricher defines the interface for location enrichment
type LocationEnricher interface {
	EnrichLocation(ctx context.Context, event *domain.LocationEnrichEvent) (*domain.LocationDoneEvent, error)
}

// BatchLocationEnricher defines the interface for batch location enrichment
type BatchLocationEnricher interface {
	EnrichLocationBatch(ctx context.Context, req dto.EnrichLocationBatchRequest) (*dto.EnrichLocationBatchResponse, error)
}

// Ensure EnrichedLocationUseCase implements BatchLocationEnricher
var _ BatchLocationEnricher = (*EnrichedLocationUseCase)(nil)
