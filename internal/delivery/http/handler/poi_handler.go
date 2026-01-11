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

// SearchByRadius godoc
// @Summary Поиск точек интереса (POI) в радиусе
// @Description Находит точки интереса (магазины, рестораны, больницы и т.д.) в указанном радиусе от точки. Поддерживает фильтрацию по категориям.
// @Tags POI
// @Accept json
// @Produce json
// @Param request body dto.RadiusPOIRequest true "Параметры поиска POI"
// @Success 200 {object} utils.SuccessResponse{data=dto.RadiusPOIResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/radius/poi [post]
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

// GetCategories godoc
// @Summary Получение списка категорий POI
// @Description Возвращает полный список доступных категорий точек интереса (healthcare, shopping, education и т.д.) на указанном языке
// @Tags POI
// @Accept json
// @Produce json
// @Param language query string false "Язык результатов (en, es, ca, ru, uk, fr, pt, it, de)" default(en)
// @Success 200 {object} utils.SuccessResponse{data=map[string]interface{}}
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/poi/categories [get]
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

// GetSubcategories godoc
// @Summary Получение подкатегорий для категории
// @Description Возвращает список подкатегорий для указанной категории POI (например, для healthcare: pharmacy, hospital, clinic)
// @Tags POI
// @Accept json
// @Produce json
// @Param id path int true "ID категории"
// @Param language query string false "Язык результатов (en, es, ca, ru, uk, fr, pt, it, de)" default(en)
// @Success 200 {object} utils.SuccessResponse{data=map[string]interface{}}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/poi/categories/{id}/subcategories [get]
func (h *POIHandler) GetSubcategories(c *fiber.Ctx) error {
	// Parse int ID from path parameter (ParamsInt already handles parsing)
	categoryID, err := c.ParamsInt("id")
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid category ID format"})
	}
	lang := c.Query("language", "en")

	subcategories, err := h.poiUC.GetSubcategories(c.Context(), int64(categoryID), lang)
	if err != nil {
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, fiber.Map{
		"subcategories": subcategories,
	}, &utils.Meta{
		Total: len(subcategories),
	})
}
