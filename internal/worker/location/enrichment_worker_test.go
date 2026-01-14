package location_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/usecase/dto"
	"github.com/location-microservice/internal/worker/location"
)

// MockStreamRepository is a mock of StreamRepository
type MockStreamRepository struct {
	mock.Mock
}

func (m *MockStreamRepository) ConsumeStream(ctx context.Context, stream, group, consumer string) (<-chan domain.StreamMessage, error) {
	args := m.Called(ctx, stream, group, consumer)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(<-chan domain.StreamMessage), args.Error(1)
}

func (m *MockStreamRepository) ConsumeBatch(ctx context.Context, stream, group, consumer string, maxCount int) ([]domain.StreamMessage, error) {
	args := m.Called(ctx, stream, group, consumer, maxCount)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.StreamMessage), args.Error(1)
}

func (m *MockStreamRepository) AckMessage(ctx context.Context, stream, group, messageID string) error {
	args := m.Called(ctx, stream, group, messageID)
	return args.Error(0)
}

func (m *MockStreamRepository) AckMessages(ctx context.Context, stream, group string, messageIDs []string) error {
	args := m.Called(ctx, stream, group, messageIDs)
	return args.Error(0)
}

func (m *MockStreamRepository) CreateConsumerGroup(ctx context.Context, stream, group string) error {
	args := m.Called(ctx, stream, group)
	return args.Error(0)
}

func (m *MockStreamRepository) PublishToStream(ctx context.Context, stream string, data interface{}) error {
	args := m.Called(ctx, stream, data)
	return args.Error(0)
}

// MockEnrichedLocationUseCase is a mock of EnrichedLocationUseCase
type MockEnrichedLocationUseCase struct {
	mock.Mock
}

func (m *MockEnrichedLocationUseCase) EnrichLocationBatch(ctx context.Context, req dto.EnrichLocationBatchRequest) (*dto.EnrichLocationBatchResponse, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.EnrichLocationBatchResponse), args.Error(1)
}

// TestLocationEnrichmentWorker_Name tests the worker name
func TestLocationEnrichmentWorker_Name(t *testing.T) {
	mockStream := &MockStreamRepository{}
	mockUseCase := &MockEnrichedLocationUseCase{}
	logger := zap.NewNop()

	worker := location.NewLocationEnrichmentWorker(
		mockStream,
		mockUseCase,
		"test-group",
		3,
		logger,
	)

	assert.Equal(t, "location-enrichment", worker.Name())
}

// TestLocationEnrichmentWorker_Stop tests graceful stop
func TestLocationEnrichmentWorker_Stop(t *testing.T) {
	mockStream := &MockStreamRepository{}
	mockUseCase := &MockEnrichedLocationUseCase{}
	logger := zap.NewNop()

	worker := location.NewLocationEnrichmentWorker(
		mockStream,
		mockUseCase,
		"test-group",
		3,
		logger,
	)

	// Stop should not error even if not started
	err := worker.Stop()
	assert.NoError(t, err)

	// Calling stop multiple times should be safe
	err = worker.Stop()
	assert.NoError(t, err)
}

// TestLocationEnrichmentWorker_ContextCancellation tests worker stops on context cancellation
func TestLocationEnrichmentWorker_ContextCancellation(t *testing.T) {
	mockStream := &MockStreamRepository{}
	mockUseCase := &MockEnrichedLocationUseCase{}
	logger := zap.NewNop()

	worker := location.NewLocationEnrichmentWorker(
		mockStream,
		mockUseCase,
		"test-group",
		3,
		logger,
	)

	// Mock CreateConsumerGroup
	mockStream.On("CreateConsumerGroup", mock.Anything, domain.StreamLocationEnrich, "test-group").
		Return(nil)

	// Mock ConsumeBatch to return empty messages (simulating empty queue)
	mockStream.On("ConsumeBatch", mock.Anything, domain.StreamLocationEnrich, "test-group", mock.AnythingOfType("string"), 20).
		Return([]domain.StreamMessage{}, nil)

	ctx, cancel := context.WithCancel(context.Background())

	// Start worker in goroutine
	done := make(chan error, 1)
	go func() {
		done <- worker.Start(ctx)
	}()

	// Give it time to start
	time.Sleep(200 * time.Millisecond)

	// Cancel context
	cancel()

	// Worker should stop
	select {
	case err := <-done:
		assert.Error(t, err) // Should return context.Canceled
		assert.Equal(t, context.Canceled, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Worker did not stop on context cancellation")
	}

	mockStream.AssertExpectations(t)
}

// TestLocationEnrichmentWorker_BatchProcessing tests batch message processing
func TestLocationEnrichmentWorker_BatchProcessing(t *testing.T) {
	mockStream := &MockStreamRepository{}
	mockUseCase := &MockEnrichedLocationUseCase{}
	logger := zap.NewNop()

	worker := location.NewLocationEnrichmentWorker(
		mockStream,
		mockUseCase,
		"test-group",
		3,
		logger,
	)

	// Create test data
	propertyID1 := uuid.New()
	propertyID2 := uuid.New()
	
	city := "Barcelona"
	lat1 := 41.3851
	lon1 := 2.1734
	lat2 := 41.3900
	lon2 := 2.1800

	event1 := &domain.LocationEnrichEvent{
		PropertyID: propertyID1,
		Country:    "Spain",
		City:       &city,
		Latitude:   &lat1,
		Longitude:  &lon1,
	}

	event2 := &domain.LocationEnrichEvent{
		PropertyID: propertyID2,
		Country:    "Spain",
		City:       &city,
		Latitude:   &lat2,
		Longitude:  &lon2,
	}

	// Marshal events
	eventJSON1, _ := json.Marshal(event1)
	eventJSON2, _ := json.Marshal(event2)

	messages := []domain.StreamMessage{
		{
			ID:     "1234567890-0",
			Stream: domain.StreamLocationEnrich,
			Data: map[string]interface{}{
				"data": string(eventJSON1),
			},
		},
		{
			ID:     "1234567890-1",
			Stream: domain.StreamLocationEnrich,
			Data: map[string]interface{}{
				"data": string(eventJSON2),
			},
		},
	}

	// Mock CreateConsumerGroup
	mockStream.On("CreateConsumerGroup", mock.Anything, domain.StreamLocationEnrich, "test-group").
		Return(nil)

	// Mock ConsumeBatch - first call returns messages, second call returns empty (to simulate stop)
	mockStream.On("ConsumeBatch", mock.Anything, domain.StreamLocationEnrich, "test-group", mock.AnythingOfType("string"), 20).
		Return(messages, nil).Once()
	mockStream.On("ConsumeBatch", mock.Anything, domain.StreamLocationEnrich, "test-group", mock.AnythingOfType("string"), 20).
		Return([]domain.StreamMessage{}, nil)

	// Mock EnrichLocationBatch
	mockUseCase.On("EnrichLocationBatch", mock.Anything, mock.MatchedBy(func(req dto.EnrichLocationBatchRequest) bool {
		return len(req.Locations) == 2
	})).Return(&dto.EnrichLocationBatchResponse{
		Results: []dto.EnrichedLocationResult{
			{
				Index: 0,
				EnrichedLocation: &dto.EnrichedLocationDTO{
					IsAddressVisible: ptrBool(true),
				},
				Error: "",
			},
			{
				Index: 1,
				EnrichedLocation: &dto.EnrichedLocationDTO{
					IsAddressVisible: ptrBool(true),
				},
				Error: "",
			},
		},
		Meta: dto.EnrichLocationBatchMeta{
			TotalLocations: 2,
			SuccessCount:   2,
			ErrorCount:     0,
		},
	}, nil)

	// Mock PublishToStream for both results
	mockStream.On("PublishToStream", mock.Anything, domain.StreamLocationDone, mock.MatchedBy(func(event *domain.LocationDoneEvent) bool {
		return event.PropertyID == propertyID1 || event.PropertyID == propertyID2
	})).Return(nil).Twice()

	// Mock AckMessages
	mockStream.On("AckMessages", mock.Anything, domain.StreamLocationEnrich, "test-group", []string{"1234567890-0", "1234567890-1"}).
		Return(nil)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Start worker in goroutine
	done := make(chan error, 1)
	go func() {
		done <- worker.Start(ctx)
	}()

	// Wait for timeout or completion
	select {
	case <-done:
		// Worker stopped
	case <-time.After(1 * time.Second):
		t.Fatal("Worker did not stop in time")
	}

	// Verify expectations
	mockStream.AssertExpectations(t)
	mockUseCase.AssertExpectations(t)
}

// Helper functions
func ptrBool(v bool) *bool {
	return &v
}
