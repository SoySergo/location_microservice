# Location Microservice

Геопространственный микросервис для работы с OSM данными Каталонии.

---

## Полная инструкция по развёртыванию (с нуля)

### Prerequisites

- Docker и Docker Compose
- Go 1.24.2+
- `migrate` CLI — `go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest`

---

### Шаг 1. Скачать OSM данные

Скачиваем `.osm.pbf` файл для нужного региона с Geofabrik:

```bash
# Каталония
wget -P osm_data/ https://download.geofabrik.de/europe/spain/cataluna-latest.osm.pbf
```

> Файл должен лежать в `osm_data/`. Если имя файла отличается от указанного в `docker-compose.yml`,
> обновите путь в `docker-compose.yml` → сервис `osm2pgsql` → `command` (последняя строка).

---

### Шаг 2. Запустить все сервисы Docker

```bash
# Запуск PostgreSQL (основная БД), Redis (кэш), Redis (стримы), OSM PostgreSQL
docker-compose up -d
```

Подождать ~10 сек, проверить что всё поднялось:

```bash
docker-compose ps
```

Должны быть healthy:

| Контейнер | Порт | Назначение |
|-----------|------|------------|
| `location_microservice_db` | 5436 | Основная БД (миграции, POI) |
| `location_redis` | 6382 | Кэш (тайлы, поиск) |
| `shared_redis` | 6381 | Redis Streams (обмен с backend_estate) |
| `osm_postgres` | 5435 | OSM данные (границы, транспорт, POI) |

---

### Шаг 3. Импорт OSM данных через osm2pgsql

```bash
docker-compose --profile osm up osm2pgsql
```

Это запустит `osm2pgsql` который:
- Прочитает `.osm.pbf` файл
- Создаст таблицы `planet_osm_point`, `planet_osm_line`, `planet_osm_polygon`, `planet_osm_roads`
- Используя style-файл `scripts/default.style` (определяет колонки: `admin_level`, `public_transport`, `railway`, `highway`, `amenity`, `shop` и т.д.)

> **Важно!** Style-файл `scripts/default.style` определяет какие OSM-теги станут колонками таблиц.
> Если тега нет в style — данные попадут только в hstore `tags`, а SQL-запросы не найдут колонку.
> Сейчас в style уже есть все нужные теги — не трогай его без необходимости.

Импорт Каталонии занимает ~3-5 минут. Дождаться завершения контейнера `osm2pgsql_loader`.

---

### Шаг 4. Post-import: колонки way_geog + индексы

**Это обязательный шаг!** Без него не будут работать запросы транспорта и границ.

```bash
make osm-post-import
```

Или вручную:

```bash
docker cp scripts/post-import.sql osm_postgres:/tmp/post-import.sql
docker exec osm_postgres psql -U osmuser -d osm -f /tmp/post-import.sql
```

Что делает `post-import.sql`:
- Добавляет колонку `way_geog` (geography) в `planet_osm_point` и `planet_osm_line` — предвычисленная WGS84-версия geometry `way`
- Создаёт 19 индексов (GIST spatial, B-tree, trigram для поиска)
- Запускает `ANALYZE` для оптимизатора запросов

> На Каталонии занимает ~2-3 минуты (1.3M точек, сотни тысяч линий).

Проверить что всё создалось:

```bash
# Должно быть 19 индексов
docker exec osm_postgres psql -U osmuser -d osm -c \
  "SELECT COUNT(*) FROM pg_indexes WHERE tablename IN ('planet_osm_point','planet_osm_line','planet_osm_polygon');"

# Должна быть колонка way_geog
docker exec osm_postgres psql -U osmuser -d osm -c \
  "SELECT column_name FROM information_schema.columns WHERE table_name='planet_osm_point' AND column_name='way_geog';"
```

---

### Шаг 5. Применить миграции основной БД

```bash
make migrate-up
```

---

### Шаг 6. Настроить окружение

```bash
cp .env.example .env
# Отредактировать .env при необходимости
```

---

### Шаг 7. Запуск

```bash
# API сервер
make run

# Worker (в отдельном терминале)
make run-worker
```

---

### Быстрая команда (шаги 3-4 одной командой)

```bash
make osm-setup
```

Запустит: OSM PostgreSQL → osm2pgsql импорт → post-import (way_geog + индексы).

---

## Обновление OSM данных

При необходимости обновить OSM данные (новая выгрузка):

```bash
# 1. Скачать свежий .osm.pbf в osm_data/
wget -P osm_data/ https://download.geofabrik.de/europe/spain/cataluna-latest.osm.pbf

# 2. Обновить имя файла в docker-compose.yml (сервис osm2pgsql → command)

# 3. Удалить старый volume и переимпортировать
docker-compose --profile osm down osm2pgsql
docker-compose down osm_db
docker volume rm location_osm_data

# 4. Полный пайплайн заново
make osm-setup
```

---

## Структура сервисов

| Компонент | Описание |
|-----------|----------|
| **API Server** (`cmd/api`) | REST API: поиск, reverse geocode, транспорт, POI, тайлы |
| **Worker** (`cmd/worker`) | Обработчик `stream:location:enrich` → обогащает локации → `stream:location:done` |

### Архитектура Redis

Два отдельных Redis:

1. **Redis Cache** (порт 6382) — кэш тайлов, результатов поиска
2. **Redis Streams** (порт 6381) — общий с backend_estate для событий обогащения
   - Читает: `stream:location:enrich`
   - Пишет: `stream:location:done`

### Тестирование Worker

```bash
# Отправить тестовое событие
make test-publish

# Проверить статус стримов
make check-streams
```

---

## Swagger Documentation

```bash
# Генерация документации
make swagger

# Просмотр: http://localhost:8080/swagger/ (при запущенном API)

# Swagger UI без API сервера
make swagger-serve   # http://localhost:8086
```

---

## Частые проблемы

### `column "public_transport" does not exist` / `column "admin_level" does not exist`

OSM данные импортированы без нужных колонок. Нужно переимпортировать:

```bash
docker-compose --profile osm down osm2pgsql && docker-compose down osm_db
docker volume rm location_osm_data
make osm-setup
```

> Причина: style-файл `scripts/default.style` не содержал эти теги. Сейчас они добавлены.

### `column "way_geog" does not exist`

Не выполнен post-import скрипт:

```bash
make osm-post-import
```

### Worker не подключается к Redis Streams

Проверить что `shared_redis` запущен:

```bash
docker-compose up -d shared_redis
```

---

## Make targets

```
make help                   — полный список команд
make run                    — запуск API
make run-worker             — запуск Worker
make dev                    — Docker + миграции + API
make osm-setup              — полный пайплайн OSM (import + post-import)
make osm-post-import        — только post-import (way_geog + индексы)
make migrate-up / down      — миграции
make test-publish           — тестовое событие в Redis Streams
make check-streams          — статус стримов
make swagger                — генерация Swagger docs
```
