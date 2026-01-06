package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/usecase"
)

// MockBoundaryRepository is a mock of BoundaryRepository
type MockBoundaryRepository struct {
	mock.Mock
}

func (m *MockBoundaryRepository) GetByID(ctx context.Context, id int64) (*domain.AdminBoundary, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.AdminBoundary), args.Error(1)
}

func (m *MockBoundaryRepository) SearchByText(ctx context.Context, query string, lang string, adminLevels []int, limit int) ([]*domain.AdminBoundary, error) {
	args := m.Called(ctx, query, lang, adminLevels, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AdminBoundary), args.Error(1)
}

func (m *MockBoundaryRepository) ReverseGeocode(ctx context.Context, lat, lon float64) (*domain.Address, error) {
	args := m.Called(ctx, lat, lon)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Address), args.Error(1)
}

func (m *MockBoundaryRepository) GetTile(ctx context.Context, z, x, y int) ([]byte, error) {
	args := m.Called(ctx, z, x, y)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockBoundaryRepository) GetByPoint(ctx context.Context, lat, lon float64) ([]*domain.AdminBoundary, error) {
	args := m.Called(ctx, lat, lon)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AdminBoundary), args.Error(1)
}

func (m *MockBoundaryRepository) Search(ctx context.Context, query string, limit int) ([]*domain.AdminBoundary, error) {
	args := m.Called(ctx, query, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AdminBoundary), args.Error(1)
}

func (m *MockBoundaryRepository) GetChildren(ctx context.Context, parentID int64) ([]*domain.AdminBoundary, error) {
	args := m.Called(ctx, parentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AdminBoundary), args.Error(1)
}

func (m *MockBoundaryRepository) GetByAdminLevel(ctx context.Context, level int, limit int) ([]*domain.AdminBoundary, error) {
	args := m.Called(ctx, level, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AdminBoundary), args.Error(1)
}

func (m *MockBoundaryRepository) GetBoundariesInRadius(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.AdminBoundary, error) {
	args := m.Called(ctx, lat, lon, radiusKm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.AdminBoundary), args.Error(1)
}

func (m *MockBoundaryRepository) GetBoundariesRadiusTile(ctx context.Context, lat, lon, radiusKm float64) ([]byte, error) {
	args := m.Called(ctx, lat, lon, radiusKm)
	return args.Get(0).([]byte), args.Error(1)
}

// MockTransportRepository is a mock of TransportRepository
type MockTransportRepository struct {
	mock.Mock
}

func (m *MockTransportRepository) GetNearestStations(ctx context.Context, lat, lon float64, types []string, maxDistance float64, limit int) ([]*domain.TransportStation, error) {
	args := m.Called(ctx, lat, lon, types, maxDistance, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TransportStation), args.Error(1)
}

func (m *MockTransportRepository) GetLineByID(ctx context.Context, id int64) (*domain.TransportLine, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.TransportLine), args.Error(1)
}

func (m *MockTransportRepository) GetLinesByIDs(ctx context.Context, ids []int64) ([]*domain.TransportLine, error) {
	args := m.Called(ctx, ids)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TransportLine), args.Error(1)
}

func (m *MockTransportRepository) GetStationsByLineID(ctx context.Context, lineID int64) ([]*domain.TransportStation, error) {
	args := m.Called(ctx, lineID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TransportStation), args.Error(1)
}

func (m *MockTransportRepository) GetTransportTile(ctx context.Context, z, x, y int) ([]byte, error) {
	args := m.Called(ctx, z, x, y)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTransportRepository) GetLineTile(ctx context.Context, lineID int64) ([]byte, error) {
	args := m.Called(ctx, lineID)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTransportRepository) GetLinesTile(ctx context.Context, lineIDs []int64) ([]byte, error) {
	args := m.Called(ctx, lineIDs)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTransportRepository) GetStationsInRadius(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.TransportStation, error) {
	args := m.Called(ctx, lat, lon, radiusKm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TransportStation), args.Error(1)
}

func (m *MockTransportRepository) GetLinesInRadius(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.TransportLine, error) {
	args := m.Called(ctx, lat, lon, radiusKm)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TransportLine), args.Error(1)
}

func (m *MockTransportRepository) GetTransportRadiusTile(ctx context.Context, lat, lon, radiusKm float64) ([]byte, error) {
	args := m.Called(ctx, lat, lon, radiusKm)
	return args.Get(0).([]byte), args.Error(1)
}

// Test EnrichLocation with city name
func TestEnrichmentUseCase_EnrichLocation_WithCityName(t *testing.T) {
	// Arrange
	mockBoundary := &MockBoundaryRepository{}
	mockTransport := &MockTransportRepository{}
	logger := zap.NewNop()

	uc := usecase.NewEnrichmentUseCase(
		mockBoundary,
		mockTransport,
		logger,
		[]string{"metro"},
		1000.0,
	)

	propertyID := uuid.New()
	city := "Barcelona"
	lat := 41.3851
	lon := 2.1734

	event := &domain.LocationEnrichEvent{
		PropertyID: propertyID,
		Country:    "Spain",
		City:       &city,
		Latitude:   &lat,
		Longitude:  &lon,
	}

	// Mock city lookup - returns Barcelona with ID 100
	barcelonaCity := &domain.AdminBoundary{
		ID:         100,
		Name:       "Barcelona",
		AdminLevel: 8,
		ParentID:   ptrInt64(50), // Province
	}
	mockBoundary.On("SearchByText", mock.Anything, "Barcelona", "", []int{8}, 1).
		Return([]*domain.AdminBoundary{barcelonaCity}, nil)

	// Mock parent hierarchy - Province
	barcelonaProvince := &domain.AdminBoundary{
		ID:         50,
		Name:       "Barcelona Province",
		AdminLevel: 6,
		ParentID:   ptrInt64(20), // Region
	}
	mockBoundary.On("GetByID", mock.Anything, int64(50)).
		Return(barcelonaProvince, nil)

	// Mock parent hierarchy - Region
	catalonia := &domain.AdminBoundary{
		ID:         20,
		Name:       "Catalonia",
		AdminLevel: 4,
		ParentID:   ptrInt64(1), // Country
	}
	mockBoundary.On("GetByID", mock.Anything, int64(20)).
		Return(catalonia, nil)

	// Mock parent hierarchy - Country
	spain := &domain.AdminBoundary{
		ID:         1,
		Name:       "Spain",
		AdminLevel: 2,
		ParentID:   nil,
	}
	mockBoundary.On("GetByID", mock.Anything, int64(1)).
		Return(spain, nil)

	// Mock city boundary (first call with city ID)
	mockBoundary.On("GetByID", mock.Anything, int64(100)).
		Return(barcelonaCity, nil)

	// Mock transport stations
	stations := []*domain.TransportStation{
		{
			ID:      500,
			Name:    "Passeig de Gràcia",
			Type:    "metro",
			Lat:     41.3950,
			Lon:     2.1640,
			LineIDs: []int64{3, 4},
		},
	}
	mockTransport.On("GetNearestStations", mock.Anything, lat, lon, []string{"metro"}, 1000.0, 10).
		Return(stations, nil)

	// Act
	result, err := uc.EnrichLocation(context.Background(), event)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, propertyID, result.PropertyID)
	assert.NotNil(t, result.EnrichedLocation)
	assert.Equal(t, int64(1), *result.EnrichedLocation.CountryID)
	assert.Equal(t, int64(20), *result.EnrichedLocation.RegionID)
	assert.Equal(t, int64(50), *result.EnrichedLocation.ProvinceID)
	assert.Equal(t, int64(100), *result.EnrichedLocation.CityID)
	assert.True(t, *result.EnrichedLocation.IsAddressVisible)
	assert.Len(t, result.NearestTransport, 1)
	assert.Equal(t, int64(500), result.NearestTransport[0].StationID)
	assert.Equal(t, "Passeig de Gràcia", result.NearestTransport[0].Name)

	mockBoundary.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

// Test EnrichLocation with coordinates only (reverse geocoding)
func TestEnrichmentUseCase_EnrichLocation_WithCoordinatesOnly(t *testing.T) {
	// Arrange
	mockBoundary := &MockBoundaryRepository{}
	mockTransport := &MockTransportRepository{}
	logger := zap.NewNop()

	uc := usecase.NewEnrichmentUseCase(
		mockBoundary,
		mockTransport,
		logger,
		[]string{"metro"},
		1000.0,
	)

	propertyID := uuid.New()
	lat := 41.3851
	lon := 2.1734

	event := &domain.LocationEnrichEvent{
		PropertyID: propertyID,
		Country:    "Spain",
		Latitude:   &lat,
		Longitude:  &lon,
	}

	// Mock reverse geocoding - returns boundaries for all levels
	boundaries := []*domain.AdminBoundary{
		{ID: 1, AdminLevel: 2, Name: "Spain"},               // Country
		{ID: 20, AdminLevel: 4, Name: "Catalonia"},          // Region
		{ID: 50, AdminLevel: 6, Name: "Barcelona Province"}, // Province
		{ID: 100, AdminLevel: 8, Name: "Barcelona"},         // City
	}
	mockBoundary.On("GetByPoint", mock.Anything, lat, lon).
		Return(boundaries, nil)

	// Mock transport stations
	stations := []*domain.TransportStation{
		{
			ID:      500,
			Name:    "Passeig de Gràcia",
			Type:    "metro",
			Lat:     41.3950,
			Lon:     2.1640,
			LineIDs: []int64{3, 4},
		},
	}
	mockTransport.On("GetNearestStations", mock.Anything, lat, lon, []string{"metro"}, 1000.0, 10).
		Return(stations, nil)

	// Act
	result, err := uc.EnrichLocation(context.Background(), event)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, propertyID, result.PropertyID)
	assert.NotNil(t, result.EnrichedLocation)
	assert.Equal(t, int64(1), *result.EnrichedLocation.CountryID)
	assert.Equal(t, int64(20), *result.EnrichedLocation.RegionID)
	assert.Equal(t, int64(50), *result.EnrichedLocation.ProvinceID)
	assert.Equal(t, int64(100), *result.EnrichedLocation.CityID)
	assert.True(t, *result.EnrichedLocation.IsAddressVisible)

	mockBoundary.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

// Test EnrichLocation with location not found
func TestEnrichmentUseCase_EnrichLocation_LocationNotFound(t *testing.T) {
	// Arrange
	mockBoundary := &MockBoundaryRepository{}
	mockTransport := &MockTransportRepository{}
	logger := zap.NewNop()

	uc := usecase.NewEnrichmentUseCase(
		mockBoundary,
		mockTransport,
		logger,
		[]string{"metro"},
		1000.0,
	)

	propertyID := uuid.New()
	city := "NonExistentCity"

	event := &domain.LocationEnrichEvent{
		PropertyID: propertyID,
		Country:    "Spain",
		City:       &city,
	}

	// Mock city lookup - returns empty
	mockBoundary.On("SearchByText", mock.Anything, "NonExistentCity", "", []int{8}, 1).
		Return([]*domain.AdminBoundary{}, nil)

	// Act
	result, err := uc.EnrichLocation(context.Background(), event)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, propertyID, result.PropertyID)
	assert.NotEmpty(t, result.Error)

	mockBoundary.AssertExpectations(t)
}

// Test EnrichLocation without coordinates (no transport lookup)
func TestEnrichmentUseCase_EnrichLocation_WithoutCoordinates(t *testing.T) {
	// Arrange
	mockBoundary := &MockBoundaryRepository{}
	mockTransport := &MockTransportRepository{}
	logger := zap.NewNop()

	uc := usecase.NewEnrichmentUseCase(
		mockBoundary,
		mockTransport,
		logger,
		[]string{"metro"},
		1000.0,
	)

	propertyID := uuid.New()
	city := "Barcelona"

	event := &domain.LocationEnrichEvent{
		PropertyID: propertyID,
		Country:    "Spain",
		City:       &city,
	}

	// Mock city lookup
	barcelonaCity := &domain.AdminBoundary{
		ID:         100,
		Name:       "Barcelona",
		AdminLevel: 8,
		ParentID:   ptrInt64(1), // Has a parent (country)
	}
	mockBoundary.On("SearchByText", mock.Anything, "Barcelona", "", []int{8}, 1).
		Return([]*domain.AdminBoundary{barcelonaCity}, nil)

	// Mock getting the city boundary
	mockBoundary.On("GetByID", mock.Anything, int64(100)).
		Return(barcelonaCity, nil)

	// Mock parent (country)
	spain := &domain.AdminBoundary{
		ID:         1,
		Name:       "Spain",
		AdminLevel: 2,
		ParentID:   nil,
	}
	mockBoundary.On("GetByID", mock.Anything, int64(1)).
		Return(spain, nil)

	// No transport lookup should be called (no coordinates)

	// Act
	result, err := uc.EnrichLocation(context.Background(), event)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, propertyID, result.PropertyID)
	assert.NotNil(t, result.EnrichedLocation)
	assert.Equal(t, int64(1), *result.EnrichedLocation.CountryID) // Has country from parent
	assert.Equal(t, int64(100), *result.EnrichedLocation.CityID)
	assert.False(t, *result.EnrichedLocation.IsAddressVisible) // No coordinates or street address
	assert.Empty(t, result.NearestTransport)

	mockBoundary.AssertExpectations(t)
	mockTransport.AssertNotCalled(t, "GetNearestStations")
}

// Test transport lookup failure doesn't fail enrichment
func TestEnrichmentUseCase_EnrichLocation_TransportLookupFails(t *testing.T) {
	// Arrange
	mockBoundary := &MockBoundaryRepository{}
	mockTransport := &MockTransportRepository{}
	logger := zap.NewNop()

	uc := usecase.NewEnrichmentUseCase(
		mockBoundary,
		mockTransport,
		logger,
		[]string{"metro"},
		1000.0,
	)

	propertyID := uuid.New()
	city := "Barcelona"
	lat := 41.3851
	lon := 2.1734

	event := &domain.LocationEnrichEvent{
		PropertyID: propertyID,
		Country:    "Spain",
		City:       &city,
		Latitude:   &lat,
		Longitude:  &lon,
	}

	// Mock city lookup
	barcelonaCity := &domain.AdminBoundary{
		ID:         100,
		Name:       "Barcelona",
		AdminLevel: 8,
		ParentID:   ptrInt64(1),
	}
	mockBoundary.On("SearchByText", mock.Anything, "Barcelona", "", []int{8}, 1).
		Return([]*domain.AdminBoundary{barcelonaCity}, nil)
	mockBoundary.On("GetByID", mock.Anything, int64(100)).
		Return(barcelonaCity, nil)

	// Mock parent (country)
	spain := &domain.AdminBoundary{
		ID:         1,
		Name:       "Spain",
		AdminLevel: 2,
		ParentID:   nil,
	}
	mockBoundary.On("GetByID", mock.Anything, int64(1)).
		Return(spain, nil)

	// Mock transport lookup failure
	mockTransport.On("GetNearestStations", mock.Anything, lat, lon, []string{"metro"}, 1000.0, 10).
		Return(nil, errors.New("transport service unavailable"))

	// Act
	result, err := uc.EnrichLocation(context.Background(), event)

	// Assert
	assert.NoError(t, err) // Enrichment should succeed
	assert.NotNil(t, result)
	assert.Equal(t, propertyID, result.PropertyID)
	assert.NotNil(t, result.EnrichedLocation)
	assert.Empty(t, result.NearestTransport) // No transport data

	mockBoundary.AssertExpectations(t)
	mockTransport.AssertExpectations(t)
}

// Helper function
func ptrInt64(v int64) *int64 {
	return &v
}
