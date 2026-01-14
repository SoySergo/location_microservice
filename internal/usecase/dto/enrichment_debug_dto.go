package dto

// EnrichmentDebugTransportRequest - запрос на получение транспорта для дебага
type EnrichmentDebugTransportRequest struct {
	Lat         float64  `json:"lat" validate:"required,min=-90,max=90"`
	Lon         float64  `json:"lon" validate:"required,min=-180,max=180"`
	Types       []string `json:"types,omitempty" validate:"omitempty,dive,oneof=metro train tram bus ferry"`
	MaxDistance float64  `json:"max_distance,omitempty" validate:"omitempty,min=100,max=10000"` // метры
	Limit       int      `json:"limit,omitempty" validate:"omitempty,min=1,max=50"`
}

// EnrichmentDebugTransportBatchRequest - batch-запрос на получение транспорта для нескольких точек
type EnrichmentDebugTransportBatchRequest struct {
	Points      []TransportSearchPointDTO `json:"points" validate:"required,min=1,max=100,dive"`
	MaxDistance float64                   `json:"max_distance,omitempty" validate:"omitempty,min=100,max=10000"` // метры
}

// TransportSearchPointDTO - точка поиска транспорта
type TransportSearchPointDTO struct {
	Lat   float64  `json:"lat" validate:"required,min=-90,max=90"`
	Lon   float64  `json:"lon" validate:"required,min=-180,max=180"`
	Types []string `json:"types,omitempty" validate:"omitempty,dive,oneof=metro train tram bus ferry"`
	Limit int      `json:"limit,omitempty" validate:"omitempty,min=1,max=20"`
}

// EnrichmentDebugTransportResponse - ответ с транспортом для дебага
type EnrichmentDebugTransportResponse struct {
	Stations []EnrichedTransportStation `json:"transport"`
	Meta     EnrichmentDebugMeta        `json:"meta"`
}

// EnrichmentDebugTransportBatchResponse - batch-ответ с транспортом для нескольких точек
type EnrichmentDebugTransportBatchResponse struct {
	Results []PointTransportResult   `json:"results"`
	Meta    EnrichmentDebugBatchMeta `json:"meta"`
}

// PointTransportResult - результат поиска транспорта для одной точки
type PointTransportResult struct {
	PointIndex  int                        `json:"point_index"`
	SearchPoint Point                      `json:"search_point"`
	Stations    []EnrichedTransportStation `json:"stations"`
}

// EnrichedTransportStation - станция транспорта с полной информацией
type EnrichedTransportStation struct {
	StationID       int64                       `json:"station_id"`
	Name            string                      `json:"name"`
	Type            string                      `json:"type"`
	Lat             float64                     `json:"lat"`
	Lon             float64                     `json:"lon"`
	LinearDistance  float64                     `json:"linear_distance"`  // метры (прямая линия)
	WalkingDistance float64                     `json:"walking_distance"` // метры (примерное пешком)
	WalkingTime     float64                     `json:"walking_time"`     // минуты
	Lines           []TransportLineInfoEnriched `json:"lines,omitempty"`
}

// TransportLineInfoEnriched - информация о линии транспорта (расширенная)
type TransportLineInfoEnriched struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Ref   string  `json:"ref,omitempty"`
	Type  string  `json:"type,omitempty"`
	Color *string `json:"color,omitempty"`
}

// EnrichmentDebugMeta - метаданные ответа
type EnrichmentDebugMeta struct {
	TotalFound  int      `json:"total_found"`
	SearchPoint Point    `json:"search_point"`
	RadiusM     float64  `json:"radius_m"`
	Types       []string `json:"types"`
}

// EnrichmentDebugBatchMeta - метаданные batch-ответа
type EnrichmentDebugBatchMeta struct {
	TotalPoints   int     `json:"total_points"`
	TotalStations int     `json:"total_stations"`
	RadiusM       float64 `json:"radius_m"`
}

// EnrichmentDebugLocationRequest - запрос на обогащение локации для дебага
type EnrichmentDebugLocationRequest struct {
	Country      string   `json:"country" validate:"required,min=2"`
	Region       *string  `json:"region,omitempty"`
	Province     *string  `json:"province,omitempty"`
	City         *string  `json:"city,omitempty"`
	District     *string  `json:"district,omitempty"`
	Neighborhood *string  `json:"neighborhood,omitempty"`
	Street       *string  `json:"street,omitempty"`
	HouseNumber  *string  `json:"house_number,omitempty"`
	PostalCode   *string  `json:"postal_code,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty" validate:"omitempty,min=-90,max=90"`
	Longitude    *float64 `json:"longitude,omitempty" validate:"omitempty,min=-180,max=180"`
}

// EnrichmentDebugLocationResponse - ответ с обогащённой локацией
type EnrichmentDebugLocationResponse struct {
	EnrichedLocation *EnrichedLocationDTO `json:"enriched_location,omitempty"`
	NearestTransport []NearestStationDTO  `json:"nearest_transport,omitempty"`
	Error            string               `json:"error,omitempty"`
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

// BoundaryInfoDTO - информация о границе
type BoundaryInfoDTO struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	TranslateNames map[string]string `json:"translate_names,omitempty"`
}

// NearestStationDTO - ближайшая станция транспорта
type NearestStationDTO struct {
	StationID int64                       `json:"station_id"`
	Name      string                      `json:"name"`
	Type      string                      `json:"type"`
	Lat       float64                     `json:"lat"`
	Lon       float64                     `json:"lon"`
	Distance  float64                     `json:"distance"` // метры
	Lines     []TransportLineInfoEnriched `json:"lines,omitempty"`
}

// ========== Batch Location Enrichment DTOs ==========

// EnrichmentDebugLocationBatchRequest - батч-запрос на обогащение локаций
type EnrichmentDebugLocationBatchRequest struct {
	Locations []LocationInput `json:"locations" validate:"required,min=1,max=100,dive"`
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

// EnrichmentDebugLocationBatchResponse - батч-ответ с обогащёнными локациями
type EnrichmentDebugLocationBatchResponse struct {
	Results []LocationEnrichmentResult `json:"results"`
	Meta    LocationBatchMeta          `json:"meta"`
}

// LocationEnrichmentResult - результат обогащения одной локации
type LocationEnrichmentResult struct {
	Index            int                  `json:"index"`
	EnrichedLocation *EnrichedLocationDTO `json:"enriched_location,omitempty"`
	Error            string               `json:"error,omitempty"`
}

// LocationBatchMeta - метаданные батч-ответа
type LocationBatchMeta struct {
	TotalLocations   int `json:"total_locations"`
	SuccessCount     int `json:"success_count"`
	ErrorCount       int `json:"error_count"`
	VisibleCount     int `json:"visible_count"`      // обработано по координатам
	NameResolveCount int `json:"name_resolve_count"` // обработано по названиям
	DBQueriesCount   int `json:"db_queries_count"`   // количество запросов в БД
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
