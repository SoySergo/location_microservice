package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/pkg/validator"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

// TileHandler - обработчик для запросов векторных тайлов
type TileHandler struct {
	tileUC *usecase.TileUseCase
	logger *zap.Logger
}

// NewTileHandler - создание нового TileHandler
func NewTileHandler(tileUC *usecase.TileUseCase, logger *zap.Logger) *TileHandler {
	return &TileHandler{
		tileUC: tileUC,
		logger: logger,
	}
}

// GetBoundaryTile godoc
// @Summary Получение векторного тайла с административными границами
// @Description Возвращает векторный тайл (Mapbox Vector Tile) с административными границами (страны, регионы, провинции, города, районы)
// @Tags Tiles
// @Accept json
// @Produce application/x-protobuf
// @Param z path int true "Zoom level (0-22)"
// @Param x path int true "Tile X coordinate"
// @Param y path int true "Tile Y coordinate"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 500 {string} string "Failed to generate tile"
// @Router /api/v1/boundaries/tiles/{z}/{x}/{y}.pbf [get]
func (h *TileHandler) GetBoundaryTile(c *fiber.Ctx) error {
	z, _ := strconv.Atoi(c.Params("z"))
	x, _ := strconv.Atoi(c.Params("x"))
	y, _ := strconv.Atoi(c.Params("y"))

	h.logger.Info("Boundary tile request",
		zap.Int("z", z),
		zap.Int("x", x),
		zap.Int("y", y))

	tile, err := h.tileUC.GetBoundaryTile(c.Context(), z, x, y)
	if err != nil {
		h.logger.Error("Failed to get boundary tile", zap.Error(err))
		return c.Status(500).SendString("Failed to generate tile")
	}

	if len(tile) == 0 {
		h.logger.Debug("Boundary tile is empty (no data in this tile)",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y))
	} else {
		h.logger.Info("Boundary tile generated",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Int("size", len(tile)))
	}

	c.Set("Content-Type", "application/x-protobuf")
	// c.Set("Content-Encoding", "gzip")
	return c.Send(tile)
}

// GetTransportTile godoc
// @Summary Получение векторного тайла с транспортом
// @Description Возвращает векторный тайл (Mapbox Vector Tile) со всеми транспортными станциями и линиями без фильтрации
// @Tags Tiles
// @Accept json
// @Produce application/x-protobuf
// @Param z path int true "Zoom level (0-22)"
// @Param x path int true "Tile X coordinate"
// @Param y path int true "Tile Y coordinate"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 500 {string} string "Failed to generate tile"
// @Router /api/v1/transport/tiles/{z}/{x}/{y}.pbf [get]
func (h *TileHandler) GetTransportTile(c *fiber.Ctx) error {
	z, _ := strconv.Atoi(c.Params("z"))
	x, _ := strconv.Atoi(c.Params("x"))
	y, _ := strconv.Atoi(c.Params("y"))

	tile, err := h.tileUC.GetTransportTile(c.Context(), z, x, y)
	if err != nil {
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/x-protobuf")
	return c.Send(tile)
}

// GetGreenSpacesTile godoc
// @Summary Получение векторного тайла с зелеными зонами
// @Description Возвращает векторный тайл (Mapbox Vector Tile) с парками, садами и другими зелеными пространствами
// @Tags Tiles
// @Accept json
// @Produce application/x-protobuf
// @Param z path int true "Zoom level (0-22)"
// @Param x path int true "Tile X coordinate"
// @Param y path int true "Tile Y coordinate"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 500 {string} string "Failed to generate tile"
// @Router /api/v1/green-spaces/tiles/{z}/{x}/{y}.pbf [get]
func (h *TileHandler) GetGreenSpacesTile(c *fiber.Ctx) error {
	z, _ := strconv.Atoi(c.Params("z"))
	x, _ := strconv.Atoi(c.Params("x"))
	y, _ := strconv.Atoi(c.Params("y"))

	tile, err := h.tileUC.GetGreenSpacesTile(c.Context(), z, x, y)
	if err != nil {
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/x-protobuf")
	return c.Send(tile)
}

// GetWaterTile godoc
// @Summary Получение векторного тайла с водными объектами
// @Description Возвращает векторный тайл (Mapbox Vector Tile) с реками, озерами, морями и другими водными объектами
// @Tags Tiles
// @Accept json
// @Produce application/vnd.mapbox-vector-tile
// @Param z path int true "Zoom level (0-22)"
// @Param x path int true "Tile X coordinate"
// @Param y path int true "Tile Y coordinate"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 500 {string} string "Failed to generate tile"
// @Router /api/v1/water/tiles/{z}/{x}/{y}.pbf [get]
func (h *TileHandler) GetWaterTile(c *fiber.Ctx) error {
	z, _ := strconv.Atoi(c.Params("z"))
	x, _ := strconv.Atoi(c.Params("x"))
	y, _ := strconv.Atoi(c.Params("y"))

	tile, err := h.tileUC.GetWaterTile(c.Context(), z, x, y)
	if err != nil {
		h.logger.Error("Failed to get water tile", zap.Error(err))
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
	c.Set("Content-Encoding", "gzip")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.Send(tile)
}

// GetBeachesTile godoc
// @Summary Получение векторного тайла с пляжами
// @Description Возвращает векторный тайл (Mapbox Vector Tile) с пляжами. Минимальный уровень зума: 12.
// @Tags Tiles
// @Accept json
// @Produce application/vnd.mapbox-vector-tile
// @Param z path int true "Zoom level (min: 12, max: 22)"
// @Param x path int true "Tile X coordinate"
// @Param y path int true "Tile Y coordinate"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 400 {object} map[string]string "Minimum zoom level is 12"
// @Failure 500 {string} string "Failed to generate tile"
// @Router /api/v1/beaches/tiles/{z}/{x}/{y}.pbf [get]
func (h *TileHandler) GetBeachesTile(c *fiber.Ctx) error {
	z, _ := strconv.Atoi(c.Params("z"))
	x, _ := strconv.Atoi(c.Params("x"))
	y, _ := strconv.Atoi(c.Params("y"))

	// Валидация zoom >= 12
	if z < 12 {
		return c.Status(400).JSON(fiber.Map{"error": "Minimum zoom level is 12 for beaches"})
	}

	tile, err := h.tileUC.GetBeachesTile(c.Context(), z, x, y)
	if err != nil {
		h.logger.Error("Failed to get beaches tile", zap.Error(err))
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
	c.Set("Content-Encoding", "gzip")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.Send(tile)
}

// GetNoiseSourcesTile godoc
// @Summary Получение векторного тайла с источниками шума
// @Description Возвращает векторный тайл (Mapbox Vector Tile) с источниками шума (дороги, железные дороги, аэропорты)
// @Tags Tiles
// @Accept json
// @Produce application/vnd.mapbox-vector-tile
// @Param z path int true "Zoom level (0-22)"
// @Param x path int true "Tile X coordinate"
// @Param y path int true "Tile Y coordinate"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 500 {string} string "Failed to generate tile"
// @Router /api/v1/noise-sources/tiles/{z}/{x}/{y}.pbf [get]
func (h *TileHandler) GetNoiseSourcesTile(c *fiber.Ctx) error {
	z, _ := strconv.Atoi(c.Params("z"))
	x, _ := strconv.Atoi(c.Params("x"))
	y, _ := strconv.Atoi(c.Params("y"))

	tile, err := h.tileUC.GetNoiseSourcesTile(c.Context(), z, x, y)
	if err != nil {
		h.logger.Error("Failed to get noise sources tile", zap.Error(err))
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
	c.Set("Content-Encoding", "gzip")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.Send(tile)
}

// GetTouristZonesTile godoc
// @Summary Получение векторного тайла с туристическими зонами
// @Description Возвращает векторный тайл (Mapbox Vector Tile) с туристическими зонами и достопримечательностями. Минимальный уровень зума: 11.
// @Tags Tiles
// @Accept json
// @Produce application/vnd.mapbox-vector-tile
// @Param z path int true "Zoom level (min: 11, max: 22)"
// @Param x path int true "Tile X coordinate"
// @Param y path int true "Tile Y coordinate"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 400 {object} map[string]string "Minimum zoom level is 11"
// @Failure 500 {string} string "Failed to generate tile"
// @Router /api/v1/tourist-zones/tiles/{z}/{x}/{y}.pbf [get]
func (h *TileHandler) GetTouristZonesTile(c *fiber.Ctx) error {
	z, _ := strconv.Atoi(c.Params("z"))
	x, _ := strconv.Atoi(c.Params("x"))
	y, _ := strconv.Atoi(c.Params("y"))

	// Валидация zoom >= 11
	if z < 11 {
		return c.Status(400).JSON(fiber.Map{"error": "Minimum zoom level is 11 for tourist zones"})
	}

	tile, err := h.tileUC.GetTouristZonesTile(c.Context(), z, x, y)
	if err != nil {
		h.logger.Error("Failed to get tourist zones tile", zap.Error(err))
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
	c.Set("Content-Encoding", "gzip")
	c.Set("Cache-Control", "public, max-age=86400")
	return c.Send(tile)
}

// GetTransportLineTile godoc
// @Summary Получение векторного тайла для одной транспортной линии
// @Description Возвращает векторный тайл (Mapbox Vector Tile) с геометрией и станциями указанной транспортной линии
// @Tags Transport Tiles
// @Accept json
// @Produce application/vnd.mapbox-vector-tile
// @Param id path int true "ID транспортной линии"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 400 {object} map[string]string "Invalid line ID"
// @Failure 500 {string} string "Failed to generate tile"
// @Router /api/v1/transport/lines/{id}.pbf [get]
func (h *TileHandler) GetTransportLineTile(c *fiber.Ctx) error {
	// Parse string ID from path parameter
	lineIDStr := c.Params("id")
	lineID, err := strconv.ParseInt(lineIDStr, 10, 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid line ID format"})
	}

	tile, err := h.tileUC.GetTransportLineTile(c.Context(), lineID)
	if err != nil {
		h.logger.Error("Failed to get transport line tile",
			zap.Int64("line_id", lineID),
			zap.Error(err))
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
	return c.Send(tile)
}

// GetTransportLinesTile godoc
// @Summary Получение векторного тайла для нескольких транспортных линий
// @Description Возвращает векторный тайл (Mapbox Vector Tile) с геометрией и станциями нескольких транспортных линий (до 50 линий за запрос)
// @Tags Transport Tiles
// @Accept json
// @Produce application/vnd.mapbox-vector-tile
// @Param request body dto.TransportLinesRequest true "Массив ID линий"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 400 {object} map[string]string "Invalid request or too many IDs"
// @Failure 500 {string} string "Failed to generate tile"
// @Router /api/v1/transport/lines.pbf [post]
func (h *TileHandler) GetTransportLinesTile(c *fiber.Ctx) error {
	var req struct {
		IDs []string `json:"ids"` // Accept string IDs from frontend
	}

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if len(req.IDs) == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "Line IDs are required"})
	}

	if len(req.IDs) > 50 {
		return c.Status(400).JSON(fiber.Map{"error": "Maximum 50 line IDs allowed"})
	}

	// Convert string IDs to int64
	lineIDs := make([]int64, 0, len(req.IDs))
	for _, idStr := range req.IDs {
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			return c.Status(400).JSON(fiber.Map{"error": "Invalid line ID format"})
		}
		lineIDs = append(lineIDs, id)
	}

	tile, err := h.tileUC.GetTransportLinesTile(c.Context(), lineIDs)
	if err != nil {
		h.logger.Error("Failed to get transport lines tile",
			zap.Int64s("line_ids", lineIDs),
			zap.Error(err))
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
	return c.Send(tile)
}

// GetRadiusTiles godoc
// @Summary Получение всех данных в радиусе в формате векторного тайла
// @Description Возвращает векторный тайл (Mapbox Vector Tile) со всеми типами данных в указанном радиусе от точки: границы, транспорт, POI, зеленые зоны, воду и т.д. Можно фильтровать слои через параметр layers.
// @Tags Tiles
// @Accept json
// @Produce application/vnd.mapbox-vector-tile
// @Param request body dto.RadiusTilesRequest true "Координаты центра, радиус и опциональный список слоев"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {string} string "Failed to generate tile"
// @Router /api/v1/radius/tiles.pbf [post]
func (h *TileHandler) GetRadiusTiles(c *fiber.Ctx) error {
	var req dto.RadiusTilesRequest

	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	tile, err := h.tileUC.GetRadiusTiles(c.Context(), req)
	if err != nil {
		h.logger.Error("Failed to get radius tiles",
			zap.Float64("lat", req.Lat),
			zap.Float64("lon", req.Lon),
			zap.Float64("radius_km", req.RadiusKm),
			zap.Error(err))
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
	c.Set("Cache-Control", "public, max-age=3600")
	return c.Send(tile)
}
