package usecase

import (
	"bytes"
	"context"
	"crypto/md5"
	"fmt"
	"sync"
	"time"

	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/usecase/dto"
	"go.uber.org/zap"
)

type TileUseCase struct {
	boundaryRepo    repository.BoundaryRepository
	transportRepo   repository.TransportRepository
	environmentRepo repository.EnvironmentRepository
	poiRepo         repository.POIRepository
	cacheRepo       repository.CacheRepository
	logger          *zap.Logger
	tileCacheTTL    time.Duration
}

func NewTileUseCase(
	boundaryRepo repository.BoundaryRepository,
	transportRepo repository.TransportRepository,
	environmentRepo repository.EnvironmentRepository,
	poiRepo repository.POIRepository,
	cacheRepo repository.CacheRepository,
	logger *zap.Logger,
	tileCacheTTL time.Duration,
) *TileUseCase {
	return &TileUseCase{
		boundaryRepo:    boundaryRepo,
		transportRepo:   transportRepo,
		environmentRepo: environmentRepo,
		poiRepo:         poiRepo,
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

// GetWaterTile возвращает MVT тайл с водными объектами
func (uc *TileUseCase) GetWaterTile(ctx context.Context, z, x, y int) ([]byte, error) {
	cacheKey := fmt.Sprintf("tile:water:%d:%d:%d", z, x, y)
	cached, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		return cached, nil
	}

	tile, err := uc.environmentRepo.GetWaterTile(ctx, z, x, y)
	if err != nil {
		uc.logger.Error("Failed to get water tile", zap.Error(err))
		return nil, err
	}

	_ = uc.cacheRepo.Set(ctx, cacheKey, tile, uc.tileCacheTTL)
	return tile, nil
}

// GetBeachesTile возвращает MVT тайл с пляжами
func (uc *TileUseCase) GetBeachesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	cacheKey := fmt.Sprintf("tile:beaches:%d:%d:%d", z, x, y)
	cached, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		return cached, nil
	}

	tile, err := uc.environmentRepo.GetBeachesTile(ctx, z, x, y)
	if err != nil {
		uc.logger.Error("Failed to get beaches tile", zap.Error(err))
		return nil, err
	}

	_ = uc.cacheRepo.Set(ctx, cacheKey, tile, uc.tileCacheTTL)
	return tile, nil
}

// GetNoiseSourcesTile возвращает MVT тайл с источниками шума
func (uc *TileUseCase) GetNoiseSourcesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	cacheKey := fmt.Sprintf("tile:noise:%d:%d:%d", z, x, y)
	cached, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		return cached, nil
	}

	tile, err := uc.environmentRepo.GetNoiseSourcesTile(ctx, z, x, y)
	if err != nil {
		uc.logger.Error("Failed to get noise sources tile", zap.Error(err))
		return nil, err
	}

	_ = uc.cacheRepo.Set(ctx, cacheKey, tile, uc.tileCacheTTL)
	return tile, nil
}

// GetTouristZonesTile возвращает MVT тайл с туристическими зонами
func (uc *TileUseCase) GetTouristZonesTile(ctx context.Context, z, x, y int) ([]byte, error) {
	cacheKey := fmt.Sprintf("tile:tourist:%d:%d:%d", z, x, y)
	cached, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		return cached, nil
	}

	tile, err := uc.environmentRepo.GetTouristZonesTile(ctx, z, x, y)
	if err != nil {
		uc.logger.Error("Failed to get tourist zones tile", zap.Error(err))
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

// GetRadiusTiles возвращает MVT тайл со всеми типами данных в радиусе от точки
func (uc *TileUseCase) GetRadiusTiles(ctx context.Context, req dto.RadiusTilesRequest) ([]byte, error) {
	// Устанавливаем слои по умолчанию если не указаны
	layers := req.Layers
	if len(layers) == 0 {
		layers = []string{"boundaries", "transport", "pois", "green"}
	}

	// Создаем cache key с округленными координатами для лучшего cache hit rate
	layersHash := fmt.Sprintf("%x", md5.Sum([]byte(fmt.Sprintf("%v", layers))))
	cacheKey := fmt.Sprintf("radius-tiles:%.4f:%.4f:%.2f:%s",
		req.Lat, req.Lon, req.RadiusKm, layersHash)

	// Проверяем кеш
	cached, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != nil {
		return cached, nil
	}

	// Определяем какие слои загружать
	layerMap := make(map[string]bool)
	for _, layer := range layers {
		layerMap[layer] = true
	}

	// Параллельная загрузка MVT тайлов из PostgreSQL
	var (
		boundariesTile  []byte
		transportTile   []byte
		poisTile        []byte
		environmentTile []byte
	)

	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []error

	// Boundaries tile
	if layerMap["boundaries"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tile, err := uc.boundaryRepo.GetBoundariesRadiusTile(ctx, req.Lat, req.Lon, req.RadiusKm)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				uc.logger.Error("Failed to load boundaries tile", zap.Error(err))
				return
			}
			mu.Lock()
			boundariesTile = tile
			mu.Unlock()
		}()
	}

	// Transport tile
	if layerMap["transport"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tile, err := uc.transportRepo.GetTransportRadiusTile(ctx, req.Lat, req.Lon, req.RadiusKm)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				uc.logger.Error("Failed to load transport tile", zap.Error(err))
				return
			}
			mu.Lock()
			transportTile = tile
			mu.Unlock()
		}()
	}

	// POIs tile
	if layerMap["pois"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tile, err := uc.poiRepo.GetPOIRadiusTile(ctx, req.Lat, req.Lon, req.RadiusKm, nil)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				uc.logger.Error("Failed to load POI tile", zap.Error(err))
				return
			}
			mu.Lock()
			poisTile = tile
			mu.Unlock()
		}()
	}

	// Environment tile (green spaces, beaches, etc.)
	if layerMap["green"] || layerMap["water"] || layerMap["beaches"] {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tile, err := uc.environmentRepo.GetEnvironmentRadiusTile(ctx, req.Lat, req.Lon, req.RadiusKm)
			if err != nil {
				mu.Lock()
				errors = append(errors, err)
				mu.Unlock()
				uc.logger.Error("Failed to load environment tile", zap.Error(err))
				return
			}
			mu.Lock()
			environmentTile = tile
			mu.Unlock()
		}()
	}

	// Ждем завершения всех goroutines
	wg.Wait()

	// Проверяем ошибки
	if len(errors) > 0 {
		return nil, errors[0]
	}

	// Объединяем все MVT тайлы
	// MVT тайлы можно просто конкатенировать, так как каждый содержит свои слои
	var result bytes.Buffer

	if len(boundariesTile) > 0 {
		result.Write(boundariesTile)
	}
	if len(transportTile) > 0 {
		result.Write(transportTile)
	}
	if len(poisTile) > 0 {
		result.Write(poisTile)
	}
	if len(environmentTile) > 0 {
		result.Write(environmentTile)
	}

	combined := result.Bytes()

	// Кешируем результат на 1 час
	if err := uc.cacheRepo.Set(ctx, cacheKey, combined, time.Hour); err != nil {
		uc.logger.Warn("Failed to cache radius tiles", zap.String("key", cacheKey), zap.Error(err))
	}

	return combined, nil
}
