package handler

import (
	"github.com/gofiber/fiber/v2"
	"github.com/location-microservice/internal/pkg/utils"
	"github.com/location-microservice/internal/usecase"
	"go.uber.org/zap"
)

// StatsHandler обрабатывает запросы для статистики
type StatsHandler struct {
	statsUC *usecase.StatsUseCase
	logger  *zap.Logger
}

// NewStatsHandler создает новый экземпляр StatsHandler
func NewStatsHandler(statsUC *usecase.StatsUseCase, logger *zap.Logger) *StatsHandler {
	return &StatsHandler{
		statsUC: statsUC,
		logger:  logger,
	}
}

// GetStatistics godoc
// @Summary Get system statistics
// @Description Возвращает агрегированную статистику по всем данным в системе
// @Tags Statistics
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Статистика системы"
// @Failure 500 {object} utils.ErrorResponse
// @Router /api/v1/stats [get]
func (h *StatsHandler) GetStatistics(c *fiber.Ctx) error {
	ctx := c.Context()

	h.logger.Debug("Handling get statistics request")

	stats, err := h.statsUC.GetStatistics(ctx)
	if err != nil {
		h.logger.Error("Failed to get statistics", zap.Error(err))
		return utils.SendError(c, err)
	}

	return utils.SendSuccess(c, stats, nil)
}
