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

type transportRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewTransportRepository создает репозиторий транспорта для OSM базы данных
func NewTransportRepository(db *DB) repository.TransportRepository {
	return &transportRepository{
		db:     db.DB,
		logger: db.logger,
	}
}

// GetNearestStations возвращает ближайшие транспортные станции
func (r *transportRepository) GetNearestStations(
	ctx context.Context,
	lat, lon float64,
	types []string,
	maxDistance float64,
	limit int,
) ([]*domain.TransportStation, error) {
	if limit <= 0 || limit > LimitStations {
		limit = LimitStations
	}

	radiusMeters := maxDistance * 1000

	// Строим фильтр по типам транспорта
	typeFilter := ""
	args := []interface{}{lon, lat, radiusMeters}
	if len(types) > 0 {
		placeholders := make([]string, len(types))
		for i, t := range types {
			placeholders[i] = fmt.Sprintf("$%d", len(args)+1)
			args = append(args, t)
		}
		typeFilter = fmt.Sprintf(" AND (public_transport IN (%s) OR railway IN (%s))",
			strings.Join(placeholders, ","), strings.Join(placeholders, ","))
	}

	args = append(args, limit)

	query := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		)
		SELECT DISTINCT ON (osm_id)
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(name, ''), NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(NULLIF(public_transport, ''), NULLIF(railway, ''), 'station') AS type,
			ST_Y(ST_Transform(way, %d)) AS lat,
			ST_X(ST_Transform(way, %d)) AS lon,
			COALESCE(tags->'operator', '') AS operator,
			COALESCE(tags->'network', '') AS network,
			COALESCE(tags->'wheelchair', '') AS wheelchair,
			ST_Distance(ST_Transform(way, %d)::geography, point.geom) AS distance
		FROM %s, point
		WHERE (public_transport IS NOT NULL OR railway IN ('station', 'halt', 'stop'))%s
		  AND ST_DWithin(ST_Transform(way, %d)::geography, point.geom, $3)
		ORDER BY osm_id, distance
		LIMIT $%d
	`, SRID4326, SRID4326, SRID4326, SRID4326, planetPointTable, typeFilter, SRID4326, len(args))

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to get nearest osm stations", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var stations []*domain.TransportStation
	for rows.Next() {
		var s domain.TransportStation
		var distance float64
		var operator, network, wheelchair string

		err := rows.Scan(
			&s.OSMId, &s.Name, &s.NameEn, &s.Type,
			&s.Lat, &s.Lon, &operator, &network, &wheelchair, &distance,
		)
		if err != nil {
			r.logger.Error("failed to scan station row", zap.Error(err))
			continue
		}

		s.ID = s.OSMId
		if operator != "" {
			s.Operator = &operator
		}
		if network != "" {
			s.Network = &network
		}
		if wheelchair != "" {
			// Конвертируем yes/no в bool
			wheelchairBool := wheelchair == "yes" || wheelchair == "true" || wheelchair == "1"
			s.Wheelchair = &wheelchairBool
		}
		s.LineIDs = []int64{} // В OSM данных связи с линиями придется получать отдельно
		s.Tags = make(map[string]string)

		stations = append(stations, &s)
	}

	return stations, nil
}

// GetLineByID возвращает транспортную линию по ID
func (r *transportRepository) GetLineByID(ctx context.Context, id int64) (*domain.TransportLine, error) {
	query := fmt.Sprintf(`
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(ref, '') AS ref,
			COALESCE(NULLIF(route, ''), NULLIF(railway, ''), 'route') AS type,
			COALESCE(tags->'colour', '') AS color,
			COALESCE(tags->'text_colour', '') AS text_color,
			COALESCE(tags->'operator', '') AS operator,
			COALESCE(tags->'network', '') AS network,
			COALESCE(tags->'from', '') AS from_station,
			COALESCE(tags->'to', '') AS to_station
		FROM %s
		WHERE osm_id = $1
		LIMIT 1
	`, planetLineTable)

	var line domain.TransportLine
	var color, textColor, operator, network, fromStation, toStation string

	err := r.db.QueryRowxContext(ctx, query, id).Scan(
		&line.OSMId, &line.Name, &line.Ref, &line.Type,
		&color, &textColor, &operator, &network, &fromStation, &toStation,
	)

	if err == sql.ErrNoRows {
		return nil, pkgerrors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("failed to get osm line", zap.Int64("osm_id", id), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	line.ID = line.OSMId
	if color != "" {
		line.Color = &color
	}
	if textColor != "" {
		line.TextColor = &textColor
	}
	if operator != "" {
		line.Operator = &operator
	}
	if network != "" {
		line.Network = &network
	}
	if fromStation != "" {
		line.FromStation = &fromStation
	}
	if toStation != "" {
		line.ToStation = &toStation
	}
	line.StationIDs = []int64{} // В OSM данных нужна дополнительная логика для извлечения станций
	line.Tags = make(map[string]string)

	return &line, nil
}

// GetLinesByIDs возвращает несколько линий по их ID
func (r *transportRepository) GetLinesByIDs(ctx context.Context, ids []int64) ([]*domain.TransportLine, error) {
	if len(ids) == 0 {
		return []*domain.TransportLine{}, nil
	}

	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT DISTINCT ON (osm_id)
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(ref, '') AS ref,
			COALESCE(NULLIF(route, ''), NULLIF(railway, ''), 'route') AS type,
			COALESCE(tags->'colour', '') AS color,
			COALESCE(tags->'text_colour', '') AS text_color,
			COALESCE(tags->'operator', '') AS operator,
			COALESCE(tags->'network', '') AS network
		FROM %s
		WHERE osm_id IN (%s)
		ORDER BY osm_id
	`, planetLineTable, strings.Join(placeholders, ","))

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to get osm lines by ids", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var lines []*domain.TransportLine
	for rows.Next() {
		var l domain.TransportLine
		var color, textColor, operator, network string

		err := rows.Scan(
			&l.OSMId, &l.Name, &l.Ref, &l.Type,
			&color, &textColor, &operator, &network,
		)
		if err != nil {
			r.logger.Error("failed to scan line row", zap.Error(err))
			continue
		}

		l.ID = l.OSMId
		if color != "" {
			l.Color = &color
		}
		if textColor != "" {
			l.TextColor = &textColor
		}
		if operator != "" {
			l.Operator = &operator
		}
		if network != "" {
			l.Network = &network
		}
		l.StationIDs = []int64{}
		l.Tags = make(map[string]string)

		lines = append(lines, &l)
	}

	return lines, nil
}

// GetStationsByLineID возвращает станции для линии (заглушка для OSM)
func (r *transportRepository) GetStationsByLineID(ctx context.Context, lineID int64) ([]*domain.TransportStation, error) {
	// В OSM данных связь линий и станций требует дополнительной обработки relation members
	// Для базовой реализации возвращаем пустой массив
	r.logger.Warn("GetStationsByLineID not fully implemented for OSM data", zap.Int64("line_id", lineID))
	return []*domain.TransportStation{}, nil
}

// GetTransportTile генерирует MVT тайл с транспортом
func (r *transportRepository) GetTransportTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Станции
	stationsQuery := fmt.Sprintf(`
		WITH bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		stations AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				CASE
					WHEN railway = 'station' AND (tags->'station' = 'subway' OR tags->'subway' = 'yes') THEN 'subway'
					WHEN railway = 'tram_stop' OR (railway = 'station' AND tags->'station' = 'light_rail') THEN 'tram_stop'
					WHEN highway = 'bus_stop' OR (public_transport IN ('platform', 'stop_position') AND tags->'bus' = 'yes') THEN 'bus_stop'
					WHEN railway IN ('station', 'halt') THEN 'station'
					WHEN public_transport IS NOT NULL THEN COALESCE(NULLIF(public_transport, ''), 'stop')
					ELSE 'station'
				END AS type,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM %s, bounds
			WHERE (public_transport IS NOT NULL OR railway IN ('station', 'halt', 'stop') OR highway = 'bus_stop')
			  AND way && bounds.geom
		)
		SELECT COALESCE(ST_AsMVT(stations.*, 'stations'), '\\x'::bytea) AS tile
		FROM stations
		WHERE geom IS NOT NULL
	`, planetPointTable)

	var stationsTile []byte
	err := r.db.QueryRowContext(ctx, stationsQuery, z, x, y, MVTExtent, MVTBuffer).Scan(&stationsTile)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("failed to build osm stations tile", zap.Int("z", z), zap.Int("x", x), zap.Int("y", y), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	// Линии
	linesQuery := fmt.Sprintf(`
		WITH bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		lines AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(ref, '') AS ref,
				COALESCE(route, '') AS type,
				COALESCE(tags->'colour', '') AS color,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM %s, bounds
			WHERE route IS NOT NULL
			  AND way && bounds.geom
		)
		SELECT COALESCE(ST_AsMVT(lines.*, 'lines'), '\\x'::bytea) AS tile
		FROM lines
		WHERE geom IS NOT NULL
	`, planetLineTable)

	var linesTile []byte
	err = r.db.QueryRowContext(ctx, linesQuery, z, x, y, MVTExtent, MVTBuffer).Scan(&linesTile)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("failed to build osm lines tile", zap.Int("z", z), zap.Int("x", x), zap.Int("y", y), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	// Объединяем тайлы
	result := append(stationsTile, linesTile...)
	return result, nil
}

// GetLineTile генерирует MVT тайл для одной линии
func (r *transportRepository) GetLineTile(ctx context.Context, lineID int64) ([]byte, error) {
	query := fmt.Sprintf(`
		WITH line_data AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(ref, '') AS ref,
				COALESCE(route, '') AS type,
				COALESCE(tags->'colour', '') AS color,
				way
			FROM %s
			WHERE osm_id = $1
		),
		bounds AS (
			SELECT ST_Expand(ST_Envelope(way), 0.01) AS geom
			FROM line_data
		),
		line_mvt AS (
			SELECT 
				ld.id, ld.name, ld.ref, ld.type, ld.color,
				ST_AsMVTGeom(ld.way, b.geom, $2, $3, true) AS geom
			FROM line_data ld, bounds b
		)
		SELECT COALESCE(ST_AsMVT(line_mvt.*, 'line'), '\\x'::bytea) AS tile
		FROM line_mvt
		WHERE geom IS NOT NULL
	`, planetLineTable)

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, lineID, MVTExtent, MVTBuffer).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("failed to build osm line tile", zap.Int64("line_id", lineID), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return tile, nil
}

// GetLinesTile генерирует MVT тайл для нескольких линий
func (r *transportRepository) GetLinesTile(ctx context.Context, lineIDs []int64) ([]byte, error) {
	if len(lineIDs) == 0 {
		return []byte{}, nil
	}

	placeholders := make([]string, len(lineIDs))
	args := make([]interface{}, len(lineIDs)+2)
	for i, id := range lineIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	args[len(lineIDs)] = MVTExtent
	args[len(lineIDs)+1] = MVTBuffer

	query := fmt.Sprintf(`
		WITH lines_data AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(ref, '') AS ref,
				COALESCE(route, '') AS type,
				COALESCE(tags->'colour', '') AS color,
				way
			FROM %s
			WHERE osm_id IN (%s)
		),
		bounds AS (
			SELECT ST_Expand(ST_Envelope(ST_Collect(way)), 0.01) AS geom
			FROM lines_data
		),
		lines_mvt AS (
			SELECT 
				ld.id, ld.name, ld.ref, ld.type, ld.color,
				ST_AsMVTGeom(ld.way, b.geom, $%d, $%d, true) AS geom
			FROM lines_data ld, bounds b
		)
		SELECT COALESCE(ST_AsMVT(lines_mvt.*, 'lines'), '\\x'::bytea) AS tile
		FROM lines_mvt
		WHERE geom IS NOT NULL
	`, planetLineTable, strings.Join(placeholders, ","), len(args)-1, len(args))

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("failed to build osm lines tile", zap.Int64s("line_ids", lineIDs), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return tile, nil
}

// GetStationsInRadius возвращает станции в радиусе от точки
func (r *transportRepository) GetStationsInRadius(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.TransportStation, error) {
	radiusMeters := radiusKm * 1000

	query := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		)
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(NULLIF(name, ''), NULLIF(tags->'name:en', ''), '') AS name_en,
			COALESCE(NULLIF(public_transport, ''), NULLIF(railway, ''), 'station') AS type,
			ST_Y(ST_Transform(way, %d)) AS lat,
			ST_X(ST_Transform(way, %d)) AS lon,
			COALESCE(tags->'operator', '') AS operator,
			COALESCE(tags->'network', '') AS network,
			COALESCE(tags->'wheelchair', '') AS wheelchair,
			ST_Distance(ST_Transform(way, %d)::geography, point.geom) AS distance
		FROM %s, point
		WHERE (public_transport IS NOT NULL OR railway IN ('station', 'halt', 'stop'))
		  AND ST_DWithin(ST_Transform(way, %d)::geography, point.geom, $3)
		ORDER BY distance
		LIMIT $4
	`, SRID4326, SRID4326, SRID4326, SRID4326, planetPointTable, SRID4326)

	rows, err := r.db.QueryxContext(ctx, query, lon, lat, radiusMeters, LimitStations)
	if err != nil {
		r.logger.Error("failed to get osm stations in radius", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var stations []*domain.TransportStation
	for rows.Next() {
		var s domain.TransportStation
		var distance float64
		var operator, network, wheelchair string

		err := rows.Scan(
			&s.OSMId, &s.Name, &s.NameEn, &s.Type,
			&s.Lat, &s.Lon, &operator, &network, &wheelchair, &distance,
		)
		if err != nil {
			r.logger.Error("failed to scan station row", zap.Error(err))
			continue
		}

		s.ID = s.OSMId
		if operator != "" {
			s.Operator = &operator
		}
		if network != "" {
			s.Network = &network
		}
		if wheelchair != "" {
			// Конвертируем yes/no в bool
			wheelchairBool := wheelchair == "yes" || wheelchair == "true" || wheelchair == "1"
			s.Wheelchair = &wheelchairBool
		}
		s.LineIDs = []int64{}
		s.Tags = make(map[string]string)

		stations = append(stations, &s)
	}

	return stations, nil
}

// GetLinesInRadius возвращает линии в радиусе от точки
func (r *transportRepository) GetLinesInRadius(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.TransportLine, error) {
	radiusMeters := radiusKm * 1000

	query := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d) AS geom
		),
		circle AS (
			SELECT ST_Buffer(point.geom::geography, $3)::geometry AS geom
			FROM point
		)
		SELECT 
			osm_id,
			COALESCE(name, '') AS name,
			COALESCE(ref, '') AS ref,
			COALESCE(route, '') AS type,
			COALESCE(tags->'colour', '') AS color,
			COALESCE(tags->'text_colour', '') AS text_color,
			COALESCE(tags->'operator', '') AS operator,
			COALESCE(tags->'network', '') AS network
		FROM %s, circle
		WHERE route IS NOT NULL
		  AND way && circle.geom
		  AND ST_Intersects(way, circle.geom)
		ORDER BY name
		LIMIT $4
	`, SRID4326, planetLineTable)

	rows, err := r.db.QueryxContext(ctx, query, lon, lat, radiusMeters, LimitLines)
	if err != nil {
		r.logger.Error("failed to get osm lines in radius", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var lines []*domain.TransportLine
	for rows.Next() {
		var l domain.TransportLine
		var color, textColor, operator, network string

		err := rows.Scan(
			&l.OSMId, &l.Name, &l.Ref, &l.Type,
			&color, &textColor, &operator, &network,
		)
		if err != nil {
			r.logger.Error("failed to scan line row", zap.Error(err))
			continue
		}

		l.ID = l.OSMId
		if color != "" {
			l.Color = &color
		}
		if textColor != "" {
			l.TextColor = &textColor
		}
		if operator != "" {
			l.Operator = &operator
		}
		if network != "" {
			l.Network = &network
		}
		l.StationIDs = []int64{}
		l.Tags = make(map[string]string)

		lines = append(lines, &l)
	}

	return lines, nil
}

// GetTransportRadiusTile генерирует MVT тайл с транспортом в радиусе
func (r *transportRepository) GetTransportRadiusTile(ctx context.Context, lat, lon, radiusKm float64) ([]byte, error) {
	radiusMeters := radiusKm * 1000

	// Станции
	stationsQuery := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		),
		circle AS (
			SELECT ST_Transform(ST_Buffer(point.geom, $3)::geometry, %d) AS geom
			FROM point
		),
		stations AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(NULLIF(public_transport, ''), NULLIF(railway, ''), 'station') AS type,
				ST_AsMVTGeom(
					ST_Transform(way, %d),
					circle.geom,
					$4, $5, true
				) AS geom
			FROM %s, circle
			WHERE (public_transport IS NOT NULL OR railway IN ('station', 'halt', 'stop'))
			  AND ST_Transform(way, %d) && circle.geom
			  AND ST_Contains(circle.geom, ST_Transform(way, %d))
			ORDER BY name
			LIMIT $6
		)
		SELECT COALESCE(ST_AsMVT(stations.*, 'transport_stations'), '\\x'::bytea) AS tile
		FROM stations
		WHERE geom IS NOT NULL
	`, SRID4326, SRID3857, SRID3857, planetPointTable, SRID3857, SRID3857)

	var stationsTile []byte
	err := r.db.QueryRowContext(ctx, stationsQuery, lon, lat, radiusMeters, MVTExtent, MVTBuffer, LimitStations).Scan(&stationsTile)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("failed to build osm stations radius tile", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	// Линии
	linesQuery := fmt.Sprintf(`
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d) AS geom
		),
		circle AS (
			SELECT ST_Transform(ST_Buffer(point.geom::geography, $3)::geometry, %d) AS geom
			FROM point
		),
		lines AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(ref, '') AS ref,
				COALESCE(route, '') AS type,
				COALESCE(tags->'colour', '') AS color,
				ST_AsMVTGeom(
					ST_Intersection(ST_Transform(way, %d), circle.geom),
					circle.geom,
					$4, $5, true
				) AS geom
			FROM %s, circle
			WHERE route IS NOT NULL
			  AND ST_Transform(way, %d) && circle.geom
			  AND ST_Intersects(ST_Transform(way, %d), circle.geom)
			ORDER BY name
			LIMIT $6
		)
		SELECT COALESCE(ST_AsMVT(lines.*, 'transport_lines'), '\\x'::bytea) AS tile
		FROM lines
		WHERE geom IS NOT NULL
	`, SRID4326, SRID3857, SRID3857, planetLineTable, SRID3857, SRID3857)

	var linesTile []byte
	err = r.db.QueryRowContext(ctx, linesQuery, lon, lat, radiusMeters, MVTExtent, MVTBuffer, LimitLines).Scan(&linesTile)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("failed to build osm lines radius tile", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	// Объединяем тайлы
	result := append(stationsTile, linesTile...)
	return result, nil
}

// GetTransportTileByTypes генерирует MVT тайл для транспорта с фильтрацией по типам
func (r *transportRepository) GetTransportTileByTypes(ctx context.Context, z, x, y int, types []string) ([]byte, error) {
	args := []interface{}{z, x, y, MVTExtent, MVTBuffer}

	// Построение фильтра станций из типов с использованием buildTransportTypeFilter
	stationTypeFilter := ""
	if len(types) > 0 {
		filters := make([]string, 0, len(types))
		for _, t := range types {
			filters = append(filters, buildTransportTypeFilter(t))
		}
		stationTypeFilter = " AND (" + strings.Join(filters, " OR ") + ")"
	}

	// Построение фильтра линий (маппинг типов на route значения OSM)
	lineTypeFilter := ""
	if len(types) > 0 {
		routeValues := make([]string, 0, len(types))
		for _, t := range types {
			switch t {
			case "metro", "subway":
				routeValues = append(routeValues, "'subway'")
			case "bus":
				routeValues = append(routeValues, "'bus'")
			case "tram", "light_rail":
				routeValues = append(routeValues, "'tram'", "'light_rail'")
			case "train", "rail", "cercania", "long_distance":
				routeValues = append(routeValues, "'train'")
			case "ferry":
				routeValues = append(routeValues, "'ferry'")
			}
		}
		if len(routeValues) > 0 {
			lineTypeFilter = fmt.Sprintf(" AND route IN (%s)", strings.Join(routeValues, ","))
		}
	}

	// Станции
	stationsQuery := fmt.Sprintf(`
		WITH bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		stations AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				CASE
					WHEN railway = 'station' AND (tags->'station' = 'subway' OR tags->'subway' = 'yes') THEN 'subway'
					WHEN railway = 'tram_stop' OR (railway = 'station' AND tags->'station' = 'light_rail') THEN 'tram_stop'
					WHEN highway = 'bus_stop' OR (public_transport IN ('platform', 'stop_position') AND tags->'bus' = 'yes') THEN 'bus_stop'
					WHEN railway IN ('station', 'halt') THEN 'station'
					WHEN public_transport IS NOT NULL THEN COALESCE(NULLIF(public_transport, ''), 'stop')
					ELSE 'station'
				END AS type,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM %s, bounds
			WHERE (public_transport IS NOT NULL OR railway IN ('station', 'halt', 'stop') OR highway = 'bus_stop')
			  AND way && bounds.geom%s
		)
		SELECT COALESCE(ST_AsMVT(stations.*, 'stations'), '\\x'::bytea) AS tile
		FROM stations
		WHERE geom IS NOT NULL
	`, planetPointTable, stationTypeFilter)

	var stationsTile []byte
	err := r.db.QueryRowContext(ctx, stationsQuery, args...).Scan(&stationsTile)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("failed to build osm stations tile by types", zap.Int("z", z), zap.Int("x", x), zap.Int("y", y), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	// Линии
	linesQuery := fmt.Sprintf(`
		WITH bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		lines AS (
			SELECT 
				osm_id AS id,
				COALESCE(name, '') AS name,
				COALESCE(ref, '') AS ref,
				COALESCE(route, '') AS type,
				COALESCE(tags->'colour', '') AS color,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM %s, bounds
			WHERE route IS NOT NULL
			  AND way && bounds.geom%s
		)
		SELECT COALESCE(ST_AsMVT(lines.*, 'lines'), '\\x'::bytea) AS tile
		FROM lines
		WHERE geom IS NOT NULL
	`, planetLineTable, lineTypeFilter)

	var linesTile []byte
	err = r.db.QueryRowContext(ctx, linesQuery, args...).Scan(&linesTile)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("failed to build osm lines tile by types", zap.Int("z", z), zap.Int("x", x), zap.Int("y", y), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	result := append(stationsTile, linesTile...)
	return result, nil
}

// GetLinesByStationID возвращает линии метро/поезда для станции
// Группирует по ref чтобы убрать дубли направлений (L3 туда и обратно = одна линия L3)
// ОПТИМИЗАЦИЯ: использует way (SRID 3857) для быстрого пространственного поиска
func (r *transportRepository) GetLinesByStationID(ctx context.Context, stationID int64) ([]*domain.TransportLine, error) {
	// В OSM данных линии и станции не связаны напрямую через foreign key.
	// Для определения линий станции используем пространственную близость.
	// Фильтруем только линии метро, поезда, легкого метро.
	// Группируем по ref чтобы L3 в обе стороны считалась как одна линия.
	// Используем way (SRID 3857) для быстрого пространственного поиска через индекс.
	query := fmt.Sprintf(`
		WITH station AS (
			SELECT way, tags->'subway' as is_subway, tags->'station' as station_type
			FROM %s WHERE osm_id = $1
		),
		lines_nearby AS (
			SELECT DISTINCT ON (COALESCE(NULLIF(l.ref, ''), l.name))
				l.osm_id,
				COALESCE(l.ref, '') AS ref,
				COALESCE(l.tags->'colour', '') AS color,
				COALESCE(l.route, '') AS route_type
			FROM %s l, station s
			WHERE l.route IN ('subway', 'light_rail', 'train')
			  AND l.ref IS NOT NULL AND l.ref != ''
			  AND ST_DWithin(l.way, s.way, 100)
			ORDER BY COALESCE(NULLIF(l.ref, ''), l.name), l.osm_id
		)
		SELECT osm_id, ref, color, route_type
		FROM lines_nearby
		ORDER BY ref
		LIMIT %d
	`, planetPointTable, planetLineTable, LimitLines)

	rows, err := r.db.QueryxContext(ctx, query, stationID)
	if err != nil {
		r.logger.Error("failed to get lines by station", zap.Int64("station_id", stationID), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var lines []*domain.TransportLine
	for rows.Next() {
		var line domain.TransportLine
		var color string
		err := rows.Scan(&line.OSMId, &line.Ref, &color, &line.Type)
		if err != nil {
			r.logger.Error("failed to scan line row", zap.Error(err))
			continue
		}
		line.ID = line.OSMId
		line.Name = line.Ref // Используем ref как name (L3, L5, S1 и т.д.)
		if color != "" {
			line.Color = &color
		}
		lines = append(lines, &line)
	}

	return lines, nil
}

// GetNearestStationsGrouped возвращает ближайшие станции транспорта с группировкой
// по нормализованному имени. Это исключает дубли выходов метро (считается как одна станция).
func (r *transportRepository) GetNearestStationsGrouped(
	ctx context.Context,
	lat, lon float64,
	priorities []domain.TransportPriority,
	maxDistance float64,
) ([]*domain.TransportStation, error) {
	var allStations []*domain.TransportStation

	// Для каждого типа транспорта получаем станции с учетом приоритета
	for _, priority := range priorities {
		stations, err := r.getGroupedStationsByType(ctx, lat, lon, priority.Type, maxDistance, priority.Limit)
		if err != nil {
			r.logger.Warn("failed to get osm stations for type",
				zap.String("type", priority.Type),
				zap.Error(err))
			continue
		}
		allStations = append(allStations, stations...)
	}

	return allStations, nil
}

// getGroupedStationsByType получает станции одного типа с группировкой по нормализованному имени
func (r *transportRepository) getGroupedStationsByType(
	ctx context.Context,
	lat, lon float64,
	transportType string,
	maxDistance float64,
	limit int,
) ([]*domain.TransportStation, error) {
	// Определяем условие фильтрации по типу транспорта
	typeFilter := buildTransportTypeFilter(transportType)

	// SQL запрос с группировкой по нормализованному имени
	// Удаляет дубли выходов метро (например, разные выходы одной станции)
	query := fmt.Sprintf(`
		SELECT DISTINCT ON (normalized_name) 
			osm_id, name, name_en, type, lat, lon, distance
		FROM (
			SELECT 
				osm_id,
				COALESCE(name, '') AS name,
				COALESCE(NULLIF(tags->'name:en', ''), name, '') AS name_en,
				COALESCE(NULLIF(public_transport, ''), NULLIF(railway, ''), 'station') AS type,
				ST_Y(ST_Transform(way, %d)) AS lat,
				ST_X(ST_Transform(way, %d)) AS lon,
				ST_Distance(ST_Transform(way, %d)::geography, ST_SetSRID(ST_MakePoint($1, $2), %d)::geography) AS distance,
				LOWER(REGEXP_REPLACE(COALESCE(name, ''), '[^a-zA-Zа-яА-Я0-9]', '', 'g')) AS normalized_name
			FROM %s
			WHERE %s
			  AND name IS NOT NULL AND name != ''
			  AND ST_DWithin(ST_Transform(way, %d)::geography, ST_SetSRID(ST_MakePoint($1, $2), %d)::geography, $3)
			ORDER BY distance
		) sub
		WHERE normalized_name != ''
		ORDER BY normalized_name, distance
		LIMIT $4
	`, SRID4326, SRID4326, SRID4326, SRID4326, planetPointTable, typeFilter, SRID4326, SRID4326)

	rows, err := r.db.QueryxContext(ctx, query, lon, lat, maxDistance, limit)
	if err != nil {
		r.logger.Error("failed to execute osm grouped stations query",
			zap.String("type", transportType),
			zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	var stations []*domain.TransportStation
	for rows.Next() {
		var s domain.TransportStation
		var distance float64

		err := rows.Scan(
			&s.OSMId, &s.Name, &s.NameEn, &s.Type,
			&s.Lat, &s.Lon, &distance,
		)
		if err != nil {
			r.logger.Error("failed to scan osm station", zap.Error(err))
			continue
		}

		// В OSM данных ID = OSM ID
		s.ID = s.OSMId
		// Сохраняем дистанцию из БД
		s.Distance = &distance
		// LineIDs в OSM не связаны напрямую, оставляем пустым
		s.LineIDs = []int64{}
		s.Tags = make(map[string]string)

		stations = append(stations, &s)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("error iterating osm stations", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	return stations, nil
}

// buildTransportTypeFilter строит SQL условие фильтрации по типу транспорта
func buildTransportTypeFilter(transportType string) string {
	switch transportType {
	case "metro", "subway":
		// Метро: только станции (railway=station + subway=yes), не входы
		return `(
			(railway = 'station' AND (tags->'station' = 'subway' OR tags->'subway' = 'yes'))
			OR (public_transport = 'station' AND (tags->'subway' = 'yes' OR tags->'station' = 'subway'))
		)`
	case "train", "rail":
		// Железнодорожные станции
		return `(
			railway IN ('station', 'halt') 
			AND (tags->'station' IS NULL OR tags->'station' NOT IN ('subway', 'light_rail'))
			AND (tags->'subway' IS NULL OR tags->'subway' != 'yes')
		)`
	case "tram", "light_rail":
		// Трамвай / легкое метро
		return `(
			railway = 'tram_stop'
			OR (railway = 'station' AND tags->'station' = 'light_rail')
			OR public_transport = 'stop_position' AND tags->'tram' = 'yes'
		)`
	case "bus":
		// Автобусные остановки
		return `(
			highway = 'bus_stop'
			OR public_transport = 'platform' AND tags->'bus' = 'yes'
			OR public_transport = 'stop_position' AND tags->'bus' = 'yes'
		)`
	case "ferry":
		// Паромные терминалы
		return `(
			amenity = 'ferry_terminal'
			OR public_transport = 'station' AND tags->'ferry' = 'yes'
		)`
	default:
		// Общий фильтр для всех станций
		return `(
			public_transport IS NOT NULL 
			OR railway IN ('station', 'halt', 'stop', 'tram_stop', 'subway_entrance')
		)`
	}
}

// GetNearestStationsBatch возвращает ближайшие станции для пачки координат одним запросом.
// Не включает информацию о линиях - используйте GetLinesByStationIDsBatch для получения линий.
func (r *transportRepository) GetNearestStationsBatch(
	ctx context.Context,
	req domain.BatchTransportRequest,
) ([]domain.TransportStationWithLines, error) {
	if len(req.Points) == 0 {
		return []domain.TransportStationWithLines{}, nil
	}

	maxDistance := req.MaxDistance
	if maxDistance <= 0 {
		maxDistance = 1500 // default 1.5km
	}

	// Шаг 1: Построить CTE для всех точек поиска
	// Формат: point_idx, lon, lat, types[], limit
	pointsCTE := r.buildPointsCTE(req.Points)

	// Шаг 2: Один запрос для получения ближайших станций для всех точек
	stationsQuery := fmt.Sprintf(`
		WITH search_points AS (
			%s
		),
		-- Находим ближайшие станции для каждой точки
		nearest_stations AS (
			SELECT DISTINCT ON (sp.point_idx, normalized_name)
				sp.point_idx,
				p.osm_id AS station_id,
				COALESCE(p.name, '') AS name,
				COALESCE(NULLIF(p.public_transport, ''), NULLIF(p.railway, ''), 'station') AS type,
				ST_Y(ST_Transform(p.way, %d)) AS lat,
				ST_X(ST_Transform(p.way, %d)) AS lon,
				ST_Distance(
					ST_Transform(p.way, %d)::geography, 
					ST_SetSRID(ST_MakePoint(sp.lon, sp.lat), %d)::geography
				) AS distance,
				LOWER(REGEXP_REPLACE(COALESCE(p.name, ''), '[^a-zA-Zа-яА-Я0-9]', '', 'g')) AS normalized_name,
				sp.limit_per_point
			FROM %s p
			CROSS JOIN search_points sp
			WHERE p.name IS NOT NULL AND p.name != ''
			  AND ST_DWithin(
				  ST_Transform(p.way, %d)::geography, 
				  ST_SetSRID(ST_MakePoint(sp.lon, sp.lat), %d)::geography, 
				  $1
			  )
			  AND (
				  -- Фильтр по типу транспорта
				  (sp.transport_type = 'metro' AND (
					  (p.railway = 'station' AND (p.tags->'station' = 'subway' OR p.tags->'subway' = 'yes'))
					  OR (p.public_transport = 'station' AND (p.tags->'subway' = 'yes' OR p.tags->'station' = 'subway'))
				  ))
				  OR (sp.transport_type = 'train' AND (
					  p.railway IN ('station', 'halt') 
					  AND (p.tags->'station' IS NULL OR p.tags->'station' NOT IN ('subway', 'light_rail'))
					  AND (p.tags->'subway' IS NULL OR p.tags->'subway' != 'yes')
				  ))
				  OR (sp.transport_type = 'tram' AND (
					  p.railway = 'tram_stop'
					  OR (p.railway = 'station' AND p.tags->'station' = 'light_rail')
				  ))
				  OR (sp.transport_type = 'bus' AND (
					  p.highway = 'bus_stop'
					  OR (p.public_transport = 'platform' AND p.tags->'bus' = 'yes')
					  OR (p.public_transport = 'stop_position' AND p.tags->'bus' = 'yes')
				  ))
				  OR (sp.transport_type = 'ferry' AND (
					  p.amenity = 'ferry_terminal'
					  OR (p.public_transport = 'station' AND p.tags->'ferry' = 'yes')
				  ))
				  OR (sp.transport_type = '' AND (
					  p.public_transport IS NOT NULL 
					  OR p.railway IN ('station', 'halt', 'stop', 'tram_stop')
				  ))
			  )
			ORDER BY sp.point_idx, normalized_name, distance
		),
		-- Ранжируем станции по расстоянию для каждой точки и отбираем по лимиту
		ranked_stations AS (
			SELECT 
				point_idx,
				station_id,
				name,
				type,
				lat,
				lon,
				distance,
				ROW_NUMBER() OVER (PARTITION BY point_idx ORDER BY distance) AS rn,
				limit_per_point
			FROM nearest_stations
			WHERE normalized_name != ''
		)
		SELECT point_idx, station_id, name, type, lat, lon, distance
		FROM ranked_stations
		WHERE rn <= limit_per_point
		ORDER BY point_idx, distance
	`, pointsCTE, SRID4326, SRID4326, SRID4326, SRID4326, planetPointTable, SRID4326, SRID4326)

	r.logger.Debug("Executing batch stations query", zap.Int("points_count", len(req.Points)))

	rows, err := r.db.QueryxContext(ctx, stationsQuery, maxDistance)
	if err != nil {
		r.logger.Error("failed to execute batch stations query", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	// Собираем станции и их ID для последующего запроса линий
	var stations []domain.TransportStationWithLines
	stationIDs := make([]int64, 0)
	stationIDSet := make(map[int64]bool)

	for rows.Next() {
		var s domain.TransportStationWithLines
		err := rows.Scan(&s.PointIdx, &s.StationID, &s.Name, &s.Type, &s.Lat, &s.Lon, &s.Distance)
		if err != nil {
			r.logger.Error("failed to scan batch station row", zap.Error(err))
			continue
		}
		stations = append(stations, s)
		if !stationIDSet[s.StationID] {
			stationIDs = append(stationIDs, s.StationID)
			stationIDSet[s.StationID] = true
		}
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("error iterating batch stations", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	r.logger.Debug("Fetched batch stations", zap.Int("stations_count", len(stations)), zap.Int("unique_stations", len(stationIDs)))

	return stations, nil
}

// buildPointsCTE строит CTE с точками поиска для batch-запроса
func (r *transportRepository) buildPointsCTE(points []domain.TransportSearchPoint) string {
	var parts []string
	for i, p := range points {
		transportType := ""
		if len(p.Types) > 0 {
			transportType = p.Types[0] // используем первый тип (можно расширить для множественных)
		}
		limit := p.Limit
		if limit <= 0 {
			limit = 3
		}
		parts = append(parts, fmt.Sprintf(
			"SELECT %d AS point_idx, %f AS lon, %f AS lat, '%s' AS transport_type, %d AS limit_per_point",
			i, p.Lon, p.Lat, transportType, limit,
		))
	}
	return strings.Join(parts, " UNION ALL ")
}

// GetNearestTransportByPriority возвращает ближайший транспорт с приоритетом по типу и расстоянию.
// Приоритет: 1) metro/train - высокий приоритет, 2) tram/bus - добавляются если высокоприоритетных < лимита.
// Возвращает станции с информацией о линиях (для метро: L2, L4 и их цвета; для автобусов: номера маршрутов).
// Использует предвычисленную колонку way_geog для оптимальной производительности.
func (r *transportRepository) GetNearestTransportByPriority(
	ctx context.Context,
	lat, lon float64,
	radiusM float64,
	limit int,
) ([]domain.NearestTransportWithLines, error) {
	if limit <= 0 || limit > LimitStations {
		limit = LimitStations
	}
	if radiusM <= 0 {
		radiusM = 1500 // default 1.5km
	}

	// SQL запрос с приоритизацией и заполнением до лимита:
	// 1. Сначала берём все metro/train (высокий приоритет)
	// 2. Если их меньше лимита - добирваем bus/tram до лимита
	// Группируем по нормализованному имени чтобы убрать дубликаты выходов
	// Используем way_geog (предвычисленная geography колонка) для быстрого пространственного поиска
	query := fmt.Sprintf(`
		WITH search_point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), %d)::geography AS geom
		),
		-- Все станции в радиусе с типами и приоритетами
		all_stations AS (
			SELECT DISTINCT ON (normalized_name)
				osm_id AS station_id,
				COALESCE(name, '') AS name,
				COALESCE(NULLIF(tags->'name:en', ''), name, '') AS name_en,
				CASE 
					WHEN railway = 'station' AND (tags->'station' = 'subway' OR tags->'subway' = 'yes') THEN 'metro'
					WHEN railway IN ('station', 'halt') AND (tags->'station' IS NULL OR tags->'station' NOT IN ('subway', 'light_rail')) THEN 'train'
					WHEN railway = 'tram_stop' OR (railway = 'station' AND tags->'station' = 'light_rail') THEN 'tram'
					WHEN highway = 'bus_stop' OR (public_transport IN ('platform', 'stop_position') AND tags->'bus' = 'yes') THEN 'bus'
					WHEN amenity = 'ferry_terminal' THEN 'ferry'
					ELSE 'other'
				END AS transport_type,
				ST_Y(way_geog::geometry) AS lat,
				ST_X(way_geog::geometry) AS lon,
				ST_Distance(way_geog, sp.geom) AS distance,
				LOWER(REGEXP_REPLACE(COALESCE(name, ''), '[^a-zA-Zа-яА-Я0-9]', '', 'g')) AS normalized_name,
				CASE 
					WHEN railway = 'station' AND (tags->'station' = 'subway' OR tags->'subway' = 'yes') THEN 1
					WHEN railway IN ('station', 'halt') THEN 1
					WHEN railway = 'tram_stop' THEN 2
					WHEN highway = 'bus_stop' OR public_transport IN ('platform', 'stop_position') THEN 2
					ELSE 3
				END AS priority_rank
			FROM %s, search_point sp
			WHERE name IS NOT NULL AND name != ''
			  AND ST_DWithin(way_geog, sp.geom, $3)
			  AND (
				  -- Metro stations
				  (railway = 'station' AND (tags->'station' = 'subway' OR tags->'subway' = 'yes'))
				  -- Train stations
				  OR (railway IN ('station', 'halt') AND (tags->'station' IS NULL OR tags->'station' NOT IN ('subway', 'light_rail')))
				  -- Tram stops
				  OR railway = 'tram_stop'
				  -- Bus stops
				  OR highway = 'bus_stop'
				  OR (public_transport IN ('platform', 'stop_position') AND tags->'bus' = 'yes')
			  )
			ORDER BY normalized_name, distance
		),
		-- Высокоприоритетные станции (metro/train)
		high_priority AS (
			SELECT station_id, name, name_en, transport_type, lat, lon, distance, priority_rank,
				   ROW_NUMBER() OVER (ORDER BY distance) AS rn
			FROM all_stations
			WHERE priority_rank = 1
		),
		-- Низкоприоритетные станции (tram/bus)
		low_priority AS (
			SELECT station_id, name, name_en, transport_type, lat, lon, distance, priority_rank,
				   ROW_NUMBER() OVER (ORDER BY distance) AS rn
			FROM all_stations
			WHERE priority_rank = 2
		),
		-- Количество высокоприоритетных
		high_count AS (
			SELECT COUNT(*) AS cnt FROM high_priority WHERE rn <= $4
		),
		-- Объединяем: сначала все high_priority до лимита, потом low_priority чтобы добить до лимита
		combined AS (
			SELECT station_id, name, name_en, transport_type, lat, lon, distance, priority_rank, rn
			FROM high_priority
			WHERE rn <= $4
			UNION ALL
			SELECT station_id, name, name_en, transport_type, lat, lon, distance, priority_rank, rn + (SELECT cnt FROM high_count)
			FROM low_priority
			WHERE rn <= $4 - (SELECT cnt FROM high_count)
		)
		SELECT station_id, name, name_en, transport_type, lat, lon, distance
		FROM combined
		ORDER BY priority_rank, distance
		LIMIT $4
	`, SRID4326, planetPointTable)

	rows, err := r.db.QueryxContext(ctx, query, lon, lat, radiusM, limit)
	if err != nil {
		r.logger.Error("failed to get nearest transport by priority", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	// Собираем станции
	var stations []domain.NearestTransportWithLines
	var stationIDs []int64

	for rows.Next() {
		var s domain.NearestTransportWithLines
		var nameEn string
		err := rows.Scan(&s.StationID, &s.Name, &nameEn, &s.Type, &s.Lat, &s.Lon, &s.Distance)
		if err != nil {
			r.logger.Error("failed to scan station row", zap.Error(err))
			continue
		}
		if nameEn != "" && nameEn != s.Name {
			s.NameEn = &nameEn
		}
		stations = append(stations, s)
		stationIDs = append(stationIDs, s.StationID)
	}

	if len(stations) == 0 {
		return stations, nil
	}

	// Получаем линии для всех станций одним запросом
	linesMap, err := r.GetLinesByStationIDsBatch(ctx, stationIDs)
	if err != nil {
		r.logger.Warn("failed to get lines for stations", zap.Error(err))
		// Продолжаем без линий
	}

	// Добавляем линии к станциям
	for i := range stations {
		if lines, ok := linesMap[stations[i].StationID]; ok {
			stations[i].Lines = lines
		}
	}

	return stations, nil
}

// GetNearestTransportByPriorityBatch возвращает ближайший транспорт с приоритетом для множества точек одним запросом.
// Для каждой точки: сначала metro/train, потом добираем bus/tram до лимита.
// Использует предвычисленную колонку way_geog для оптимальной производительности.
func (r *transportRepository) GetNearestTransportByPriorityBatch(
	ctx context.Context,
	points []domain.TransportSearchPoint,
	radiusM float64,
	limitPerPoint int,
) ([]domain.BatchTransportResult, error) {
	if len(points) == 0 {
		return []domain.BatchTransportResult{}, nil
	}
	if limitPerPoint <= 0 || limitPerPoint > 10 {
		limitPerPoint = 5
	}
	if radiusM <= 0 {
		radiusM = 1500
	}

	// Строим VALUES для всех точек
	var valuesParts []string
	for i, p := range points {
		valuesParts = append(valuesParts, fmt.Sprintf("(%d, %f, %f)", i, p.Lon, p.Lat))
	}
	valuesSQL := strings.Join(valuesParts, ", ")

	query := fmt.Sprintf(`
		WITH search_points(point_idx, lon, lat) AS (
			VALUES %s
		),
		-- Все станции в радиусе для каждой точки
		all_stations AS (
			SELECT DISTINCT ON (sp.point_idx, normalized_name)
				sp.point_idx,
				p.osm_id AS station_id,
				COALESCE(p.name, '') AS name,
				COALESCE(NULLIF(p.tags->'name:en', ''), p.name, '') AS name_en,
				CASE 
					WHEN p.railway = 'station' AND (p.tags->'station' = 'subway' OR p.tags->'subway' = 'yes') THEN 'metro'
					WHEN p.railway IN ('station', 'halt') AND (p.tags->'station' IS NULL OR p.tags->'station' NOT IN ('subway', 'light_rail')) THEN 'train'
					WHEN p.railway = 'tram_stop' OR (p.railway = 'station' AND p.tags->'station' = 'light_rail') THEN 'tram'
					WHEN p.highway = 'bus_stop' OR (p.public_transport IN ('platform', 'stop_position') AND p.tags->'bus' = 'yes') THEN 'bus'
					ELSE 'other'
				END AS transport_type,
				ST_Y(p.way_geog::geometry) AS lat,
				ST_X(p.way_geog::geometry) AS lon,
				ST_Distance(
					p.way_geog, 
					ST_SetSRID(ST_MakePoint(sp.lon, sp.lat), %d)::geography
				) AS distance,
				LOWER(REGEXP_REPLACE(COALESCE(p.name, ''), '[^a-zA-Zа-яА-Я0-9]', '', 'g')) AS normalized_name,
				CASE 
					WHEN p.railway = 'station' AND (p.tags->'station' = 'subway' OR p.tags->'subway' = 'yes') THEN 1
					WHEN p.railway IN ('station', 'halt') THEN 1
					WHEN p.railway = 'tram_stop' THEN 2
					WHEN p.highway = 'bus_stop' OR p.public_transport IN ('platform', 'stop_position') THEN 2
					ELSE 3
				END AS priority_rank
			FROM %s p
			CROSS JOIN search_points sp
			WHERE p.name IS NOT NULL AND p.name != ''
			  AND ST_DWithin(
				  p.way_geog, 
				  ST_SetSRID(ST_MakePoint(sp.lon, sp.lat), %d)::geography, 
				  $1
			  )
			  AND (
				  (p.railway = 'station' AND (p.tags->'station' = 'subway' OR p.tags->'subway' = 'yes'))
				  OR (p.railway IN ('station', 'halt') AND (p.tags->'station' IS NULL OR p.tags->'station' NOT IN ('subway', 'light_rail')))
				  OR p.railway = 'tram_stop'
				  OR p.highway = 'bus_stop'
				  OR (p.public_transport IN ('platform', 'stop_position') AND p.tags->'bus' = 'yes')
			  )
			ORDER BY sp.point_idx, normalized_name, distance
		),
		-- Высокоприоритетные станции для каждой точки
		high_priority AS (
			SELECT point_idx, station_id, name, name_en, transport_type, lat, lon, distance, priority_rank,
				   ROW_NUMBER() OVER (PARTITION BY point_idx ORDER BY distance) AS rn
			FROM all_stations
			WHERE priority_rank = 1
		),
		-- Низкоприоритетные станции для каждой точки
		low_priority AS (
			SELECT point_idx, station_id, name, name_en, transport_type, lat, lon, distance, priority_rank,
				   ROW_NUMBER() OVER (PARTITION BY point_idx ORDER BY distance) AS rn
			FROM all_stations
			WHERE priority_rank = 2
		),
		-- Количество высокоприоритетных для каждой точки
		high_counts AS (
			SELECT point_idx, COUNT(*) AS cnt 
			FROM high_priority 
			WHERE rn <= $2 
			GROUP BY point_idx
		),
		-- Объединяем с заполнением до лимита
		combined AS (
			SELECT point_idx, station_id, name, name_en, transport_type, lat, lon, distance, priority_rank, rn
			FROM high_priority
			WHERE rn <= $2
			UNION ALL
			SELECT lp.point_idx, lp.station_id, lp.name, lp.name_en, lp.transport_type, lp.lat, lp.lon, lp.distance, lp.priority_rank, 
				   lp.rn + COALESCE(hc.cnt, 0) AS rn
			FROM low_priority lp
			LEFT JOIN high_counts hc ON lp.point_idx = hc.point_idx
			WHERE lp.rn <= $2 - COALESCE(hc.cnt, 0)
		)
		SELECT point_idx, station_id, name, name_en, transport_type, lat, lon, distance
		FROM combined
		WHERE rn <= $2
		ORDER BY point_idx, priority_rank, distance
	`, valuesSQL, SRID4326, planetPointTable, SRID4326)

	rows, err := r.db.QueryxContext(ctx, query, radiusM, limitPerPoint)
	if err != nil {
		r.logger.Error("failed to execute batch priority transport query", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	// Группируем результаты по point_idx
	resultMap := make(map[int][]domain.NearestTransportWithLines)
	var allStationIDs []int64
	stationIDSet := make(map[int64]bool)

	for rows.Next() {
		var pointIdx int
		var s domain.NearestTransportWithLines
		var nameEn string

		err := rows.Scan(&pointIdx, &s.StationID, &s.Name, &nameEn, &s.Type, &s.Lat, &s.Lon, &s.Distance)
		if err != nil {
			r.logger.Error("failed to scan batch station row", zap.Error(err))
			continue
		}
		if nameEn != "" && nameEn != s.Name {
			s.NameEn = &nameEn
		}

		resultMap[pointIdx] = append(resultMap[pointIdx], s)
		if !stationIDSet[s.StationID] {
			allStationIDs = append(allStationIDs, s.StationID)
			stationIDSet[s.StationID] = true
		}
	}

	// Получаем линии для всех станций
	linesMap, err := r.GetLinesByStationIDsBatch(ctx, allStationIDs)
	if err != nil {
		r.logger.Warn("failed to get lines for batch stations", zap.Error(err))
	}

	// Формируем результат
	results := make([]domain.BatchTransportResult, len(points))
	for i := range points {
		stations := resultMap[i]
		// Добавляем линии
		for j := range stations {
			if lines, ok := linesMap[stations[j].StationID]; ok {
				stations[j].Lines = lines
			}
		}
		results[i] = domain.BatchTransportResult{
			PointIndex:  i,
			SearchPoint: domain.Coordinate{Lat: points[i].Lat, Lon: points[i].Lon},
			Stations:    stations,
		}
	}

	return results, nil
}

// GetLinesByStationIDsBatch возвращает линии для множества станций одним запросом.
// Используется совместно с GetNearestStationsBatch для batch-обогащения.
// ОПТИМИЗАЦИЯ: использует way (SRID 3857) вместо way_geog для быстрого пространственного поиска
// через существующий GIST индекс planet_osm_line_way_idx.
func (r *transportRepository) GetLinesByStationIDsBatch(
	ctx context.Context,
	stationIDs []int64,
) (map[int64][]domain.TransportLineInfo, error) {
	if len(stationIDs) == 0 {
		return make(map[int64][]domain.TransportLineInfo), nil
	}

	// Строим плейсхолдеры для IN clause
	placeholders := make([]string, len(stationIDs))
	args := make([]interface{}, len(stationIDs))
	for i, id := range stationIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}

	// Запрос: для каждой станции находим ближайшие линии через пространственную близость
	// Группируем по ref чтобы убрать дубликаты направлений (L3 туда и обратно = одна линия L3)
	// Используем way (SRID 3857) для быстрого пространственного поиска через индекс planet_osm_line_way_idx
	// 100 единиц в SRID 3857 ≈ 100 метров (Web Mercator в метрах)
	query := fmt.Sprintf(`
		WITH station_points AS (
			SELECT osm_id, way 
			FROM %s 
			WHERE osm_id IN (%s)
		),
		station_lines AS (
			SELECT DISTINCT ON (sp.osm_id, COALESCE(NULLIF(l.ref, ''), l.name))
				sp.osm_id AS station_id,
				l.osm_id AS line_id,
				COALESCE(l.ref, l.name, '') AS name,
				COALESCE(l.ref, '') AS ref,
				COALESCE(l.route, '') AS line_type,
				COALESCE(l.tags->'colour', '') AS color
			FROM station_points sp
			JOIN %s l ON ST_DWithin(l.way, sp.way, 100)
			WHERE l.route IN ('subway', 'light_rail', 'train', 'tram', 'bus')
			  AND (l.ref IS NOT NULL AND l.ref != '' OR l.name IS NOT NULL AND l.name != '')
			ORDER BY sp.osm_id, COALESCE(NULLIF(l.ref, ''), l.name), l.osm_id
		)
		SELECT station_id, line_id, name, ref, line_type, color
		FROM station_lines
		ORDER BY station_id, ref
	`, planetPointTable, strings.Join(placeholders, ","), planetLineTable)

	rows, err := r.db.QueryxContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("failed to get batch lines for stations", zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}
	defer rows.Close()

	result := make(map[int64][]domain.TransportLineInfo)
	seenLines := make(map[int64]map[string]bool) // station_id -> ref -> seen

	for rows.Next() {
		var stationID, lineID int64
		var name, ref, lineType, color string

		err := rows.Scan(&stationID, &lineID, &name, &ref, &lineType, &color)
		if err != nil {
			r.logger.Error("failed to scan line row", zap.Error(err))
			continue
		}

		// Дедупликация по ref для каждой станции
		if seenLines[stationID] == nil {
			seenLines[stationID] = make(map[string]bool)
		}
		refKey := ref
		if refKey == "" {
			refKey = name
		}
		if seenLines[stationID][refKey] {
			continue
		}
		seenLines[stationID][refKey] = true

		lineInfo := domain.TransportLineInfo{
			ID:   lineID,
			Name: name,
			Ref:  ref,
			Type: lineType,
		}
		if color != "" {
			lineInfo.Color = &color
		}

		result[stationID] = append(result[stationID], lineInfo)
	}

	return result, nil
}
