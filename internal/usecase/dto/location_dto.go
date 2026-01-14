package dto

// DetectLocationBatchRequest - запрос на детекцию локаций
type DetectLocationBatchRequest struct {
	Locations []LocationInput `json:"locations" validate:"required,min=1,max=100,dive"`
}

// DetectLocationBatchResponse - ответ детекции локаций
type DetectLocationBatchResponse struct {
	Results []LocationDetectionResult `json:"results"`
	Meta    LocationBatchMeta         `json:"meta"`
}

// LocationDetectionResult - результат детекции одной локации
type LocationDetectionResult struct {
	Index            int                  `json:"index"`
	EnrichedLocation *EnrichedLocationDTO `json:"enriched_location,omitempty"`
	Error            string               `json:"error,omitempty"`
}
