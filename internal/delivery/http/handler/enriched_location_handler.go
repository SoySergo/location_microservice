package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/pkg/validator"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

// EnrichedLocationHandler - обработчик для обогащения локаций
type EnrichedLocationHandler struct {
	enrichedLocationUC *usecase.EnrichedLocationUseCase
	logger             *zap.Logger
}

// NewEnrichedLocationHandler создает новый EnrichedLocationHandler
func NewEnrichedLocationHandler(
	enrichedLocationUC *usecase.EnrichedLocationUseCase,
	logger *zap.Logger,
) *EnrichedLocationHandler {
	return &EnrichedLocationHandler{
		enrichedLocationUC: enrichedLocationUC,
		logger:             logger,
	}
}

// EnrichLocationBatch godoc
// @Summary Batch обогащение локаций
// @Description Обогащает пачку локаций: определяет административные границы и находит ближайший транспорт для visible локаций
// @Tags Location Enrichment
// @Accept json
// @Produce json
// @Param request body dto.EnrichLocationBatchRequest true "Массив локаций для обогащения (до 100)"
// @Success 200 {object} utils.SuccessResponse{data=dto.EnrichLocationBatchResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/locations/enrich/batch [post]
func (h *EnrichedLocationHandler) EnrichLocationBatch(c *fiber.Ctx) error {
	var req dto.EnrichLocationBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	h.logger.Info("EnrichLocationBatch request",
		zap.Int("locations_count", len(req.Locations)))

	result, err := h.enrichedLocationUC.EnrichLocationBatch(c.Context(), req)
	if err != nil {
		h.logger.Error("EnrichLocationBatch failed", zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Meta.TotalLocations,
	})
}

// EnrichSingleLocation godoc
// @Summary Обогащение одной локации
// @Description Обогащает одну локацию: определяет административные границы и находит ближайший транспорт
// @Tags Location Enrichment
// @Accept json
// @Produce json
// @Param request body dto.EnrichSingleLocationRequest true "Данные локации"
// @Success 200 {object} utils.SuccessResponse{data=dto.EnrichedLocationResult}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/locations/enrich [post]
func (h *EnrichedLocationHandler) EnrichSingleLocation(c *fiber.Ctx) error {
	var req dto.EnrichSingleLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	h.logger.Info("EnrichSingleLocation request",
		zap.String("country", req.Country),
		zap.Stringp("city", req.City))

	// Конвертируем в batch request с одним элементом
	batchReq := dto.EnrichLocationBatchRequest{
		Locations: []dto.LocationInput{
			{
				Index:        0,
				Country:      req.Country,
				Region:       req.Region,
				Province:     req.Province,
				City:         req.City,
				District:     req.District,
				Neighborhood: req.Neighborhood,
				Latitude:     req.Latitude,
				Longitude:    req.Longitude,
				IsVisible:    req.IsVisible,
			},
		},
	}

	result, err := h.enrichedLocationUC.EnrichLocationBatch(c.Context(), batchReq)
	if err != nil {
		h.logger.Error("EnrichSingleLocation failed", zap.Error(err))
		return utils.SendError(c, err)
	}

	if len(result.Results) == 0 {
		return c.Status(500).JSON(fiber.Map{"error": "No results returned"})
	}

	return utils.SendSuccess(c, result.Results[0], nil)
}

// DetectLocationBatch godoc
// @Summary Batch детекция локаций (без транспорта)
// @Description Определяет административные границы для пачки локаций без поиска транспорта
// @Tags Location Detection
// @Accept json
// @Produce json
// @Param request body dto.DetectLocationBatchRequest true "Массив локаций для детекции (до 100)"
// @Success 200 {object} utils.SuccessResponse{data=dto.DetectLocationBatchResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/locations/detect/batch [post]
func (h *EnrichedLocationHandler) DetectLocationBatch(c *fiber.Ctx) error {
	var req dto.DetectLocationBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	h.logger.Info("DetectLocationBatch request",
		zap.Int("locations_count", len(req.Locations)))

	// Используем SearchUseCase напрямую через enrichedLocationUC
	// Или можно инжектить SearchUseCase отдельно
	result, err := h.enrichedLocationUC.DetectLocationBatch(c.Context(), req)
	if err != nil {
		h.logger.Error("DetectLocationBatch failed", zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Meta.TotalLocations,
	})
}

// GetPriorityTransport godoc
// @Summary Поиск ближайшего транспорта с приоритетом
// @Description Находит ближайший транспорт с приоритетом (metro/train → bus/tram)
// @Tags Transport
// @Accept json
// @Produce json
// @Param lat query number true "Широта"
// @Param lon query number true "Долгота"
// @Param radius query number false "Радиус поиска в метрах" default(1500)
// @Param limit query int false "Максимальное количество станций" default(5)
// @Success 200 {object} utils.SuccessResponse{data=dto.PriorityTransportResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/transport/priority [get]
func (h *EnrichedLocationHandler) GetPriorityTransport(c *fiber.Ctx) error {
	lat := c.QueryFloat("lat", 0)
	lon := c.QueryFloat("lon", 0)
	radius := c.QueryFloat("radius", 1500)
	limit := c.QueryInt("limit", 5)

	if lat == 0 || lon == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "lat and lon are required"})
	}

	req := dto.PriorityTransportRequest{
		Lat:    lat,
		Lon:    lon,
		Radius: radius,
		Limit:  limit,
	}

	h.logger.Info("GetPriorityTransport request",
		zap.Float64("lat", lat),
		zap.Float64("lon", lon))

	result, err := h.enrichedLocationUC.GetPriorityTransport(c.Context(), req)
	if err != nil {
		h.logger.Error("GetPriorityTransport failed", zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Meta.TotalFound,
	})
}

// GetPriorityTransportBatch godoc
// @Summary Batch поиск ближайшего транспорта с приоритетом
// @Description Находит ближайший транспорт для нескольких точек одним запросом
// @Tags Transport
// @Accept json
// @Produce json
// @Param request body dto.PriorityTransportBatchRequest true "Массив точек"
// @Success 200 {object} utils.SuccessResponse{data=dto.PriorityTransportBatchResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/transport/priority/batch [post]
func (h *EnrichedLocationHandler) GetPriorityTransportBatch(c *fiber.Ctx) error {
	var req dto.PriorityTransportBatchRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if err := validator.Validate(&req); err != nil {
		return utils.SendError(c, err)
	}

	h.logger.Info("GetPriorityTransportBatch request",
		zap.Int("points_count", len(req.Points)))

	result, err := h.enrichedLocationUC.GetPriorityTransportBatch(c.Context(), req)
	if err != nil {
		h.logger.Error("GetPriorityTransportBatch failed", zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Meta.TotalStations,
	})
}
