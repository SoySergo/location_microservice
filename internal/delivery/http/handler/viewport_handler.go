package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/domain/repository"
	"go.uber.org/zap"
)

// ViewportHandler — обработчик для получения данных по видимой области карты (bbox).
// Используется в debug explorer для сайдбара с детальной информацией.
type ViewportHandler struct {
	transportRepo repository.TransportRepository
	poiRepo       repository.POIRepository
	logger        *zap.Logger
}

// NewViewportHandler создаёт новый ViewportHandler
func NewViewportHandler(transportRepo repository.TransportRepository, poiRepo repository.POIRepository, logger *zap.Logger) *ViewportHandler {
	return &ViewportHandler{
		transportRepo: transportRepo,
		poiRepo:       poiRepo,
		logger:        logger,
	}
}

// GetTransportInViewport — получение станций транспорта в visible bbox с пагинацией.
// GET /api/v1/viewport/transport?sw_lat=...&sw_lon=...&ne_lat=...&ne_lon=...&types=metro,bus&limit=30&offset=0
func (h *ViewportHandler) GetTransportInViewport(c *fiber.Ctx) error {
	swLat, err := strconv.ParseFloat(c.Query("sw_lat"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sw_lat"})
	}
	swLon, err := strconv.ParseFloat(c.Query("sw_lon"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sw_lon"})
	}
	neLat, err := strconv.ParseFloat(c.Query("ne_lat"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid ne_lat"})
	}
	neLon, err := strconv.ParseFloat(c.Query("ne_lon"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid ne_lon"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "30"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	var types []string
	if t := c.Query("types", ""); t != "" {
		types = strings.Split(t, ",")
		for i := range types {
			types[i] = strings.TrimSpace(types[i])
		}
	}

	stations, total, err := h.transportRepo.GetStationsInBBox(c.Context(), swLat, swLon, neLat, neLon, types, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get transport in viewport", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}

	// Формируем ответ
	items := make([]fiber.Map, 0, len(stations))
	for _, s := range stations {
		item := fiber.Map{
			"id":   s.StationID,
			"name": s.Name,
			"type": s.Type,
			"lat":  s.Lat,
			"lon":  s.Lon,
		}

		if len(s.Lines) > 0 {
			lines := make([]fiber.Map, 0, len(s.Lines))
			for _, l := range s.Lines {
				lineMap := fiber.Map{
					"id":   l.ID,
					"name": l.Name,
					"ref":  l.Ref,
					"type": l.Type,
				}
				if l.Color != nil {
					lineMap["color"] = *l.Color
				}
				lines = append(lines, lineMap)
			}
			item["lines"] = lines
		}

		items = append(items, item)
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"stations": items,
			"total":    total,
			"limit":    limit,
			"offset":   offset,
		},
	})
}

// GetPOIInViewport — получение POI в visible bbox с пагинацией.
// GET /api/v1/viewport/poi?sw_lat=...&sw_lon=...&ne_lat=...&ne_lon=...&categories=healthcare,shopping&subcategories=pharmacy&limit=30&offset=0
func (h *ViewportHandler) GetPOIInViewport(c *fiber.Ctx) error {
	swLat, err := strconv.ParseFloat(c.Query("sw_lat"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sw_lat"})
	}
	swLon, err := strconv.ParseFloat(c.Query("sw_lon"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sw_lon"})
	}
	neLat, err := strconv.ParseFloat(c.Query("ne_lat"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid ne_lat"})
	}
	neLon, err := strconv.ParseFloat(c.Query("ne_lon"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid ne_lon"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "30"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	var categories []string
	if cats := c.Query("categories", ""); cats != "" {
		categories = strings.Split(cats, ",")
		for i := range categories {
			categories[i] = strings.TrimSpace(categories[i])
		}
	}

	var subcategories []string
	if subs := c.Query("subcategories", ""); subs != "" {
		subcategories = strings.Split(subs, ",")
		for i := range subcategories {
			subcategories[i] = strings.TrimSpace(subcategories[i])
		}
	}

	pois, total, err := h.poiRepo.GetPOIInBBox(c.Context(), swLat, swLon, neLat, neLon, categories, subcategories, limit, offset)
	if err != nil {
		h.logger.Error("Failed to get POI in viewport", zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "internal error"})
	}

	items := make([]fiber.Map, 0, len(pois))
	for _, p := range pois {
		item := fiber.Map{
			"id":          p.OSMId,
			"name":        p.Name,
			"category":    p.Category,
			"subcategory": p.Subcategory,
			"lat":         p.Lat,
			"lon":         p.Lon,
		}
		if p.NameEn != nil {
			item["name_en"] = *p.NameEn
		}
		if p.Address != nil {
			item["address"] = *p.Address
		}
		if p.Phone != nil {
			item["phone"] = *p.Phone
		}
		if p.Website != nil {
			item["website"] = *p.Website
		}
		if p.OpeningHours != nil {
			item["opening_hours"] = *p.OpeningHours
		}
		if p.Wheelchair != nil {
			item["wheelchair"] = *p.Wheelchair
		}
		if p.Brand != nil {
			item["brand"] = *p.Brand
		}
		if p.Operator != nil {
			item["operator"] = *p.Operator
		}
		if p.Cuisine != nil {
			item["cuisine"] = *p.Cuisine
		}
		if p.Stars != nil {
			item["stars"] = *p.Stars
		}
		if p.Description != nil {
			item["description"] = *p.Description
		}
		items = append(items, item)
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"pois":   items,
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	})
}
