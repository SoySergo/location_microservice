package domain

import "time"

type Point struct {
	Lat float64 `json:"lat" db:"lat"`
	Lon float64 `json:"lon" db:"lon"`
}

type BoundingBox struct {
	MinLat float64 `json:"min_lat" db:"min_lat"`
	MinLon float64 `json:"min_lon" db:"min_lon"`
	MaxLat float64 `json:"max_lat" db:"max_lat"`
	MaxLon float64 `json:"max_lon" db:"max_lon"`
}

// Statistics представляет общую статистику по данным из OSM
type Statistics struct {
	Boundaries  BoundaryStats    `json:"boundaries"`
	Transport   TransportStats   `json:"transport"`
	POIs        POIStats         `json:"pois"`
	Environment EnvironmentStats `json:"environment"`
	Coverage    CoverageStats    `json:"coverage"`
	LastUpdated time.Time        `json:"last_updated"`
	DataVersion string           `json:"data_version"`
}

// BoundaryStats статистика по границам
type BoundaryStats struct {
	TotalBoundaries int         `json:"total_boundaries"`
	ByAdminLevel    map[int]int `json:"by_admin_level"`
	Countries       int         `json:"countries"`
	Regions         int         `json:"regions"`
	Cities          int         `json:"cities"`
}

// TransportStats статистика по транспорту
type TransportStats struct {
	TotalStations int            `json:"total_stations"`
	TotalLines    int            `json:"total_lines"`
	ByType        map[string]int `json:"by_type"`
}

// POIStats статистика по POI
type POIStats struct {
	TotalPOIs  int            `json:"total_pois"`
	ByCategory map[string]int `json:"by_category"`
}

// EnvironmentStats статистика по окружению
type EnvironmentStats struct {
	GreenSpaces  int `json:"green_spaces"`
	WaterBodies  int `json:"water_bodies"`
	Beaches      int `json:"beaches"`
	NoiseSources int `json:"noise_sources"`
	TouristZones int `json:"tourist_zones"`
}

// CoverageStats статистика покрытия территории
type CoverageStats struct {
	BBoxMinLat float64 `json:"bbox_min_lat"`
	BBoxMaxLat float64 `json:"bbox_max_lat"`
	BBoxMinLon float64 `json:"bbox_min_lon"`
	BBoxMaxLon float64 `json:"bbox_max_lon"`
	CenterLat  float64 `json:"center_lat"`
	CenterLon  float64 `json:"center_lon"`
	AreaSqKm   float64 `json:"area_sq_km"`
}
