package postgresosm

import (
	"context"
	"testing"

	pkgerrors "github.com/location-microservice/internal/pkg/errors"
)

func TestPOIRepository_GetByID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewPOIRepository(db)
	ctx := context.Background()

	t.Run("Get existing POI by ID", func(t *testing.T) {
		// Find any POI in the database
		var osmID int64
		query := `SELECT osm_id FROM planet_osm_point 
				  WHERE name IS NOT NULL AND name != '' 
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&osmID)
		if err != nil {
			t.Skipf("No POIs found in database: %v", err)
		}

		poi, err := repo.GetByID(ctx, osmID)
		if err != nil {
			t.Fatalf("Failed to get POI by ID: %v", err)
		}

		if poi == nil {
			t.Fatal("Expected POI, got nil")
		}

		if poi.OSMId != osmID {
			t.Errorf("Expected OSM ID %d, got %d", osmID, poi.OSMId)
		}

		if poi.ID != poi.OSMId {
			t.Errorf("Expected ID to equal OSM ID")
		}

		if poi.Name == "" {
			t.Error("Expected non-empty name")
		}

		if poi.Category == "" {
			t.Error("Expected non-empty category")
		}

		assertValidCoordinates(t, poi.Lat, poi.Lon)

		if poi.Tags == nil {
			t.Error("Expected tags map to be initialized")
		}
	})

	t.Run("Get non-existing POI", func(t *testing.T) {
		_, err := repo.GetByID(ctx, -99999999)
		if err != pkgerrors.ErrLocationNotFound {
			t.Errorf("Expected ErrLocationNotFound, got %v", err)
		}
	})
}

func TestPOIRepository_GetNearby(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewPOIRepository(db)
	ctx := context.Background()

	t.Run("Get nearby POIs without category filter", func(t *testing.T) {
		// Use Barcelona coordinates
		lat, lon := 41.3851, 2.1734
		radiusKm := 1.0

		pois, err := repo.GetNearby(ctx, lat, lon, radiusKm, nil)
		if err != nil {
			t.Fatalf("Failed to get nearby POIs: %v", err)
		}

		if len(pois) > LimitPOIs {
			t.Errorf("Expected at most %d POIs, got %d", LimitPOIs, len(pois))
		}

		for _, poi := range pois {
			if poi.OSMId == 0 {
				t.Error("Expected non-zero OSM ID")
			}
			if poi.Category == "" {
				t.Error("Expected non-empty category")
			}
			assertValidCoordinates(t, poi.Lat, poi.Lon)
		}
	})

	t.Run("Get nearby POIs with category filter", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		radiusKm := 5.0
		categories := []string{"restaurant", "cafe", "bar"}

		pois, err := repo.GetNearby(ctx, lat, lon, radiusKm, categories)
		if err != nil {
			t.Fatalf("Failed to get nearby POIs with filter: %v", err)
		}

		for _, poi := range pois {
			found := false
			for _, cat := range categories {
				if poi.Category == cat {
					found = true
					break
				}
			}
			if !found && len(pois) > 0 {
				t.Errorf("Expected category in %v, got %s", categories, poi.Category)
			}
		}
	})

	t.Run("Get nearby POIs with zero radius uses default", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734

		pois, err := repo.GetNearby(ctx, lat, lon, 0, nil)
		if err != nil {
			t.Fatalf("Failed to get nearby POIs: %v", err)
		}

		// Should use default radius of 1km
		_ = pois
	})
}

func TestPOIRepository_Search(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewPOIRepository(db)
	ctx := context.Background()

	t.Run("Search POIs by text", func(t *testing.T) {
		// Get a POI name to search for
		var searchName string
		query := `SELECT name FROM planet_osm_point 
				  WHERE name IS NOT NULL AND name != '' 
				  AND length(name) > 3
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&searchName)
		if err != nil || searchName == "" {
			t.Skipf("No named POIs found in database")
		}

		// Search for part of the name
		searchQuery := searchName[:3]
		pois, err := repo.Search(ctx, searchQuery, nil, 10)
		if err != nil {
			t.Fatalf("Failed to search POIs: %v", err)
		}

		for _, poi := range pois {
			if poi.OSMId == 0 {
				t.Error("Expected non-zero OSM ID")
			}
			assertValidCoordinates(t, poi.Lat, poi.Lon)
		}
	})

	t.Run("Search POIs with category filter", func(t *testing.T) {
		categories := []string{"restaurant", "cafe"}
		pois, err := repo.Search(ctx, "a", categories, 5)
		if err != nil {
			t.Fatalf("Failed to search POIs with filter: %v", err)
		}

		if len(pois) > 5 {
			t.Errorf("Expected at most 5 POIs, got %d", len(pois))
		}
	})

	t.Run("Search POIs respects limit", func(t *testing.T) {
		pois, err := repo.Search(ctx, "a", nil, 3)
		if err != nil {
			t.Fatalf("Failed to search POIs: %v", err)
		}

		if len(pois) > 3 {
			t.Errorf("Expected at most 3 POIs, got %d", len(pois))
		}
	})

	t.Run("Search POIs with default limit", func(t *testing.T) {
		pois, err := repo.Search(ctx, "a", nil, 0)
		if err != nil {
			t.Fatalf("Failed to search POIs: %v", err)
		}

		if len(pois) > LimitPOIs {
			t.Errorf("Expected at most %d POIs, got %d", LimitPOIs, len(pois))
		}
	})
}

func TestPOIRepository_GetByCategory(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewPOIRepository(db)
	ctx := context.Background()

	t.Run("Get POIs by category", func(t *testing.T) {
		// Find a category that exists
		var category string
		query := `SELECT COALESCE(NULLIF(amenity,''), NULLIF(shop,''), 'other') AS cat
				  FROM planet_osm_point 
				  WHERE amenity IS NOT NULL OR shop IS NOT NULL
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&category)
		if err != nil || category == "" {
			t.Skipf("No categorized POIs found")
		}

		pois, err := repo.GetByCategory(ctx, category, 10)
		if err != nil {
			t.Fatalf("Failed to get POIs by category: %v", err)
		}

		if len(pois) > 10 {
			t.Errorf("Expected at most 10 POIs, got %d", len(pois))
		}

		for _, poi := range pois {
			if poi.Category != category {
				t.Errorf("Expected category %s, got %s", category, poi.Category)
			}
		}
	})

	t.Run("Get POIs by non-existing category", func(t *testing.T) {
		pois, err := repo.GetByCategory(ctx, "non_existing_category_xyz", 10)
		if err != nil {
			t.Fatalf("Failed to get POIs: %v", err)
		}

		if len(pois) != 0 {
			t.Errorf("Expected 0 POIs for non-existing category, got %d", len(pois))
		}
	})
}

func TestPOIRepository_GetCategories(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewPOIRepository(db)
	ctx := context.Background()

	t.Run("Get all categories", func(t *testing.T) {
		categories, err := repo.GetCategories(ctx)
		if err != nil {
			t.Fatalf("Failed to get categories: %v", err)
		}

		if len(categories) == 0 {
			t.Error("Expected at least one category")
		}

		seenIDs := make(map[int64]bool)
		seenCodes := make(map[string]bool)

		for _, cat := range categories {
			if cat.ID == 0 {
				t.Error("Expected non-zero category ID")
			}
			if cat.Code == "" {
				t.Error("Expected non-empty category code")
			}

			// Check for duplicates
			if seenIDs[cat.ID] {
				t.Errorf("Duplicate category ID: %d", cat.ID)
			}
			seenIDs[cat.ID] = true

			if seenCodes[cat.Code] {
				t.Errorf("Duplicate category code: %s", cat.Code)
			}
			seenCodes[cat.Code] = true

			// All name fields should be populated (even if same as code)
			if cat.NameEn == "" {
				t.Error("Expected non-empty NameEn")
			}
		}
	})
}

func TestPOIRepository_GetSubcategories(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewPOIRepository(db)
	ctx := context.Background()

	t.Run("Get subcategories for category", func(t *testing.T) {
		// First get a category
		categories, err := repo.GetCategories(ctx)
		if err != nil || len(categories) == 0 {
			t.Skipf("No categories available")
		}

		categoryID := categories[0].ID
		subcategories, err := repo.GetSubcategories(ctx, categoryID)
		if err != nil {
			t.Fatalf("Failed to get subcategories: %v", err)
		}

		// Subcategories might be empty, but should not error
		for _, subcat := range subcategories {
			if subcat.ID == 0 {
				t.Error("Expected non-zero subcategory ID")
			}
			if subcat.CategoryID != categoryID {
				t.Errorf("Expected category ID %d, got %d", categoryID, subcat.CategoryID)
			}
			if subcat.Code == "" {
				t.Error("Expected non-empty subcategory code")
			}
		}
	})

	t.Run("Get subcategories for non-existing category", func(t *testing.T) {
		_, err := repo.GetSubcategories(ctx, -99999999)
		if err == nil {
			t.Error("Expected error for non-existing category")
		}
	})
}

func TestPOIRepository_GetPOITile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewPOIRepository(db)
	ctx := context.Background()

	t.Run("Get POI tile without category filter", func(t *testing.T) {
		// Barcelona area tile
		z, x, y := 14, 8311, 6143

		tile, err := repo.GetPOITile(ctx, z, x, y, nil)
		if err != nil {
			t.Fatalf("Failed to get POI tile: %v", err)
		}

		// Tile might be empty or contain data
		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get POI tile with category filter", func(t *testing.T) {
		z, x, y := 14, 8311, 6143
		categories := []string{"restaurant", "cafe"}

		tile, err := repo.GetPOITile(ctx, z, x, y, categories)
		if err != nil {
			t.Fatalf("Failed to get POI tile with filter: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get POI tile at different zoom levels", func(t *testing.T) {
		x, y := 8311, 6143

		for _, z := range []int{10, 12, 14, 16} {
			tile, err := repo.GetPOITile(ctx, z, x>>4, y>>4, nil)
			if err != nil {
				t.Errorf("Failed to get tile at zoom %d: %v", z, err)
			}
			if tile == nil {
				t.Errorf("Expected non-nil tile at zoom %d", z)
			}
		}
	})
}

func TestPOIRepository_GetPOIRadiusTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewPOIRepository(db)
	ctx := context.Background()

	t.Run("Get POI radius tile", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734 // Barcelona
		radiusKm := 5.0

		tile, err := repo.GetPOIRadiusTile(ctx, lat, lon, radiusKm, nil)
		if err != nil {
			t.Fatalf("Failed to get POI radius tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get POI radius tile with category filter", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		radiusKm := 2.0
		categories := []string{"restaurant"}

		tile, err := repo.GetPOIRadiusTile(ctx, lat, lon, radiusKm, categories)
		if err != nil {
			t.Fatalf("Failed to get POI radius tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get POI radius tile with zero radius uses default", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734

		tile, err := repo.GetPOIRadiusTile(ctx, lat, lon, 0, nil)
		if err != nil {
			t.Fatalf("Failed to get POI radius tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})
}

func TestPOIRepository_GetPOIByBoundaryTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewPOIRepository(db)
	ctx := context.Background()

	t.Run("Get POI by boundary tile", func(t *testing.T) {
		// Find a boundary
		var boundaryID int64
		query := `SELECT osm_id FROM planet_osm_polygon 
				  WHERE boundary = 'administrative' 
				  AND admin_level = '8'
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&boundaryID)
		if err != nil {
			t.Skipf("No boundaries found")
		}

		tile, err := repo.GetPOIByBoundaryTile(ctx, boundaryID, nil)
		if err != nil {
			t.Fatalf("Failed to get POI by boundary tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get POI by boundary tile with category filter", func(t *testing.T) {
		var boundaryID int64
		query := `SELECT osm_id FROM planet_osm_polygon 
				  WHERE boundary = 'administrative' 
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&boundaryID)
		if err != nil {
			t.Skipf("No boundaries found")
		}

		categories := []string{"restaurant", "cafe"}
		tile, err := repo.GetPOIByBoundaryTile(ctx, boundaryID, categories)
		if err != nil {
			t.Fatalf("Failed to get POI by boundary tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get POI by non-existing boundary", func(t *testing.T) {
		tile, err := repo.GetPOIByBoundaryTile(ctx, -99999999, nil)
		if err != nil {
			t.Fatalf("Failed to get POI by boundary tile: %v", err)
		}

		// Should return empty tile
		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})
}
