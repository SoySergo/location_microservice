package postgresosm

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	pkgerrors "github.com/location-microservice/internal/pkg/errors"
	"go.uber.org/zap"
)

var (
	poiSelectFull = fmt.Sprintf(`
		SELECT
			osm_id,
			COALESCE(name, '') AS name,
			%s AS category,
			%s AS subcategory,
			ST_Y(ST_Transform(way, %d)) AS lat,
			ST_X(ST_Transform(way, %d)) AS lon,
			ST_AsBinary(ST_Transform(way, %d)) AS geometry,
			COALESCE(hstore_to_json(tags), '{}'::json)::text AS tags_json
		FROM %s
	`, categoryExpr, subcategoryExpr, SRID4326, SRID4326, SRID4326, planetPointTable)

	poiSelectLite = fmt.Sprintf(`
		SELECT
			osm_id,
			COALESCE(name, '') AS name,
			%s AS category,
			%s AS subcategory,
			ST_Y(ST_Transform(way, %d)) AS lat,
			ST_X(ST_Transform(way, %d)) AS lon,
			way
		FROM %s
	`, categoryExpr, subcategoryExpr, SRID4326, SRID4326, planetPointTable)
)

type poiRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

type poiRow struct {
	OSMID       int64   `db:"osm_id"`
	Name        string  `db:"name"`
	Category    string  `db:"category"`
	Subcategory string  `db:"subcategory"`
	Lat         float64 `db:"lat"`
	Lon         float64 `db:"lon"`
	Geometry    []byte  `db:"geometry"`
	TagsJSON    []byte  `db:"tags_json"`
}

type poiShortRow struct {
	OSMID       int64   `db:"osm_id"`
	Name        string  `db:"name"`
	Category    string  `db:"category"`
	Subcategory string  `db:"subcategory"`
	Lat         float64 `db:"lat"`
	Lon         float64 `db:"lon"`
}

type poiDistanceRow struct {
	poiShortRow
	Distance float64 `db:"distance"`
}

func (r poiShortRow) toDomain() *domain.POI {
	return &domain.POI{
		ID:          r.OSMID,
		OSMId:       r.OSMID,
		Name:        ensureName(r.Name, r.Category, r.OSMID),
		Category:    r.Category,
		Subcategory: r.Subcategory,
		Lat:         r.Lat,
		Lon:         r.Lon,
	}
}

// NewPOIRepository создает репозиторий POI для OSM базы данных
func NewPOIRepository(db *DB) repository.POIRepository {
	return &poiRepository{
		db:     db.DB,
		logger: db.logger,
	}
}

func (r *poiRepository) GetByID(ctx context.Context, id int64) (*domain.POI, error) {
	query := poiSelectFull + " WHERE osm_id = $1 LIMIT 1"

	var row poiRow
	err := r.db.QueryRowxContext(ctx, query, id).StructScan(&row)
	if err == sql.ErrNoRows {
		return nil, pkgerrors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("failed to get osm poi", zap.Int64("osm_id", id), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return parsePOIFromRow(&row), nil
}

func (r *poiRepository) GetNearby(ctx context.Context, lat, lon, radiusKm float64, categories []string) ([]*domain.POI, error) {
	if radiusKm <= 0 {
		radiusKm = 1
	}

	radiusMeters := radiusKm * 1000

	base := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		), data AS (
			SELECT *, ST_Transform(way, %d) AS w4326 FROM (
				%s
			) src
		)
		SELECT
			osm_id,
			name,
			category,
			subcategory,
			ST_Y(w4326) AS lat,
			ST_X(w4326) AS lon,
			ST_Distance(w4326::geography, point.geom) AS distance
		FROM data, point
		WHERE ST_DWithin(w4326::geography, point.geom, $3)
	`, SRID4326, SRID4326, poiSelectLite)

	args := []interface{}{lon, lat, radiusMeters}
	argIdx := 4

	if len(categories) > 0 {
		base += fmt.Sprintf(" AND category = ANY($%d)", argIdx)
		args = append(args, pq.Array(categories))
		argIdx++
	}

	base += fmt.Sprintf(" ORDER BY distance LIMIT $%d", argIdx)
	args = append(args, LimitPOIs)

	rows, err := r.db.QueryxContext(ctx, base, args...)
	if err != nil {
		r.logger.Error("failed to query nearby osm pois", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var result []*domain.POI
	for rows.Next() {
		var row poiDistanceRow
		if err := rows.StructScan(&row); err != nil {
			r.logger.Error("failed to scan poi row", zap.Error(err))
			continue
		}
		result = append(result, row.poiShortRow.toDomain())
	}

	return result, nil
}

func (r *poiRepository) Search(ctx context.Context, query string, categories []string, limit int) ([]*domain.POI, error) {
	if limit <= 0 {
		limit = LimitPOIs
	}
	if limit > LimitPOIsCategory {
		limit = LimitPOIsCategory
	}

	searchSQL := fmt.Sprintf(`
		SELECT
			osm_id,
			name,
			category,
			subcategory,
			lat,
			lon,
			rank
		FROM (
			SELECT
				osm_id,
				COALESCE(name, '') AS name,
				category,
				subcategory,
				lat,
				lon,
				to_tsvector('simple', COALESCE(name, '')) AS document,
				plainto_tsquery('simple', $1) AS query_ts,
				COALESCE(ts_rank_cd(to_tsvector('simple', COALESCE(name, '')), plainto_tsquery('simple', $1)), 0) AS rank
			FROM (
				%s
			) data
		) ranked
		WHERE (ranked.document @@ ranked.query_ts) OR ranked.name ILIKE '%%' || $1 || '%%'
	`, poiSelectLite)

	args := []interface{}{query}
	argIdx := 2

	if len(categories) > 0 {
		searchSQL += fmt.Sprintf(" AND ranked.category = ANY($%d)", argIdx)
		args = append(args, pq.Array(categories))
		argIdx++
	}

	searchSQL += fmt.Sprintf(" ORDER BY rank DESC, name LIMIT $%d", argIdx)
	args = append(args, limit)

	rows, err := r.db.QueryxContext(ctx, searchSQL, args...)
	if err != nil {
		r.logger.Error("failed to search osm pois", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var result []*domain.POI
	for rows.Next() {
		var row struct {
			poiShortRow
			Rank float64 `db:"rank"`
		}
		if err := rows.StructScan(&row); err != nil {
			r.logger.Error("failed to scan search row", zap.Error(err))
			continue
		}
		result = append(result, row.poiShortRow.toDomain())
	}

	return result, nil
}

func (r *poiRepository) GetByCategory(ctx context.Context, category string, limit int) ([]*domain.POI, error) {
	if limit <= 0 {
		limit = LimitPOIs
	}

	query := fmt.Sprintf(`
		SELECT
			osm_id,
			name,
			category,
			subcategory,
			lat,
			lon
		FROM (
			%s
		) data
		WHERE category = $1
		ORDER BY name
		LIMIT $2
	`, poiSelectLite)

	rows, err := r.db.QueryxContext(ctx, query, category, limit)
	if err != nil {
		r.logger.Error("failed to get osm pois by category", zap.String("category", category), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var result []*domain.POI
	for rows.Next() {
		var row poiShortRow
		if err := rows.StructScan(&row); err != nil {
			continue
		}
		result = append(result, row.toDomain())
	}

	return result, nil
}

func (r *poiRepository) GetCategories(ctx context.Context) ([]*domain.POICategory, error) {
	query := fmt.Sprintf(`
		SELECT DISTINCT category
		FROM (
			%s
		) data
		WHERE category IS NOT NULL AND category <> ''
		ORDER BY category
	`, poiSelectLite)

	rows, err := r.db.QueryxContext(ctx, query)
	if err != nil {
		r.logger.Error("failed to list osm categories", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var categories []*domain.POICategory
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			continue
		}
		id := hashCategory(code)
		categories = append(categories, &domain.POICategory{
			ID:        id,
			Code:      code,
			NameEn:    code,
			NameEs:    code,
			NameCa:    code,
			NameRu:    code,
			NameUk:    code,
			NameFr:    code,
			NamePt:    code,
			NameIt:    code,
			NameDe:    code,
			SortOrder: len(categories) + 1,
		})
	}

	return categories, nil
}

func (r *poiRepository) GetSubcategories(ctx context.Context, categoryID int64) ([]*domain.POISubcategory, error) {
	code, err := r.resolveCategoryCode(ctx, categoryID)
	if err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT subcategory
		FROM (
			%s
		) data
		WHERE category = $1
		ORDER BY subcategory
	`, poiSelectLite)

	rows, err := r.db.QueryxContext(ctx, query, code)
	if err != nil {
		r.logger.Error("failed to list osm subcategories", zap.String("category", code), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var subcategories []*domain.POISubcategory
	idx := 1
	for rows.Next() {
		var subcode string
		if err := rows.Scan(&subcode); err != nil {
			continue
		}
		subcategories = append(subcategories, &domain.POISubcategory{
			ID:         hashCategory(code + ":" + subcode),
			CategoryID: categoryID,
			Code:       subcode,
			NameEn:     subcode,
			NameEs:     subcode,
			NameCa:     subcode,
			NameRu:     subcode,
			NameUk:     subcode,
			NameFr:     subcode,
			NamePt:     subcode,
			NameIt:     subcode,
			NameDe:     subcode,
			SortOrder:  idx,
		})
		idx++
	}

	return subcategories, nil
}

func (r *poiRepository) GetPOITile(ctx context.Context, z, x, y int, categories []string) ([]byte, error) {
	limit := getPOILimitByZoom(z)
	categoryFilter := ""
	argOffset := 6
	args := []interface{}{z, x, y, MVTExtent, MVTBuffer}
	if len(categories) > 0 {
		categoryFilter = fmt.Sprintf(" AND category = ANY($%d)", argOffset)
		args = append(args, pq.Array(categories))
	}

	query := fmt.Sprintf(`
		WITH bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		data AS (
			SELECT osm_id, name, category, subcategory, way
			FROM (
				%s
			) src
			WHERE way && (SELECT geom FROM bounds)%s
		),
		mvt_geom AS (
			SELECT
				osm_id AS id,
				name,
				category,
				subcategory,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM data, bounds
			WHERE way && bounds.geom
			ORDER BY category, name
			LIMIT %d
		)
		SELECT COALESCE(ST_AsMVT(mvt_geom.*, 'pois'), '\\x') AS tile
		FROM mvt_geom
		WHERE geom IS NOT NULL
	`, poiSelectLite, categoryFilter, limit)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("failed to build osm poi tile", zap.Int("z", z), zap.Int("x", x), zap.Int("y", y), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return tile, nil
}

func (r *poiRepository) GetPOIRadiusTile(ctx context.Context, lat, lon, radiusKm float64, categories []string) ([]byte, error) {
	if radiusKm <= 0 {
		radiusKm = 1
	}

	radiusMeters := radiusKm * 1000

	categoryFilter := ""
	args := []interface{}{lon, lat, radiusMeters, MVTExtent, MVTBuffer}
	if len(categories) > 0 {
		categoryFilter = " AND category = ANY($6)"
		args = append(args, pq.Array(categories))
	}

	query := fmt.Sprintf(`
		WITH center AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		), circle AS (
			SELECT ST_Buffer(center.geom, $3)::geometry AS geom
			FROM center
		), data AS (
			SELECT osm_id, name, category, subcategory, way
			FROM (
				%s
			) src
		), mvt_geom AS (
			SELECT
				osm_id AS id,
				name,
				category,
				subcategory,
				ST_AsMVTGeom(
					way,
					ST_Transform(circle.geom, %d),
					$4,
					$5,
					true
				) AS geom
			FROM data, circle, center
			WHERE ST_DWithin(ST_Transform(way, %d)::geography, center.geom, $3)%s
			ORDER BY category, name
			LIMIT %d
		)
		SELECT COALESCE(ST_AsMVT(mvt_geom.*, 'pois'), '\\x') AS tile
		FROM mvt_geom
		WHERE geom IS NOT NULL
	`, SRID4326, poiSelectLite, SRID3857, SRID4326, categoryFilter, LimitPOIsRadius)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("failed to build osm poi radius tile", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return tile, nil
}

func (r *poiRepository) GetPOIByBoundaryTile(ctx context.Context, boundaryID int64, categories []string) ([]byte, error) {
	categoryFilter := ""
	args := []interface{}{boundaryID, MVTExtent, MVTBuffer}
	if len(categories) > 0 {
		categoryFilter = " AND data.category = ANY($4)"
		args = append(args, pq.Array(categories))
	}

	query := fmt.Sprintf(`
		WITH boundary AS (
			SELECT way FROM %s WHERE osm_id = $1 AND boundary = 'administrative'
		), data AS (
			SELECT osm_id, name, category, subcategory, way
			FROM (
				%s
			) src
		), mvt_geom AS (
			SELECT
				data.osm_id AS id,
				data.name,
				data.category,
				data.subcategory,
				ST_AsMVTGeom(data.way, boundary.way, $2, $3, true) AS geom
			FROM data, boundary
			WHERE boundary.way IS NOT NULL
				AND ST_Contains(boundary.way, data.way)%s
			ORDER BY data.category, data.name
			LIMIT %d
		)
		SELECT COALESCE(ST_AsMVT(mvt_geom.*, 'pois'), '\\x') AS tile
		FROM mvt_geom
		WHERE geom IS NOT NULL
	`, planetPolygonTable, poiSelectLite, categoryFilter, LimitPOIsCategory)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("failed to build osm poi boundary tile", zap.Int64("boundary", boundaryID), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return tile, nil
}

func (r *poiRepository) resolveCategoryCode(ctx context.Context, categoryID int64) (string, error) {
	categories, err := r.GetCategories(ctx)
	if err != nil {
		return "", err
	}
	for _, cat := range categories {
		if cat.ID == categoryID {
			return cat.Code, nil
		}
	}
	r.logger.Warn("category not found for id", zap.Int64("category_id", categoryID))
	return "", pkgerrors.ErrLocationNotFound
}

// GetPOITileByCategories генерирует MVT тайл с POI по координатам тайла с фильтрацией по категориям и подкатегориям
func (r *poiRepository) GetPOITileByCategories(ctx context.Context, z, x, y int, categories, subcategories []string) ([]byte, error) {
	limit := getPOILimitByZoom(z)
	args := []interface{}{z, x, y, MVTExtent, MVTBuffer}
	argOffset := 6

	var filters []string
	if len(categories) > 0 {
		filters = append(filters, fmt.Sprintf("category = ANY($%d)", argOffset))
		args = append(args, pq.Array(categories))
		argOffset++
	}
	if len(subcategories) > 0 {
		filters = append(filters, fmt.Sprintf("subcategory = ANY($%d)", argOffset))
		args = append(args, pq.Array(subcategories))
		argOffset++
	}

	filterClause := ""
	if len(filters) > 0 {
		filterClause = " AND (" + strings.Join(filters, " OR ") + ")"
	}

	query := fmt.Sprintf(`
		WITH bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		data AS (
			SELECT osm_id, name, category, subcategory, way
			FROM (
				%s
			) src
			WHERE way && (SELECT geom FROM bounds)%s
		),
		mvt_geom AS (
			SELECT
				osm_id AS id,
				name,
				category,
				subcategory,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM data, bounds
			WHERE way && bounds.geom
			ORDER BY category, name
			LIMIT %d
		)
		SELECT COALESCE(ST_AsMVT(mvt_geom.*, 'pois'), '\\x') AS tile
		FROM mvt_geom
		WHERE geom IS NOT NULL
	`, poiSelectLite, filterClause, limit)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("failed to build osm poi tile by categories",
			zap.Int("z", z), zap.Int("x", x), zap.Int("y", y),
			zap.Strings("categories", categories),
			zap.Strings("subcategories", subcategories),
			zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return tile, nil
}
