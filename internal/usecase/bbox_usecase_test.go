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

// TestPOIUseCase_GetPOIInBBox tests the GetPOIInBBox usecase method
func TestPOIUseCase_GetPOIInBBox(t *testing.T) {
	logger := zap.NewNop()
	addr := "Test Street 1"
	phone := "+7 123 456"

	t.Run("success", func(t *testing.T) {
		mockPOI := &mockPOIRepository{}
		uc := usecase.NewPOIUseCase(mockPOI, logger)
		ctx := context.Background()

		mockPOI.On("GetPOIInBBox", mock.Anything,
			41.38, 2.17, 41.40, 2.19,
			[]string{"healthcare"}, []string{"pharmacy"},
			10, 0,
		).Return([]*domain.POI{
			{
				ID:          1,
				OSMId:       100,
				Name:        "Farmacia Central",
				Category:    "healthcare",
				Subcategory: "pharmacy",
				Lat:         41.39,
				Lon:         2.18,
				Address:     &addr,
				Phone:       &phone,
			},
		}, 1, nil)

		req := dto.BBoxPOIRequest{
			SwLat:         41.38,
			SwLon:         2.17,
			NeLat:         41.40,
			NeLon:         2.19,
			Categories:    []string{"healthcare"},
			Subcategories: []string{"pharmacy"},
			Limit:         10,
			Offset:        0,
		}

		result, err := uc.GetPOIInBBox(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.POIs, 1)
		assert.Equal(t, "100", result.POIs[0].ID)
		assert.Equal(t, "Farmacia Central", result.POIs[0].Name)
		assert.Equal(t, "healthcare", result.POIs[0].Category)
		assert.Equal(t, &addr, result.POIs[0].Address)
		assert.Equal(t, &phone, result.POIs[0].Phone)
		assert.Equal(t, 1, result.Total)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 0, result.Offset)
		mockPOI.AssertExpectations(t)
	})

	t.Run("invalid coordinates", func(t *testing.T) {
		mockPOI := &mockPOIRepository{}
		uc := usecase.NewPOIUseCase(mockPOI, logger)

		req := dto.BBoxPOIRequest{
			SwLat: 999,
			SwLon: 999,
			NeLat: 41.40,
			NeLon: 2.19,
			Limit: 10,
		}

		result, err := uc.GetPOIInBBox(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("default limit applied when zero", func(t *testing.T) {
		mockPOI := &mockPOIRepository{}
		uc := usecase.NewPOIUseCase(mockPOI, logger)
		ctx := context.Background()

		// Expect default limit of 10
		mockPOI.On("GetPOIInBBox", mock.Anything,
			41.38, 2.17, 41.40, 2.19,
			[]string(nil), []string(nil),
			10, 0,
		).Return([]*domain.POI{}, 0, nil)

		req := dto.BBoxPOIRequest{
			SwLat:  41.38,
			SwLon:  2.17,
			NeLat:  41.40,
			NeLon:  2.19,
			Limit:  0, // should default to 10
			Offset: 0,
		}

		result, err := uc.GetPOIInBBox(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 10, result.Limit)
		mockPOI.AssertExpectations(t)
	})

	t.Run("limit capped at 100", func(t *testing.T) {
		mockPOI := &mockPOIRepository{}
		uc := usecase.NewPOIUseCase(mockPOI, logger)
		ctx := context.Background()

		mockPOI.On("GetPOIInBBox", mock.Anything,
			41.38, 2.17, 41.40, 2.19,
			[]string(nil), []string(nil),
			100, 0,
		).Return([]*domain.POI{}, 0, nil)

		req := dto.BBoxPOIRequest{
			SwLat:  41.38,
			SwLon:  2.17,
			NeLat:  41.40,
			NeLon:  2.19,
			Limit:  500, // should be capped to 100
			Offset: 0,
		}

		result, err := uc.GetPOIInBBox(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 100, result.Limit)
		mockPOI.AssertExpectations(t)
	})
}

// TestTransportUseCase_GetTransportInBBox tests the GetTransportInBBox usecase method
func TestTransportUseCase_GetTransportInBBox(t *testing.T) {
	logger := zap.NewNop()
	color := "#E32019"

	t.Run("success", func(t *testing.T) {
		mockTransport := &MockTransportRepository{}
		uc := usecase.NewTransportUseCase(mockTransport, logger)
		ctx := context.Background()

		mockTransport.On("GetStationsInBBox", mock.Anything,
			41.38, 2.17, 41.40, 2.19,
			[]string{"metro"}, 10, 0,
		).Return([]domain.TransportStationWithLines{
			{
				StationID: 42,
				Name:      "Catalunya",
				Type:      "metro",
				Lat:       41.39,
				Lon:       2.18,
				Lines: []domain.TransportLineInfo{
					{ID: 1, Name: "L1", Ref: "L1", Type: "metro", Color: &color},
				},
			},
		}, 1, nil)

		req := dto.BBoxTransportRequest{
			SwLat:  41.38,
			SwLon:  2.17,
			NeLat:  41.40,
			NeLon:  2.19,
			Types:  []string{"metro"},
			Limit:  10,
			Offset: 0,
		}

		result, err := uc.GetTransportInBBox(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Stations, 1)
		assert.Equal(t, "42", result.Stations[0].ID)
		assert.Equal(t, "Catalunya", result.Stations[0].Name)
		assert.Equal(t, "metro", result.Stations[0].Type)
		assert.Len(t, result.Stations[0].Lines, 1)
		assert.Equal(t, "1", result.Stations[0].Lines[0].ID)
		assert.Equal(t, &color, result.Stations[0].Lines[0].Color)
		assert.Equal(t, 1, result.Total)
		assert.Equal(t, 10, result.Limit)
		assert.Equal(t, 0, result.Offset)
		mockTransport.AssertExpectations(t)
	})

	t.Run("invalid coordinates", func(t *testing.T) {
		mockTransport := &MockTransportRepository{}
		uc := usecase.NewTransportUseCase(mockTransport, logger)

		req := dto.BBoxTransportRequest{
			SwLat: 999,
			SwLon: 999,
			NeLat: 41.40,
			NeLon: 2.19,
			Limit: 10,
		}

		result, err := uc.GetTransportInBBox(context.Background(), req)

		assert.Error(t, err)
		assert.Nil(t, result)
	})

	t.Run("default limit applied when zero", func(t *testing.T) {
		mockTransport := &MockTransportRepository{}
		uc := usecase.NewTransportUseCase(mockTransport, logger)
		ctx := context.Background()

		mockTransport.On("GetStationsInBBox", mock.Anything,
			41.38, 2.17, 41.40, 2.19,
			[]string(nil), 10, 0,
		).Return([]domain.TransportStationWithLines{}, 0, nil)

		req := dto.BBoxTransportRequest{
			SwLat:  41.38,
			SwLon:  2.17,
			NeLat:  41.40,
			NeLon:  2.19,
			Limit:  0, // should default to 10
			Offset: 0,
		}

		result, err := uc.GetTransportInBBox(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 10, result.Limit)
		mockTransport.AssertExpectations(t)
	})

	t.Run("limit capped at 100", func(t *testing.T) {
		mockTransport := &MockTransportRepository{}
		uc := usecase.NewTransportUseCase(mockTransport, logger)
		ctx := context.Background()

		mockTransport.On("GetStationsInBBox", mock.Anything,
			41.38, 2.17, 41.40, 2.19,
			[]string(nil), 100, 0,
		).Return([]domain.TransportStationWithLines{}, 0, nil)

		req := dto.BBoxTransportRequest{
			SwLat:  41.38,
			SwLon:  2.17,
			NeLat:  41.40,
			NeLon:  2.19,
			Limit:  500, // should be capped to 100
			Offset: 0,
		}

		result, err := uc.GetTransportInBBox(ctx, req)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 100, result.Limit)
		mockTransport.AssertExpectations(t)
	})
}
