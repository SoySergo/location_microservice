package dto

// PropertyLocationRequest — запрос агрегированных данных о локации объекта недвижимости
type PropertyLocationRequest struct {
	Lat    float64 `json:"lat" validate:"required,min=-90,max=90"`
	Lon    float64 `json:"lon" validate:"required,min=-180,max=180"`
	Radius int     `json:"radius,omitempty" validate:"omitempty,min=100,max=10000"` // метры, default 1000
}

// PropertyLocationResponse — агрегированный ответ с данными локации объекта
type PropertyLocationResponse struct {
	NearestTransport []PriorityTransportStation `json:"nearest_transport"`
	POISummary       map[string]int             `json:"poi_summary"`
	Environment      EnvironmentSummary         `json:"environment"`
}

// EnvironmentSummary — наличие экологических объектов поблизости
type EnvironmentSummary struct {
	GreenSpacesNearby bool `json:"green_spaces_nearby"`
	WaterNearby       bool `json:"water_nearby"`
	BeachNearby       bool `json:"beach_nearby"`
}
