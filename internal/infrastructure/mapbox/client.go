package mapbox

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/location-microservice/internal/config"
	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/domain/repository"
	"go.uber.org/zap"
)

type client struct {
	httpClient  *http.Client
	baseURL     string
	accessToken string
	profile     string
	logger      *zap.Logger
}

// NewMapboxClient создает новый клиент для Mapbox API
func NewMapboxClient(cfg *config.MapboxConfig, logger *zap.Logger) repository.MapboxRepository {
	return &client{
		httpClient: &http.Client{
			Timeout: time.Duration(cfg.RequestTimeout) * time.Second,
		},
		baseURL:     cfg.BaseURL,
		accessToken: cfg.AccessToken,
		profile:     cfg.WalkingProfile,
		logger:      logger,
	}
}

// GetWalkingMatrix возвращает матрицу пешеходных расстояний и времени
func (c *client) GetWalkingMatrix(
	ctx context.Context,
	origins []domain.Coordinate,
	destinations []domain.Coordinate,
) (*domain.MatrixResponse, error) {
	if len(origins) == 0 || len(destinations) == 0 {
		return nil, fmt.Errorf("origins and destinations cannot be empty")
	}

	// Проверка лимита Mapbox (25 точек максимум)
	if len(origins)+len(destinations) > 25 {
		return nil, fmt.Errorf("total coordinates exceed Mapbox limit of 25 points")
	}

	// Формируем список координат: сначала origins, потом destinations
	var coordinates []string
	for _, coord := range origins {
		coordinates = append(coordinates, fmt.Sprintf("%f,%f", coord.Lon, coord.Lat))
	}
	for _, coord := range destinations {
		coordinates = append(coordinates, fmt.Sprintf("%f,%f", coord.Lon, coord.Lat))
	}

	coordinatesStr := strings.Join(coordinates, ";")

	// Формируем индексы источников и назначений
	sourcesIndices := make([]string, len(origins))
	for i := range origins {
		sourcesIndices[i] = fmt.Sprintf("%d", i)
	}
	destinationsIndices := make([]string, len(destinations))
	for i := range destinations {
		destinationsIndices[i] = fmt.Sprintf("%d", i+len(origins))
	}

	// Строим URL
	url := fmt.Sprintf("%s/directions-matrix/v1/%s/%s?sources=%s&destinations=%s&access_token=%s",
		c.baseURL,
		c.profile,
		coordinatesStr,
		strings.Join(sourcesIndices, ";"),
		strings.Join(destinationsIndices, ";"),
		c.accessToken,
	)

	c.logger.Debug("Calling Mapbox Matrix API",
		zap.String("url", url),
		zap.Int("origins_count", len(origins)),
		zap.Int("destinations_count", len(destinations)))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		c.logger.Error("Failed to create request", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("Failed to execute request", zap.Error(err))
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("Mapbox API returned error",
			zap.Int("status_code", resp.StatusCode),
			zap.String("body", string(body)))
		return nil, fmt.Errorf("mapbox API error: status %d, body: %s", resp.StatusCode, string(body))
	}

	var matrixResp domain.MatrixResponse
	if err := json.NewDecoder(resp.Body).Decode(&matrixResp); err != nil {
		c.logger.Error("Failed to decode response", zap.Error(err))
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if matrixResp.Code != "Ok" {
		c.logger.Error("Mapbox API returned non-OK code",
			zap.String("code", matrixResp.Code))
		return nil, fmt.Errorf("mapbox API returned code: %s", matrixResp.Code)
	}

	c.logger.Debug("Mapbox Matrix API call successful",
		zap.Int("distances_rows", len(matrixResp.Distances)),
		zap.Int("durations_rows", len(matrixResp.Durations)))

	return &matrixResp, nil
}
