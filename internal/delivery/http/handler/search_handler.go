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

// Search godoc
// @Summary Поиск административных границ по тексту
// @Description Выполняет полнотекстовый поиск по административным границам (страны, регионы, города, районы). Поддерживает поиск на разных языках и фильтрацию по административным уровням.
// @Tags Search
// @Accept json
// @Produce json
// @Param q query string true "Поисковый запрос (минимум 2 символа)"
// @Param language query string false "Язык результатов (en, es, ca, ru, uk, fr, pt, it, de)" default(en)
// @Param limit query int false "Максимальное количество результатов" default(10)
// @Success 200 {object} utils.SuccessResponse{data=dto.SearchResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/search [get]
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

// ReverseGeocode godoc
// @Summary Обратное геокодирование
// @Description Определяет административный адрес (страна, регион, провинция, город, район) по географическим координатам
// @Tags Search
// @Accept json
// @Produce json
// @Param request body dto.ReverseGeocodeRequest true "Координаты точки"
// @Success 200 {object} utils.SuccessResponse{data=dto.ReverseGeocodeResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/reverse-geocode [post]
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

// BatchReverseGeocode godoc
// @Summary Пакетное обратное геокодирование
// @Description Определяет административные адреса для нескольких точек за один запрос (до 100 точек)
// @Tags Search
// @Accept json
// @Produce json
// @Param request body dto.BatchReverseGeocodeRequest true "Массив координат точек"
// @Success 200 {object} utils.SuccessResponse{data=dto.BatchReverseGeocodeResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/batch/reverse-geocode [post]
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

// GetBoundaryByID godoc
// @Summary Получение границы по ID
// @Description Возвращает подробную информацию об административной границе по её идентификатору
// @Tags Search
// @Accept json
// @Produce json
// @Param id path string true "ID административной границы"
// @Success 200 {object} map[string]interface{} "Информация о границе"
// @Failure 400 {object} utils.ErrorResponse
// @Failure 404 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/boundaries/{id} [get]
func (h *SearchHandler) GetBoundaryByID(c *fiber.Ctx) error {
	id := c.Params("id")
	if id == "" {
		return c.Status(400).JSON(fiber.Map{"error": "ID required"})
	}

	// TODO: Добавить метод GetByID в use case
	return c.JSON(fiber.Map{"message": "Get boundary by ID - not implemented yet"})
}
