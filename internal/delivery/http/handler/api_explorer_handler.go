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

// GetDefaultMethods - возвращает список методов по умолчанию
func GetDefaultMethods() []APIMethodDef {
	return []APIMethodDef{
		{
			ID:          "priority-single",
			Name:        "Priority Transport",
			Icon:        "🚇",
			Description: "Ближайший транспорт с приоритетом. Metro/Train → Bus/Tram",
			Endpoint:    "/debug/transport/priority",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   false,
			Active:      true,
		},
		{
			ID:          "priority-batch",
			Name:        "Priority Transport Batch",
			Icon:        "🚇",
			Description: "Batch версия - несколько точек одним запросом",
			Endpoint:    "/debug/transport/priority/batch",
			HTTPMethod:  "POST",
			IsBatch:     true,
			ShowTypes:   false,
			Active:      false,
		},
		{
			ID:          "enrichment-single",
			Name:        "Enrichment Transport",
			Icon:        "🚉",
			Description: "Транспорт с фильтром по типам и расширенной информацией",
			Endpoint:    "/debug/enrichment/transport",
			HTTPMethod:  "GET",
			IsBatch:     false,
			ShowTypes:   true,
			Active:      false,
		},
		{
			ID:          "enrichment-batch",
			Name:        "Enrichment Batch",
			Icon:        "🚉",
			Description: "Batch версия enrichment транспорта",
			Endpoint:    "/debug/enrichment/transport/batch",
			HTTPMethod:  "POST",
			IsBatch:     true,
			ShowTypes:   true,
			Active:      false,
		},
	}
}

// RenderExplorer - рендеринг страницы API Explorer
func (h *APIExplorerHandler) RenderExplorer(c *fiber.Ctx) error {
	data := APIExplorerData{
		Title:         "API Explorer",
		DefaultMethod: "priority-single",
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
