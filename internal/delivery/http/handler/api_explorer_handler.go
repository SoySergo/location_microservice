package handler

import (
	"html/template"
	"path/filepath"

	"github.com/gofiber/fiber/v2"
)

// APIExplorerData - данные для шаблона API Explorer
type APIExplorerData struct {
	Title         string
	DefaultMethod string
	MapStyle      string
	MapCenter     MapCenterCoords
	MapZoom       int
	Methods       []APIMethodDef
}

// MapCenterCoords - координаты центра карты
type MapCenterCoords struct {
	Lat float64
	Lon float64
}

// APIMethodDef - определение API метода для шаблона
type APIMethodDef struct {
	ID          string
	Name        string
	Icon        string
	Description string
	Endpoint    string
	HTTPMethod  string
	IsBatch     bool
	ShowTypes   bool
	Active      bool
}

// APIExplorerHandler - хендлер для рендеринга API Explorer
type APIExplorerHandler struct {
	templates *template.Template
}

// NewAPIExplorerHandler - создание нового хендлера API Explorer
func NewAPIExplorerHandler() (*APIExplorerHandler, error) {
	// Загружаем все шаблоны из директории templates/api-explorer
	tmpl, err := template.ParseGlob(filepath.Join("templates", "api-explorer", "*.html"))
	if err != nil {
		return nil, err
	}

	return &APIExplorerHandler{
		templates: tmpl,
	}, nil
}

// GetDefaultMethods - возвращает ВСЕ методы API сервиса
func GetDefaultMethods() []APIMethodDef {
	return []APIMethodDef{
		// ========== SEARCH ==========
		{
			ID:          "search",
			Name:        "Search Boundaries",
			Icon:        "🔍",
			Description: "Полнотекстовый поиск по административным границам",
			Endpoint:    "/api/v1/search",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      true,
		},
		{
			ID:          "reverse-geocode",
			Name:        "Reverse Geocode",
			Icon:        "📍",
			Description: "Обратное геокодирование - адрес по координатам",
			Endpoint:    "/api/v1/reverse-geocode",
			HTTPMethod:  "POST",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "reverse-geocode-batch",
			Name:        "Reverse Geocode Batch",
			Icon:        "📍",
			Description: "Пакетное обратное геокодирование",
			Endpoint:    "/api/v1/batch/reverse-geocode",
			HTTPMethod:  "POST",
			IsBatch:     true,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "boundary-by-id",
			Name:        "Get Boundary",
			Icon:        "🗺️",
			Description: "Получение границы по ID",
			Endpoint:    "/api/v1/boundaries/{id}",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},

		// ========== LOCATION ENRICHMENT (новые) ==========
		{
			ID:          "enrich-location",
			Name:        "Enrich Location",
			Icon:        "✨",
			Description: "Обогащение одной локации (границы + транспорт)",
			Endpoint:    "/api/v1/locations/enrich",
			HTTPMethod:  "POST",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "enrich-location-batch",
			Name:        "Enrich Location Batch",
			Icon:        "✨",
			Description: "Batch обогащение локаций",
			Endpoint:    "/api/v1/locations/enrich/batch",
			HTTPMethod:  "POST",
			IsBatch:     true,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "detect-location-batch",
			Name:        "Detect Location Batch",
			Icon:        "🎯",
			Description: "Batch детекция локаций (без транспорта)",
			Endpoint:    "/api/v1/locations/detect/batch",
			HTTPMethod:  "POST",
			IsBatch:     true,
			ShowTypes:   false,
			Active:      false,
		},

		// ========== TRANSPORT ==========
		{
			ID:          "transport-nearest",
			Name:        "Nearest Transport",
			Icon:        "🚌",
			Description: "Поиск ближайших станций транспорта",
			Endpoint:    "/api/v1/transport/nearest",
			HTTPMethod:  "POST",
			IsBatch:     false,
			ShowTypes:   true,
			Active:      false,
		},
		{
			ID:          "transport-nearest-batch",
			Name:        "Nearest Transport Batch",
			Icon:        "🚌",
			Description: "Пакетный поиск ближайших станций",
			Endpoint:    "/api/v1/batch/transport/nearest",
			HTTPMethod:  "POST",
			IsBatch:     true,
			ShowTypes:   true,
			Active:      false,
		},
		{
			ID:          "transport-priority",
			Name:        "Priority Transport",
			Icon:        "🚇",
			Description: "Ближайший транспорт с приоритетом (metro/train → bus/tram)",
			Endpoint:    "/api/v1/transport/priority",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "transport-priority-batch",
			Name:        "Priority Transport Batch",
			Icon:        "🚇",
			Description: "Batch приоритетный транспорт",
			Endpoint:    "/api/v1/transport/priority/batch",
			HTTPMethod:  "POST",
			IsBatch:     true,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "station-lines",
			Name:        "Station Lines",
			Icon:        "🚉",
			Description: "Линии проходящие через станцию",
			Endpoint:    "/api/v1/transport/station/{station_id}/lines",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},

		// ========== POI ==========
		{
			ID:          "poi-radius",
			Name:        "POI by Radius",
			Icon:        "📌",
			Description: "Поиск точек интереса в радиусе",
			Endpoint:    "/api/v1/radius/poi",
			HTTPMethod:  "POST",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "poi-categories",
			Name:        "POI Categories",
			Icon:        "📋",
			Description: "Список категорий POI",
			Endpoint:    "/api/v1/poi/categories",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "poi-subcategories",
			Name:        "POI Subcategories",
			Icon:        "📋",
			Description: "Подкатегории для категории",
			Endpoint:    "/api/v1/poi/categories/{id}/subcategories",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},

		// ========== TILES ==========
		{
			ID:          "tile-boundaries",
			Name:        "Boundary Tiles",
			Icon:        "🗺️",
			Description: "Векторные тайлы с границами",
			Endpoint:    "/api/v1/boundaries/tiles/{z}/{x}/{y}.pbf",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "tile-transport",
			Name:        "Transport Tiles",
			Icon:        "🚇",
			Description: "Векторные тайлы с транспортом",
			Endpoint:    "/api/v1/transport/tiles/{z}/{x}/{y}.pbf",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   true,
			Active:      false,
		},
		{
			ID:          "tile-transport-filtered",
			Name:        "Transport Tiles (Filtered)",
			Icon:        "🚇",
			Description: "Тайлы транспорта с фильтрацией по типам",
			Endpoint:    "/api/v1/tiles/transport/{z}/{x}/{y}.pbf",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   true,
			Active:      false,
		},
		{
			ID:          "tile-poi",
			Name:        "POI Tiles",
			Icon:        "📌",
			Description: "Векторные тайлы с POI",
			Endpoint:    "/api/v1/tiles/poi/{z}/{x}/{y}.pbf",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "tile-green-spaces",
			Name:        "Green Spaces Tiles",
			Icon:        "🌳",
			Description: "Тайлы с парками и зелёными зонами",
			Endpoint:    "/api/v1/green-spaces/tiles/{z}/{x}/{y}.pbf",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "tile-water",
			Name:        "Water Tiles",
			Icon:        "💧",
			Description: "Тайлы с водными объектами",
			Endpoint:    "/api/v1/water/tiles/{z}/{x}/{y}.pbf",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "tile-beaches",
			Name:        "Beaches Tiles",
			Icon:        "🏖️",
			Description: "Тайлы с пляжами",
			Endpoint:    "/api/v1/beaches/tiles/{z}/{x}/{y}.pbf",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},

		// ========== STATISTICS ==========
		{
			ID:          "stats",
			Name:        "Statistics",
			Icon:        "📊",
			Description: "Статистика системы",
			Endpoint:    "/api/v1/stats",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},

		// ========== HEALTH ==========
		{
			ID:          "health",
			Name:        "Health Check",
			Icon:        "💚",
			Description: "Проверка состояния сервиса",
			Endpoint:    "/api/v1/health",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      false,
		},
	}
}

// RenderExplorer - рендеринг страницы API Explorer
func (h *APIExplorerHandler) RenderExplorer(c *fiber.Ctx) error {
	data := APIExplorerData{
		Title:         "API Explorer",
		DefaultMethod: "search",
		MapStyle:      "mapbox://styles/serhii11/cmhuvoz2c001o01sfgppw7m5n",
		MapCenter: MapCenterCoords{
			Lat: 41.3851,
			Lon: 2.1734,
		},
		MapZoom: 13,
		Methods: GetDefaultMethods(),
	}

	c.Set("Content-Type", "text/html; charset=utf-8")
	return h.templates.ExecuteTemplate(c.Response().BodyWriter(), "base.html", data)
}
