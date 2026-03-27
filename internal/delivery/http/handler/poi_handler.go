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

// GetPOIInBBox godoc
// @Summary Получение POI в видимой области карты (bbox)
// @Description Возвращает точки интереса в указанном прямоугольнике карты с пагинацией. Поддерживает фильтрацию по категориям и подкатегориям.
// @Tags POI
// @Accept json
// @Produce json
// @Param sw_lat query number true "Широта юго-западного угла"
// @Param sw_lon query number true "Долгота юго-западного угла"
// @Param ne_lat query number true "Широта северо-восточного угла"
// @Param ne_lon query number true "Долгота северо-восточного угла"
// @Param categories query string false "Категории через запятую"
// @Param subcategories query string false "Подкатегории через запятую"
// @Param limit query int false "Лимит результатов (по умолчанию 10, максимум 100)"
// @Param offset query int false "Смещение для пагинации"
// @Success 200 {object} utils.SuccessResponse{data=dto.BBoxPOIResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/poi/bbox [get]
func (h *POIHandler) GetPOIInBBox(c *fiber.Ctx) error {
	swLat, err := strconv.ParseFloat(c.Query("sw_lat"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sw_lat"})
	}
	swLon, err := strconv.ParseFloat(c.Query("sw_lon"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid sw_lon"})
	}
	neLat, err := strconv.ParseFloat(c.Query("ne_lat"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid ne_lat"})
	}
	neLon, err := strconv.ParseFloat(c.Query("ne_lon"), 64)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "invalid ne_lon"})
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	var categories []string
	if cats := c.Query("categories", ""); cats != "" {
		categories = strings.Split(cats, ",")
		for i := range categories {
			categories[i] = strings.TrimSpace(categories[i])
		}
	}

	var subcategories []string
	if subs := c.Query("subcategories", ""); subs != "" {
		subcategories = strings.Split(subs, ",")
		for i := range subcategories {
			subcategories[i] = strings.TrimSpace(subcategories[i])
		}
	}

	req := dto.BBoxPOIRequest{
		SwLat:         swLat,
		SwLon:         swLon,
		NeLat:         neLat,
		NeLon:         neLon,
		Categories:    categories,
		Subcategories: subcategories,
		Limit:         limit,
		Offset:        offset,
	}

	result, err := h.poiUC.GetPOIInBBox(c.Context(), req)
	if err != nil {
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{Total: result.Total})
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
