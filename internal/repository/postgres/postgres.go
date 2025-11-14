package postgres

import (
	"context"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"
	"github.com/location-microservice/internal/config"
	"go.uber.org/zap"
)

type DB struct {
	*sqlx.DB
	logger *zap.Logger
}

func New(cfg *config.DatabaseConfig, logger *zap.Logger) (*DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
		cfg.Host, cfg.Port, cfg.User, cfg.Password, cfg.DBName, cfg.SSLMode,
	)

	db, err := sqlx.Connect("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// Connection pool settings
	db.SetMaxOpenConns(cfg.MaxConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("PostgreSQL connected",
		zap.String("host", cfg.Host),
		zap.Int("port", cfg.Port),
		zap.String("database", cfg.DBName),
	)

	return &DB{DB: db, logger: logger}, nil
}

func (db *DB) Close() error {
	db.logger.Info("Closing PostgreSQL connection")
	return db.DB.Close()
}

func (db *DB) Health(ctx context.Context) error {
	return db.PingContext(ctx)
}

// NewDBForTest creates a DB instance for testing with provided database and logger
func NewDBForTest(sqlxDB *sqlx.DB, logger *zap.Logger) *DB {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &DB{
		DB:     sqlxDB,
		logger: logger,
	}
}
