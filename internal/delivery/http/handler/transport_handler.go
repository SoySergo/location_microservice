package handler

import (
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/pkg/validator"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

// TransportHandler - обработчик для транспортных запросов
type TransportHandler struct {
	transportUC *usecase.TransportUseCase
	logger      *zap.Logger
}

// NewTransportHandler - создание нового TransportHandler
func NewTransportHandler(transportUC *usecase.TransportUseCase, logger *zap.Logger) *TransportHandler {
	return &TransportHandler{
		transportUC: transportUC,
		logger:      logger,
	}
}

// GetNearestStations - поиск ближайших станций транспорта
func (h *TransportHandler) GetNearestStations(c *fiber.Ctx) error {
	var req dto.NearestTransportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.transportUC.GetNearestStations(c.Context(), req)
	if err != nil {
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: len(result.Stations),
	})
}

// BatchGetNearestStations - пакетный поиск ближайших станций для нескольких точек
func (h *TransportHandler) BatchGetNearestStations(c *fiber.Ctx) error {
	var req dto.BatchNearestTransportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.transportUC.BatchGetNearestStations(c.Context(), req)
	if err != nil {
		return utils.SendError(c, err)
	}

	// Подсчет общего количества найденных станций
	totalStations := 0
	for _, stations := range result.Results {
		totalStations += len(stations)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: totalStations,
	})
}

// GetTransportTileByTypes - получение тайла с транспортом с фильтрацией по типам
// GET /api/v1/tiles/transport/{z}/{x}/{y}.pbf?types=metro,bus
func (h *TransportHandler) GetTransportTileByTypes(c *fiber.Ctx) error {
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
typesParam := c.Query("types", "")
var types []string
if typesParam != "" {
types = strings.Split(typesParam, ",")
// Trim spaces
for i, t := range types {
types[i] = strings.TrimSpace(t)
}
}

// Получение тайла
tile, err := h.transportUC.GetTransportTileByTypes(c.Context(), z, x, y, types)
if err != nil {
h.logger.Error("Failed to get transport tile by types",
zap.Int("z", z),
zap.Int("x", x),
zap.Int("y", y),
zap.Strings("types", types),
zap.Error(err))
return c.Status(500).JSON(fiber.Map{"error": "Failed to generate tile"})
}

// Устанавливаем заголовки
c.Set("Content-Type", "application/x-protobuf")
c.Set("Content-Encoding", "gzip")
c.Set("Cache-Control", "public, max-age=3600")

return c.Send(tile)
}

// GetLinesByStationID - получение линий для станции
// GET /api/v1/transport/station/{station_id}/lines
func (h *TransportHandler) GetLinesByStationID(c *fiber.Ctx) error {
// Парсинг station ID
stationID, err := strconv.ParseInt(c.Params("station_id"), 10, 64)
if err != nil {
return c.Status(400).JSON(fiber.Map{"error": "Invalid station ID"})
}

// Получение линий
lines, err := h.transportUC.GetLinesByStationID(c.Context(), stationID)
if err != nil {
h.logger.Error("Failed to get lines by station ID",
zap.Int64("station_id", stationID),
zap.Error(err))
return c.Status(500).JSON(fiber.Map{"error": "Failed to get lines"})
}

// Преобразование в DTO
result := make([]fiber.Map, 0, len(lines))
for _, line := range lines {
lineMap := fiber.Map{
"id":   strconv.FormatInt(line.ID, 10),
"name": line.Name,
"ref":  line.Ref,
"type": line.Type,
}
if line.Color != nil {
lineMap["color"] = *line.Color
}
if line.TextColor != nil {
lineMap["text_color"] = *line.TextColor
}
if line.Operator != nil {
lineMap["operator"] = *line.Operator
}
if line.Network != nil {
lineMap["network"] = *line.Network
}
result = append(result, lineMap)
}

return c.JSON(fiber.Map{
"lines": result,
})
}
