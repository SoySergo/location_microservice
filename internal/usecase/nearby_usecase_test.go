package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/usecase"
)

// ---- Mock POI Repository (for NearbyUseCase tests) ----

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

// ---- Tests ----

func TestNearbyUseCase_GetNearbyTransport_Success(t *testing.T) {
	logger := zap.NewNop()
	mockTransport := &MockTransportRepository{}
	mockPOI := &mockPOIRepository{}
	ctx := context.Background()

	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	poiUC := usecase.NewPOIUseCase(mockPOI, logger)
	uc := usecase.NewNearbyUseCase(transportUC, poiUC, logger)

	// Mock transport priority search
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

	result, err := uc.GetNearbyTransport(ctx, 41.3851, 2.1734, 1500, 10)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Len(t, result.Stations, 1)
	assert.Equal(t, "Passeig de Gràcia", result.Stations[0].Name)
	mockTransport.AssertExpectations(t)
}

func TestNearbyUseCase_GetNearbyTransport_InvalidCoordinates(t *testing.T) {
	logger := zap.NewNop()
	mockTransport := &MockTransportRepository{}
	mockPOI := &mockPOIRepository{}

	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	poiUC := usecase.NewPOIUseCase(mockPOI, logger)
	uc := usecase.NewNearbyUseCase(transportUC, poiUC, logger)

	result, err := uc.GetNearbyTransport(context.Background(), 999, 999, 1500, 10)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestNearbyUseCase_GetNearbyTransport_DefaultValues(t *testing.T) {
	logger := zap.NewNop()
	mockTransport := &MockTransportRepository{}
	mockPOI := &mockPOIRepository{}
	ctx := context.Background()

	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	poiUC := usecase.NewPOIUseCase(mockPOI, logger)
	uc := usecase.NewNearbyUseCase(transportUC, poiUC, logger)

	mockTransport.On("GetNearestTransportByPriority", mock.Anything,
		mock.Anything, mock.Anything, mock.Anything, mock.Anything,
	).Return([]domain.NearestTransportWithLines{}, nil)

	// radius=0 and limit=0 → defaults
	result, err := uc.GetNearbyTransport(ctx, 41.3851, 2.1734, 0, 0)

	assert.NoError(t, err)
	assert.NotNil(t, result)
}

func TestNearbyUseCase_GetNearbyPOI_Success(t *testing.T) {
	logger := zap.NewNop()
	mockTransport := &MockTransportRepository{}
	mockPOI := &mockPOIRepository{}
	ctx := context.Background()

	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	poiUC := usecase.NewPOIUseCase(mockPOI, logger)
	uc := usecase.NewNearbyUseCase(transportUC, poiUC, logger)

	// Mock POI GetNearby — medical category maps to [pharmacy, hospital, clinic, doctors, dentist, veterinary]
	mockPOI.On("GetNearby", mock.Anything,
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("float64"),
		mock.AnythingOfType("float64"),
		mock.MatchedBy(func(cats []string) bool {
			return len(cats) > 0 && cats[0] == "pharmacy"
		}),
	).Return([]*domain.POI{
		{
			ID:          1,
			OSMId:       1,
			Name:        "Farmacia Central",
			Category:    "pharmacy",
			Subcategory: "general",
			Lat:         41.386,
			Lon:         2.175,
		},
		{
			ID:          2,
			OSMId:       2,
			Name:        "Hospital del Mar",
			Category:    "hospital",
			Subcategory: "general",
			Lat:         41.384,
			Lon:         2.200,
		},
	}, nil)

	result, err := uc.GetNearbyPOI(ctx, "medical", 41.3851, 2.1734, 1.0, 20)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "medical", result.Category)
	assert.Len(t, result.Items, 2)
	assert.Equal(t, "Farmacia Central", result.Items[0].Name)
}

func TestNearbyUseCase_GetNearbyPOI_InvalidCategory(t *testing.T) {
	logger := zap.NewNop()
	mockTransport := &MockTransportRepository{}
	mockPOI := &mockPOIRepository{}

	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	poiUC := usecase.NewPOIUseCase(mockPOI, logger)
	uc := usecase.NewNearbyUseCase(transportUC, poiUC, logger)

	result, err := uc.GetNearbyPOI(context.Background(), "invalid_category", 41.3851, 2.1734, 1.0, 20)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestNearbyUseCase_GetNearbyPOI_InvalidCoordinates(t *testing.T) {
	logger := zap.NewNop()
	mockTransport := &MockTransportRepository{}
	mockPOI := &mockPOIRepository{}

	transportUC := usecase.NewTransportUseCase(mockTransport, logger)
	poiUC := usecase.NewPOIUseCase(mockPOI, logger)
	uc := usecase.NewNearbyUseCase(transportUC, poiUC, logger)

	result, err := uc.GetNearbyPOI(context.Background(), "medical", 999, 999, 1.0, 20)

	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestIsValidNearbyCategory(t *testing.T) {
	assert.True(t, domain.IsValidNearbyCategory("transport"))
	assert.True(t, domain.IsValidNearbyCategory("schools"))
	assert.True(t, domain.IsValidNearbyCategory("medical"))
	assert.True(t, domain.IsValidNearbyCategory("groceries"))
	assert.True(t, domain.IsValidNearbyCategory("shopping"))
	assert.True(t, domain.IsValidNearbyCategory("restaurants"))
	assert.True(t, domain.IsValidNearbyCategory("sports"))
	assert.True(t, domain.IsValidNearbyCategory("entertainment"))
	assert.True(t, domain.IsValidNearbyCategory("parks"))
	assert.True(t, domain.IsValidNearbyCategory("beauty"))
	assert.True(t, domain.IsValidNearbyCategory("attractions"))

	assert.False(t, domain.IsValidNearbyCategory("invalid"))
	assert.False(t, domain.IsValidNearbyCategory(""))
}

func TestGetOSMCategories(t *testing.T) {
	// medical → pharmacy, hospital, clinic, doctors, dentist, veterinary
	medical := domain.GetOSMCategories("medical")
	assert.Contains(t, medical, "pharmacy")
	assert.Contains(t, medical, "hospital")
	assert.Contains(t, medical, "clinic")
	assert.Len(t, medical, 6)

	// schools → school, kindergarten, college, university, library, language_school
	schools := domain.GetOSMCategories("schools")
	assert.Contains(t, schools, "school")
	assert.Contains(t, schools, "university")
	assert.Len(t, schools, 6)

	// transport → nil (special case)
	assert.Nil(t, domain.GetOSMCategories("transport"))

	// invalid → nil
	assert.Nil(t, domain.GetOSMCategories("invalid"))
}
