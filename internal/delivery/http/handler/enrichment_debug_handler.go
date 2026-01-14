package handler

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/pkg/validator"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

// EnrichmentDebugHandler - обработчик для дебага/тестирования обогащения
type EnrichmentDebugHandler struct {
	enrichmentDebugUC *usecase.EnrichmentDebugUseCase
	logger            *zap.Logger
}

// NewEnrichmentDebugHandler - создание нового EnrichmentDebugHandler
func NewEnrichmentDebugHandler(
	enrichmentDebugUC *usecase.EnrichmentDebugUseCase,
	logger *zap.Logger,
) *EnrichmentDebugHandler {
	return &EnrichmentDebugHandler{
		enrichmentDebugUC: enrichmentDebugUC,
		logger:            logger,
	}
}

// GetNearestTransportEnriched godoc
// @Summary Получение ближайших станций транспорта (дебаг/тест)
// @Description Возвращает ближайшие станции общественного транспорта с детальной информацией для тестирования логики обогащения. Включает линейное расстояние, примерное пешеходное расстояние и время, информацию о линиях с цветами.
// @Tags Enrichment Debug
// @Accept json
// @Produce json
// @Param request body dto.EnrichmentDebugTransportRequest true "Параметры поиска"
// @Success 200 {object} utils.SuccessResponse{data=dto.EnrichmentDebugTransportResponse} "Список ближайших станций"
// @Failure 400 {object} utils.ErrorResponse "Неверные параметры запроса"
// @Failure 500 {object} utils.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/debug/enrichment/transport [post]
func (h *EnrichmentDebugHandler) GetNearestTransportEnriched(c *fiber.Ctx) error {
	var req dto.EnrichmentDebugTransportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.enrichmentDebugUC.GetNearestTransportEnriched(c.Context(), req)
	if err != nil {
		h.logger.Error("Failed to get nearest transport enriched",
			zap.Float64("lat", req.Lat),
			zap.Float64("lon", req.Lon),
			zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Meta.TotalFound,
	})
}

// GetNearestTransportEnrichedGET godoc
// @Summary Получение ближайших станций транспорта (GET дебаг/тест)
// @Description Возвращает ближайшие станции общественного транспорта с детальной информацией. GET версия для быстрого тестирования в браузере.
// @Tags Enrichment Debug
// @Accept json
// @Produce json
// @Param lat query number true "Широта" minimum(-90) maximum(90)
// @Param lon query number true "Долгота" minimum(-180) maximum(180)
// @Param types query string false "Типы транспорта через запятую (metro,train,tram,bus)"
// @Param max_distance query number false "Максимальное расстояние в метрах (100-10000)"
// @Param limit query int false "Максимальное количество результатов (1-50)"
// @Success 200 {object} utils.SuccessResponse{data=dto.EnrichmentDebugTransportResponse} "Список ближайших станций"
// @Failure 400 {object} utils.ErrorResponse "Неверные параметры запроса"
// @Failure 500 {object} utils.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/debug/enrichment/transport [get]
func (h *EnrichmentDebugHandler) GetNearestTransportEnrichedGET(c *fiber.Ctx) error {
	lat := c.QueryFloat("lat", 0)
	lon := c.QueryFloat("lon", 0)
	maxDistance := c.QueryFloat("max_distance", 1500)
	limit := c.QueryInt("limit", 10)

	// Parse types
	var types []string
	typesParam := c.Query("types", "")
	if typesParam != "" {
		types = splitAndTrim(typesParam, ",")
	}

	req := dto.EnrichmentDebugTransportRequest{
		Lat:         lat,
		Lon:         lon,
		Types:       types,
		MaxDistance: maxDistance,
		Limit:       limit,
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.enrichmentDebugUC.GetNearestTransportEnriched(c.Context(), req)
	if err != nil {
		h.logger.Error("Failed to get nearest transport enriched",
			zap.Float64("lat", req.Lat),
			zap.Float64("lon", req.Lon),
			zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Meta.TotalFound,
	})
}

// splitAndTrim splits a string by separator and trims whitespace
func splitAndTrim(s, sep string) []string {
	if s == "" {
		return nil
	}
	parts := make([]string, 0)
	for _, p := range strings.Split(s, sep) {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			parts = append(parts, trimmed)
		}
	}
	return parts
}

// GetNearestTransportEnrichedBatch godoc
// @Summary Batch получение ближайших станций транспорта для нескольких точек
// @Description Возвращает ближайшие станции общественного транспорта с информацией о линиях для пачки координат одним эффективным запросом. Идеально для обогащения множества объектов.
// @Tags Enrichment Debug
// @Accept json
// @Produce json
// @Param request body dto.EnrichmentDebugTransportBatchRequest true "Batch параметры поиска"
// @Success 200 {object} utils.SuccessResponse{data=dto.EnrichmentDebugTransportBatchResponse} "Результаты для каждой точки"
// @Failure 400 {object} utils.ErrorResponse "Неверные параметры запроса"
// @Failure 500 {object} utils.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/debug/enrichment/transport/batch [post]
func (h *EnrichmentDebugHandler) GetNearestTransportEnrichedBatch(c *fiber.Ctx) error {
	var req dto.EnrichmentDebugTransportBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.enrichmentDebugUC.GetNearestTransportEnrichedBatch(c.Context(), req)
	if err != nil {
		h.logger.Error("Failed to get batch nearest transport enriched",
			zap.Int("points_count", len(req.Points)),
			zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Meta.TotalStations,
	})
}

// EnrichLocation godoc
// @Summary Обогащение локации (дебаг/тест)
// @Description Тестирует полный процесс обогащения локации: резолвинг границ, поиск транспорта. Возвращает обогащённые данные в том же формате, что и worker обогащения.
// @Tags Enrichment Debug
// @Accept json
// @Produce json
// @Param request body dto.EnrichmentDebugLocationRequest true "Данные локации для обогащения"
// @Success 200 {object} utils.SuccessResponse{data=dto.EnrichmentDebugLocationResponse} "Обогащённая локация"
// @Failure 400 {object} utils.ErrorResponse "Неверные параметры запроса"
// @Failure 500 {object} utils.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/debug/enrichment/location [post]
func (h *EnrichmentDebugHandler) EnrichLocation(c *fiber.Ctx) error {
	var req dto.EnrichmentDebugLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.enrichmentDebugUC.EnrichLocation(c.Context(), req)
	if err != nil {
		h.logger.Error("Failed to enrich location",
			zap.String("country", req.Country),
			zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, nil)
}

// EnrichLocationGET godoc
// @Summary Обогащение локации (GET дебаг/тест)
// @Description Тестирует полный процесс обогащения локации. GET версия для быстрого тестирования в браузере.
// @Tags Enrichment Debug
// @Accept json
// @Produce json
// @Param country query string true "Страна (обязательно)"
// @Param region query string false "Регион"
// @Param province query string false "Провинция"
// @Param city query string false "Город"
// @Param district query string false "Район"
// @Param neighborhood query string false "Район/квартал"
// @Param street query string false "Улица"
// @Param house_number query string false "Номер дома"
// @Param postal_code query string false "Почтовый индекс"
// @Param lat query number false "Широта" minimum(-90) maximum(90)
// @Param lon query number false "Долгота" minimum(-180) maximum(180)
// @Success 200 {object} utils.SuccessResponse{data=dto.EnrichmentDebugLocationResponse} "Обогащённая локация"
// @Failure 400 {object} utils.ErrorResponse "Неверные параметры запроса"
// @Failure 500 {object} utils.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/debug/enrichment/location [get]
func (h *EnrichmentDebugHandler) EnrichLocationGET(c *fiber.Ctx) error {
	country := c.Query("country", "")
	if country == "" {
		return c.Status(400).JSON(fiber.Map{"error": "country parameter is required"})
	}

	req := dto.EnrichmentDebugLocationRequest{
		Country: country,
	}

	// Optional string fields
	if region := c.Query("region", ""); region != "" {
		req.Region = &region
	}
	if province := c.Query("province", ""); province != "" {
		req.Province = &province
	}
	if city := c.Query("city", ""); city != "" {
		req.City = &city
	}
	if district := c.Query("district", ""); district != "" {
		req.District = &district
	}
	if neighborhood := c.Query("neighborhood", ""); neighborhood != "" {
		req.Neighborhood = &neighborhood
	}
	if street := c.Query("street", ""); street != "" {
		req.Street = &street
	}
	if houseNumber := c.Query("house_number", ""); houseNumber != "" {
		req.HouseNumber = &houseNumber
	}
	if postalCode := c.Query("postal_code", ""); postalCode != "" {
		req.PostalCode = &postalCode
	}

	// Optional coordinates
	if lat := c.QueryFloat("lat", -999); lat != -999 {
		req.Latitude = &lat
	}
	if lon := c.QueryFloat("lon", -999); lon != -999 {
		req.Longitude = &lon
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.enrichmentDebugUC.EnrichLocation(c.Context(), req)
	if err != nil {
		h.logger.Error("Failed to enrich location",
			zap.String("country", req.Country),
			zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, nil)
}

// EnrichLocationBatch godoc
// @Summary Батч обогащение локаций (дебаг/тест)
// @Description Обогащает пачку локаций эффективно (2 параллельных запроса в БД). Локации делятся на 2 группы: с координатами (reverse geocoding) и без (поиск по названиям). Оптимально для обработки 50+ локаций за раз.
// @Tags Enrichment Debug
// @Accept json
// @Produce json
// @Param request body dto.EnrichmentDebugLocationBatchRequest true "Данные локаций для обогащения"
// @Success 200 {object} utils.SuccessResponse{data=dto.EnrichmentDebugLocationBatchResponse} "Обогащённые локации"
// @Failure 400 {object} utils.ErrorResponse "Неверные параметры запроса"
// @Failure 500 {object} utils.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/debug/enrichment/location/batch [post]
func (h *EnrichmentDebugHandler) EnrichLocationBatch(c *fiber.Ctx) error {
	var req dto.EnrichmentDebugLocationBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.enrichmentDebugUC.EnrichLocationBatch(c.Context(), req)
	if err != nil {
		h.logger.Error("Failed to batch enrich locations",
			zap.Int("locations_count", len(req.Locations)),
			zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Meta.TotalLocations,
	})
}

// ========== Priority Transport Handlers ==========

// GetPriorityTransport godoc
// @Summary Получение ближайшего транспорта с приоритетом
// @Description Возвращает ближайший транспорт с приоритизацией по типу. Если в радиусе есть metro/train - возвращает только их. Иначе возвращает bus/tram. Включает информацию о линиях (L2, L4 для метро, номера для автобусов) и их цветах.
// @Tags Enrichment Debug
// @Accept json
// @Produce json
// @Param lat query number true "Широта" minimum(-90) maximum(90)
// @Param lon query number true "Долгота" minimum(-180) maximum(180)
// @Param radius query number false "Радиус поиска в метрах (100-10000, default 1500)"
// @Param limit query int false "Максимальное количество результатов (1-20, default 5)"
// @Success 200 {object} utils.SuccessResponse{data=dto.PriorityTransportResponse} "Ближайшие станции с приоритетом"
// @Failure 400 {object} utils.ErrorResponse "Неверные параметры запроса"
// @Failure 500 {object} utils.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/debug/transport/priority [get]
func (h *EnrichmentDebugHandler) GetPriorityTransport(c *fiber.Ctx) error {
	lat := c.QueryFloat("lat", 0)
	lon := c.QueryFloat("lon", 0)
	radius := c.QueryFloat("radius", 1500)
	limit := c.QueryInt("limit", 5)

	req := dto.PriorityTransportRequest{
		Lat:    lat,
		Lon:    lon,
		Radius: radius,
		Limit:  limit,
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.enrichmentDebugUC.GetNearestTransportByPriority(c.Context(), req)
	if err != nil {
		h.logger.Error("Failed to get priority transport",
			zap.Float64("lat", req.Lat),
			zap.Float64("lon", req.Lon),
			zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Meta.TotalFound,
	})
}

// GetPriorityTransportPOST godoc
// @Summary Получение ближайшего транспорта с приоритетом (POST)
// @Description POST версия метода для программного использования
// @Tags Enrichment Debug
// @Accept json
// @Produce json
// @Param request body dto.PriorityTransportRequest true "Параметры поиска"
// @Success 200 {object} utils.SuccessResponse{data=dto.PriorityTransportResponse} "Ближайшие станции с приоритетом"
// @Failure 400 {object} utils.ErrorResponse "Неверные параметры запроса"
// @Failure 500 {object} utils.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/debug/transport/priority [post]
func (h *EnrichmentDebugHandler) GetPriorityTransportPOST(c *fiber.Ctx) error {
	var req dto.PriorityTransportRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.enrichmentDebugUC.GetNearestTransportByPriority(c.Context(), req)
	if err != nil {
		h.logger.Error("Failed to get priority transport",
			zap.Float64("lat", req.Lat),
			zap.Float64("lon", req.Lon),
			zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Meta.TotalFound,
	})
}

// GetPriorityTransportBatch godoc
// @Summary Batch получение ближайшего транспорта с приоритетом
// @Description Возвращает ближайший транспорт с приоритетом для множества точек одним эффективным запросом. Для каждой точки применяется логика приоритизации: metro/train -> bus/tram.
// @Tags Enrichment Debug
// @Accept json
// @Produce json
// @Param request body dto.PriorityTransportBatchRequest true "Batch параметры поиска"
// @Success 200 {object} utils.SuccessResponse{data=dto.PriorityTransportBatchResponse} "Результаты для каждой точки"
// @Failure 400 {object} utils.ErrorResponse "Неверные параметры запроса"
// @Failure 500 {object} utils.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/debug/transport/priority/batch [post]
func (h *EnrichmentDebugHandler) GetPriorityTransportBatch(c *fiber.Ctx) error {
	var req dto.PriorityTransportBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	result, err := h.enrichmentDebugUC.GetNearestTransportByPriorityBatch(c.Context(), req)
	if err != nil {
		h.logger.Error("Failed to get batch priority transport",
			zap.Int("points_count", len(req.Points)),
			zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Meta.TotalStations,
	})
}
