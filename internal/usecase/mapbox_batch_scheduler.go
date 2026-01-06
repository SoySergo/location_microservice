package usecase

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"go.uber.org/zap"
)

// MapboxBatchRequest represents a single property enrichment request
type MapboxBatchRequest struct {
	PropertyID uuid.UUID
	Lat        float64
	Lon        float64
	Stations   []*domain.TransportStation
	ResultChan chan *MapboxBatchResult
}

// MapboxBatchResult represents the result for a single property
type MapboxBatchResult struct {
	PropertyID uuid.UUID
	Transport  []domain.TransportWithDistance
	Error      error
}

// MapboxBatchScheduler handles batching of Mapbox requests
type MapboxBatchScheduler struct {
	mapboxRepo    repository.MapboxRepository
	logger        *zap.Logger
	batchSize     int
	batchInterval time.Duration
	requestQueue  chan *MapboxBatchRequest
	stopChan      chan struct{}
	wg            sync.WaitGroup
}

// NewMapboxBatchScheduler creates a new batch scheduler
func NewMapboxBatchScheduler(
	mapboxRepo repository.MapboxRepository,
	logger *zap.Logger,
	batchSize int,
	batchInterval time.Duration,
) *MapboxBatchScheduler {
	return &MapboxBatchScheduler{
		mapboxRepo:    mapboxRepo,
		logger:        logger,
		batchSize:     batchSize,
		batchInterval: batchInterval,
		requestQueue:  make(chan *MapboxBatchRequest, 100),
		stopChan:      make(chan struct{}),
	}
}

// Start starts the batch scheduler
func (s *MapboxBatchScheduler) Start(ctx context.Context) {
	s.wg.Add(1)
	go s.processBatches(ctx)
}

// Stop stops the batch scheduler
func (s *MapboxBatchScheduler) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// ScheduleRequest adds a request to the batch queue
func (s *MapboxBatchScheduler) ScheduleRequest(req *MapboxBatchRequest) {
	select {
	case s.requestQueue <- req:
	case <-s.stopChan:
		req.ResultChan <- &MapboxBatchResult{
			PropertyID: req.PropertyID,
			Error:      fmt.Errorf("scheduler stopped"),
		}
	}
}

// processBatches processes batched requests
func (s *MapboxBatchScheduler) processBatches(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.batchInterval)
	defer ticker.Stop()

	var batch []*MapboxBatchRequest

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Batch scheduler context cancelled")
			return

		case <-s.stopChan:
			s.logger.Info("Batch scheduler stopped")
			return

		case req := <-s.requestQueue:
			batch = append(batch, req)

			// Process batch if we've reached the batch size
			if len(batch) >= s.batchSize {
				s.processBatch(ctx, batch)
				batch = nil
			}

		case <-ticker.C:
			// Process accumulated batch on timer
			if len(batch) > 0 {
				s.processBatch(ctx, batch)
				batch = nil
			}
		}
	}
}

// processBatch processes a batch of requests
func (s *MapboxBatchScheduler) processBatch(ctx context.Context, batch []*MapboxBatchRequest) {
	s.logger.Debug("Processing Mapbox batch",
		zap.Int("batch_size", len(batch)))

	// Calculate total coordinates needed
	totalCoords := 0
	for _, req := range batch {
		totalCoords += len(req.Stations)
	}

	// If we exceed Mapbox limit (25 total), split into multiple requests
	maxCoords := 25 // 1 origin per property + destinations

	if totalCoords+len(batch) > maxCoords {
		// Process in chunks
		s.processBatchInChunks(ctx, batch, maxCoords)
		return
	}

	// Build origins and destinations
	var origins []domain.Coordinate
	var destinations []domain.Coordinate
	var requestMapping []struct {
		PropertyID     uuid.UUID
		OriginIndex    int
		StationIndices []int
		Stations       []*domain.TransportStation
		ResultChan     chan *MapboxBatchResult
	}

	destIndex := 0
	for _, req := range batch {
		originIdx := len(origins)
		origins = append(origins, domain.Coordinate{
			Lat: req.Lat,
			Lon: req.Lon,
		})

		stationIndices := make([]int, len(req.Stations))
		for i, station := range req.Stations {
			destinations = append(destinations, domain.Coordinate{
				Lat: station.Lat,
				Lon: station.Lon,
			})
			stationIndices[i] = destIndex
			destIndex++
		}

		requestMapping = append(requestMapping, struct {
			PropertyID     uuid.UUID
			OriginIndex    int
			StationIndices []int
			Stations       []*domain.TransportStation
			ResultChan     chan *MapboxBatchResult
		}{
			PropertyID:     req.PropertyID,
			OriginIndex:    originIdx,
			StationIndices: stationIndices,
			Stations:       req.Stations,
			ResultChan:     req.ResultChan,
		})
	}

	// Call Mapbox Matrix API
	matrixResp, err := s.mapboxRepo.GetWalkingMatrix(ctx, origins, destinations)
	if err != nil {
		s.logger.Error("Failed to get walking matrix for batch", zap.Error(err))
		// Send error to all requests in batch
		for _, mapping := range requestMapping {
			mapping.ResultChan <- &MapboxBatchResult{
				PropertyID: mapping.PropertyID,
				Error:      err,
			}
		}
		return
	}

	// Distribute results back to each property
	for _, mapping := range requestMapping {
		transport := make([]domain.TransportWithDistance, 0, len(mapping.Stations))

		for i, station := range mapping.Stations {
			stationDestIdx := mapping.StationIndices[i]

			// Get distance and duration from matrix response
			var walkingDistance, walkingDuration *float64
			if mapping.OriginIndex < len(matrixResp.Distances) &&
				stationDestIdx < len(matrixResp.Distances[mapping.OriginIndex]) {

				dist := matrixResp.Distances[mapping.OriginIndex][stationDestIdx]
				dur := matrixResp.Durations[mapping.OriginIndex][stationDestIdx]
				walkingDistance = &dist
				walkingDuration = &dur
			}

			// Calculate linear distance
			linearDist := calculateDistance(
				origins[mapping.OriginIndex].Lat,
				origins[mapping.OriginIndex].Lon,
				station.Lat,
				station.Lon,
			)

			transport = append(transport, domain.TransportWithDistance{
				StationID:       station.ID,
				Name:            station.Name,
				Type:            station.Type,
				Lat:             station.Lat,
				Lon:             station.Lon,
				LineIDs:         station.LineIDs,
				LinearDistance:  linearDist,
				WalkingDistance: walkingDistance,
				WalkingDuration: walkingDuration,
			})
		}

		mapping.ResultChan <- &MapboxBatchResult{
			PropertyID: mapping.PropertyID,
			Transport:  transport,
			Error:      nil,
		}
	}
}

// processBatchInChunks splits a large batch into smaller chunks
func (s *MapboxBatchScheduler) processBatchInChunks(ctx context.Context, batch []*MapboxBatchRequest, maxCoords int) {
	s.logger.Warn("Batch exceeds Mapbox limit, processing in chunks",
		zap.Int("batch_size", len(batch)),
		zap.Int("max_coords", maxCoords))

	// Process each request individually as fallback
	for _, req := range batch {
		s.processBatch(ctx, []*MapboxBatchRequest{req})
	}
}

// calculateDistance calculates distance between two coordinates using Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0 // meters

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	lat1Rad := lat1 * (math.Pi / 180.0)
	lat2Rad := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}
