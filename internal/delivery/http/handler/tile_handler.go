package handler

import (
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/usecase"
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
