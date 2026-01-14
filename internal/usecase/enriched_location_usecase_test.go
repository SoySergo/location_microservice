package usecase_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
)

// NOTE: EnrichedLocationUseCase uses concrete SearchUseCase and TransportUseCase types,
// so we test it integration-style with real usecases backed by mocked repositories.

// Helper functions
func ptrString(s string) *string {
	return &s
}

func ptrFloat64(f float64) *float64 {
	return &f
}

func ptrBool(b bool) *bool {
	return &b
}

func TestEnrichedLocationUseCase_EnrichLocationBatch_EmptyLocations(t *testing.T) {
	// Test empty locations returns error
	logger := zap.NewNop()
	mockBoundary := &MockBoundaryRepository{}
	mockCache := &MockCacheRepository{}
	mockTransport := &MockTransportRepository{}

	searchUC := usecase.NewSearchUseCase(mockBoundary, mockCache, logger, 1*time.Hour)
	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	uc := usecase.NewEnrichedLocationUseCase(searchUC, transportUC, logger)
	ctx := context.Background()

	req := dto.EnrichLocationBatchRequest{
		Locations: []dto.LocationInput{},
	}

	// Act
	result, err := uc.EnrichLocationBatch(ctx, req)

	// Assert
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestEnrichedLocationUseCase_EnrichLocationBatch_Success(t *testing.T) {
	// Arrange
	logger := zap.NewNop()
	mockBoundary := &MockBoundaryRepository{}
	mockCache := &MockCacheRepository{}
	mockTransport := &MockTransportRepository{}
	ctx := context.Background()

	searchUC := usecase.NewSearchUseCase(mockBoundary, mockCache, logger, 1*time.Hour)
	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	uc := usecase.NewEnrichedLocationUseCase(searchUC, transportUC, logger)

	searchUC := usecase.NewSearchUseCase(mockBoundary, mockCache, logger, 1*time.Hour)
	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	uc := usecase.NewEnrichedLocationUseCase(searchUC, transportUC, logger)

	// Input: one visible location
	locations := []dto.LocationInput{
		{
			Index:     0,
			Country:   "Spain",
			Latitude:  ptrFloat64(41.3851),
			Longitude: ptrFloat64(2.1734),
			IsVisible: ptrBool(true),
		},
	}

	req := dto.EnrichLocationBatchRequest{
		Locations: locations,
	}

	// Mock GetByPointBatch for DetectLocationBatch
	boundariesByPoint := map[int][]*domain.AdminBoundary{
		0: {
			{ID: 1, AdminLevel: 2, Name: "Spain", NameEn: "Spain"},
			{ID: 100, AdminLevel: 8, Name: "Barcelona", NameEn: "Barcelona"},
		},
	}

	mockBoundary.On("GetByPointBatch", ctx, mock.MatchedBy(func(points []domain.LatLon) bool {
		return len(points) == 1 && points[0].Lat == 41.3851
	})).Return(boundariesByPoint, nil)

	// Mock GetNearestTransportByPriorityBatch
	batchResults := []domain.BatchTransportResult{
		{
			PointIndex: 0,
			SearchPoint: domain.TransportSearchPoint{
				Lat:   41.3851,
				Lon:   2.1734,
				Limit: 5,
			},
			Stations: []domain.NearestTransportWithLines{
				{
					StationID: 500,
					Name:      "Passeig de Gràcia",
					Type:      "metro",
					Lat:       41.3950,
					Lon:       2.1640,
					Distance:  250.5,
					Lines: []domain.TransportLineInfo{
						{ID: 1, Name: "L1", Type: "metro"},
					},
				},
			},
		},
	}

	mockTransport.On("GetNearestTransportByPriorityBatch", ctx,
		mock.MatchedBy(func(points []domain.TransportSearchPoint) bool {
			return len(points) == 1
		}), 1500.0, 5).
		Return(batchResults, nil)

	// Act
	result, err := uc.EnrichLocationBatch(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Results, 1)
	assert.Equal(t, 1, result.Meta.SuccessCount)
	assert.Equal(t, 0, result.Meta.ErrorCount)
	assert.Equal(t, 1, result.Meta.WithTransport)

	// Check location result
	assert.Equal(t, 0, result.Results[0].Index)
	assert.NotNil(t, result.Results[0].EnrichedLocation)
	assert.Equal(t, "Spain", result.Results[0].EnrichedLocation.Country.Name)
	assert.Equal(t, "Barcelona", result.Results[0].EnrichedLocation.City.Name)
	assert.Len(t, result.Results[0].NearestTransport, 1)
	assert.Equal(t, int64(500), result.Results[0].NearestTransport[0].StationID)

	mockBoundary.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

func TestEnrichedLocationUseCase_EnrichLocationBatch_WithoutVisibleLocations(t *testing.T) {
	// Test that transport is NOT called for non-visible locations
	logger := zap.NewNop()
	mockBoundary := &MockBoundaryRepository{}
	mockCache := &MockCacheRepository{}
	mockTransport := &MockTransportRepository{}
	ctx := context.Background()

	searchUC := usecase.NewSearchUseCase(mockBoundary, mockCache, logger, 1*time.Hour)
	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	uc := usecase.NewEnrichedLocationUseCase(searchUC, transportUC, logger)

	// Input: non-visible location (no coordinates or IsVisible=false)
	locations := []dto.LocationInput{
		{
			Index:   0,
			Country: "Spain",
			City:    ptrString("Barcelona"),
			// No coordinates, or IsVisible=false
		},
	}

	req := dto.EnrichLocationBatchRequest{
		Locations: locations,
	}

	// Mock SearchByTextBatch for name-based location
	searchResults := []domain.BoundarySearchResult{
		{
			Index:    2, // loc.Index*100 + 2 (country)
			Found:    true,
			Boundary: &domain.AdminBoundary{ID: 1, AdminLevel: 2, Name: "Spain"},
		},
		{
			Index:    8, // loc.Index*100 + 8 (city)
			Found:    true,
			Boundary: &domain.AdminBoundary{ID: 100, AdminLevel: 8, Name: "Barcelona"},
		},
	}

	mockBoundary.On("SearchByTextBatch", ctx, mock.MatchedBy(func(requests []domain.BoundarySearchRequest) bool {
		return len(requests) >= 1
	})).Return(searchResults, nil)

	// Transport should NOT be called

	// Act
	result, err := uc.EnrichLocationBatch(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, result.Meta.WithTransport) // No visible locations

	mockBoundary.AssertExpectations(t)
	mockTransport.AssertNotCalled(t, "GetNearestTransportByPriorityBatch")
}

func TestEnrichedLocationUseCase_EnrichLocationBatch_TransportError(t *testing.T) {
	// Test: transport error should not block detection result
	logger := zap.NewNop()
	mockBoundary := &MockBoundaryRepository{}
	mockCache := &MockCacheRepository{}
	mockTransport := &MockTransportRepository{}
	ctx := context.Background()

	searchUC := usecase.NewSearchUseCase(mockBoundary, mockCache, logger, 1*time.Hour)
	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	uc := usecase.NewEnrichedLocationUseCase(searchUC, transportUC, logger)

	locations := []dto.LocationInput{
		{
			Index:     0,
			Country:   "Spain",
			Latitude:  ptrFloat64(41.3851),
			Longitude: ptrFloat64(2.1734),
			IsVisible: ptrBool(true),
		},
	}

	req := dto.EnrichLocationBatchRequest{
		Locations: locations,
	}

	// Mock GetByPointBatch
	boundariesByPoint := map[int][]*domain.AdminBoundary{
		0: {
			{ID: 1, AdminLevel: 2, Name: "Spain"},
		},
	}

	mockBoundary.On("GetByPointBatch", ctx, mock.Anything).Return(boundariesByPoint, nil)

	// Mock transport error
	mockTransport.On("GetNearestTransportByPriorityBatch", ctx, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	// Act
	result, err := uc.EnrichLocationBatch(ctx, req)

	// Assert
	assert.NoError(t, err) // Should not fail
	assert.NotNil(t, result)
	assert.Equal(t, 1, result.Meta.SuccessCount)
	// Detection should succeed, but no transport data
	assert.NotNil(t, result.Results[0].EnrichedLocation)
	assert.Empty(t, result.Results[0].NearestTransport)

	mockBoundary.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

func TestEnrichedLocationUseCase_ImplementsInterfaces(t *testing.T) {
	// Test that EnrichedLocationUseCase implements the required interfaces
	logger := zap.NewNop()
	mockBoundary := &MockBoundaryRepository{}
	mockCache := &MockCacheRepository{}
	mockTransport := &MockTransportRepository{}

	searchUC := usecase.NewSearchUseCase(mockBoundary, mockCache, logger, 1*time.Hour)
	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	uc := usecase.NewEnrichedLocationUseCase(searchUC, transportUC, logger)

	// Verify interfaces
	var _ usecase.BatchLocationEnricher = uc
	var _ usecase.LocationEnricher = uc
	assert.NotNil(t, uc)
}
