package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/location-microservice/internal/config"
	"github.com/location-microservice/internal/pkg/logger"
	"github.com/location-microservice/internal/repository/cache"
	"github.com/location-microservice/internal/repository/postgresosm"
	redisRepo "github.com/location-microservice/internal/repository/redis"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/worker"
	"github.com/location-microservice/internal/worker/location"
	"go.uber.org/zap"
)

func main() {
	// 1. Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load config: %v", err))
	}

	// Check if worker is enabled
	if !cfg.Worker.Enabled {
		fmt.Println("Worker is disabled in configuration. Set WORKER_ENABLED=true to enable.")
		os.Exit(0)
	}

	// 2. Initialize logger
	log, err := logger.New(cfg.Log.Level)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer log.Sync()

	log.Info("Starting Location Enrichment Worker")
	log.Info("Configuration loaded",
		zap.String("consumer_group", cfg.Worker.ConsumerGroup),
		zap.Int("max_retries", cfg.Worker.MaxRetries))

	// 3. Connect to OSM PostgreSQL (planet_osm_* tables)
	osmDB, err := postgresosm.New(&cfg.OSMDB, log)
	if err != nil {
		log.Fatal("Failed to connect to OSM PostgreSQL", zap.Error(err))
	}
	defer func() {
		if err := osmDB.Close(); err != nil {
			log.Error("Failed to close OSM PostgreSQL connection", zap.Error(err))
		}
	}()

	// Health check for OSM database
	healthCtx, healthCancel := context.WithTimeout(context.Background(), 5*time.Second)
	if err := osmDB.Health(healthCtx); err != nil {
		log.Fatal("OSM PostgreSQL health check failed", zap.Error(err))
	}
	healthCancel()
	log.Info("OSM PostgreSQL connected and healthy")

	// 4. Connect to Redis (cache - local)
	cacheRedis, err := cache.NewRedis(&cfg.Redis, log)
	if err != nil {
		log.Fatal("Failed to connect to cache Redis", zap.Error(err))
	}
	defer func() {
		if err := cacheRedis.Close(); err != nil {
			log.Error("Failed to close cache Redis connection", zap.Error(err))
		}
	}()

	// 5. Connect to Redis Streams (shared with backend_estate)
	streamsRedis, err := cache.NewRedisStreams(&cfg.RedisStreams, log)
	if err != nil {
		log.Fatal("Failed to connect to streams Redis", zap.Error(err))
	}
	defer func() {
		if err := streamsRedis.Close(); err != nil {
			log.Error("Failed to close streams Redis connection", zap.Error(err))
		}
	}()

	// 6. Initialize repositories (using OSM database)
	boundaryRepo := postgresosm.NewBoundaryRepository(osmDB)
	transportRepo := postgresosm.NewTransportRepository(osmDB)
	streamRepo := redisRepo.NewStreamRepository(streamsRedis, log)
	cacheRepo := cache.NewCacheRepository(cacheRedis)

	// 7. Initialize use cases
	searchUC := usecase.NewSearchUseCase(boundaryRepo, cacheRepo, log, cfg.Cache.SearchCacheTTL)
	transportUC := usecase.NewTransportUseCase(transportRepo, log)
	enrichedLocationUC := usecase.NewEnrichedLocationUseCase(searchUC, transportUC, log)

	// 8. Initialize worker
	locationWorker := location.NewLocationEnrichmentWorker(
		streamRepo,
		enrichedLocationUC,
		cfg.Worker.ConsumerGroup,
		cfg.Worker.MaxRetries,
		log,
	)

	// 9. Create worker manager and register workers
	workerManager := worker.NewWorkerManager(log)
	workerManager.Register(locationWorker)

	// 10. Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start workers
	if err := workerManager.Start(ctx); err != nil {
		log.Fatal("Failed to start workers", zap.Error(err))
	}

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	log.Info("Received shutdown signal")

	// Cancel context to stop workers
	cancel()

	// Stop worker manager
	if err := workerManager.Stop(); err != nil {
		log.Error("Error stopping workers", zap.Error(err))
	}

	log.Info("Worker shutdown complete")
}
