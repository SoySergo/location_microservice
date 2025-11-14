package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/pkg/errors"
)

type boundaryRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewBoundaryRepository создает новый экземпляр BoundaryRepository
func NewBoundaryRepository(db *DB) repository.BoundaryRepository {
	return &boundaryRepository{
		db:     db.DB,
		logger: db.logger,
	}
}

// GetByID возвращает административную границу по ID
func (r *boundaryRepository) GetByID(ctx context.Context, id int64) (*domain.AdminBoundary, error) {
	query := `
		SELECT 
			id, osm_id, name, 
			COALESCE(name_en, ''), COALESCE(name_es, ''), COALESCE(name_ca, ''), 
			COALESCE(name_ru, ''), COALESCE(name_uk, ''), COALESCE(name_fr, ''), 
			COALESCE(name_pt, ''), COALESCE(name_it, ''), COALESCE(name_de, ''), 
			type, admin_level,
			center_lat, center_lon, parent_id, population, area_sq_km,
			tags, created_at, updated_at,
			ST_AsGeoJSON(geometry) as geometry_json
		FROM admin_boundaries
		WHERE id = $1
	`

	var boundary domain.AdminBoundary
	var geojson string
	var tags []byte

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&boundary.ID, &boundary.OSMId, &boundary.Name,
		&boundary.NameEn, &boundary.NameEs, &boundary.NameCa,
		&boundary.NameRu, &boundary.NameUk, &boundary.NameFr,
		&boundary.NamePt, &boundary.NameIt, &boundary.NameDe,
		&boundary.Type, &boundary.AdminLevel,
		&boundary.CenterLat, &boundary.CenterLon,
		&boundary.ParentID, &boundary.Population, &boundary.AreaSqKm,
		&tags, &boundary.CreatedAt, &boundary.UpdatedAt,
		&geojson,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("Failed to get boundary by ID", zap.Int64("id", id), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return &boundary, nil
}

// SearchByText выполняет текстовый поиск по названиям границ с поддержкой языков
func (r *boundaryRepository) SearchByText(
	ctx context.Context,
	query string,
	lang string,
	adminLevels []int,
	limit int,
) ([]*domain.AdminBoundary, error) {
	// Определяем поле для названия в зависимости от языка
	nameField := "name"
	if lang != "" {
		nameField = fmt.Sprintf("COALESCE(name_%s, name)", lang)
	}

	sqlQuery := fmt.Sprintf(`
		SELECT 
			id, osm_id, %s as name, type, admin_level,
			center_lat, center_lon, area_sq_km,
			ts_rank(search_vector, plainto_tsquery('simple', $1)) AS rank
		FROM admin_boundaries
		WHERE search_vector @@ plainto_tsquery('simple', $1)
	`, nameField)

	args := []interface{}{query}
	argIndex := 2

	// Фильтр по административным уровням
	if len(adminLevels) > 0 {
		placeholders := ""
		for i, level := range adminLevels {
			if i > 0 {
				placeholders += ","
			}
			placeholders += fmt.Sprintf("$%d", argIndex)
			args = append(args, level)
			argIndex++
		}
		sqlQuery += fmt.Sprintf(" AND admin_level IN (%s)", placeholders)
	}

	sqlQuery += " ORDER BY rank DESC, admin_level ASC"

	// Ограничение результатов
	if limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
	}

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		r.logger.Error("Failed to search boundaries", zap.String("query", query), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var boundaries []*domain.AdminBoundary
	for rows.Next() {
		var b domain.AdminBoundary
		var rank float64

		err := rows.Scan(
			&b.ID, &b.OSMId, &b.Name, &b.Type, &b.AdminLevel,
			&b.CenterLat, &b.CenterLon, &b.AreaSqKm, &rank,
		)
		if err != nil {
			r.logger.Error("Failed to scan boundary", zap.Error(err))
			continue
		}

		boundaries = append(boundaries, &b)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating boundary rows", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return boundaries, nil
}

// ReverseGeocode возвращает адрес по координатам
func (r *boundaryRepository) ReverseGeocode(
	ctx context.Context,
	lat, lon float64,
) (*domain.Address, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326) AS geom
		)
		SELECT 
			MAX(CASE WHEN admin_level = 2 THEN name END) AS country,
			MAX(CASE WHEN admin_level = 4 THEN name END) AS region,
			MAX(CASE WHEN admin_level = 6 THEN name END) AS province,
			MAX(CASE WHEN admin_level = 8 THEN name END) AS city,
			MAX(CASE WHEN admin_level = 9 THEN name END) AS district
		FROM admin_boundaries, point
		WHERE geometry && ST_Expand(point.geom, $3)
		  AND ST_Contains(geometry, point.geom)
	`

	var country, region, province, city, district sql.NullString

	err := r.db.QueryRowContext(ctx, query, lon, lat, BoundaryExpansionDegrees).Scan(
		&country, &region, &province, &city, &district,
	)

	if err == sql.ErrNoRows || (!country.Valid && !region.Valid && !province.Valid && !city.Valid) {
		return nil, errors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("Failed to reverse geocode",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	addr := &domain.Address{
		Country:  country.String,
		Region:   region.String,
		Province: province.String,
		City:     city.String,
	}

	if district.Valid {
		addr.District = &district.String
	}

	return addr, nil
}

// GetTile генерирует MVT тайл для административных границ
func (r *boundaryRepository) GetTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Определяем уровни административных границ в зависимости от zoom
	adminLevels := getAdminLevelsString(z)

	query := fmt.Sprintf(`
		WITH 
		bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		mvt_geom AS (
			SELECT 
				id, name, type, admin_level,
				ST_AsMVTGeom(
					geometry,
					bounds.geom,
					$4,
					$5,
					true
				) AS geom
			FROM admin_boundaries, bounds
			WHERE geometry && bounds.geom
			  AND admin_level IN (%s)
		)
		SELECT ST_AsMVT(mvt_geom.*, 'boundaries') AS tile
		FROM mvt_geom
		WHERE geom IS NOT NULL
	`, adminLevels)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y, MVTExtent, MVTBuffer).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil // Пустой тайл
	}
	if err != nil {
		r.logger.Error("Failed to generate boundary tile",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	return tile, nil
}

// GetByPoint возвращает административные границы для точки
func (r *boundaryRepository) GetByPoint(ctx context.Context, lat, lon float64) ([]*domain.AdminBoundary, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326) AS geom
		)
		SELECT 
			id, osm_id, name, name_en, type, admin_level,
			center_lat, center_lon, area_sq_km
		FROM admin_boundaries, point
		WHERE geometry && ST_Expand(point.geom, $3)
		  AND ST_Contains(geometry, point.geom)
		ORDER BY admin_level ASC
	`

	rows, err := r.db.QueryContext(ctx, query, lon, lat, BoundaryExpansionDegrees)
	if err != nil {
		r.logger.Error("Failed to get boundaries by point",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var boundaries []*domain.AdminBoundary
	for rows.Next() {
		var b domain.AdminBoundary
		err := rows.Scan(
			&b.ID, &b.OSMId, &b.Name, &b.NameEn, &b.Type, &b.AdminLevel,
			&b.CenterLat, &b.CenterLon, &b.AreaSqKm,
		)
		if err != nil {
			r.logger.Error("Failed to scan boundary", zap.Error(err))
			continue
		}
		boundaries = append(boundaries, &b)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating boundary rows", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return boundaries, nil
}

// Search выполняет простой текстовый поиск по названиям границ
func (r *boundaryRepository) Search(ctx context.Context, query string, limit int) ([]*domain.AdminBoundary, error) {
	return r.SearchByText(ctx, query, "", nil, limit)
}

// GetChildren возвращает дочерние границы для родительской
func (r *boundaryRepository) GetChildren(ctx context.Context, parentID int64) ([]*domain.AdminBoundary, error) {
	query := `
		SELECT 
			id, osm_id, name, name_en, type, admin_level,
			center_lat, center_lon, area_sq_km
		FROM admin_boundaries
		WHERE parent_id = $1
		ORDER BY admin_level ASC, name ASC
	`

	rows, err := r.db.QueryContext(ctx, query, parentID)
	if err != nil {
		r.logger.Error("Failed to get children boundaries",
			zap.Int64("parent_id", parentID),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var boundaries []*domain.AdminBoundary
	for rows.Next() {
		var b domain.AdminBoundary
		err := rows.Scan(
			&b.ID, &b.OSMId, &b.Name, &b.NameEn, &b.Type, &b.AdminLevel,
			&b.CenterLat, &b.CenterLon, &b.AreaSqKm,
		)
		if err != nil {
			r.logger.Error("Failed to scan boundary", zap.Error(err))
			continue
		}
		boundaries = append(boundaries, &b)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating boundary rows", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return boundaries, nil
}

// GetByAdminLevel возвращает границы определенного административного уровня
func (r *boundaryRepository) GetByAdminLevel(ctx context.Context, level int, limit int) ([]*domain.AdminBoundary, error) {
	query := `
		SELECT 
			id, osm_id, name, name_en, type, admin_level,
			center_lat, center_lon, area_sq_km
		FROM admin_boundaries
		WHERE admin_level = $1
		ORDER BY name ASC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, level, limit)
	if err != nil {
		r.logger.Error("Failed to get boundaries by admin level",
			zap.Int("level", level),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var boundaries []*domain.AdminBoundary
	for rows.Next() {
		var b domain.AdminBoundary
		err := rows.Scan(
			&b.ID, &b.OSMId, &b.Name, &b.NameEn, &b.Type, &b.AdminLevel,
			&b.CenterLat, &b.CenterLon, &b.AreaSqKm,
		)
		if err != nil {
			r.logger.Error("Failed to scan boundary", zap.Error(err))
			continue
		}
		boundaries = append(boundaries, &b)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating boundary rows", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return boundaries, nil
}

// GetBoundariesInRadius возвращает границы в радиусе от точки
func (r *boundaryRepository) GetBoundariesInRadius(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.AdminBoundary, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326) AS geom
		),
		circle AS (
			SELECT ST_Buffer(point.geom::geography, $3 * 1000)::geometry AS geom
			FROM point
		)
		SELECT 
			id, osm_id, name, name_en, type, admin_level,
			center_lat, center_lon, area_sq_km,
			ST_AsGeoJSON(ST_Simplify(geometry, 0.0001)) as geometry_json
		FROM admin_boundaries, circle
		WHERE geometry && circle.geom
		  AND ST_Intersects(geometry, circle.geom)
			  AND admin_level IN (6, 8, 9)
		ORDER BY admin_level ASC, area_sq_km ASC
		LIMIT $4
	`

	rows, err := r.db.QueryContext(ctx, query, lon, lat, radiusKm, LimitBoundariesRadius)
	if err != nil {
		r.logger.Error("Failed to get boundaries in radius",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon),
			zap.Float64("radius_km", radiusKm),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var boundaries []*domain.AdminBoundary
	for rows.Next() {
		var b domain.AdminBoundary
		var geojson string

		err := rows.Scan(
			&b.ID, &b.OSMId, &b.Name, &b.NameEn, &b.Type, &b.AdminLevel,
			&b.CenterLat, &b.CenterLon, &b.AreaSqKm, &geojson,
		)
		if err != nil {
			r.logger.Error("Failed to scan boundary", zap.Error(err))
			continue
		}

		boundaries = append(boundaries, &b)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating boundary rows", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return boundaries, nil
}

// GetBoundariesRadiusTile генерирует MVT тайл с границами в радиусе от точки
func (r *boundaryRepository) GetBoundariesRadiusTile(ctx context.Context, lat, lon, radiusKm float64) ([]byte, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326) AS geom
		),
		circle AS (
			SELECT ST_Buffer(point.geom::geography, $3 * 1000)::geometry AS geom
			FROM point
		),
		mvt_geom AS (
			SELECT 
				id, name, type, admin_level,
				ST_AsMVTGeom(
					ST_Simplify(geometry, $4),
					circle.geom,
					$5,
					$6,
					true
				) AS geom
			FROM admin_boundaries, circle
			WHERE geometry && circle.geom
			  AND ST_Intersects(geometry, circle.geom)
			  AND admin_level IN (6, 8, 9)
			ORDER BY admin_level ASC, area_sq_km ASC
			LIMIT $7
		)
		SELECT ST_AsMVT(mvt_geom.*, 'boundaries') AS tile
		FROM mvt_geom
		WHERE geom IS NOT NULL
	`

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, lon, lat, radiusKm, MVTSimplifyTolerance, MVTExtent, MVTBuffer, LimitBoundariesRadius).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("Failed to generate boundaries radius tile",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon),
			zap.Float64("radius_km", radiusKm),
			zap.Error(err),
		)
		return nil, errors.ErrDatabaseError
	}

	return tile, nil
}
