# Mapbox Integration for Location Enrichment

## Overview

This feature extends location enrichment with infrastructure data including:
- Transport stations (metro, train, tram, bus) with walking distances
- Points of Interest (POIs) like shops, pharmacies, schools, parks
- Walking distances and durations calculated via Mapbox Matrix API

## Configuration

### Environment Variables

Add these to your `.env` file:

```bash
# Mapbox Configuration
MAPBOX_ACCESS_TOKEN=your_mapbox_access_token_here
MAPBOX_BASE_URL=https://api.mapbox.com
MAPBOX_MAX_MATRIX_POINTS=25
MAPBOX_WALKING_PROFILE=mapbox/walking
MAPBOX_REQUEST_TIMEOUT=30

# Extended Worker Configuration
WORKER_INFRASTRUCTURE_ENABLED=true
WORKER_MAX_METRO=3
WORKER_MAX_TRAIN=2
WORKER_MAX_TRAM=2
WORKER_MAX_BUS=2
WORKER_POI_RADIUS=1500
```

### Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `MAPBOX_ACCESS_TOKEN` | Your Mapbox API token | (required) |
| `MAPBOX_BASE_URL` | Mapbox API base URL | `https://api.mapbox.com` |
| `MAPBOX_MAX_MATRIX_POINTS` | Maximum coordinates per request | `25` |
| `MAPBOX_WALKING_PROFILE` | Routing profile | `mapbox/walking` |
| `MAPBOX_REQUEST_TIMEOUT` | HTTP request timeout (seconds) | `30` |
| `WORKER_INFRASTRUCTURE_ENABLED` | Enable/disable infrastructure enrichment | `false` |
| `WORKER_MAX_METRO` | Maximum metro stations | `3` |
| `WORKER_MAX_TRAIN` | Maximum train stations | `2` |
| `WORKER_MAX_TRAM` | Maximum tram stations | `2` |
| `WORKER_MAX_BUS` | Maximum bus stations | `2` |
| `WORKER_POI_RADIUS` | POI search radius (meters) | `1500` |

## How It Works

### Conditional Enrichment

The system uses conditional logic based on address completeness:

1. **Without street address** (only city/district):
   - Basic enrichment: resolves country, region, city, etc.
   - No transport or POI lookup
   
2. **With street address** (street + house number):
   - Full enrichment with infrastructure
   - Finds nearby transport stations
   - Finds nearby POIs
   - Calculates walking distances via Mapbox

### Transport Grouping

Transport stations are grouped by normalized name to eliminate duplicates:
- Multiple exits of the same metro station count as one station
- Normalization removes special characters and converts to lowercase
- Example: "Sagrada Família (Exit A)" and "Sagrada Familia (Exit B)" → 1 station

### Point Balancing

Mapbox Matrix API has a 25-coordinate limit (1 source + 24 destinations):
- **Priority**: Transport stations (up to 9)
- **Remaining**: POIs (typically 15)
- Typical scenario: 2-3 metro + 1 bus + 16 POIs = 20 total points

### POI Categories

The system searches for these POI categories:
- **Shops**: Supermarkets (3), Convenience stores (2)
- **Healthcare**: Pharmacies (2), Hospitals (2)
- **Education**: Schools (3), Kindergartens (2)
- **Leisure**: Parks (2), Playgrounds (1)

## API Response Format

### Basic Enrichment (without street address)

```json
{
  "property_id": "uuid",
  "enriched_location": {
    "country_id": 123,
    "region_id": 456,
    "city_id": 789,
    "is_address_visible": false
  },
  "nearest_transport": [
    {
      "station_id": 1001,
      "name": "Sagrada Família",
      "type": "metro",
      "distance": 850.5,
      "line_ids": [2, 5]
    }
  ]
}
```

### Extended Enrichment (with street address)

```json
{
  "property_id": "uuid",
  "enriched_location": {
    "country_id": 123,
    "region_id": 456,
    "city_id": 789,
    "is_address_visible": true
  },
  "nearest_transport": [...],
  "infrastructure": {
    "transport": [
      {
        "station_id": 1001,
        "name": "Sagrada Família",
        "type": "metro",
        "lat": 41.4036,
        "lon": 2.1744,
        "line_ids": [2, 5],
        "linear_distance": 850.5,
        "walking_distance": 920.3,
        "walking_duration": 660.0
      }
    ],
    "pois": [
      {
        "id": 5001,
        "name": "Carrefour Express",
        "category": "shop",
        "subcategory": "supermarket",
        "lat": 41.4020,
        "lon": 2.1750,
        "linear_distance": 450.2,
        "walking_distance": 520.5,
        "walking_duration": 380.0
      }
    ],
    "walking_distances": {
      "transport_1001": 920.3,
      "poi_5001": 520.5
    }
  }
}
```

## Data Structure

### TransportWithDistance

| Field | Type | Description |
|-------|------|-------------|
| `station_id` | int64 | Unique station ID |
| `name` | string | Station name |
| `type` | string | Transport type (metro/train/tram/bus) |
| `lat` | float64 | Latitude |
| `lon` | float64 | Longitude |
| `line_ids` | []int64 | Associated line IDs |
| `linear_distance` | float64 | Straight-line distance (meters) |
| `walking_distance` | float64 | Walking distance via Mapbox (meters) |
| `walking_duration` | float64 | Walking time via Mapbox (seconds) |

### POIWithDistance

| Field | Type | Description |
|-------|------|-------------|
| `id` | int64 | Unique POI ID |
| `name` | string | POI name |
| `category` | string | Main category |
| `subcategory` | string | Subcategory |
| `lat` | float64 | Latitude |
| `lon` | float64 | Longitude |
| `linear_distance` | float64 | Straight-line distance (meters) |
| `walking_distance` | float64 | Walking distance via Mapbox (meters) |
| `walking_duration` | float64 | Walking time via Mapbox (seconds) |

## Database Queries

### Transport Grouping Query

The infrastructure repository uses this SQL pattern to group stations:

```sql
SELECT DISTINCT ON (normalized_name) 
    id, osm_id, name, name_en, type, lat, lon, line_ids
FROM (
    SELECT 
        s.id, s.osm_id, s.name, s.name_en, s.type, s.lat, s.lon, s.line_ids,
        ST_Distance(s.geometry::geography, 
                    ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS distance,
        LOWER(REGEXP_REPLACE(s.name, '[^a-zA-Zа-яА-Я0-9]', '', 'g')) AS normalized_name
    FROM transport_stations s
    WHERE s.type = $3
      AND ST_DWithin(s.geometry::geography, 
                     ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $4)
    ORDER BY distance
) sub
ORDER BY normalized_name, distance
LIMIT $5
```

### POI Search Query

POIs are searched with category filters:

```sql
SELECT 
    p.id, p.osm_id, p.name, p.category, p.subcategory,
    p.lat, p.lon,
    ST_Distance(p.geometry::geography, 
                ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography) AS distance
FROM pois p
WHERE (p.category = $3 AND p.subcategory = $4) OR (...)
  AND ST_DWithin(p.geometry::geography, 
                 ST_SetSRID(ST_MakePoint($1, $2), 4326)::geography, $5)
ORDER BY distance
LIMIT $6
```

## Performance Considerations

1. **Mapbox API Calls**: Limited to 1 call per property enrichment
2. **Point Limit**: Maximum 25 coordinates (1 source + 24 destinations)
3. **Database Queries**: Spatial indexes on `geometry` columns for performance
4. **Caching**: Consider caching Mapbox results for frequently queried locations

## Error Handling

- **Missing Mapbox Token**: Worker logs error but continues with basic enrichment
- **Mapbox API Failure**: Falls back to linear distances without walking data
- **Database Errors**: Logged but not fatal; enrichment continues with available data
- **Empty Results**: Valid response with empty arrays for transport/POIs

## Testing

### Unit Tests

Run unit tests:
```bash
go test ./internal/domain/...
go test ./internal/infrastructure/mapbox/...
```

### Integration Tests

Requires PostgreSQL and Redis:
```bash
make test-db-up
go test ./internal/repository/postgres/...
```

### Manual Testing

1. Start services:
   ```bash
   docker-compose up -d
   ```

2. Run worker:
   ```bash
   WORKER_INFRASTRUCTURE_ENABLED=true go run cmd/worker/main.go
   ```

3. Publish test event to Redis:
   ```bash
   redis-cli XADD stream:location:enrich * data '{"property_id":"...","country":"Spain","street":"Passeig de Gracia","house_number":"123","latitude":41.3851,"longitude":2.1734}'
   ```

4. Check results:
   ```bash
   redis-cli XREAD STREAMS stream:location:done 0
   ```

## Troubleshooting

### Infrastructure not included in response

- Check `WORKER_INFRASTRUCTURE_ENABLED=true`
- Verify property has `street` and `house_number`
- Ensure coordinates are present

### No walking distances

- Verify `MAPBOX_ACCESS_TOKEN` is valid
- Check Mapbox API quota limits
- Review worker logs for API errors

### Duplicate metro exits

- Verify normalized name grouping is working
- Check database query execution
- Review station names in database

## Migration Guide

### Enabling the Feature

1. Get Mapbox access token from https://account.mapbox.com
2. Add token to `.env`: `MAPBOX_ACCESS_TOKEN=pk.xxx`
3. Enable infrastructure: `WORKER_INFRASTRUCTURE_ENABLED=true`
4. Restart worker: `systemctl restart location-worker`

### Disabling the Feature

Set `WORKER_INFRASTRUCTURE_ENABLED=false` to revert to basic enrichment.

### Monitoring

Monitor these metrics:
- Mapbox API call rate
- Enrichment processing time
- Infrastructure data completeness
- Error rates for Mapbox/DB calls
