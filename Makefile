.PHONY: help build run migrate-up migrate-down migrate-force test lint clean dev run-api test-health test-endpoints
.PHONY: test-db-up test-db-down test-db-reset test-integration test-integration-coverage

DB_DSN := "postgres://postgres:postgres@localhost:5434/location_microservice?sslmode=disable"
TEST_DB_DSN := "postgres://postgres:postgres@localhost:5433/location_test?sslmode=disable"

help:
	@echo "Available commands:"
	@echo "  make build        - Build the API server"
	@echo "  make run          - Run the API server"
	@echo "  make dev          - Start Docker, apply migrations, run server"
	@echo "  make run-api      - Run API server (alias for run)"
	@echo "  make migrate-up   - Apply database migrations"
	@echo "  make migrate-down - Rollback migrations"
	@echo "  make migrate-force- Force migration version"
	@echo "  make test         - Run tests"
	@echo "  make test-health  - Test health endpoint"
	@echo "  make test-endpoints - Test all endpoints"
	@echo "  make lint         - Run linter"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "Test database commands:"
	@echo "  make test-db-up   - Start test database"
	@echo "  make test-db-down - Stop test database"
	@echo "  make test-db-reset- Reset test database (remove all data)"
	@echo "  make test-integration - Run integration tests"
	@echo "  make test-integration-coverage - Run integration tests with coverage"

build:
	go build -o bin/api cmd/api/main.go

run:
	go run cmd/api/main.go

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
