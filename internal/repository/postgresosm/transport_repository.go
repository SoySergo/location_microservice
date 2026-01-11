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
				COALESCE(NULLIF(public_transport, ''), NULLIF(railway, ''), 'station') AS type,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM %s, bounds
			WHERE (public_transport IS NOT NULL OR railway IN ('station', 'halt', 'stop'))
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
	argOffset := 6

	typeFilter := ""
	if len(types) > 0 {
		placeholders := make([]string, len(types))
		for i, t := range types {
			placeholders[i] = fmt.Sprintf("$%d", argOffset+i)
			args = append(args, t)
		}
		typeFilter = fmt.Sprintf(" AND (public_transport IN (%s) OR railway IN (%s) OR route IN (%s))",
			strings.Join(placeholders, ","), strings.Join(placeholders, ","), strings.Join(placeholders, ","))
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
				COALESCE(NULLIF(public_transport, ''), NULLIF(railway, ''), 'station') AS type,
				ST_AsMVTGeom(way, bounds.geom, $4, $5, true) AS geom
			FROM %s, bounds
			WHERE (public_transport IS NOT NULL OR railway IN ('station', 'halt', 'stop'))
			  AND way && bounds.geom%s
		)
		SELECT COALESCE(ST_AsMVT(stations.*, 'stations'), '\\x'::bytea) AS tile
		FROM stations
		WHERE geom IS NOT NULL
	`, planetPointTable, typeFilter)

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
	`, planetLineTable, typeFilter)

	var linesTile []byte
	err = r.db.QueryRowContext(ctx, linesQuery, args...).Scan(&linesTile)
	if err != nil && err != sql.ErrNoRows {
		r.logger.Error("failed to build osm lines tile by types", zap.Int("z", z), zap.Int("x", x), zap.Int("y", y), zap.Error(err))
		return nil, pkgerrors.ErrDatabaseError
	}

	result := append(stationsTile, linesTile...)
	return result, nil
}

// GetLinesByStationID возвращает линии для станции (для hover логики)
func (r *transportRepository) GetLinesByStationID(ctx context.Context, stationID int64) ([]*domain.TransportLine, error) {
	// В OSM данных линии и станции не связаны напрямую через foreign key.
	// Для определения линий станции используем пространственную близость
	query := fmt.Sprintf(`
		WITH station AS (
			SELECT way FROM %s WHERE osm_id = $1
		)
		SELECT DISTINCT
			l.osm_id,
			COALESCE(l.name, '') AS name,
			COALESCE(l.ref, '') AS ref,
			COALESCE(l.route, '') AS type,
			COALESCE(l.tags->'colour', '') AS color,
			COALESCE(l.tags->'operator', '') AS operator,
			COALESCE(l.tags->'network', '') AS network
		FROM %s l, station s
		WHERE l.route IS NOT NULL
		  AND ST_DWithin(l.way, s.way, 100)
		ORDER BY l.name
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
		var operator, network string
		err := rows.Scan(&line.OSMId, &line.Name, &line.Ref, &line.Type, &line.Color, &operator, &network)
		if err != nil {
			r.logger.Error("failed to scan line row", zap.Error(err))
			continue
		}
		line.ID = line.OSMId
		lines = append(lines, &line)
	}

	return lines, nil
}
