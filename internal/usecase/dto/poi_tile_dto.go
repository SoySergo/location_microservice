package dto

// POITileRequest - запрос на получение POI тайла
type POITileRequest struct {
	Z             int      `json:"z" validate:"required,min=0,max=18"`
	X             int      `json:"x" validate:"required,min=0"`
	Y             int      `json:"y" validate:"required,min=0"`
	Categories    []string `json:"categories,omitempty" validate:"omitempty,dive,oneof=healthcare shopping education leisure food_drink"`
	Subcategories []string `json:"subcategories,omitempty"`
}

// TransportLinesByStationResponse - ответ со списком линий для станции
type TransportLinesByStationResponse struct {
	Lines []TransportLineInfo `json:"lines"`
}

// TransportLineInfo - информация о транспортной линии
type TransportLineInfo struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Ref       string  `json:"ref"`
	Type      string  `json:"type"`
	Color     *string `json:"color,omitempty"`
	TextColor *string `json:"text_color,omitempty"`
	Operator  *string `json:"operator,omitempty"`
	Network   *string `json:"network,omitempty"`
}
