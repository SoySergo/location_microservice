package domain

// InfrastructureResult - результат обогащения инфраструктурой
type InfrastructureResult struct {
	Transport        []TransportWithDistance `json:"transport,omitempty"`
	POIs             []POIWithDistance       `json:"pois,omitempty"`
	WalkingDistances map[string]float64      `json:"walking_distances,omitempty"`
}

// TransportWithDistance - транспортная станция с расстояниями
type TransportWithDistance struct {
	StationID       int64    `json:"station_id"`
	Name            string   `json:"name"`
	Type            string   `json:"type"`
	Lat             float64  `json:"lat"`
	Lon             float64  `json:"lon"`
	LineIDs         []int64  `json:"line_ids,omitempty"`
	LinearDistance  float64  `json:"linear_distance"`
	WalkingDistance *float64 `json:"walking_distance,omitempty"`
	WalkingDuration *float64 `json:"walking_duration,omitempty"`
}

// POIWithDistance - точка интереса с расстояниями
type POIWithDistance struct {
	ID              int64    `json:"id"`
	Name            string   `json:"name"`
	Category        string   `json:"category"`
	Subcategory     string   `json:"subcategory"`
	Lat             float64  `json:"lat"`
	Lon             float64  `json:"lon"`
	LinearDistance  float64  `json:"linear_distance"`
	WalkingDistance *float64 `json:"walking_distance,omitempty"`
	WalkingDuration *float64 `json:"walking_duration,omitempty"`
}

// TransportPriority - конфигурация приоритета транспорта
type TransportPriority struct {
	Type  string
	Limit int
}

// POICategoryConfig - конфигурация категории POI
type POICategoryConfig struct {
	Category    string
	Subcategory string
	Limit       int
}

// Coordinate - координата для Mapbox
type Coordinate struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// MatrixResponse - ответ Mapbox Matrix API
type MatrixResponse struct {
	Code         string      `json:"code"`
	Distances    [][]float64 `json:"distances"` // в метрах
	Durations    [][]float64 `json:"durations"` // в секундах
	Destinations []Location  `json:"destinations"`
	Sources      []Location  `json:"sources"`
}

// Location - локация в ответе Mapbox
type Location struct {
	Name     string    `json:"name"`
	Location []float64 `json:"location"` // [lon, lat]
}
