package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/pkg/validator"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

// SearchHandler - обработчик для поисковых запросов
type SearchHandler struct {
	searchUC *usecase.SearchUseCase
	logger   *zap.Logger
}

// NewSearchHandler - создание нового SearchHandler
func NewSearchHandler(searchUC *usecase.SearchUseCase, logger *zap.Logger) *SearchHandler {
	return &SearchHandler{
		searchUC: searchUC,
		logger:   logger,
	}
}

// Search - поиск по текстовому запросу
func (h *SearchHandler) Search(c *fiber.Ctx) error {
	var req dto.SearchRequest
	req.Query = c.Query("q")
	req.Language = c.Query("language", "en")
	req.Limit = c.QueryInt("limit", 10)

	// Валидация
	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	// Выполнение use case
	result, err := h.searchUC.Search(c.Context(), req)
	if err != nil {
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Total,
	})
}

// ReverseGeocode - обратное геокодирование
func (h *SearchHandler) ReverseGeocode(c *fiber.Ctx) error {
	var req dto.ReverseGeocodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.searchUC.ReverseGeocode(c.Context(), req)
	if err != nil {
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, nil)
}

// BatchReverseGeocode - пакетное обратное геокодирование
func (h *SearchHandler) BatchReverseGeocode(c *fiber.Ctx) error {
	var req dto.BatchReverseGeocodeRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.searchUC.BatchReverseGeocode(c.Context(), req)
	if err != nil {
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, nil)
}

// GetBoundaryByID - получение границы по ID
func (h *SearchHandler) GetBoundaryByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ID required"})
	}

	// TODO: Добавить метод GetByID в use case
	return c.JSON(fiber.Map{"message": "Get boundary by ID - not implemented yet"})
}
