package testhelpers

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
)

// TestDB represents a test database connection
type TestDB struct {
	DB     *sqlx.DB
	Logger *zap.Logger
}

// SetupTestDB initializes a test database connection
func SetupTestDB(t *testing.T) *TestDB {
	// Priority:
	// 1. Environment variables
	// 2. Default values

	host := getEnv("TEST_DB_HOST", "localhost")
	port := getEnv("TEST_DB_PORT", "5433")
	user := getEnv("TEST_DB_USER", "postgres")
	password := getEnv("TEST_DB_PASSWORD", "postgres")
	dbname := getEnv("TEST_DB_NAME", "location_test")
	sslmode := getEnv("TEST_DB_SSLMODE", "disable")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)

	// Retry connection with exponential backoff to wait for DB recovery
	var db *sqlx.DB
	var err error
	maxRetries := 10
	retryDelay := 500 * time.Millisecond

	for i := 0; i < maxRetries; i++ {
		db, err = sqlx.Connect("postgres", connStr)
		if err == nil {
			break
		}

		if i < maxRetries-1 {
			t.Logf("Database not ready (attempt %d/%d), waiting %v...", i+1, maxRetries, retryDelay)
			time.Sleep(retryDelay)
			retryDelay *= 2 // exponential backoff
		}
	}

	if err != nil {
		t.Fatalf("Failed to connect to test database after %d attempts: %v", maxRetries, err)
	}

	// Check PostGIS availability
	var version string
	err = db.Get(&version, "SELECT PostGIS_Version()")
	if err != nil {
		t.Fatalf("PostGIS not available: %v", err)
	}
	t.Logf("PostGIS version: %s", version)

	logger, _ := zap.NewDevelopment()
	if logger == nil {
		logger = zap.NewNop()
	}

	return &TestDB{
		DB:     db,
		Logger: logger,
	}
}

// Close closes the database connection
func (tdb *TestDB) Close() {
	if tdb.DB != nil {
		tdb.DB.Close()
	}
}

// Cleanup cleans up test data
func (tdb *TestDB) Cleanup(ctx context.Context) error {
	// Truncate tables in correct order (respecting FK constraints)
	tables := []string{
		"pois",
		"poi_subcategories",
		"poi_categories",
		"transport_stations",
		"transport_lines",
		"green_spaces",
		"water_bodies",
		"beaches",
		"noise_sources",
		"tourist_zones",
		"admin_boundaries",
	}

	for _, table := range tables {
		_, err := tdb.DB.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			// Ignore errors if table doesn't exist
			continue
		}
	}

	return nil
}

// getEnv gets environment variable or returns default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
