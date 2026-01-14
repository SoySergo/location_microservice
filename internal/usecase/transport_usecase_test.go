package usecase_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/location-microservice/internal/domain"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
)

func TestTransportUseCase_GetNearestTransportByPriority(t *testing.T) {
	logger := zap.NewNop()
	mockTransportRepo := &MockTransportRepository{}
	ctx := context.Background()

	uc := usecase.NewTransportUseCase(mockTransportRepo, logger)

	t.Run("success with high priority stations", func(t *testing.T) {
		// Mock response with metro stations (high priority)
		stations := []domain.NearestTransportWithLines{
			{
				StationID: 100,
				Name:      "Catalunya",
				Type:      "metro",
				Lat:       41.3850,
				Lon:       2.1700,
				Distance:  250.5,
				Lines: []domain.TransportLineInfo{
					{
						ID:    1,
						Name:  "L1",
						Type:  "metro",
						Color: "#E32019",
					},
				},
			},
		}

		mockTransportRepo.On("GetNearestTransportByPriority", ctx, 41.3851, 2.1734, 1500.0, 5).
			Return(stations, nil)

		req := dto.PriorityTransportRequest{
			Lat:    41.3851,
			Lon:    2.1734,
			Radius: 1500,
			Limit:  5,
		}

		resp, err := uc.GetNearestTransportByPriority(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Stations, 1)
		assert.Equal(t, int64(100), resp.Stations[0].StationID)
		assert.Equal(t, "Catalunya", resp.Stations[0].Name)
		assert.Equal(t, "metro", resp.Stations[0].Type)
		assert.True(t, resp.Meta.HasHighPriority)
		assert.Equal(t, "metro/train", resp.Meta.PriorityType)

		mockTransportRepo.AssertExpectations(t)
	})

	t.Run("success with low priority stations only", func(t *testing.T) {
		mockTransportRepo2 := &MockTransportRepository{}
		uc2 := usecase.NewTransportUseCase(mockTransportRepo2, logger)

		// Mock response with bus stations (low priority)
		stations := []domain.NearestTransportWithLines{
			{
				StationID: 200,
				Name:      "Bus Stop 1",
				Type:      "bus",
				Lat:       41.3855,
				Lon:       2.1740,
				Distance:  150.0,
				Lines:     []domain.TransportLineInfo{},
			},
		}

		mockTransportRepo2.On("GetNearestTransportByPriority", ctx, 41.3851, 2.1734, 1500.0, 5).
			Return(stations, nil)

		req := dto.PriorityTransportRequest{
			Lat:    41.3851,
			Lon:    2.1734,
			Radius: 1500,
			Limit:  5,
		}

		resp, err := uc2.GetNearestTransportByPriority(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Stations, 1)
		assert.False(t, resp.Meta.HasHighPriority)
		assert.Equal(t, "bus/tram", resp.Meta.PriorityType)

		mockTransportRepo2.AssertExpectations(t)
	})

	t.Run("invalid coordinates", func(t *testing.T) {
		req := dto.PriorityTransportRequest{
			Lat:    999.0, // Invalid
			Lon:    2.1734,
			Radius: 1500,
			Limit:  5,
		}

		resp, err := uc.GetNearestTransportByPriority(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("uses default values when not provided", func(t *testing.T) {
		mockTransportRepo3 := &MockTransportRepository{}
		uc3 := usecase.NewTransportUseCase(mockTransportRepo3, logger)

		mockTransportRepo3.On("GetNearestTransportByPriority", ctx, 41.3851, 2.1734, 1500.0, 5).
			Return([]domain.NearestTransportWithLines{}, nil)

		req := dto.PriorityTransportRequest{
			Lat: 41.3851,
			Lon: 2.1734,
			// Radius and Limit not provided - should use defaults
		}

		resp, err := uc3.GetNearestTransportByPriority(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 1500.0, resp.Meta.RadiusM)

		mockTransportRepo3.AssertExpectations(t)
	})
}

func TestTransportUseCase_GetNearestTransportByPriorityBatch(t *testing.T) {
	logger := zap.NewNop()
	mockTransportRepo := &MockTransportRepository{}
	ctx := context.Background()

	uc := usecase.NewTransportUseCase(mockTransportRepo, logger)

	t.Run("success with multiple points", func(t *testing.T) {
		// Mock batch response
		batchResults := []domain.BatchTransportResult{
			{
				PointIndex: 0,
				SearchPoint: domain.TransportSearchPoint{
					Lat:   41.3851,
					Lon:   2.1734,
					Limit: 3,
				},
				Stations: []domain.NearestTransportWithLines{
					{
						StationID: 100,
						Name:      "Catalunya",
						Type:      "metro",
						Lat:       41.3850,
						Lon:       2.1700,
						Distance:  250.5,
						Lines: []domain.TransportLineInfo{
							{
								ID:   1,
								Name: "L1",
								Type: "metro",
							},
						},
					},
				},
			},
			{
				PointIndex: 1,
				SearchPoint: domain.TransportSearchPoint{
					Lat:   48.8566,
					Lon:   2.3522,
					Limit: 3,
				},
				Stations: []domain.NearestTransportWithLines{
					{
						StationID: 200,
						Name:      "Châtelet",
						Type:      "metro",
						Lat:       48.8584,
						Lon:       2.3470,
						Distance:  180.0,
						Lines:     []domain.TransportLineInfo{},
					},
				},
			},
		}

		mockTransportRepo.On("GetNearestTransportByPriorityBatch", ctx,
			mock.MatchedBy(func(points []domain.TransportSearchPoint) bool {
				return len(points) == 2
			}), 1500.0, 3).
			Return(batchResults, nil)

		req := dto.PriorityTransportBatchRequest{
			Points: []dto.PriorityTransportPoint{
				{Lat: 41.3851, Lon: 2.1734},
				{Lat: 48.8566, Lon: 2.3522},
			},
			Radius: 1500,
			Limit:  3,
		}

		resp, err := uc.GetNearestTransportByPriorityBatch(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Len(t, resp.Results, 2)
		assert.Equal(t, 2, resp.Meta.TotalPoints)
		assert.Equal(t, 2, resp.Meta.TotalStations)

		// Check first point result
		assert.Equal(t, 0, resp.Results[0].PointIndex)
		assert.Len(t, resp.Results[0].Stations, 1)
		assert.Equal(t, int64(100), resp.Results[0].Stations[0].StationID)

		// Check second point result
		assert.Equal(t, 1, resp.Results[1].PointIndex)
		assert.Len(t, resp.Results[1].Stations, 1)
		assert.Equal(t, int64(200), resp.Results[1].Stations[0].StationID)

		mockTransportRepo.AssertExpectations(t)
	})

	t.Run("empty points returns error", func(t *testing.T) {
		req := dto.PriorityTransportBatchRequest{
			Points: []dto.PriorityTransportPoint{},
		}

		resp, err := uc.GetNearestTransportByPriorityBatch(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("invalid coordinates in batch", func(t *testing.T) {
		req := dto.PriorityTransportBatchRequest{
			Points: []dto.PriorityTransportPoint{
				{Lat: 999.0, Lon: 2.1734}, // Invalid
			},
			Radius: 1500,
			Limit:  3,
		}

		resp, err := uc.GetNearestTransportByPriorityBatch(ctx, req)

		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("uses default values when not provided", func(t *testing.T) {
		mockTransportRepo2 := &MockTransportRepository{}
		uc2 := usecase.NewTransportUseCase(mockTransportRepo2, logger)

		mockTransportRepo2.On("GetNearestTransportByPriorityBatch", ctx,
			mock.Anything, 1500.0, 3).
			Return([]domain.BatchTransportResult{
				{
					PointIndex:  0,
					SearchPoint: domain.TransportSearchPoint{Lat: 41.3851, Lon: 2.1734, Limit: 3},
					Stations:    []domain.NearestTransportWithLines{},
				},
			}, nil)

		req := dto.PriorityTransportBatchRequest{
			Points: []dto.PriorityTransportPoint{
				{Lat: 41.3851, Lon: 2.1734},
			},
			// Radius and Limit not provided - should use defaults
		}

		resp, err := uc2.GetNearestTransportByPriorityBatch(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, resp)
		assert.Equal(t, 1500.0, resp.Meta.RadiusM)

		mockTransportRepo2.AssertExpectations(t)
	})
}
