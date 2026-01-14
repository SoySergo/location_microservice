package repository

import (
	"context"

	"github.com/location-microservice/internal/domain"
)

// TransportRepository определяет методы для работы с транспортом
type TransportRepository interface {
	// GetNearestStations возвращает ближайшие станции
	GetNearestStations(ctx context.Context, lat, lon float64, types []string, maxDistance float64, limit int) ([]*domain.TransportStation, error)

	// GetNearestStationsGrouped возвращает ближайшие станции транспорта с группировкой
	// по нормализованному имени. Это исключает дубли выходов метро (считается как одна станция).
	GetNearestStationsGrouped(ctx context.Context, lat, lon float64, priorities []domain.TransportPriority, maxDistance float64) ([]*domain.TransportStation, error)

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

	// GetTransportTileByTypes генерирует MVT тайл для транспорта с фильтрацией по типам
	GetTransportTileByTypes(ctx context.Context, z, x, y int, types []string) ([]byte, error)

	// GetLinesByStationID возвращает линии для станции (для hover логики)
	GetLinesByStationID(ctx context.Context, stationID int64) ([]*domain.TransportLine, error)

	// GetNearestStationsBatch возвращает ближайшие станции для пачки координат одним запросом.
	// Не включает информацию о линиях - используйте GetLinesByStationIDsBatch для получения линий.
	GetNearestStationsBatch(ctx context.Context, req domain.BatchTransportRequest) ([]domain.TransportStationWithLines, error)

	// GetLinesByStationIDsBatch возвращает линии для множества станций одним запросом.
	// Используется совместно с GetNearestStationsBatch для batch-обогащения.
	GetLinesByStationIDsBatch(ctx context.Context, stationIDs []int64) (map[int64][]domain.TransportLineInfo, error)

	// GetNearestTransportByPriority возвращает ближайший транспорт с приоритетом по типу и расстоянию.
	// Приоритет: metro/train -> bus/tram. Включает информацию о линиях.
	GetNearestTransportByPriority(ctx context.Context, lat, lon float64, radiusM float64, limit int) ([]domain.NearestTransportWithLines, error)

	// GetNearestTransportByPriorityBatch возвращает ближайший транспорт с приоритетом для множества точек.
	// Один SQL запрос для всех точек с применением логики приоритизации.
	GetNearestTransportByPriorityBatch(ctx context.Context, points []domain.TransportSearchPoint, radiusM float64, limitPerPoint int) ([]domain.BatchTransportResult, error)
}
