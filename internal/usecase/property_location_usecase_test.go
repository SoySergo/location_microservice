package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
)

// ---- Mock POI Repository ----

type mockPOIRepository struct {
	mock.Mock
}

func (m *mockPOIRepository) GetByID(ctx context.Context, id int64) (*domain.POI, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.POI), args.Error(1)
}

func (m *mockPOIRepository) GetNearby(ctx context.Context, lat, lon float64, radiusKm float64, categories []string) ([]*domain.POI, error) {
	args := m.Called(ctx, lat, lon, radiusKm, categories)
	return args.Get(0).([]*domain.POI), args.Error(1)
}

func (m *mockPOIRepository) Search(ctx context.Context, query string, categories []string, limit int) ([]*domain.POI, error) {
	args := m.Called(ctx, query, categories, limit)
	return args.Get(0).([]*domain.POI), args.Error(1)
}

func (m *mockPOIRepository) GetByCategory(ctx context.Context, category string, limit int) ([]*domain.POI, error) {
	args := m.Called(ctx, category, limit)
	return args.Get(0).([]*domain.POI), args.Error(1)
}

func (m *mockPOIRepository) GetCategories(ctx context.Context) ([]*domain.POICategory, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.POICategory), args.Error(1)
}

func (m *mockPOIRepository) GetSubcategories(ctx context.Context, categoryID int64) ([]*domain.POISubcategory, error) {
	args := m.Called(ctx, categoryID)
	return args.Get(0).([]*domain.POISubcategory), args.Error(1)
}

func (m *mockPOIRepository) GetPOITile(ctx context.Context, z, x, y int, categories []string) ([]byte, error) {
	args := m.Called(ctx, z, x, y, categories)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockPOIRepository) GetPOIRadiusTile(ctx context.Context, lat, lon, radiusKm float64, categories []string) ([]byte, error) {
	args := m.Called(ctx, lat, lon, radiusKm, categories)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockPOIRepository) GetPOIByBoundaryTile(ctx context.Context, boundaryID int64, categories []string) ([]byte, error) {
	args := m.Called(ctx, boundaryID, categories)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockPOIRepository) GetPOITileByCategories(ctx context.Context, z, x, y int, categories, subcategories []string) ([]byte, error) {
	args := m.Called(ctx, z, x, y, categories, subcategories)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockPOIRepository) GetPOIInBBox(ctx context.Context, swLat, swLon, neLat, neLon float64, categories, subcategories []string, limit, offset int) ([]*domain.POI, int, error) {
	args := m.Called(ctx, swLat, swLon, neLat, neLon, categories, subcategories, limit, offset)
	return args.Get(0).([]*domain.POI), args.Int(1), args.Error(2)
}

func (m *mockPOIRepository) CountByCategories(ctx context.Context, lat, lon float64, radiusMeters int) (map[string]int, error) {
	args := m.Called(ctx, lat, lon, radiusMeters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[string]int), args.Error(1)
}

// ---- Mock Environment Repository ----

type mockEnvironmentRepository struct {
	mock.Mock
}

func (m *mockEnvironmentRepository) GetGreenSpacesNearby(ctx context.Context, lat, lon float64, radiusKm float64) ([]*domain.GreenSpace, error) {
	args := m.Called(ctx, lat, lon, radiusKm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.GreenSpace), args.Error(1)
}

func (m *mockEnvironmentRepository) GetWaterBodiesNearby(ctx context.Context, lat, lon float64, radiusKm float64) ([]*domain.WaterBody, error) {
	args := m.Called(ctx, lat, lon, radiusKm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.WaterBody), args.Error(1)
}

func (m *mockEnvironmentRepository) GetBeachesNearby(ctx context.Context, lat, lon float64, radiusKm float64) ([]*domain.Beach, error) {
	args := m.Called(ctx, lat, lon, radiusKm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Beach), args.Error(1)
}

func (m *mockEnvironmentRepository) GetNoiseSourcesNearby(ctx context.Context, lat, lon float64, radiusKm float64) ([]*domain.NoiseSource, error) {
	args := m.Called(ctx, lat, lon, radiusKm)
	return args.Get(0).([]*domain.NoiseSource), args.Error(1)
}

func (m *mockEnvironmentRepository) GetTouristZonesNearby(ctx context.Context, lat, lon float64, radiusKm float64) ([]*domain.TouristZone, error) {
	args := m.Called(ctx, lat, lon, radiusKm)
	return args.Get(0).([]*domain.TouristZone), args.Error(1)
}

func (m *mockEnvironmentRepository) GetGreenSpaceByID(ctx context.Context, id int64) (*domain.GreenSpace, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.GreenSpace), args.Error(1)
}

func (m *mockEnvironmentRepository) GetBeachByID(ctx context.Context, id int64) (*domain.Beach, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.Beach), args.Error(1)
}

func (m *mockEnvironmentRepository) GetTouristZoneByID(ctx context.Context, id int64) (*domain.TouristZone, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(*domain.TouristZone), args.Error(1)
}

func (m *mockEnvironmentRepository) GetGreenSpacesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	args := m.Called(ctx, z, x, y)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockEnvironmentRepository) GetWaterTile(ctx context.Context, z, x, y int) ([]byte, error) {
	args := m.Called(ctx, z, x, y)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockEnvironmentRepository) GetBeachesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	args := m.Called(ctx, z, x, y)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockEnvironmentRepository) GetNoiseSourcesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	args := m.Called(ctx, z, x, y)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockEnvironmentRepository) GetTouristZonesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	args := m.Called(ctx, z, x, y)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *mockEnvironmentRepository) GetEnvironmentRadiusTile(ctx context.Context, lat, lon, radiusKm float64) ([]byte, error) {
	args := m.Called(ctx, lat, lon, radiusKm)
	return args.Get(0).([]byte), args.Error(1)
}

// ---- Tests ----

func TestPropertyLocationUseCase_GetPropertyLocationData_Success(t *testing.T) {
	logger := zap.NewNop()
	mockTransport := &MockTransportRepository{}
	mockPOI := &mockPOIRepository{}
	mockEnv := &mockEnvironmentRepository{}
	ctx := context.Background()

	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	uc := usecase.NewPropertyLocationUseCase(transportUC, mockPOI, mockEnv, logger)

	req := dto.PropertyLocationRequest{
		Lat:    41.3851,
		Lon:    2.1734,
		Radius: 1000,
	}

	// Mock transport
	mockTransport.On("GetNearestTransportByPriority", mock.Anything,
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("int"),
	).Return([]domain.NearestTransportWithLines{
		{
			StationID: 1,
			Name:      "Passeig de Gràcia",
			Type:      "metro",
			Lat:       41.39,
			Lon:       2.16,
			Distance:  300.5,
			Lines: []domain.TransportLineInfo{
				{ID: 1, Name: "L3", Type: "metro"},
			},
		},
	}, nil)

	// Mock POI counts
	mockPOI.On("CountByCategories", mock.Anything,
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("int"),
	).Return(map[string]int{
		"healthcare": 5,
		"shopping":   12,
		"education":  3,
		"food_drink": 8,
	}, nil)

	// Mock environment
	mockEnv.On("GetGreenSpacesNearby", mock.Anything,
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("float64"),
	).Return([]*domain.GreenSpace{{ID: 1}}, nil)

	mockEnv.On("GetWaterBodiesNearby", mock.Anything,
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("float64"),
	).Return([]*domain.WaterBody{}, nil)

	mockEnv.On("GetBeachesNearby", mock.Anything,
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("float64"),
	).Return([]*domain.Beach{}, nil)

	// Act
	result, err := uc.GetPropertyLocationData(ctx, req)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.NearestTransport, 1)
	assert.Equal(t, "Passeig de Gràcia", result.NearestTransport[0].Name)
	assert.Equal(t, 5, result.POISummary["healthcare"])
	assert.Equal(t, 12, result.POISummary["shopping"])
	assert.True(t, result.Environment.GreenSpacesNearby)
	assert.False(t, result.Environment.WaterNearby)
	assert.False(t, result.Environment.BeachNearby)
}

func TestPropertyLocationUseCase_GetPropertyLocationData_DefaultRadius(t *testing.T) {
	logger := zap.NewNop()
	mockTransport := &MockTransportRepository{}
	mockPOI := &mockPOIRepository{}
	mockEnv := &mockEnvironmentRepository{}
	ctx := context.Background()

	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	uc := usecase.NewPropertyLocationUseCase(transportUC, mockPOI, mockEnv, logger)

	req := dto.PropertyLocationRequest{
		Lat: 41.3851,
		Lon: 2.1734,
		// Radius == 0 → should use default (1000)
	}

	mockTransport.On("GetNearestTransportByPriority", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]domain.NearestTransportWithLines{}, nil)

	// CountByCategories should be called with default radius (1000 meters)
	mockPOI.On("CountByCategories", mock.Anything, req.Lat, req.Lon, 1000).
		Return(map[string]int{}, nil)

	mockEnv.On("GetGreenSpacesNearby", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*domain.GreenSpace{}, nil)
	mockEnv.On("GetWaterBodiesNearby", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*domain.WaterBody{}, nil)
	mockEnv.On("GetBeachesNearby", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*domain.Beach{}, nil)

	result, err := uc.GetPropertyLocationData(ctx, req)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Empty(t, result.NearestTransport)
	assert.Empty(t, result.POISummary)
	assert.False(t, result.Environment.GreenSpacesNearby)
}

func TestPropertyLocationUseCase_GetPropertyLocationData_InvalidCoordinates(t *testing.T) {
	logger := zap.NewNop()
	mockTransport := &MockTransportRepository{}
	mockPOI := &mockPOIRepository{}
	mockEnv := &mockEnvironmentRepository{}

	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	uc := usecase.NewPropertyLocationUseCase(transportUC, mockPOI, mockEnv, logger)

	req := dto.PropertyLocationRequest{
		Lat:    999, // invalid
		Lon:    999, // invalid
		Radius: 1000,
	}

	result, err := uc.GetPropertyLocationData(context.Background(), req)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestPropertyLocationUseCase_GetPropertyLocationData_PartialFailures(t *testing.T) {
	// If one of the parallel requests fails, others should still return data
	logger := zap.NewNop()
	mockTransport := &MockTransportRepository{}
	mockPOI := &mockPOIRepository{}
	mockEnv := &mockEnvironmentRepository{}
	ctx := context.Background()

	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	uc := usecase.NewPropertyLocationUseCase(transportUC, mockPOI, mockEnv, logger)

	req := dto.PropertyLocationRequest{
		Lat:    41.3851,
		Lon:    2.1734,
		Radius: 1000,
	}

	// Transport fails
	mockTransport.On("GetNearestTransportByPriority", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	// POI counts fail
	mockPOI.On("CountByCategories", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return(nil, assert.AnError)

	// Environment succeeds
	mockEnv.On("GetGreenSpacesNearby", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*domain.GreenSpace{{ID: 1}}, nil)
	mockEnv.On("GetWaterBodiesNearby", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*domain.WaterBody{{ID: 1}}, nil)
	mockEnv.On("GetBeachesNearby", mock.Anything, mock.Anything, mock.Anything, mock.Anything).
		Return([]*domain.Beach{}, nil)

	result, err := uc.GetPropertyLocationData(ctx, req)

	assert.NoError(t, err) // should not fail even with partial errors
	assert.NotNil(t, result)
	assert.Empty(t, result.NearestTransport) // transport failed → empty
	assert.Empty(t, result.POISummary)       // POI failed → empty
	assert.True(t, result.Environment.GreenSpacesNearby)
	assert.True(t, result.Environment.WaterNearby)
	assert.False(t, result.Environment.BeachNearby)
}
