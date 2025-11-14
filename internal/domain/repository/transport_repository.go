package repository

import (
	"context"

	"github.com/location-microservice/internal/domain"
)

// TransportRepository определяет методы для работы с транспортом
type TransportRepository interface {
	// GetNearestStations возвращает ближайшие станции
	GetNearestStations(ctx context.Context, lat, lon float64, types []string, maxDistance float64, limit int) ([]*domain.TransportStation, error)

	// GetLineByID возвращает линию по ID
	GetLineByID(ctx context.Context, id int64) (*domain.TransportLine, error)

	// GetLinesByIDs возвращает линии по списку ID
	GetLinesByIDs(ctx context.Context, ids []int64) ([]*domain.TransportLine, error)

	// GetStationsByLineID возвращает все станции для линии
	GetStationsByLineID(ctx context.Context, lineID int64) ([]*domain.TransportStation, error)

	// GetTransportTile генерирует MVT тайл для транспорта
	GetTransportTile(ctx context.Context, z, x, y int) ([]byte, error)

	// GetLineTile генерирует MVT тайл для одной транспортной линии
	GetLineTile(ctx context.Context, lineID int64) ([]byte, error)

	// GetLinesTile генерирует MVT тайл для нескольких транспортных линий
	GetLinesTile(ctx context.Context, lineIDs []int64) ([]byte, error)

	// GetStationsInRadius возвращает станции в радиусе от точки (для использования в коде)
	GetStationsInRadius(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.TransportStation, error)

	// GetLinesInRadius возвращает линии пересекающиеся с радиусом от точки (для использования в коде)
	GetLinesInRadius(ctx context.Context, lat, lon, radiusKm float64) ([]*domain.TransportLine, error)

	// GetTransportRadiusTile генерирует MVT тайл с транспортом в радиусе от точки
	GetTransportRadiusTile(ctx context.Context, lat, lon, radiusKm float64) ([]byte, error)
}
