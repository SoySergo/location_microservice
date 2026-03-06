# Инструкции для агента: location (сервис локаций)

**Сервис**: `/home/sergio/realbro/location`  
**Язык**: Go (Fiber v2, sqlx, lib/pq, pgx/v5, PostGIS)  
**Архитектура**: Clean Architecture — `domain/` → `domain/repository/` (интерфейсы) → `usecase/` → `repository/postgresosm/` → `delivery/http/`  
**БД**: PostgreSQL с PostGIS, данные OSM (таблицы `planet_osm_point`, `planet_osm_line`, `planet_osm_polygon`)

---

## Задача B2.1: Фильтрация транспортных линий при обогащении

Две подзадачи:
1. Приоритет транспорта: metro > train > tram > bus (4 уровня вместо 2)
2. Исключение длинных названий линий без `ref`

---

### Подзадача 1: 4-уровневый приоритет транспорта

#### Текущее состояние

**Файл**: `internal/repository/postgresosm/transport_repository.go`

В функциях `GetNearestTransportByPriority` (~строка 1146) и `GetNearestTransportByPriorityBatch` (~строка 1293) используется 2-уровневый `priority_rank`:

```sql
CASE 
    WHEN railway = 'station' AND (tags->'station' = 'subway' OR tags->'subway' = 'yes') THEN 1   -- metro
    WHEN railway IN ('station', 'halt') THEN 1                                                      -- train (тот же ранг что metro!)
    WHEN railway = 'tram_stop' THEN 2                                                              -- tram
    WHEN highway = 'bus_stop' OR public_transport IN ('platform', 'stop_position') THEN 2          -- bus
    ELSE 3
END AS priority_rank
```

**Проблема**: metro и train имеют одинаковый `priority_rank = 1`. Требуется 4 уровня.

#### Как исправить

Изменить `CASE` выражение на 4 уровня:

```sql
CASE 
    WHEN railway = 'station' AND (tags->'station' = 'subway' OR tags->'subway' = 'yes') THEN 1   -- metro (высший приоритет)
    WHEN railway IN ('station', 'halt') THEN 2                                                      -- train/cercania (R2, FGC и т.д.)
    WHEN railway = 'tram_stop' THEN 3                                                              -- tram
    WHEN highway = 'bus_stop' OR public_transport IN ('platform', 'stop_position') THEN 4          -- bus (низший приоритет)
    ELSE 5
END AS priority_rank
```

#### Алгоритм fill-up

Текущий алгоритм (CTE chain) работает с 2 группами: `high_priority` (rank=1) и `low_priority` (rank=2).

Нужно переделать на 4 группы, но **не 4 отдельных CTE** — это неэффективно. Вместо этого использовать **единый ранжированный подход**:

```sql
WITH all_stations AS (
    -- Существующий CTE с 4-уровневым priority_rank
    ...
),
ranked_stations AS (
    SELECT *,
        ROW_NUMBER() OVER (
            ORDER BY priority_rank ASC, distance ASC
        ) AS global_rank
    FROM all_stations
)
SELECT * FROM ranked_stations
WHERE global_rank <= $limit
```

Это обеспечит: сначала все metro, потом train, потом tram, потом bus — в пределах лимита, отсортированные по расстоянию внутри каждого ранга.

**Для batch-варианта** (`GetNearestTransportByPriorityBatch`):
```sql
ranked_stations AS (
    SELECT *,
        ROW_NUMBER() OVER (
            PARTITION BY point_idx
            ORDER BY priority_rank ASC, distance ASC
        ) AS global_rank
    FROM all_stations
)
SELECT * FROM ranked_stations
WHERE global_rank <= $limit
```

**Преимущество**: Один CTE вместо двух (high/low) — проще и производительнее. Тот же принцип: заполняем слоты от высшего приоритета к низшему.

#### Обновление в обеих функциях

Изменения нужно сделать в **двух местах**:
1. `GetNearestTransportByPriority` (~строка 1146) — single-point вариант
2. `GetNearestTransportByPriorityBatch` (~строка 1293) — batch вариант

#### Константы

В `internal/domain/transport_types.go` добавить константы приоритетов:

```go
const (
    TransportPriorityMetro      = 1
    TransportPriorityTrain      = 2
    TransportPriorityTram       = 3
    TransportPriorityBus        = 4
    TransportPriorityUnknown    = 5
)
```

---

### Подзадача 2: Исключение длинных/описательных названий линий

#### Текущее состояние

**Файл**: `internal/repository/postgresosm/transport_repository.go`, функция `GetLinesByStationIDsBatch` (~строка 1530)

SQL:
```sql
SELECT DISTINCT ON (sp.osm_id, COALESCE(NULLIF(l.ref, ''), l.name))
    sp.osm_id AS station_id,
    l.osm_id AS line_id,
    COALESCE(l.ref, l.name, '') AS name,     -- ← fallback на длинное имя!
    COALESCE(l.ref, '') AS ref,
    ...
FROM station_points sp
JOIN planet_osm_line l ON ST_DWithin(l.way, sp.way, 100)
WHERE l.route IN ('subway', 'light_rail', 'train', 'tram', 'bus')
  AND (l.ref IS NOT NULL AND l.ref != '' OR l.name IS NOT NULL AND l.name != '')
```

**Проблема**: Линии без `ref` — это дублирующие записи. Их нужно полностью исключить. Пример дубля: `"602 Figueres - Girona - Barcelona - Aeroport del Prat"` (type: bus, ref: NULL).

#### Как исправить

Полностью исключить линии без `ref` из результатов. Линии без `ref` — дубли, не несут пользы для UI.

**SQL в `GetLinesByStationIDsBatch`**:

Изменить WHERE условие:
```sql
-- Было:
WHERE l.route IN ('subway', 'light_rail', 'train', 'tram', 'bus')
  AND (l.ref IS NOT NULL AND l.ref != '' OR l.name IS NOT NULL AND l.name != '')

-- Стало:
WHERE l.route IN ('subway', 'light_rail', 'train', 'tram', 'bus')
  AND l.ref IS NOT NULL AND l.ref != ''    -- только линии с ref
```

Изменить SELECT:
```sql
-- Было:
COALESCE(l.ref, l.name, '') AS name,
COALESCE(l.ref, '') AS ref,

-- Стало:
l.ref AS name,     -- ref = name (всегда есть, т.к. фильтруем в WHERE)
l.ref AS ref,
```

Обновить DISTINCT ON:
```sql
-- Было:
DISTINCT ON (sp.osm_id, COALESCE(NULLIF(l.ref, ''), l.name))

-- Стало:
DISTINCT ON (sp.osm_id, l.ref)
```

**Также обновить `GetLinesByStationID`** (single-point вариант, если есть) — аналогичные изменения.

#### Go-side dedup

В Go-коде после SQL (~строки 1580-1610) есть дедупликация:
```go
seenLines := make(map[int64]map[string]bool) // station_id -> ref -> seen
```

Добавить guard на пустой ref (на случай если в БД есть аномалии):
```go
if line.Ref == "" {
    continue // пропустить линии без ref — это дубли
}
```

---

### Подзадача 3: Обновление hasHighPriority логики

**Файл**: `internal/usecase/transport_usecase.go` (~строка 270-280)

**Текущий код**:
```go
hasHighPriority := false
priorityType := "bus/tram"
for _, s := range stations {
    if s.Type == "metro" || s.Type == "train" {
        hasHighPriority = true
        priorityType = "metro/train"
        break
    }
}
```

**Обновить** для 4-уровневой системы:

```go
func determinePriorityMeta(stations []dto.PriorityTransportStation) (bool, string) {
    highestPriority := TransportPriorityUnknown
    for _, s := range stations {
        switch s.Type {
        case domain.TransportTypeMetro:
            if TransportPriorityMetro < highestPriority {
                highestPriority = TransportPriorityMetro
            }
        case domain.TransportTypeTrain, domain.TransportTypeCercania:
            if TransportPriorityTrain < highestPriority {
                highestPriority = TransportPriorityTrain
            }
        case domain.TransportTypeTram:
            if TransportPriorityTram < highestPriority {
                highestPriority = TransportPriorityTram
            }
        case domain.TransportTypeBus:
            if TransportPriorityBus < highestPriority {
                highestPriority = TransportPriorityBus
            }
        }
    }
    
    hasHigh := highestPriority <= TransportPriorityTrain // metro или train
    priorityLabels := map[int]string{
        TransportPriorityMetro:   "metro",
        TransportPriorityTrain:   "train",
        TransportPriorityTram:    "tram",
        TransportPriorityBus:     "bus",
        TransportPriorityUnknown: "unknown",
    }
    return hasHigh, priorityLabels[highestPriority]
}
```

Использовать в месте где сейчас стоит цикл определения `hasHighPriority`.

**Важно**: Не сломать поведение `PriorityTransportMeta` — поля `HasHighPriority` и `PriorityType` используются фронтендом.

---

## Файлы для изменения (итого)

| Файл | Что изменить |
|------|--------------|
| `internal/repository/postgresosm/transport_repository.go` | 1) `priority_rank` CASE → 4 уровня в обеих функциях (Priority/PriorityBatch) 2) `GetLinesByStationIDsBatch` — ограничить длину name |
| `internal/repository/postgresosm/constants.go` | Добавить `MaxLineNameLength = 20` |
| `internal/domain/transport_types.go` | Добавить константы приоритетов: `TransportPriorityMetro..Bus` |
| `internal/usecase/transport_usecase.go` | Обновить `hasHighPriority` логику для 4 уровней |

---

## Тестирование

### Существующие тесты
- `internal/repository/postgresosm/transport_repository_test.go` (540 строк) — интеграционные тесты с OSM DB
- `internal/usecase/transport_usecase_test.go` — unit тесты usecase
- `internal/usecase/enriched_location_usecase_test.go` — тесты обогащения

### Что добавить/обновить

1. В `transport_repository_test.go` — тест с координатами Барселоны (`lat=41.3851, lon=2.1734`):
```go
func TestGetNearestTransportByPriority_4LevelPriority(t *testing.T) {
    // Проверить что metro имеет приоритет над train
    // Проверить что при наличии metro — он возвращается первым
    // Проверить что bus/tram возвращаются только когда нет metro/train
}
```

2. В `transport_repository_test.go` — тест на фильтрацию длинных имён:
```go
func TestGetLinesByStationIDsBatch_FiltersLongNames(t *testing.T) {
    // Проверить что линии с ref возвращаются
    // Проверить что линии без ref с коротким name возвращаются
    // Проверить что линии без ref с длинным name (>20 символов) НЕ возвращаются
}
```

3. В `transport_usecase_test.go` — тест `determinePriorityMeta`:
```go
func TestDeterminePriorityMeta(t *testing.T) {
    tests := []struct{
        name     string
        types    []string
        wantHigh bool
        wantType string
    }{
        {"metro only", []string{"metro"}, true, "metro"},
        {"train only", []string{"train"}, true, "train"},
        {"tram only", []string{"tram"}, false, "tram"},
        {"bus only", []string{"bus"}, false, "bus"},
        {"metro + bus", []string{"bus", "metro"}, true, "metro"},
        {"train + tram", []string{"tram", "train"}, true, "train"},
    }
    ...
}
```

### Запуск тестов
```bash
cd /home/sergio/realbro/location
go test ./internal/repository/postgresosm/ -run TestGetNearestTransport -v
go test ./internal/usecase/ -run TestDeterminePriority -v
```

---

## Важные правила

1. **Производительность SQL**: Не добавлять лишних CTE. Уменьшить количество CTE если можно (2 группы → 1 ranked). Использовать `ROW_NUMBER() OVER (ORDER BY priority_rank, distance)` — PostGIS оптимизирует это эффективно
2. **Не хардкодить**: Длина имени (`MaxLineNameLength`), приоритеты — через константы
3. **Совместимость**: `priority_rank` обратно совместим — результат для фронтенда не меняет формат, только порядок
4. **geography vs geometry**: Использовать `way_geog` (pre-computed geography) для `ST_DWithin` — НЕ делать runtime cast `::geography`
5. **Batch-first**: Изменения в single-point варианте дублировать в batch
6. **Nullable fields**: Указатели (`*string`, `*float64`) для опциональных полей в domain
7. **Логирование**: `zap.Logger`, structured — `zap.Int64("station_id", id)`, `zap.String("type", stationType)`
