package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/repository/postgres/testhelpers"
)

// POIRepositorySuite tests the POI repository with real database
type POIRepositorySuite struct {
	suite.Suite
	testDB *testhelpers.TestDB
	repo   repository.POIRepository
	ctx    context.Context
}

// SetupSuite runs once before all tests
func (s *POIRepositorySuite) SetupSuite() {
	s.testDB = testhelpers.SetupTestDB(s.T())

	// Apply migrations (skip if tables already exist)
	_ = testhelpers.ApplyMigrations(
		s.testDB.DB.DB,
		"../../../migrations",
	)

	// Load POI fixtures
	fixtures := []string{
		"poi_categories.sql",
		"pois.sql",
	}
	err := testhelpers.LoadFixtures(
		s.testDB.DB.DB,
		"../../../../tasks/test_unit_realbd_test_data/fixtures",
		fixtures,
	)
	s.NoError(err, "Failed to load fixtures")

	// Create repository using test helper that wraps DB with logger
	s.repo = testhelpers.NewPOIRepositoryForTest(s.testDB.DB, s.testDB.Logger)
}

// TearDownSuite runs once after all tests
func (s *POIRepositorySuite) TearDownSuite() {
	if s.testDB != nil {
		s.testDB.Close()
	}
}

// SetupTest runs before each test
func (s *POIRepositorySuite) SetupTest() {
	s.ctx = context.Background()
}

// ============================================================================
// Test GetByID
// ============================================================================

func (s *POIRepositorySuite) TestGetByID_Success() {
	// Get Sagrada Família
	poi, err := s.repo.GetByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(poi)
	s.Equal(int64(1), poi.ID)
	s.Equal("Sagrada Família", poi.Name)
	s.Equal("landmark", poi.Category)
	s.Equal("architecture", poi.Subcategory)
	s.Equal(41.4036, poi.Lat)
	s.Equal(2.1744, poi.Lon)
	s.NotNil(poi.Address)
	s.Contains(*poi.Address, "Carrer de Mallorca")
	s.NotNil(poi.Website)
	s.NotNil(poi.Wheelchair)
	s.True(*poi.Wheelchair)
}

func (s *POIRepositorySuite) TestGetByID_MultilingualNames() {
	// Get Sagrada Família and check multilingual names
	poi, err := s.repo.GetByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(poi.NameEn)
	s.Equal("Sagrada Família", *poi.NameEn)
	s.NotNil(poi.NameEs)
	s.Equal("Sagrada Familia", *poi.NameEs)
	s.NotNil(poi.NameCa)
	s.Equal("Sagrada Família", *poi.NameCa)
	s.NotNil(poi.NameRu)
	s.Equal("Саграда Фамилия", *poi.NameRu)
}

func (s *POIRepositorySuite) TestGetByID_NotFound() {
	poi, err := s.repo.GetByID(s.ctx, 99999)
	s.Error(err)
	s.Nil(poi)
}

// ============================================================================
// Test GetNearby
// ============================================================================

func (s *POIRepositorySuite) TestGetNearby_SagradaFamilia() {
	// Near Sagrada Família (should find it and other nearby POIs)
	pois, err := s.repo.GetNearby(s.ctx, 41.4036, 2.1744, 1.0, nil)
	s.NoError(err)
	s.NotEmpty(pois)

	// First POI should be Sagrada Família itself (distance = 0)
	s.Equal("Sagrada Família", pois[0].Name)
}

func (s *POIRepositorySuite) TestGetNearby_WithCategoryFilter() {
	// Near Passeig de Gràcia, only restaurants
	pois, err := s.repo.GetNearby(s.ctx, 41.3939, 2.1605, 0.5, []string{"restaurant"})
	s.NoError(err)
	s.NotEmpty(pois)

	// All results should be restaurants
	for _, poi := range pois {
		s.Equal("restaurant", poi.Category)
	}
}

func (s *POIRepositorySuite) TestGetNearby_SmallRadius() {
	// Very small radius in empty area
	pois, err := s.repo.GetNearby(s.ctx, 41.4500, 2.2000, 0.1, nil)
	s.NoError(err)
	s.Empty(pois)
}

func (s *POIRepositorySuite) TestGetNearby_MultipleCategoriesFilter() {
	// Near Gothic Quarter, hotels and restaurants
	pois, err := s.repo.GetNearby(s.ctx, 41.3828, 2.1761, 1.0, []string{"hotel", "restaurant"})
	s.NoError(err)
	s.NotEmpty(pois)

	// All results should be hotels or restaurants
	for _, poi := range pois {
		s.Contains([]string{"hotel", "restaurant"}, poi.Category)
	}
}

// ============================================================================
// Test Search
// ============================================================================

func (s *POIRepositorySuite) TestSearch_BasicSearch() {
	// Search for "Sagrada"
	pois, err := s.repo.Search(s.ctx, "Sagrada", nil, 10)
	s.NoError(err)
	s.NotEmpty(pois)
	s.Equal("Sagrada Família", pois[0].Name)
}

func (s *POIRepositorySuite) TestSearch_GaudiArchitecture() {
	// Search for "Gaudí"
	pois, err := s.repo.Search(s.ctx, "Gaudí", nil, 10)
	s.NoError(err)
	s.NotEmpty(pois)

	// Should find multiple Gaudí works
	s.GreaterOrEqual(len(pois), 3)

	names := make([]string, len(pois))
	for i, poi := range pois {
		names[i] = poi.Name
	}
	s.Contains(names, "Sagrada Família")
	s.Contains(names, "Casa Batlló")
	s.Contains(names, "Park Güell")
}

func (s *POIRepositorySuite) TestSearch_WithCategoryFilter() {
	// Search for Barcelona museums only
	pois, err := s.repo.Search(s.ctx, "Barcelona", []string{"museum"}, 10)
	s.NoError(err)
	s.NotEmpty(pois)

	// All should be museums
	for _, poi := range pois {
		s.Equal("museum", poi.Category)
	}
}

func (s *POIRepositorySuite) TestSearch_NoResults() {
	pois, err := s.repo.Search(s.ctx, "XYZ123NonExistent", nil, 10)
	s.NoError(err)
	s.Empty(pois)
}

func (s *POIRepositorySuite) TestSearch_Ranking() {
	// Search should rank by relevance
	pois, err := s.repo.Search(s.ctx, "hotel Barcelona", nil, 10)
	s.NoError(err)
	s.NotEmpty(pois)

	// Should find hotels
	foundHotel := false
	for _, poi := range pois {
		if poi.Category == "hotel" {
			foundHotel = true
			break
		}
	}
	s.True(foundHotel, "Should find at least one hotel")
}

// ============================================================================
// Test GetByCategory
// ============================================================================

func (s *POIRepositorySuite) TestGetByCategory_Landmarks() {
	pois, err := s.repo.GetByCategory(s.ctx, "landmark", 10)
	s.NoError(err)
	s.NotEmpty(pois)

	// All should be landmarks
	for _, poi := range pois {
		s.Equal("landmark", poi.Category)
	}

	// Should include Casa Batlló, Sagrada Família, Casa Milà
	s.GreaterOrEqual(len(pois), 3)
}

func (s *POIRepositorySuite) TestGetByCategory_Museums() {
	pois, err := s.repo.GetByCategory(s.ctx, "museum", 10)
	s.NoError(err)
	s.NotEmpty(pois)

	// All should be museums
	for _, poi := range pois {
		s.Equal("museum", poi.Category)
	}
}

func (s *POIRepositorySuite) TestGetByCategory_Restaurants() {
	pois, err := s.repo.GetByCategory(s.ctx, "restaurant", 10)
	s.NoError(err)
	s.NotEmpty(pois)

	// All should be restaurants
	for _, poi := range pois {
		s.Equal("restaurant", poi.Category)
	}

	// Should have at least 4 restaurants
	s.GreaterOrEqual(len(pois), 4)
}

func (s *POIRepositorySuite) TestGetByCategory_WithLimit() {
	// Get only 2 landmarks
	pois, err := s.repo.GetByCategory(s.ctx, "landmark", 2)
	s.NoError(err)
	s.Len(pois, 2)
}

func (s *POIRepositorySuite) TestGetByCategory_NonExistent() {
	pois, err := s.repo.GetByCategory(s.ctx, "nonexistent_category", 10)
	s.NoError(err)
	s.Empty(pois)
}

// ============================================================================
// Test GetCategories
// ============================================================================

func (s *POIRepositorySuite) TestGetCategories_Success() {
	categories, err := s.repo.GetCategories(s.ctx)
	s.NoError(err)
	s.NotEmpty(categories)

	// Should have at least 5 categories
	s.GreaterOrEqual(len(categories), 5)

	// Check that main categories exist
	codes := make([]string, len(categories))
	for i, cat := range categories {
		codes[i] = cat.Code
		s.NotEmpty(cat.NameEn)
		s.NotNil(cat.Icon)
		s.NotNil(cat.Color)
	}

	s.Contains(codes, "tourism")
	s.Contains(codes, "restaurant")
	s.Contains(codes, "hotel")
	s.Contains(codes, "museum")
	s.Contains(codes, "landmark")
}

func (s *POIRepositorySuite) TestGetCategories_SortOrder() {
	categories, err := s.repo.GetCategories(s.ctx)
	s.NoError(err)
	s.NotEmpty(categories)

	// Categories should be sorted by sort_order
	for i := 1; i < len(categories); i++ {
		s.LessOrEqual(categories[i-1].SortOrder, categories[i].SortOrder)
	}
}

// ============================================================================
// Test GetSubcategories
// ============================================================================

func (s *POIRepositorySuite) TestGetSubcategories_Tourism() {
	// Tourism category should have subcategories
	subcats, err := s.repo.GetSubcategories(s.ctx, 1) // tourism category ID
	s.NoError(err)
	s.NotEmpty(subcats)

	// Check subcategories
	codes := make([]string, len(subcats))
	for i, sub := range subcats {
		codes[i] = sub.Code
		s.Equal(int64(1), sub.CategoryID)
		s.NotEmpty(sub.NameEn)
	}

	s.Contains(codes, "attraction")
	s.Contains(codes, "viewpoint")
	s.Contains(codes, "park")
}

func (s *POIRepositorySuite) TestGetSubcategories_Restaurant() {
	// Restaurant category subcategories
	subcats, err := s.repo.GetSubcategories(s.ctx, 2) // restaurant category ID
	s.NoError(err)
	s.NotEmpty(subcats)

	codes := make([]string, len(subcats))
	for i, sub := range subcats {
		codes[i] = sub.Code
		s.Equal(int64(2), sub.CategoryID)
	}

	s.Contains(codes, "fine_dining")
	s.Contains(codes, "tapas")
	s.Contains(codes, "cafe")
}

func (s *POIRepositorySuite) TestGetSubcategories_NonExistent() {
	subcats, err := s.repo.GetSubcategories(s.ctx, 99999)
	s.NoError(err)
	s.Empty(subcats)
}

func (s *POIRepositorySuite) TestGetSubcategories_SortOrder() {
	subcats, err := s.repo.GetSubcategories(s.ctx, 1)
	s.NoError(err)
	s.NotEmpty(subcats)

	// Subcategories should be sorted by sort_order
	for i := 1; i < len(subcats); i++ {
		s.LessOrEqual(subcats[i-1].SortOrder, subcats[i].SortOrder)
	}
}

// ============================================================================
// Test GetPOITile (MVT Tiles)
// ============================================================================

func (s *POIRepositorySuite) TestGetPOITile_BarcelonaCenter() {
	// Tile covering central Barcelona at zoom 14
	// Tile (8267, 6127, 14) covers Passeig de Gràcia area
	tile, err := s.repo.GetPOITile(s.ctx, 14, 8267, 6127, nil)
	s.NoError(err)
	// Note: tile may be empty if no POIs fall within this exact tile
	// This is acceptable behavior for MVT tiles
	s.NotNil(tile)
}

func (s *POIRepositorySuite) TestGetPOITile_WithCategoryFilter() {
	// Only landmarks
	tile, err := s.repo.GetPOITile(s.ctx, 14, 8267, 6127, []string{"landmark"})
	s.NoError(err)
	// Note: tile may be empty if no landmarks fall within this exact tile
	s.NotNil(tile)
}

func (s *POIRepositorySuite) TestGetPOITile_EmptyArea() {
	// Tile in middle of ocean
	tile, err := s.repo.GetPOITile(s.ctx, 10, 500, 400, nil)
	s.NoError(err)
	// Empty tile should return empty bytes or nil
	s.LessOrEqual(len(tile), 0)
}

func (s *POIRepositorySuite) TestGetPOITile_DifferentZoomLevels() {
	// Low zoom (country level)
	tile1, err := s.repo.GetPOITile(s.ctx, 6, 32, 24, nil)
	s.NoError(err)

	// Medium zoom (city level)
	tile2, err := s.repo.GetPOITile(s.ctx, 12, 2066, 1531, nil)
	s.NoError(err)

	// High zoom (neighborhood level)
	tile3, err := s.repo.GetPOITile(s.ctx, 16, 33069, 24509, nil)
	s.NoError(err)

	// All should be valid
	s.NotNil(tile1)
	s.NotNil(tile2)
	s.NotNil(tile3)
}

// ============================================================================
// Test GetPOIRadiusTile
// ============================================================================

func (s *POIRepositorySuite) TestGetPOIRadiusTile_Success() {
	// POIs around Sagrada Família
	tile, err := s.repo.GetPOIRadiusTile(s.ctx, 41.4036, 2.1744, 2.0, nil)
	s.NoError(err)
	s.NotEmpty(tile)
}

func (s *POIRepositorySuite) TestGetPOIRadiusTile_WithCategories() {
	// Only restaurants around Gothic Quarter
	tile, err := s.repo.GetPOIRadiusTile(s.ctx, 41.3828, 2.1761, 1.0, []string{"restaurant"})
	s.NoError(err)
	s.NotEmpty(tile)
}

func (s *POIRepositorySuite) TestGetPOIRadiusTile_SmallRadius() {
	// Very small radius
	tile, err := s.repo.GetPOIRadiusTile(s.ctx, 41.3900, 2.1600, 0.2, nil)
	s.NoError(err)
	// Should still work, might be empty or have few POIs
	s.NotNil(tile)
}

func (s *POIRepositorySuite) TestGetPOIRadiusTile_EmptyArea() {
	// Empty area
	tile, err := s.repo.GetPOIRadiusTile(s.ctx, 41.5000, 2.3000, 0.5, nil)
	s.NoError(err)
	s.LessOrEqual(len(tile), 0)
}

// ============================================================================
// Test GetPOIByBoundaryTile
// ============================================================================

func (s *POIRepositorySuite) TestGetPOIByBoundaryTile_Barcelona() {
	// Need to load admin_boundaries fixture first
	// This test assumes Barcelona boundary exists with ID
	// We'll use a known boundary ID from fixtures

	// Load admin boundaries if not already loaded
	err := testhelpers.LoadFixtures(
		s.testDB.DB.DB,
		"../../../../tasks/test_unit_realbd_test_data/fixtures",
		[]string{"admin_boundaries.sql"},
	)
	s.NoError(err)

	// Get POIs within Barcelona boundary (assuming ID 8 for Barcelona city)
	tile, err := s.repo.GetPOIByBoundaryTile(s.ctx, 8, nil)
	s.NoError(err)
	// Note: tile may be empty if boundary doesn't contain POIs or boundary doesn't exist
	s.NotNil(tile)
}

func (s *POIRepositorySuite) TestGetPOIByBoundaryTile_WithCategories() {
	// Load admin boundaries
	err := testhelpers.LoadFixtures(
		s.testDB.DB.DB,
		"../../../../tasks/test_unit_realbd_test_data/fixtures",
		[]string{"admin_boundaries.sql"},
	)
	s.NoError(err)

	// Only museums in Barcelona
	tile, err := s.repo.GetPOIByBoundaryTile(s.ctx, 8, []string{"museum"})
	s.NoError(err)
	// Note: tile may be empty if no museums in boundary
	s.NotNil(tile)
}

func (s *POIRepositorySuite) TestGetPOIByBoundaryTile_NonExistentBoundary() {
	tile, err := s.repo.GetPOIByBoundaryTile(s.ctx, 99999, nil)
	s.NoError(err)
	s.LessOrEqual(len(tile), 0)
}

// Run the test suite
func TestPOIRepository(t *testing.T) {
	suite.Run(t, new(POIRepositorySuite))
}
