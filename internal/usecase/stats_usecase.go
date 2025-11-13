package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"go.uber.org/zap"
)

// StatsUseCase обрабатывает бизнес-логику для статистики
type StatsUseCase struct {
	statsRepo repository.StatsRepository
	cacheRepo repository.CacheRepository
	logger    *zap.Logger
}

// NewStatsUseCase создает новый экземпляр StatsUseCase
func NewStatsUseCase(
	statsRepo repository.StatsRepository,
	cacheRepo repository.CacheRepository,
	logger *zap.Logger,
) *StatsUseCase {
	return &StatsUseCase{
		statsRepo: statsRepo,
		cacheRepo: cacheRepo,
		logger:    logger,
	}
}

// GetStatistics возвращает статистику, используя кеш когда возможно
func (uc *StatsUseCase) GetStatistics(ctx context.Context) (*domain.Statistics, error) {
	// 1. Проверяем кеш
	cached, err := uc.cacheRepo.GetStats(ctx)
	if err == nil && cached != nil {
		uc.logger.Debug("Statistics fetched from cache")
		return cached, nil
	}

	if err != nil {
		uc.logger.Warn("Failed to get stats from cache", zap.Error(err))
	}

	// 2. Получаем из БД
	uc.logger.Debug("Fetching statistics from database")
	stats, err := uc.statsRepo.GetStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("get statistics from db: %w", err)
	}

	// 3. Кешируем на 1 час
	if err := uc.cacheRepo.SetStats(ctx, stats, time.Hour); err != nil {
		uc.logger.Warn("Failed to cache stats", zap.Error(err))
		// Не возвращаем ошибку, т.к. данные уже получены
	} else {
		uc.logger.Debug("Statistics cached successfully")
	}

	return stats, nil
}

// RefreshStatistics принудительно обновляет статистику
func (uc *StatsUseCase) RefreshStatistics(ctx context.Context) (*domain.Statistics, error) {
	uc.logger.Info("Refreshing statistics")

	// Получаем свежую статистику из БД
	stats, err := uc.statsRepo.GetStatistics(ctx)
	if err != nil {
		return nil, fmt.Errorf("refresh statistics: %w", err)
	}

	// Обновляем кеш
	if err := uc.cacheRepo.SetStats(ctx, stats, time.Hour); err != nil {
		uc.logger.Warn("Failed to cache refreshed stats", zap.Error(err))
	}

	uc.logger.Info("Statistics refreshed successfully")
	return stats, nil
}
