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

// LocationBatchMeta - метаданные батч-ответа
type LocationBatchMeta struct {
	TotalLocations   int `json:"total_locations"`
	SuccessCount     int `json:"success_count"`
	ErrorCount       int `json:"error_count"`
	VisibleCount     int `json:"visible_count"`      // обработано по координатам
	NameResolveCount int `json:"name_resolve_count"` // обработано по названиям
	DBQueriesCount   int `json:"db_queries_count"`   // количество запросов в БД
}
