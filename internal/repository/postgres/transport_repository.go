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

type transportRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewTransportRepository(db *DB) repository.TransportRepository {
	return &transportRepository{
		db:     db.DB,
		logger: db.logger,
	}
}

func (r *transportRepository) GetNearestStations(
	ctx context.Context,
	lat, lon float64,
	types []string,
	maxDistance float64,
	limit int,
) ([]*domain.TransportStation, error) {
	query := `
		WITH point AS (
			SELECT ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography AS geom
		),
		nearest_stations AS (
			SELECT DISTINCT ON (s.id)
				s.id, s.osm_id, s.name, s.name_en, s.type,
				s.lat, s.lon, s.line_ids, s.operator, s.network, s.wheelchair,
				ST_Distance(s.geometry::geography, point.geom) AS distance
			FROM transport_stations s, point
			WHERE s.type = ANY($3)
			  AND ST_DWithin(s.geometry::geography, point.geom, $4)
			ORDER BY s.id, distance
		)
		SELECT *
		FROM nearest_stations
		ORDER BY distance
		LIMIT $5
	`

	rows, err := r.db.QueryContext(ctx, query, lon, lat, types, maxDistance, limit)
	if err != nil {
		r.logger.Error("Failed to get nearest stations", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var stations []*domain.TransportStation
	for rows.Next() {
		var s domain.TransportStation
		var lineIDs []string
		var distance float64

		err := rows.Scan(
			&s.ID, &s.OSMId, &s.Name, &s.NameEn, &s.Type,
			&s.Lat, &s.Lon, &lineIDs, &s.Operator, &s.Network, &s.Wheelchair,
			&distance,
		)
		if err != nil {
			r.logger.Error("Failed to scan station", zap.Error(err))
			continue
		}

		s.LineIDs = lineIDs
		stations = append(stations, &s)
	}

	return stations, nil
}

func (r *transportRepository) GetLineByID(ctx context.Context, id string) (*domain.TransportLine, error) {
	query := `
		SELECT 
			id, osm_id, name, ref, type, color, text_color,
			operator, network, from_station, to_station, station_ids, tags
		FROM transport_lines
		WHERE id = $1
	`

	var line domain.TransportLine
	var stationIDs []string

	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&line.ID, &line.OSMId, &line.Name, &line.Ref, &line.Type,
		&line.Color, &line.TextColor, &line.Operator, &line.Network,
		&line.FromStation, &line.ToStation, &stationIDs, &line.Tags,
	)

	if err == sql.ErrNoRows {
		return nil, errors.ErrLocationNotFound
	}
	if err != nil {
		r.logger.Error("Failed to get line by ID", zap.String("id", id), zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	line.StationIDs = stationIDs
	return &line, nil
}

func (r *transportRepository) GetLinesByIDs(ctx context.Context, ids []string) ([]*domain.TransportLine, error) {
	query := `
		SELECT 
			id, osm_id, name, ref, type, color, text_color,
			operator, network, from_station, to_station, station_ids
		FROM transport_lines
		WHERE id = ANY($1)
	`

	rows, err := r.db.QueryContext(ctx, query, ids)
	if err != nil {
		r.logger.Error("Failed to get lines by IDs", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var lines []*domain.TransportLine
	for rows.Next() {
		var l domain.TransportLine
		var stationIDs []string

		err := rows.Scan(
			&l.ID, &l.OSMId, &l.Name, &l.Ref, &l.Type,
			&l.Color, &l.TextColor, &l.Operator, &l.Network,
			&l.FromStation, &l.ToStation, &stationIDs,
		)
		if err != nil {
			r.logger.Error("Failed to scan line", zap.Error(err))
			continue
		}

		l.StationIDs = stationIDs
		lines = append(lines, &l)
	}

	return lines, nil
}

func (r *transportRepository) GetStationsByLineID(ctx context.Context, lineID string) ([]*domain.TransportStation, error) {
	query := `
		SELECT 
			id, osm_id, name, name_en, type, lat, lon,
			line_ids, operator, network, wheelchair
		FROM transport_stations
		WHERE $1 = ANY(line_ids)
		ORDER BY name
	`

	rows, err := r.db.QueryContext(ctx, query, lineID)
	if err != nil {
		r.logger.Error("Failed to get stations by line ID", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var stations []*domain.TransportStation
	for rows.Next() {
		var s domain.TransportStation
		var lineIDs []string

		err := rows.Scan(
			&s.ID, &s.OSMId, &s.Name, &s.NameEn, &s.Type,
			&s.Lat, &s.Lon, &lineIDs, &s.Operator, &s.Network, &s.Wheelchair,
		)
		if err != nil {
			r.logger.Error("Failed to scan station", zap.Error(err))
			continue
		}

		s.LineIDs = lineIDs
		stations = append(stations, &s)
	}

	return stations, nil
}

func (r *transportRepository) GetTransportTile(ctx context.Context, z, x, y int) ([]byte, error) {
	query := `
		WITH 
		bounds AS (
			SELECT ST_TileEnvelope($1, $2, $3) AS geom
		),
		stations_mvt AS (
			SELECT 
				id, name, type,
				ST_AsMVTGeom(geometry, bounds.geom, 4096, 256, true) AS geom
			FROM transport_stations, bounds
			WHERE geometry && bounds.geom
		),
		lines_mvt AS (
			SELECT 
				id, name, ref, type, color,
				ST_AsMVTGeom(geometry, bounds.geom, 4096, 256, true) AS geom
			FROM transport_lines, bounds
			WHERE geometry && bounds.geom
		)
		SELECT 
			ST_AsMVT(stations_mvt.*, 'stations') ||
			ST_AsMVT(lines_mvt.*, 'lines') AS tile
		FROM stations_mvt, lines_mvt
	`

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, z, x, y).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("Failed to generate transport tile", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return tile, nil
}

// GetLineTile генерирует MVT тайл для одной транспортной линии
func (r *transportRepository) GetLineTile(ctx context.Context, lineID string) ([]byte, error) {
	query := `
		WITH 
		line_data AS (
			SELECT 
				id, osm_id, name, ref, type, color, text_color,
				operator, network, from_station, to_station,
				geometry
			FROM transport_lines
			WHERE id = $1
		),
		line_mvt AS (
			SELECT 
				id, name, ref, type, color, text_color,
				operator, network, from_station, to_station,
				ST_AsMVTGeom(
					geometry,
					ST_Expand(ST_Envelope(geometry), 0.1),
					4096, 256, true
				) AS geom
			FROM line_data
		)
		SELECT ST_AsMVT(line_mvt.*, 'line') AS tile
		FROM line_mvt
	`

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, lineID).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("Failed to generate line tile",
			zap.String("line_id", lineID),
			zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return tile, nil
}

// GetLinesTile генерирует MVT тайл для нескольких транспортных линий
func (r *transportRepository) GetLinesTile(ctx context.Context, lineIDs []string) ([]byte, error) {
	query := `
		WITH 
		lines_data AS (
			SELECT 
				id, osm_id, name, ref, type, color, text_color,
				operator, network, from_station, to_station,
				geometry
			FROM transport_lines
			WHERE id = ANY($1)
		),
		bounds AS (
			SELECT ST_Expand(ST_Envelope(ST_Collect(geometry)), 0.1) AS geom
			FROM lines_data
		),
		lines_mvt AS (
			SELECT 
				ld.id, ld.name, ld.ref, ld.type, ld.color, ld.text_color,
				ld.operator, ld.network, ld.from_station, ld.to_station,
				ST_AsMVTGeom(ld.geometry, b.geom, 4096, 256, true) AS geom
			FROM lines_data ld, bounds b
		)
		SELECT ST_AsMVT(lines_mvt.*, 'lines') AS tile
		FROM lines_mvt
	`

	var tile []byte
	err := r.db.QueryRowContext(ctx, query, lineIDs).Scan(&tile)
	if err == sql.ErrNoRows {
		return []byte{}, nil
	}
	if err != nil {
		r.logger.Error("Failed to generate lines tile",
			zap.Strings("line_ids", lineIDs),
			zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return tile, nil
}
