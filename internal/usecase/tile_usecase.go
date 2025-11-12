package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/location-microservice/internal/domain/repository"
	"go.uber.org/zap"
)

type TileUseCase struct {
	boundaryRepo    repository.BoundaryRepository
	transportRepo   repository.TransportRepository
	environmentRepo repository.EnvironmentRepository
	cacheRepo       repository.CacheRepository
	logger          *zap.Logger
	tileCacheTTL    time.Duration
}

func NewTileUseCase(
	boundaryRepo repository.BoundaryRepository,
	transportRepo repository.TransportRepository,
	environmentRepo repository.EnvironmentRepository,
	cacheRepo repository.CacheRepository,
	logger *zap.Logger,
	tileCacheTTL time.Duration,
) *TileUseCase {
	return &TileUseCase{
		boundaryRepo:    boundaryRepo,
		transportRepo:   transportRepo,
		environmentRepo: environmentRepo,
		cacheRepo:       cacheRepo,
		logger:          logger,
		tileCacheTTL:    tileCacheTTL,
	}
}

func (uc *TileUseCase) GetBoundaryTile(ctx context.Context, z, x, y int) ([]byte, error) {
	// Check cache first
	cacheKey := fmt.Sprintf("tile:boundaries:%d:%d:%d", z, x, y)
	cached, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Generate tile
	tile, err := uc.boundaryRepo.GetTile(ctx, z, x, y)
	if err != nil {
		uc.logger.Error("Failed to get boundary tile", zap.Error(err))
		return nil, err
	}

	// Cache tile
	if err := uc.cacheRepo.Set(ctx, cacheKey, tile, uc.tileCacheTTL); err != nil {
		uc.logger.Warn("Failed to cache tile", zap.String("key", cacheKey), zap.Error(err))
	}

	return tile, nil
}

func (uc *TileUseCase) GetTransportTile(ctx context.Context, z, x, y int) ([]byte, error) {
	cacheKey := fmt.Sprintf("tile:transport:%d:%d:%d", z, x, y)
	cached, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		return cached, nil
	}

	tile, err := uc.transportRepo.GetTransportTile(ctx, z, x, y)
	if err != nil {
		return nil, err
	}

	_ = uc.cacheRepo.Set(ctx, cacheKey, tile, uc.tileCacheTTL)
	return tile, nil
}

func (uc *TileUseCase) GetGreenSpacesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	cacheKey := fmt.Sprintf("tile:greenspaces:%d:%d:%d", z, x, y)
	cached, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Используем environmentRepo для генерации тайла с зелеными зонами
	tile, err := uc.environmentRepo.GetGreenSpacesTile(ctx, z, x, y)
	if err != nil {
		uc.logger.Error("Failed to get green spaces tile", zap.Error(err))
		return nil, err
	}

	_ = uc.cacheRepo.Set(ctx, cacheKey, tile, uc.tileCacheTTL)
	return tile, nil
}

// GetTransportLineTile возвращает MVT тайл для одной транспортной линии
func (uc *TileUseCase) GetTransportLineTile(ctx context.Context, lineID string) ([]byte, error) {
	cacheKey := fmt.Sprintf("tile:line:%s", lineID)
	cached, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		return cached, nil
	}

	tile, err := uc.transportRepo.GetLineTile(ctx, lineID)
	if err != nil {
		uc.logger.Error("Failed to get transport line tile",
			zap.String("line_id", lineID),
			zap.Error(err))
		return nil, err
	}

	_ = uc.cacheRepo.Set(ctx, cacheKey, tile, uc.tileCacheTTL)
	return tile, nil
}

// GetTransportLinesTile возвращает MVT тайл для нескольких транспортных линий
func (uc *TileUseCase) GetTransportLinesTile(ctx context.Context, lineIDs []string) ([]byte, error) {
	// Создаем хеш-ключ из массива IDs для кеширования
	cacheKey := fmt.Sprintf("tile:lines:%v", lineIDs)
	cached, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		return cached, nil
	}

	tile, err := uc.transportRepo.GetLinesTile(ctx, lineIDs)
	if err != nil {
		uc.logger.Error("Failed to get transport lines tile",
			zap.Strings("line_ids", lineIDs),
			zap.Error(err))
		return nil, err
	}

	_ = uc.cacheRepo.Set(ctx, cacheKey, tile, uc.tileCacheTTL)
	return tile, nil
}
