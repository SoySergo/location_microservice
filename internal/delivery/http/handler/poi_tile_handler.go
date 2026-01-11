package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/usecase"
	"go.uber.org/zap"
)

// POITileHandler - обработчик для POI тайлов
type POITileHandler struct {
	poiTileUC *usecase.POITileUseCase
	logger    *zap.Logger
}

// NewPOITileHandler создает новый POITileHandler
func NewPOITileHandler(poiTileUC *usecase.POITileUseCase, logger *zap.Logger) *POITileHandler {
	return &POITileHandler{
		poiTileUC: poiTileUC,
		logger:    logger,
	}
}

// GetPOITile godoc
// @Summary Получение векторного тайла с POI
// @Description Возвращает векторный тайл (Mapbox Vector Tile) с точками интереса. Поддерживает фильтрацию по категориям и подкатегориям через query параметры.
// @Tags POI Tiles
// @Accept json
// @Produce application/x-protobuf
// @Param z path int true "Zoom level (0-22)"
// @Param x path int true "Tile X coordinate"
// @Param y path int true "Tile Y coordinate"
// @Param categories query string false "Категории через запятую (healthcare,shopping,education)"
// @Param subcategories query string false "Подкатегории через запятую (pharmacy,hospital,school)"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tiles/poi/{z}/{x}/{y}.pbf [get]
func (h *POITileHandler) GetPOITile(c *fiber.Ctx) error {
	// Парсинг параметров тайла
	z, err := strconv.Atoi(c.Params("z"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid zoom parameter"})
	}

	x, err := strconv.Atoi(c.Params("x"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid x parameter"})
	}

	y, err := strconv.Atoi(c.Params("y"))
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid y parameter"})
	}

	// Парсинг query параметров
	categoriesParam := c.Query("categories", "")
	var categories []string
	if categoriesParam != "" {
		categories = strings.Split(categoriesParam, ",")
		// Trim spaces
		for i, cat := range categories {
			categories[i] = strings.TrimSpace(cat)
		}
	}

	subcategoriesParam := c.Query("subcategories", "")
	var subcategories []string
	if subcategoriesParam != "" {
		subcategories = strings.Split(subcategoriesParam, ",")
		// Trim spaces
		for i, subcat := range subcategories {
			subcategories[i] = strings.TrimSpace(subcat)
		}
	}

	// Получение тайла
	tile, err := h.poiTileUC.GetPOITile(c.Context(), z, x, y, categories, subcategories)
	if err != nil {
		h.logger.Error("Failed to get POI tile",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Strings("categories", categories),
			zap.Strings("subcategories", subcategories),
			zap.Error(err))
		return c.Status(500).JSON(fiber.Map{"error": "Failed to generate tile"})
	}

	// Устанавливаем заголовки
	c.Set("Content-Type", "application/x-protobuf")
	c.Set("Content-Encoding", "gzip")
	c.Set("Cache-Control", "public, max-age=3600")

	return c.Send(tile)
}
