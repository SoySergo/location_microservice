package handler

import (
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
