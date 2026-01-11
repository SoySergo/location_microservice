.PHONY: help build run migrate-up migrate-down migrate-force test lint clean dev run-api test-health test-endpoints
.PHONY: test-db-up test-db-down test-db-reset test-integration test-integration-coverage
.PHONY: build-worker run-worker
.PHONY: build-importer run-importer import-osm build-boundary-importer run-boundary-importer
.PHONY: swagger swagger-serve
.PHONY: test-publish test-publish-custom check-streams

DB_DSN := "postgres://postgres:postgres@localhost:5434/location_microservice?sslmode=disable"
TEST_DB_DSN := "postgres://postgres:postgres@localhost:5433/location_test?sslmode=disable"

help:
	@echo "Available commands:"
	@echo "  make build        - Build the API server"
	@echo "  make run          - Run the API server"
	@echo "  make dev          - Start Docker, apply migrations, run server"
	@echo "  make run-api      - Run API server (alias for run)"
	@echo "  make build-worker - Build the worker"
	@echo "  make run-worker   - Run the worker"
	@echo "  make migrate-up   - Apply database migrations"
	@echo "  make migrate-down - Rollback migrations"
	@echo "  make migrate-force- Force migration version"
	@echo "  make test         - Run tests"
	@echo "  make test-health  - Test health endpoint"
	@echo "  make test-endpoints - Test all endpoints"
	@echo "  make lint         - Run linter"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "OSM Importer commands:"
	@echo "  make build-importer - Build OSM importer binary"
	@echo "  make run-importer   - Run OSM importer"
	@echo "  make import-osm     - Full pipeline: build, migrate, import OSM data"
	@echo ""
	@echo "Boundary Importer commands:"
	@echo "  make build-boundary-importer - Build boundary importer binary"
	@echo "  make run-boundary-importer   - Run boundary importer"
	@echo ""
	@echo "Swagger commands:"
	@echo "  make swagger       - Generate Swagger documentation"
	@echo "  make swagger-serve - Serve Swagger UI in Docker"
	@echo ""
	@echo "Test database commands:"
	@echo "  make test-db-up   - Start test database"
	@echo "  make test-db-down - Stop test database"
	@echo "  make test-db-reset- Reset test database (remove all data)"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-integration-coverage - Run integration tests with coverage"

build:
	go build -o bin/api cmd/api/main.go

build-worker:
	go build -o bin/worker cmd/worker/main.go

run:
	go run cmd/api/main.go

run-worker:
	go run cmd/worker/main.go

dev:
	@echo "Starting Docker Compose services..."
	docker-compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 5
	@echo "Applying migrations..."
	make migrate-up
	@echo "Starting API server..."
	go run cmd/api/main.go

run-api:
	go run cmd/api/main.go

migrate-up:
	migrate -path migrations -database $(DB_DSN) up

migrate-down:
	migrate -path migrations -database $(DB_DSN) down

migrate-force:
	@echo "Usage: make migrate-force VERSION=1"
	migrate -path migrations -database $(DB_DSN) force $(VERSION)

test:
	go test -v ./...

test-health:
	@echo "Testing health endpoint..."
	@curl -s http://localhost:8080/api/v1/health | jq

test-endpoints:
	@echo "\n=== Testing Health ==="
	@curl -s http://localhost:8080/api/v1/health | jq
	@echo "\n=== Testing Search ==="
	@curl -s "http://localhost:8080/api/v1/search?q=barcelona&language=en&limit=5" | jq
	@echo "\n=== Testing Reverse Geocode ==="
	@curl -s -X POST http://localhost:8080/api/v1/reverse-geocode \
		-H "Content-Type: application/json" \
		-d '{"lat":41.3851,"lon":2.1734}' | jq
	@echo "\n=== Testing Nearest Transport ==="
	@curl -s -X POST http://localhost:8080/api/v1/transport/nearest \
		-H "Content-Type: application/json" \
		-d '{"lat":41.3851,"lon":2.1734,"types":["metro"],"max_distance":2000}' | jq

lint:
	golangci-lint run

clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Test database commands
test-db-up:
	@echo "Starting test database..."
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Test database is ready on port 5433"

test-db-down:
	@echo "Stopping test database..."
	docker-compose -f docker-compose.test.yml down

test-db-reset:
	@echo "Resetting test database (removing all data)..."
	docker-compose -f docker-compose.test.yml down -v
	@echo "Starting fresh test database..."
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for database to be ready..."
	@sleep 5
	@echo "Test database reset complete"

test-integration: test-db-up
	@echo "Running integration tests..."
	go test -v ./internal/repository/postgres/...

test-integration-coverage: test-db-up
	@echo "Running integration tests with coverage..."
	go test -v -cover -coverprofile=coverage.out ./internal/repository/postgres/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# OSM Importer commands
build-importer:
	@echo "Building OSM importer..."
	go build -o bin/importer cmd/importer/main.go cmd/importer/config.go
	@echo "Importer built successfully: bin/importer"

run-importer:
	@echo "Running OSM importer..."
	go run cmd/importer/main.go cmd/importer/config.go

import-osm:
	@echo "Starting full OSM import pipeline..."
	@echo "1. Building importer..."
	@make build-importer
	@echo "2. Starting database..."
	docker-compose up -d postgres
	@echo "3. Waiting for database to be ready..."
	@sleep 5
	@echo "4. Applying migrations..."
	@make migrate-up
	@echo "5. Running importer..."
	./bin/importer
	@echo "OSM import completed successfully!"

# Boundary Importer commands
build-boundary-importer:
	@echo "Building boundary importer..."
	go build -o bin/boundary-importer cmd/boundary-importer/*.go
	@echo "Boundary importer built successfully: bin/boundary-importer"

run-boundary-importer:
	@echo "Running boundary importer..."
	@echo "Usage: ADMIN_LEVELS=2,4,6,8 OSM_FILE_PATH=../test/osm_data/cataluna-251111.osm.pbf make run-boundary-importer"
	go run cmd/boundary-importer/*.go

# Swagger commands
swagger:
	@echo "Generating Swagger documentation..."
	@if [ -f ~/go/bin/swag ]; then \
		~/go/bin/swag init -g cmd/api/main.go -o docs/swagger --parseDependency --parseInternal; \
	elif command -v swag >/dev/null 2>&1; then \
		swag init -g cmd/api/main.go -o docs/swagger --parseDependency --parseInternal; \
	else \
		echo "Error: swag is not installed. Install it with: go install github.com/swaggo/swag/cmd/swag@latest"; \
		exit 1; \
	fi
	@echo "Swagger documentation generated in docs/swagger/"

swagger-serve:
	@echo "Starting Swagger UI on http://localhost:8081"
	docker run -p 8081:8080 -e SWAGGER_JSON=/docs/swagger.json -v $(PWD)/docs/swagger:/docs swaggerapi/swagger-ui

# Redis Streams testing
test-publish:
	go run scripts/test_publish.go -redis=localhost:6380

test-publish-custom:
	@if [ -z "$(REDIS_STREAMS_ADDR)" ]; then \
		echo "Usage: make test-publish-custom REDIS_STREAMS_ADDR=localhost:6380"; \
		exit 1; \
	fi
	go run scripts/test_publish.go -redis=$(REDIS_STREAMS_ADDR)

check-streams:
	@echo "=== stream:location:enrich ==="
	@redis-cli -p 6380 XINFO STREAM stream:location:enrich 2>/dev/null || echo "Stream does not exist"
	@echo ""
	@echo "=== stream:location:done ==="
	@redis-cli -p 6380 XINFO STREAM stream:location:done 2>/dev/null || echo "Stream does not exist"
