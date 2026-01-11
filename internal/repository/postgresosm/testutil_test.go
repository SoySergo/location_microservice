package postgresosm

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// testDBConfig holds the test database configuration
type testDBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

// getTestDBConfig returns the test database configuration from environment variables
// or defaults to the osm_db service from docker-compose.yml
func getTestDBConfig() testDBConfig {
	return testDBConfig{
		Host:     getEnv("OSM_DB_HOST", "localhost"),
		Port:     getEnv("OSM_DB_PORT", "5435"),
		User:     getEnv("OSM_DB_USER", "osmuser"),
		Password: getEnv("OSM_DB_PASSWORD", "osmpass"),
		DBName:   getEnv("OSM_DB_NAME", "osm"),
		SSLMode:  getEnv("OSM_DB_SSLMODE", "disable"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// setupTestDB creates a connection to the OSM test database
func setupTestDB(t *testing.T) *DB {
	t.Helper()

	cfg := getTestDBConfig()
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)
	db.SetConnMaxIdleTime(1 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Fatalf("Failed to ping test database: %v", err)
	}

	logger := zap.NewNop()
	return NewDBForTest(db, logger)
}

// teardownTestDB closes the database connection
func teardownTestDB(t *testing.T, db *DB) {
	t.Helper()
	if err := db.Close(); err != nil {
		t.Logf("Warning: failed to close test database: %v", err)
	}
}

// skipIfNoOSMData skips the test if OSM data is not available
func skipIfNoOSMData(t *testing.T, db *DB) {
	t.Helper()

	ctx := context.Background()
	var count int

	query := fmt.Sprintf("SELECT COUNT(*) FROM %s LIMIT 1", planetPointTable)
	err := db.QueryRowContext(ctx, query).Scan(&count)

	if err != nil {
		t.Skipf("OSM data not available: %v", err)
	}
}

// assertNotEmpty checks if a string is not empty
func assertNotEmpty(t *testing.T, value string, fieldName string) {
	t.Helper()
	if value == "" {
		t.Errorf("Expected %s to be not empty", fieldName)
	}
}

// assertPositive checks if a number is positive
func assertPositive(t *testing.T, value float64, fieldName string) {
	t.Helper()
	if value <= 0 {
		t.Errorf("Expected %s to be positive, got %f", fieldName, value)
	}
}

// assertValidCoordinates checks if coordinates are valid
func assertValidCoordinates(t *testing.T, lat, lon float64) {
	t.Helper()
	if lat < -90 || lat > 90 {
		t.Errorf("Invalid latitude: %f (must be between -90 and 90)", lat)
	}
	if lon < -180 || lon > 180 {
		t.Errorf("Invalid longitude: %f (must be between -180 and 180)", lon)
	}
}

// assertInRange checks if a value is within a range
func assertInRange(t *testing.T, value, min, max float64, fieldName string) {
	t.Helper()
	if value < min || value > max {
		t.Errorf("Expected %s to be in range [%f, %f], got %f", fieldName, min, max, value)
	}
}
