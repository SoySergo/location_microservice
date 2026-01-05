package usecase

import (
	"context"

	"github.com/location-microservice/internal/domain"
)

// LocationEnricher defines the interface for location enrichment
type LocationEnricher interface {
	EnrichLocation(ctx context.Context, event *domain.LocationEnrichEvent) (*domain.LocationDoneEvent, error)
}
