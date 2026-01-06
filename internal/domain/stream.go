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
	CountryID        *int64 `json:"country_id,omitempty"`
	RegionID         *int64 `json:"region_id,omitempty"`
	ProvinceID       *int64 `json:"province_id,omitempty"`
	CityID           *int64 `json:"city_id,omitempty"`
	DistrictID       *int64 `json:"district_id,omitempty"`
	NeighborhoodID   *int64 `json:"neighborhood_id,omitempty"`
	IsAddressVisible *bool  `json:"is_address_visible,omitempty"`
}

// NearestStation - ближайшая станция транспорта
type NearestStation struct {
	StationID int64   `json:"station_id"`
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Distance  float64 `json:"distance"`
	LineIDs   []int64 `json:"line_ids"`
}

// LocationDoneEventExtended - расширенный результат обогащения с инфраструктурой
type LocationDoneEventExtended struct {
	PropertyID       uuid.UUID              `json:"property_id"`
	EnrichedLocation *EnrichedLocation      `json:"enriched_location,omitempty"`
	NearestTransport []NearestStation       `json:"nearest_transport,omitempty"`
	Infrastructure   *InfrastructureResult  `json:"infrastructure,omitempty"`
	Error            string                 `json:"error,omitempty"`
}

// StreamMessage - сообщение из Redis Stream
type StreamMessage struct {
	ID   string
	Data string
}
