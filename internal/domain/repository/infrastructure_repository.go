package repository

import (
	"context"

	"github.com/location-microservice/internal/domain"
)

// InfrastructureRepository определяет методы для работы с инфраструктурой
type InfrastructureRepository interface {
	// GetNearestTransportGrouped возвращает ближайшие станции транспорта с группировкой по нормализованному имени
	// Это исключает дубли выходов метро (считается как одна станция)
	GetNearestTransportGrouped(
		ctx context.Context,
		lat, lon float64,
		priorities []domain.TransportPriority,
		maxDistance float64,
	) ([]*domain.TransportStation, error)

	// GetNearestPOIs возвращает ближайшие POI по категориям
	GetNearestPOIs(
		ctx context.Context,
		lat, lon float64,
		categories []domain.POICategoryConfig,
		maxDistance float64,
	) ([]*domain.POI, error)
}
