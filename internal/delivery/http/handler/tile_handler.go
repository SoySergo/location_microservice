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

// GetBoundaryTile - получение тайла с границами
func (h *TileHandler) GetBoundaryTile(c *fiber.Ctx) error {
	z, _ := strconv.Atoi(c.Params("z"))
	x, _ := strconv.Atoi(c.Params("x"))
	y, _ := strconv.Atoi(c.Params("y"))

	tile, err := h.tileUC.GetBoundaryTile(c.Context(), z, x, y)
	if err != nil {
		h.logger.Error("Failed to get boundary tile", zap.Error(err))
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/x-protobuf")
	c.Set("Content-Encoding", "gzip")
	return c.Send(tile)
}

// GetTransportTile - получение тайла с транспортом
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

// GetGreenSpacesTile - получение тайла с зелеными зонами
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

// GetWaterTile - получение тайла с водными объектами
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

// GetBeachesTile - получение тайла с пляжами
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

// GetNoiseSourcesTile - получение тайла с источниками шума
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

// GetTouristZonesTile - получение тайла с туристическими зонами
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

// GetTransportLineTile - получение тайла для одной транспортной линии
func (h *TileHandler) GetTransportLineTile(c *fiber.Ctx) error {
	lineID := c.Params("id")
	if lineID == "" {
		return c.Status(400).JSON(fiber.Map{"error": "Line ID is required"})
	}

	tile, err := h.tileUC.GetTransportLineTile(c.Context(), lineID)
	if err != nil {
		h.logger.Error("Failed to get transport line tile",
			zap.String("line_id", lineID),
			zap.Error(err))
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
	return c.Send(tile)
}

// GetTransportLinesTile - получение тайла для нескольких транспортных линий
func (h *TileHandler) GetTransportLinesTile(c *fiber.Ctx) error {
	var req struct {
		IDs []string `json:"ids"`
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

	tile, err := h.tileUC.GetTransportLinesTile(c.Context(), req.IDs)
	if err != nil {
		h.logger.Error("Failed to get transport lines tile",
			zap.Strings("line_ids", req.IDs),
			zap.Error(err))
		return c.Status(500).SendString("Failed to generate tile")
	}

	c.Set("Content-Type", "application/vnd.mapbox-vector-tile")
	return c.Send(tile)
}

// GetRadiusTiles - получение тайла со всеми данными в радиусе от точки
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
