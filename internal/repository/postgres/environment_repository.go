package postgres

import (
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/pkg/errors"
	"go.uber.org/zap"
)

type environmentRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewEnvironmentRepository(db *DB) repository.EnvironmentRepository {
	return &environmentRepository{
		db:     db.DB,
		logger: db.logger,
	}
}

func (r *environmentRepository) GetGreenSpacesNearby(
	ctx context.Context,
	lat, lon, radiusKm float64,
) ([]*domain.GreenSpace, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS geom
		)
		SELECT 
			id, osm_id, type, name, name_en, area_sq_m, center_lat, center_lon,
			ST_Distance(ST_MakePoint(center_lon, center_lat)::geography, point.geom) AS distance
		FROM green_spaces, point
		WHERE ST_DWithin(ST_MakePoint(center_lon, center_lat)::geography, point.geom, $3 * 1000)
		ORDER BY distance
		LIMIT 50
	`

	rows, err := r.db.QueryContext(ctx, query, lon, lat, radiusKm)
	if err != nil {
		r.logger.Error("Failed to get green spaces by radius", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var spaces []*domain.GreenSpace
	for rows.Next() {
		var g domain.GreenSpace
		var distance float64
		err := rows.Scan(&g.ID, &g.OSMId, &g.Type, &g.Name, &g.NameEn, &g.AreaSqM,
			&g.CenterLat, &g.CenterLon, &distance)
		if err != nil {
			continue
		}
		spaces = append(spaces, &g)
	}

	return spaces, nil
}

func (r *environmentRepository) GetWaterBodiesNearby(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.WaterBody, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS geom
		)
		SELECT 
			id, osm_id, type, name, name_en, area_sq_m, length,
			ST_Distance(geometry::geography, point.geom) AS distance
		FROM water_bodies, point
		WHERE ST_DWithin(geometry::geography, point.geom, $3 * 1000)
		ORDER BY distance
		LIMIT 50
	`

	rows, err := r.db.QueryContext(ctx, query, lon, lat, radiusKm)
	if err != nil {
		r.logger.Error("Failed to get water bodies by radius", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var waterBodies []*domain.WaterBody
	for rows.Next() {
		var w domain.WaterBody
		var distance float64
		err := rows.Scan(&w.ID, &w.OSMId, &w.Type, &w.Name, &w.NameEn, &w.AreaSqM,
			&w.Length, &distance)
		if err != nil {
			continue
		}
		waterBodies = append(waterBodies, &w)
	}

	return waterBodies, nil
}

func (r *environmentRepository) GetBeachesNearby(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.Beach, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS geom
		)
		SELECT 
			id, osm_id, name, name_en, surface, lat, lon, length,
			ST_Distance(geometry::geography, point.geom) AS distance
		FROM beaches, point
		WHERE ST_DWithin(geometry::geography, point.geom, $3 * 1000)
		ORDER BY distance
		LIMIT 20
	`

	rows, err := r.db.QueryContext(ctx, query, lon, lat, radiusKm)
	if err != nil {
		r.logger.Error("Failed to get beaches by radius", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var beaches []*domain.Beach
	for rows.Next() {
		var b domain.Beach
		var distance float64
		err := rows.Scan(&b.ID, &b.OSMId, &b.Name, &b.NameEn, &b.Surface,
			&b.Lat, &b.Lon, &b.Length, &distance)
		if err != nil {
			continue
		}
		beaches = append(beaches, &b)
	}

	return beaches, nil
}

func (r *environmentRepository) GetNoiseSourcesNearby(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.NoiseSource, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS geom
		)
		SELECT 
			id, osm_id, type, name, intensity, lat, lon,
			ST_Distance(geometry::geography, point.geom) AS distance
		FROM noise_sources, point
		WHERE ST_DWithin(geometry::geography, point.geom, $3 * 1000)
		ORDER BY distance
		LIMIT 50
	`

	rows, err := r.db.QueryContext(ctx, query, lon, lat, radiusKm)
	if err != nil {
		r.logger.Error("Failed to get noise sources by radius", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var noiseSources []*domain.NoiseSource
	for rows.Next() {
		var n domain.NoiseSource
		var distance float64
		err := rows.Scan(&n.ID, &n.OSMId, &n.Type, &n.Name, &n.Intensity,
			&n.Lat, &n.Lon, &distance)
		if err != nil {
			continue
		}
		noiseSources = append(noiseSources, &n)
	}

	return noiseSources, nil
}

func (r *environmentRepository) GetTouristZonesNearby(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.TouristZone, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS geom
		)
		SELECT 
			id, osm_id, type, name, name_en, name_es, name_ca, name_ru, name_uk,
			name_fr, name_pt, name_it, name_de, lat, lon, visitors_per_year, fee,
			opening_hours, website,
			ST_Distance(geometry::geography, point.geom) AS distance
		FROM tourist_zones, point
		WHERE ST_DWithin(geometry::geography, point.geom, $3 * 1000)
		ORDER BY distance
		LIMIT 50
	`

	rows, err := r.db.QueryContext(ctx, query, lon, lat, radiusKm)
	if err != nil {
		r.logger.Error("Failed to get tourist zones by radius", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var zones []*domain.TouristZone
	for rows.Next() {
		var z domain.TouristZone
		var distance float64
		err := rows.Scan(&z.ID, &z.OSMId, &z.Type, &z.Name, &z.NameEn, &z.NameEs, &z.NameCa,
			&z.NameRu, &z.NameUk, &z.NameFr, &z.NamePt, &z.NameIt, &z.NameDe,
			&z.Lat, &z.Lon, &z.VisitorsPerYear, &z.Fee, &z.OpeningHours, &z.Website, &distance)
		if err != nil {
			continue
		}
		zones = append(zones, &z)
	}

	return zones, nil
}

func (r *environmentRepository) GetGreenSpaceByID(ctx context.Context, id string) (*domain.GreenSpace, error) {
	query := `
		SELECT 
			id, osm_id, type, name, name_en, area_sq_m, center_lat, center_lon, access, tags
		FROM green_spaces
		WHERE id = $1
	`

	var g domain.GreenSpace
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&g.ID, &g.OSMId, &g.Type, &g.Name, &g.NameEn, &g.AreaSqM,
		&g.CenterLat, &g.CenterLon, &g.Access, &g.Tags,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("Failed to get green space by ID", zap.String("id", id), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return &g, nil
}

func (r *environmentRepository) GetBeachByID(ctx context.Context, id string) (*domain.Beach, error) {
	query := `
		SELECT 
			id, osm_id, name, name_en, surface, lat, lon, length, blue_flag, tags
		FROM beaches
		WHERE id = $1
	`

	var b domain.Beach
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&b.ID, &b.OSMId, &b.Name, &b.NameEn, &b.Surface,
		&b.Lat, &b.Lon, &b.Length, &b.BlueFlag, &b.Tags,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("Failed to get beach by ID", zap.String("id", id), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return &b, nil
}

func (r *environmentRepository) GetTouristZoneByID(ctx context.Context, id string) (*domain.TouristZone, error) {
	query := `
		SELECT 
			id, osm_id, type, name, name_en, name_es, name_ca, name_ru, name_uk,
			name_fr, name_pt, name_it, name_de, lat, lon, visitors_per_year, fee,
			opening_hours, website, tags
		FROM tourist_zones
		WHERE id = $1
	`

	var z domain.TouristZone
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&z.ID, &z.OSMId, &z.Type, &z.Name, &z.NameEn, &z.NameEs, &z.NameCa,
		&z.NameRu, &z.NameUk, &z.NameFr, &z.NamePt, &z.NameIt, &z.NameDe,
		&z.Lat, &z.Lon, &z.VisitorsPerYear, &z.Fee, &z.OpeningHours, &z.Website, &z.Tags,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("Failed to get tourist zone by ID", zap.String("id", id), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return &z, nil
}

// GetGreenSpacesTile генерирует MVT tile с зелеными зонами
func (r *environmentRepository) GetGreenSpacesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	query := `
		SELECT ST_AsMVT(tile, 'green_spaces', 4096, 'geom')
		FROM (
			SELECT
				id,
				osm_id,
				type,
				name,
				area_sq_m,
				ST_AsMVTGeom(
					geometry,
					ST_TileEnvelope($1, $2, $3),
					4096,
					256,
					true
				) AS geom
			FROM green_spaces
			WHERE geometry && ST_TileEnvelope($1, $2, $3)
			  AND ST_Intersects(geometry, ST_TileEnvelope($1, $2, $3))
		) AS tile
		WHERE geom IS NOT NULL
	`

	var mvt []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y).Scan(&mvt)
	if err != nil {
		if err == sql.ErrNoRows {
			// Возвращаем пустой тайл
			return []byte{}, nil
		}
		r.logger.Error("Failed to generate green spaces tile",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	return mvt, nil
}
