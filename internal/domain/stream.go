package domain

import "github.com/google/uuid"

// Stream names (должны совпадать с backend_estate)
const (
	StreamLocationEnrich = "stream:location:enrich"
	StreamLocationDone   = "stream:location:done"
)

// LocationEnrichEvent - входящее событие на обогащение
type LocationEnrichEvent struct {
	PropertyID   uuid.UUID `json:"property_id"`
	Country      string    `json:"country"`
	Region       *string   `json:"region,omitempty"`
	Province     *string   `json:"province,omitempty"`
	City         *string   `json:"city,omitempty"`
	District     *string   `json:"district,omitempty"`
	Neighborhood *string   `json:"neighborhood,omitempty"`
	Street       *string   `json:"street,omitempty"`
	HouseNumber  *string   `json:"house_number,omitempty"`
	PostalCode   *string   `json:"postal_code,omitempty"`
	Latitude     *float64  `json:"latitude,omitempty"`
	Longitude    *float64  `json:"longitude,omitempty"`
	IsVisible    *bool     `json:"is_visible,omitempty"`
}

// HasStreetAddress проверяет наличие полного адреса (улица + дом)
func (e *LocationEnrichEvent) HasStreetAddress() bool {
	return e.Street != nil && *e.Street != "" &&
		e.HouseNumber != nil && *e.HouseNumber != ""
}

// LocationDoneEvent - результат обогащения
type LocationDoneEvent struct {
	PropertyID       uuid.UUID         `json:"property_id"`
	EnrichedLocation *EnrichedLocation `json:"enriched_location,omitempty"`
	NearestTransport []NearestStation  `json:"nearest_transport,omitempty"`
	Error            string            `json:"error,omitempty"`
}

// EnrichedLocation - обогащённые данные локации
type EnrichedLocation struct {
	Country          *BoundaryInfo `json:"country,omitempty"`
	Region           *BoundaryInfo `json:"region,omitempty"`
	Province         *BoundaryInfo `json:"province,omitempty"`
	City             *BoundaryInfo `json:"city,omitempty"`
	District         *BoundaryInfo `json:"district,omitempty"`
	Neighborhood     *BoundaryInfo `json:"neighborhood,omitempty"`
	IsAddressVisible *bool         `json:"is_address_visible,omitempty"`
}

// BoundaryInfo - информация о границе с переводами
type BoundaryInfo struct {
	ID             int64             `json:"id"`
	Name           string            `json:"name"`
	TranslateNames map[string]string `json:"translate_names,omitempty"`
}

// NearestStation - ближайшая станция транспорта
type NearestStation struct {
	StationID       int64               `json:"station_id"`
	Name            string              `json:"name"`
	Type            string              `json:"type"`
	Lat             float64             `json:"lat"`
	Lon             float64             `json:"lon"`
	Distance        float64             `json:"distance"`
	WalkingDuration *float64            `json:"walking_duration,omitempty"`
	WalkingDistance *float64            `json:"walking_distance,omitempty"`
	Lines           []TransportLineInfo `json:"lines,omitempty"`
}

// TransportLineInfo - информация о линии транспорта
type TransportLineInfo struct {
	ID    int64   `json:"id"`
	Name  string  `json:"name"`
	Ref   string  `json:"ref,omitempty"`
	Type  string  `json:"type,omitempty"`
	Color *string `json:"color,omitempty"`
}

// NearestTransportWithLines - ближайшая станция транспорта с информацией о линиях
// Используется для метода GetNearestTransportByPriority
type NearestTransportWithLines struct {
	StationID int64               `json:"station_id"`
	Name      string              `json:"name"`
	NameEn    *string             `json:"name_en,omitempty"`
	Type      string              `json:"type"` // metro, train, tram, bus, ferry
	Lat       float64             `json:"lat"`
	Lon       float64             `json:"lon"`
	Distance  float64             `json:"distance"` // метры
	Lines     []TransportLineInfo `json:"lines,omitempty"`
}

// BatchTransportResult - результат batch-запроса транспорта для одной точки
type BatchTransportResult struct {
	PointIndex  int                         `json:"point_index"`
	SearchPoint Coordinate                  `json:"search_point"`
	Stations    []NearestTransportWithLines `json:"stations"`
}

// LocationDoneEventExtended - расширенный результат обогащения с инфраструктурой
type LocationDoneEventExtended struct {
	PropertyID       uuid.UUID             `json:"property_id"`
	EnrichedLocation *EnrichedLocation     `json:"enriched_location,omitempty"`
	NearestTransport []NearestStation      `json:"nearest_transport,omitempty"`
	Infrastructure   *InfrastructureResult `json:"infrastructure,omitempty"`
	Error            string                `json:"error,omitempty"`
}

// StreamMessage - сообщение из Redis Stream
type StreamMessage struct {
	ID   string
	Data string
}
