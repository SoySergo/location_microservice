package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"go.uber.org/zap"
)

type statsRepository struct {
	db     *DB
	logger *zap.Logger
}

// NewStatsRepository создает новый экземпляр stats repository
func NewStatsRepository(db *DB, logger *zap.Logger) repository.StatsRepository {
	return &statsRepository{
		db:     db,
		logger: logger,
	}
}

// GetStatistics возвращает агрегированную статистику по всем данным
func (r *statsRepository) GetStatistics(ctx context.Context) (*domain.Statistics, error) {
	stats := &domain.Statistics{
		LastUpdated: time.Now(),
		DataVersion: "1.0",
	}

	// Получаем статистику по boundaries
	boundaryStats, err := r.getBoundaryStats(ctx)
	if err != nil {
		r.logger.Error("failed to get boundary stats", zap.Error(err))
		return nil, fmt.Errorf("get boundary stats: %w", err)
	}
	stats.Boundaries = *boundaryStats

	// Получаем статистику по transport
	transportStats, err := r.getTransportStats(ctx)
	if err != nil {
		r.logger.Error("failed to get transport stats", zap.Error(err))
		return nil, fmt.Errorf("get transport stats: %w", err)
	}
	stats.Transport = *transportStats

	// Получаем статистику по POI
	poiStats, err := r.getPOIStats(ctx)
	if err != nil {
		r.logger.Error("failed to get poi stats", zap.Error(err))
		return nil, fmt.Errorf("get poi stats: %w", err)
	}
	stats.POIs = *poiStats

	// Получаем статистику по environment
	envStats, err := r.getEnvironmentStats(ctx)
	if err != nil {
		r.logger.Error("failed to get environment stats", zap.Error(err))
		return nil, fmt.Errorf("get environment stats: %w", err)
	}
	stats.Environment = *envStats

	// Получаем покрытие территории
	coverage, err := r.getCoverageStats(ctx)
	if err != nil {
		r.logger.Error("failed to get coverage stats", zap.Error(err))
		return nil, fmt.Errorf("get coverage stats: %w", err)
	}
	stats.Coverage = *coverage

	return stats, nil
}

// RefreshStatistics обновляет кешированную статистику (stub для будущего использования)
func (r *statsRepository) RefreshStatistics(ctx context.Context) error {
	// Эта функция может быть использована для предварительного вычисления статистики
	// и сохранения её в отдельную таблицу для быстрого доступа
	return nil
}

// getBoundaryStats получает статистику по границам
func (r *statsRepository) getBoundaryStats(ctx context.Context) (*domain.BoundaryStats, error) {
	stats := &domain.BoundaryStats{
		ByAdminLevel: make(map[int]int),
	}

	// Общее количество и группировка по admin_level
	query := `
		SELECT 
			admin_level,
			COUNT(*) as count
		FROM admin_boundaries
		GROUP BY admin_level
		ORDER BY admin_level
	`

	rows, err := r.db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query boundary stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var adminLevel int
		var count int
		if err := rows.Scan(&adminLevel, &count); err != nil {
			return nil, fmt.Errorf("scan boundary stats: %w", err)
		}

		stats.ByAdminLevel[adminLevel] = count
		stats.TotalBoundaries += count

		// Подсчет по типам (примерная классификация)
		switch adminLevel {
		case 2:
			stats.Countries = count
		case 4:
			stats.Regions = count
		case 8:
			stats.Cities = count
		}
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("boundary stats rows error: %w", err)
	}

	return stats, nil
}

// getTransportStats получает статистику по транспорту
func (r *statsRepository) getTransportStats(ctx context.Context) (*domain.TransportStats, error) {
	stats := &domain.TransportStats{
		ByType: make(map[string]int),
	}

	// Статистика по станциям
	stationsQuery := `
		SELECT 
			type,
			COUNT(*) as count
		FROM transport_stations
		GROUP BY type
	`

	rows, err := r.db.DB.QueryContext(ctx, stationsQuery)
	if err != nil {
		return nil, fmt.Errorf("query transport stations stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var transportType string
		var count int
		if err := rows.Scan(&transportType, &count); err != nil {
			return nil, fmt.Errorf("scan transport stations stats: %w", err)
		}

		stats.ByType[transportType] = count
		stats.TotalStations += count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("transport stations stats rows error: %w", err)
	}

	// Количество линий
	linesQuery := `SELECT COUNT(*) FROM transport_lines`
	if err := r.db.DB.QueryRowContext(ctx, linesQuery).Scan(&stats.TotalLines); err != nil {
		return nil, fmt.Errorf("query transport lines count: %w", err)
	}

	return stats, nil
}

// getPOIStats получает статистику по POI
func (r *statsRepository) getPOIStats(ctx context.Context) (*domain.POIStats, error) {
	stats := &domain.POIStats{
		ByCategory: make(map[string]int),
	}

	query := `
		SELECT 
			category,
			COUNT(*) as count
		FROM pois
		GROUP BY category
	`

	rows, err := r.db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query poi stats: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var category string
		var count int
		if err := rows.Scan(&category, &count); err != nil {
			return nil, fmt.Errorf("scan poi stats: %w", err)
		}

		stats.ByCategory[category] = count
		stats.TotalPOIs += count
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("poi stats rows error: %w", err)
	}

	return stats, nil
}

// getEnvironmentStats получает статистику по окружению
func (r *statsRepository) getEnvironmentStats(ctx context.Context) (*domain.EnvironmentStats, error) {
	stats := &domain.EnvironmentStats{}

	// Получаем все счетчики параллельно через prepared statements
	queries := []struct {
		query  string
		target *int
	}{
		{"SELECT COUNT(*) FROM green_spaces", &stats.GreenSpaces},
		{"SELECT COUNT(*) FROM water_bodies", &stats.WaterBodies},
		{"SELECT COUNT(*) FROM beaches", &stats.Beaches},
		{"SELECT COUNT(*) FROM noise_sources", &stats.NoiseSources},
		{"SELECT COUNT(*) FROM tourist_zones", &stats.TouristZones},
	}

	for _, q := range queries {
		if err := r.db.DB.QueryRowContext(ctx, q.query).Scan(q.target); err != nil {
			r.logger.Warn("failed to get count", zap.String("query", q.query), zap.Error(err))
			*q.target = 0 // Устанавливаем 0 если таблица не существует
		}
	}

	return stats, nil
}

// getCoverageStats получает статистику покрытия территории
func (r *statsRepository) getCoverageStats(ctx context.Context) (*domain.CoverageStats, error) {
	stats := &domain.CoverageStats{}

	query := `
		SELECT 
			ST_XMin(extent_box) as min_lon,
			ST_YMin(extent_box) as min_lat,
			ST_XMax(extent_box) as max_lon,
			ST_YMax(extent_box) as max_lat,
			ST_Area(ST_MakeEnvelope(
				ST_XMin(extent_box), ST_YMin(extent_box),
				ST_XMax(extent_box), ST_YMax(extent_box),
				4326
			)::geography) / 1000000 as area_sqkm,
			(ST_XMin(extent_box) + ST_XMax(extent_box)) / 2 as center_lon,
			(ST_YMin(extent_box) + ST_YMax(extent_box)) / 2 as center_lat
		FROM (
			SELECT ST_Extent(geometry) as extent_box
			FROM admin_boundaries
			WHERE admin_level = 2
		) as subquery
	`

	err := r.db.DB.QueryRowContext(ctx, query).Scan(
		&stats.BBoxMinLon,
		&stats.BBoxMinLat,
		&stats.BBoxMaxLon,
		&stats.BBoxMaxLat,
		&stats.AreaSqKm,
		&stats.CenterLon,
		&stats.CenterLat,
	)

	if err != nil {
		return nil, fmt.Errorf("query coverage stats: %w", err)
	}

	return stats, nil
}
