package postgresosm

import (
	"context"
	"testing"

	pkgerrors "github.com/location-microservice/internal/pkg/errors"
)

func TestEnvironmentRepository_GetGreenSpacesNearby(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get green spaces nearby", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734 // Barcelona
		radiusKm := 5.0

		spaces, err := repo.GetGreenSpacesNearby(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get green spaces: %v", err)
		}

		if len(spaces) > LimitGreenSpaces {
			t.Errorf("Expected at most %d green spaces, got %d", LimitGreenSpaces, len(spaces))
		}

		for _, space := range spaces {
			if space.OSMId == 0 {
				t.Error("Expected non-zero OSM ID")
			}
			if space.ID != space.OSMId {
				t.Error("Expected ID to equal OSM ID")
			}
			if space.Type == "" {
				t.Error("Expected non-empty type")
			}
			if space.AreaSqM <= 0 {
				t.Errorf("Expected positive area, got %f", space.AreaSqM)
			}
			assertValidCoordinates(t, space.CenterLat, space.CenterLon)
		}
	})

	t.Run("Get green spaces in small radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		radiusKm := 1.0

		spaces, err := repo.GetGreenSpacesNearby(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get green spaces: %v", err)
		}

		// Small radius might have fewer or no spaces
		_ = spaces
	})
}

func TestEnvironmentRepository_GetWaterBodiesNearby(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get water bodies nearby", func(t *testing.T) {
		// Coastal city coordinates
		lat, lon := 41.3851, 2.1734 // Barcelona
		radiusKm := 10.0

		waterBodies, err := repo.GetWaterBodiesNearby(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get water bodies: %v", err)
		}

		if len(waterBodies) > LimitWaterBodies {
			t.Errorf("Expected at most %d water bodies, got %d", LimitWaterBodies, len(waterBodies))
		}

		for _, water := range waterBodies {
			if water.OSMId == 0 {
				t.Error("Expected non-zero OSM ID")
			}
			if water.ID != water.OSMId {
				t.Error("Expected ID to equal OSM ID")
			}
			if water.Type == "" {
				t.Error("Expected non-empty type")
			}
		}
	})

	t.Run("Get water bodies in inland area", func(t *testing.T) {
		// Inland coordinates might have fewer water bodies
		lat, lon := 41.6488, -0.8891 // Zaragoza
		radiusKm := 5.0

		waterBodies, err := repo.GetWaterBodiesNearby(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get water bodies: %v", err)
		}

		// Inland area might have no or few water bodies
		_ = waterBodies
	})
}

func TestEnvironmentRepository_GetBeachesNearby(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get beaches nearby coastal city", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734 // Barcelona
		radiusKm := 10.0

		beaches, err := repo.GetBeachesNearby(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get beaches: %v", err)
		}

		if len(beaches) > LimitBeaches {
			t.Errorf("Expected at most %d beaches, got %d", LimitBeaches, len(beaches))
		}

		for _, beach := range beaches {
			if beach.OSMId == 0 {
				t.Error("Expected non-zero OSM ID")
			}
			if beach.ID != beach.OSMId {
				t.Error("Expected ID to equal OSM ID")
			}
			assertValidCoordinates(t, beach.Lat, beach.Lon)

			if beach.BlueFlag == nil {
				t.Error("Expected BlueFlag to be initialized")
			}
		}
	})

	t.Run("Get beaches in inland area", func(t *testing.T) {
		// Inland coordinates should have no beaches
		lat, lon := 41.6488, -0.8891 // Zaragoza
		radiusKm := 5.0

		beaches, err := repo.GetBeachesNearby(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get beaches: %v", err)
		}

		// Inland area should have no beaches
		if len(beaches) > 0 {
			t.Logf("Found %d beaches in inland area (unexpected but not an error)", len(beaches))
		}
	})
}

func TestEnvironmentRepository_GetNoiseSourcesNearby(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get noise sources nearby", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734 // Barcelona
		radiusKm := 10.0

		noiseSources, err := repo.GetNoiseSourcesNearby(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get noise sources: %v", err)
		}

		if len(noiseSources) > LimitNoiseSources {
			t.Errorf("Expected at most %d noise sources, got %d", LimitNoiseSources, len(noiseSources))
		}

		validTypes := map[string]bool{
			"airport":    true,
			"industrial": true,
			"highway":    true,
			"railway":    true,
			"other":      true,
		}

		for _, noise := range noiseSources {
			if noise.OSMId == 0 {
				t.Error("Expected non-zero OSM ID")
			}
			if noise.ID != noise.OSMId {
				t.Error("Expected ID to equal OSM ID")
			}
			if !validTypes[noise.Type] {
				t.Errorf("Invalid noise source type: %s", noise.Type)
			}
			assertValidCoordinates(t, noise.Lat, noise.Lon)
		}
	})

	t.Run("Get noise sources in small radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		radiusKm := 2.0

		noiseSources, err := repo.GetNoiseSourcesNearby(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get noise sources: %v", err)
		}

		// Small radius might have fewer noise sources
		_ = noiseSources
	})
}

func TestEnvironmentRepository_GetTouristZonesNearby(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get tourist zones nearby", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734 // Barcelona
		radiusKm := 5.0

		zones, err := repo.GetTouristZonesNearby(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get tourist zones: %v", err)
		}

		if len(zones) > LimitTouristZones {
			t.Errorf("Expected at most %d tourist zones, got %d", LimitTouristZones, len(zones))
		}

		for _, zone := range zones {
			if zone.OSMId == 0 {
				t.Error("Expected non-zero OSM ID")
			}
			if zone.ID != zone.OSMId {
				t.Error("Expected ID to equal OSM ID")
			}
			if zone.Type == "" {
				t.Error("Expected non-empty type")
			}
			assertValidCoordinates(t, zone.Lat, zone.Lon)
		}
	})

	t.Run("Get tourist zones in small radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		radiusKm := 1.0

		zones, err := repo.GetTouristZonesNearby(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get tourist zones: %v", err)
		}

		// Small radius might have fewer zones
		_ = zones
	})
}

func TestEnvironmentRepository_GetGreenSpaceByID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get existing green space by ID", func(t *testing.T) {
		// Find a green space
		var osmID int64
		query := `SELECT osm_id FROM planet_osm_polygon 
				  WHERE (leisure IN ('park', 'garden', 'nature_reserve') 
				     OR landuse IN ('forest', 'meadow', 'grass'))
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&osmID)
		if err != nil {
			t.Skipf("No green spaces found in database: %v", err)
		}

		space, err := repo.GetGreenSpaceByID(ctx, osmID)
		if err != nil {
			t.Fatalf("Failed to get green space by ID: %v", err)
		}

		if space == nil {
			t.Fatal("Expected green space, got nil")
		}

		if space.OSMId != osmID {
			t.Errorf("Expected OSM ID %d, got %d", osmID, space.OSMId)
		}

		if space.AreaSqM <= 0 {
			t.Errorf("Expected positive area, got %f", space.AreaSqM)
		}

		assertValidCoordinates(t, space.CenterLat, space.CenterLon)
	})

	t.Run("Get non-existing green space", func(t *testing.T) {
		_, err := repo.GetGreenSpaceByID(ctx, -99999999)
		if err != pkgerrors.ErrLocationNotFound {
			t.Errorf("Expected ErrLocationNotFound, got %v", err)
		}
	})
}

func TestEnvironmentRepository_GetBeachByID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get existing beach by ID", func(t *testing.T) {
		// Find a beach
		var osmID int64
		query := `SELECT osm_id FROM planet_osm_polygon 
				  WHERE "natural" = 'beach' 
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&osmID)
		if err != nil {
			t.Skipf("No beaches found in database: %v", err)
		}

		beach, err := repo.GetBeachByID(ctx, osmID)
		if err != nil {
			t.Fatalf("Failed to get beach by ID: %v", err)
		}

		if beach == nil {
			t.Fatal("Expected beach, got nil")
		}

		if beach.OSMId != osmID {
			t.Errorf("Expected OSM ID %d, got %d", osmID, beach.OSMId)
		}

		assertValidCoordinates(t, beach.Lat, beach.Lon)

		if beach.BlueFlag == nil {
			t.Error("Expected BlueFlag to be initialized")
		}
	})

	t.Run("Get non-existing beach", func(t *testing.T) {
		_, err := repo.GetBeachByID(ctx, -99999999)
		if err != pkgerrors.ErrLocationNotFound {
			t.Errorf("Expected ErrLocationNotFound, got %v", err)
		}
	})
}

func TestEnvironmentRepository_GetTouristZoneByID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get existing tourist zone by ID", func(t *testing.T) {
		// Find a tourist zone
		var osmID int64
		query := `SELECT osm_id FROM planet_osm_polygon 
				  WHERE tourism IN ('attraction', 'museum', 'theme_park', 'zoo', 'aquarium', 'viewpoint')
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&osmID)
		if err != nil {
			t.Skipf("No tourist zones found in database: %v", err)
		}

		zone, err := repo.GetTouristZoneByID(ctx, osmID)
		if err != nil {
			t.Fatalf("Failed to get tourist zone by ID: %v", err)
		}

		if zone == nil {
			t.Fatal("Expected tourist zone, got nil")
		}

		if zone.OSMId != osmID {
			t.Errorf("Expected OSM ID %d, got %d", osmID, zone.OSMId)
		}

		if zone.Type == "" {
			t.Error("Expected non-empty type")
		}

		assertValidCoordinates(t, zone.Lat, zone.Lon)
	})

	t.Run("Get non-existing tourist zone", func(t *testing.T) {
		_, err := repo.GetTouristZoneByID(ctx, -99999999)
		if err != pkgerrors.ErrLocationNotFound {
			t.Errorf("Expected ErrLocationNotFound, got %v", err)
		}
	})
}

func TestEnvironmentRepository_GetGreenSpacesTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get green spaces tile", func(t *testing.T) {
		// Barcelona area tile
		z, x, y := 14, 8311, 6143

		tile, err := repo.GetGreenSpacesTile(ctx, z, x, y)
		if err != nil {
			t.Fatalf("Failed to get green spaces tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get green spaces tile at different zoom levels", func(t *testing.T) {
		x, y := 8311, 6143

		for _, z := range []int{10, 12, 14, 16} {
			tile, err := repo.GetGreenSpacesTile(ctx, z, x>>4, y>>4)
			if err != nil {
				t.Errorf("Failed to get tile at zoom %d: %v", z, err)
			}
			if tile == nil {
				t.Errorf("Expected non-nil tile at zoom %d", z)
			}
		}
	})
}

func TestEnvironmentRepository_GetWaterTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get water tile", func(t *testing.T) {
		// Barcelona area tile
		z, x, y := 14, 8311, 6143

		tile, err := repo.GetWaterTile(ctx, z, x, y)
		if err != nil {
			t.Fatalf("Failed to get water tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})
}

func TestEnvironmentRepository_GetBeachesTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get beaches tile at high zoom", func(t *testing.T) {
		// Beaches are visible at zoom >= 12
		z, x, y := 14, 8311, 6143

		tile, err := repo.GetBeachesTile(ctx, z, x, y)
		if err != nil {
			t.Fatalf("Failed to get beaches tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get beaches tile at low zoom returns empty", func(t *testing.T) {
		// Beaches not visible at zoom < 12
		z, x, y := 10, 512, 384

		tile, err := repo.GetBeachesTile(ctx, z, x, y)
		if err != nil {
			t.Fatalf("Failed to get beaches tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
		if len(tile) != 0 {
			t.Error("Expected empty tile at low zoom")
		}
	})
}

func TestEnvironmentRepository_GetNoiseSourcesTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get noise sources tile at high zoom", func(t *testing.T) {
		// Noise sources visible at zoom >= 10
		z, x, y := 12, 2077, 1535

		tile, err := repo.GetNoiseSourcesTile(ctx, z, x, y)
		if err != nil {
			t.Fatalf("Failed to get noise sources tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get noise sources tile at low zoom returns empty", func(t *testing.T) {
		// Noise sources not visible at zoom < 10
		z, x, y := 8, 128, 96

		tile, err := repo.GetNoiseSourcesTile(ctx, z, x, y)
		if err != nil {
			t.Fatalf("Failed to get noise sources tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
		if len(tile) != 0 {
			t.Error("Expected empty tile at low zoom")
		}
	})
}

func TestEnvironmentRepository_GetTouristZonesTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get tourist zones tile at high zoom", func(t *testing.T) {
		// Tourist zones visible at zoom >= 11
		z, x, y := 14, 8311, 6143

		tile, err := repo.GetTouristZonesTile(ctx, z, x, y)
		if err != nil {
			t.Fatalf("Failed to get tourist zones tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get tourist zones tile at low zoom returns empty", func(t *testing.T) {
		// Tourist zones not visible at zoom < 11
		z, x, y := 9, 256, 192

		tile, err := repo.GetTouristZonesTile(ctx, z, x, y)
		if err != nil {
			t.Fatalf("Failed to get tourist zones tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
		if len(tile) != 0 {
			t.Error("Expected empty tile at low zoom")
		}
	})
}

func TestEnvironmentRepository_GetEnvironmentRadiusTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewEnvironmentRepository(db)
	ctx := context.Background()

	t.Run("Get environment radius tile", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734 // Barcelona
		radiusKm := 5.0

		tile, err := repo.GetEnvironmentRadiusTile(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get environment radius tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get environment radius tile with small radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		radiusKm := 1.0

		tile, err := repo.GetEnvironmentRadiusTile(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get environment radius tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get environment radius tile with large radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		radiusKm := 20.0

		tile, err := repo.GetEnvironmentRadiusTile(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get environment radius tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})
}
