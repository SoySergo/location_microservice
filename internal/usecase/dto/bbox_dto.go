package dto

// BBoxPOIRequest — запрос на получение POI в видимой области карты (bbox)
type BBoxPOIRequest struct {
	SwLat         float64  `json:"sw_lat"`
	SwLon         float64  `json:"sw_lon"`
	NeLat         float64  `json:"ne_lat"`
	NeLon         float64  `json:"ne_lon"`
	Categories    []string `json:"categories,omitempty"`
	Subcategories []string `json:"subcategories,omitempty"`
	Limit         int      `json:"limit"`
	Offset        int      `json:"offset"`
}

// BBoxTransportRequest — запрос на получение транспортных станций в видимой области карты (bbox)
type BBoxTransportRequest struct {
	SwLat  float64  `json:"sw_lat"`
	SwLon  float64  `json:"sw_lon"`
	NeLat  float64  `json:"ne_lat"`
	NeLon  float64  `json:"ne_lon"`
	Types  []string `json:"types,omitempty"`
	Limit  int      `json:"limit"`
	Offset int      `json:"offset"`
}
