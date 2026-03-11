package dto

// NearbyRequest — запрос данных поблизости по категории
type NearbyRequest struct {
	Category string  `json:"category" validate:"required"`
	Lat      float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon      float64 `json:"lon" validate:"required,min=-180,max=180"`
	RadiusKm float64 `json:"radius_km,omitempty" validate:"omitempty,min=0.1,max=10"` // км, default 1
	Limit    int     `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`       // default 20
}

// NearbyPOIResponse — ответ для POI-категорий (schools, medical, groceries, ...)
type NearbyPOIResponse struct {
	Category string      `json:"category"`
	Items    []POISimple `json:"items"`
	Total    int         `json:"total"`
}
