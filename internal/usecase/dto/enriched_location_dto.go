package dto

// EnrichLocationBatchRequest - запрос на полное обогащение локаций
type EnrichLocationBatchRequest struct {
	Locations []LocationInput `json:"locations" validate:"required,min=1,max=100"`
}

// EnrichLocationBatchResponse - ответ с обогащёнными локациями
type EnrichLocationBatchResponse struct {
	Results []EnrichedLocationResult `json:"results"`
	Meta    EnrichLocationBatchMeta  `json:"meta"`
}

// EnrichedLocationResult - результат обогащения одной локации
type EnrichedLocationResult struct {
	Index            int                        `json:"index"`
	EnrichedLocation *EnrichedLocationDTO       `json:"enriched_location,omitempty"`
	NearestTransport []PriorityTransportStation `json:"nearest_transport,omitempty"`
	Error            string                     `json:"error,omitempty"`
}

// EnrichLocationBatchMeta - метаданные batch обогащения
type EnrichLocationBatchMeta struct {
	TotalLocations int `json:"total_locations"`
	SuccessCount   int `json:"success_count"`
	ErrorCount     int `json:"error_count"`
	WithTransport  int `json:"with_transport"` // кол-во локаций с транспортом
}

// EnrichSingleLocationRequest - запрос на обогащение одной локации
type EnrichSingleLocationRequest struct {
	Country      string   `json:"country" validate:"required"`
	Region       *string  `json:"region,omitempty"`
	Province     *string  `json:"province,omitempty"`
	City         *string  `json:"city,omitempty"`
	District     *string  `json:"district,omitempty"`
	Neighborhood *string  `json:"neighborhood,omitempty"`
	Latitude     *float64 `json:"latitude,omitempty"`
	Longitude    *float64 `json:"longitude,omitempty"`
	IsVisible    *bool    `json:"is_visible,omitempty"`
}
