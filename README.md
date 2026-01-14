# Location Microservice

Геопространственный микросервис для работы с OSM данными Каталонии.

## Quick Start

### Prerequisites
- Docker and Docker Compose
- Go 1.24.2 or later
- PostgreSQL (for location data)
- Redis (two instances: local cache and shared streams)

### API Server
1. `docker-compose up -d` - запуск PostgreSQL + Redis (local + shared)
2. `cp .env.example .env` - настроить переменные окружения
3. `make migrate-up` - применить миграции
4. `go run cmd/importer/main.go` - импорт OSM данных
5. `go run cmd/api/main.go` - запуск API сервера

### Worker
1. Убедитесь, что PostgreSQL и оба Redis запущены (local cache + shared streams)
2. Настройте `WORKER_ENABLED=true` в `.env`
3. Настройте Redis Streams подключение (REDIS_STREAMS_HOST, REDIS_STREAMS_PORT)
4. `make run-worker` - запуск воркера обогащения локаций

### Testing Worker
```bash
# Опубликовать тестовое событие в shared Redis
make test-publish

# Проверить статус стримов
make check-streams
```

## Components

- **API Server** (`cmd/api`): REST API для поиска локаций, транспорта, POI
- **Worker** (`cmd/worker`): Обработчик событий обогащения локаций из Redis Streams

### Redis Architecture

The microservice uses **two separate Redis instances**:

1. **Redis Cache (local, port 6379)**: Caching tiles, search results, and other local data
2. **Redis Streams (shared, port 6380)**: Shared with backend_estate for location enrichment events
   - Reads from: `stream:location:enrich`
   - Writes to: `stream:location:done`

This separation ensures stream processing is isolated from cache operations and allows the worker to communicate with backend_estate through a dedicated Redis instance.

## Documentation

- API Documentation: `docs/INDEX.md`
- Worker Documentation: `docs/WORKER.md`
- **Swagger API Documentation**: `http://localhost:8080/swagger/` (when server is running)

## Swagger Documentation

Проект использует Swagger/OpenAPI для документирования API endpoints.

### Просмотр документации

1. Запустите API сервер: `make run` или `go run cmd/api/main.go`
2. Откройте в браузере: http://localhost:8080/swagger/
3. Интерактивная документация Swagger UI будет доступна для тестирования всех endpoints

### Генерация Swagger документации

После изменения API handlers или добавления новых endpoints:

```bash
make swagger
```

Это сгенерирует файлы в `docs/swagger/`:
- `swagger.json` - OpenAPI спецификация в JSON
- `swagger.yaml` - OpenAPI спецификация в YAML  
- `docs.go` - Go файл с встроенной документацией

### Swagger UI в Docker

Для просмотра документации без запуска всего сервера:

```bash
make swagger-serve
```

Откроется Swagger UI на http://localhost:8081


Переделать воркер (берём пачку до 50 штук -> находим адрес, обогащаем -> пушим в очередь готовых -> Возвращаемся)

### INDEX OSMDB:

-- Index for administrative boundaries
CREATE INDEX idx_admin_boundaries 
ON planet_osm_polygon (boundary, admin_level) 
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

-- Trigram index for fuzzy name search (requires pg_trgm extension)
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE INDEX idx_admin_name_trgm 
ON planet_osm_polygon USING GIN (name gin_trgm_ops) 
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

-- Composite index for exact name matches
CREATE INDEX idx_admin_name_level 
ON planet_osm_polygon (name, admin_level) 
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

-- Indexes for translated names
CREATE INDEX idx_admin_name_en 
ON planet_osm_polygon ((tags->'name:en')) 
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

CREATE INDEX idx_admin_name_es 
ON planet_osm_polygon ((tags->'name:es')) 
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

-- Filtered spatial index
CREATE INDEX idx_admin_boundaries_way 
ON planet_osm_polygon USING GIST (way) 
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

### INDEX TRANSPORT (Required for fast transport queries ~30ms instead of ~2500ms):

-- Step 1: Add geography columns for fast spatial queries (no ST_Transform overhead)
ALTER TABLE planet_osm_point ADD COLUMN IF NOT EXISTS way_geog geography(Point, 4326);
UPDATE planet_osm_point SET way_geog = ST_Transform(way, 4326)::geography WHERE way_geog IS NULL;

ALTER TABLE planet_osm_line ADD COLUMN IF NOT EXISTS way_geog geography(LineString, 4326);
UPDATE planet_osm_line SET way_geog = ST_Transform(way, 4326)::geography WHERE way_geog IS NULL;

-- Step 2: Create GIST indexes on geography columns
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_planet_osm_point_way_geog 
ON planet_osm_point USING GIST (way_geog);

-- Partial index for transport stations only (most efficient)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_planet_osm_point_transport_geog
ON planet_osm_point USING GIST (way_geog)
WHERE (
    (railway IN ('station', 'halt', 'tram_stop'))
    OR highway = 'bus_stop'
    OR public_transport IN ('platform', 'stop_position', 'station')
);

-- Full GIST index for lines (NOT partial - needed for proximity)
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_planet_osm_line_way_geog 
ON planet_osm_line USING GIST (way_geog);

-- Step 3: B-tree indexes for filtering
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_planet_osm_point_railway 
ON planet_osm_point (railway) WHERE railway IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_planet_osm_point_highway 
ON planet_osm_point (highway) WHERE highway IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_planet_osm_point_public_transport 
ON planet_osm_point (public_transport) WHERE public_transport IS NOT NULL;

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_planet_osm_point_name 
ON planet_osm_point (name) WHERE name IS NOT NULL AND name != '';

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_planet_osm_line_route 
ON planet_osm_line (route) WHERE route IS NOT NULL;

-- Step 4: Update statistics
ANALYZE planet_osm_point;
ANALYZE planet_osm_line;ANALYZE planet_osm_line;

-- Step 5: osm_id indexes for fast lookups
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_planet_osm_point_osm_id 
ON planet_osm_point (osm_id);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_planet_osm_line_osm_id 
ON planet_osm_line (osm_id);

-- Step 6: Update statistics again
ANALYZE planet_osm_point;
ANALYZE planet_osm_line;

-- PERFORMANCE NOTE: 
-- GetLinesByStationIDsBatch uses way (SRID 3857) with native planet_osm_line_way_idx (~100ms)
-- Station search uses way_geog for accurate distance in meters (~32ms)
```
