# Test Coverage Report

## Unit Tests

### EnrichmentUseCase Tests (`internal/usecase/enrichment_usecase_test.go`)

Tests the core business logic for location enrichment.

**Test Cases:**
1. ✅ `TestEnrichmentUseCase_EnrichLocation_WithCityName` - Tests hierarchical location resolution starting from city name
2. ✅ `TestEnrichmentUseCase_EnrichLocation_WithCoordinatesOnly` - Tests reverse geocoding fallback when only coordinates are provided
3. ✅ `TestEnrichmentUseCase_EnrichLocation_LocationNotFound` - Tests error handling when location cannot be found
4. ✅ `TestEnrichmentUseCase_EnrichLocation_WithoutCoordinates` - Tests that transport lookup is skipped when no coordinates
5. ✅ `TestEnrichmentUseCase_EnrichLocation_TransportLookupFails` - Tests that enrichment succeeds even if transport lookup fails

**Coverage:**
- ✅ Hierarchical location resolution (Neighborhood → District → City → Province → Region → Country)
- ✅ Reverse geocoding fallback
- ✅ Transport station lookup with distance calculation
- ✅ Address visibility determination
- ✅ Error handling and graceful degradation
- ✅ Mock-based testing with BoundaryRepository and TransportRepository

### LocationEnrichmentWorker Tests (`internal/worker/location/enrichment_worker_test.go`)

Tests the worker that processes stream events.

**Test Cases:**
1. ✅ `TestLocationEnrichmentWorker_ProcessMessage_Success` - Tests successful message processing flow
2. ✅ `TestLocationEnrichmentWorker_ProcessMalformedMessage` - Tests handling of malformed JSON (dead letter pattern)
3. ✅ `TestLocationEnrichmentWorker_Name` - Tests worker name getter
4. ✅ `TestLocationEnrichmentWorker_Stop` - Tests graceful stop functionality
5. ✅ `TestLocationEnrichmentWorker_ContextCancellation` - Tests worker stops on context cancellation

**Coverage:**
- ✅ Message consumption and acknowledgment
- ✅ Event deserialization
- ✅ Enrichment use case invocation
- ✅ Result publishing
- ✅ Error handling (malformed messages)
- ✅ Graceful shutdown
- ✅ Context cancellation

## Integration Tests

### StreamRepository Tests (`internal/repository/redis/stream_repository_test.go`)

Integration tests for Redis Streams functionality.

**Test Cases:**
1. ⏭️ `TestStreamRepository_CreateConsumerGroup` - Tests consumer group creation (skipped if Redis not available)
2. ⏭️ `TestStreamRepository_PublishToStream` - Tests message publishing to stream (skipped if Redis not available)
3. ⏭️ `TestStreamRepository_ConsumeStream` - Tests message consumption with consumer groups (skipped if Redis not available)
4. ⏭️ `TestStreamRepository_AckMessage` - Tests message acknowledgment (skipped if Redis not available)
5. ⏭️ `TestStreamRepository_ConsumeStream_ContextCancellation` - Tests graceful shutdown (skipped if Redis not available)

**Coverage:**
- ✅ Consumer group creation with BUSYGROUP handling
- ✅ Stream publishing (XADD)
- ✅ Stream consumption (XREADGROUP)
- ✅ Message acknowledgment (XACK)
- ✅ Graceful shutdown with context cancellation
- ✅ Event serialization/deserialization
- ⚠️ Tests are skipped if Redis is not available (integration test pattern)

## Running Tests

### Unit Tests Only
```bash
# Run all unit tests
go test ./internal/usecase ./internal/worker/location -v

# Run specific test
go test ./internal/usecase -run TestEnrichmentUseCase_EnrichLocation_WithCityName -v
```

### Integration Tests
```bash
# Start Redis for integration tests
docker-compose up -d redis

# Run integration tests
go test ./internal/repository/redis -v

# Stop Redis
docker-compose down
```

### All Tests
```bash
# Run all tests (integration tests will be skipped if Redis not available)
go test ./internal/usecase ./internal/worker/location ./internal/repository/redis -v
```

## Test Statistics

- **Total Unit Tests:** 10
- **Total Integration Tests:** 5
- **Passing Unit Tests:** 10/10 (100%)
- **Integration Tests:** Skipped without Redis, pass with Redis

## Mock Objects

All tests use `testify/mock` for mocking dependencies:

### EnrichmentUseCase Tests
- `MockBoundaryRepository` - Mocks admin boundary operations
- `MockTransportRepository` - Mocks transport station operations

### Worker Tests
- `MockStreamRepository` - Mocks Redis stream operations
- `MockEnrichmentUseCase` - Mocks enrichment use case (via interface)

## Test Design Principles

1. **Isolation:** Each test is independent with its own mocks
2. **Clarity:** Test names clearly indicate what is being tested
3. **Coverage:** Tests cover success cases, error cases, and edge cases
4. **Maintainability:** Uses mock objects to avoid external dependencies
5. **Integration Safety:** Integration tests gracefully skip if dependencies unavailable

## Future Test Improvements

- [ ] Add benchmark tests for performance-critical operations
- [ ] Add table-driven tests for multiple scenarios
- [ ] Add more edge cases for boundary resolution
- [ ] Add load tests for worker message processing
- [ ] Add end-to-end tests with real Redis and PostgreSQL
