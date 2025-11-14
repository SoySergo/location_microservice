package domain

import "time"

// GreenSpace представляет зеленую зону
type GreenSpace struct {
	ID        int64     `json:"id" db:"id"`
	OSMId     int64     `json:"osm_id" db:"osm_id"`
	Type      string    `json:"type" db:"type"`
	Name      *string   `json:"name,omitempty" db:"name"`
	NameEn    *string   `json:"name_en,omitempty" db:"name_en"`
	AreaSqM   float64   `json:"area_sq_m" db:"area_sq_m"`
	Geometry  []byte    `json:"-" db:"geometry"`
	CenterLat float64   `json:"center_lat" db:"center_lat"`
	CenterLon float64   `json:"center_lon" db:"center_lon"`
	Access    *string   `json:"access,omitempty" db:"access"`
	Tags      *JSONBMap `json:"tags,omitempty" db:"tags"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// WaterBody представляет водный объект
type WaterBody struct {
	ID        int64     `json:"id" db:"id"`
	OSMId     int64     `json:"osm_id" db:"osm_id"`
	Type      string    `json:"type" db:"type"`
	Name      *string   `json:"name,omitempty" db:"name"`
	NameEn    *string   `json:"name_en,omitempty" db:"name_en"`
	Geometry  []byte    `json:"-" db:"geometry"`
	Length    *float64  `json:"length,omitempty" db:"length"`
	AreaSqM   *float64  `json:"area_sq_m,omitempty" db:"area_sq_m"`
	Tags      *JSONBMap `json:"tags,omitempty" db:"tags"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Beach представляет пляж
type Beach struct {
	ID        int64     `json:"id" db:"id"`
	OSMId     int64     `json:"osm_id" db:"osm_id"`
	Name      *string   `json:"name,omitempty" db:"name"`
	NameEn    *string   `json:"name_en,omitempty" db:"name_en"`
	Surface   string    `json:"surface" db:"surface"`
	Lat       float64   `json:"lat" db:"lat"`
	Lon       float64   `json:"lon" db:"lon"`
	Geometry  []byte    `json:"-" db:"geometry"`
	Length    *float64  `json:"length,omitempty" db:"length"`
	BlueFlag  *bool     `json:"blue_flag,omitempty" db:"blue_flag"`
	Tags      *JSONBMap `json:"tags,omitempty" db:"tags"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// NoiseSource представляет источник шума
type NoiseSource struct {
	ID        int64     `json:"id" db:"id"`
	OSMId     int64     `json:"osm_id" db:"osm_id"`
	Type      string    `json:"type" db:"type"`
	Name      *string   `json:"name,omitempty" db:"name"`
	Lat       float64   `json:"lat" db:"lat"`
	Lon       float64   `json:"lon" db:"lon"`
	Geometry  []byte    `json:"-" db:"geometry"`
	Intensity *string   `json:"intensity,omitempty" db:"intensity"`
	Tags      *JSONBMap `json:"tags,omitempty" db:"tags"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// TouristZone представляет туристическую зону
type TouristZone struct {
	ID              int64     `json:"id" db:"id"`
	OSMId           int64     `json:"osm_id" db:"osm_id"`
	Type            string    `json:"type" db:"type"`
	Name            string    `json:"name" db:"name"`
	NameEn          string    `json:"name_en" db:"name_en"`
	NameEs          string    `json:"name_es" db:"name_es"`
	NameCa          string    `json:"name_ca" db:"name_ca"`
	NameRu          string    `json:"name_ru" db:"name_ru"`
	NameUk          string    `json:"name_uk" db:"name_uk"`
	NameFr          string    `json:"name_fr" db:"name_fr"`
	NamePt          string    `json:"name_pt" db:"name_pt"`
	NameIt          string    `json:"name_it" db:"name_it"`
	NameDe          string    `json:"name_de" db:"name_de"`
	Lat             float64   `json:"lat" db:"lat"`
	Lon             float64   `json:"lon" db:"lon"`
	Geometry        []byte    `json:"-" db:"geometry"`
	VisitorsPerYear *int      `json:"visitors_per_year,omitempty" db:"visitors_per_year"`
	Fee             *bool     `json:"fee,omitempty" db:"fee"`
	OpeningHours    *string   `json:"opening_hours,omitempty" db:"opening_hours"`
	Website         *string   `json:"website,omitempty" db:"website"`
	Tags            *JSONBMap `json:"tags,omitempty" db:"tags"`
	SearchVector    string    `json:"-" db:"search_vector"`
	CreatedAt       time.Time `json:"created_at" db:"created_at"`
	UpdatedAt       time.Time `json:"updated_at" db:"updated_at"`
}
