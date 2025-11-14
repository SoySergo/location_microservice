package dto

import (
	"strconv"

	"github.com/location-microservice/internal/domain"
)

// SearchResponse - ответ на поиск границ
type SearchResponse struct {
	Results []SearchResult `json:"results"`
	Total   int            `json:"total"`
}

// SearchResult - результат поиска границы
type SearchResult struct {
	ID         string   `json:"id"` // Converted to string for frontend
	Name       string   `json:"name"`
	Type       string   `json:"type"`
	AdminLevel int      `json:"admin_level,omitempty"`
	CenterLat  float64  `json:"center_lat"`
	CenterLon  float64  `json:"center_lon"`
	AreaSqKm   *float64 `json:"area_sq_km,omitempty"`
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
	ID       string                `json:"id"` // Converted to string for frontend
	Name     string                `json:"name"`
	Type     string                `json:"type"`
	Lat      float64               `json:"lat"`
	Lon      float64               `json:"lon"`
	Distance float64               `json:"distance"` // meters
	Lines    []TransportLineSimple `json:"lines"`
}

// TransportLineSimple - упрощенная информация о транспортной линии
type TransportLineSimple struct {
	ID    string  `json:"id"` // Converted to string for frontend
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
	ID          string  `json:"id"` // Converted to string for frontend
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Subcategory string  `json:"subcategory"`
	Lat         float64 `json:"lat"`
	Lon         float64 `json:"lon"`
	Distance    float64 `json:"distance,omitempty"` // meters
}

// Helper functions to convert domain models to DTOs with string IDs

// ConvertSearchResult converts domain boundary to SearchResult DTO
func ConvertSearchResult(b *domain.AdminBoundary) SearchResult {
	return SearchResult{
		ID:         strconv.FormatInt(b.ID, 10),
		Name:       b.Name,
		Type:       b.Type,
		AdminLevel: b.AdminLevel,
		CenterLat:  b.CenterLat,
		CenterLon:  b.CenterLon,
		AreaSqKm:   b.AreaSqKm,
	}
}

// ConvertTransportStation converts domain station to DTO with string IDs
func ConvertTransportStation(station *domain.TransportStation, lines []*domain.TransportLine, distance float64) TransportStationWithLines {
	linesDTOs := make([]TransportLineSimple, 0, len(lines))
	for _, line := range lines {
		linesDTOs = append(linesDTOs, TransportLineSimple{
			ID:    strconv.FormatInt(line.ID, 10),
			Name:  line.Name,
			Ref:   line.Ref,
			Color: line.Color,
		})
	}

	return TransportStationWithLines{
		ID:       strconv.FormatInt(station.ID, 10),
		Name:     station.Name,
		Type:     station.Type,
		Lat:      station.Lat,
		Lon:      station.Lon,
		Distance: distance,
		Lines:    linesDTOs,
	}
}

// ConvertPOI converts domain POI to POISimple DTO
func ConvertPOI(poi *domain.POI, distance float64) POISimple {
	return POISimple{
		ID:          strconv.FormatInt(poi.ID, 10),
		Name:        poi.Name,
		Category:    poi.Category,
		Subcategory: poi.Subcategory,
		Lat:         poi.Lat,
		Lon:         poi.Lon,
		Distance:    distance,
	}
}
