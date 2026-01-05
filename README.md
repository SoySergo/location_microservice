# Location Microservice

Геопространственный микросервис для работы с OSM данными Каталонии.

## Quick Start

### API Server
1. `docker-compose up -d` - запуск PostgreSQL + Redis
2. `cp .env.example .env` - настроить переменные окружения
3. `make migrate-up` - применить миграции
4. `go run cmd/importer/main.go` - импорт OSM данных
5. `go run cmd/api/main.go` - запуск API сервера

### Worker
1. Убедитесь, что PostgreSQL и Redis запущены
2. Настройте `WORKER_ENABLED=true` в `.env`
3. `make run-worker` - запуск воркера обогащения локаций

## Components

- **API Server** (`cmd/api`): REST API для поиска локаций, транспорта, POI
- **Worker** (`cmd/worker`): Обработчик событий обогащения локаций из Redis Streams

## Documentation

- API Documentation: `docs/INDEX.md`
- Worker Documentation: `docs/WORKER.md`
