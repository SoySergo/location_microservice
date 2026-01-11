package postgresosm

import (
	"context"
	"testing"

	pkgerrors "github.com/location-microservice/internal/pkg/errors"
)

func TestTransportRepository_GetNearestStations(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewTransportRepository(db)
	ctx := context.Background()

	t.Run("Get nearest stations without type filter", func(t *testing.T) {
		// Use Barcelona coordinates
		lat, lon := 41.3851, 2.1734
		maxDistance := 2.0

		stations, err := repo.GetNearestStations(ctx, lat, lon, nil, maxDistance, 10)
		if err != nil {
			t.Fatalf("Failed to get nearest stations: %v", err)
		}

		if len(stations) > 10 {
			t.Errorf("Expected at most 10 stations, got %d", len(stations))
		}

		for _, station := range stations {
			if station.OSMId == 0 {
				t.Error("Expected non-zero OSM ID")
			}
			if station.ID != station.OSMId {
				t.Error("Expected ID to equal OSM ID")
			}
			if station.Type == "" {
				t.Error("Expected non-empty type")
			}
			assertValidCoordinates(t, station.Lat, station.Lon)

			if station.LineIDs == nil {
				t.Error("Expected LineIDs to be initialized")
			}
			if station.Tags == nil {
				t.Error("Expected Tags to be initialized")
			}
		}
	})

	t.Run("Get nearest stations with type filter", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		maxDistance := 5.0
		types := []string{"station", "stop"}

		stations, err := repo.GetNearestStations(ctx, lat, lon, types, maxDistance, 20)
		if err != nil {
			t.Fatalf("Failed to get nearest stations with filter: %v", err)
		}

		if len(stations) > 20 {
			t.Errorf("Expected at most 20 stations, got %d", len(stations))
		}
	})

	t.Run("Get nearest stations with default limit", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		maxDistance := 3.0

		stations, err := repo.GetNearestStations(ctx, lat, lon, nil, maxDistance, 0)
		if err != nil {
			t.Fatalf("Failed to get nearest stations: %v", err)
		}

		if len(stations) > LimitStations {
			t.Errorf("Expected at most %d stations, got %d", LimitStations, len(stations))
		}
	})

	t.Run("Get nearest stations respects limit", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		maxDistance := 10.0
		limit := 5

		stations, err := repo.GetNearestStations(ctx, lat, lon, nil, maxDistance, limit)
		if err != nil {
			t.Fatalf("Failed to get nearest stations: %v", err)
		}

		if len(stations) > limit {
			t.Errorf("Expected at most %d stations, got %d", limit, len(stations))
		}
	})
}

func TestTransportRepository_GetLineByID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewTransportRepository(db)
	ctx := context.Background()

	t.Run("Get existing line by ID", func(t *testing.T) {
		// Find a transport line
		var osmID int64
		query := `SELECT osm_id FROM planet_osm_line 
				  WHERE route IS NOT NULL 
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&osmID)
		if err != nil {
			t.Skipf("No transport lines found in database: %v", err)
		}

		line, err := repo.GetLineByID(ctx, osmID)
		if err != nil {
			t.Fatalf("Failed to get line by ID: %v", err)
		}

		if line == nil {
			t.Fatal("Expected line, got nil")
		}

		if line.OSMId != osmID {
			t.Errorf("Expected OSM ID %d, got %d", osmID, line.OSMId)
		}

		if line.ID != line.OSMId {
			t.Error("Expected ID to equal OSM ID")
		}

		if line.Type == "" {
			t.Error("Expected non-empty type")
		}

		if line.StationIDs == nil {
			t.Error("Expected StationIDs to be initialized")
		}

		if line.Tags == nil {
			t.Error("Expected Tags to be initialized")
		}
	})

	t.Run("Get non-existing line", func(t *testing.T) {
		_, err := repo.GetLineByID(ctx, -99999999)
		if err != pkgerrors.ErrLocationNotFound {
			t.Errorf("Expected ErrLocationNotFound, got %v", err)
		}
	})
}

func TestTransportRepository_GetLinesByIDs(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewTransportRepository(db)
	ctx := context.Background()

	t.Run("Get multiple lines by IDs", func(t *testing.T) {
		// Find some line IDs
		var ids []int64
		query := `SELECT DISTINCT osm_id FROM planet_osm_line 
				  WHERE route IS NOT NULL 
				  LIMIT 3`
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			t.Skipf("No transport lines found")
		}
		defer rows.Close()

		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err == nil {
				ids = append(ids, id)
			}
		}

		if len(ids) == 0 {
			t.Skip("No line IDs found")
		}

		lines, err := repo.GetLinesByIDs(ctx, ids)
		if err != nil {
			t.Fatalf("Failed to get lines by IDs: %v", err)
		}

		if len(lines) == 0 {
			t.Error("Expected at least one line")
		}

		if len(lines) > len(ids) {
			t.Errorf("Expected at most %d lines, got %d", len(ids), len(lines))
		}

		for _, line := range lines {
			found := false
			for _, id := range ids {
				if line.OSMId == id {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Line %d not in requested IDs", line.OSMId)
			}
		}
	})

	t.Run("Get lines with empty IDs", func(t *testing.T) {
		lines, err := repo.GetLinesByIDs(ctx, []int64{})
		if err != nil {
			t.Fatalf("Failed with empty IDs: %v", err)
		}

		if len(lines) != 0 {
			t.Errorf("Expected 0 lines, got %d", len(lines))
		}
	})

	t.Run("Get lines with non-existing IDs", func(t *testing.T) {
		lines, err := repo.GetLinesByIDs(ctx, []int64{-99999999, -88888888})
		if err != nil {
			t.Fatalf("Failed to get lines: %v", err)
		}

		if len(lines) != 0 {
			t.Errorf("Expected 0 lines for non-existing IDs, got %d", len(lines))
		}
	})
}

func TestTransportRepository_GetStationsByLineID(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewTransportRepository(db)
	ctx := context.Background()

	t.Run("Get stations by line ID returns empty for now", func(t *testing.T) {
		// This is a stub implementation in OSM
		var lineID int64
		query := `SELECT osm_id FROM planet_osm_line 
				  WHERE route IS NOT NULL 
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&lineID)
		if err != nil {
			t.Skipf("No transport lines found")
		}

		stations, err := repo.GetStationsByLineID(ctx, lineID)
		if err != nil {
			t.Fatalf("Failed to get stations by line ID: %v", err)
		}

		// Currently returns empty array as it's not fully implemented
		if stations == nil {
			t.Error("Expected non-nil stations slice")
		}
	})
}

func TestTransportRepository_GetTransportTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewTransportRepository(db)
	ctx := context.Background()

	t.Run("Get transport tile", func(t *testing.T) {
		// Barcelona area tile
		z, x, y := 14, 8311, 6143

		tile, err := repo.GetTransportTile(ctx, z, x, y)
		if err != nil {
			t.Fatalf("Failed to get transport tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get transport tile at different zoom levels", func(t *testing.T) {
		x, y := 8311, 6143

		for _, z := range []int{10, 12, 14, 16} {
			tile, err := repo.GetTransportTile(ctx, z, x>>4, y>>4)
			if err != nil {
				t.Errorf("Failed to get tile at zoom %d: %v", z, err)
			}
			if tile == nil {
				t.Errorf("Expected non-nil tile at zoom %d", z)
			}
		}
	})
}

func TestTransportRepository_GetLineTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewTransportRepository(db)
	ctx := context.Background()

	t.Run("Get line tile", func(t *testing.T) {
		// Find a line
		var lineID int64
		query := `SELECT osm_id FROM planet_osm_line 
				  WHERE route IS NOT NULL 
				  LIMIT 1`
		err := db.QueryRowContext(ctx, query).Scan(&lineID)
		if err != nil {
			t.Skipf("No transport lines found")
		}

		tile, err := repo.GetLineTile(ctx, lineID)
		if err != nil {
			t.Fatalf("Failed to get line tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get line tile for non-existing line", func(t *testing.T) {
		tile, err := repo.GetLineTile(ctx, -99999999)
		if err != nil {
			t.Fatalf("Failed to get line tile: %v", err)
		}

		// Should return empty tile
		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})
}

func TestTransportRepository_GetLinesTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewTransportRepository(db)
	ctx := context.Background()

	t.Run("Get lines tile", func(t *testing.T) {
		// Find some line IDs
		var ids []int64
		query := `SELECT osm_id FROM planet_osm_line 
				  WHERE route IS NOT NULL 
				  LIMIT 3`
		rows, err := db.QueryContext(ctx, query)
		if err != nil {
			t.Skipf("No transport lines found")
		}
		defer rows.Close()

		for rows.Next() {
			var id int64
			if err := rows.Scan(&id); err == nil {
				ids = append(ids, id)
			}
		}

		if len(ids) == 0 {
			t.Skip("No line IDs found")
		}

		tile, err := repo.GetLinesTile(ctx, ids)
		if err != nil {
			t.Fatalf("Failed to get lines tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get lines tile with empty IDs", func(t *testing.T) {
		tile, err := repo.GetLinesTile(ctx, []int64{})
		if err != nil {
			t.Fatalf("Failed with empty IDs: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
		if len(tile) != 0 {
			t.Error("Expected empty tile for empty IDs")
		}
	})
}

func TestTransportRepository_GetStationsInRadius(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewTransportRepository(db)
	ctx := context.Background()

	t.Run("Get stations in radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734 // Barcelona
		radiusKm := 3.0

		stations, err := repo.GetStationsInRadius(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get stations in radius: %v", err)
		}

		if len(stations) > LimitStations {
			t.Errorf("Expected at most %d stations, got %d", LimitStations, len(stations))
		}

		for _, station := range stations {
			if station.OSMId == 0 {
				t.Error("Expected non-zero OSM ID")
			}
			assertValidCoordinates(t, station.Lat, station.Lon)
		}
	})

	t.Run("Get stations in small radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		radiusKm := 0.5

		stations, err := repo.GetStationsInRadius(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get stations: %v", err)
		}

		// Small radius might have no stations
		_ = stations
	})
}

func TestTransportRepository_GetLinesInRadius(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewTransportRepository(db)
	ctx := context.Background()

	t.Run("Get lines in radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734 // Barcelona
		radiusKm := 5.0

		lines, err := repo.GetLinesInRadius(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get lines in radius: %v", err)
		}

		if len(lines) > LimitLines {
			t.Errorf("Expected at most %d lines, got %d", LimitLines, len(lines))
		}

		for _, line := range lines {
			if line.OSMId == 0 {
				t.Error("Expected non-zero OSM ID")
			}
			if line.Type == "" {
				t.Error("Expected non-empty type")
			}
		}
	})

	t.Run("Get lines in small radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		radiusKm := 1.0

		lines, err := repo.GetLinesInRadius(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get lines: %v", err)
		}

		// Small radius might have fewer or no lines
		_ = lines
	})
}

func TestTransportRepository_GetTransportRadiusTile(t *testing.T) {
	db := setupTestDB(t)
	defer teardownTestDB(t, db)
	skipIfNoOSMData(t, db)

	repo := NewTransportRepository(db)
	ctx := context.Background()

	t.Run("Get transport radius tile", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734 // Barcelona
		radiusKm := 5.0

		tile, err := repo.GetTransportRadiusTile(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get transport radius tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get transport radius tile with small radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		radiusKm := 1.0

		tile, err := repo.GetTransportRadiusTile(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get transport radius tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})

	t.Run("Get transport radius tile with large radius", func(t *testing.T) {
		lat, lon := 41.3851, 2.1734
		radiusKm := 20.0

		tile, err := repo.GetTransportRadiusTile(ctx, lat, lon, radiusKm)
		if err != nil {
			t.Fatalf("Failed to get transport radius tile: %v", err)
		}

		if tile == nil {
			t.Error("Expected non-nil tile")
		}
	})
}
