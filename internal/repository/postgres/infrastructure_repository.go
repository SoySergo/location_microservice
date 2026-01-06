package postgres

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/pkg/errors"
	"go.uber.org/zap"
)

type infrastructureRepository struct {
	db     *sqlx.DB
	logger *zap.Logger
}

func NewInfrastructureRepository(db *DB) repository.InfrastructureRepository {
	return &infrastructureRepository{
		db:     db.DB,
		logger: db.logger,
	}
}

// GetNearestTransportGrouped возвращает ближайшие станции транспорта с группировкой
func (r *infrastructureRepository) GetNearestTransportGrouped(
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
			r.logger.Warn("Failed to get stations for type",
				zap.String("type", priority.Type),
				zap.Error(err))
			continue
		}
		allStations = append(allStations, stations...)
	}

	return allStations, nil
}

// getGroupedStationsByType получает станции одного типа с группировкой по нормализованному имени
func (r *infrastructureRepository) getGroupedStationsByType(
	ctx context.Context,
	lat, lon float64,
	transportType string,
	maxDistance float64,
	limit int,
) ([]*domain.TransportStation, error) {
	// SQL запрос с группировкой по нормализованному имени
	// Удаляет дубли выходов метро (например, разные выходы одной станции)
	query := `
		SELECT DISTINCT ON (normalized_name) 
			id, osm_id, name, name_en, type, lat, lon, line_ids
		FROM (
			SELECT 
				s.id, s.osm_id, s.name, s.name_en, s.type, s.lat, s.lon, s.line_ids,
				ST_Distance(s.geometry::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS distance,
				LOWER(REGEXP_REPLACE(s.name, '[^a-zA-Zа-яА-Я0-9]', '', 'g')) AS normalized_name
			FROM transport_stations s
			WHERE s.type = $3
			  AND ST_DWithin(s.geometry::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $4)
			ORDER BY distance
		) sub
		ORDER BY normalized_name, distance
		LIMIT $5
	`

	rows, err := r.db.QueryContext(ctx, query, lon, lat, transportType, maxDistance, limit)
	if err != nil {
		r.logger.Error("Failed to execute grouped stations query",
			zap.String("type", transportType),
			zap.Error(err))
		return nil, errors.ErrDatabaseError
	}
	defer rows.Close()

	var stations []*domain.TransportStation
	for rows.Next() {
		var s domain.TransportStation
		var lineIDsRaw interface{}

		err := rows.Scan(
			&s.ID, &s.OSMId, &s.Name, &s.NameEn, &s.Type,
			&s.Lat, &s.Lon, &lineIDsRaw,
		)
		if err != nil {
			r.logger.Error("Failed to scan station", zap.Error(err))
			continue
		}

		s.LineIDs, err = scanInt64Array(lineIDsRaw)
		if err != nil {
			r.logger.Error("Failed to parse line_ids array", zap.Error(err))
			continue
		}

		stations = append(stations, &s)
	}

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating stations", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return stations, nil
}

// GetNearestPOIs возвращает ближайшие POI по категориям
func (r *infrastructureRepository) GetNearestPOIs(
	ctx context.Context,
	lat, lon float64,
	categories []domain.POICategoryConfig,
	maxDistance float64,
) ([]*domain.POI, error) {
	if len(categories) == 0 {
		return []*domain.POI{}, nil
	}

	// Строим WHERE условие для категорий
	var whereClauses []string
	var args []interface{}
	argIdx := 3 // $1 и $2 для lon, lat

	for _, cat := range categories {
		argIdx++
		if cat.Subcategory != "" {
			whereClauses = append(whereClauses, fmt.Sprintf("(p.category = $%d AND p.subcategory = $%d)", argIdx, argIdx+1))
			args = append(args, cat.Category, cat.Subcategory)
			argIdx++
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("p.category = $%d", argIdx))
			args = append(args, cat.Category)
		}
	}

	whereClause := strings.Join(whereClauses, " OR ")

	// Вычисляем общий лимит (сумма лимитов всех категорий)
	totalLimit := 0
	for _, cat := range categories {
		totalLimit += cat.Limit
	}

	query := fmt.Sprintf(`
		SELECT 
			p.id, p.osm_id, p.name, p.category, p.subcategory,
			p.lat, p.lon,
			ST_Distance(p.geometry::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS distance
		FROM pois p
		WHERE (%s)
		  AND ST_DWithin(p.geometry::geography, ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $%d)
		ORDER BY distance
		LIMIT $%d
	`, whereClause, argIdx+1, argIdx+2)

	// Добавляем lon, lat в начало
	finalArgs := []interface{}{lon, lat}
	finalArgs = append(finalArgs, args...)
	finalArgs = append(finalArgs, maxDistance, totalLimit)

	rows, err := r.db.QueryContext(ctx, query, finalArgs...)
	if err != nil {
		r.logger.Error("Failed to execute POI query", zap.Error(err))
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

	if err = rows.Err(); err != nil {
		r.logger.Error("Error iterating POIs", zap.Error(err))
		return nil, errors.ErrDatabaseError
	}

	return pois, nil
}
