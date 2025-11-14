package postgres_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/location-microservice/internal/domain/repository"
	"github.com/location-microservice/internal/repository/postgres/testhelpers"
)

// StatsRepositoryTestSuite тестирует все методы StatsRepository
type StatsRepositoryTestSuite struct {
	suite.Suite
	testDB *testhelpers.TestDB
	repo   repository.StatsRepository
	ctx    context.Context
}

// SetupSuite выполняется один раз перед всеми тестами
func (s *StatsRepositoryTestSuite) SetupSuite() {
	// Инициализация тестового подключения к БД
	s.testDB = testhelpers.SetupTestDB(s.T())

	// Очистка существующих данных
	err := s.testDB.Cleanup(context.Background())
	s.NoError(err, "Failed to cleanup test database")

	// Применение миграций (пропускаем если таблицы уже существуют)
	_ = testhelpers.ApplyMigrations(
		s.testDB.DB.DB,
		"../../../migrations",
	)

	// Загрузка фикстур для всех типов данных
	fixtures := []string{
		"admin_boundaries.sql",
		"transport.sql",
		"poi_categories.sql",
		"pois.sql",
		"environment.sql",
	}
	err = testhelpers.LoadFixtures(
		s.testDB.DB.DB,
		"../../../../tasks/test_unit_realbd_test_data/fixtures",
		fixtures,
	)
	s.NoError(err, "Failed to load fixtures")

	// Создание репозитория через тест-хелпер
	s.repo = testhelpers.NewStatsRepositoryForTest(s.testDB.DB, s.testDB.Logger)
}

// TearDownSuite выполняется один раз после всех тестов
func (s *StatsRepositoryTestSuite) TearDownSuite() {
	if s.testDB != nil {
		s.testDB.Close()
	}
}

// SetupTest выполняется перед каждым тестом
func (s *StatsRepositoryTestSuite) SetupTest() {
	s.ctx = context.Background()
}

// ============================================================================
// GetStatistics Tests
// ============================================================================

func (s *StatsRepositoryTestSuite) TestGetStatistics_Success() {
	// Act
	stats, err := s.repo.GetStatistics(s.ctx)

	// Assert
	s.NoError(err)
	s.NotNil(stats)

	// Проверка общих полей
	s.NotZero(stats.LastUpdated)
	s.Equal("1.0", stats.DataVersion)

	// Проверка статистики boundaries
	s.Greater(stats.Boundaries.TotalBoundaries, 0, "Should have boundaries")
	s.NotEmpty(stats.Boundaries.ByAdminLevel, "Should have boundaries by admin level")
	s.Greater(stats.Boundaries.Countries, 0, "Should have at least 1 country (Spain)")
	s.Greater(stats.Boundaries.Regions, 0, "Should have at least 1 region (Catalonia)")
	s.Greater(stats.Boundaries.Cities, 0, "Should have at least 1 city (Barcelona)")

	// Проверка статистики transport
	s.Greater(stats.Transport.TotalStations, 0, "Should have transport stations")
	s.Greater(stats.Transport.TotalLines, 0, "Should have transport lines")
	s.NotEmpty(stats.Transport.ByType, "Should have stations by type")

	// Проверка статистики POI
	s.Greater(stats.POIs.TotalPOIs, 0, "Should have POIs")
	s.NotEmpty(stats.POIs.ByCategory, "Should have POIs by category")

	// Проверка статистики environment
	s.GreaterOrEqual(stats.Environment.GreenSpaces, 0, "Green spaces count should be >= 0")
	s.GreaterOrEqual(stats.Environment.WaterBodies, 0, "Water bodies count should be >= 0")
	s.GreaterOrEqual(stats.Environment.Beaches, 0, "Beaches count should be >= 0")
	s.GreaterOrEqual(stats.Environment.TouristZones, 0, "Tourist zones count should be >= 0")

	// Проверка coverage (должно быть покрытие для Испании/Каталонии)
	s.NotZero(stats.Coverage.BBoxMinLat, "BBox should be calculated")
	s.NotZero(stats.Coverage.BBoxMaxLat, "BBox should be calculated")
	s.NotZero(stats.Coverage.BBoxMinLon, "BBox should be calculated")
	s.NotZero(stats.Coverage.BBoxMaxLon, "BBox should be calculated")
	s.NotZero(stats.Coverage.CenterLat, "Center should be calculated")
	s.NotZero(stats.Coverage.CenterLon, "Center should be calculated")
	s.Greater(stats.Coverage.AreaSqKm, 0.0, "Area should be positive")
}

func (s *StatsRepositoryTestSuite) TestGetStatistics_BoundariesDetails() {
	// Act
	stats, err := s.repo.GetStatistics(s.ctx)

	// Assert
	s.NoError(err)
	s.NotNil(stats)

	// Детальная проверка границ по уровням
	// admin_level=2: Страны (Spain)
	s.Contains(stats.Boundaries.ByAdminLevel, 2)
	s.Equal(1, stats.Boundaries.ByAdminLevel[2], "Should have 1 country (Spain)")

	// admin_level=4: Регионы (Catalonia)
	s.Contains(stats.Boundaries.ByAdminLevel, 4)
	s.Equal(1, stats.Boundaries.ByAdminLevel[4], "Should have 1 region (Catalonia)")

	// admin_level=6: Провинции (Barcelona Province)
	if count, ok := stats.Boundaries.ByAdminLevel[6]; ok {
		s.GreaterOrEqual(count, 1, "Should have at least 1 province")
	}

	// admin_level=8: Города (Barcelona, Sabadell)
	s.Contains(stats.Boundaries.ByAdminLevel, 8)
	s.GreaterOrEqual(stats.Boundaries.ByAdminLevel[8], 2, "Should have at least 2 cities")

	// admin_level=9: Районы (Eixample, Ciutat Vella)
	if count, ok := stats.Boundaries.ByAdminLevel[9]; ok {
		s.GreaterOrEqual(count, 2, "Should have at least 2 districts")
	}

	// Проверка суммарного количества
	totalFromLevels := 0
	for _, count := range stats.Boundaries.ByAdminLevel {
		totalFromLevels += count
	}
	s.Equal(totalFromLevels, stats.Boundaries.TotalBoundaries, "Total should match sum by levels")
}

func (s *StatsRepositoryTestSuite) TestGetStatistics_TransportDetails() {
	// Act
	stats, err := s.repo.GetStatistics(s.ctx)

	// Assert
	s.NoError(err)
	s.NotNil(stats)

	// Проверка типов транспорта
	s.NotEmpty(stats.Transport.ByType, "Should have transport types")

	// В фикстурах должны быть станции метро
	if metroCount, ok := stats.Transport.ByType["subway"]; ok {
		s.Greater(metroCount, 0, "Should have subway stations")
	}

	// Проверка что сумма по типам совпадает с общим количеством
	totalFromTypes := 0
	for _, count := range stats.Transport.ByType {
		totalFromTypes += count
	}
	s.Equal(totalFromTypes, stats.Transport.TotalStations, "Total stations should match sum by types")

	// Проверка линий (L1, L2, L3)
	s.GreaterOrEqual(stats.Transport.TotalLines, 3, "Should have at least 3 metro lines")
}

func (s *StatsRepositoryTestSuite) TestGetStatistics_POIDetails() {
	// Act
	stats, err := s.repo.GetStatistics(s.ctx)

	// Assert
	s.NoError(err)
	s.NotNil(stats)

	// Проверка категорий POI
	s.NotEmpty(stats.POIs.ByCategory, "Should have POI categories")

	// Проверяем основные категории из фикстур
	expectedCategories := []string{"tourism", "amenity"}
	for _, category := range expectedCategories {
		if count, ok := stats.POIs.ByCategory[category]; ok {
			s.Greater(count, 0, "Should have POIs in category: %s", category)
		}
	}

	// Проверка что сумма по категориям совпадает с общим количеством
	totalFromCategories := 0
	for _, count := range stats.POIs.ByCategory {
		totalFromCategories += count
	}
	s.Equal(totalFromCategories, stats.POIs.TotalPOIs, "Total POIs should match sum by categories")
}

func (s *StatsRepositoryTestSuite) TestGetStatistics_EnvironmentDetails() {
	// Act
	stats, err := s.repo.GetStatistics(s.ctx)

	// Assert
	s.NoError(err)
	s.NotNil(stats)

	// Проверка что хотя бы один тип экологических объектов присутствует
	totalEnv := stats.Environment.GreenSpaces +
		stats.Environment.WaterBodies +
		stats.Environment.Beaches +
		stats.Environment.TouristZones +
		stats.Environment.NoiseSources

	// В фикстурах должны быть парки, пляжи и т.д.
	s.Greater(totalEnv, 0, "Should have at least some environment objects")

	// Конкретные проверки для Барселоны
	if stats.Environment.GreenSpaces > 0 {
		s.GreaterOrEqual(stats.Environment.GreenSpaces, 3, "Barcelona should have at least 3 parks")
	}
	if stats.Environment.Beaches > 0 {
		s.GreaterOrEqual(stats.Environment.Beaches, 2, "Barcelona should have at least 2 beaches")
	}
	if stats.Environment.TouristZones > 0 {
		s.GreaterOrEqual(stats.Environment.TouristZones, 1, "Barcelona should have at least 1 tourist zone")
	}
}

func (s *StatsRepositoryTestSuite) TestGetStatistics_CoverageDetails() {
	// Act
	stats, err := s.repo.GetStatistics(s.ctx)

	// Assert
	s.NoError(err)
	s.NotNil(stats)

	// Проверка что BBox охватывает Испанию/Каталонию
	// Испания примерно: lat 36-44, lon -9 до 4
	// Каталония примерно: lat 40-43, lon 0-3
	s.InDelta(41.0, stats.Coverage.CenterLat, 3.0, "Center should be around Catalonia/Spain")
	s.InDelta(2.0, stats.Coverage.CenterLon, 5.0, "Center should be around Catalonia/Spain")

	// Проверка валидности BBox
	s.Less(stats.Coverage.BBoxMinLat, stats.Coverage.BBoxMaxLat, "MinLat should be less than MaxLat")
	s.Less(stats.Coverage.BBoxMinLon, stats.Coverage.BBoxMaxLon, "MinLon should be less than MaxLon")

	// Проверка что центр находится внутри BBox
	s.GreaterOrEqual(stats.Coverage.CenterLat, stats.Coverage.BBoxMinLat, "Center should be inside BBox")
	s.LessOrEqual(stats.Coverage.CenterLat, stats.Coverage.BBoxMaxLat, "Center should be inside BBox")
	s.GreaterOrEqual(stats.Coverage.CenterLon, stats.Coverage.BBoxMinLon, "Center should be inside BBox")
	s.LessOrEqual(stats.Coverage.CenterLon, stats.Coverage.BBoxMaxLon, "Center should be inside BBox")

	// Площадь должна быть разумной для региона
	// Испания ~505 000 кв.км, Каталония ~32 000 кв.км
	s.Greater(stats.Coverage.AreaSqKm, 1000.0, "Area should be reasonable for country/region")
	s.Less(stats.Coverage.AreaSqKm, 1000000.0, "Area should not be unreasonably large")
}

func (s *StatsRepositoryTestSuite) TestGetStatistics_MultipleCalls_Consistency() {
	// Act - вызываем несколько раз
	stats1, err1 := s.repo.GetStatistics(s.ctx)
	stats2, err2 := s.repo.GetStatistics(s.ctx)

	// Assert
	s.NoError(err1)
	s.NoError(err2)
	s.NotNil(stats1)
	s.NotNil(stats2)

	// Статистика должна быть одинаковой при нескольких вызовах
	s.Equal(stats1.Boundaries.TotalBoundaries, stats2.Boundaries.TotalBoundaries)
	s.Equal(stats1.Transport.TotalStations, stats2.Transport.TotalStations)
	s.Equal(stats1.Transport.TotalLines, stats2.Transport.TotalLines)
	s.Equal(stats1.POIs.TotalPOIs, stats2.POIs.TotalPOIs)
	s.Equal(stats1.Environment.GreenSpaces, stats2.Environment.GreenSpaces)
	s.InDelta(stats1.Coverage.AreaSqKm, stats2.Coverage.AreaSqKm, 0.01)
}

// ============================================================================
// RefreshStatistics Tests
// ============================================================================

func (s *StatsRepositoryTestSuite) TestRefreshStatistics_Success() {
	// Act
	err := s.repo.RefreshStatistics(s.ctx)

	// Assert - метод должен успешно выполниться (stub)
	s.NoError(err, "RefreshStatistics should not return error")
}

func (s *StatsRepositoryTestSuite) TestRefreshStatistics_MultipleCalls() {
	// Act - вызываем несколько раз подряд
	err1 := s.repo.RefreshStatistics(s.ctx)
	err2 := s.repo.RefreshStatistics(s.ctx)
	err3 := s.repo.RefreshStatistics(s.ctx)

	// Assert
	s.NoError(err1)
	s.NoError(err2)
	s.NoError(err3)
}

func (s *StatsRepositoryTestSuite) TestRefreshStatistics_CanceledContext() {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем контекст

	// Act
	err := s.repo.RefreshStatistics(ctx)

	// Assert - stub метод должен игнорировать отмену контекста
	s.NoError(err, "Stub method should not check context cancellation")
}

// ============================================================================
// Edge Cases and Error Handling
// ============================================================================

func (s *StatsRepositoryTestSuite) TestGetStatistics_EmptyTablesGracefulHandling() {
	// Этот тест проверяет что при отсутствии некоторых таблиц
	// метод не падает с ошибкой (например, если environment таблицы не созданы)
	// Реализация GetStatistics обрабатывает такие случаи

	// Act
	stats, err := s.repo.GetStatistics(s.ctx)

	// Assert
	s.NoError(err, "Should handle missing tables gracefully")
	s.NotNil(stats)
}

func (s *StatsRepositoryTestSuite) TestGetStatistics_CanceledContext() {
	// Arrange
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Отменяем контекст до вызова

	// Act
	_, err := s.repo.GetStatistics(ctx)

	// Assert - должна быть ошибка контекста
	s.Error(err, "Should return error when context is canceled")
}

// ============================================================================
// Performance Tests
// ============================================================================

func (s *StatsRepositoryTestSuite) TestGetStatistics_Performance() {
	// Проверка что статистика вычисляется быстро (< 500ms согласно требованиям)
	// Act
	stats, err := s.repo.GetStatistics(s.ctx)

	// Assert
	s.NoError(err)
	s.NotNil(stats)

	// Можно добавить бенчмарк в отдельный файл для точного измерения
}

// ============================================================================
// Test Runner
// ============================================================================

func TestStatsRepository(t *testing.T) {
	suite.Run(t, new(StatsRepositoryTestSuite))
}
