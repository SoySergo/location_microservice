package http

import (
	"context"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/compress"
	"github.com/location-microservice/internal/config"
	"github.com/location-microservice/internal/delivery/http/handler"
	"github.com/location-microservice/internal/delivery/http/middleware"
	fiberSwagger "github.com/swaggo/fiber-swagger"
	"go.uber.org/zap"
)

// Server - HTTP сервер на основе Fiber
type Server struct {
	app    *fiber.App
	config *config.Config
	logger *zap.Logger

	// Handlers
	searchHandler          *handler.SearchHandler
	transportHandler       *handler.TransportHandler
	poiHandler             *handler.POIHandler
	tileHandler            *handler.TileHandler
	poiTileHandler         *handler.POITileHandler
	statsHandler           *handler.StatsHandler
	enrichmentDebugHandler *handler.EnrichmentDebugHandler
	apiExplorerHandler     *handler.APIExplorerHandler
}

// NewServer - создание нового HTTP сервера
func NewServer(
	cfg *config.Config,
	logger *zap.Logger,
	searchHandler *handler.SearchHandler,
	transportHandler *handler.TransportHandler,
	poiHandler *handler.POIHandler,
	tileHandler *handler.TileHandler,
	poiTileHandler *handler.POITileHandler,
	statsHandler *handler.StatsHandler,
	enrichmentDebugHandler *handler.EnrichmentDebugHandler,
) *Server {
	app := fiber.New(fiber.Config{
		AppName:      "Location Microservice",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
		ErrorHandler: customErrorHandler(logger),
	})

	// Создаём API Explorer handler
	apiExplorerHandler, err := handler.NewAPIExplorerHandler()
	if err != nil {
		logger.Warn("Failed to initialize API Explorer handler, falling back to static file", zap.Error(err))
	}

	s := &Server{
		app:                    app,
		config:                 cfg,
		logger:                 logger,
		searchHandler:          searchHandler,
		transportHandler:       transportHandler,
		poiHandler:             poiHandler,
		tileHandler:            tileHandler,
		poiTileHandler:         poiTileHandler,
		statsHandler:           statsHandler,
		enrichmentDebugHandler: enrichmentDebugHandler,
		apiExplorerHandler:     apiExplorerHandler,
	}

	s.setupMiddlewares()
	s.setupRoutes()

	return s
}

// setupMiddlewares - настройка middleware
func (s *Server) setupMiddlewares() {
	s.app.Use(middleware.Recovery())
	s.app.Use(middleware.Logger(s.logger))
	s.app.Use(middleware.CORS())
	s.app.Use(compress.New(compress.Config{
		Level: compress.LevelBestSpeed,
	}))
}

// setupRoutes - настройка маршрутов
func (s *Server) setupRoutes() {
	// Swagger documentation route
	s.app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// Static files for debug UI
	s.app.Static("/static", "./static")

	// Debug map page redirect
	s.app.Get("/debug/map", func(c *fiber.Ctx) error {
		return c.Redirect("/static/transport-map.html")
	})

	// API Explorer page - рендеринг из шаблонов или fallback на статический файл
	s.app.Get("/debug/explorer", func(c *fiber.Ctx) error {
		if s.apiExplorerHandler != nil {
			return s.apiExplorerHandler.RenderExplorer(c)
		}
		// Fallback на статический файл
		return c.Redirect("/static/api-explorer.html")
	})

	api := s.app.Group("/api/v1")

	// Health check
	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status": "healthy",
			"time":   time.Now(),
		})
	})

	// Search routes
	api.Get("/search", s.searchHandler.Search)
	api.Post("/reverse-geocode", s.searchHandler.ReverseGeocode)
	api.Post("/batch/reverse-geocode", s.searchHandler.BatchReverseGeocode)

	// Boundary routes
	api.Get("/boundaries/:id", s.searchHandler.GetBoundaryByID)
	api.Get("/boundaries/tiles/:z/:x/:y.pbf", s.tileHandler.GetBoundaryTile)

	// Transport routes
	api.Post("/transport/nearest", s.transportHandler.GetNearestStations)
	api.Get("/transport/tiles/:z/:x/:y.pbf", s.tileHandler.GetTransportTile)
	api.Post("/batch/transport/nearest", s.transportHandler.BatchGetNearestStations)
	api.Get("/transport/lines/:id.pbf", s.tileHandler.GetTransportLineTile)
	api.Post("/transport/lines.pbf", s.tileHandler.GetTransportLinesTile)
	api.Get("/transport/station/:station_id/lines", s.transportHandler.GetLinesByStationID)

	// POI routes
	api.Post("/radius/poi", s.poiHandler.SearchByRadius)
	api.Get("/poi/categories", s.poiHandler.GetCategories)
	api.Get("/poi/categories/:id/subcategories", s.poiHandler.GetSubcategories)

	// POI Tile routes - новые эндпоинты
	api.Get("/tiles/poi/:z/:x/:y.pbf", s.poiTileHandler.GetPOITile)

	// Transport Tile routes - новые эндпоинты с фильтрацией
	api.Get("/tiles/transport/:z/:x/:y.pbf", s.transportHandler.GetTransportTileByTypes)

	// Environment tiles
	api.Get("/green-spaces/tiles/:z/:x/:y.pbf", s.tileHandler.GetGreenSpacesTile)
	api.Get("/water/tiles/:z/:x/:y.pbf", s.tileHandler.GetWaterTile)
	api.Get("/beaches/tiles/:z/:x/:y.pbf", s.tileHandler.GetBeachesTile)
	api.Get("/noise-sources/tiles/:z/:x/:y.pbf", s.tileHandler.GetNoiseSourcesTile)
	api.Get("/tourist-zones/tiles/:z/:x/:y.pbf", s.tileHandler.GetTouristZonesTile)

	// Radius tiles - комплексный endpoint для получения всех данных в радиусе
	api.Post("/radius/tiles.pbf", s.tileHandler.GetRadiusTiles)

	// Debug/Test routes - для тестирования логики обогащения
	debug := api.Group("/debug")
	debug.Post("/enrichment/transport", s.enrichmentDebugHandler.GetNearestTransportEnriched)
	debug.Get("/enrichment/transport", s.enrichmentDebugHandler.GetNearestTransportEnrichedGET)
	debug.Post("/enrichment/transport/batch", s.enrichmentDebugHandler.GetNearestTransportEnrichedBatch)
	debug.Post("/enrichment/location", s.enrichmentDebugHandler.EnrichLocation)
	debug.Get("/enrichment/location", s.enrichmentDebugHandler.EnrichLocationGET)
	debug.Post("/enrichment/location/batch", s.enrichmentDebugHandler.EnrichLocationBatch)

	// Priority Transport routes - транспорт с приоритетом metro/train -> bus/tram
	debug.Get("/transport/priority", s.enrichmentDebugHandler.GetPriorityTransport)
	debug.Post("/transport/priority", s.enrichmentDebugHandler.GetPriorityTransportPOST)
	debug.Post("/transport/priority/batch", s.enrichmentDebugHandler.GetPriorityTransportBatch)

	// Mapbox config endpoint for debug map UI
	debug.Get("/config/mapbox", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"token": s.config.Mapbox.AccessToken,
		})
	})

	// Stats
	api.Get("/stats", s.statsHandler.GetStatistics)
}

// Start - запуск HTTP сервера
func (s *Server) Start() error {
	addr := s.config.GetServerAddr()
	s.logger.Info("Starting HTTP server", zap.String("address", addr))
	return s.app.Listen(addr)
}

// Shutdown - graceful shutdown HTTP сервера
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("Shutting down HTTP server")
	return s.app.ShutdownWithContext(ctx)
}

// customErrorHandler - кастомный обработчик ошибок
func customErrorHandler(logger *zap.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		logger.Error("HTTP Error",
			zap.String("path", c.Path()),
			zap.Int("status", code),
			zap.Error(err),
		)

		return c.Status(code).JSON(fiber.Map{
			"error": fiber.Map{
				"code":    "INTERNAL_SERVER_ERROR",
				"message": err.Error(),
			},
		})
	}
}
