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
