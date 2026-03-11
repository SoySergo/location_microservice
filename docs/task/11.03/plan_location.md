# План — Location Service (location)

> Задачи привязаны к пунктам из `init.md`. Сервис location по текущему анализу работает корректно — основные баги на стороне фронтенда, который не использует доступные API. Ниже описаны задачи по интеграции и улучшению.

---

## Текущее состояние API

Сервис предоставляет все необходимые endpoints:

| Endpoint | Статус | Используется фронтом |
|----------|--------|---------------------|
| `GET /api/v1/boundaries/tiles/:z/:x/:y.pbf` | ✅ Работает | ✅ Поиск (location-search-mode) |
| `GET /api/v1/tiles/transport/:z/:x/:y.pbf` | ✅ Работает | ❌ Не на деталях объекта |
| `GET /api/v1/tiles/poi/:z/:x/:y.pbf` | ✅ Работает | ❌ Не на деталях объекта |
| `POST /api/v1/transport/nearest` | ✅ Работает | ❓ Частично |
| `GET /api/v1/transport/priority` | ✅ Работает | ❌ Не на деталях объекта |
| `POST /api/v1/radius/poi` | ✅ Работает | ❌ Не на деталях объекта |
| `GET /api/v1/poi/categories` | ✅ Работает | ❌ Не используется |
| `POST /api/v1/locations/enrich` | ✅ Работает | ✅ Worker pipeline |
| `GET /api/v1/green-spaces/tiles/:z/:x/:y.pbf` | ✅ Работает | ❌ Не используется |
| `GET /api/v1/water/tiles/:z/:x/:y.pbf` | ✅ Работает | ❌ Не используется |

---

## 1. Границы исчезают после загрузки (Bug 2.1)

### Анализ

**Проблема не в location service.** PBF тайлы отдаются корректно. Ошибка:
```
"The source 'boundaries' does not exist in the map's style."
```
— это фронтенд race condition: source добавляется, затем стиль карты перезагружается (тема, рендер) и source теряется.

### Что нужно от location service
**Ничего.** Тайл-сервер работает правильно.

### Возможные улучшения (опционально)

1. **Добавить HTTP заголовки кеширования** для тайлов:

   **Файл:** `internal/delivery/http/handler.go` (tile handlers)
   ```go
   c.Set("Cache-Control", "public, max-age=3600")
   c.Set("Access-Control-Allow-Origin", "*")
   ```
   — Уменьшит повторные запросы при переинициализации source на фронте.

2. **Проверить CORS** — убедиться что фронтенд может запрашивать тайлы без ошибок CORS:

   **Файл:** `internal/delivery/http/server.go` — middleware CORS
   ```go
   app.Use(cors.New(cors.Config{
       AllowOrigins: "*",
       AllowMethods: "GET,POST",
       AllowHeaders: "Content-Type",
   }))
   ```

---

## 2. Карта деталей объекта — транспорт и POI не отображаются (Bug 5.2, 5.7, 5.8)

### Анализ

Фронтенд не вызывает следующие endpoints location service на странице деталей:

| Нужный запрос | Endpoint | Назначение |
|---------------|----------|-----------|
| Ближайший транспорт | `GET /api/v1/transport/priority?lat=X&lon=Y&radius=1100&limit=3` | Станции метро/трамвая вверху страницы |
| Тайлы транспорта | `GET /api/v1/tiles/transport/:z/:x/:y.pbf?types=metro,tram,bus,train` | Отображение станций на карте |
| Тайлы POI | `GET /api/v1/tiles/poi/:z/:x/:y.pbf?categories=healthcare,shopping,education,food_drink` | Отображение POI на карте |
| POI по радиусу | `POST /api/v1/radius/poi` с `{lat, lon, radius, categories}` | Список POI в сайдбаре под картой |
| Категории POI | `GET /api/v1/poi/categories` | Вкладки категорий в сайдбаре |

### Что нужно от location service

**API уже готов.** Нужна только интеграция на фронтенде (см. `plan_frontend.md`, секция 5.8).

### Возможные улучшения

1. **Endpoint для композитных данных детальной страницы:**

   Создать агрегирующий endpoint, который вернёт всё за один запрос:

   **Файл:** `internal/delivery/http/handler.go` — новый handler
   ```
   GET /api/v1/property-location?lat=X&lon=Y&radius=1000
   ```
   Ответ:
   ```json
   {
       "nearest_transport": [...],     // priority transport
       "poi_summary": {               // количество POI по категориям
           "healthcare": 5,
           "shopping": 12,
           "education": 3,
           "food_drink": 8
       },
       "environment": {               // наличие зелёных зон, воды
           "green_spaces_nearby": true,
           "water_nearby": false,
           "beach_nearby": false
       }
   }
   ```

   **Файл:** `internal/usecase/property_location_usecase.go` (новый)
   ```go
   type PropertyLocationUseCase struct {
       transportUC  TransportUseCase
       poiUC        POIUseCase
       envUC        EnvironmentUseCase
   }
   
   func (uc *PropertyLocationUseCase) GetPropertyLocationData(ctx context.Context, lat, lon float64, radius int) (*PropertyLocationData, error) {
       // Параллельные запросы
       g, ctx := errgroup.WithContext(ctx)
       
       var transport []TransportStation
       var poiCounts map[string]int
       
       g.Go(func() error {
           var err error
           transport, err = uc.transportUC.GetNearestByPriority(ctx, lat, lon, radius, 5)
           return err
       })
       
       g.Go(func() error {
           var err error
           poiCounts, err = uc.poiUC.CountByCategories(ctx, lat, lon, radius)
           return err
       })
       
       if err := g.Wait(); err != nil { return nil, err }
       
       return &PropertyLocationData{
           NearestTransport: transport,
           POISummary:       poiCounts,
       }, nil
   }
   ```

   **Преимущество:** 1 запрос вместо 3-4 от фронтенда.

2. **Endpoint подсчёта POI по категориям:**

   **Файл:** `internal/usecase/poi_usecase.go`
   ```go
   func (uc *POIUseCase) CountByCategories(ctx context.Context, lat, lon float64, radius int) (map[string]int, error) {
       // SELECT category, COUNT(*) FROM poi 
       // WHERE ST_DWithin(geom, ST_MakePoint($1,$2)::geography, $3)
       // GROUP BY category
   }
   ```

   **Файл:** `internal/repository/postgresosm/poi_repository.go` — добавить метод
   ```go
   func (r *POIRepository) CountByCategories(ctx context.Context, lat, lon float64, radiusMeters int) (map[string]int, error)
   ```

---

## 3. Enrichment pipeline — транспорт для деталей

### Текущее состояние
Worker обогащает `LocationEnrichEvent` → `EnrichedLocation` с:
- Административные границы (country → neighborhood IDs)
- Ближайший транспорт (до 2 станций на точку, приоритетные)

Эти данные записываются в `stream:location:done` и сохраняются бекендом в property.

### Что проверить

1. **Данные транспорта при обогащении сохраняются?**

   **Файл:** `internal/usecase/enriched_location_usecase.go` (строка ~80-120)
   - `GetNearestTransportByPriorityBatch()` — вызывается только для `isVisible=true`
   - Возвращает: station name, type, lat/lon, distance, walking_time, lines
   - Если property не `isVisible` — транспорт не обогащается

   **Проверить:** бекенд (realbro_backend) должен сохранять `nearest_transport` из enrichment response в property.

2. **Walking time корректен?**

   **Файл:** `internal/usecase/transport_usecase.go` (строка ~256)
   ```go
   walkingTime = distance * 1.2 / 1.39 / 60  // Manhattan-distance estimate
   ```
   — Это оценка. Реальное время через Mapbox Matrix API:
   
   **Файл:** `internal/usecase/enrichment_usecase_extended.go`
   - Вызывает `InfrastructureUseCase` → Mapbox Directions Matrix
   - Только для properties с полным адресом

---

## 4. Тайлы для деталей — фильтрация по категориям

### Текущее состояние

POI тайлы уже поддерживают фильтрацию:
```
GET /api/v1/tiles/poi/:z/:x/:y.pbf?categories=healthcare,shopping&subcategories=pharmacy,supermarket
```

Transport тайлы тоже:
```
GET /api/v1/tiles/transport/:z/:x/:y.pbf?types=metro,bus,tram
```

### Что может потребоваться

Фронтенд на карте детальной страницы переключает вкладки: Транспорт, Школы, Медицина и т.д. Для каждой вкладки нужен свой набор source/layer с фильтрацией по категориям.

**Маппинг вкладок → параметров запроса:**

| Вкладка | Endpoint | Параметры |
|---------|----------|-----------|
| Транспорт | `/tiles/transport` | `?types=metro,tram,bus,train` |
| Школы | `/tiles/poi` | `?categories=education` |
| Медицина | `/tiles/poi` | `?categories=healthcare` |
| Магазины | `/tiles/poi` | `?categories=shopping` |
| Рестораны | `/tiles/poi` | `?categories=food_drink` |
| Досуг | `/tiles/poi` | `?categories=leisure` |

**Документация для фронтенда:** уже есть в `docs/POI_TRANSPORT_TILES_API.md`.

---

## 5. Environment тайлы — зелёные зоны, вода, пляжи

### Текущее состояние

Endpoints существуют, но не используются фронтендом:
```
GET /api/v1/green-spaces/tiles/:z/:x/:y.pbf
GET /api/v1/water/tiles/:z/:x/:y.pbf
GET /api/v1/beaches/tiles/:z/:x/:y.pbf
GET /api/v1/noise-sources/tiles/:z/:x/:y.pbf
GET /api/v1/tourist-zones/tiles/:z/:x/:y.pbf
```

### Рекомендация

Добавить эти слои на карту деталей объекта как опциональные (toggle). Фронтенд может переключать видимость:
- Зелёные зоны (парки, сады) — полупрозрачный зелёный полигон
- Вода (реки, озёра) — голубой
- Шумные зоны — красный полупрозрачный overlay
- Туристические зоны — жёлтый

---

## 6. Кеширование и производительность

### Текущее состояние
- Redis кеш для тайлов: `tile:boundaries:z:x:y`, TTL 1 час
- Redis кеш для поиска: `search:query_hash`, TTL 5 минут

### Рекомендации

1. **Увеличить TTL для boundaries тайлов** до 24ч — административные границы не меняются часто:
   
   **Файл:** `internal/usecase/tile_usecase.go`
   ```go
   const boundariesTileTTL = 24 * time.Hour  // было 1 hour
   ```

2. **Добавить ETag/If-None-Match** для тайлов — клиент не перескачивает неизменённые тайлы:
   
   **Файл:** `internal/delivery/http/handler.go` — tile handlers
   ```go
   etag := fmt.Sprintf(`"%x"`, md5.Sum(tileData))
   if c.Get("If-None-Match") == etag {
       return c.SendStatus(304)
   }
   c.Set("ETag", etag)
   ```

3. **Prefetch тайлов** для популярных zoom levels (10-14) — warm cache при старте:
   
   **Файл:** `internal/worker/` — новый prefetch worker
   ```go
   // При запуске — прогреть кеш для основных городов (Barcelona, Madrid) на zoom 10-14
   ```

---

## Порядок приоритетов

| # | Задача | Тип | Критичность | Сложность |
|---|--------|-----|-------------|-----------|
| 1 | CORS headers для тайлов | Fix | 🔴 Высокая | Низкая |
| 2 | Агрегирующий endpoint property-location | Feature | 🟡 Средняя | Средняя |
| 3 | POI count by categories | Feature | 🟡 Средняя | Низкая |
| 4 | Cache TTL для boundaries (24h) | Optimization | 🟢 Низкая | Низкая |
| 5 | ETag для тайлов | Optimization | 🟢 Низкая | Низкая |
| 6 | Документация интеграции для фронта | Docs | 🟡 Средняя | Низкая |

---

## Резюме

**Location service работает корректно.** Все API endpoints функционируют. Основные проблемы:
1. **Фронтенд race condition** при инициализации boundaries source → исправлять на фронте
2. **Фронтенд не запрашивает** транспорт/POI тайлы на странице деталей → интеграция на фронте
3. **Опциональные улучшения**: агрегирующий endpoint, кеширование, ETag
