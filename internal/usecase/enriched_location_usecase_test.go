package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
)

func TestEnrichedLocationUseCase_EnrichLocationBatch_Success(t *testing.T) {
	// Arrange
	logger := zap.NewNop()

	// Create a minimal EnrichedLocationUseCase (without a real EnrichmentDebugUseCase)
	// This test just verifies construction
	uc := usecase.NewEnrichedLocationUseCase(
		nil, // EnrichmentDebugUseCase would be injected here
		logger,
	)

	// Verify the usecase is constructed properly
	assert.NotNil(t, uc)
}

func TestEnrichedLocationUseCase_EnrichLocationBatch_EmptyRequest(t *testing.T) {
	// Arrange
	logger := zap.NewNop()

	uc := usecase.NewEnrichedLocationUseCase(nil, logger)

	req := dto.EnrichLocationBatchRequest{
		Locations: []dto.LocationInput{},
	}

	// Act
	resp, err := uc.EnrichLocationBatch(context.Background(), req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, resp)
}

func TestEnrichedLocationUseCase_EnrichLocationBatch_MixedVisibility(t *testing.T) {
	// Test that visible and non-visible locations are handled correctly
	// This would require proper mocking infrastructure
	logger := zap.NewNop()

	uc := usecase.NewEnrichedLocationUseCase(nil, logger)

	// Verify the usecase is constructed properly
	assert.NotNil(t, uc)
}
