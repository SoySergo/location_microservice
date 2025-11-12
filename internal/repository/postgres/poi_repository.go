package postgres

import (
	"context"
	"database/sql"
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

func (r *poiRepository) GetByID(ctx context.Context, id string) (*domain.POI, error) {
	query := `
		SELECT 
			id, osm_id, name, name_en, name_es, name_ca, name_ru, name_uk,
			name_fr, name_pt, name_it, name_de, category, subcategory,
			lat, lon, address, phone, website, opening_hours, wheelchair, tags
		FROM pois
		WHERE id = $1
	`

	var poi domain.POI
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&poi.ID, &poi.OSMId, &poi.Name,
		&poi.NameEn, &poi.NameEs, &poi.NameCa, &poi.NameRu, &poi.NameUk,
		&poi.NameFr, &poi.NamePt, &poi.NameIt, &poi.NameDe,
		&poi.Category, &poi.Subcategory, &poi.Lat, &poi.Lon,
		&poi.Address, &poi.Phone, &poi.Website, &poi.OpeningHours,
		&poi.Wheelchair, &poi.Tags,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("Failed to get POI by ID", zap.String("id", id), zap.Error(err))
		return nil, errors.ErrDatabaseError
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
		WHERE ST_DWithin(geometry::geography, point.geom, $3 * 1000)
	`

	args := []interface{}{lon, lat, radiusKm}
	argIdx := 4

	if len(categories) > 0 {
		query += fmt.Sprintf(" AND category = ANY($%d)", argIdx)
		args = append(args, pq.Array(categories))
		argIdx++
	}

	query += " ORDER BY distance LIMIT 100"

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

func (r *poiRepository) GetSubcategories(ctx context.Context, categoryID string) ([]*domain.POISubcategory, error) {
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
