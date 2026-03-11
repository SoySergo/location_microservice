package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

// PropertyLocationHandler — обработчик для агрегированных данных локации объекта
type PropertyLocationHandler struct {
	propertyLocationUC *usecase.PropertyLocationUseCase
	logger             *zap.Logger
}

// NewPropertyLocationHandler создает новый PropertyLocationHandler
func NewPropertyLocationHandler(
	propertyLocationUC *usecase.PropertyLocationUseCase,
	logger *zap.Logger,
) *PropertyLocationHandler {
	return &PropertyLocationHandler{
		propertyLocationUC: propertyLocationUC,
		logger:             logger,
	}
}

// GetPropertyLocation godoc
// @Summary Агрегированные данные локации объекта
// @Description Возвращает агрегированные данные для страницы деталей объекта: ближайший транспорт с приоритетом, количество POI по категориям, наличие зелёных зон, воды и пляжей поблизости. Один запрос вместо 3-4 от фронтенда.
// @Tags Property Location
// @Produce json
// @Param lat query number true "Широта"
// @Param lon query number true "Долгота"
// @Param radius query int false "Радиус поиска в метрах" default(1000)
// @Success 200 {object} utils.SuccessResponse{data=dto.PropertyLocationResponse}
// @Failure 400 {object} utils.ErrorResponse
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/property-location [get]
func (h *PropertyLocationHandler) GetPropertyLocation(c *fiber.Ctx) error {
	lat := c.QueryFloat("lat", 0)
	lon := c.QueryFloat("lon", 0)
	radius := c.QueryInt("radius", 0)

	if lat == 0 || lon == 0 {
		return c.Status(400).JSON(fiber.Map{"error": "lat and lon are required"})
	}

	req := dto.PropertyLocationRequest{
		Lat:    lat,
		Lon:    lon,
		Radius: radius,
	}

	h.logger.Info("GetPropertyLocation request",
		zap.Float64("lat", lat),
		zap.Float64("lon", lon),
		zap.Int("radius", radius))

	result, err := h.propertyLocationUC.GetPropertyLocationData(c.Context(), req)
	if err != nil {
		h.logger.Error("GetPropertyLocation failed", zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, result, nil)
}
