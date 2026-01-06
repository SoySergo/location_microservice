package usecase

import (
	"context"
	"crypto/md5"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/pkg/errors"
	"go.uber.org/zap"
)

type POITileUseCase struct {
	poiRepo      repository.POIRepository
	cacheRepo    repository.CacheRepository
	logger       *zap.Logger
	tileCacheTTL time.Duration
	maxFeatures  int
}

func NewPOITileUseCase(
	poiRepo repository.POIRepository,
	cacheRepo repository.CacheRepository,
	logger *zap.Logger,
	tileCacheTTL time.Duration,
	maxFeatures int,
) *POITileUseCase {
	if maxFeatures == 0 {
		maxFeatures = 1000 // Default max features
	}
	return &POITileUseCase{
		poiRepo:      poiRepo,
		cacheRepo:    cacheRepo,
		logger:       logger,
		tileCacheTTL: tileCacheTTL,
		maxFeatures:  maxFeatures,
	}
}

// GetPOITile возвращает MVT тайл с POI с фильтрацией по категориям и подкатегориям
func (uc *POITileUseCase) GetPOITile(ctx context.Context, z, x, y int, categories, subcategories []string) ([]byte, error) {
	// Валидация zoom level (consistent with existing tile endpoints)
	if z < 0 || z > 18 {
		return nil, errors.ErrInvalidZoom
	}

	// Валидация категорий
	if len(categories) > 0 {
		for _, cat := range categories {
			if !domain.IsValidPOICategory(cat) {
				return nil, errors.New("INVALID_POI_CATEGORY", fmt.Sprintf("invalid category: %s", cat), 400)
			}
		}
	}

	// Создаем cache key
	cacheKey := uc.createCacheKey(z, x, y, categories, subcategories)

	// Проверяем кеш
	cached, err := uc.cacheRepo.Get(ctx, cacheKey)
	if err == nil && cached != nil && len(cached) > 0 {
		uc.logger.Debug("POI tile cache hit", zap.String("key", cacheKey))
		return cached, nil
	}

	// Генерируем тайл из БД
	tile, err := uc.poiRepo.GetPOITileByCategories(ctx, z, x, y, categories, subcategories)
	if err != nil {
		uc.logger.Error("Failed to get POI tile",
			zap.Int("z", z),
			zap.Int("x", x),
			zap.Int("y", y),
			zap.Strings("categories", categories),
			zap.Strings("subcategories", subcategories),
			zap.Error(err))
		return nil, err
	}

	// Кешируем результат
	if err := uc.cacheRepo.Set(ctx, cacheKey, tile, uc.tileCacheTTL); err != nil {
		uc.logger.Warn("Failed to cache POI tile",
			zap.String("key", cacheKey),
			zap.Error(err))
	}

	return tile, nil
}

// createCacheKey создает ключ для кеширования с учетом параметров фильтрации
func (uc *POITileUseCase) createCacheKey(z, x, y int, categories, subcategories []string) string {
	// Сортируем массивы для стабильного хеша
	sortedCategories := make([]string, len(categories))
	copy(sortedCategories, categories)
	sort.Strings(sortedCategories)

	sortedSubcategories := make([]string, len(subcategories))
	copy(sortedSubcategories, subcategories)
	sort.Strings(sortedSubcategories)

	// Создаем строку параметров
	params := fmt.Sprintf("%s|%s",
		strings.Join(sortedCategories, ","),
		strings.Join(sortedSubcategories, ","))

	// Хешируем параметры
	hash := fmt.Sprintf("%x", md5.Sum([]byte(params)))

	return fmt.Sprintf("tile:poi:%d:%d:%d:%s", z, x, y, hash)
}
