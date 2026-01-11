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

// GetNearestStations godoc
// @Summary Поиск ближайших станций транспорта
// @Description Находит ближайшие станции общественного транспорта (метро, автобусы, трамваи, поезда) в указанном радиусе от точки. Возвращает информацию о станциях и проходящих через них линиях.
// @Tags Transport
// @Accept json
// @Produce json
// @Param request body dto.NearestTransportRequest true "Параметры поиска станций"
// @Success 200 {object} utils.SuccessResponse{data=dto.NearestTransportResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/transport/nearest [post]
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

// BatchGetNearestStations godoc
// @Summary Пакетный поиск ближайших станций для нескольких точек
// @Description Находит ближайшие станции общественного транспорта для нескольких точек одновременно (до 100 точек за запрос)
// @Tags Transport
// @Accept json
// @Produce json
// @Param request body dto.BatchNearestTransportRequest true "Массив точек и параметры поиска"
// @Success 200 {object} utils.SuccessResponse{data=dto.BatchNearestTransportResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/batch/transport/nearest [post]
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

// GetTransportTileByTypes godoc
// @Summary Получение векторного тайла с транспортом по типам
// @Description Возвращает векторный тайл (Mapbox Vector Tile) с транспортными станциями, отфильтрованными по типам. Поддерживает фильтрацию по metro, bus, tram, train.
// @Tags Transport Tiles
// @Accept json
// @Produce application/x-protobuf
// @Param z path int true "Zoom level (0-22)"
// @Param x path int true "Tile X coordinate"
// @Param y path int true "Tile Y coordinate"
// @Param types query string false "Типы транспорта через запятую (metro,bus,tram,train)"
// @Success 200 {file} byte "Vector tile in PBF format"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/tiles/transport/{z}/{x}/{y}.pbf [get]
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

// GetLinesByStationID godoc
// @Summary Получение линий для станции
// @Description Возвращает список транспортных линий, которые проходят через указанную станцию (с информацией о цветах, операторах и т.д.)
// @Tags Transport
// @Accept json
// @Produce json
// @Param station_id path int true "ID станции"
// @Success 200 {object} map[string]interface{} "Список линий транспорта"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/v1/transport/station/{station_id}/lines [get]
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
