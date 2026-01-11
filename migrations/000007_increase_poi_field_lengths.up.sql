-- Увеличение размеров полей в таблице POIs для предотвращения ошибок усечения данных

-- Увеличиваем размеры полей с ограничениями VARCHAR(100) до VARCHAR(255)
ALTER TABLE pois ALTER COLUMN brand TYPE VARCHAR(255);
ALTER TABLE pois ALTER COLUMN operator TYPE VARCHAR(255);
ALTER TABLE pois ALTER COLUMN cuisine TYPE VARCHAR(200);
ALTER TABLE pois ALTER COLUMN charge TYPE VARCHAR(200);

-- Увеличиваем размеры полей с ограничениями VARCHAR(50)
ALTER TABLE pois ALTER COLUMN phone TYPE VARCHAR(100);
ALTER TABLE pois ALTER COLUMN diet TYPE VARCHAR(100);
ALTER TABLE pois ALTER COLUMN internet TYPE VARCHAR(100);
ALTER TABLE pois ALTER COLUMN smoking TYPE VARCHAR(100);

-- Изменение типа геометрии beaches для поддержки разных типов (Point, Polygon)
-- В OSM пляжи могут быть представлены как точками, так и полигонами
ALTER TABLE beaches ALTER COLUMN geometry TYPE GEOMETRY(GEOMETRY, 4326);
COMMENT ON COLUMN beaches.geometry IS 'Геометрия пляжа: может быть Point (для точечной отметки) или Polygon (для площади пляжа)';
