package mapbox

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/location-microservice/internal/config"
	"github.com/location-microservice/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestClient_GetWalkingMatrix(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	t.Run("successful request", func(t *testing.T) {
		// Mock server
		mockResp := domain.MatrixResponse{
			Code: "Ok",
			Distances: [][]float64{
				{100.0, 200.0, 300.0},
			},
			Durations: [][]float64{
				{60.0, 120.0, 180.0},
			},
			Sources: []domain.Location{
				{Name: "source", Location: []float64{2.1734, 41.3851}},
			},
			Destinations: []domain.Location{
				{Name: "dest1", Location: []float64{2.1800, 41.3900}},
				{Name: "dest2", Location: []float64{2.1850, 41.3950}},
				{Name: "dest3", Location: []float64{2.1900, 41.4000}},
			},
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockResp)
		}))
		defer server.Close()

		cfg := &config.MapboxConfig{
			AccessToken:     "test_token",
			BaseURL:         server.URL,
			MaxMatrixPoints: 25,
			WalkingProfile:  "mapbox/walking",
			RequestTimeout:  30,
		}

		client := NewMapboxClient(cfg, logger)

		origins := []domain.Coordinate{
			{Lat: 41.3851, Lon: 2.1734},
		}
		destinations := []domain.Coordinate{
			{Lat: 41.3900, Lon: 2.1800},
			{Lat: 41.3950, Lon: 2.1850},
			{Lat: 41.4000, Lon: 2.1900},
		}

		result, err := client.GetWalkingMatrix(context.Background(), origins, destinations)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Ok", result.Code)
		assert.Len(t, result.Distances, 1)
		assert.Len(t, result.Distances[0], 3)
		assert.Equal(t, 100.0, result.Distances[0][0])
		assert.Equal(t, 200.0, result.Distances[0][1])
		assert.Equal(t, 300.0, result.Distances[0][2])
	})

	t.Run("empty origins", func(t *testing.T) {
		cfg := &config.MapboxConfig{
			AccessToken:     "test_token",
			BaseURL:         "https://api.mapbox.com",
			MaxMatrixPoints: 25,
			WalkingProfile:  "mapbox/walking",
			RequestTimeout:  30,
		}

		client := NewMapboxClient(cfg, logger)

		destinations := []domain.Coordinate{
			{Lat: 41.3900, Lon: 2.1800},
		}

		result, err := client.GetWalkingMatrix(context.Background(), []domain.Coordinate{}, destinations)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("empty destinations", func(t *testing.T) {
		cfg := &config.MapboxConfig{
			AccessToken:     "test_token",
			BaseURL:         "https://api.mapbox.com",
			MaxMatrixPoints: 25,
			WalkingProfile:  "mapbox/walking",
			RequestTimeout:  30,
		}

		client := NewMapboxClient(cfg, logger)

		origins := []domain.Coordinate{
			{Lat: 41.3851, Lon: 2.1734},
		}

		result, err := client.GetWalkingMatrix(context.Background(), origins, []domain.Coordinate{})
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "cannot be empty")
	})

	t.Run("exceeds mapbox limit", func(t *testing.T) {
		cfg := &config.MapboxConfig{
			AccessToken:     "test_token",
			BaseURL:         "https://api.mapbox.com",
			MaxMatrixPoints: 25,
			WalkingProfile:  "mapbox/walking",
			RequestTimeout:  30,
		}

		client := NewMapboxClient(cfg, logger)

		origins := []domain.Coordinate{
			{Lat: 41.3851, Lon: 2.1734},
		}

		// Create 25 destinations (1 origin + 25 destinations = 26 > 25 limit)
		destinations := make([]domain.Coordinate, 25)
		for i := 0; i < 25; i++ {
			destinations[i] = domain.Coordinate{Lat: 41.3900, Lon: 2.1800}
		}

		result, err := client.GetWalkingMatrix(context.Background(), origins, destinations)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "exceed Mapbox limit")
	})

	t.Run("api error response", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"code":"InvalidInput","message":"Invalid coordinates"}`))
		}))
		defer server.Close()

		cfg := &config.MapboxConfig{
			AccessToken:     "test_token",
			BaseURL:         server.URL,
			MaxMatrixPoints: 25,
			WalkingProfile:  "mapbox/walking",
			RequestTimeout:  30,
		}

		client := NewMapboxClient(cfg, logger)

		origins := []domain.Coordinate{
			{Lat: 41.3851, Lon: 2.1734},
		}
		destinations := []domain.Coordinate{
			{Lat: 41.3900, Lon: 2.1800},
		}

		result, err := client.GetWalkingMatrix(context.Background(), origins, destinations)
		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "mapbox API error")
	})
}
