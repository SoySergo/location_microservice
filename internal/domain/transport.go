package domain

import "time"

type TransportStation struct {
	ID         int64             `json:"id" db:"id"`
	OSMId      int64             `json:"osm_id" db:"osm_id"`
	Name       string            `json:"name" db:"name"`
	NameEn     string            `json:"name_en" db:"name_en"`
	Type       string            `json:"type" db:"type"`
	Lat        float64           `json:"lat" db:"lat"`
	Lon        float64           `json:"lon" db:"lon"`
	Distance   *float64          `json:"distance,omitempty" db:"distance"` // Дистанция в метрах от точки запроса
	Geometry   []byte            `json:"-" db:"geometry"`
	LineIDs    []int64           `json:"line_ids" db:"line_ids"`
	Operator   *string           `json:"operator,omitempty" db:"operator"`
	Network    *string           `json:"network,omitempty" db:"network"`
	Wheelchair *bool             `json:"wheelchair,omitempty" db:"wheelchair"`
	Tags       map[string]string `json:"tags,omitempty" db:"tags"`
	CreatedAt  time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at" db:"updated_at"`
}

type TransportLine struct {
	ID          int64             `json:"id" db:"id"`
	OSMId       int64             `json:"osm_id" db:"osm_id"`
	Name        string            `json:"name" db:"name"`
	Ref         string            `json:"ref" db:"ref"`
	Type        string            `json:"type" db:"type"`
	Color       *string           `json:"color,omitempty" db:"color"`
	TextColor   *string           `json:"text_color,omitempty" db:"text_color"`
	Operator    *string           `json:"operator,omitempty" db:"operator"`
	Network     *string           `json:"network,omitempty" db:"network"`
	FromStation *string           `json:"from_station,omitempty" db:"from_station"`
	ToStation   *string           `json:"to_station,omitempty" db:"to_station"`
	Geometry    []byte            `json:"-" db:"geometry"`
	StationIDs  []int64           `json:"station_ids" db:"station_ids"`
	Tags        map[string]string `json:"tags,omitempty" db:"tags"`
	CreatedAt   time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at" db:"updated_at"`
}

// TransportStationWithLines - станция транспорта с информацией о линиях
// Используется для batch-запросов обогащения
type TransportStationWithLines struct {
	StationID int64               `json:"station_id" db:"station_id"`
	Name      string              `json:"name" db:"name"`
	Type      string              `json:"type" db:"type"`
	Lat       float64             `json:"lat" db:"lat"`
	Lon       float64             `json:"lon" db:"lon"`
	Distance  float64             `json:"distance" db:"distance"`   // расстояние от точки запроса в метрах
	PointIdx  int                 `json:"point_idx" db:"point_idx"` // индекс точки из batch запроса
	Lines     []TransportLineInfo `json:"lines,omitempty"`          // TransportLineInfo определён в stream.go
}

// BatchTransportRequest - запрос на batch-получение станций
type BatchTransportRequest struct {
	Points      []TransportSearchPoint `json:"points"`
	MaxDistance float64                `json:"max_distance"` // метры
}

// TransportSearchPoint - точка для поиска транспорта
type TransportSearchPoint struct {
	Lat   float64  `json:"lat"`
	Lon   float64  `json:"lon"`
	Types []string `json:"types"`
	Limit int      `json:"limit"`
}
