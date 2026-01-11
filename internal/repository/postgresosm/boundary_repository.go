package postgresosm

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	pkgerrors "github.com/location-microservice/internal/pkg/errors"
	"go.uber.org/zap"
)

type boundaryRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewBoundaryRepository создает репозиторий административных границ для OSM базы данных
func NewBoundaryRepository(db *DB) repository.BoundaryRepository {
	return &boundaryRepository{
		db:     db.DB,
		logger: db.logger,
	}
}

// GetByID возвращает административную границу по OSM ID
func (r *boundaryRepository) GetByID(ctx context.Context, id int64) (*domain.AdminBoundary, error) {
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
			COALESCE(boundary, 'administrative') AS type,
			COALESCE(admin_level::integer, 0) AS admin_level,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS center_lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS center_lon,
			COALESCE((tags->'population')::bigint, 0) AS population,
			ST_Area(ST_Transform(way, %d)::geography) / 1000000 AS area_sq_km
		FROM %s
		WHERE osm_id = $1
		  AND boundary = 'administrative'
		  AND admin_level IS NOT NULL
		LIMIT 1
	`, SRID4326, SRID4326, SRID4326, planetPolygonTable)

	var b domain.AdminBoundary
	var population int64
	var adminLevelInt int

	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&b.OSMId, &b.Name,
		&b.NameEn, &b.NameEs, &b.NameCa,
		&b.NameRu, &b.NameUk, &b.NameFr,
		&b.NamePt, &b.NameIt, &b.NameDe,
		&b.Type, &adminLevelInt,
		&b.CenterLat, &b.CenterLon,
		&population, &b.AreaSqKm,
	)

	if err == sql.ErrNoRows {
		return nil, pkgerrors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("failed to get osm boundary", zap.Int64("osm_id", id), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	b.ID = b.OSMId
	b.AdminLevel = adminLevelInt
	if population > 0 {
		populationInt := int(population)
		b.Population = &populationInt
	}

	return &b, nil
}

// SearchByText выполняет текстовый поиск по названиям границ
func (r *boundaryRepository) SearchByText(
	ctx context.Context,
	searchQuery string,
	lang string,
	adminLevels []int,
	limit int,
) ([]*domain.AdminBoundary, error) {
	if limit <= 0 || limit > LimitBoundaries {
		limit = LimitBoundaries
	}

	// Определяем поле для поиска в зависимости от языка
	nameField := "name"
	if lang != "" {
		nameField = fmt.Sprintf("COALESCE(NULLIF(tags->'name:%s', ''), name)", lang)
	}

	// Базовый запрос
	sqlQuery := fmt.Sprintf(`
		SELECT 
			osm_id,
			%s AS name,
			COALESCE(boundary, 'administrative') AS type,
			COALESCE((admin_level)::integer, 0) AS admin_level,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS center_lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS center_lon,
			ST_Area(ST_Transform(way, %d)::geography) / 1000000 AS area_sq_km
		FROM %s
		WHERE boundary = 'administrative'
		  AND admin_level IS NOT NULL
		  AND (%s ILIKE '%%' || $1 || '%%' OR name ILIKE '%%' || $1 || '%%')
	`, nameField, SRID4326, SRID4326, SRID4326, planetPolygonTable, nameField)

	args := []interface{}{searchQuery}
	argIndex := 2

	// Фильтр по административным уровням
	if len(adminLevels) > 0 {
		placeholders := make([]string, len(adminLevels))
		for i, level := range adminLevels {
			placeholders[i] = fmt.Sprintf("$%d", argIndex)
			args = append(args, level)
			argIndex++
		}
		sqlQuery += fmt.Sprintf(" AND (admin_level)::integer IN (%s)", strings.Join(placeholders, ","))
	}

	sqlQuery += fmt.Sprintf(" ORDER BY (admin_level)::integer ASC, name ASC LIMIT $%d", argIndex)
	args = append(args, limit)

	rows, err := r.db.QueryxContext(ctx, sqlQuery, args...)
	if err != nil {
		r.logger.Error("failed to search osm boundaries", zap.String("query", searchQuery), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var boundaries []*domain.AdminBoundary
	for rows.Next() {
		var b domain.AdminBoundary
		var adminLevelInt int

		err := rows.Scan(
			&b.OSMId, &b.Name, &b.Type, &adminLevelInt,
			&b.CenterLat, &b.CenterLon, &b.AreaSqKm,
		)
		if err != nil {
			r.logger.Error("failed to scan boundary row", zap.Error(err))
			continue
		}

		b.ID = b.OSMId
		b.AdminLevel = adminLevelInt

		boundaries = append(boundaries, &b)
	}

	return boundaries, nil
}

// Search выполняет простой текстовый поиск по названиям границ
func (r *boundaryRepository) Search(ctx context.Context, query string, limit int) ([]*domain.AdminBoundary, error) {
	return r.SearchByText(ctx, query, "", nil, limit)
}

// ReverseGeocode возвращает адрес по координатам (поддержка admin_level 2, 4, 6, 7, 8, 9, 10, 11)
func (r *boundaryRepository) ReverseGeocode(
	ctx context.Context,
	lat, lon float64,
) (*domain.Address, error) {
	query := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_Transform(ST_SetSRID(ST_MakePoint($1, $2), %d), %d) AS geom
		)
		SELECT 
			MAX(CASE WHEN (admin_level)::integer = 2 THEN name END) AS country,
			MAX(CASE WHEN (admin_level)::integer = 4 THEN name END) AS region,
			MAX(CASE WHEN (admin_level)::integer = 6 THEN name END) AS province,
			MAX(CASE WHEN (admin_level)::integer = 7 THEN name END) AS subprovince,
			MAX(CASE WHEN (admin_level)::integer = 8 THEN name END) AS city,
			MAX(CASE WHEN (admin_level)::integer = 9 THEN name END) AS district,
			MAX(CASE WHEN (admin_level)::integer = 10 THEN name END) AS subdistrict,
			MAX(CASE WHEN (admin_level)::integer = 11 THEN name END) AS neighborhood
		FROM %s, point
		WHERE boundary = 'administrative'
		  AND admin_level IS NOT NULL
		  AND way && ST_Expand(point.geom, $3)
		  AND ST_Contains(way, point.geom)
	`, SRID4326, SRID3857, planetPolygonTable)

	var country, region, province, subprovince, city, district, subdistrict, neighborhood sql.NullString

	err := r.db.QueryRowContext(ctx, query, lon, lat, BoundaryExpansionDegrees).Scan(
		&country, &region, &province, &subprovince, &city, &district, &subdistrict, &neighborhood,
	)

	if err == sql.ErrNoRows || (!country.Valid && !region.Valid && !province.Valid && !city.Valid) {
		return nil, pkgerrors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("failed to reverse geocode from osm",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon),
			zap.Error(err),
		)
		return nil, pkgerrors.ErrDatabaseError
	}

	addr := &domain.Address{
		Country:  country.String,
		Region:   region.String,
		Province: province.String,
		City:     city.String,
	}

	if subprovince.Valid && subprovince.String != "" {
		addr.Subprovince = &subprovince.String
	}
	if district.Valid && district.String != "" {
		addr.District = &district.String
	}
	if subdistrict.Valid && subdistrict.String != "" {
		addr.Subdistrict = &subdistrict.String
	}
	if neighborhood.Valid && neighborhood.String != "" {
		addr.Neighborhood = &neighborhood.String
	}

	return addr, nil
}

// ReverseGeocodeBatch возвращает адреса для нескольких точек одним запросом (производительный батчевый метод)
func (r *boundaryRepository) ReverseGeocodeBatch(
	ctx context.Context,
	points []domain.LatLon,
) ([]*domain.Address, error) {
	if len(points) == 0 {
		return []*domain.Address{}, nil
	}

	// Строим VALUES для всех точек с явным приведением типов для совместимости с pgx
	valueStrings := make([]string, len(points))
	valueArgs := make([]interface{}, 0, len(points)*2)

	for i, point := range points {
		valueStrings[i] = fmt.Sprintf("(($%d)::float8, ($%d)::float8)", i*2+1, i*2+2)
		valueArgs = append(valueArgs, point.Lon, point.Lat)
	}

	query := fmt.Sprintf(`
		WITH input_points AS (
			SELECT 
				row_number() OVER () AS point_id,
				ST_SetSRID(ST_MakePoint(lon, lat), %d) AS geom_4326,
				ST_Transform(ST_SetSRID(ST_MakePoint(lon, lat), %d), %d) AS geom_3857
			FROM (VALUES %s) AS t(lon, lat)
		),
		boundaries_per_point AS (
			SELECT 
				ip.point_id,
				(b.admin_level)::integer AS admin_level,
				b.name
			FROM input_points ip
			JOIN %s b ON b.boundary = 'administrative'
				AND b.admin_level IS NOT NULL
				AND b.way && ST_Transform(ST_Expand(ip.geom_4326, $%d), %d)
				AND ST_Contains(b.way, ip.geom_3857)
		)
		SELECT 
			point_id,
			MAX(CASE WHEN admin_level = 2 THEN name END) AS country,
			MAX(CASE WHEN admin_level = 4 THEN name END) AS region,
			MAX(CASE WHEN admin_level = 6 THEN name END) AS province,
			MAX(CASE WHEN admin_level = 7 THEN name END) AS subprovince,
			MAX(CASE WHEN admin_level = 8 THEN name END) AS city,
			MAX(CASE WHEN admin_level = 9 THEN name END) AS district,
			MAX(CASE WHEN admin_level = 10 THEN name END) AS subdistrict,
			MAX(CASE WHEN admin_level = 11 THEN name END) AS neighborhood
		FROM boundaries_per_point
		GROUP BY point_id
		ORDER BY point_id
	`, SRID4326, SRID4326, SRID3857, strings.Join(valueStrings, ","), planetPolygonTable, len(valueArgs)+1, SRID3857)

	valueArgs = append(valueArgs, BoundaryExpansionDegrees)

	rows, err := r.db.QueryxContext(ctx, query, valueArgs...)
	if err != nil {
		r.logger.Error("failed to batch reverse geocode from osm", zap.Int("points_count", len(points)), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	// Создаем результирующий массив с nil значениями
	results := make([]*domain.Address, len(points))

	for rows.Next() {
		var pointID int
		var country, region, province, subprovince, city, district, subdistrict, neighborhood sql.NullString

		err := rows.Scan(
			&pointID, &country, &region, &province, &subprovince, &city, &district, &subdistrict, &neighborhood,
		)
		if err != nil {
			r.logger.Error("failed to scan batch reverse geocode row", zap.Error(err))
			continue
		}

		// pointID начинается с 1, индексы массива с 0
		idx := pointID - 1
		if idx >= 0 && idx < len(results) {
			addr := &domain.Address{
				Country:  country.String,
				Region:   region.String,
				Province: province.String,
				City:     city.String,
			}

			if subprovince.Valid && subprovince.String != "" {
				addr.Subprovince = &subprovince.String
			}
			if district.Valid && district.String != "" {
				addr.District = &district.String
			}
			if subdistrict.Valid && subdistrict.String != "" {
				addr.Subdistrict = &subdistrict.String
			}
			if neighborhood.Valid && neighborhood.String != "" {
				addr.Neighborhood = &neighborhood.String
			}

			results[idx] = addr
		}
	}

	return results, nil
}

// GetByPoint возвращает административные границы для точки
func (r *boundaryRepository) GetByPoint(ctx context.Context, lat, lon float64) ([]*domain.AdminBoundary, error) {
	query := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_Transform(ST_SetSRID(ST_MakePoint($1, $2), %d), %d) AS geom
		)
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(boundary, 'administrative') AS type,
			COALESCE((admin_level)::integer, 0) AS admin_level,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS center_lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS center_lon,
			ST_Area(ST_Transform(way, %d)::geography) / 1000000 AS area_sq_km
		FROM %s, point
		WHERE boundary = 'administrative'
		  AND admin_level IS NOT NULL
		  AND way && ST_Expand(point.geom, $3)
		  AND ST_Contains(way, point.geom)
		ORDER BY (admin_level)::integer ASC
	`, SRID4326, SRID3857, SRID4326, SRID4326, SRID4326, planetPolygonTable)

	rows, err := r.db.QueryxContext(ctx, query, lon, lat, BoundaryExpansionDegrees)
	if err != nil {
		r.logger.Error("failed to get osm boundaries by point",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon),
			zap.Error(err),
		)
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var boundaries []*domain.AdminBoundary
	for rows.Next() {
		var b domain.AdminBoundary
		var adminLevelInt int

		err := rows.Scan(
			&b.OSMId, &b.Name, &b.NameEn, &b.Type, &adminLevelInt,
			&b.CenterLat, &b.CenterLon, &b.AreaSqKm,
		)
		if err != nil {
			r.logger.Error("failed to scan boundary row", zap.Error(err))
			continue
		}

		b.ID = b.OSMId
		b.AdminLevel = adminLevelInt

		boundaries = append(boundaries, &b)
	}

	return boundaries, nil
}

// GetChildren возвращает дочерние границы для родительской (в OSM данных связи parent-child могут отсутствовать)
func (r *boundaryRepository) GetChildren(ctx context.Context, parentID int64) ([]*domain.AdminBoundary, error) {
	// В OSM данных нет явной связи parent_id, нужно искать через геометрию
	// Ищем границы следующего уровня, которые содержатся в родительской
	query := fmt.Sprintf(`
		WITH parent AS (
			SELECT way, (admin_level)::integer AS parent_level
			FROM %s
			WHERE osm_id = $1
			  AND boundary = 'administrative'
			  AND admin_level IS NOT NULL
		)
		SELECT 
			b.osm_id,
			COALESCE(b.name, '') AS name,
			COALESCE(NULLIF(b.tags->'name:en', ''), '') AS name_en,
			COALESCE(b.boundary, 'administrative') AS type,
			COALESCE((b.admin_level)::integer, 0) AS admin_level,
			ST_Y(ST_Centroid(ST_Transform(b.way, %d))) AS center_lat,
			ST_X(ST_Centroid(ST_Transform(b.way, %d))) AS center_lon,
			ST_Area(ST_Transform(b.way, %d)::geography) / 1000000 AS area_sq_km
		FROM %s b, parent
		WHERE b.boundary = 'administrative'
		  AND b.admin_level IS NOT NULL
		  AND b.osm_id != $1
		  AND (b.admin_level)::integer > parent.parent_level
		  AND ST_Within(ST_Centroid(b.way), parent.way)
		ORDER BY (b.admin_level)::integer ASC, b.name ASC
		LIMIT $2
	`, planetPolygonTable, SRID4326, SRID4326, SRID4326, planetPolygonTable)

	rows, err := r.db.QueryxContext(ctx, query, parentID, LimitBoundaries)
	if err != nil {
		r.logger.Error("failed to get osm children boundaries",
			zap.Int64("parent_id", parentID),
			zap.Error(err),
		)
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var boundaries []*domain.AdminBoundary
	for rows.Next() {
		var b domain.AdminBoundary
		var adminLevelInt int

		err := rows.Scan(
			&b.OSMId, &b.Name, &b.NameEn, &b.Type, &adminLevelInt,
			&b.CenterLat, &b.CenterLon, &b.AreaSqKm,
		)
		if err != nil {
			r.logger.Error("failed to scan boundary row", zap.Error(err))
			continue
		}

		b.ID = b.OSMId
		b.AdminLevel = adminLevelInt

		boundaries = append(boundaries, &b)
	}

	return boundaries, nil
}

// GetByAdminLevel возвращает границы определенного административного уровня
func (r *boundaryRepository) GetByAdminLevel(ctx context.Context, level int, limit int) ([]*domain.AdminBoundary, error) {
	if limit <= 0 || limit > LimitBoundaries {
		limit = LimitBoundaries
	}

	query := fmt.Sprintf(`
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(boundary, 'administrative') AS type,
			COALESCE((admin_level)::integer, 0) AS admin_level,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS center_lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS center_lon,
			ST_Area(ST_Transform(way, %d)::geography) / 1000000 AS area_sq_km
		FROM %s
		WHERE boundary = 'administrative'
		  AND admin_level IS NOT NULL
		  AND (admin_level)::integer = $1
		ORDER BY name ASC
		LIMIT $2
	`, SRID4326, SRID4326, SRID4326, planetPolygonTable)

	rows, err := r.db.QueryxContext(ctx, query, level, limit)
	if err != nil {
		r.logger.Error("failed to get osm boundaries by admin level",
			zap.Int("level", level),
			zap.Error(err),
		)
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var boundaries []*domain.AdminBoundary
	for rows.Next() {
		var b domain.AdminBoundary
		var adminLevelInt int

		err := rows.Scan(
			&b.OSMId, &b.Name, &b.NameEn, &b.Type, &adminLevelInt,
			&b.CenterLat, &b.CenterLon, &b.AreaSqKm,
		)
		if err != nil {
			r.logger.Error("failed to scan boundary row", zap.Error(err))
			continue
		}

		b.ID = b.OSMId
		b.AdminLevel = adminLevelInt

		boundaries = append(boundaries, &b)
	}

	return boundaries, nil
}

// GetBoundariesInRadius возвращает границы в радиусе от точки
func (r *boundaryRepository) GetBoundariesInRadius(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.AdminBoundary, error) {
	radiusMeters := radiusKm * 1000

	query := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d) AS geom
		),
		circle AS (
			SELECT ST_Transform(ST_Buffer(point.geom::geography, $3)::geometry, %d) AS geom
			FROM point
		)
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(boundary, 'administrative') AS type,
			COALESCE((admin_level)::integer, 0) AS admin_level,
			ST_Y(ST_Centroid(ST_Transform(way, %d))) AS center_lat,
			ST_X(ST_Centroid(ST_Transform(way, %d))) AS center_lon,
			ST_Area(ST_Transform(way, %d)::geography) / 1000000 AS area_sq_km
		FROM %s, circle
		WHERE boundary = 'administrative'
		  AND admin_level IS NOT NULL
		  AND way && circle.geom
		  AND ST_Intersects(way, circle.geom)
		  AND (admin_level)::integer IN (6, 8, 9)
		ORDER BY (admin_level)::integer ASC, area_sq_km ASC
		LIMIT $4
	`, SRID4326, SRID3857, SRID4326, SRID4326, SRID4326, planetPolygonTable)

	rows, err := r.db.QueryxContext(ctx, query, lon, lat, radiusMeters, LimitBoundariesRadius)
	if err != nil {
		r.logger.Error("failed to get osm boundaries in radius",
			zap.Float64("lat", lat),
			zap.Float64("lon", lon),
			zap.Float64("radius_km", radiusKm),
			zap.Error(err),
		)
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var boundaries []*domain.AdminBoundary
	for rows.Next() {
		var b domain.AdminBoundary
		var adminLevelInt int

		err := rows.Scan(
			&b.OSMId, &b.Name, &b.NameEn, &b.Type, &adminLevelInt,
			&b.CenterLat, &b.CenterLon, &b.AreaSqKm,
		)
		if err != nil {
			r.logger.Error("failed to scan boundary row", zap.Error(err))
			continue
		}

		b.ID = b.OSMId
		b.AdminLevel = adminLevelInt

		boundaries = append(boundaries, &b)
	}

	return boundaries, nil
}

// GetTile - генерация MVT тайла с полигонами административных границ
func (r *boundaryRepository) GetTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Валидация уровня зума
	if z < 0 || z > 18 {
		r.logger.Warn("Invalid zoom level for boundary tile", zap.Int("z", z))
		return []byte{}, nil
	}

	// Динамический фильтр уровней границ в зависимости от зума
	// До z=12: показываем только ОДИН уровень без наложений
	// После z=12: показываем несколько уровней, вырезая более детальные из крупных
	// z 0-4: только страны (admin_level=2)
	// z 5-6: только регионы (admin_level=4)
	// z 7-8: только провинции (admin_level=6)
	// z 9-10: только субпровинции/города (admin_level=7,8)
	// z 11-12: только города (admin_level=8)
	// z 13+: города + районы + кварталы с вырезанием (admin_level=8,9,10)

	var query string

	if z < 12 {
		// До зума 12 - простая логика, один уровень
		var adminLevelFilter string
		switch {
		case z <= 4:
			adminLevelFilter = "(admin_level)::integer = 2"
		case z <= 6:
			adminLevelFilter = "(admin_level)::integer = 4"
		case z <= 8:
			adminLevelFilter = "(admin_level)::integer = 6"
		case z <= 10:
			adminLevelFilter = "(admin_level)::integer = 7"
		default: // z 11-12
			adminLevelFilter = "(admin_level)::integer = 8"
		}

		query = fmt.Sprintf(`
			WITH tile_bounds AS (
				SELECT ST_TileEnvelope($1, $2, $3) AS geom
			),
			mvt_geom AS (
				SELECT
					osm_id,
					COALESCE(name, '') AS name,
					COALESCE(NULLIF(tags->'name:en', ''), '') AS name_en,
					COALESCE(NULLIF(tags->'name:es', ''), '') AS name_es,
					COALESCE(NULLIF(tags->'name:ca', ''), '') AS name_ca,
					COALESCE(NULLIF(tags->'name:ru', ''), '') AS name_ru,
					COALESCE(NULLIF(tags->'wikidata', ''), '') AS wikidata,
					COALESCE(boundary, 'administrative') AS boundary_type,
					(admin_level)::integer AS admin_level,
					COALESCE((tags->'population')::bigint, 0) AS population,
					ST_AsMVTGeom(
						ST_Transform(way, %d),
						tile_bounds.geom,
						%d,
						%d,
						true
					) AS geom
				FROM %s, tile_bounds
				WHERE boundary = 'administrative'
				  AND admin_level IS NOT NULL
				  AND %s
				  AND ST_Intersects(way, ST_Transform(tile_bounds.geom, %d))
			)
			SELECT ST_AsMVT(mvt_geom.*, 'boundaries', %d, 'geom')
			FROM mvt_geom
			WHERE geom IS NOT NULL
		`, SRID3857, MVTExtent, MVTBuffer, planetPolygonTable, adminLevelFilter, SRID3857, MVTExtent)
	} else {
		// После зума 12 - используем ST_Difference для вырезания
		query = fmt.Sprintf(`
			WITH tile_bounds AS (
				SELECT ST_TileEnvelope($1, $2, $3) AS geom
			),
			-- Выбираем все границы нужных уровней
			all_boundaries AS (
				SELECT
					osm_id,
					COALESCE(name, '') AS name,
					COALESCE(NULLIF(tags->'name:en', ''), '') AS name_en,
					COALESCE(NULLIF(tags->'name:es', ''), '') AS name_es,
					COALESCE(NULLIF(tags->'name:ca', ''), '') AS name_ca,
					COALESCE(NULLIF(tags->'name:ru', ''), '') AS name_ru,
					COALESCE(NULLIF(tags->'wikidata', ''), '') AS wikidata,
					COALESCE(boundary, 'administrative') AS boundary_type,
					(admin_level)::integer AS admin_level,
					COALESCE((tags->'population')::bigint, 0) AS population,
					way
				FROM %s, tile_bounds
				WHERE boundary = 'administrative'
				  AND admin_level IS NOT NULL
				  AND (admin_level)::integer IN (8, 9, 10)
				  AND ST_Intersects(way, ST_Transform(tile_bounds.geom, %d))
			),
			-- Вырезаем более детальные границы из крупных
			boundaries_with_holes AS (
				SELECT
					b1.osm_id,
					b1.name,
					b1.name_en,
					b1.name_es,
					b1.name_ca,
					b1.name_ru,
					b1.wikidata,
					b1.boundary_type,
					b1.admin_level,
					b1.population,
					CASE 
						-- Для уровня 8: вырезаем все районы (9) и кварталы (10)
						WHEN b1.admin_level = 8 THEN
							COALESCE(
								ST_Difference(
									b1.way,
									(SELECT ST_Union(way) FROM all_boundaries b2 
									 WHERE b2.admin_level IN (9, 10) 
									   AND ST_Intersects(b1.way, b2.way))
								),
								b1.way
							)
						-- Для уровня 9: вырезаем кварталы (10)
						WHEN b1.admin_level = 9 THEN
							COALESCE(
								ST_Difference(
									b1.way,
									(SELECT ST_Union(way) FROM all_boundaries b2 
									 WHERE b2.admin_level = 10 
									   AND ST_Intersects(b1.way, b2.way))
								),
								b1.way
							)
						-- Для уровня 10: не вырезаем ничего
						ELSE b1.way
					END AS way
				FROM all_boundaries b1
			),
			mvt_geom AS (
				SELECT
					osm_id,
					name,
					name_en,
					name_es,
					name_ca,
					name_ru,
					wikidata,
					boundary_type,
					admin_level,
					population,
					ST_AsMVTGeom(
						ST_Transform(way, %d),
						(SELECT geom FROM tile_bounds),
						%d,
						%d,
						true
					) AS geom
				FROM boundaries_with_holes
				WHERE NOT ST_IsEmpty(way)
			)
			SELECT ST_AsMVT(mvt_geom.*, 'boundaries', %d, 'geom')
			FROM mvt_geom
			WHERE geom IS NOT NULL
		`, planetPolygonTable, SRID3857, SRID3857, MVTExtent, MVTBuffer, MVTExtent)
	}

	var tile []byte
	err := r.db.QueryRowxContext(ctx, query, z, x, y).Scan(&tile)

	if err == sql.ErrNoRows || len(tile) == 0 {
		r.logger.Debug("Empty boundary tile",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y))
		return []byte{}, nil
	}

	if err != nil {
		r.logger.Error("failed to generate boundary tile",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	r.logger.Debug("Boundary tile generated successfully",
		zap.Int("z", z),
		zap.Int("x", x),
		zap.Int("y", y),
		zap.Int("size_bytes", len(tile)))

	return tile, nil
}

// GetBoundariesRadiusTile - метод генерации тайлов не реализован, будет сделан отдельно
func (r *boundaryRepository) GetBoundariesRadiusTile(ctx context.Context, lat, lon, radiusKm float64) ([]byte, error) {
	r.logger.Warn("GetBoundariesRadiusTile not implemented for OSM boundary repository")
	return []byte{}, nil
}
