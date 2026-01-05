package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/location-microservice/internal/config"
	"github.com/location-microservice/internal/pkg/logger"
	"github.com/location-microservice/internal/repository/cache"
	"github.com/location-microservice/internal/repository/postgres"
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
		zap.Int("max_retries", cfg.Worker.MaxRetries),
		zap.Float64("transport_radius", cfg.Worker.TransportRadius),
		zap.Strings("transport_types", cfg.Worker.TransportTypes))

	// 3. Connect to PostgreSQL
	db, err := postgres.New(&cfg.Database, log)
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL", zap.Error(err))
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Error("Failed to close PostgreSQL connection", zap.Error(err))
		}
	}()

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

	// 5. Initialize repositories
	boundaryRepo := postgres.NewBoundaryRepository(db)
	transportRepo := postgres.NewTransportRepository(db)
	streamRepo := redisRepo.NewStreamRepository(redisClient.Client(), log)

	// 6. Initialize use cases
	enrichmentUC := usecase.NewEnrichmentUseCase(
		boundaryRepo,
		transportRepo,
		log,
		cfg.Worker.TransportTypes,
		cfg.Worker.TransportRadius,
	)

	// 7. Initialize workers
	locationWorker := location.NewLocationEnrichmentWorker(
		streamRepo,
		enrichmentUC,
		cfg.Worker.ConsumerGroup,
		cfg.Worker.MaxRetries,
		log,
	)

	// 8. Create worker manager and register workers
	workerManager := worker.NewWorkerManager(log)
	workerManager.Register(locationWorker)

	// 9. Setup graceful shutdown
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
