package dto

import "github.com/location-microservice/internal/domain"

// SearchResponse - ответ на поиск границ
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

// SearchResult - результат поиска границы
type SearchResult struct {
	ID          string       `json:"id"`
	Name        string       `json:"name"`
	Type        string       `json:"type"`
	AdminLevel  int          `json:"admin_level,omitempty"`
	CenterPoint domain.Point `json:"center_point"`
	AreaSqKm    *float64     `json:"area_sq_km,omitempty"`
}

// ReverseGeocodeResponse - ответ на обратное геокодирование
type ReverseGeocodeResponse struct {
	Address domain.Address `json:"address"`
}

// BatchReverseGeocodeResponse - ответ на пакетное обратное геокодирование
type BatchReverseGeocodeResponse struct {
	Addresses []domain.Address `json:"addresses"`
}

// NearestTransportResponse - ответ на поиск ближайших транспортных станций
type NearestTransportResponse struct {
	Stations []TransportStationWithLines `json:"stations"`
}

// TransportStationWithLines - транспортная станция с линиями
type TransportStationWithLines struct {
	ID       string                `json:"id"`
	Name     string                `json:"name"`
	Type     string                `json:"type"`
	Lat      float64               `json:"lat"`
	Lon      float64               `json:"lon"`
	Distance float64               `json:"distance"` // meters
	Lines    []TransportLineSimple `json:"lines"`
}

// TransportLineSimple - упрощенная информация о транспортной линии
type TransportLineSimple struct {
	ID    string  `json:"id"`
	Name  string  `json:"name"`
	Ref   string  `json:"ref"`
	Color *string `json:"color,omitempty"`
}

// RadiusPOIResponse - ответ на поиск POI в радиусе
type RadiusPOIResponse struct {
	POIs  []POISimple `json:"pois"`
	Total int         `json:"total"`
}

// BatchNearestTransportResponse - ответ на пакетный поиск ближайших транспортных станций
type BatchNearestTransportResponse struct {
	Results [][]TransportStationWithLines `json:"results"`
}

// POISimple - упрощенная информация о POI
type POISimple struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Subcategory string  `json:"subcategory"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Distance    float64 `json:"distance,omitempty"` // meters
}
