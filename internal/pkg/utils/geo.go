package utils

import "math"

const earthRadiusKm = 6371.0

// HaversineDistance вычисляет расстояние между двумя точками в километрах
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180.0
	dLon := (lon2 - lon1) * math.Pi / 180.0

	lat1Rad := lat1 * math.Pi / 180.0
	lat2Rad := lat2 * math.Pi / 180.0

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Sin(dLon/2)*math.Sin(dLon/2)*math.Cos(lat1Rad)*math.Cos(lat2Rad)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusKm * c
}

// ValidateCoordinates проверяет валидность координат
func ValidateCoordinates(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

// ValidateRadius проверяет валидность радиуса (0.1 - 100 км)
func ValidateRadius(radiusKm float64) bool {
	return radiusKm >= 0.1 && radiusKm <= 100
}
