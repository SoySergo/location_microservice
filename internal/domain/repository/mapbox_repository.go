package repository

import (
	"context"

	"github.com/location-microservice/internal/domain"
)

// MapboxRepository определяет методы для работы с Mapbox API
type MapboxRepository interface {
	// GetWalkingMatrix возвращает матрицу пешеходных расстояний и времени
	// между источниками и пунктами назначения
	GetWalkingMatrix(
		ctx context.Context,
		origins []domain.Coordinate,
		destinations []domain.Coordinate,
	) (*domain.MatrixResponse, error)
}
