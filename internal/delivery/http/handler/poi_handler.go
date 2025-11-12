package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/pkg/validator"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

// POIHandler - обработчик для POI (точки интереса) запросов
type POIHandler struct {
	poiUC  *usecase.POIUseCase
	logger *zap.Logger
}

// NewPOIHandler - создание нового POIHandler
func NewPOIHandler(poiUC *usecase.POIUseCase, logger *zap.Logger) *POIHandler {
	return &POIHandler{
		poiUC:  poiUC,
		logger: logger,
	}
}

// SearchByRadius - поиск POI в радиусе
func (h *POIHandler) SearchByRadius(c *fiber.Ctx) error {
	var req dto.RadiusPOIRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.poiUC.SearchByRadius(c.Context(), req)
	if err != nil {
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Total,
	})
}

// GetCategories - получение списка категорий POI
func (h *POIHandler) GetCategories(c *fiber.Ctx) error {
	lang := c.Query("language", "en")

	categories, err := h.poiUC.GetCategories(c.Context(), lang)
	if err != nil {
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, fiber.Map{
		"categories": categories,
	}, &utils.Meta{
		Total: len(categories),
	})
}

// GetSubcategories - получение подкатегорий для категории
func (h *POIHandler) GetSubcategories(c *fiber.Ctx) error {
	categoryID := c.Params("id")
	lang := c.Query("language", "en")

	subcategories, err := h.poiUC.GetSubcategories(c.Context(), categoryID, lang)
	if err != nil {
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, fiber.Map{
		"subcategories": subcategories,
	}, &utils.Meta{
		Total: len(subcategories),
	})
}
