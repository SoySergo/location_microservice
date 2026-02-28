package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/location-microservice/internal/domain"
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

func (m *MockBoundaryRepository) ReverseGeocodeBatch(ctx context.Context, points []domain.LatLon) ([]*domain.Address, error) {
	args := m.Called(ctx, points)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.Address), args.Error(1)
}

func (m *MockBoundaryRepository) SearchByTextBatch(ctx context.Context, requests []domain.BoundarySearchRequest) ([]domain.BoundarySearchResult, error) {
	args := m.Called(ctx, requests)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.BoundarySearchResult), args.Error(1)
}

func (m *MockBoundaryRepository) GetByPointBatch(ctx context.Context, points []domain.LatLon) (map[int][]*domain.AdminBoundary, error) {
	args := m.Called(ctx, points)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[int][]*domain.AdminBoundary), args.Error(1)
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

func (m *MockTransportRepository) GetTransportTileByTypes(ctx context.Context, z, x, y int, types []string) ([]byte, error) {
	args := m.Called(ctx, z, x, y, types)
	return args.Get(0).([]byte), args.Error(1)
}

func (m *MockTransportRepository) GetLinesByStationID(ctx context.Context, stationID int64) ([]*domain.TransportLine, error) {
	args := m.Called(ctx, stationID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TransportLine), args.Error(1)
}

func (m *MockTransportRepository) GetNearestStationsGrouped(ctx context.Context, lat, lon float64, priorities []domain.TransportPriority, maxDistance float64) ([]*domain.TransportStation, error) {
	args := m.Called(ctx, lat, lon, priorities, maxDistance)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*domain.TransportStation), args.Error(1)
}

func (m *MockTransportRepository) GetNearestStationsWithLinesBatch(ctx context.Context, req domain.BatchTransportRequest) ([]domain.TransportStationWithLines, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.TransportStationWithLines), args.Error(1)
}

func (m *MockTransportRepository) GetNearestStationsBatch(ctx context.Context, req domain.BatchTransportRequest) ([]domain.TransportStationWithLines, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.TransportStationWithLines), args.Error(1)
}

func (m *MockTransportRepository) GetLinesByStationIDsBatch(ctx context.Context, stationIDs []int64) (map[int64][]domain.TransportLineInfo, error) {
	args := m.Called(ctx, stationIDs)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(map[int64][]domain.TransportLineInfo), args.Error(1)
}

func (m *MockTransportRepository) GetNearestTransportByPriority(ctx context.Context, lat, lon float64, radiusM float64, limit int) ([]domain.NearestTransportWithLines, error) {
	args := m.Called(ctx, lat, lon, radiusM, limit)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.NearestTransportWithLines), args.Error(1)
}

func (m *MockTransportRepository) GetNearestTransportByPriorityBatch(ctx context.Context, points []domain.TransportSearchPoint, radiusM float64, limitPerPoint int) ([]domain.BatchTransportResult, error) {
	args := m.Called(ctx, points, radiusM, limitPerPoint)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.BatchTransportResult), args.Error(1)
}

func (m *MockTransportRepository) GetStationsInBBox(ctx context.Context, swLat, swLon, neLat, neLon float64, types []string, limit, offset int) ([]domain.TransportStationWithLines, int, error) {
	args := m.Called(ctx, swLat, swLon, neLat, neLon, types, limit, offset)
	if args.Get(0) == nil {
		return nil, args.Int(1), args.Error(2)
	}
	return args.Get(0).([]domain.TransportStationWithLines), args.Int(1), args.Error(2)
}

// NOTE: The old EnrichmentUseCase tests have been removed as the usecase has been refactored.
// The new enrichment logic is now in EnrichedLocationUseCase which is tested in enriched_location_usecase_test.go
// The old EnrichmentUseCase is kept for backward compatibility but is no longer the primary interface.

// Test basic mock repository methods work
func TestMockRepositories(t *testing.T) {
	mockBoundary := &MockBoundaryRepository{}
	mockTransport := &MockTransportRepository{}

	assert.NotNil(t, mockBoundary)
	assert.NotNil(t, mockTransport)
}

// Helper function
func ptrInt64(v int64) *int64 {
	return &v
}
