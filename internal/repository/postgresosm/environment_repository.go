package postgresosm

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	pkgerrors "github.com/location-microservice/internal/pkg/errors"
	"go.uber.org/zap"
)

type environmentRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewEnvironmentRepository создает репозиторий окружающей среды для OSM базы данных
func NewEnvironmentRepository(db *DB) repository.EnvironmentRepository {
	return &environmentRepository{
		db:     db.DB,
		logger: db.logger,
	}
}

// GetGreenSpacesNearby возвращает зеленые зоны рядом с точкой
func (r *environmentRepository) GetGreenSpacesNearby(
	ctx context.Context,
	lat, lon, radiusKm float64,
) ([]*domain.GreenSpace, error) {
	radiusMeters := radiusKm * 1000

	query := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		)
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(name, ''), NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(NULLIF(leisure, ''), NULLIF(landuse, ''), 'park') AS type,
			ST_Area(ST_Transform(way, %d)::geography) AS area_sq_m,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS center_lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS center_lon,
			COALESCE(tags->'access', '') AS access,
			ST_Distance(ST_Transform(way, %d)::geography, point.geom) AS distance
		FROM %s, point
		WHERE (leisure IN ('park', 'garden', 'nature_reserve', 'playground', 'pitch')
		   OR landuse IN ('forest', 'meadow', 'grass', 'recreation_ground'))
		  AND ST_DWithin(ST_Transform(way, %d)::geography, point.geom, $3)
		ORDER BY distance
		LIMIT $4
	`, SRID4326, SRID4326, SRID4326, SRID4326, SRID4326, planetPolygonTable, SRID4326)

	rows, err := r.db.QueryxContext(ctx, query, lon, lat, radiusMeters, LimitGreenSpaces)
	if err != nil {
		r.logger.Error("failed to get osm green spaces", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var spaces []*domain.GreenSpace
	for rows.Next() {
		var g domain.GreenSpace
		var distance float64
		var access string

		err := rows.Scan(&g.OSMId, &g.Name, &g.NameEn, &g.Type, &g.AreaSqM,
			&g.CenterLat, &g.CenterLon, &access, &distance)
		if err != nil {
			r.logger.Error("failed to scan green space row", zap.Error(err))
			continue
		}

		g.ID = g.OSMId
		if access != "" {
			g.Access = &access
		}

		spaces = append(spaces, &g)
	}

	return spaces, nil
}

// GetWaterBodiesNearby возвращает водные объекты рядом с точкой
func (r *environmentRepository) GetWaterBodiesNearby(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.WaterBody, error) {
	radiusMeters := radiusKm * 1000

	query := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		)
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(name, ''), NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(NULLIF("natural", ''), NULLIF(waterway, ''), NULLIF("water", ''), 'water') AS type,
			ST_Area(ST_Transform(way, %d)::geography) AS area_sq_m,
			ST_Length(ST_Transform(way, %d)::geography) AS length,
			ST_Distance(ST_Transform(way, %d)::geography, point.geom) AS distance
		FROM %s, point
		WHERE ("natural" IN ('water', 'bay', 'coastline')
		   OR waterway IN ('river', 'stream', 'canal', 'drain')
		   OR "water" IS NOT NULL)
		  AND ST_DWithin(ST_Transform(way, %d)::geography, point.geom, $3)
		ORDER BY distance
		LIMIT $4
	`, SRID4326, SRID4326, SRID4326, SRID4326, planetPolygonTable, SRID4326)

	rows, err := r.db.QueryxContext(ctx, query, lon, lat, radiusMeters, LimitWaterBodies)
	if err != nil {
		r.logger.Error("failed to get osm water bodies", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var waterBodies []*domain.WaterBody
	for rows.Next() {
		var w domain.WaterBody
		var distance float64

		err := rows.Scan(&w.OSMId, &w.Name, &w.NameEn, &w.Type, &w.AreaSqM, &w.Length, &distance)
		if err != nil {
			r.logger.Error("failed to scan water body row", zap.Error(err))
			continue
		}

		w.ID = w.OSMId

		waterBodies = append(waterBodies, &w)
	}

	return waterBodies, nil
}

// GetBeachesNearby возвращает пляжи рядом с точкой
func (r *environmentRepository) GetBeachesNearby(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.Beach, error) {
	radiusMeters := radiusKm * 1000

	query := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		)
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(name, ''), NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(tags->'surface', '') AS surface,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS lon,
			ST_Length(ST_Transform(way, %d)::geography) AS length,
			ST_Distance(ST_Transform(way, %d)::geography, point.geom) AS distance
		FROM %s, point
		WHERE "natural" = 'beach'
		  AND ST_DWithin(ST_Transform(way, %d)::geography, point.geom, $3)
		ORDER BY distance
		LIMIT $4
	`, SRID4326, SRID4326, SRID4326, SRID4326, SRID4326, planetPolygonTable, SRID4326)

	rows, err := r.db.QueryxContext(ctx, query, lon, lat, radiusMeters, LimitBeaches)
	if err != nil {
		r.logger.Error("failed to get osm beaches", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var beaches []*domain.Beach
	for rows.Next() {
		var b domain.Beach
		var distance float64
		var surface string

		err := rows.Scan(&b.OSMId, &b.Name, &b.NameEn, &surface, &b.Lat, &b.Lon, &b.Length, &distance)
		if err != nil {
			r.logger.Error("failed to scan beach row", zap.Error(err))
			continue
		}

		b.ID = b.OSMId
		b.Surface = surface
		blueFlag := false // OSM не содержит информацию о голубом флаге напрямую
		b.BlueFlag = &blueFlag

		beaches = append(beaches, &b)
	}

	return beaches, nil
}

// GetNoiseSourcesNearby возвращает источники шума рядом с точкой
func (r *environmentRepository) GetNoiseSourcesNearby(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.NoiseSource, error) {
	radiusMeters := radiusKm * 1000

	query := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		)
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			CASE 
				WHEN aeroway IS NOT NULL THEN 'airport'
				WHEN landuse = 'industrial' THEN 'industrial'
				WHEN highway IN ('motorway', 'trunk', 'primary') THEN 'highway'
				WHEN railway IS NOT NULL THEN 'railway'
				ELSE 'other'
			END AS type,
			CASE 
				WHEN aeroway IS NOT NULL THEN 'high'
				WHEN landuse = 'industrial' THEN 'medium'
				WHEN highway IN ('motorway', 'trunk') THEN 'high'
				WHEN highway = 'primary' THEN 'medium'
				WHEN railway IS NOT NULL THEN 'medium'
				ELSE 'low'
			END AS intensity,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS lon,
			ST_Distance(ST_Transform(way, %d)::geography, point.geom) AS distance
		FROM %s, point
		WHERE (aeroway IN ('aerodrome', 'heliport')
		   OR landuse = 'industrial'
		   OR highway IN ('motorway', 'trunk', 'primary')
		   OR railway IN ('rail', 'light_rail', 'subway'))
		  AND ST_DWithin(ST_Transform(way, %d)::geography, point.geom, $3)
		ORDER BY distance
		LIMIT $4
	`, SRID4326, SRID4326, SRID4326, SRID4326, planetPolygonTable, SRID4326)

	rows, err := r.db.QueryxContext(ctx, query, lon, lat, radiusMeters, LimitNoiseSources)
	if err != nil {
		r.logger.Error("failed to get osm noise sources", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var noiseSources []*domain.NoiseSource
	for rows.Next() {
		var n domain.NoiseSource
		var distance float64
		var intensity string

		err := rows.Scan(&n.OSMId, &n.Name, &n.Type, &intensity, &n.Lat, &n.Lon, &distance)
		if err != nil {
			r.logger.Error("failed to scan noise source row", zap.Error(err))
			continue
		}

		n.ID = n.OSMId
		if intensity != "" {
			n.Intensity = &intensity
		}

		noiseSources = append(noiseSources, &n)
	}

	return noiseSources, nil
}

// GetTouristZonesNearby возвращает туристические зоны рядом с точкой
func (r *environmentRepository) GetTouristZonesNearby(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.TouristZone, error) {
	radiusMeters := radiusKm * 1000

	query := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		)
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(NULLIF(tags->'name:es', ''), '') AS name_es,
			COALESCE(NULLIF(tags->'name:ca', ''), '') AS name_ca,
			COALESCE(NULLIF(tags->'name:ru', ''), '') AS name_ru,
			COALESCE(NULLIF(tags->'name:uk', ''), '') AS name_uk,
			COALESCE(NULLIF(tags->'name:fr', ''), '') AS name_fr,
			COALESCE(NULLIF(tags->'name:pt', ''), '') AS name_pt,
			COALESCE(NULLIF(tags->'name:it', ''), '') AS name_it,
			COALESCE(NULLIF(tags->'name:de', ''), '') AS name_de,
			COALESCE(tourism, '') AS type,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS lon,
			COALESCE(tags->'fee', '') AS fee,
			COALESCE(tags->'opening_hours', '') AS opening_hours,
			COALESCE(tags->'website', '') AS website,
			ST_Distance(ST_Transform(way, %d)::geography, point.geom) AS distance
		FROM %s, point
		WHERE tourism IN ('attraction', 'museum', 'theme_park', 'zoo', 'aquarium', 'viewpoint')
		  AND ST_DWithin(ST_Transform(way, %d)::geography, point.geom, $3)
		ORDER BY distance
		LIMIT $4
	`, SRID4326, SRID4326, SRID4326, SRID4326, planetPolygonTable, SRID4326)

	rows, err := r.db.QueryxContext(ctx, query, lon, lat, radiusMeters, LimitTouristZones)
	if err != nil {
		r.logger.Error("failed to get osm tourist zones", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var zones []*domain.TouristZone
	for rows.Next() {
		var z domain.TouristZone
		var distance float64
		var fee, openingHours, website string

		err := rows.Scan(&z.OSMId, &z.Name, &z.NameEn, &z.NameEs, &z.NameCa,
			&z.NameRu, &z.NameUk, &z.NameFr, &z.NamePt, &z.NameIt, &z.NameDe,
			&z.Type, &z.Lat, &z.Lon, &fee, &openingHours, &website, &distance)
		if err != nil {
			r.logger.Error("failed to scan tourist zone row", zap.Error(err))
			continue
		}

		z.ID = z.OSMId
		if fee != "" {
			// Конвертируем yes/no в bool
			feeBool := fee == "yes" || fee == "true" || fee == "1"
			z.Fee = &feeBool
		}
		if openingHours != "" {
			z.OpeningHours = &openingHours
		}
		if website != "" {
			z.Website = &website
		}

		zones = append(zones, &z)
	}

	return zones, nil
}

// GetGreenSpaceByID возвращает зеленую зону по ID
func (r *environmentRepository) GetGreenSpaceByID(ctx context.Context, id int64) (*domain.GreenSpace, error) {
	query := fmt.Sprintf(`
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(name, ''), NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(NULLIF(leisure, ''), NULLIF(landuse, ''), 'park') AS type,
			ST_Area(ST_Transform(way, %d)::geography) AS area_sq_m,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS center_lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS center_lon,
			COALESCE(tags->'access', '') AS access
		FROM %s
		WHERE osm_id = $1
		  AND (leisure IN ('park', 'garden', 'nature_reserve', 'playground', 'pitch')
		   OR landuse IN ('forest', 'meadow', 'grass', 'recreation_ground'))
		LIMIT 1
	`, SRID4326, SRID4326, SRID4326, planetPolygonTable)

	var g domain.GreenSpace
	var access string

	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&g.OSMId, &g.Name, &g.NameEn, &g.Type, &g.AreaSqM,
		&g.CenterLat, &g.CenterLon, &access,
	)

	if err == sql.ErrNoRows {
		return nil, pkgerrors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("failed to get osm green space", zap.Int64("osm_id", id), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	g.ID = g.OSMId
	if access != "" {
		g.Access = &access
	}

	return &g, nil
}

// GetBeachByID возвращает пляж по ID
func (r *environmentRepository) GetBeachByID(ctx context.Context, id int64) (*domain.Beach, error) {
	query := fmt.Sprintf(`
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(name, ''), NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(tags->'surface', '') AS surface,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS lon,
			ST_Length(ST_Transform(way, %d)::geography) AS length
		FROM %s
		WHERE osm_id = $1
		  AND "natural" = 'beach'
		LIMIT 1
	`, SRID4326, SRID4326, SRID4326, planetPolygonTable)

	var b domain.Beach
	var surface string

	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&b.OSMId, &b.Name, &b.NameEn, &surface, &b.Lat, &b.Lon, &b.Length,
	)

	if err == sql.ErrNoRows {
		return nil, pkgerrors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("failed to get osm beach", zap.Int64("osm_id", id), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	b.ID = b.OSMId
	b.Surface = surface
	blueFlag := false
	b.BlueFlag = &blueFlag

	return &b, nil
}

// GetTouristZoneByID возвращает туристическую зону по ID
func (r *environmentRepository) GetTouristZoneByID(ctx context.Context, id int64) (*domain.TouristZone, error) {
	query := fmt.Sprintf(`
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(NULLIF(tags->'name:es', ''), '') AS name_es,
			COALESCE(NULLIF(tags->'name:ca', ''), '') AS name_ca,
			COALESCE(NULLIF(tags->'name:ru', ''), '') AS name_ru,
			COALESCE(NULLIF(tags->'name:uk', ''), '') AS name_uk,
			COALESCE(NULLIF(tags->'name:fr', ''), '') AS name_fr,
			COALESCE(NULLIF(tags->'name:pt', ''), '') AS name_pt,
			COALESCE(NULLIF(tags->'name:it', ''), '') AS name_it,
			COALESCE(NULLIF(tags->'name:de', ''), '') AS name_de,
			COALESCE(tourism, '') AS type,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS lon,
			COALESCE(tags->'fee', '') AS fee,
			COALESCE(tags->'opening_hours', '') AS opening_hours,
			COALESCE(tags->'website', '') AS website
		FROM %s
		WHERE osm_id = $1
		  AND tourism IN ('attraction', 'museum', 'theme_park', 'zoo', 'aquarium', 'viewpoint')
		LIMIT 1
	`, SRID4326, SRID4326, planetPolygonTable)

	var z domain.TouristZone
	var fee, openingHours, website string

	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&z.OSMId, &z.Name, &z.NameEn, &z.NameEs, &z.NameCa,
		&z.NameRu, &z.NameUk, &z.NameFr, &z.NamePt, &z.NameIt, &z.NameDe,
		&z.Type, &z.Lat, &z.Lon, &fee, &openingHours, &website,
	)

	if err == sql.ErrNoRows {
		return nil, pkgerrors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("failed to get osm tourist zone", zap.Int64("osm_id", id), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	z.ID = z.OSMId
	if fee != "" {
		// Конвертируем yes/no в bool
		feeBool := fee == "yes" || fee == "true" || fee == "1"
		z.Fee = &feeBool
	}
	if openingHours != "" {
		z.OpeningHours = &openingHours
	}
	if website != "" {
		z.Website = &website
	}

	return &z, nil
}

// GetGreenSpacesTile генерирует MVT тайл с зелеными зонами
func (r *environmentRepository) GetGreenSpacesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	query := fmt.Sprintf(`
		WITH bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		green_data AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(NULLIF(leisure, ''), NULLIF(landuse, ''), 'park') AS type,
				ST_Area(ST_Transform(way, %d)::geography) AS area_sq_m,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM %s, bounds
			WHERE (leisure IN ('park', 'garden', 'nature_reserve', 'playground', 'pitch')
			   OR landuse IN ('forest', 'meadow', 'grass', 'recreation_ground'))
			  AND way && bounds.geom
		)
		SELECT COALESCE(ST_AsMVT(green_data.*, 'green_spaces'), '\\x'::bytea) AS tile
		FROM green_data
		WHERE geom IS NOT NULL
	`, SRID4326, planetPolygonTable)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y, MVTExtent, MVTBuffer).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("failed to build osm green spaces tile", zap.Int("z", z), zap.Int("x", x), zap.Int("y", y), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return tile, nil
}

// GetWaterTile генерирует MVT тайл с водными объектами
func (r *environmentRepository) GetWaterTile(ctx context.Context, z, x, y int) ([]byte, error) {
	query := fmt.Sprintf(`
		WITH bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		water_data AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(NULLIF("natural", ''), NULLIF(waterway, ''), NULLIF("water", ''), 'water') AS type,
				ST_Area(ST_Transform(way, %d)::geography) AS area_sq_m,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM %s, bounds
			WHERE ("natural" IN ('water', 'bay', 'coastline')
			   OR waterway IN ('river', 'stream', 'canal', 'drain')
			   OR "water" IS NOT NULL)
			  AND way && bounds.geom
		)
		SELECT COALESCE(ST_AsMVT(water_data.*, 'water'), '\\x'::bytea) AS tile
		FROM water_data
		WHERE geom IS NOT NULL
	`, SRID4326, planetPolygonTable)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y, MVTExtent, MVTBuffer).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("failed to build osm water tile", zap.Int("z", z), zap.Int("x", x), zap.Int("y", y), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return tile, nil
}

// GetBeachesTile генерирует MVT тайл с пляжами
func (r *environmentRepository) GetBeachesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Пляжи видны с zoom >= 12
	if z < 12 {
		return []byte{}, nil
	}

	query := fmt.Sprintf(`
		WITH bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		beach_data AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(tags->'surface', '') AS surface,
				ST_Length(ST_Transform(way, %d)::geography) AS width_m,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM %s, bounds
			WHERE "natural" = 'beach'
			  AND way && bounds.geom
		)
		SELECT COALESCE(ST_AsMVT(beach_data.*, 'beaches'), '\\x'::bytea) AS tile
		FROM beach_data
		WHERE geom IS NOT NULL
	`, SRID4326, planetPolygonTable)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y, MVTExtent, MVTBuffer).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("failed to build osm beaches tile", zap.Int("z", z), zap.Int("x", x), zap.Int("y", y), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return tile, nil
}

// GetNoiseSourcesTile генерирует MVT тайл с источниками шума
func (r *environmentRepository) GetNoiseSourcesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Источники шума видны с zoom >= 10
	if z < 10 {
		return []byte{}, nil
	}

	query := fmt.Sprintf(`
		WITH bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		noise_data AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				CASE 
					WHEN aeroway IS NOT NULL THEN 'airport'
					WHEN landuse = 'industrial' THEN 'industrial'
					WHEN highway IN ('motorway', 'trunk', 'primary') THEN 'highway'
					WHEN railway IS NOT NULL THEN 'railway'
					ELSE 'other'
				END AS type,
				CASE 
					WHEN aeroway IS NOT NULL THEN 'high'
					WHEN landuse = 'industrial' THEN 'medium'
					WHEN highway IN ('motorway', 'trunk') THEN 'high'
					WHEN highway = 'primary' THEN 'medium'
					WHEN railway IS NOT NULL THEN 'medium'
					ELSE 'low'
				END AS noise_level,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM %s, bounds
			WHERE (aeroway IN ('aerodrome', 'heliport')
			   OR landuse = 'industrial'
			   OR highway IN ('motorway', 'trunk', 'primary')
			   OR railway IN ('rail', 'light_rail', 'subway'))
			  AND way && bounds.geom
		)
		SELECT COALESCE(ST_AsMVT(noise_data.*, 'noise_sources'), '\\x'::bytea) AS tile
		FROM noise_data
		WHERE geom IS NOT NULL
	`, planetPolygonTable)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y, MVTExtent, MVTBuffer).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("failed to build osm noise sources tile", zap.Int("z", z), zap.Int("x", x), zap.Int("y", y), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return tile, nil
}

// GetTouristZonesTile генерирует MVT тайл с туристическими зонами
func (r *environmentRepository) GetTouristZonesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Туристические зоны видны с zoom >= 11
	if z < 11 {
		return []byte{}, nil
	}

	query := fmt.Sprintf(`
		WITH bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		tourist_data AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(tourism, '') AS type,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM %s, bounds
			WHERE tourism IN ('attraction', 'museum', 'theme_park', 'zoo', 'aquarium', 'viewpoint')
			  AND way && bounds.geom
		)
		SELECT COALESCE(ST_AsMVT(tourist_data.*, 'tourist_zones'), '\\x'::bytea) AS tile
		FROM tourist_data
		WHERE geom IS NOT NULL
	`, planetPolygonTable)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y, MVTExtent, MVTBuffer).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("failed to build osm tourist zones tile", zap.Int("z", z), zap.Int("x", x), zap.Int("y", y), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return tile, nil
}

// GetEnvironmentRadiusTile генерирует MVT тайл со всеми экологическими объектами в радиусе
func (r *environmentRepository) GetEnvironmentRadiusTile(ctx context.Context, lat, lon, radiusKm float64) ([]byte, error) {
	radiusMeters := radiusKm * 1000

	// Зеленые зоны
	greenQuery := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		),
		circle AS (
			SELECT ST_Buffer(point.geom, $3)::geometry AS geom
			FROM point
		),
		green_data AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(NULLIF(leisure, ''), NULLIF(landuse, ''), 'park') AS type,
				ST_Area(ST_Transform(way, %d)::geography) AS area_sq_m,
				ST_AsMVTGeom(way, circle.geom, $4, $5, true) AS geom
			FROM %s, circle
			WHERE (leisure IN ('park', 'garden', 'nature_reserve', 'playground', 'pitch')
			   OR landuse IN ('forest', 'meadow', 'grass', 'recreation_ground'))
			  AND way && circle.geom
			  AND ST_Intersects(way, circle.geom)
			ORDER BY area_sq_m DESC
			LIMIT $6
		)
		SELECT COALESCE(ST_AsMVT(green_data.*, 'green_spaces'), '\\x'::bytea) AS tile
		FROM green_data
		WHERE geom IS NOT NULL
	`, SRID4326, SRID4326, planetPolygonTable)

	var greenTile []byte
	err := r.db.QueryRowContext(ctx, greenQuery, lon, lat, radiusMeters, MVTExtent, MVTBuffer, LimitGreenSpaces).Scan(&greenTile)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("failed to build osm green spaces radius tile", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	// Пляжи
	beachesQuery := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		),
		circle AS (
			SELECT ST_Buffer(point.geom, $3)::geometry AS geom
			FROM point
		),
		beach_data AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(tags->'surface', '') AS surface,
				ST_AsMVTGeom(way, circle.geom, $4, $5, true) AS geom
			FROM %s, circle
			WHERE "natural" = 'beach'
			  AND way && circle.geom
			  AND ST_Intersects(way, circle.geom)
			ORDER BY name
			LIMIT $6
		)
		SELECT COALESCE(ST_AsMVT(beach_data.*, 'beaches'), '\\x'::bytea) AS tile
		FROM beach_data
		WHERE geom IS NOT NULL
	`, SRID4326, planetPolygonTable)

	var beachesTile []byte
	err = r.db.QueryRowContext(ctx, beachesQuery, lon, lat, radiusMeters, MVTExtent, MVTBuffer, LimitBeaches).Scan(&beachesTile)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("failed to build osm beaches radius tile", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	// Объединяем тайлы
	result := append(greenTile, beachesTile...)
	return result, nil
}
