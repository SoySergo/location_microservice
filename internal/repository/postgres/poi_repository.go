package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/pkg/errors"
	"go.uber.org/zap"
)

type poiRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewPOIRepository(db *DB) repository.POIRepository {
	return &poiRepository{
		db:     db.DB,
		logger: db.logger,
	}
}

func (r *poiRepository) GetByID(ctx context.Context, id int64) (*domain.POI, error) {
	query := `
		SELECT 
			id, osm_id, name, name_en, name_es, name_ca, name_ru, name_uk,
			name_fr, name_pt, name_it, name_de, category, subcategory,
			lat, lon, address, phone, website, opening_hours, wheelchair, tags
		FROM pois
		WHERE id = $1
	`

	var poi domain.POI
	var tagsJSON []byte
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&poi.ID, &poi.OSMId, &poi.Name,
		&poi.NameEn, &poi.NameEs, &poi.NameCa, &poi.NameRu, &poi.NameUk,
		&poi.NameFr, &poi.NamePt, &poi.NameIt, &poi.NameDe,
		&poi.Category, &poi.Subcategory, &poi.Lat, &poi.Lon,
		&poi.Address, &poi.Phone, &poi.Website, &poi.OpeningHours,
		&poi.Wheelchair, &tagsJSON,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("Failed to get POI by ID", zap.Int64("id", id), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	// Unmarshal tags JSON if present
	if len(tagsJSON) > 0 {
		tags := make(map[string]string)
		if err := json.Unmarshal(tagsJSON, &tags); err != nil {
			r.logger.Warn("Failed to unmarshal tags", zap.Int64("id", id), zap.Error(err))
		} else {
			poi.Tags = tags
		}
	}

	return &poi, nil
}

func (r *poiRepository) GetNearby(
	ctx context.Context,
	lat, lon, radiusKm float64,
	categories []string,
) ([]*domain.POI, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS geom
		)
		SELECT 
			id, osm_id, name, category, subcategory, lat, lon,
			ST_Distance(geometry::geography, point.geom) AS distance
		FROM pois, point
		WHERE ST_DWithin(geometry::geography, point.geom, $3)
	`

	// Convert radius from km to meters
	radiusMeters := radiusKm * 1000
	args := []interface{}{lon, lat, radiusMeters}
	argIdx := 4

	if len(categories) > 0 {
		query += fmt.Sprintf(" AND category = ANY($%d)", argIdx)
		args = append(args, pq.Array(categories))
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY distance LIMIT $%d", argIdx)
	args = append(args, LimitPOIs)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to get nearby POIs", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var pois []*domain.POI
	for rows.Next() {
		var p domain.POI
		var distance float64

		err := rows.Scan(
			&p.ID, &p.OSMId, &p.Name, &p.Category, &p.Subcategory,
			&p.Lat, &p.Lon, &distance,
		)
		if err != nil {
			r.logger.Error("Failed to scan POI", zap.Error(err))
			continue
		}

		pois = append(pois, &p)
	}

	return pois, nil
}

func (r *poiRepository) Search(
	ctx context.Context,
	query string,
	categories []string,
	limit int,
) ([]*domain.POI, error) {
	sqlQuery := `
		SELECT 
			id, osm_id, name, name_en, category, subcategory, lat, lon, address
		FROM pois
		WHERE search_vector @@ plainto_tsquery('simple', $1)
	`

	args := []interface{}{query}
	argIdx := 2

	if len(categories) > 0 {
		sqlQuery += fmt.Sprintf(" AND category = ANY($%d)", argIdx)
		args = append(args, pq.Array(categories))
		argIdx++
	}

	sqlQuery += fmt.Sprintf(" ORDER BY ts_rank(search_vector, plainto_tsquery('simple', $1)) DESC LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		r.logger.Error("Failed to search POIs", zap.String("query", query), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var pois []*domain.POI
	for rows.Next() {
		var p domain.POI
		err := rows.Scan(&p.ID, &p.OSMId, &p.Name, &p.NameEn, &p.Category, &p.Subcategory,
			&p.Lat, &p.Lon, &p.Address)
		if err != nil {
			continue
		}
		pois = append(pois, &p)
	}

	return pois, nil
}

func (r *poiRepository) GetByCategory(
	ctx context.Context,
	category string,
	limit int,
) ([]*domain.POI, error) {
	query := `
		SELECT 
			id, osm_id, name, name_en, category, subcategory, lat, lon
		FROM pois
		WHERE category = $1
		ORDER BY name
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, category, limit)
	if err != nil {
		r.logger.Error("Failed to get POIs by category", zap.String("category", category), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var pois []*domain.POI
	for rows.Next() {
		var p domain.POI
		err := rows.Scan(&p.ID, &p.OSMId, &p.Name, &p.NameEn, &p.Category, &p.Subcategory, &p.Lat, &p.Lon)
		if err != nil {
			continue
		}
		pois = append(pois, &p)
	}

	return pois, nil
}

func (r *poiRepository) SearchByRadius(
	ctx context.Context,
	lat, lon, radiusKm float64,
	categories []string,
	limit int,
) ([]*domain.POI, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS geom
		)
		SELECT 
			id, osm_id, name, category, subcategory, lat, lon,
			ST_Distance(geometry::geography, point.geom) AS distance
		FROM pois, point
		WHERE ST_DWithin(geometry::geography, point.geom, $3 * 1000)
	`

	args := []interface{}{lon, lat, radiusKm}
	argIdx := 4

	if len(categories) > 0 {
		query += fmt.Sprintf(" AND category = ANY($%d)", argIdx)
		args = append(args, pq.Array(categories))
		argIdx++
	}

	query += fmt.Sprintf(" ORDER BY distance LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to search POIs by radius", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var pois []*domain.POI
	for rows.Next() {
		var p domain.POI
		var distance float64

		err := rows.Scan(
			&p.ID, &p.OSMId, &p.Name, &p.Category, &p.Subcategory,
			&p.Lat, &p.Lon, &distance,
		)
		if err != nil {
			r.logger.Error("Failed to scan POI", zap.Error(err))
			continue
		}

		pois = append(pois, &p)
	}

	return pois, nil
}

func (r *poiRepository) SearchByBoundary(
	ctx context.Context,
	boundaryID string,
	categories []string,
	limit int,
) ([]*domain.POI, error) {
	query := `
		SELECT 
			p.id, p.osm_id, p.name, p.category, p.subcategory, p.lat, p.lon
		FROM pois p
		JOIN admin_boundaries b ON ST_Contains(b.geometry, p.geometry)
		WHERE b.id = $1
	`

	args := []interface{}{boundaryID}
	argIdx := 2

	if len(categories) > 0 {
		query += fmt.Sprintf(" AND p.category = ANY($%d)", argIdx)
		args = append(args, pq.Array(categories))
		argIdx++
	}

	query += fmt.Sprintf(" LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to search POIs by boundary", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var pois []*domain.POI
	for rows.Next() {
		var p domain.POI
		err := rows.Scan(&p.ID, &p.OSMId, &p.Name, &p.Category, &p.Subcategory, &p.Lat, &p.Lon)
		if err != nil {
			continue
		}
		pois = append(pois, &p)
	}

	return pois, nil
}

func (r *poiRepository) GetCategories(ctx context.Context) ([]*domain.POICategory, error) {
	query := `
		SELECT id, code, name_en, icon, color, sort_order
		FROM poi_categories
		ORDER BY sort_order, code
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		r.logger.Error("Failed to get POI categories", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var categories []*domain.POICategory
	for rows.Next() {
		var c domain.POICategory
		err := rows.Scan(&c.ID, &c.Code, &c.NameEn, &c.Icon, &c.Color, &c.SortOrder)
		if err != nil {
			continue
		}
		categories = append(categories, &c)
	}

	return categories, nil
}

func (r *poiRepository) GetSubcategories(ctx context.Context, categoryID int64) ([]*domain.POISubcategory, error) {
	query := `
		SELECT id, category_id, code, name_en, icon, sort_order
		FROM poi_subcategories
		WHERE category_id = $1
		ORDER BY sort_order, code
	`

	rows, err := r.db.QueryContext(ctx, query, categoryID)
	if err != nil {
		r.logger.Error("Failed to get POI subcategories", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var subcategories []*domain.POISubcategory
	for rows.Next() {
		var s domain.POISubcategory
		err := rows.Scan(&s.ID, &s.CategoryID, &s.Code, &s.NameEn, &s.Icon, &s.SortOrder)
		if err != nil {
			continue
		}
		subcategories = append(subcategories, &s)
	}

	return subcategories, nil
}

// GetPOIRadiusTile генерирует MVT тайл с POI в радиусе от точки
func (r *poiRepository) GetPOIRadiusTile(ctx context.Context, lat, lon, radiusKm float64, categories []string) ([]byte, error) {
	// Convert radius from km to meters
	radiusMeters := radiusKm * 1000

	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS geom
		),
		circle AS (
			SELECT ST_Buffer(point.geom, $3)::geometry AS geom
			FROM point
		),
		mvt_geom AS (
			SELECT 
				p.id, p.name, p.category, p.subcategory,
				ST_AsMVTGeom(
					p.geometry,
					circle.geom,
					$4,
					$5,
					true
				) AS geom
			FROM pois p, circle
			WHERE p.geometry && circle.geom
			  AND ST_Contains(circle.geom, p.geometry)
	`

	args := []interface{}{lon, lat, radiusMeters, MVTExtent, MVTBuffer}

	limitArg := "$6"
	if len(categories) > 0 {
		query += " AND p.category = ANY($6)"
		args = append(args, pq.Array(categories))
		limitArg = "$7"
	}

	query += fmt.Sprintf(`
			ORDER BY p.name
			LIMIT %s
		)
		SELECT ST_AsMVT(mvt_geom.*, 'pois') AS tile
		FROM mvt_geom
		WHERE geom IS NOT NULL
	`, limitArg)

	args = append(args, LimitPOIsRadius)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("Failed to generate POI radius tile",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon),
			zap.Float64("radius_km", radiusKm),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	return tile, nil
}

// GetPOITile генерирует MVT тайл с POI для заданных координат тайла
func (r *poiRepository) GetPOITile(ctx context.Context, z, x, y int, categories []string) ([]byte, error) {
	query := `
		WITH 
		bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		mvt_geom AS (
			SELECT 
				p.id, 
				p.name, 
				p.category, 
				p.subcategory,
				ST_AsMVTGeom(
					p.geometry,
					bounds.geom,
					$4,
					$5,
					true
				) AS geom
			FROM pois p, bounds
			WHERE p.geometry && bounds.geom
	`

	args := []interface{}{z, x, y, MVTExtent, MVTBuffer}

	// Фильтрация по категориям если указаны
	if len(categories) > 0 {
		query += " AND p.category = ANY($6)"
		args = append(args, pq.Array(categories))
	}

	// Адаптивная фильтрация и лимит по zoom level
	poiLimit := getPOILimitByZoom(z)
	query += fmt.Sprintf(`
			ORDER BY 
				CASE p.category 
					WHEN 'landmark' THEN 1
					WHEN 'tourism' THEN 2
					WHEN 'restaurant' THEN 3
					WHEN 'hotel' THEN 4
					ELSE 5
				END,
				p.name
			LIMIT %d
		)
		SELECT ST_AsMVT(mvt_geom.*, 'pois') AS tile
		FROM mvt_geom
		WHERE geom IS NOT NULL
	`, poiLimit)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("Failed to generate POI tile",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	return tile, nil
}

// GetPOIByBoundaryTile генерирует MVT тайл с POI внутри административной границы
func (r *poiRepository) GetPOIByBoundaryTile(ctx context.Context, boundaryID int64, categories []string) ([]byte, error) {
	query := `
		WITH 
		boundary AS (
			SELECT geometry FROM admin_boundaries WHERE id = $1
		),
		mvt_geom AS (
			SELECT 
				p.id,
				p.name,
				p.category,
				p.subcategory,
				ST_AsMVTGeom(
					p.geometry,
					b.geometry,
					$2,
					$3,
					true
				) AS geom
			FROM pois p, boundary b
			WHERE p.geometry && b.geometry
			  AND ST_Contains(b.geometry, p.geometry)
	`

	args := []interface{}{boundaryID, MVTExtent, MVTBuffer}

	// Фильтрация по категориям если указаны
	if len(categories) > 0 {
		query += " AND p.category = ANY($4)"
		args = append(args, pq.Array(categories))
	}

	query += fmt.Sprintf(`
			ORDER BY 
				CASE p.category 
					WHEN 'landmark' THEN 1
					WHEN 'tourism' THEN 2
					WHEN 'restaurant' THEN 3
					WHEN 'hotel' THEN 4
					ELSE 5
				END,
				p.name
			LIMIT %d
		)
		SELECT ST_AsMVT(mvt_geom.*, 'pois') AS tile
		FROM mvt_geom
		WHERE geom IS NOT NULL
	`, LimitPOIsCategory)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("Failed to generate POI boundary tile",
			zap.Int64("boundary_id", boundaryID),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	return tile, nil
}
