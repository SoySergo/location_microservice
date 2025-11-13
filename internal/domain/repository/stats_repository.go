package repository

import (
	"context"

	"github.com/location-microservice/internal/domain"
)

// StatsRepository интерфейс для работы со статистикой
type StatsRepository interface {
	// GetStatistics возвращает агрегированную статистику по всем данным
	GetStatistics(ctx context.Context) (*domain.Statistics, error)

	// RefreshStatistics обновляет кешированную статистику
	RefreshStatistics(ctx context.Context) error
}
