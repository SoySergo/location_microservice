# Location Enrichment Worker

This worker processes location enrichment events from Redis Streams, resolves location hierarchies, finds nearest transport, and publishes results.

## Overview

The worker subscribes to `stream:location:enrich` from backend_estate, processes location data, and publishes enriched results to `stream:location:done`.

## Architecture

```
Redis Stream (stream:location:enrich)
          ↓
LocationEnrichmentWorker
          ↓
   EnrichmentUseCase
    ↓            ↓
BoundaryRepo  TransportRepo
          ↓
Redis Stream (stream:location:done)
```

## Configuration

Configure the worker using environment variables in `.env`:

```bash
# Worker Configuration
WORKER_ENABLED=true
WORKER_CONSUMER_GROUP=location-enrichment-workers
WORKER_STREAM_READ_TIMEOUT=5000  # milliseconds
WORKER_MAX_RETRIES=3
WORKER_TRANSPORT_RADIUS=1000     # meters
WORKER_TRANSPORT_TYPES=metro,train,tram,bus

# Redis Cache (local) - for caching tiles, search results
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Redis Streams (shared with backend_estate) - for location enrichment events
REDIS_STREAMS_HOST=localhost
REDIS_STREAMS_PORT=6380
REDIS_STREAMS_PASSWORD=
REDIS_STREAMS_DB=0
```

## Redis Architecture

The microservice uses **two separate Redis instances**:

1. **Redis Cache (local)**: Used for caching tiles, search results, and other local data
   - Default port: 6379
   - Used by: API server and internal caching

2. **Redis Streams (shared)**: Dedicated Redis instance shared with backend_estate
   - Default port: 6380
   - Used by: Worker for reading `stream:location:enrich` and writing `stream:location:done`
   - This separation ensures stream processing is isolated from cache operations

## Building

```bash
# Build worker binary
make build-worker

# Build both API and worker
make build
```

## Running

### Development
```bash
# Run worker directly
make run-worker

# Or with go run
go run cmd/worker/main.go
```

### Production
```bash
# Run compiled binary
./bin/worker
```

## How It Works

### 1. Location Resolution

The worker resolves locations using a hierarchical strategy:

1. **Name-based lookup**: Starting from the most specific level (Neighborhood → District → City → Province → Region)
2. **Reverse geocoding**: If coordinates are provided but no names
3. **Fallback**: Country-only lookup if nothing else works

### 2. Transport Lookup

For properties with coordinates, the worker finds nearby transport stations:
- Searches within configurable radius (default: 1000m)
- Supports multiple transport types: metro, train, tram, bus
- Returns up to 10 nearest stations with distances

### 3. Event Processing

**Input Event** (`stream:location:enrich`):
```json
{
  "property_id": "123e4567-e89b-12d3-a456-426614174000",
  "country": "Spain",
  "region": "Catalonia",
  "city": "Barcelona",
  "district": "Eixample",
  "latitude": 41.3851,
  "longitude": 2.1734
}
```

**Output Event** (`stream:location:done`):
```json
{
  "property_id": "123e4567-e89b-12d3-a456-426614174000",
  "enriched_location": {
    "country_id": 1,
    "region_id": 10,
    "city_id": 100,
    "district_id": 1001,
    "is_address_visible": true
  },
  "nearest_transport": [
    {
      "station_id": 500,
      "name": "Passeig de Gràcia",
      "type": "metro",
      "distance": 350.5,
      "line_ids": [3, 4]
    }
  ]
}
```

## Graceful Shutdown

The worker supports graceful shutdown via:
- SIGINT (Ctrl+C)
- SIGTERM (from orchestration systems)

All in-flight messages are completed before shutdown.

## Monitoring

The worker logs all events:
- Message processing (INFO level)
- Errors and retries (ERROR level)
- Debug information (DEBUG level)

Set `LOG_LEVEL` environment variable to control logging:
```bash
LOG_LEVEL=debug    # Detailed debugging
LOG_LEVEL=info     # Normal operation (default)
LOG_LEVEL=warn     # Warnings and errors only
LOG_LEVEL=error    # Errors only
```

## Scaling

Multiple worker instances can run concurrently:
- All workers share the same consumer group
- Redis Streams automatically distributes messages
- Each message is processed by exactly one worker

Example with Docker:
```bash
docker-compose up --scale worker=3
```

## Error Handling

- **Malformed messages**: Logged and skipped (ACK'd to prevent reprocessing)
- **Location not found**: Error message published to `stream:location:done`
- **Transport lookup failures**: Non-critical, logged as warnings
- **Database errors**: Message not ACK'd, will be retried

## Testing

The worker requires:
- PostgreSQL with location data
- Redis for streams (shared Redis on port 6380)

```bash
# Start test infrastructure
docker-compose up -d postgres osm_db shared_redis

# Run worker
make run-worker
```

### Testing with test_publish.go

A test script is provided to simulate location enrichment events from backend_estate:

```bash
# Publish a test event to shared Redis
make test-publish

# Or with custom Redis address
make test-publish-custom REDIS_STREAMS_ADDR=localhost:6380

# Check stream status
make check-streams
```

The test script will:
1. Connect to shared Redis (default: localhost:6380)
2. Publish a test location event to `stream:location:enrich`
3. Wait for the worker to process it
4. Display the enriched result from `stream:location:done`

Example output:
```
✅ Event published successfully!
   Stream: stream:location:enrich
   Message ID: 1234567890123-0
   Property ID: 123e4567-e89b-12d3-a456-426614174000
   Location: Barcelona, España
   Coordinates: 41.402704, 2.159956

⏳ Waiting for response in stream:location:done...

✅ Response received!
{
  "property_id": "123e4567-e89b-12d3-a456-426614174000",
  "enriched_location": { ... },
  "nearest_transport": [ ... ]
}
```

## Troubleshooting

### Worker doesn't start
- Check `WORKER_ENABLED=true` in `.env`
- Verify Redis Streams connection (port 6380)
- Verify PostgreSQL connection

### No messages processed
- Check Redis stream exists: `redis-cli -p 6380 XINFO STREAM stream:location:enrich`
- Verify consumer group: `redis-cli -p 6380 XINFO GROUPS stream:location:enrich`
- Check backend_estate is publishing messages to the correct Redis instance (shared_redis)
- Use `make check-streams` to verify stream status

### Location not found errors
- Verify database has location data
- Check country name matches database
- Enable debug logging to see search queries
