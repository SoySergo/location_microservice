package dto

// SearchRequest - запрос на поиск границ по тексту
type SearchRequest struct {
	Query       string `json:"query" validate:"required,min=2"`
	Language    string `json:"language" validate:"required,oneof=en es ca ru uk fr pt it de"`
	AdminLevels []int  `json:"admin_levels,omitempty" validate:"omitempty,dive,oneof=2 4 6 8 9"`
	Limit       int    `json:"limit" validate:"omitempty,min=1,max=100"`
}

// ReverseGeocodeRequest - запрос на обратное геокодирование
type ReverseGeocodeRequest struct {
	Lat float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon float64 `json:"lon" validate:"required,min=-180,max=180"`
}

// BatchReverseGeocodeRequest - пакетный запрос на обратное геокодирование
type BatchReverseGeocodeRequest struct {
	Points []Point `json:"points" validate:"required,min=1,max=100,dive"`
}

// Point - координаты точки
type Point struct {
	Lat float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon float64 `json:"lon" validate:"required,min=-180,max=180"`
}

// NearestTransportRequest - запрос на поиск ближайших транспортных станций
type NearestTransportRequest struct {
	Lat         float64  `json:"lat" validate:"required,min=-90,max=90"`
	Lon         float64  `json:"lon" validate:"required,min=-180,max=180"`
	Types       []string `json:"types" validate:"required,min=1,dive,oneof=metro train tram bus"`
	MaxDistance float64  `json:"max_distance" validate:"omitempty,min=100,max=10000"` // meters
}

// RadiusPOIRequest - запрос на поиск POI в радиусе
type RadiusPOIRequest struct {
	Lat        float64  `json:"lat" validate:"required,min=-90,max=90"`
	Lon        float64  `json:"lon" validate:"required,min=-180,max=180"`
	RadiusKm   float64  `json:"radius_km" validate:"required,min=0.1,max=100"`
	Categories []string `json:"categories,omitempty"`
	Limit      int      `json:"limit" validate:"omitempty,min=1,max=500"`
}

// BatchNearestTransportRequest - пакетный запрос на поиск ближайших транспортных станций
type BatchNearestTransportRequest struct {
	Points      []Point  `json:"points" validate:"required,min=1,max=100,dive"`
	Types       []string `json:"types" validate:"required,min=1,dive,oneof=metro train tram bus"`
	MaxDistance float64  `json:"max_distance" validate:"omitempty,min=100,max=10000"` // meters
}

// TransportLinesRequest - запрос на получение данных нескольких транспортных линий
type TransportLinesRequest struct {
	IDs []string `json:"ids" validate:"required,min=1,max=50"`
}
