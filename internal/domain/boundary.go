package domain

import "time"

type AdminBoundary struct {
	ID           int64                  `json:"id" db:"id"`
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
	ParentID     *int64                 `json:"parent_id,omitempty" db:"parent_id"`
	Population   *int                   `json:"population,omitempty" db:"population"`
	AreaSqKm     *float64               `json:"area_sq_km,omitempty" db:"area_sq_km"`
	Tags         map[string]string      `json:"tags,omitempty" db:"tags"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
}

type Address struct {
	Country      string  `json:"country"`                // admin_level 2
	Region       string  `json:"region"`                 // admin_level 4
	Province     string  `json:"province"`               // admin_level 6
	Subprovince  *string `json:"subprovince,omitempty"`  // admin_level 7
	City         string  `json:"city"`                   // admin_level 8
	District     *string `json:"district,omitempty"`     // admin_level 9
	Subdistrict  *string `json:"subdistrict,omitempty"`  // admin_level 10
	Neighborhood *string `json:"neighborhood,omitempty"` // admin_level 11
}

// LatLon представляет координаты точки
type LatLon struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

// BoundarySearchRequest - запрос на поиск границы по тексту для батча
type BoundarySearchRequest struct {
	Index       int    // индекс запроса для маппинга результатов
	Name        string // искомое название
	AdminLevel  int    // уровень административной единицы (2, 4, 6, 8, 9, 10)
	CountryHint string // опциональная подсказка страны для уточнения поиска
}

// BoundarySearchResult - результат поиска границы для батча
type BoundarySearchResult struct {
	Index    int            // индекс запроса
	Boundary *AdminBoundary // найденная граница (nil если не найдена)
	Found    bool           // флаг успешного поиска
}
