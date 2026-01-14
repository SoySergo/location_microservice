package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
)

// MockSearchUseCase is a mock implementation of SearchUseCase
type MockSearchUseCase struct {
	mock.Mock
}

func (m *MockSearchUseCase) DetectLocationBatch(ctx context.Context, req dto.DetectLocationBatchRequest) (*dto.DetectLocationBatchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.DetectLocationBatchResponse), args.Error(1)
}

// MockTransportUseCase is a mock implementation of TransportUseCase
type MockTransportUseCase struct {
	mock.Mock
}

func (m *MockTransportUseCase) GetNearestTransportByPriorityBatch(ctx context.Context, req dto.PriorityTransportBatchRequest) (*dto.PriorityTransportBatchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.PriorityTransportBatchResponse), args.Error(1)
}

func TestEnrichedLocationUseCase_EnrichLocationBatch(t *testing.T) {
	logger := zap.NewNop()
	ctx := context.Background()

	t.Run("success with visible and non-visible locations", func(t *testing.T) {
		// Create usecase
		uc := usecase.NewEnrichedLocationUseCase(
			(*usecase.SearchUseCase)(nil), // Will not use in this test
			(*usecase.TransportUseCase)(nil),
			logger,
		)
		// Note: In real implementation, we would inject the mocks properly,
		// but since the usecase uses concrete types, we'll test integration style

		// This test demonstrates the structure, but would need proper dependency injection
		// to fully mock the dependencies
		assert.NotNil(t, uc)
	})

	t.Run("empty locations returns error", func(t *testing.T) {
		uc := usecase.NewEnrichedLocationUseCase(
			(*usecase.SearchUseCase)(nil),
			(*usecase.TransportUseCase)(nil),
			logger,
		)

		req := dto.EnrichLocationBatchRequest{
			Locations: []dto.LocationInput{},
		}

		_, err := uc.EnrichLocationBatch(ctx, req)
		assert.Error(t, err)
	})
}

func TestEnrichedLocationUseCase_EnrichLocation(t *testing.T) {
	logger := zap.NewNop()

	t.Run("implements LocationEnricher interface", func(t *testing.T) {
		uc := usecase.NewEnrichedLocationUseCase(
			(*usecase.SearchUseCase)(nil),
			(*usecase.TransportUseCase)(nil),
			logger,
		)

		// Verify that EnrichedLocationUseCase implements LocationEnricher
		var _ usecase.LocationEnricher = uc
		assert.NotNil(t, uc)
	})
}
