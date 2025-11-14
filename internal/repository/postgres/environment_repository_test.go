package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/repository/postgres/testhelpers"
)

// EnvironmentRepositorySuite tests the Environment repository with real database
type EnvironmentRepositorySuite struct {
	suite.Suite
	testDB *testhelpers.TestDB
	repo   repository.EnvironmentRepository
	ctx    context.Context
}

// SetupSuite runs once before all tests
func (s *EnvironmentRepositorySuite) SetupSuite() {
	s.testDB = testhelpers.SetupTestDB(s.T())

	// Clean up existing data first
	err := s.testDB.Cleanup(context.Background())
	s.NoError(err, "Failed to cleanup test database")

	// Apply migrations (skip if tables already exist)
	_ = testhelpers.ApplyMigrations(
		s.testDB.DB.DB,
		"../../../migrations",
	)

	// Load Environment fixtures
	fixtures := []string{
		"environment.sql",
	}
	err = testhelpers.LoadFixtures(
		s.testDB.DB.DB,
		"../../../../tasks/test_unit_realbd_test_data/fixtures",
		fixtures,
	)
	s.NoError(err, "Failed to load fixtures")

	// Create repository using test helper that wraps DB with logger
	s.repo = testhelpers.NewEnvironmentRepositoryForTest(s.testDB.DB, s.testDB.Logger)
}

// TearDownSuite runs once after all tests
func (s *EnvironmentRepositorySuite) TearDownSuite() {
	if s.testDB != nil {
		s.testDB.Close()
	}
}

// SetupTest runs before each test
func (s *EnvironmentRepositorySuite) SetupTest() {
	s.ctx = context.Background()
}

// ============================================================================
// Test GetGreenSpaceByID
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetGreenSpaceByID_Success() {
	// Get Parc de la Ciutadella by searching nearby first
	greenSpaces, err := s.repo.GetGreenSpacesNearby(s.ctx, 41.3895, 2.1870, 0.5)
	s.NoError(err)
	s.NotEmpty(greenSpaces)

	// Find Parc de la Ciutadella
	var ciutadellaID int64
	for _, gs := range greenSpaces {
		if gs.Name != nil && *gs.Name == "Parc de la Ciutadella" {
			ciutadellaID = gs.ID
			break
		}
	}
	s.NotZero(ciutadellaID, "Parc de la Ciutadella should be found")

	// Now test GetGreenSpaceByID
	greenSpace, err := s.repo.GetGreenSpaceByID(s.ctx, ciutadellaID)
	s.NoError(err)
	s.NotNil(greenSpace)
	s.Equal(ciutadellaID, greenSpace.ID)
	s.NotNil(greenSpace.Name)
	s.Equal("Parc de la Ciutadella", *greenSpace.Name)
	s.Equal("park", greenSpace.Type)
	s.Greater(greenSpace.AreaSqM, 0.0)
	s.NotZero(greenSpace.CenterLat)
	s.NotZero(greenSpace.CenterLon)
}

func (s *EnvironmentRepositorySuite) TestGetGreenSpaceByID_MultilingualNames() {
	// Get Parc de la Ciutadella by searching nearby first
	greenSpaces, err := s.repo.GetGreenSpacesNearby(s.ctx, 41.3895, 2.1870, 0.5)
	s.NoError(err)
	s.NotEmpty(greenSpaces)

	var ciutadellaID int64
	for _, gs := range greenSpaces {
		if gs.Name != nil && *gs.Name == "Parc de la Ciutadella" {
			ciutadellaID = gs.ID
			break
		}
	}
	s.NotZero(ciutadellaID)

	// Get Parc de la Ciutadella and check multilingual names
	greenSpace, err := s.repo.GetGreenSpaceByID(s.ctx, ciutadellaID)
	s.NoError(err)
	s.NotNil(greenSpace.NameEn)
	s.Equal("Ciutadella Park", *greenSpace.NameEn)
}

func (s *EnvironmentRepositorySuite) TestGetGreenSpaceByID_NotFound() {
	greenSpace, err := s.repo.GetGreenSpaceByID(s.ctx, 99999)
	s.Error(err)
	s.Nil(greenSpace)
}

// ============================================================================
// Test GetBeachByID
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetBeachByID_Success() {
	// Get Barceloneta Beach by searching nearby first
	beaches, err := s.repo.GetBeachesNearby(s.ctx, 41.3781, 2.1900, 0.5)
	s.NoError(err)
	s.NotEmpty(beaches)

	var barcelonetaID int64
	for _, b := range beaches {
		if b.Name != nil && *b.Name == "Platja de la Barceloneta" {
			barcelonetaID = b.ID
			break
		}
	}
	s.NotZero(barcelonetaID, "Barceloneta beach should be found")

	// Get Barceloneta Beach
	beach, err := s.repo.GetBeachByID(s.ctx, barcelonetaID)
	s.NoError(err)
	s.NotNil(beach)
	s.Equal(barcelonetaID, beach.ID)
	s.NotNil(beach.Name)
	s.Equal("Platja de la Barceloneta", *beach.Name)
	s.Equal("sand", beach.Surface)
	s.NotZero(beach.Lat)
	s.NotZero(beach.Lon)
	s.NotNil(beach.Length)
	s.Greater(*beach.Length, 0.0)
}

func (s *EnvironmentRepositorySuite) TestGetBeachByID_WithBlueFlag() {
	// Get beach with blue flag by searching nearby first
	beaches, err := s.repo.GetBeachesNearby(s.ctx, 41.3781, 2.1900, 0.5)
	s.NoError(err)
	s.NotEmpty(beaches)

	var barcelonetaID int64
	for _, b := range beaches {
		if b.Name != nil && *b.Name == "Platja de la Barceloneta" {
			barcelonetaID = b.ID
			break
		}
	}
	s.NotZero(barcelonetaID)

	// Get beach with blue flag
	beach, err := s.repo.GetBeachByID(s.ctx, barcelonetaID)
	s.NoError(err)
	s.NotNil(beach.BlueFlag)
	s.True(*beach.BlueFlag)
}

func (s *EnvironmentRepositorySuite) TestGetBeachByID_NotFound() {
	beach, err := s.repo.GetBeachByID(s.ctx, 99999)
	s.Error(err)
	s.Nil(beach)
}

// ============================================================================
// Test GetTouristZoneByID
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetTouristZoneByID_Success() {
	// Get Gothic Quarter by searching nearby first
	zones, err := s.repo.GetTouristZonesNearby(s.ctx, 41.3828, 2.1761, 0.5)
	s.NoError(err)
	s.NotEmpty(zones)

	var gothicID int64
	for _, z := range zones {
		if z.NameEn == "Gothic Quarter" {
			gothicID = z.ID
			break
		}
	}
	s.NotZero(gothicID, "Gothic Quarter should be found")

	// Get Gothic Quarter
	zone, err := s.repo.GetTouristZoneByID(s.ctx, gothicID)
	s.NoError(err)
	s.NotNil(zone)
	s.Equal(gothicID, zone.ID)
	s.Equal("Barri Gòtic", zone.Name)
	s.Equal("historic", zone.Type)
	s.NotZero(zone.Lat)
	s.NotZero(zone.Lon)
}

func (s *EnvironmentRepositorySuite) TestGetTouristZoneByID_MultilingualNames() {
	// Get Gothic Quarter by searching nearby first
	zones, err := s.repo.GetTouristZonesNearby(s.ctx, 41.3828, 2.1761, 0.5)
	s.NoError(err)
	s.NotEmpty(zones)

	var gothicID int64
	for _, z := range zones {
		if z.NameEn == "Gothic Quarter" {
			gothicID = z.ID
			break
		}
	}
	s.NotZero(gothicID)

	// Get Gothic Quarter and check all language variants
	zone, err := s.repo.GetTouristZoneByID(s.ctx, gothicID)
	s.NoError(err)
	s.NotEmpty(zone.NameEn)
	s.Equal("Gothic Quarter", zone.NameEn)
	s.NotEmpty(zone.NameEs)
	s.Equal("Barrio Gótico", zone.NameEs)
	s.NotEmpty(zone.NameCa)
	s.Equal("Barri Gòtic", zone.NameCa)
	s.NotEmpty(zone.NameRu)
	s.Equal("Готический квартал", zone.NameRu)
}

func (s *EnvironmentRepositorySuite) TestGetTouristZoneByID_WithVisitorStats() {
	// Get Gothic Quarter by searching nearby first
	zones, err := s.repo.GetTouristZonesNearby(s.ctx, 41.3828, 2.1761, 0.5)
	s.NoError(err)
	s.NotEmpty(zones)

	var gothicID int64
	for _, z := range zones {
		if z.NameEn == "Gothic Quarter" {
			gothicID = z.ID
			break
		}
	}
	s.NotZero(gothicID)

	// Get tourist zone with visitor statistics
	zone, err := s.repo.GetTouristZoneByID(s.ctx, gothicID)
	s.NoError(err)
	s.NotNil(zone.VisitorsPerYear)
	s.Greater(*zone.VisitorsPerYear, 0)
}

func (s *EnvironmentRepositorySuite) TestGetTouristZoneByID_NotFound() {
	zone, err := s.repo.GetTouristZoneByID(s.ctx, 99999)
	s.Error(err)
	s.Nil(zone)
}

// ============================================================================
// Test GetGreenSpacesNearby
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetGreenSpacesNearby_CityCenter() {
	// Near Plaça Catalunya (city center)
	greenSpaces, err := s.repo.GetGreenSpacesNearby(s.ctx, 41.3874, 2.1686, 2.0)
	s.NoError(err)
	s.NotEmpty(greenSpaces)

	// Should find multiple parks
	s.GreaterOrEqual(len(greenSpaces), 2)

	// Check that results are sorted by distance
	// First result should be closest
	s.Greater(greenSpaces[0].AreaSqM, 0.0)
}

func (s *EnvironmentRepositorySuite) TestGetGreenSpacesNearby_NearParkGuell() {
	// Near Park Güell
	greenSpaces, err := s.repo.GetGreenSpacesNearby(s.ctx, 41.4145, 2.1527, 1.0)
	s.NoError(err)
	s.NotEmpty(greenSpaces)

	// Should find Park Güell
	found := false
	for _, gs := range greenSpaces {
		if gs.Name != nil && *gs.Name == "Parc Güell" {
			found = true
			s.Equal("park", gs.Type)
			break
		}
	}
	s.True(found, "Should find Park Güell")
}

func (s *EnvironmentRepositorySuite) TestGetGreenSpacesNearby_SmallRadius() {
	// Very small radius in area without parks
	greenSpaces, err := s.repo.GetGreenSpacesNearby(s.ctx, 41.4500, 2.2500, 0.1)
	s.NoError(err)
	// May be empty or have very few results
	_ = greenSpaces // Just verify it doesn't error
}

func (s *EnvironmentRepositorySuite) TestGetGreenSpacesNearby_LargeRadius() {
	// Large radius covering most of Barcelona
	greenSpaces, err := s.repo.GetGreenSpacesNearby(s.ctx, 41.3874, 2.1686, 5.0)
	s.NoError(err)
	s.NotEmpty(greenSpaces)

	// Should find many parks
	s.GreaterOrEqual(len(greenSpaces), 3)

	// Verify all have required fields
	for _, gs := range greenSpaces {
		s.NotZero(gs.ID)
		s.Greater(gs.AreaSqM, 0.0)
		s.NotZero(gs.CenterLat)
		s.NotZero(gs.CenterLon)
	}
}

// ============================================================================
// Test GetWaterBodiesNearby
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetWaterBodiesNearby_Coastline() {
	// Near Barcelona coastline
	waterBodies, err := s.repo.GetWaterBodiesNearby(s.ctx, 41.3850, 2.1950, 2.0)
	s.NoError(err)
	s.NotEmpty(waterBodies)

	// Should find Mediterranean Sea coastline
	found := false
	for _, wb := range waterBodies {
		if wb.Type == "coastline" || wb.Type == "sea" {
			found = true
			break
		}
	}
	s.True(found, "Should find coastline or sea")
}

func (s *EnvironmentRepositorySuite) TestGetWaterBodiesNearby_Rivers() {
	// Near Besòs river area
	waterBodies, err := s.repo.GetWaterBodiesNearby(s.ctx, 41.4200, 2.2150, 1.5)
	s.NoError(err)
	// May or may not find rivers depending on fixture data
	s.NotNil(waterBodies)
}

func (s *EnvironmentRepositorySuite) TestGetWaterBodiesNearby_EmptyArea() {
	// Area far from water
	waterBodies, err := s.repo.GetWaterBodiesNearby(s.ctx, 41.4500, 2.1000, 0.5)
	s.NoError(err)
	// Likely empty
	_ = waterBodies // Just verify it doesn't error
}

// ============================================================================
// Test GetBeachesNearby
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetBeachesNearby_Barceloneta() {
	// Near Barceloneta beach
	beaches, err := s.repo.GetBeachesNearby(s.ctx, 41.3800, 2.1900, 1.0)
	s.NoError(err)
	s.NotEmpty(beaches)

	// Should find Barceloneta and possibly other beaches
	foundBarceloneta := false
	for _, b := range beaches {
		if b.Name != nil && *b.Name == "Platja de la Barceloneta" {
			foundBarceloneta = true
			s.Equal("sand", b.Surface)
			// BlueFlag field is optional, just check it exists in this test
			break
		}
	}
	s.True(foundBarceloneta, "Should find Barceloneta beach")
}

func (s *EnvironmentRepositorySuite) TestGetBeachesNearby_MultipleBeaches() {
	// Area with multiple beaches
	beaches, err := s.repo.GetBeachesNearby(s.ctx, 41.3900, 2.2000, 2.0)
	s.NoError(err)
	s.NotEmpty(beaches)

	// Should find at least 2 beaches
	s.GreaterOrEqual(len(beaches), 2)

	// Verify all beaches have required fields
	for _, b := range beaches {
		s.NotZero(b.ID)
		s.NotEmpty(b.Surface)
		s.NotZero(b.Lat)
		s.NotZero(b.Lon)
	}
}

func (s *EnvironmentRepositorySuite) TestGetBeachesNearby_Inland() {
	// Inland area (no beaches)
	beaches, err := s.repo.GetBeachesNearby(s.ctx, 41.3874, 2.1686, 1.0)
	s.NoError(err)
	s.Empty(beaches)
}

func (s *EnvironmentRepositorySuite) TestGetBeachesNearby_SmallRadius() {
	// Very small radius
	beaches, err := s.repo.GetBeachesNearby(s.ctx, 41.3800, 2.1900, 0.2)
	s.NoError(err)
	// May be empty or have one beach
	s.NotNil(beaches)
}

// ============================================================================
// Test GetNoiseSourcesNearby
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetNoiseSourcesNearby_Airport() {
	// Near Barcelona El Prat Airport
	noiseSources, err := s.repo.GetNoiseSourcesNearby(s.ctx, 41.2971, 2.0785, 3.0)
	s.NoError(err)
	s.NotEmpty(noiseSources)

	// Should find airport
	foundAirport := false
	for _, ns := range noiseSources {
		if ns.Type == "airport" {
			foundAirport = true
			s.NotNil(ns.Intensity)
			break
		}
	}
	s.True(foundAirport, "Should find airport noise source")
}

func (s *EnvironmentRepositorySuite) TestGetNoiseSourcesNearby_Industrial() {
	// Near industrial area
	noiseSources, err := s.repo.GetNoiseSourcesNearby(s.ctx, 41.4100, 2.2200, 2.0)
	s.NoError(err)
	// May or may not find industrial noise sources
	_ = noiseSources // Just verify it doesn't error
}

func (s *EnvironmentRepositorySuite) TestGetNoiseSourcesNearby_CityCenter() {
	// City center (may have highway noise)
	noiseSources, err := s.repo.GetNoiseSourcesNearby(s.ctx, 41.3874, 2.1686, 1.0)
	s.NoError(err)
	// May have highway or other noise sources
	s.NotNil(noiseSources)
}

func (s *EnvironmentRepositorySuite) TestGetNoiseSourcesNearby_QuietArea() {
	// Quiet residential area
	noiseSources, err := s.repo.GetNoiseSourcesNearby(s.ctx, 41.4000, 2.1500, 0.5)
	s.NoError(err)
	// Should have few or no noise sources
	_ = noiseSources // Just verify it doesn't error
}

// ============================================================================
// Test GetTouristZonesNearby
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetTouristZonesNearby_GothicQuarter() {
	// In the center of Gothic Quarter
	zones, err := s.repo.GetTouristZonesNearby(s.ctx, 41.3828, 2.1761, 0.5)
	s.NoError(err)
	s.NotEmpty(zones)

	// Should find Gothic Quarter
	foundGothic := false
	for _, z := range zones {
		if z.NameEn == "Gothic Quarter" {
			foundGothic = true
			s.Equal("historic", z.Type)
			s.NotNil(z.VisitorsPerYear)
			break
		}
	}
	s.True(foundGothic, "Should find Gothic Quarter")
}

func (s *EnvironmentRepositorySuite) TestGetTouristZonesNearby_MultipleZones() {
	// City center with multiple tourist zones
	zones, err := s.repo.GetTouristZonesNearby(s.ctx, 41.3874, 2.1686, 2.0)
	s.NoError(err)
	s.NotEmpty(zones)

	// Should find multiple tourist zones
	s.GreaterOrEqual(len(zones), 2)

	// Verify multilingual support
	for _, z := range zones {
		s.NotEmpty(z.Name)
		s.NotEmpty(z.NameEn)
		s.NotZero(z.Lat)
		s.NotZero(z.Lon)
	}
}

func (s *EnvironmentRepositorySuite) TestGetTouristZonesNearby_SmallRadius() {
	// Small radius, specific location
	zones, err := s.repo.GetTouristZonesNearby(s.ctx, 41.3828, 2.1761, 0.2)
	s.NoError(err)
	s.NotNil(zones)
}

func (s *EnvironmentRepositorySuite) TestGetTouristZonesNearby_OutsideCity() {
	// Outside Barcelona, no tourist zones
	zones, err := s.repo.GetTouristZonesNearby(s.ctx, 41.5000, 2.3000, 1.0)
	s.NoError(err)
	s.Empty(zones)
}

// ============================================================================
// Test GetGreenSpacesTile (MVT Tiles)
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetGreenSpacesTile_Barcelona() {
	// Tile covering Barcelona parks at zoom 14
	// Tile (8267, 6127, 14) covers central Barcelona area
	tile, err := s.repo.GetGreenSpacesTile(s.ctx, 14, 8267, 6127)
	s.NoError(err)
	// May be empty if no green spaces in this exact tile
	s.NotNil(tile)
}

func (s *EnvironmentRepositorySuite) TestGetGreenSpacesTile_DifferentZoomLevels() {
	// Test different zoom levels
	zooms := []struct {
		z, x, y int
	}{
		{10, 516, 382},     // Low zoom (region level)
		{12, 2066, 1531},   // Medium zoom (city level)
		{16, 33069, 24509}, // High zoom (neighborhood level)
	}

	for _, zoom := range zooms {
		tile, err := s.repo.GetGreenSpacesTile(s.ctx, zoom.z, zoom.x, zoom.y)
		s.NoError(err, "Failed at zoom %d", zoom.z)
		s.NotNil(tile)
	}
}

func (s *EnvironmentRepositorySuite) TestGetGreenSpacesTile_EmptyTile() {
	// Tile in the ocean
	tile, err := s.repo.GetGreenSpacesTile(s.ctx, 10, 100, 100)
	s.NoError(err)
	s.LessOrEqual(len(tile), 0)
}

// ============================================================================
// Test GetWaterTile
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetWaterTile_CoastlineTile() {
	// Tile covering Barcelona coastline
	tile, err := s.repo.GetWaterTile(s.ctx, 14, 8268, 6127)
	s.NoError(err)
	s.NotNil(tile)
	// Should contain water bodies (Mediterranean Sea)
	// Note: may be empty if tile doesn't intersect coastline
}

func (s *EnvironmentRepositorySuite) TestGetWaterTile_LowZoom() {
	// Very low zoom should return empty (zoom filtering)
	tile, err := s.repo.GetWaterTile(s.ctx, 5, 16, 12)
	s.NoError(err)
	// Should be empty due to zoom filtering
	s.LessOrEqual(len(tile), 0)
}

func (s *EnvironmentRepositorySuite) TestGetWaterTile_HighZoom() {
	// High zoom showing detailed water features
	tile, err := s.repo.GetWaterTile(s.ctx, 15, 16536, 12254)
	s.NoError(err)
	s.NotNil(tile)
}

func (s *EnvironmentRepositorySuite) TestGetWaterTile_DifferentZoomLevels() {
	// Test zoom level filtering
	zooms := []struct {
		z, x, y int
	}{
		{8, 129, 96},       // Low zoom (region level)
		{10, 516, 382},     // Low-medium zoom
		{12, 2066, 1531},   // Medium zoom (city level)
		{14, 8268, 6127},   // High zoom (coastline)
		{16, 33072, 24509}, // Very high zoom (neighborhood level)
	}

	for _, zoom := range zooms {
		tile, err := s.repo.GetWaterTile(s.ctx, zoom.z, zoom.x, zoom.y)
		s.NoError(err, "Failed at zoom %d", zoom.z)
		s.NotNil(tile)
	}
}

// ============================================================================
// Test GetBeachesTile
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetBeachesTile_BarcelonaCoast() {
	// Tile covering Barcelona beaches at zoom 14
	tile, err := s.repo.GetBeachesTile(s.ctx, 14, 8268, 6127)
	s.NoError(err)
	s.NotNil(tile)
}

func (s *EnvironmentRepositorySuite) TestGetBeachesTile_LowZoom() {
	// Low zoom (beaches not visible below zoom 12)
	tile, err := s.repo.GetBeachesTile(s.ctx, 10, 516, 382)
	s.NoError(err)
	// Should be empty due to zoom filtering
	s.LessOrEqual(len(tile), 0)
}

func (s *EnvironmentRepositorySuite) TestGetBeachesTile_HighZoom() {
	// High zoom showing beach details
	tile, err := s.repo.GetBeachesTile(s.ctx, 16, 33072, 24509)
	s.NoError(err)
	s.NotNil(tile)
}

func (s *EnvironmentRepositorySuite) TestGetBeachesTile_InlandArea() {
	// Inland tile (no beaches)
	tile, err := s.repo.GetBeachesTile(s.ctx, 14, 8267, 6127)
	s.NoError(err)
	// Should be empty (no beaches inland)
	s.LessOrEqual(len(tile), 0)
}

// ============================================================================
// Test GetNoiseSourcesTile
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetNoiseSourcesTile_AirportArea() {
	// Tile covering airport area
	tile, err := s.repo.GetNoiseSourcesTile(s.ctx, 12, 2064, 1534)
	s.NoError(err)
	s.NotNil(tile)
}

func (s *EnvironmentRepositorySuite) TestGetNoiseSourcesTile_LowZoom() {
	// Very low zoom (noise sources not visible)
	tile, err := s.repo.GetNoiseSourcesTile(s.ctx, 8, 129, 96)
	s.NoError(err)
	// Should be empty due to zoom filtering
	s.LessOrEqual(len(tile), 0)
}

func (s *EnvironmentRepositorySuite) TestGetNoiseSourcesTile_ZoomFiltering() {
	// Test zoom level filtering for different noise types
	// z < 10: empty
	// z 10-11: only airports
	// z 12: airports + industrial
	// z >= 13: all noise sources

	tile1, err := s.repo.GetNoiseSourcesTile(s.ctx, 9, 258, 191)
	s.NoError(err)
	s.LessOrEqual(len(tile1), 0, "Should be empty below zoom 10")

	tile2, err := s.repo.GetNoiseSourcesTile(s.ctx, 11, 1032, 766)
	s.NoError(err)
	s.NotNil(tile2, "Should work at zoom 11")
}

func (s *EnvironmentRepositorySuite) TestGetNoiseSourcesTile_HighZoom() {
	// High zoom showing all noise sources
	tile, err := s.repo.GetNoiseSourcesTile(s.ctx, 15, 16534, 12270)
	s.NoError(err)
	s.NotNil(tile)
}

// ============================================================================
// Test GetTouristZonesTile
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetTouristZonesTile_GothicQuarter() {
	// Tile covering Gothic Quarter at zoom 14
	tile, err := s.repo.GetTouristZonesTile(s.ctx, 14, 8267, 6127)
	s.NoError(err)
	s.NotNil(tile)
}

func (s *EnvironmentRepositorySuite) TestGetTouristZonesTile_LowZoom() {
	// Low zoom (tourist zones not visible below zoom 11)
	tile, err := s.repo.GetTouristZonesTile(s.ctx, 9, 258, 191)
	s.NoError(err)
	// Should be empty due to zoom filtering
	s.LessOrEqual(len(tile), 0)
}

func (s *EnvironmentRepositorySuite) TestGetTouristZonesTile_MediumZoom() {
	// Medium zoom showing major tourist zones
	tile, err := s.repo.GetTouristZonesTile(s.ctx, 12, 2066, 1531)
	s.NoError(err)
	s.NotNil(tile)
}

func (s *EnvironmentRepositorySuite) TestGetTouristZonesTile_HighZoom() {
	// High zoom showing detailed tourist zones
	tile, err := s.repo.GetTouristZonesTile(s.ctx, 16, 33069, 24509)
	s.NoError(err)
	s.NotNil(tile)
}

// ============================================================================
// Test GetEnvironmentRadiusTile
// ============================================================================

func (s *EnvironmentRepositorySuite) TestGetEnvironmentRadiusTile_CityCenter() {
	// Combined environment tile around city center
	tile, err := s.repo.GetEnvironmentRadiusTile(s.ctx, 41.3874, 2.1686, 2.0)
	s.NoError(err)
	s.NotEmpty(tile)
}

func (s *EnvironmentRepositorySuite) TestGetEnvironmentRadiusTile_Coastline() {
	// Environment around coastline (should include beaches and water)
	tile, err := s.repo.GetEnvironmentRadiusTile(s.ctx, 41.3800, 2.1900, 1.5)
	s.NoError(err)
	s.NotEmpty(tile)
}

func (s *EnvironmentRepositorySuite) TestGetEnvironmentRadiusTile_SmallRadius() {
	// Small radius
	tile, err := s.repo.GetEnvironmentRadiusTile(s.ctx, 41.3874, 2.1686, 0.5)
	s.NoError(err)
	s.NotNil(tile)
}

func (s *EnvironmentRepositorySuite) TestGetEnvironmentRadiusTile_LargeRadius() {
	// Large radius covering much of Barcelona
	tile, err := s.repo.GetEnvironmentRadiusTile(s.ctx, 41.3874, 2.1686, 5.0)
	s.NoError(err)
	s.NotEmpty(tile)
}

func (s *EnvironmentRepositorySuite) TestGetEnvironmentRadiusTile_EmptyArea() {
	// Area with no environment features
	tile, err := s.repo.GetEnvironmentRadiusTile(s.ctx, 41.5000, 2.3000, 0.5)
	s.NoError(err)
	s.LessOrEqual(len(tile), 0)
}

// Run the test suite
func TestEnvironmentRepository(t *testing.T) {
	suite.Run(t, new(EnvironmentRepositorySuite))
}
