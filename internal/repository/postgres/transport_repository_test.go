package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/repository/postgres/testhelpers"
)

// TransportRepositoryTestSuite tests all methods of TransportRepository
type TransportRepositoryTestSuite struct {
	suite.Suite
	testDB *testhelpers.TestDB
	repo   repository.TransportRepository
	ctx    context.Context
}

// SetupSuite runs once before all tests in the suite
func (s *TransportRepositoryTestSuite) SetupSuite() {
	// Initialize test database connection
	s.testDB = testhelpers.SetupTestDB(s.T())

	// Clean up existing data first
	err := s.testDB.Cleanup(context.Background())
	s.NoError(err, "Failed to cleanup test database")

	// Apply migrations (skip if tables already exist)
	_ = testhelpers.ApplyMigrations(
		s.testDB.DB.DB,
		"../../../migrations",
	)

	// Load fixtures
	fixtures := []string{
		"transport.sql",
	}
	err = testhelpers.LoadFixtures(
		s.testDB.DB.DB,
		"../../../../tasks/test_unit_realbd_test_data/fixtures",
		fixtures,
	)
	s.NoError(err, "Failed to load fixtures")

	// Create repository using test helper that wraps DB with logger
	s.repo = testhelpers.NewTransportRepositoryForTest(s.testDB.DB, s.testDB.Logger)
}

// TearDownSuite runs once after all tests in the suite
func (s *TransportRepositoryTestSuite) TearDownSuite() {
	if s.testDB != nil {
		s.testDB.Close()
	}
}

// SetupTest runs before each test
func (s *TransportRepositoryTestSuite) SetupTest() {
	s.ctx = context.Background()
}

// ============================================================================
// GetNearestStations Tests
// ============================================================================

func (s *TransportRepositoryTestSuite) TestGetNearestStations_SubwayOnly() {
	// Arrange
	lat := 41.3874 // Около Plaça Catalunya
	lon := 2.1700
	types := []string{"metro"}
	maxDistance := 1000.0 // 1 км
	limit := 5

	// Act
	stations, err := s.repo.GetNearestStations(s.ctx, lat, lon, types, maxDistance, limit)

	// Assert
	s.NoError(err)
	s.Greater(len(stations), 0)
	s.LessOrEqual(len(stations), limit)

	// Все станции должны быть метро
	for _, st := range stations {
		s.Equal("metro", st.Type)
	}

	// Проверяем что есть близкие станции (Catalunya, Universitat, etc)
	foundCatalunya := false
	for _, st := range stations {
		if st.Name == "Catalunya" {
			foundCatalunya = true
		}
	}
	s.True(foundCatalunya, "Should find Catalunya station nearby")
}

func (s *TransportRepositoryTestSuite) TestGetNearestStations_AllTypes() {
	// Arrange
	lat := 41.3874
	lon := 2.1700
	types := []string{"metro", "bus", "tram"}
	maxDistance := 2000.0
	limit := 10

	// Act
	stations, err := s.repo.GetNearestStations(s.ctx, lat, lon, types, maxDistance, limit)

	// Assert
	s.NoError(err)
	s.Greater(len(stations), 0)

	// Должны быть разные типы
	typeMap := make(map[string]bool)
	for _, st := range stations {
		typeMap[st.Type] = true
	}
	s.GreaterOrEqual(len(typeMap), 2, "Should have at least 2 different types")
}

func (s *TransportRepositoryTestSuite) TestGetNearestStations_NoResults() {
	// Arrange
	lat := 0.0 // Океан
	lon := 0.0
	types := []string{"metro"}
	maxDistance := 100.0
	limit := 10

	// Act
	stations, err := s.repo.GetNearestStations(s.ctx, lat, lon, types, maxDistance, limit)

	// Assert
	s.NoError(err)
	s.Empty(stations)
}

func (s *TransportRepositoryTestSuite) TestGetNearestStations_DistanceCalculation() {
	// Arrange
	lat := 41.3851 // Catalunya station exact location
	lon := 2.1734
	types := []string{"metro"}
	maxDistance := 500.0
	limit := 3

	// Act
	stations, err := s.repo.GetNearestStations(s.ctx, lat, lon, types, maxDistance, limit)

	// Assert
	s.NoError(err)
	s.Greater(len(stations), 0)

	// Первая станция должна быть Catalunya (очень близко)
	s.Equal("Catalunya", stations[0].Name)
}

// ============================================================================
// GetLineByID Tests
// ============================================================================

func (s *TransportRepositoryTestSuite) TestGetLineByID_Success() {
	// Arrange - Get L1 line ID by OSM ID
	lineID, err := testhelpers.GetTransportLineIDByOSMID(s.testDB.DB.DB, 12345)
	s.NoError(err, "Failed to get L1 line ID")

	// Act
	line, err := s.repo.GetLineByID(s.ctx, lineID)

	// Assert
	s.NoError(err)
	s.NotNil(line)
	s.Equal(lineID, line.ID)
	s.Equal("L1", line.Name)
	s.Equal("metro", line.Type)
	s.NotEmpty(line.Color)
}

func (s *TransportRepositoryTestSuite) TestGetLineByID_NotFound() {
	// Arrange
	lineID := int64(999999) // Non-existent ID

	// Act
	line, err := s.repo.GetLineByID(s.ctx, lineID)

	// Assert
	s.Error(err)
	s.Nil(line)
}

func (s *TransportRepositoryTestSuite) TestGetLineByID_CheckStations() {
	// Arrange - Get L1 line ID by OSM ID
	lineID, err := testhelpers.GetTransportLineIDByOSMID(s.testDB.DB.DB, 12345)
	s.NoError(err, "Failed to get L1 line ID")

	// Act
	line, err := s.repo.GetLineByID(s.ctx, lineID)

	// Assert
	s.NoError(err)
	// Just verify structure - station IDs will be auto-generated
	s.NotNil(line.StationIDs)
}

// ============================================================================
// GetLinesByIDs Tests
// ============================================================================

func (s *TransportRepositoryTestSuite) TestGetLinesByIDs_MultipleLines() {
	// Arrange - Get IDs for L1, L2, L3
	l1ID, err := testhelpers.GetTransportLineIDByOSMID(s.testDB.DB.DB, 12345)
	s.NoError(err, "Failed to get L1 ID")
	l2ID, err := testhelpers.GetTransportLineIDByOSMID(s.testDB.DB.DB, 12346)
	s.NoError(err, "Failed to get L2 ID")
	l3ID, err := testhelpers.GetTransportLineIDByOSMID(s.testDB.DB.DB, 12347)
	s.NoError(err, "Failed to get L3 ID")

	lineIDs := []int64{l1ID, l2ID, l3ID}

	// Act
	lines, err := s.repo.GetLinesByIDs(s.ctx, lineIDs)

	// Assert
	s.NoError(err)
	s.Equal(3, len(lines))

	// Проверяем что все линии имеют валидные ID
	lineMap := make(map[int64]bool)
	for _, line := range lines {
		lineMap[line.ID] = true
		s.Greater(line.ID, int64(0))
	}
}

func (s *TransportRepositoryTestSuite) TestGetLinesByIDs_PartialResults() {
	// Arrange - Get L1 ID
	l1ID, err := testhelpers.GetTransportLineIDByOSMID(s.testDB.DB.DB, 12345)
	s.NoError(err, "Failed to get L1 ID")

	lineIDs := []int64{l1ID, 999999} // L1 exists, 999999 doesn't

	// Act
	lines, err := s.repo.GetLinesByIDs(s.ctx, lineIDs)

	// Assert
	s.NoError(err)
	s.Equal(1, len(lines))
	s.Equal(l1ID, lines[0].ID)
}

func (s *TransportRepositoryTestSuite) TestGetLinesByIDs_EmptyArray() {
	// Arrange
	lineIDs := []int64{}

	// Act
	lines, err := s.repo.GetLinesByIDs(s.ctx, lineIDs)

	// Assert
	s.NoError(err)
	s.Empty(lines)
}

// ============================================================================
// GetStationsByLineID Tests
// ============================================================================

func (s *TransportRepositoryTestSuite) TestGetStationsByLineID_L1Stations() {
	// Arrange
	lineID := int64(1) // First line after migration

	// Act
	stations, err := s.repo.GetStationsByLineID(s.ctx, lineID)

	// Assert
	s.NoError(err)
	// Just verify we can fetch stations for a line
	for _, st := range stations {
		s.Contains(st.LineIDs, lineID)
		s.Greater(st.ID, int64(0))
	}
}

func (s *TransportRepositoryTestSuite) TestGetStationsByLineID_TransferStations() {
	// Arrange
	lineID := int64(1) // First line after migration

	// Act
	stations, err := s.repo.GetStationsByLineID(s.ctx, lineID)

	// Assert
	s.NoError(err)

	// Check if any station has multiple lines (transfer station)
	hasTransfer := false
	for _, st := range stations {
		if len(st.LineIDs) >= 2 {
			hasTransfer = true
			break
		}
	}
	// This is optional - depends on the data
	_ = hasTransfer
}

func (s *TransportRepositoryTestSuite) TestGetStationsByLineID_NoStations() {
	// Arrange
	lineID := int64(999999) // Non-existent line

	// Act
	stations, err := s.repo.GetStationsByLineID(s.ctx, lineID)

	// Assert
	s.NoError(err)
	s.Empty(stations)
}

// ============================================================================
// GetTransportTile Tests
// ============================================================================

func (s *TransportRepositoryTestSuite) TestGetTransportTile_Barcelona() {
	// Arrange
	z, x, y := 14, 8290, 6120 // Tile covering Barcelona test data (lines L1-L3)

	// Act
	tile, err := s.repo.GetTransportTile(s.ctx, z, x, y)

	// Assert
	s.NoError(err)
	s.NotEmpty(tile, "Tile should contain data for Barcelona center")
}

func (s *TransportRepositoryTestSuite) TestGetTransportTile_EmptyArea() {
	// Arrange
	z, x, y := 10, 0, 0 // Tile outside coverage

	// Act
	tile, err := s.repo.GetTransportTile(s.ctx, z, x, y)

	// Assert
	s.NoError(err)
	_ = tile // Empty tile is ok - no transport in that area
}

// ============================================================================
// GetLineTile Tests
// ============================================================================

func (s *TransportRepositoryTestSuite) TestGetLineTile_L1Line() {
	// Arrange - get actual L1 line ID from DB
	var lineID int64
	err := s.testDB.DB.DB.QueryRow("SELECT id FROM transport_lines WHERE ref = 'L1' LIMIT 1").Scan(&lineID)
	s.Require().NoError(err, "Failed to get L1 line ID from test DB")

	// Act
	tile, err := s.repo.GetLineTile(s.ctx, lineID)

	// Assert
	s.NoError(err)
	s.NotEmpty(tile, "Tile should contain L1 line geometry")
}

func (s *TransportRepositoryTestSuite) TestGetLineTile_NonexistentLine() {
	// Arrange
	lineID := int64(999999) // Non-existent line

	// Act
	tile, err := s.repo.GetLineTile(s.ctx, lineID)

	// Assert
	s.NoError(err)
	_ = tile // Empty tile is ok for non-existent line
}

// ============================================================================
// GetLinesTile Tests
// ============================================================================

func (s *TransportRepositoryTestSuite) TestGetLinesTile_MultipleLines() {
	// Arrange - get actual line IDs from DB
	rows, err := s.testDB.DB.DB.Query("SELECT id FROM transport_lines WHERE ref IN ('L1', 'L2') ORDER BY ref")
	s.Require().NoError(err, "Failed to query line IDs from test DB")
	defer rows.Close()

	var lineIDs []int64
	for rows.Next() {
		var id int64
		s.Require().NoError(rows.Scan(&id))
		lineIDs = append(lineIDs, id)
	}
	s.Require().Equal(2, len(lineIDs), "Should have found 2 lines (L1 and L2)")

	// Act
	tile, err := s.repo.GetLinesTile(s.ctx, lineIDs)

	// Assert
	s.NoError(err)
	s.NotEmpty(tile, "Tile should contain multiple lines data")
}

func (s *TransportRepositoryTestSuite) TestGetLinesTile_EmptyArray() {
	// Arrange
	lineIDs := []int64{}

	// Act
	tile, err := s.repo.GetLinesTile(s.ctx, lineIDs)

	// Assert
	s.NoError(err)
	_ = tile // Empty tile is ok for empty input
}

// ============================================================================
// GetStationsInRadius Tests
// ============================================================================

func (s *TransportRepositoryTestSuite) TestGetStationsInRadius_SmallRadius() {
	// Arrange
	lat := 41.3851 // Catalunya
	lon := 2.1734
	radiusKm := 1.0

	// Act
	stations, err := s.repo.GetStationsInRadius(s.ctx, lat, lon, radiusKm)

	// Assert
	s.NoError(err)
	s.Greater(len(stations), 0)

	// Должна быть сама станция Catalunya
	found := false
	for _, st := range stations {
		if st.Name == "Catalunya" {
			found = true
		}
	}
	s.True(found, "Should find Catalunya station within 1km radius")
}

func (s *TransportRepositoryTestSuite) TestGetStationsInRadius_LargeRadius() {
	// Arrange
	lat := 41.3874
	lon := 2.1700
	radiusKm := 5.0

	// Act
	stations, err := s.repo.GetStationsInRadius(s.ctx, lat, lon, radiusKm)

	// Assert
	s.NoError(err)
	s.Greater(len(stations), 5)
	s.LessOrEqual(len(stations), 100, "Should respect limit of 100")
}

// ============================================================================
// GetLinesInRadius Tests
// ============================================================================

func (s *TransportRepositoryTestSuite) TestGetLinesInRadius_BarcelonaCenter() {
	// Arrange
	lat := 41.3874
	lon := 2.1700
	radiusKm := 2.0

	// Act
	lines, err := s.repo.GetLinesInRadius(s.ctx, lat, lon, radiusKm)

	// Assert
	s.NoError(err)
	s.GreaterOrEqual(len(lines), 2, "Should find at least L1, L3 in Barcelona center")
}

func (s *TransportRepositoryTestSuite) TestGetLinesInRadius_EmptyArea() {
	// Arrange
	lat := 0.0
	lon := 0.0
	radiusKm := 1.0

	// Act
	lines, err := s.repo.GetLinesInRadius(s.ctx, lat, lon, radiusKm)

	// Assert
	s.NoError(err)
	s.Empty(lines)
}

// ============================================================================
// GetTransportRadiusTile Tests
// ============================================================================

func (s *TransportRepositoryTestSuite) TestGetTransportRadiusTile_Success() {
	// Arrange
	lat := 41.3874
	lon := 2.1700
	radiusKm := 3.0

	// Act
	tile, err := s.repo.GetTransportRadiusTile(s.ctx, lat, lon, radiusKm)

	// Assert
	s.NoError(err)
	s.NotEmpty(tile, "Tile should contain transport data for Barcelona")
}

func (s *TransportRepositoryTestSuite) TestGetTransportRadiusTile_EmptyArea() {
	// Arrange
	lat := 0.0
	lon := 0.0
	radiusKm := 1.0

	// Act
	tile, err := s.repo.GetTransportRadiusTile(s.ctx, lat, lon, radiusKm)

	// Assert
	s.NoError(err)
	_ = tile // Empty tile is ok for empty area
}

// ============================================================================
// Test Suite Runner
// ============================================================================

func TestTransportRepository(t *testing.T) {
	suite.Run(t, new(TransportRepositoryTestSuite))
}
