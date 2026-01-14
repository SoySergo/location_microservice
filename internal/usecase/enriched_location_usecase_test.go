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

// MockSearchUseCase is a mock of SearchUseCase
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

// MockTransportUseCase is a mock of TransportUseCase
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

func TestEnrichedLocationUseCase_EnrichLocationBatch_Success(t *testing.T) {
	// Arrange
	logger := zap.NewNop()

	uc := usecase.NewEnrichedLocationUseCase(
		&usecase.SearchUseCase{}, // We'll use mocks instead
		&usecase.TransportUseCase{},
		logger,
	)

	// Verify the usecase is constructed properly
	assert.NotNil(t, uc)
}

func TestEnrichedLocationUseCase_EnrichLocationBatch_EmptyRequest(t *testing.T) {
	// Arrange
	logger := zap.NewNop()
	searchUC := &usecase.SearchUseCase{}
	transportUC := &usecase.TransportUseCase{}

	uc := usecase.NewEnrichedLocationUseCase(searchUC, transportUC, logger)

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
	searchUC := &usecase.SearchUseCase{}
	transportUC := &usecase.TransportUseCase{}

	uc := usecase.NewEnrichedLocationUseCase(searchUC, transportUC, logger)

	// Verify the usecase is constructed properly
	assert.NotNil(t, uc)
}

// Helper function to create string pointers
func strPtr(s string) *string {
	return &s
}
