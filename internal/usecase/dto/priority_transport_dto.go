package dto

// This file contains DTOs for priority transport and location enrichment features.
// These types are used by production code including:
// - TransportUseCase (internal/usecase/transport_usecase.go)
// - EnrichedLocationUseCase (internal/usecase/enriched_location_usecase.go)
// - LocationEnrichmentWorker (internal/worker/location/enrichment_worker.go)
// Note: Previously named enrichment_debug_dto.go, renamed after removing debug-specific types.

// TransportLineInfoEnriched - информация о линии транспорта (расширенная)
type TransportLineInfoEnriched struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Ref   string  `json:"ref,omitempty"`
	Type  string  `json:"type,omitempty"`
	Color *string `json:"color,omitempty"`
}

// BoundaryInfoDTO - информация о границе
type BoundaryInfoDTO struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	TranslateNames map[string]string `json:"translate_names,omitempty"`
}

// EnrichedLocationDTO - обогащённые данные локации
type EnrichedLocationDTO struct {
	Country          *BoundaryInfoDTO `json:"country,omitempty"`
	Region           *BoundaryInfoDTO `json:"region,omitempty"`
	Province         *BoundaryInfoDTO `json:"province,omitempty"`
	City             *BoundaryInfoDTO `json:"city,omitempty"`
	District         *BoundaryInfoDTO `json:"district,omitempty"`
	Neighborhood     *BoundaryInfoDTO `json:"neighborhood,omitempty"`
	IsAddressVisible *bool            `json:"is_address_visible,omitempty"`
}

// LocationInput - входные данные одной локации для обогащения
type LocationInput struct {
	Index        int      `json:"index"`                             // индекс для маппинга результатов
	Country      string   `json:"country" validate:"required,min=2"` // страна (обязательно)
	Region       *string  `json:"region,omitempty"`                  // регион
	Province     *string  `json:"province,omitempty"`                // провинция
	City         *string  `json:"city,omitempty"`                    // город
	District     *string  `json:"district,omitempty"`                // район
	Neighborhood *string  `json:"neighborhood,omitempty"`            // квартал
	Street       *string  `json:"street,omitempty"`                  // улица
	HouseNumber  *string  `json:"house_number,omitempty"`            // номер дома
	Latitude     *float64 `json:"latitude,omitempty" validate:"omitempty,min=-90,max=90"`
	Longitude    *float64 `json:"longitude,omitempty" validate:"omitempty,min=-180,max=180"`
	IsVisible    *bool    `json:"is_visible,omitempty"` // флаг видимости адреса
}

// ========== Priority Transport DTOs ==========

// PriorityTransportRequest - запрос на поиск транспорта с приоритетом
// Приоритет: metro/train -> bus/tram (если нет высокоприоритетного в радиусе)
type PriorityTransportRequest struct {
	Lat    float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon    float64 `json:"lon" validate:"required,min=-180,max=180"`
	Radius float64 `json:"radius,omitempty" validate:"omitempty,min=100,max=10000"` // метры, default 1500
	Limit  int     `json:"limit,omitempty" validate:"omitempty,min=1,max=20"`       // default 5
}

// PriorityTransportBatchRequest - batch-запрос на поиск транспорта с приоритетом
type PriorityTransportBatchRequest struct {
	Points []PriorityTransportPoint `json:"points" validate:"required,min=1,max=100,dive"`
	Radius float64                  `json:"radius,omitempty" validate:"omitempty,min=100,max=10000"` // метры для всех точек
	Limit  int                      `json:"limit,omitempty" validate:"omitempty,min=1,max=10"`       // лимит на точку
}

// PriorityTransportPoint - точка для batch-запроса
type PriorityTransportPoint struct {
	Lat float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon float64 `json:"lon" validate:"required,min=-180,max=180"`
}

// PriorityTransportResponse - ответ на запрос транспорта с приоритетом
type PriorityTransportResponse struct {
	Stations []PriorityTransportStation `json:"stations"`
	Meta     PriorityTransportMeta      `json:"meta"`
}

// PriorityTransportBatchResponse - batch-ответ на запрос транспорта с приоритетом
type PriorityTransportBatchResponse struct {
	Results []PriorityTransportPointResult `json:"results"`
	Meta    PriorityTransportBatchMeta     `json:"meta"`
}

// PriorityTransportPointResult - результат для одной точки в batch-запросе
type PriorityTransportPointResult struct {
	PointIndex  int                        `json:"point_index"`
	SearchPoint Point                      `json:"search_point"`
	Stations    []PriorityTransportStation `json:"stations"`
}

// PriorityTransportStation - станция транспорта с приоритетом и полной информацией
type PriorityTransportStation struct {
	StationID       int64                       `json:"station_id"`
	Name            string                      `json:"name"`
	NameEn          *string                     `json:"name_en,omitempty"`
	Type            string                      `json:"type"` // metro, train, tram, bus
	Lat             float64                     `json:"lat"`
	Lon             float64                     `json:"lon"`
	LinearDistance  float64                     `json:"linear_distance"`  // метры
	WalkingDistance float64                     `json:"walking_distance"` // метры (примерно)
	WalkingTime     float64                     `json:"walking_time"`     // минуты
	Lines           []TransportLineInfoEnriched `json:"lines,omitempty"`
}

// PriorityTransportMeta - метаданные ответа
type PriorityTransportMeta struct {
	TotalFound      int     `json:"total_found"`
	SearchPoint     Point   `json:"search_point"`
	RadiusM         float64 `json:"radius_m"`
	HasHighPriority bool    `json:"has_high_priority"` // есть ли metro/train в радиусе
	PriorityType    string  `json:"priority_type"`     // "metro/train" или "bus/tram"
	WalkingSpeedKmH float64 `json:"walking_speed_kmh"` // скорость ходьбы для расчёта
}

// PriorityTransportBatchMeta - метаданные batch-ответа
type PriorityTransportBatchMeta struct {
	TotalPoints     int     `json:"total_points"`
	TotalStations   int     `json:"total_stations"`
	RadiusM         float64 `json:"radius_m"`
	WalkingSpeedKmH float64 `json:"walking_speed_kmh"`
}
