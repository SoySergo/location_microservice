package postgres

import (
	"context"
	"database/sql"
	"fmt"

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
		WHERE ST_DWithin(ST_MakePoint(center_lon, center_lat)::geography, point.geom, $3)
		ORDER BY distance
		LIMIT $4
	`

	radiusMeters := radiusKm * 1000
	rows, err := r.db.QueryContext(ctx, query, lon, lat, radiusMeters, LimitGreenSpaces)
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
		WHERE ST_DWithin(geometry::geography, point.geom, $3)
		ORDER BY distance
		LIMIT $4
	`

	radiusMeters := radiusKm * 1000
	rows, err := r.db.QueryContext(ctx, query, lon, lat, radiusMeters, LimitWaterBodies)
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
		WHERE ST_DWithin(geometry::geography, point.geom, $3)
		ORDER BY distance
		LIMIT $4
	`

	radiusMeters := radiusKm * 1000
	rows, err := r.db.QueryContext(ctx, query, lon, lat, radiusMeters, LimitBeaches)
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
		WHERE ST_DWithin(geometry::geography, point.geom, $3)
		ORDER BY distance
		LIMIT $4
	`

	radiusMeters := radiusKm * 1000
	rows, err := r.db.QueryContext(ctx, query, lon, lat, radiusMeters, LimitNoiseSources)
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
		WHERE ST_DWithin(geometry::geography, point.geom, $3)
		ORDER BY distance
		LIMIT $4
	`

	radiusMeters := radiusKm * 1000
	rows, err := r.db.QueryContext(ctx, query, lon, lat, radiusMeters, LimitTouristZones)
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

func (r *environmentRepository) GetGreenSpaceByID(ctx context.Context, id int64) (*domain.GreenSpace, error) {
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
		r.logger.Error("Failed to get green space by ID", zap.Int64("id", id), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return &g, nil
}

func (r *environmentRepository) GetBeachByID(ctx context.Context, id int64) (*domain.Beach, error) {
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
		r.logger.Error("Failed to get beach by ID", zap.Int64("id", id), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return &b, nil
}

func (r *environmentRepository) GetTouristZoneByID(ctx context.Context, id int64) (*domain.TouristZone, error) {
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
		r.logger.Error("Failed to get tourist zone by ID", zap.Int64("id", id), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return &z, nil
}

// GetGreenSpacesTile генерирует MVT tile с зелеными зонами
func (r *environmentRepository) GetGreenSpacesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	query := `
		WITH tile AS (
			SELECT
				id,
				osm_id,
				type,
				name,
				area_sq_m,
				ST_AsMVTGeom(
					ST_Transform(geometry, 3857),
					ST_TileEnvelope($1, $2, $3),
					$4,
					$5,
					true
				) AS geom
			FROM green_spaces
			WHERE ST_Transform(geometry, 3857) && ST_TileEnvelope($1, $2, $3)
			  AND ST_Intersects(ST_Transform(geometry, 3857), ST_TileEnvelope($1, $2, $3))
		)
		SELECT ST_AsMVT(tile.*, 'green_spaces') AS mvt
		FROM tile
		WHERE geom IS NOT NULL
	`

	var mvt []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y, MVTExtent, MVTBuffer).Scan(&mvt)
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

// GetWaterTile генерирует MVT tile с водными объектами
func (r *environmentRepository) GetWaterTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Фильтрация по zoom level: реки видны с z10, озера с z8
	var typeFilter string
	if z < ZoomWaterBodiesMin {
		return []byte{}, nil // Пустой тайл для низких zoom
	} else if z < ZoomWaterRiversMin {
		typeFilter = " AND type IN ('lake', 'reservoir', 'coastline')"
	}

	simplifyTolerance := getMVTSimplifyTolerance(z)

	query := fmt.Sprintf(`
		WITH tile AS (
			SELECT
				id,
				osm_id,
				name,
				type,
				area_sq_m,
				ST_AsMVTGeom(
					ST_Transform(ST_Simplify(geometry, $4), 3857),
					ST_TileEnvelope($1, $2, $3),
					$5,
					$6,
					true
				) AS geom
			FROM water_bodies
			WHERE ST_Transform(geometry, 3857) && ST_TileEnvelope($1, $2, $3)
			  AND ST_Intersects(ST_Transform(geometry, 3857), ST_TileEnvelope($1, $2, $3))
			  %s
		)
		SELECT ST_AsMVT(tile.*, 'water') AS mvt
		FROM tile
		WHERE geom IS NOT NULL
	`, typeFilter)

	var mvt []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y, simplifyTolerance, MVTExtent, MVTBuffer).Scan(&mvt)
	if err != nil {
		if err == sql.ErrNoRows {
			return []byte{}, nil
		}
		r.logger.Error("Failed to generate water tile",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	return mvt, nil
}

// GetBeachesTile генерирует MVT tile с пляжами
func (r *environmentRepository) GetBeachesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Пляжи видны только с zoom >= 12
	if z < ZoomBeachesMin {
		return []byte{}, nil
	}

	query := `
		WITH tile AS (
			SELECT
				id,
				osm_id,
				name,
				surface,
				length as width_m,
				blue_flag,
				ST_AsMVTGeom(
					ST_Transform(geometry, 3857),
					ST_TileEnvelope($1, $2, $3),
					$4,
					$5,
					true
				) AS geom
			FROM beaches
			WHERE ST_Transform(geometry, 3857) && ST_TileEnvelope($1, $2, $3)
			  AND ST_Intersects(ST_Transform(geometry, 3857), ST_TileEnvelope($1, $2, $3))
		)
		SELECT ST_AsMVT(tile.*, 'beaches') AS mvt
		FROM tile
		WHERE geom IS NOT NULL
	`

	var mvt []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y, MVTExtent, MVTBuffer).Scan(&mvt)
	if err != nil {
		if err == sql.ErrNoRows {
			return []byte{}, nil
		}
		r.logger.Error("Failed to generate beaches tile",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	return mvt, nil
}

// GetNoiseSourcesTile генерирует MVT tile с источниками шума
func (r *environmentRepository) GetNoiseSourcesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Zoom level filtering
	var typeFilter string
	if z < ZoomNoiseSourcesMin {
		return []byte{}, nil
	} else if z < ZoomNoiseIndustrialMin {
		typeFilter = " AND type = 'airport'"
	} else if z < ZoomNoiseAllMin {
		typeFilter = " AND type IN ('airport', 'industrial')"
	}
	// z >= 13: все источники шума

	query := fmt.Sprintf(`
		WITH tile AS (
			SELECT
				id,
				osm_id,
				name,
				type,
				intensity as noise_level,
				ST_AsMVTGeom(
					ST_Transform(geometry, 3857),
					ST_TileEnvelope($1, $2, $3),
					$4,
					$5,
					true
				) AS geom
			FROM noise_sources
			WHERE ST_Transform(geometry, 3857) && ST_TileEnvelope($1, $2, $3)
			  AND ST_Intersects(ST_Transform(geometry, 3857), ST_TileEnvelope($1, $2, $3))
			  %s
		)
		SELECT ST_AsMVT(tile.*, 'noise_sources') AS mvt
		FROM tile
		WHERE geom IS NOT NULL
	`, typeFilter)

	var mvt []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y, MVTExtent, MVTBuffer).Scan(&mvt)
	if err != nil {
		if err == sql.ErrNoRows {
			return []byte{}, nil
		}
		r.logger.Error("Failed to generate noise sources tile",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	return mvt, nil
}

// GetTouristZonesTile генерирует MVT tile с туристическими зонами
func (r *environmentRepository) GetTouristZonesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Туристические зоны видны с zoom >= 11
	if z < ZoomTouristZonesMin {
		return []byte{}, nil
	}

	query := `
		WITH tile AS (
			SELECT
				id,
				osm_id,
				name,
				type,
				CASE 
					WHEN visitors_per_year > 1000000 THEN 'international'
					WHEN visitors_per_year > 100000 THEN 'national'
					ELSE 'local'
				END as importance,
				visitors_per_year as visitor_count,
				ST_AsMVTGeom(
					ST_Transform(geometry, 3857),
					ST_TileEnvelope($1, $2, $3),
					$4,
					$5,
					true
				) AS geom
			FROM tourist_zones
			WHERE ST_Transform(geometry, 3857) && ST_TileEnvelope($1, $2, $3)
			  AND ST_Intersects(ST_Transform(geometry, 3857), ST_TileEnvelope($1, $2, $3))
		)
		SELECT ST_AsMVT(tile.*, 'tourist_zones') AS mvt
		FROM tile
		WHERE geom IS NOT NULL
	`

	var mvt []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y, MVTExtent, MVTBuffer).Scan(&mvt)
	if err != nil {
		if err == sql.ErrNoRows {
			return []byte{}, nil
		}
		r.logger.Error("Failed to generate tourist zones tile",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	return mvt, nil
}

// GetEnvironmentRadiusTile генерирует MVT тайл со всеми экологическими объектами в радиусе от точки
func (r *environmentRepository) GetEnvironmentRadiusTile(ctx context.Context, lat, lon, radiusKm float64) ([]byte, error) {
	// Запрос для зеленых зон
	queryGreen := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS geom
		),
		circle AS (
			SELECT ST_Buffer(point.geom, $3)::geometry AS geom
			FROM point
		),
		green_data AS (
			SELECT 
				g.id, g.name, g.type, g.area_sq_m,
				ST_AsMVTGeom(
					g.geometry,
					circle.geom,
					$4,
					$5,
					true
				) AS geom
			FROM green_spaces g, circle
			WHERE g.geometry && circle.geom
			  AND ST_Intersects(g.geometry, circle.geom)
			ORDER BY g.area_sq_m DESC
			LIMIT $6
		)
		SELECT ST_AsMVT(green_data.*, 'green_spaces') AS tile
		FROM green_data
	`

	// Запрос для пляжей
	queryBeaches := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS geom
		),
		circle AS (
			SELECT ST_Buffer(point.geom, $3)::geometry AS geom
			FROM point
		),
		beaches_data AS (
			SELECT 
				b.id, b.name, b.surface,
				ST_AsMVTGeom(
					b.geometry,
					circle.geom,
					$4,
					$5,
					true
				) AS geom
			FROM beaches b, circle
			WHERE b.geometry && circle.geom
			  AND ST_Intersects(circle.geom, b.geometry)
			ORDER BY b.name
			LIMIT $6
		)
		SELECT ST_AsMVT(beaches_data.*, 'beaches') AS tile
		FROM beaches_data
	`

	radiusMeters := radiusKm * 1000

	// Получаем тайл зеленых зон
	var greenTile []byte
	err := r.db.QueryRowContext(ctx, queryGreen, lon, lat, radiusMeters, MVTExtent, MVTBuffer, LimitGreenSpaces).Scan(&greenTile)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("Failed to generate green spaces tile",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon),
			zap.Float64("radius_km", radiusKm),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	// Получаем тайл пляжей
	var beachesTile []byte
	err = r.db.QueryRowContext(ctx, queryBeaches, lon, lat, radiusMeters, MVTExtent, MVTBuffer, LimitBeaches).Scan(&beachesTile)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("Failed to generate beaches tile",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon),
			zap.Float64("radius_km", radiusKm),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	// Объединяем тайлы
	result := append(greenTile, beachesTile...)
	return result, nil
}
