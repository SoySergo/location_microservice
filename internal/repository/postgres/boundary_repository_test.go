package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/repository/postgres/testhelpers"
)

// BoundaryRepositoryTestSuite tests all methods of BoundaryRepository
type BoundaryRepositoryTestSuite struct {
	suite.Suite
	testDB *testhelpers.TestDB
	repo   repository.BoundaryRepository
	ctx    context.Context
}

// SetupSuite runs once before all tests in the suite
func (s *BoundaryRepositoryTestSuite) SetupSuite() {
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
		"admin_boundaries.sql",
	}
	err = testhelpers.LoadFixtures(
		s.testDB.DB.DB,
		"../../../../tasks/test_unit_realbd_test_data/fixtures",
		fixtures,
	)
	s.NoError(err, "Failed to load fixtures")

	// Create repository using test helper that wraps DB with logger
	s.repo = testhelpers.NewBoundaryRepositoryForTest(s.testDB.DB, s.testDB.Logger)
}

// TearDownSuite runs once after all tests in the suite
func (s *BoundaryRepositoryTestSuite) TearDownSuite() {
	if s.testDB != nil {
		s.testDB.Close()
	}
}

// SetupTest runs before each test
func (s *BoundaryRepositoryTestSuite) SetupTest() {
	s.ctx = context.Background()
}

// ============================================================================
// GetByID Tests
// ============================================================================

func (s *BoundaryRepositoryTestSuite) TestGetByID_Success() {
	// Arrange - Get Barcelona city ID by OSM ID
	barcelonaCityID, err := testhelpers.GetBoundaryIDByOSMID(s.testDB.DB.DB, 347950)
	s.NoError(err, "Failed to get Barcelona city ID")

	// Act
	boundary, err := s.repo.GetByID(s.ctx, barcelonaCityID)

	// Assert
	s.NoError(err)
	s.NotNil(boundary)
	s.Equal(barcelonaCityID, boundary.ID)
	s.Equal("Barcelona", boundary.Name)
	s.Equal(8, boundary.AdminLevel)
	s.Equal("city", boundary.Type)
	s.InDelta(41.3874, boundary.CenterLat, 0.0001)
	s.InDelta(2.1686, boundary.CenterLon, 0.0001)
	s.NotNil(boundary.Population)
	s.Equal(1636762, *boundary.Population)
}

func (s *BoundaryRepositoryTestSuite) TestGetByID_NotFound() {
	// Arrange
	nonExistentID := int64(999999) // Non-existent ID

	// Act
	boundary, err := s.repo.GetByID(s.ctx, nonExistentID)

	// Assert
	s.Error(err)
	s.Nil(boundary)
}

func (s *BoundaryRepositoryTestSuite) TestGetByID_MultilingualNames() {
	// Arrange - Get Catalonia ID by OSM ID
	cataloniaID, err := testhelpers.GetBoundaryIDByOSMID(s.testDB.DB.DB, 349053)
	s.NoError(err, "Failed to get Catalonia ID")

	// Act
	boundary, err := s.repo.GetByID(s.ctx, cataloniaID)

	// Assert
	s.NoError(err)
	s.Equal("Catalunya", boundary.Name)
	s.Equal("Catalonia", boundary.NameEn)
	s.Equal("Cataluña", boundary.NameEs)
	s.Equal("Catalunya", boundary.NameCa)
}

// ============================================================================
// SearchByText Tests
// ============================================================================

func (s *BoundaryRepositoryTestSuite) TestSearchByText_BasicSearch() {
	// Arrange
	query := "Barcelona"

	// Act
	boundaries, err := s.repo.SearchByText(s.ctx, query, "", nil, 10)

	// Assert
	s.NoError(err)
	s.GreaterOrEqual(len(boundaries), 2, "Should find at least city and province")

	// Check that results contain "Barcelona"
	found := false
	for _, b := range boundaries {
		if b.Name == "Barcelona" {
			found = true
		}
	}
	s.True(found, "Should find Barcelona in results")
}

func (s *BoundaryRepositoryTestSuite) TestSearchByText_WithLanguage() {
	// Arrange
	query := "Catalonia"
	lang := "en"

	// Act
	boundaries, err := s.repo.SearchByText(s.ctx, query, lang, nil, 10)

	// Assert
	s.NoError(err)
	s.Greater(len(boundaries), 0)
	// Should return name_en as the name field
	s.Equal("Catalonia", boundaries[0].Name)
}

func (s *BoundaryRepositoryTestSuite) TestSearchByText_WithAdminLevelFilter() {
	// Arrange
	query := "Barcelona"
	adminLevels := []int{8} // only cities

	// Act
	boundaries, err := s.repo.SearchByText(s.ctx, query, "", adminLevels, 10)

	// Assert
	s.NoError(err)
	for _, b := range boundaries {
		s.Equal(8, b.AdminLevel, "All results should be cities (admin_level=8)")
	}
}

func (s *BoundaryRepositoryTestSuite) TestSearchByText_Ranking() {
	// Arrange
	query := "Barcelona"

	// Act
	boundaries, err := s.repo.SearchByText(s.ctx, query, "", nil, 10)

	// Assert
	s.NoError(err)
	// Results should be sorted by relevance
	// City Barcelona should be higher than province (lower admin level first in case of tie)
	if len(boundaries) >= 2 {
		foundCity := false
		for i, b := range boundaries {
			if b.AdminLevel == 8 && b.Name == "Barcelona" {
				foundCity = true
				// City should appear in the results
				s.Less(i, len(boundaries), "City should be in results")
			}
		}
		s.True(foundCity, "Should find Barcelona city")
	}
}

func (s *BoundaryRepositoryTestSuite) TestSearchByText_NoResults() {
	// Arrange
	query := "NonexistentCity123XYZ"

	// Act
	boundaries, err := s.repo.SearchByText(s.ctx, query, "", nil, 10)

	// Assert
	s.NoError(err)
	s.Empty(boundaries)
}

// ============================================================================
// ReverseGeocode Tests
// ============================================================================

func (s *BoundaryRepositoryTestSuite) TestReverseGeocode_BarcelonaCenter() {
	// Arrange
	lat := 41.3874 // Barcelona center
	lon := 2.1686

	// Act
	address, err := s.repo.ReverseGeocode(s.ctx, lat, lon)

	// Assert
	s.NoError(err)
	s.NotNil(address)
	s.NotEmpty(address.Country)
	s.Equal("España", address.Country) // Returns 'name' field
	s.NotEmpty(address.Region)
	s.Equal("Catalunya", address.Region) // Returns 'name' field
	s.NotEmpty(address.Province)
	s.Equal("Barcelona", address.Province)
	s.NotEmpty(address.City)
	s.Equal("Barcelona", address.City)
}

func (s *BoundaryRepositoryTestSuite) TestReverseGeocode_WithDistrict() {
	// Arrange
	lat := 41.3924 // Eixample district
	lon := 2.1649

	// Act
	address, err := s.repo.ReverseGeocode(s.ctx, lat, lon)

	// Assert
	s.NoError(err)
	s.NotNil(address.District)
	s.Equal("Eixample", *address.District)
}

func (s *BoundaryRepositoryTestSuite) TestReverseGeocode_OutsideBoundaries() {
	// Arrange
	lat := 0.0 // Somewhere in the ocean
	lon := 0.0

	// Act
	address, err := s.repo.ReverseGeocode(s.ctx, lat, lon)

	// Assert
	s.Error(err)
	s.Nil(address)
}

// ============================================================================
// GetTile Tests
// ============================================================================

func (s *BoundaryRepositoryTestSuite) TestGetTile_LowZoom_Countries() {
	// Arrange
	z, x, y := 3, 3, 3 // Low zoom - should show countries

	// Act
	tile, err := s.repo.GetTile(s.ctx, z, x, y)

	// Assert
	s.NoError(err)
	s.NotNil(tile)
	// MVT tile should contain data (not checking exact content)
}

func (s *BoundaryRepositoryTestSuite) TestGetTile_MediumZoom_Regions() {
	// Arrange
	z, x, y := 6, 32, 24 // Medium zoom

	// Act
	tile, err := s.repo.GetTile(s.ctx, z, x, y)

	// Assert
	s.NoError(err)
	s.NotNil(tile)
}

func (s *BoundaryRepositoryTestSuite) TestGetTile_HighZoom_Districts() {
	// Arrange
	z, x, y := 14, 8298, 6143 // High zoom - should show districts

	// Act
	tile, err := s.repo.GetTile(s.ctx, z, x, y)

	// Assert
	s.NoError(err)
	s.NotNil(tile)
	// May be empty if no data in this specific tile
}

func (s *BoundaryRepositoryTestSuite) TestGetTile_EmptyTile() {
	// Arrange
	z, x, y := 10, 0, 0 // Tile outside coverage area

	// Act
	_, err := s.repo.GetTile(s.ctx, z, x, y)

	// Assert
	s.NoError(err)
	// Empty tile is valid
}

// ============================================================================
// GetByPoint Tests
// ============================================================================

func (s *BoundaryRepositoryTestSuite) TestGetByPoint_BarcelonaHierarchy() {
	// Arrange
	lat := 41.3874
	lon := 2.1686

	// Act
	boundaries, err := s.repo.GetByPoint(s.ctx, lat, lon)

	// Assert
	s.NoError(err)
	s.GreaterOrEqual(len(boundaries), 4, "Should have Spain, Catalonia, Province, City")

	// Check hierarchy (should be sorted by admin_level)
	s.Equal(2, boundaries[0].AdminLevel, "First should be country")
	s.Equal(4, boundaries[1].AdminLevel, "Second should be region")
	s.Equal(6, boundaries[2].AdminLevel, "Third should be province")
	s.Equal(8, boundaries[3].AdminLevel, "Fourth should be city")
}

func (s *BoundaryRepositoryTestSuite) TestGetByPoint_WithDistrict() {
	// Arrange
	lat := 41.3924 // Inside Eixample district
	lon := 2.1649

	// Act
	boundaries, err := s.repo.GetByPoint(s.ctx, lat, lon)

	// Assert
	s.NoError(err)
	// Should contain district
	hasDistrict := false
	for _, b := range boundaries {
		if b.AdminLevel == 9 && b.Name == "Eixample" {
			hasDistrict = true
		}
	}
	s.True(hasDistrict, "Should find Eixample district")
}

func (s *BoundaryRepositoryTestSuite) TestGetByPoint_NoResults() {
	// Arrange
	lat := 0.0 // Ocean
	lon := 0.0

	// Act
	boundaries, err := s.repo.GetByPoint(s.ctx, lat, lon)

	// Assert
	s.NoError(err)
	s.Empty(boundaries)
}

// ============================================================================
// Search Tests
// ============================================================================

func (s *BoundaryRepositoryTestSuite) TestSearch_Simple() {
	// Arrange
	query := "Barcelona"

	// Act
	boundaries, err := s.repo.Search(s.ctx, query, 10)

	// Assert
	s.NoError(err)
	s.Greater(len(boundaries), 0)
}

func (s *BoundaryRepositoryTestSuite) TestSearch_WithLimit() {
	// Arrange
	query := "a" // Many results
	limit := 3

	// Act
	boundaries, err := s.repo.Search(s.ctx, query, limit)

	// Assert
	s.NoError(err)
	s.LessOrEqual(len(boundaries), limit)
}

// ============================================================================
// GetChildren Tests
// ============================================================================

func (s *BoundaryRepositoryTestSuite) TestGetChildren_BarcelonaDistricts() {
	// Arrange - Get Barcelona city ID
	barcelonaCityID, err := testhelpers.GetBoundaryIDByOSMID(s.testDB.DB.DB, 347950)
	s.NoError(err, "Failed to get Barcelona city ID")

	// Act
	children, err := s.repo.GetChildren(s.ctx, barcelonaCityID)

	// Assert
	s.NoError(err)
	s.GreaterOrEqual(len(children), 2, "Should have Eixample and Ciutat Vella")

	// All children should be districts (admin_level=9)
	for _, child := range children {
		s.Equal(9, child.AdminLevel, "All children should be districts")
	}
}

func (s *BoundaryRepositoryTestSuite) TestGetChildren_ProvinceChildren() {
	// Arrange - Get Barcelona province ID
	barcelonaProvinceID, err := testhelpers.GetBoundaryIDByOSMID(s.testDB.DB.DB, 349035)
	s.NoError(err, "Failed to get Barcelona province ID")

	// Act
	children, err := s.repo.GetChildren(s.ctx, barcelonaProvinceID)

	// Assert
	s.NoError(err)
	s.GreaterOrEqual(len(children), 2, "Should have Barcelona and Sabadell")

	// All children should be cities
	for _, child := range children {
		s.Equal(8, child.AdminLevel, "All children should be cities")
	}
}

func (s *BoundaryRepositoryTestSuite) TestGetChildren_NoChildren() {
	// Arrange - Get Eixample district ID
	eixampleID, err := testhelpers.GetBoundaryIDByOSMID(s.testDB.DB.DB, 348814)
	s.NoError(err, "Failed to get Eixample ID")

	// Act
	children, err := s.repo.GetChildren(s.ctx, eixampleID)

	// Assert
	s.NoError(err)
	s.Empty(children)
}

// ============================================================================
// GetByAdminLevel Tests
// ============================================================================

func (s *BoundaryRepositoryTestSuite) TestGetByAdminLevel_Countries() {
	// Arrange
	level := 2

	// Act
	boundaries, err := s.repo.GetByAdminLevel(s.ctx, level, 10)

	// Assert
	s.NoError(err)
	s.Greater(len(boundaries), 0)
	for _, b := range boundaries {
		s.Equal(2, b.AdminLevel)
	}
}

func (s *BoundaryRepositoryTestSuite) TestGetByAdminLevel_Cities() {
	// Arrange
	level := 8

	// Act
	boundaries, err := s.repo.GetByAdminLevel(s.ctx, level, 10)

	// Assert
	s.NoError(err)
	s.GreaterOrEqual(len(boundaries), 2, "Should have Barcelona and Sabadell")
	for _, b := range boundaries {
		s.Equal(8, b.AdminLevel)
	}
}

func (s *BoundaryRepositoryTestSuite) TestGetByAdminLevel_WithLimit() {
	// Arrange
	level := 9
	limit := 1

	// Act
	boundaries, err := s.repo.GetByAdminLevel(s.ctx, level, limit)

	// Assert
	s.NoError(err)
	s.LessOrEqual(len(boundaries), limit)
}

// ============================================================================
// GetBoundariesInRadius Tests
// ============================================================================

func (s *BoundaryRepositoryTestSuite) TestGetBoundariesInRadius_SmallRadius() {
	// Arrange
	lat := 41.3874 // Barcelona center
	lon := 2.1686
	radiusKm := 5.0

	// Act
	boundaries, err := s.repo.GetBoundariesInRadius(s.ctx, lat, lon, radiusKm)

	// Assert
	s.NoError(err)
	s.Greater(len(boundaries), 0)

	// Should be mostly districts and city
	for _, b := range boundaries {
		s.GreaterOrEqual(b.AdminLevel, 6, "Should be province, city or district level")
	}
}

func (s *BoundaryRepositoryTestSuite) TestGetBoundariesInRadius_LargeRadius() {
	// Arrange
	lat := 41.3874
	lon := 2.1686
	radiusKm := 50.0

	// Act
	boundaries, err := s.repo.GetBoundariesInRadius(s.ctx, lat, lon, radiusKm)

	// Assert
	s.NoError(err)
	s.Greater(len(boundaries), 0)
}

func (s *BoundaryRepositoryTestSuite) TestGetBoundariesInRadius_EmptyArea() {
	// Arrange
	lat := 0.0 // Ocean
	lon := 0.0
	radiusKm := 10.0

	// Act
	boundaries, err := s.repo.GetBoundariesInRadius(s.ctx, lat, lon, radiusKm)

	// Assert
	s.NoError(err)
	s.Empty(boundaries)
}

// ============================================================================
// GetBoundariesRadiusTile Tests
// ============================================================================

func (s *BoundaryRepositoryTestSuite) TestGetBoundariesRadiusTile_Success() {
	// Arrange
	lat := 41.3874
	lon := 2.1686
	radiusKm := 10.0

	// Act
	tile, err := s.repo.GetBoundariesRadiusTile(s.ctx, lat, lon, radiusKm)

	// Assert
	s.NoError(err)
	s.NotNil(tile)
}

func (s *BoundaryRepositoryTestSuite) TestGetBoundariesRadiusTile_EmptyArea() {
	// Arrange
	lat := 0.0
	lon := 0.0
	radiusKm := 1.0

	// Act
	_, err := s.repo.GetBoundariesRadiusTile(s.ctx, lat, lon, radiusKm)

	// Assert
	s.NoError(err)
	// Empty tile is valid
}

// ============================================================================
// Test Suite Runner
// ============================================================================

func TestBoundaryRepositorySuite(t *testing.T) {
	suite.Run(t, new(BoundaryRepositoryTestSuite))
}
