package postgres

import (
	"fmt"
	"strings"
)

// Константы для MVT тайлов
const (
	// MVTExtent - размер тайла в единицах координат
	MVTExtent = 4096
	// MVTBuffer - буфер вокруг тайла для предотвращения обрезки геометрии
	MVTBuffer = 256
	// MVTSimplifyTolerance - базовая толерантность для упрощения геометрии
	MVTSimplifyTolerance = 0.0001
)

// Константы для радиусов и лимитов запросов
const (
	// DefaultQueryLimit - лимит по умолчанию для запросов
	DefaultQueryLimit = 100
	// MaxQueryLimit - максимальный лимит для запросов
	MaxQueryLimit = 1000
	// DefaultRadiusKm - радиус поиска по умолчанию в километрах
	DefaultRadiusKm = 5.0
	// MaxRadiusKm - максимальный радиус поиска в километрах
	MaxRadiusKm = 50.0
)

// Константы для геометрии
const (
	// SRID4326 - WGS84 coordinate system
	SRID4326 = 4326
	// SRID3857 - Web Mercator projection
	SRID3857 = 3857
	// EarthRadiusMeters - радиус Земли в метрах
	EarthRadiusMeters = 6371000
)

// Константы для zoom levels
const (
	MinZoomLevel = 0
	MaxZoomLevel = 18
	// Zoom thresholds для различных типов объектов
	ZoomBoundariesCountries = 4
	ZoomBoundariesRegions   = 7
	ZoomBoundariesProvinces = 10
	ZoomBoundariesCities    = 13
	ZoomWaterBodiesMin      = 8
	ZoomWaterRiversMin      = 10
	ZoomBeachesMin          = 12
	ZoomNoiseSourcesMin     = 8
	ZoomNoiseIndustrialMin  = 11
	ZoomNoiseAllMin         = 13
	ZoomTouristZonesMin     = 11
)

// Константы для line padding (в метрах)
const (
	LinePaddingShort  = 1000  // <5км
	LinePaddingMedium = 2000  // 5-20км
	LinePaddingLong   = 5000  // 20-100км
	LinePaddingXLong  = 10000 // >100км
)

// Константы для длины линий (в метрах)
const (
	LineLengthShort  = 5000
	LineLengthMedium = 20000
	LineLengthLong   = 100000
)

// Константы для лимитов по типам объектов
const (
	LimitStations         = 100
	LimitLines            = 50
	LimitPOIs             = 100
	LimitPOIsRadius       = 200
	LimitPOIsCategory     = 1000
	LimitGreenSpaces      = 50
	LimitWaterBodies      = 50
	LimitBeaches          = 20
	LimitNoiseSources     = 50
	LimitTouristZones     = 50
	LimitBoundariesRadius = 50
)

// Константы для расширения границ при геопоиске
const (
	BoundaryExpansionDegrees = 0.1 // ~11км на экваторе
)

// scanInt64Array парсит PostgreSQL array bigint[] в []int64
// Формат PostgreSQL: {1,2,3} или {}
func scanInt64Array(src interface{}) ([]int64, error) {
	if src == nil {
		return []int64{}, nil
	}

	switch v := src.(type) {
	case []byte:
		// PostgreSQL array format: {1,2,3}
		str := string(v)
		if str == "{}" || str == "" {
			return []int64{}, nil
		}

		// Удаляем фигурные скобки
		str = strings.Trim(str, "{}")
		if str == "" {
			return []int64{}, nil
		}

		// Разделяем по запятой
		parts := strings.Split(str, ",")
		result := make([]int64, 0, len(parts))

		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			var val int64
			_, err := fmt.Sscanf(part, "%d", &val)
			if err != nil {
				return nil, fmt.Errorf("failed to parse int64 from array: %w", err)
			}
			result = append(result, val)
		}
		return result, nil
	default:
		return nil, fmt.Errorf("unsupported type for int64 array: %T", src)
	}
}

// int64ArrayToString конвертирует []int64 в формат PostgreSQL array
// Результат: {1,2,3} или {}
func int64ArrayToString(arr []int64) string {
	if len(arr) == 0 {
		return "{}"
	}

	strs := make([]string, len(arr))
	for i, v := range arr {
		strs[i] = fmt.Sprintf("%d", v)
	}
	return "{" + strings.Join(strs, ",") + "}"
}

// stringArrayToString конвертирует []string в формат PostgreSQL array
// Результат: {"str1","str2"} или {}
// Экранирует кавычки и обратные слеши
func stringArrayToString(arr []string) string {
	if len(arr) == 0 {
		return "{}"
	}

	strs := make([]string, len(arr))
	for i, v := range arr {
		// Экранируем обратные слеши и кавычки
		escaped := strings.ReplaceAll(v, "\\", "\\\\")
		escaped = strings.ReplaceAll(escaped, "\"", "\\\"")
		strs[i] = "\"" + escaped + "\""
	}
	return "{" + strings.Join(strs, ",") + "}"
}

// getLinePadding возвращает padding для линии в зависимости от её длины
func getLinePadding(lengthMeters float64) int {
	switch {
	case lengthMeters < LineLengthShort:
		return LinePaddingShort
	case lengthMeters < LineLengthMedium:
		return LinePaddingMedium
	case lengthMeters < LineLengthLong:
		return LinePaddingLong
	default:
		return LinePaddingXLong
	}
}

// getMVTSimplifyTolerance возвращает tolerance для упрощения геометрии в зависимости от zoom
func getMVTSimplifyTolerance(zoom int) float64 {
	switch {
	case zoom < 10:
		return 0.01
	case zoom < 12:
		return 0.001
	default:
		return MVTSimplifyTolerance
	}
}

// getPOILimitByZoom возвращает лимит POI в зависимости от zoom level
func getPOILimitByZoom(zoom int) int {
	switch {
	case zoom < 10:
		return 50
	case zoom < 13:
		return 200
	case zoom < 15:
		return 500
	default:
		return MaxQueryLimit
	}
}

// getAdminLevelsByZoom возвращает список административных уровней для отображения на заданном zoom
func getAdminLevelsByZoom(zoom int) []int {
	switch {
	case zoom <= ZoomBoundariesCountries:
		return []int{2}
	case zoom <= ZoomBoundariesRegions:
		return []int{2, 4}
	case zoom <= ZoomBoundariesProvinces:
		return []int{2, 4, 6}
	case zoom <= ZoomBoundariesCities:
		return []int{2, 4, 6, 8}
	default:
		return []int{2, 4, 6, 8, 9}
	}
}

// getAdminLevelsString возвращает строку с admin levels для SQL запроса
func getAdminLevelsString(zoom int) string {
	levels := getAdminLevelsByZoom(zoom)
	strs := make([]string, len(levels))
	for i, level := range levels {
		strs[i] = fmt.Sprintf("%d", level)
	}
	return strings.Join(strs, ",")
}
