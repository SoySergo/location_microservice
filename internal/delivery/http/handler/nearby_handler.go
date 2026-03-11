package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/usecase"
	"go.uber.org/zap"
)

// NearbyHandler — обработчик для получения данных поблизости по категории
type NearbyHandler struct {
	nearbyUC *usecase.NearbyUseCase
	logger   *zap.Logger
}

// NewNearbyHandler создает новый NearbyHandler
func NewNearbyHandler(
	nearbyUC *usecase.NearbyUseCase,
	logger *zap.Logger,
) *NearbyHandler {
	return &NearbyHandler{
		nearbyUC: nearbyUC,
		logger:   logger,
	}
}

// GetNearby godoc
// @Summary Данные поблизости по категории
// @Description Возвращает список объектов поблизости для выбранной категории.
// @Description Для "transport" — станции с приоритетом (metro/train → tram → bus), линиями и временем пешком.
// @Description Для остальных категорий — POI с именем, координатами и расстоянием.
// @Description Категории: transport, schools, medical, groceries, shopping, restaurants, sports, entertainment, parks, beauty, attractions
// @Tags Nearby
// @Produce json
// @Param category path string true "Категория фильтра" Enums(transport, schools, medical, groceries, shopping, restaurants, sports, entertainment, parks, beauty, attractions)
// @Param lat query number true "Широта"
// @Param lon query number true "Долгота"
// @Param radius query number false "Радиус поиска в км (для POI) или метрах (для transport)" default(1)
// @Param limit query int false "Максимальное количество результатов" default(20)
// @Success 200 {object} utils.SuccessResponse "Для transport: data=dto.PriorityTransportResponse, для остальных: data=dto.NearbyPOIResponse"
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/nearby/{category} [get]
func (h *NearbyHandler) GetNearby(c *fiber.Ctx) error {
	category := c.Params("category")
	if !domain.IsValidNearbyCategory(category) {
		return c.Status(400).JSON(fiber.Map{"error": "invalid category: " + category})
	}

	lat := c.QueryFloat("lat", 0)
	lon := c.QueryFloat("lon", 0)
	if lat == 0 || lon == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "lat and lon are required"})
	}

	radius := c.QueryFloat("radius", 0)
	limit := c.QueryInt("limit", 0)

	h.logger.Info("GetNearby request",
		zap.String("category", category),
		zap.Float64("lat", lat),
		zap.Float64("lon", lon),
		zap.Float64("radius", radius),
		zap.Int("limit", limit))

	if category == domain.TransportCategory {
		// Для транспорта radius в метрах
		result, err := h.nearbyUC.GetNearbyTransport(c.Context(), lat, lon, radius, limit)
		if err != nil {
			h.logger.Error("GetNearbyTransport failed", zap.Error(err))
			return utils.SendError(c, err)
		}
		return utils.SendSuccess(c, result, &utils.Meta{
			Total: result.Meta.TotalFound,
		})
	}

	// Для POI radius в километрах
	result, err := h.nearbyUC.GetNearbyPOI(c.Context(), category, lat, lon, radius, limit)
	if err != nil {
		h.logger.Error("GetNearbyPOI failed", zap.String("category", category), zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, &utils.Meta{
		Total: result.Total,
	})
}
