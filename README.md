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
