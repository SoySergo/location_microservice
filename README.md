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
