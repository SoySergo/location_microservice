package testhelpers

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
)

// LoadFixtures loads SQL fixture files into the database
func LoadFixtures(db *sql.DB, fixturesPath string, files []string) error {
	for _, file := range files {
		path := filepath.Join(fixturesPath, file)
		content, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("read fixture %s: %w", file, err)
		}

		if _, err := db.Exec(string(content)); err != nil {
			return fmt.Errorf("load fixture %s: %w", file, err)
		}
		fmt.Printf("Loaded fixture: %s\n", file)
	}

	return nil
}

// GetBoundaryIDByOSMID returns the internal ID for a boundary given its OSM ID
func GetBoundaryIDByOSMID(db *sql.DB, osmID int64) (int64, error) {
	var id int64
	err := db.QueryRowContext(context.Background(),
		"SELECT id FROM admin_boundaries WHERE osm_id = $1", osmID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get boundary ID by OSM ID %d: %w", osmID, err)
	}
	return id, nil
}

// GetTransportLineIDByOSMID returns the internal ID for a transport line given its OSM ID
func GetTransportLineIDByOSMID(db *sql.DB, osmID int64) (int64, error) {
	var id int64
	err := db.QueryRowContext(context.Background(),
		"SELECT id FROM transport_lines WHERE osm_id = $1", osmID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get transport line ID by OSM ID %d: %w", osmID, err)
	}
	return id, nil
}

// GetTransportStationIDByOSMID returns the internal ID for a transport station given its OSM ID
func GetTransportStationIDByOSMID(db *sql.DB, osmID int64) (int64, error) {
	var id int64
	err := db.QueryRowContext(context.Background(),
		"SELECT id FROM transport_stations WHERE osm_id = $1", osmID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("get transport station ID by OSM ID %d: %w", osmID, err)
	}
	return id, nil
}
