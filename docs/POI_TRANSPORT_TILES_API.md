# POI and Transport Tile API Documentation

## Overview

This document describes the new API endpoints for retrieving POI (Points of Interest) and Transport data as MVT (Mapbox Vector Tile) tiles with category and type filtering.

## POI Tile Endpoints

### Get POI Tile by Categories

Retrieve a vector tile containing POIs filtered by categories and subcategories.

**Endpoint:** `GET /api/v1/tiles/poi/{z}/{x}/{y}.pbf`

**Parameters:**
- `z` (path parameter, required): Zoom level (0-18)
- `x` (path parameter, required): Tile X coordinate
- `y` (path parameter, required): Tile Y coordinate
- `categories` (query parameter, optional): Comma-separated list of categories to filter
  - Valid values: `healthcare`, `shopping`, `education`, `leisure`, `food_drink`
- `subcategories` (query parameter, optional): Comma-separated list of subcategories to filter

**Response:**
- Content-Type: `application/x-protobuf`
- Content-Encoding: `gzip`
- Cache-Control: `public, max-age=3600`
- Body: MVT tile data in Protocol Buffer format

**Layer Structure:**
The tile contains a single layer named `pois` with the following properties:
- `id` (int64): POI identifier
- `name` (string): POI name
- `name_es` (string): Spanish name
- `name_ca` (string): Catalan name
- `category` (string): POI category
- `subcategory` (string): POI subcategory
- `address` (string, optional): Address
- `phone` (string, optional): Phone number
- `website` (string, optional): Website URL
- `opening_hours` (string, optional): Opening hours

**Example Requests:**

```bash
# Get all POIs
GET /api/v1/tiles/poi/14/8192/6144.pbf

# Get only healthcare POIs
GET /api/v1/tiles/poi/14/8192/6144.pbf?categories=healthcare

# Get multiple categories
GET /api/v1/tiles/poi/14/8192/6144.pbf?categories=healthcare,shopping,education

# Get specific subcategories
GET /api/v1/tiles/poi/14/8192/6144.pbf?categories=healthcare&subcategories=pharmacy,hospital

# Get shopping with specific store types
GET /api/v1/tiles/poi/14/8192/6144.pbf?categories=shopping&subcategories=supermarket,mall
```

### POI Categories

#### Healthcare (`healthcare`)
Subcategories:
- `pharmacy` - Pharmacies
- `hospital` - Hospitals
- `clinic` - Clinics
- `doctors` - Doctor offices
- `dentist` - Dental clinics
- `veterinary` - Veterinary clinics

OSM Mapping:
- `amenity=pharmacy` → pharmacy
- `amenity=hospital` → hospital
- `amenity=clinic` → clinic
- `amenity=doctors` → doctors
- `amenity=dentist` → dentist
- `amenity=veterinary` → veterinary
- `healthcare=*` → value becomes subcategory

#### Shopping (`shopping`)
Subcategories:
- `supermarket` - Supermarkets
- `convenience` - Convenience stores
- `mall` - Shopping malls
- `grocery` - Grocery stores
- `department_store` - Department stores
- `bakery` - Bakeries
- `butcher` - Butcher shops
- `greengrocer` - Fruit and vegetable shops

OSM Mapping:
- `shop=supermarket` → supermarket
- `shop=convenience` → convenience
- `shop=mall` → mall
- `shop=grocery` → grocery
- `shop=department_store` → department_store
- `shop=bakery` → bakery
- `shop=butcher` → butcher
- `shop=greengrocer` → greengrocer

#### Education (`education`)
Subcategories:
- `school` - Schools
- `kindergarten` - Kindergartens
- `college` - Colleges
- `university` - Universities
- `library` - Libraries
- `language_school` - Language schools

OSM Mapping:
- `amenity=school` → school
- `amenity=kindergarten` → kindergarten
- `amenity=college` → college
- `amenity=university` → university
- `amenity=library` → library
- `amenity=language_school` → language_school

#### Leisure (`leisure`)
Subcategories:
- `park` - Parks
- `garden` - Gardens
- `playground` - Playgrounds
- `sports_centre` - Sports centres
- `attraction` - Tourist attractions
- `viewpoint` - Viewpoints
- `museum` - Museums
- `monument` - Monuments
- `castle` - Castles

OSM Mapping:
- `leisure=park` → park
- `leisure=garden` → garden
- `leisure=playground` → playground
- `leisure=sports_centre` → sports_centre
- `tourism=attraction` → attraction
- `tourism=viewpoint` → viewpoint
- `tourism=museum` → museum
- `historic=monument` → monument
- `historic=castle` → castle

#### Food & Drink (`food_drink`)
Subcategories:
- `restaurant` - Restaurants
- `cafe` - Cafes
- `bar` - Bars
- `fast_food` - Fast food restaurants

OSM Mapping:
- `amenity=restaurant` → restaurant
- `amenity=cafe` → cafe
- `amenity=bar` → bar
- `amenity=fast_food` → fast_food

## Transport Tile Endpoints

### Get Transport Tile by Types

Retrieve a vector tile containing transport stations filtered by transport types.

**Endpoint:** `GET /api/v1/tiles/transport/{z}/{x}/{y}.pbf`

**Parameters:**
- `z` (path parameter, required): Zoom level (0-18)
- `x` (path parameter, required): Tile X coordinate
- `y` (path parameter, required): Tile Y coordinate
- `types` (query parameter, optional): Comma-separated list of transport types to filter
  - Valid values: `metro`, `bus`, `tram`, `cercania`, `long_distance`

**Response:**
- Content-Type: `application/x-protobuf`
- Content-Encoding: `gzip`
- Cache-Control: `public, max-age=3600`
- Body: MVT tile data in Protocol Buffer format

**Layer Structure:**
The tile contains a single layer named `transport_stations` with the following properties:
- `id` (int64): Station identifier
- `name` (string): Station name
- `type` (string): Transport type
- `line_ids` (int64[]): Array of line IDs serving this station
- `line_count` (int): Number of lines serving this station

**Example Requests:**

```bash
# Get all transport stations
GET /api/v1/tiles/transport/14/8192/6144.pbf

# Get only metro stations
GET /api/v1/tiles/transport/14/8192/6144.pbf?types=metro

# Get multiple transport types
GET /api/v1/tiles/transport/14/8192/6144.pbf?types=metro,bus,tram

# Get only train stations (cercania and long_distance)
GET /api/v1/tiles/transport/14/8192/6144.pbf?types=cercania,long_distance
```

### Transport Types

- `metro` - Metro/Subway stations
  - OSM: `railway=subway`, `station=subway`
- `bus` - Bus stops
  - OSM: `highway=bus_stop`, `route=bus`
- `tram` - Tram stops
  - OSM: `railway=tram_stop`, `route=tram`
- `cercania` - Commuter trains (Cercanías/Rodalies)
  - OSM: `railway=station` with `network` containing "Rodalies" or "Cercanías"
- `long_distance` - Long-distance trains
  - OSM: `railway=station` with `network` containing "Renfe" or "AVE"

### Get Lines by Station ID

Retrieve the list of transport lines serving a specific station. Useful for implementing hover effects that highlight all stations on a line.

**Endpoint:** `GET /api/v1/transport/station/{station_id}/lines`

**Parameters:**
- `station_id` (path parameter, required): Station identifier

**Response:**
- Content-Type: `application/json`
- Body: JSON object with array of lines

**Response Schema:**
```json
{
  "lines": [
    {
      "id": "string",
      "name": "string",
      "ref": "string",
      "type": "string",
      "color": "string (optional)",
      "text_color": "string (optional)",
      "operator": "string (optional)",
      "network": "string (optional)"
    }
  ]
}
```

**Example Request:**

```bash
GET /api/v1/transport/station/12345/lines
```

**Example Response:**

```json
{
  "lines": [
    {
      "id": "101",
      "name": "L1",
      "ref": "L1",
      "type": "metro",
      "color": "#FF0000",
      "text_color": "#FFFFFF",
      "operator": "TMB",
      "network": "Metro de Barcelona"
    },
    {
      "id": "102",
      "name": "L2",
      "ref": "L2",
      "type": "metro",
      "color": "#9B4F96",
      "text_color": "#FFFFFF",
      "operator": "TMB",
      "network": "Metro de Barcelona"
    }
  ]
}
```

## Caching

All tile endpoints implement Redis caching with the following behavior:

- **POI Tiles**: Cached with key `tile:poi:{z}:{x}:{y}:{categories_hash}`
  - Default TTL: 1 hour (configurable via `POI_TILE_CACHE_TTL`)
  
- **Transport Tiles**: Cached with key `tile:transport:{z}:{x}:{y}:{types_hash}`
  - Default TTL: 1 hour (configurable via `TRANSPORT_TILE_CACHE_TTL`)

The categories/types hash ensures that different filter combinations are cached separately.

## Performance Considerations

### Adaptive Filtering by Zoom Level

POI tiles automatically adjust the number of features returned based on zoom level:
- Zoom < 10: Maximum 50 POIs
- Zoom 10-12: Maximum 200 POIs
- Zoom 13-14: Maximum 500 POIs
- Zoom ≥ 15: Maximum 1000 POIs (configurable via `POI_TILE_MAX_FEATURES`)

### Priority Sorting

POIs are sorted by category priority:
1. Healthcare
2. Education
3. Shopping
4. Leisure
5. Food & Drink
6. Others

Within each category, items are sorted alphabetically by name.

## Configuration

### Environment Variables

Add the following to your `.env` file:

```bash
# POI Tile Cache TTL in seconds (default: 3600 = 1 hour)
POI_TILE_CACHE_TTL=3600

# Transport Tile Cache TTL in seconds (default: 3600 = 1 hour)
TRANSPORT_TILE_CACHE_TTL=3600

# Maximum number of POI features per tile (default: 1000)
POI_TILE_MAX_FEATURES=1000
```

## Integration Example

### JavaScript/TypeScript with Mapbox GL JS

```javascript
// Add POI layer
map.addSource('pois', {
  type: 'vector',
  tiles: [
    `${API_BASE_URL}/api/v1/tiles/poi/{z}/{x}/{y}.pbf?categories=healthcare,shopping`
  ],
  minzoom: 12,
  maxzoom: 18
});

map.addLayer({
  id: 'poi-layer',
  type: 'circle',
  source: 'pois',
  'source-layer': 'pois',
  paint: {
    'circle-radius': 6,
    'circle-color': [
      'match',
      ['get', 'category'],
      'healthcare', '#FF0000',
      'shopping', '#00FF00',
      'education', '#0000FF',
      'leisure', '#FFFF00',
      'food_drink', '#FF00FF',
      '#CCCCCC'
    ]
  }
});

// Add transport layer
map.addSource('transport', {
  type: 'vector',
  tiles: [
    `${API_BASE_URL}/api/v1/tiles/transport/{z}/{x}/{y}.pbf?types=metro,bus`
  ],
  minzoom: 10,
  maxzoom: 18
});

map.addLayer({
  id: 'transport-layer',
  type: 'symbol',
  source: 'transport',
  'source-layer': 'transport_stations',
  layout: {
    'icon-image': [
      'match',
      ['get', 'type'],
      'metro', 'metro-icon',
      'bus', 'bus-icon',
      'tram', 'tram-icon',
      'default-icon'
    ]
  }
});

// Handle hover to highlight lines
map.on('mouseover', 'transport-layer', async (e) => {
  const stationId = e.features[0].properties.id;
  const response = await fetch(
    `${API_BASE_URL}/api/v1/transport/station/${stationId}/lines`
  );
  const data = await response.json();
  
  // Highlight all stations on these lines
  // Implementation depends on your requirements
});
```

## Error Handling

The API returns standard HTTP status codes:

- `200 OK`: Successful request
- `400 Bad Request`: Invalid parameters (e.g., invalid zoom level, invalid category)
- `500 Internal Server Error`: Server error

Error responses include a JSON body:
```json
{
  "error": "Error description"
}
```

## Notes

- Tiles are generated on-demand and cached in Redis
- Empty tiles return an empty MVT structure, not a 404
- All coordinate parameters are validated (zoom: 0-18, x/y: valid for zoom level)
- Category and type parameters are case-sensitive
- Multiple categories/types are combined with OR logic (union)
