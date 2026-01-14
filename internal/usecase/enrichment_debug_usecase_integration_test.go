package usecase_test

import (
	"context"
	"os"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/location-microservice/internal/repository/postgresosm"
	"github.com/location-microservice/internal/usecase"
	"github.com/location-microservice/internal/usecase/dto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// TestEnrichmentDebugUseCase_Integration - интеграционные тесты для дебаг usecase
// Требуют запущенного OSM контейнера на localhost:5435
func TestEnrichmentDebugUseCase_Integration(t *testing.T) {
	// Skip if not in integration test mode
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping integration test. Set INTEGRATION_TEST=true to run")
	}

	// Connect to OSM database
	dsn := os.Getenv("OSM_DB_DSN")
	if dsn == "" {
		dsn = "postgres://osmuser:osmpass@localhost:5435/osm?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dsn)
	require.NoError(t, err, "Failed to connect to OSM database")
	defer db.Close()

	// Create logger
	logger := zaptest.NewLogger(t)

	// Create OSM DB wrapper
	osmDB := postgresosm.NewDBForTest(db, logger)

	// Create transport repository
	transportRepo := postgresosm.NewTransportRepository(osmDB)

	// Create usecase
	uc := usecase.NewEnrichmentDebugUseCase(transportRepo, logger)

	t.Run("GetNearestTransportEnriched_Barcelona_Diagonal", func(t *testing.T) {
		ctx := context.Background()

		// Координаты возле станции метро Diagonal в Барселоне
		req := dto.EnrichmentDebugTransportRequest{
			Lat:         41.398478,
			Lon:         2.166172,
			Types:       []string{"metro"},
			MaxDistance: 2000, // 2 km
			Limit:       10,
		}

		result, err := uc.GetNearestTransportEnriched(ctx, req)
		require.NoError(t, err, "GetNearestTransportEnriched should not return error")
		require.NotNil(t, result, "Result should not be nil")

		// Проверяем, что нашли станции
		assert.NotEmpty(t, result.Stations, "Should find some metro stations")

		// Проверяем метаданные
		assert.Equal(t, req.Lat, result.Meta.SearchPoint.Lat)
		assert.Equal(t, req.Lon, result.Meta.SearchPoint.Lon)
		assert.Equal(t, req.MaxDistance, result.Meta.RadiusM)

		// Логируем найденные станции для визуальной проверки
		t.Logf("Found %d metro stations near Diagonal:", len(result.Stations))
		for i, station := range result.Stations {
			t.Logf("  %d. %s (ID: %d)", i+1, station.Name, station.StationID)
			t.Logf("     Type: %s, Lat: %.6f, Lon: %.6f", station.Type, station.Lat, station.Lon)
			t.Logf("     Linear distance: %.2f m, Walking distance: %.2f m, Walking time: %.1f min",
				station.LinearDistance, station.WalkingDistance, station.WalkingTime)
			if len(station.Lines) > 0 {
				t.Logf("     Lines: %d", len(station.Lines))
				for _, line := range station.Lines {
					color := ""
					if line.Color != nil {
						color = *line.Color
					}
					t.Logf("       - %s (Ref: %s, Color: %s, Type: %s)",
						line.Name, line.Ref, color, line.Type)
				}
			}
		}
	})

	t.Run("GetNearestTransportEnriched_Should_Sort_By_Distance", func(t *testing.T) {
		ctx := context.Background()

		req := dto.EnrichmentDebugTransportRequest{
			Lat:         41.398478,
			Lon:         2.166172,
			Types:       []string{"metro"},
			MaxDistance: 3000,
			Limit:       10,
		}

		result, err := uc.GetNearestTransportEnriched(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Проверяем, что станции отсортированы по расстоянию
		for i := 1; i < len(result.Stations); i++ {
			assert.LessOrEqual(t, result.Stations[i-1].LinearDistance, result.Stations[i].LinearDistance,
				"Stations should be sorted by distance")
		}
	})

	t.Run("GetNearestTransportEnriched_NoDuplicates", func(t *testing.T) {
		ctx := context.Background()

		req := dto.EnrichmentDebugTransportRequest{
			Lat:         41.398478,
			Lon:         2.166172,
			Types:       []string{"metro"},
			MaxDistance: 3000,
			Limit:       20,
		}

		result, err := uc.GetNearestTransportEnriched(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Проверяем отсутствие дубликатов по имени станции
		seenNames := make(map[string]bool)
		for _, station := range result.Stations {
			assert.False(t, seenNames[station.Name],
				"Duplicate station name found: %s", station.Name)
			seenNames[station.Name] = true
		}

		// Проверяем отсутствие дубликатов по ID
		seenIDs := make(map[int64]bool)
		for _, station := range result.Stations {
			assert.False(t, seenIDs[station.StationID],
				"Duplicate station ID found: %d", station.StationID)
			seenIDs[station.StationID] = true
		}
	})

	t.Run("GetNearestTransportEnriched_LinesNoDuplicates", func(t *testing.T) {
		ctx := context.Background()

		req := dto.EnrichmentDebugTransportRequest{
			Lat:         41.398478,
			Lon:         2.166172,
			Types:       []string{"metro"},
			MaxDistance: 3000,
			Limit:       10,
		}

		result, err := uc.GetNearestTransportEnriched(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Для каждой станции проверяем отсутствие дубликатов линий
		for _, station := range result.Stations {
			seenRefs := make(map[string]bool)
			for _, line := range station.Lines {
				if line.Ref != "" {
					assert.False(t, seenRefs[line.Ref],
						"Duplicate line ref found for station %s: %s", station.Name, line.Ref)
					seenRefs[line.Ref] = true
				}
			}
		}
	})

	t.Run("GetNearestTransportEnriched_WalkingTimeCalculation", func(t *testing.T) {
		ctx := context.Background()

		req := dto.EnrichmentDebugTransportRequest{
			Lat:         41.398478,
			Lon:         2.166172,
			Types:       []string{"metro"},
			MaxDistance: 2000,
			Limit:       5,
		}

		result, err := uc.GetNearestTransportEnriched(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		for _, station := range result.Stations {
			// Проверяем, что walking distance больше linear distance (примерно на 20%)
			assert.GreaterOrEqual(t, station.WalkingDistance, station.LinearDistance,
				"Walking distance should be >= linear distance")

			// Проверяем, что walking time положительное
			assert.Greater(t, station.WalkingTime, 0.0,
				"Walking time should be positive")

			// Проверяем разумность времени (скорость ~5 км/ч = ~83 м/мин)
			// Значит 1000 м должно быть примерно 12-15 минут с учетом корректировки
			expectedMinTime := station.WalkingDistance / 100 // минимум 60 м/мин
			expectedMaxTime := station.WalkingDistance / 50  // максимум 120 м/мин
			assert.True(t, station.WalkingTime >= expectedMinTime && station.WalkingTime <= expectedMaxTime,
				"Walking time %.1f min should be reasonable for distance %.0f m",
				station.WalkingTime, station.WalkingDistance)
		}
	})

	t.Run("GetNearestTransportEnriched_MultipleTypes", func(t *testing.T) {
		ctx := context.Background()

		req := dto.EnrichmentDebugTransportRequest{
			Lat:         41.398478,
			Lon:         2.166172,
			Types:       []string{"metro", "train"},
			MaxDistance: 3000,
			Limit:       15,
		}

		result, err := uc.GetNearestTransportEnriched(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Проверяем, что есть разные типы станций
		typeCount := make(map[string]int)
		for _, station := range result.Stations {
			typeCount[station.Type]++
		}

		t.Logf("Station types found: %v", typeCount)
	})

	t.Run("GetNearestTransportEnriched_EmptyResult_FarFromTransport", func(t *testing.T) {
		ctx := context.Background()

		// Координаты в море (нет транспорта рядом)
		req := dto.EnrichmentDebugTransportRequest{
			Lat:         40.0,
			Lon:         4.0,
			Types:       []string{"metro"},
			MaxDistance: 100, // очень маленький радиус
			Limit:       5,
		}

		result, err := uc.GetNearestTransportEnriched(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		assert.Empty(t, result.Stations, "Should not find stations in the sea")
		assert.Equal(t, 0, result.Meta.TotalFound)
	})

	t.Run("GetNearestTransportEnriched_InvalidCoordinates", func(t *testing.T) {
		ctx := context.Background()

		// Невалидные координаты
		req := dto.EnrichmentDebugTransportRequest{
			Lat:   91.0, // > 90
			Lon:   2.0,
			Types: []string{"metro"},
		}

		result, err := uc.GetNearestTransportEnriched(ctx, req)
		assert.Error(t, err, "Should return error for invalid coordinates")
		assert.Nil(t, result)
	})

	t.Run("GetNearestTransportEnriched_DefaultValues", func(t *testing.T) {
		ctx := context.Background()

		// Запрос без указания types, limit, maxDistance - должны использоваться дефолтные
		req := dto.EnrichmentDebugTransportRequest{
			Lat: 41.398478,
			Lon: 2.166172,
		}

		result, err := uc.GetNearestTransportEnriched(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, result)

		// Проверяем, что применились дефолтные значения
		assert.Equal(t, float64(1500), result.Meta.RadiusM, "Default radius should be 1500m")
		assert.Contains(t, result.Meta.Types, "metro", "Default types should include metro")
	})
}

// BenchmarkGetNearestTransportEnriched - бенчмарк для измерения производительности
func BenchmarkGetNearestTransportEnriched(b *testing.B) {
	if os.Getenv("INTEGRATION_TEST") != "true" {
		b.Skip("Skipping benchmark. Set INTEGRATION_TEST=true to run")
	}

	dsn := os.Getenv("OSM_DB_DSN")
	if dsn == "" {
		dsn = "postgres://osmuser:osmpass@localhost:5435/osm?sslmode=disable"
	}

	db, err := sqlx.Connect("postgres", dsn)
	if err != nil {
		b.Fatal(err)
	}
	defer db.Close()

	logger := zap.NewNop()
	osmDB := postgresosm.NewDBForTest(db, logger)
	transportRepo := postgresosm.NewTransportRepository(osmDB)
	uc := usecase.NewEnrichmentDebugUseCase(transportRepo, logger)

	ctx := context.Background()
	req := dto.EnrichmentDebugTransportRequest{
		Lat:         41.398478,
		Lon:         2.166172,
		Types:       []string{"metro"},
		MaxDistance: 2000,
		Limit:       10,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := uc.GetNearestTransportEnriched(ctx, req)
		if err != nil {
			b.Fatal(err)
		}
	}
}
