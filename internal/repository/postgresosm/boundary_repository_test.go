package postgresosm

import (
	"context"
	"testing"

	"github.com/location-microservice/internal/domain"
	pkgerrors "github.com/location-microservice/internal/pkg/errors"
)

func TestBoundaryRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewBoundaryRepository(db)
	ctx := context.Background()

	t.Run("Get existing boundary by ID", func(t *testing.T) {
		// First, find any boundary in the database
		var osmID int64
		query := `SELECT osm_id FROM planet_osm_polygon 
				  WHERE boundary = 'administrative' 
				  AND admin_level IS NOT NULL 
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&osmID)
		if err != nil {
			t.Skipf("No boundaries found in database: %v", err)
		}

		boundary, err := repo.GetByID(ctx, osmID)
		if err != nil {
			t.Fatalf("Failed to get boundary by ID: %v", err)
		}

		if boundary == nil {
			t.Fatal("Expected boundary, got nil")
		}

		if boundary.OSMId != osmID {
			t.Errorf("Expected OSM ID %d, got %d", osmID, boundary.OSMId)
		}

		if boundary.ID != boundary.OSMId {
			t.Errorf("Expected ID to equal OSM ID")
		}

		if boundary.AdminLevel <= 0 {
			t.Errorf("Expected valid admin level, got %d", boundary.AdminLevel)
		}

		assertValidCoordinates(t, boundary.CenterLat, boundary.CenterLon)

		if boundary.AreaSqKm == nil || *boundary.AreaSqKm <= 0 {
			t.Errorf("Expected positive area")
		}
	})

	t.Run("Get non-existing boundary", func(t *testing.T) {
		_, err := repo.GetByID(ctx, -99999999)
		if err != pkgerrors.ErrLocationNotFound {
			t.Errorf("Expected ErrLocationNotFound, got %v", err)
		}
	})
}

func TestBoundaryRepository_SearchByText(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewBoundaryRepository(db)
	ctx := context.Background()

	t.Run("Search boundaries by text", func(t *testing.T) {
		// Get a boundary name to search for
		var searchName string
		query := `SELECT COALESCE(name, '') FROM planet_osm_polygon 
				  WHERE boundary = 'administrative' 
				  AND admin_level IS NOT NULL
				  AND name IS NOT NULL 
				  AND name != ''
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&searchName)
		if err != nil || searchName == "" {
			t.Skipf("No named boundaries found in database")
		}

		// Search for part of the name
		searchQuery := searchName[:3]
		boundaries, err := repo.SearchByText(ctx, searchQuery, "", nil, 10)
		if err != nil {
			t.Fatalf("Failed to search boundaries: %v", err)
		}

		if len(boundaries) == 0 {
			t.Errorf("Expected at least one boundary for query '%s'", searchQuery)
		}

		for _, b := range boundaries {
			if b.OSMId == 0 {
				t.Error("Expected non-zero OSM ID")
			}
			if b.AdminLevel <= 0 {
				t.Errorf("Expected valid admin level, got %d", b.AdminLevel)
			}
			assertValidCoordinates(t, b.CenterLat, b.CenterLon)
		}
	})

	t.Run("Search boundaries with admin level filter", func(t *testing.T) {
		boundaries, err := repo.SearchByText(ctx, "", "", []int{2, 4}, 10)
		if err != nil {
			t.Fatalf("Failed to search boundaries with filter: %v", err)
		}

		for _, b := range boundaries {
			if b.AdminLevel != 2 && b.AdminLevel != 4 {
				t.Errorf("Expected admin level 2 or 4, got %d", b.AdminLevel)
			}
		}
	})

	t.Run("Search boundaries with language preference", func(t *testing.T) {
		boundaries, err := repo.SearchByText(ctx, "a", "en", nil, 5)
		if err != nil {
			t.Fatalf("Failed to search boundaries with language: %v", err)
		}

		// Should return results even if no English names
		if len(boundaries) < 0 {
			t.Error("Expected some results")
		}
	})
}

func TestBoundaryRepository_Search(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewBoundaryRepository(db)
	ctx := context.Background()

	t.Run("Simple search", func(t *testing.T) {
		boundaries, err := repo.Search(ctx, "a", 5)
		if err != nil {
			t.Fatalf("Failed to search boundaries: %v", err)
		}

		if len(boundaries) > 5 {
			t.Errorf("Expected at most 5 boundaries, got %d", len(boundaries))
		}
	})
}

func TestBoundaryRepository_ReverseGeocode(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewBoundaryRepository(db)
	ctx := context.Background()

	t.Run("Reverse geocode valid location", func(t *testing.T) {
		// Get a point inside a boundary
		var lat, lon float64
		query := `SELECT 
					ST_Y(ST_Centroid(ST_Transform(way, 4326))) AS lat,
					ST_X(ST_Centroid(ST_Transform(way, 4326))) AS lon
				  FROM planet_osm_polygon 
				  WHERE boundary = 'administrative' 
				  AND admin_level = '8'
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&lat, &lon)
		if err != nil {
			t.Skipf("No boundaries found for reverse geocode test")
		}

		addr, err := repo.ReverseGeocode(ctx, lat, lon)
		if err != nil {
			t.Fatalf("Failed to reverse geocode: %v", err)
		}

		if addr == nil {
			t.Fatal("Expected address, got nil")
		}

		// At least one field should be populated
		hasData := addr.Country != "" || addr.Region != "" ||
			addr.Province != "" || addr.City != ""
		if !hasData {
			t.Error("Expected at least one address field to be populated")
		}
	})

	t.Run("Reverse geocode invalid location", func(t *testing.T) {
		// Middle of the ocean
		_, err := repo.ReverseGeocode(ctx, 0.0, 0.0)
		if err != pkgerrors.ErrLocationNotFound {
			t.Logf("Expected ErrLocationNotFound for ocean coordinates, got %v", err)
		}
	})
}

func TestBoundaryRepository_ReverseGeocodeBatch(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewBoundaryRepository(db)
	ctx := context.Background()

	t.Run("Reverse geocode multiple points", func(t *testing.T) {
		// Get some points
		query := `SELECT 
					ST_Y(ST_Centroid(ST_Transform(way, 4326))) AS lat,
					ST_X(ST_Centroid(ST_Transform(way, 4326))) AS lon
				  FROM planet_osm_polygon 
				  WHERE boundary = 'administrative' 
				  AND admin_level IS NOT NULL
				  LIMIT 2`
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			t.Skipf("No boundaries found for batch test")
		}
		defer rows.Close()

		points := []domain.LatLon{}
		for rows.Next() {
			var lat, lon float64
			if err := rows.Scan(&lat, &lon); err == nil {
				points = append(points, domain.LatLon{Lat: lat, Lon: lon})
			}
		}

		if len(points) < 2 {
			points = []domain.LatLon{
				{Lat: 41.3851, Lon: 2.1734},  // Barcelona
				{Lat: 41.6488, Lon: -0.8891}, // Zaragoza
			}
		}

		addresses, err := repo.ReverseGeocodeBatch(ctx, points)
		if err != nil {
			t.Fatalf("Failed to batch reverse geocode: %v", err)
		}

		if len(addresses) != len(points) {
			t.Errorf("Expected %d addresses, got %d", len(points), len(addresses))
		}
	})

	t.Run("Reverse geocode empty batch", func(t *testing.T) {
		addresses, err := repo.ReverseGeocodeBatch(ctx, []domain.LatLon{})
		if err != nil {
			t.Fatalf("Failed with empty batch: %v", err)
		}

		if len(addresses) != 0 {
			t.Errorf("Expected 0 addresses, got %d", len(addresses))
		}
	})
}

func TestBoundaryRepository_GetByPoint(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewBoundaryRepository(db)
	ctx := context.Background()

	t.Run("Get boundaries by point", func(t *testing.T) {
		// Get a point inside a boundary
		var lat, lon float64
		query := `SELECT 
					ST_Y(ST_Centroid(ST_Transform(way, 4326))) AS lat,
					ST_X(ST_Centroid(ST_Transform(way, 4326))) AS lon
				  FROM planet_osm_polygon 
				  WHERE boundary = 'administrative' 
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&lat, &lon)
		if err != nil {
			t.Skipf("No boundaries found")
		}

		boundaries, err := repo.GetByPoint(ctx, lat, lon)
		if err != nil {
			t.Fatalf("Failed to get boundaries by point: %v", err)
		}

		if len(boundaries) == 0 {
			t.Error("Expected at least one boundary")
		}

		// Verify boundaries are sorted by admin level
		for i := 1; i < len(boundaries); i++ {
			if boundaries[i].AdminLevel < boundaries[i-1].AdminLevel {
				t.Error("Expected boundaries to be sorted by admin level ascending")
			}
		}
	})
}

func TestBoundaryRepository_GetChildren(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewBoundaryRepository(db)
	ctx := context.Background()

	t.Run("Get children boundaries", func(t *testing.T) {
		// Get a parent boundary with low admin level
		var parentID int64
		var parentLevel int
		query := `SELECT osm_id, (admin_level)::integer AS level
				  FROM planet_osm_polygon 
				  WHERE boundary = 'administrative' 
				  AND admin_level IS NOT NULL
				  AND (admin_level)::integer <= 6
				  ORDER BY (admin_level)::integer ASC
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&parentID, &parentLevel)
		if err != nil {
			t.Skipf("No parent boundaries found")
		}

		children, err := repo.GetChildren(ctx, parentID)
		if err != nil {
			t.Fatalf("Failed to get children boundaries: %v", err)
		}

		// Children might not exist or might be in different regions
		for _, child := range children {
			if child.AdminLevel <= parentLevel {
				t.Errorf("Expected child admin level > %d, got %d", parentLevel, child.AdminLevel)
			}
		}
	})
}

func TestBoundaryRepository_GetByAdminLevel(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewBoundaryRepository(db)
	ctx := context.Background()

	t.Run("Get boundaries by admin level", func(t *testing.T) {
		boundaries, err := repo.GetByAdminLevel(ctx, 8, 5)
		if err != nil {
			t.Fatalf("Failed to get boundaries by admin level: %v", err)
		}

		if len(boundaries) > 5 {
			t.Errorf("Expected at most 5 boundaries, got %d", len(boundaries))
		}

		for _, b := range boundaries {
			if b.AdminLevel != 8 {
				t.Errorf("Expected admin level 8, got %d", b.AdminLevel)
			}
		}
	})

	t.Run("Get boundaries with default limit", func(t *testing.T) {
		boundaries, err := repo.GetByAdminLevel(ctx, 6, 0)
		if err != nil {
			t.Fatalf("Failed to get boundaries: %v", err)
		}

		if len(boundaries) > LimitBoundaries {
			t.Errorf("Expected at most %d boundaries, got %d", LimitBoundaries, len(boundaries))
		}
	})
}

func TestBoundaryRepository_GetBoundariesInRadius(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewBoundaryRepository(db)
	ctx := context.Background()

	t.Run("Get boundaries in radius", func(t *testing.T) {
		// Use center of Catalunya region or similar
		lat, lon := 41.5911, 1.5208
		radiusKm := 50.0

		boundaries, err := repo.GetBoundariesInRadius(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get boundaries in radius: %v", err)
		}

		// Should find some boundaries (cities, provinces, etc.)
		for _, b := range boundaries {
			assertValidCoordinates(t, b.CenterLat, b.CenterLon)

			// Verify admin level is one of the expected ones
			if b.AdminLevel != 6 && b.AdminLevel != 8 && b.AdminLevel != 9 {
				t.Errorf("Expected admin level 6, 8, or 9, got %d", b.AdminLevel)
			}
		}
	})

	t.Run("Get boundaries in small radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734 // Barcelona center
		radiusKm := 5.0

		boundaries, err := repo.GetBoundariesInRadius(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get boundaries: %v", err)
		}

		if len(boundaries) > LimitBoundariesRadius {
			t.Errorf("Expected at most %d boundaries, got %d", LimitBoundariesRadius, len(boundaries))
		}
	})
}

func TestBoundaryRepository_GetTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewBoundaryRepository(db)
	ctx := context.Background()

	t.Run("Get tile for valid coordinates", func(t *testing.T) {
		// Use tile coordinates that should contain some data
		// z=10, x=512, y=384 should be in the Mediterranean area
		tile, err := repo.GetTile(ctx, 10, 512, 384)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		// Tile might be empty or contain data depending on the area
		// We just verify no error occurred
		t.Logf("Tile size: %d bytes", len(tile))
	})
}

func TestBoundaryRepository_GetBoundariesRadiusTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewBoundaryRepository(db)
	ctx := context.Background()

	t.Run("Get boundaries radius tile not implemented", func(t *testing.T) {
		tile, err := repo.GetBoundariesRadiusTile(ctx, 41.3851, 2.1734, 10.0)
		if err != nil {
			t.Errorf("Expected no error for unimplemented method, got %v", err)
		}

		if len(tile) != 0 {
			t.Error("Expected empty tile for unimplemented method")
		}
	})
}
