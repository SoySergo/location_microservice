# Location Microservice

Геопространственный микросервис для работы с OSM данными Каталонии.

## Quick Start

1. `docker-compose up -d` - запуск PostgreSQL + Redis
2. `cp .env.example .env` - настроить переменные окружения
3. `make migrate-up` - применить миграции
4. `go run cmd/importer/main.go` - импорт OSM данных
5. `go run cmd/api/main.go` - запуск API сервера

## Documentation

См. `docs/INDEX.md`
