package domain

import "time"

type AdminBoundary struct {
	ID           string                 `json:"id" db:"id"`
	OSMId        int64                  `json:"osm_id" db:"osm_id"`
	Name         string                 `json:"name" db:"name"`
	NameEn       string                 `json:"name_en" db:"name_en"`
	NameEs       string                 `json:"name_es" db:"name_es"`
	NameCa       string                 `json:"name_ca" db:"name_ca"`
	NameRu       string                 `json:"name_ru" db:"name_ru"`
	NameUk       string                 `json:"name_uk" db:"name_uk"`
	NameFr       string                 `json:"name_fr" db:"name_fr"`
	NamePt       string                 `json:"name_pt" db:"name_pt"`
	NameIt       string                 `json:"name_it" db:"name_it"`
	NameDe       string                 `json:"name_de" db:"name_de"`
	Type         string                 `json:"type" db:"type"`
	AdminLevel   int                    `json:"admin_level" db:"admin_level"`
	CenterLat    float64                `json:"center_lat" db:"center_lat"`
	CenterLon    float64                `json:"center_lon" db:"center_lon"`
	Geometry     []byte                 `json:"-" db:"geometry"`
	GeometryJSON map[string]interface{} `json:"geometry,omitempty" db:"-"`
	ParentID     *string                `json:"parent_id,omitempty" db:"parent_id"`
	Population   *int                   `json:"population,omitempty" db:"population"`
	AreaSqKm     *float64               `json:"area_sq_km,omitempty" db:"area_sq_km"`
	Tags         map[string]string      `json:"tags,omitempty" db:"tags"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

type Address struct {
	Country  string  `json:"country"`
	Region   string  `json:"region"`
	Province string  `json:"province"`
	City     string  `json:"city"`
	District *string `json:"district,omitempty"`
}
