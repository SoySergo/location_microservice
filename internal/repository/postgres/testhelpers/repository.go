package testhelpers

import (
	"github.com/jmoiron/sqlx"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/repository/postgres"
	"go.uber.org/zap"
)

// NewDBForTest creates a postgres.DB with test database and logger
func NewDBForTest(db *sqlx.DB, logger *zap.Logger) *postgres.DB {
	return postgres.NewDBForTest(db, logger)
}

// NewBoundaryRepositoryForTest creates a boundary repository with test database and logger
func NewBoundaryRepositoryForTest(db *sqlx.DB, logger *zap.Logger) repository.BoundaryRepository {
	pgDB := NewDBForTest(db, logger)
	return postgres.NewBoundaryRepository(pgDB)
}

// NewTransportRepositoryForTest creates a transport repository with test database and logger
func NewTransportRepositoryForTest(db *sqlx.DB, logger *zap.Logger) repository.TransportRepository {
	pgDB := NewDBForTest(db, logger)
	return postgres.NewTransportRepository(pgDB)
}

// NewPOIRepositoryForTest creates a POI repository with test database and logger
func NewPOIRepositoryForTest(db *sqlx.DB, logger *zap.Logger) repository.POIRepository {
	pgDB := NewDBForTest(db, logger)
	return postgres.NewPOIRepository(pgDB)
}

// NewEnvironmentRepositoryForTest creates an environment repository with test database and logger
func NewEnvironmentRepositoryForTest(db *sqlx.DB, logger *zap.Logger) repository.EnvironmentRepository {
	pgDB := NewDBForTest(db, logger)
	return postgres.NewEnvironmentRepository(pgDB)
}

// NewStatsRepositoryForTest creates a stats repository with test database and logger
func NewStatsRepositoryForTest(db *sqlx.DB, logger *zap.Logger) repository.StatsRepository {
	pgDB := NewDBForTest(db, logger)
	return postgres.NewStatsRepository(pgDB, logger)
}
