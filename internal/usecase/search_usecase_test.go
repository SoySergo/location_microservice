package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
)

// MockCacheRepository is a mock of CacheRepository
type MockCacheRepository struct {
	mock.Mock
}

func (m *MockCacheRepository) Get(ctx context.Context, key string) ([]byte, error) {
	args := m.Called(ctx, key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCacheRepository) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockCacheRepository) Delete(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockCacheRepository) Exists(ctx context.Context, key string) (bool, error) {
	args := m.Called(ctx, key)
	return args.Bool(0), args.Error(1)
}

func (m *MockCacheRepository) GetTile(ctx context.Context, z, x, y int) ([]byte, error) {
	args := m.Called(ctx, z, x, y)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockCacheRepository) SetTile(ctx context.Context, z, x, y int, data []byte, ttl time.Duration) error {
	args := m.Called(ctx, z, x, y, data, ttl)
	return args.Error(0)
}

func (m *MockCacheRepository) GetStats(ctx context.Context) (*domain.Statistics, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Statistics), args.Error(1)
}

func (m *MockCacheRepository) SetStats(ctx context.Context, stats *domain.Statistics, ttl time.Duration) error {
	args := m.Called(ctx, stats, ttl)
	return args.Error(0)
}

func TestSearchUseCase_DetectLocationBatch(t *testing.T) {
	logger := zap.NewNop()
	mockBoundary := &MockBoundaryRepository{}
	mockCache := &MockCacheRepository{}
	ctx := context.Background()

	uc := usecase.NewSearchUseCase(mockBoundary, mockCache, logger, 1*time.Hour)

	t.Run("success with mixed visible and name-based locations", func(t *testing.T) {
		// Location 0: visible (has coordinates) - will use GetByPointBatch
		// Location 1: name-based (no coordinates or not visible) - will use SearchByTextBatch
		locations := []dto.LocationInput{
			{
				Index:     0,
				Country:   "Spain",
				Latitude:  ptrFloat64(41.3851),
				Longitude: ptrFloat64(2.1734),
				IsVisible: ptrBool(true),
			},
			{
				Index:   1,
				Country: "France",
				City:    ptrString("Paris"),
			},
		}

		req := dto.DetectLocationBatchRequest{
			Locations: locations,
		}

		// Mock GetByPointBatch for visible location
		boundariesByPoint := map[int][]*domain.AdminBoundary{
			0: {
				{ID: 1, AdminLevel: 2, Name: "Spain", NameEn: "Spain"},
				{ID: 100, AdminLevel: 8, Name: "Barcelona", NameEn: "Barcelona"},
			},
		}

		mockBoundary.On("GetByPointBatch", ctx, mock.MatchedBy(func(points []domain.LatLon) bool {
			return len(points) == 1 && points[0].Lat == 41.3851
		})).Return(boundariesByPoint, nil)

		// Mock SearchByTextBatch for name-based location
		searchResults := []domain.BoundarySearchResult{
			{
				Index:    100, // loc.Index*100 + 2 (country)
				Found:    true,
				Boundary: &domain.AdminBoundary{ID: 2, AdminLevel: 2, Name: "France", NameEn: "France"},
			},
			{
				Index:    108, // loc.Index*100 + 8 (city)
				Found:    true,
				Boundary: &domain.AdminBoundary{ID: 200, AdminLevel: 8, Name: "Paris", NameEn: "Paris"},
			},
		}

		mockBoundary.On("SearchByTextBatch", ctx, mock.MatchedBy(func(requests []domain.BoundarySearchRequest) bool {
			return len(requests) >= 1 // At least country search
		})).Return(searchResults, nil)

		resp, err := uc.DetectLocationBatch(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Results, 2)
		assert.Equal(t, 2, resp.Meta.SuccessCount)
		assert.Equal(t, 0, resp.Meta.ErrorCount)
		assert.Equal(t, 1, resp.Meta.VisibleCount)
		assert.Equal(t, 1, resp.Meta.NameResolveCount)
		assert.Equal(t, 2, resp.Meta.DBQueriesCount) // 2 queries: GetByPointBatch + SearchByTextBatch

		// Check visible location result
		assert.Equal(t, 0, resp.Results[0].Index)
		assert.NotNil(t, resp.Results[0].EnrichedLocation)
		assert.Equal(t, "Spain", resp.Results[0].EnrichedLocation.Country.Name)
		assert.Equal(t, "Barcelona", resp.Results[0].EnrichedLocation.City.Name)

		// Check name-based location result
		assert.Equal(t, 1, resp.Results[1].Index)
		assert.NotNil(t, resp.Results[1].EnrichedLocation)
		assert.Equal(t, "France", resp.Results[1].EnrichedLocation.Country.Name)
		assert.Equal(t, "Paris", resp.Results[1].EnrichedLocation.City.Name)

		mockBoundary.AssertExpectations(t)
	})

	t.Run("empty locations returns error", func(t *testing.T) {
		req := dto.DetectLocationBatchRequest{
			Locations: []dto.LocationInput{},
		}

		resp, err := uc.DetectLocationBatch(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestSearchUseCase_DetectLocationBatch_VisibleLocations(t *testing.T) {
	// Test for visible locations (reverse geocoding)
	logger := zap.NewNop()
	mockBoundary := &MockBoundaryRepository{}
	mockCache := &MockCacheRepository{}
	ctx := context.Background()

	uc := usecase.NewSearchUseCase(mockBoundary, mockCache, logger, 1*time.Hour)

	t.Run("success with only visible locations", func(t *testing.T) {
		locations := []dto.LocationInput{
			{
				Index:     0,
				Country:   "Spain",
				Latitude:  ptrFloat64(41.3851),
				Longitude: ptrFloat64(2.1734),
				IsVisible: ptrBool(true),
			},
			{
				Index:     1,
				Country:   "France",
				Latitude:  ptrFloat64(48.8566),
				Longitude: ptrFloat64(2.3522),
				IsVisible: ptrBool(true),
			},
		}

		req := dto.DetectLocationBatchRequest{
			Locations: locations,
		}

		// Mock GetByPointBatch
		boundariesByPoint := map[int][]*domain.AdminBoundary{
			0: {
				{ID: 1, AdminLevel: 2, Name: "Spain"},
				{ID: 100, AdminLevel: 8, Name: "Barcelona"},
			},
			1: {
				{ID: 2, AdminLevel: 2, Name: "France"},
				{ID: 200, AdminLevel: 8, Name: "Paris"},
			},
		}

		mockBoundary.On("GetByPointBatch", ctx, mock.MatchedBy(func(points []domain.LatLon) bool {
			return len(points) == 2
		})).Return(boundariesByPoint, nil)

		resp, err := uc.DetectLocationBatch(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Results, 2)
		assert.Equal(t, 2, resp.Meta.SuccessCount)
		assert.Equal(t, 2, resp.Meta.VisibleCount)
		assert.Equal(t, 0, resp.Meta.NameResolveCount)
		assert.Equal(t, 1, resp.Meta.DBQueriesCount) // Only 1 batch query

		mockBoundary.AssertExpectations(t)
	})

	t.Run("visible location with no boundaries found", func(t *testing.T) {
		mockBoundary2 := &MockBoundaryRepository{}
		mockCache2 := &MockCacheRepository{}
		uc2 := usecase.NewSearchUseCase(mockBoundary2, mockCache2, logger, 1*time.Hour)

		locations := []dto.LocationInput{
			{
				Index:     0,
				Country:   "Spain",
				Latitude:  ptrFloat64(0.0), // Ocean coordinates
				Longitude: ptrFloat64(0.0),
				IsVisible: ptrBool(true),
			},
		}

		req := dto.DetectLocationBatchRequest{
			Locations: locations,
		}

		// Mock GetByPointBatch returning empty
		boundariesByPoint := map[int][]*domain.AdminBoundary{
			0: {}, // No boundaries found
		}

		mockBoundary2.On("GetByPointBatch", ctx, mock.Anything).Return(boundariesByPoint, nil)

		resp, err := uc2.DetectLocationBatch(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 0, resp.Meta.SuccessCount)
		assert.Equal(t, 1, resp.Meta.ErrorCount)
		assert.NotEmpty(t, resp.Results[0].Error)

		mockBoundary2.AssertExpectations(t)
	})
}

func TestSearchUseCase_DetectLocationBatch_NameBasedLocations(t *testing.T) {
	// Test for name-based locations
	logger := zap.NewNop()
	mockBoundary := &MockBoundaryRepository{}
	mockCache := &MockCacheRepository{}
	ctx := context.Background()

	uc := usecase.NewSearchUseCase(mockBoundary, mockCache, logger, 1*time.Hour)

	t.Run("success with only name-based locations", func(t *testing.T) {
		locations := []dto.LocationInput{
			{
				Index:   0,
				Country: "Spain",
				City:    ptrString("Barcelona"),
			},
			{
				Index:   1,
				Country: "France",
				City:    ptrString("Paris"),
				Region:  ptrString("Île-de-France"),
			},
		}

		req := dto.DetectLocationBatchRequest{
			Locations: locations,
		}

		// Mock SearchByTextBatch
		searchResults := []domain.BoundarySearchResult{
			// Location 0
			{Index: 2, Found: true, Boundary: &domain.AdminBoundary{ID: 1, AdminLevel: 2, Name: "Spain"}},
			{Index: 8, Found: true, Boundary: &domain.AdminBoundary{ID: 100, AdminLevel: 8, Name: "Barcelona"}},
			// Location 1
			{Index: 102, Found: true, Boundary: &domain.AdminBoundary{ID: 2, AdminLevel: 2, Name: "France"}},
			{Index: 104, Found: true, Boundary: &domain.AdminBoundary{ID: 20, AdminLevel: 4, Name: "Île-de-France"}},
			{Index: 108, Found: true, Boundary: &domain.AdminBoundary{ID: 200, AdminLevel: 8, Name: "Paris"}},
		}

		mockBoundary.On("SearchByTextBatch", ctx, mock.MatchedBy(func(requests []domain.BoundarySearchRequest) bool {
			return len(requests) >= 2 // At least 2 country searches
		})).Return(searchResults, nil)

		resp, err := uc.DetectLocationBatch(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Results, 2)
		assert.Equal(t, 2, resp.Meta.SuccessCount)
		assert.Equal(t, 0, resp.Meta.VisibleCount)
		assert.Equal(t, 2, resp.Meta.NameResolveCount)

		// Check location 0
		assert.Equal(t, "Spain", resp.Results[0].EnrichedLocation.Country.Name)
		assert.Equal(t, "Barcelona", resp.Results[0].EnrichedLocation.City.Name)

		// Check location 1
		assert.Equal(t, "France", resp.Results[1].EnrichedLocation.Country.Name)
		assert.Equal(t, "Île-de-France", resp.Results[1].EnrichedLocation.Region.Name)
		assert.Equal(t, "Paris", resp.Results[1].EnrichedLocation.City.Name)

		mockBoundary.AssertExpectations(t)
	})

	t.Run("name-based location with no results", func(t *testing.T) {
		mockBoundary2 := &MockBoundaryRepository{}
		mockCache2 := &MockCacheRepository{}
		uc2 := usecase.NewSearchUseCase(mockBoundary2, mockCache2, logger, 1*time.Hour)

		locations := []dto.LocationInput{
			{
				Index:   0,
				Country: "NonExistentCountry",
			},
		}

		req := dto.DetectLocationBatchRequest{
			Locations: locations,
		}

		// Mock SearchByTextBatch returning no results
		searchResults := []domain.BoundarySearchResult{
			{Index: 2, Found: false, Boundary: nil},
		}

		mockBoundary2.On("SearchByTextBatch", ctx, mock.Anything).Return(searchResults, nil)

		resp, err := uc2.DetectLocationBatch(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 0, resp.Meta.SuccessCount)
		assert.Equal(t, 1, resp.Meta.ErrorCount)
		assert.NotEmpty(t, resp.Results[0].Error)

		mockBoundary2.AssertExpectations(t)
	})
}

func TestSearchUseCase_DetectLocationBatch_Errors(t *testing.T) {
	logger := zap.NewNop()
	mockBoundary := &MockBoundaryRepository{}
	mockCache := &MockCacheRepository{}
	ctx := context.Background()

	uc := usecase.NewSearchUseCase(mockBoundary, mockCache, logger, 1*time.Hour)

	t.Run("GetByPointBatch error", func(t *testing.T) {
		locations := []dto.LocationInput{
			{
				Index:     0,
				Country:   "Spain",
				Latitude:  ptrFloat64(41.3851),
				Longitude: ptrFloat64(2.1734),
				IsVisible: ptrBool(true),
			},
		}

		req := dto.DetectLocationBatchRequest{
			Locations: locations,
		}

		mockBoundary.On("GetByPointBatch", ctx, mock.Anything).
			Return(nil, errors.New("database error"))

		resp, err := uc.DetectLocationBatch(ctx, req)

		assert.NoError(t, err) // Should not fail completely
		assert.NotNil(t, resp)
		assert.Equal(t, 1, resp.Meta.ErrorCount)
		assert.NotEmpty(t, resp.Results[0].Error)

		mockBoundary.AssertExpectations(t)
	})

	t.Run("SearchByTextBatch error", func(t *testing.T) {
		mockBoundary2 := &MockBoundaryRepository{}
		mockCache2 := &MockCacheRepository{}
		uc2 := usecase.NewSearchUseCase(mockBoundary2, mockCache2, logger, 1*time.Hour)

		locations := []dto.LocationInput{
			{
				Index:   0,
				Country: "Spain",
			},
		}

		req := dto.DetectLocationBatchRequest{
			Locations: locations,
		}

		mockBoundary2.On("SearchByTextBatch", ctx, mock.Anything).
			Return(nil, errors.New("database error"))

		resp, err := uc2.DetectLocationBatch(ctx, req)

		assert.NoError(t, err) // Should not fail completely
		assert.NotNil(t, resp)
		assert.Equal(t, 1, resp.Meta.ErrorCount)
		assert.NotEmpty(t, resp.Results[0].Error)

		mockBoundary2.AssertExpectations(t)
	})
}
