package main

// @title Location Microservice API
// @version 1.0.0
// @description Микросервис для работы с геопространственными данными из OpenStreetMap. Предоставляет API для поиска административных границ, транспортной инфраструктуры, точек интереса (POI), а также векторных тайлов для визуализации на картах.
// @description
// @description Основные возможности:
// @description - Поиск и обратное геокодирование административных границ
// @description - Поиск ближайших транспортных станций (метро, автобусы, трамваи)
// @description - Поиск точек интереса по категориям в радиусе
// @description - Получение векторных тайлов (MVT/PBF) для всех типов данных
// @description - Статистика по загруженным данным

// @contact.name API Support
// @contact.email support@location-microservice.com

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /
// @schemes http https

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/location-microservice/docs/swagger"
	"github.com/location-microservice/internal/config"
	httpDelivery "github.com/location-microservice/internal/delivery/http"
	"github.com/location-microservice/internal/delivery/http/handler"
	"github.com/location-microservice/internal/pkg/logger"
	"github.com/location-microservice/internal/repository/cache"
	"github.com/location-microservice/internal/repository/postgresosm"
	"github.com/location-microservice/internal/usecase"
	"go.uber.org/zap"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// 2. Initialize logger
	log, err := logger.New(cfg.Log.Level)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer log.Sync()

	log.Info("Starting Location Microservice")
	log.Info("Configuration loaded",
		zap.String("env", cfg.Server.Env),
		zap.String("server_addr", cfg.GetServerAddr()),
	)

	// 3b. Connect to OSM PostgreSQL (osm_db with planet_osm_* tables)
	osmDB, err := postgresosm.New(&cfg.OSMDB, log)
	if err != nil {
		log.Fatal("Failed to connect to OSM PostgreSQL", zap.Error(err))
	}
	defer func() {
		if err := osmDB.Close(); err != nil {
			log.Error("Failed to close OSM PostgreSQL connection", zap.Error(err))
		}
	}()
	log.Info("OSM PostgreSQL connected")

	// 4. Connect to Redis
	redisClient, err := cache.NewRedis(&cfg.Redis, log)
	if err != nil {
		log.Fatal("Failed to connect to Redis", zap.Error(err))
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			log.Error("Failed to close Redis connection", zap.Error(err))
		}
	}()
	log.Info("Redis connected")

	// 5. Health checks
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := osmDB.Health(ctx); err != nil {
		log.Fatal("OSM PostgreSQL health check failed", zap.Error(err))
	}

	if err := redisClient.Health(ctx); err != nil {
		log.Fatal("Redis health check failed", zap.Error(err))
	}

	log.Info("All connections healthy")

	// 6. Initialize Repositories
	// OSM репозитории (работают с planet_osm_* таблицами из OSM базы)
	boundaryRepo := postgresosm.NewBoundaryRepository(osmDB)
	transportRepo := postgresosm.NewTransportRepository(osmDB)
	poiRepo := postgresosm.NewPOIRepository(osmDB)
	environmentRepo := postgresosm.NewEnvironmentRepository(osmDB)

	// Postgres репозитории (основная база данных для статистики и других данных)
	cacheRepo := cache.NewCacheRepository(redisClient)

	log.Info("Repositories initialized")

	// 7. Initialize Use Cases
	searchUC := usecase.NewSearchUseCase(
		boundaryRepo,
		cacheRepo,
		log,
		cfg.Cache.SearchCacheTTL,
	)

	transportUC := usecase.NewTransportUseCase(
		transportRepo,
		log,
	)

	poiUC := usecase.NewPOIUseCase(
		poiRepo,
		log,
	)

	tileUC := usecase.NewTileUseCase(
		boundaryRepo,
		transportRepo,
		environmentRepo,
		poiRepo,
		cacheRepo,
		log,
		cfg.Cache.TilesCacheTTL,
	)

	poiTileUC := usecase.NewPOITileUseCase(
		poiRepo,
		cacheRepo,
		log,
		cfg.Cache.POITileCacheTTL,
		cfg.Tile.POIMaxFeatures,
	)

	// TODO: statsRepo not implemented yet, using nil for now
	statsUC := usecase.NewStatsUseCase(
		nil, // statsRepo
		cacheRepo,
		log,
	)

	// EnrichedLocationUseCase - для полного обогащения локаций
	enrichedLocationUC := usecase.NewEnrichedLocationUseCase(searchUC, transportUC, log)

	log.Info("Use cases initialized")

	// 8. Initialize HTTP Handlers
	searchHandler := handler.NewSearchHandler(searchUC, log)
	transportHandler := handler.NewTransportHandler(transportUC, log)
	poiHandler := handler.NewPOIHandler(poiUC, log)
	tileHandler := handler.NewTileHandler(tileUC, log)
	poiTileHandler := handler.NewPOITileHandler(poiTileUC, log)
	statsHandler := handler.NewStatsHandler(statsUC, log)
	enrichedLocationHandler := handler.NewEnrichedLocationHandler(enrichedLocationUC, log)
	viewportHandler := handler.NewViewportHandler(transportRepo, poiRepo, log)

	log.Info("HTTP handlers initialized")

	// 9. Initialize HTTP Server
	server := httpDelivery.NewServer(
		cfg,
		log,
		searchHandler,
		transportHandler,
		poiHandler,
		tileHandler,
		poiTileHandler,
		statsHandler,
		enrichedLocationHandler,
		viewportHandler,
	)

	log.Info("HTTP server initialized")

	// 10. Start server in goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	log.Info("Server started successfully",
		zap.String("address", cfg.GetServerAddr()),
		zap.String("env", cfg.Server.Env),
	)

	// 11. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down server gracefully...")

	ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP server
	if err := server.Shutdown(ctx); err != nil {
		log.Error("Server shutdown error", zap.Error(err))
	}

	// Close OSM database connection
	if err := osmDB.Close(); err != nil {
		log.Error("Failed to close OSM database", zap.Error(err))
	}

	// Close Redis connection
	if err := redisClient.Close(); err != nil {
		log.Error("Failed to close Redis", zap.Error(err))
	}

	log.Info("Server stopped successfully")
}
