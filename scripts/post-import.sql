-- Post-import setup for OSM database
-- Run this AFTER osm2pgsql import to add computed columns and indexes

-- ============================================================
-- 1. pg_trgm extension (for fuzzy search)
-- ============================================================
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- ============================================================
-- 2. Geography columns (way_geog) for fast spatial queries
--    Avoids ST_Transform overhead at query time
-- ============================================================
ALTER TABLE planet_osm_point ADD COLUMN IF NOT EXISTS way_geog geography(Point, 4326);
UPDATE planet_osm_point SET way_geog = ST_Transform(way, 4326)::geography WHERE way IS NOT NULL AND way_geog IS NULL;

ALTER TABLE planet_osm_line ADD COLUMN IF NOT EXISTS way_geog geography(LineString, 4326);
UPDATE planet_osm_line SET way_geog = ST_Transform(way, 4326)::geography WHERE way IS NOT NULL AND way_geog IS NULL;

-- ============================================================
-- 3. Boundary indexes (admin boundaries)
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_admin_boundaries
ON planet_osm_polygon (boundary, admin_level)
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_admin_name_trgm
ON planet_osm_polygon USING GIN (name gin_trgm_ops)
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_admin_name_level
ON planet_osm_polygon (name, admin_level)
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_admin_name_en
ON planet_osm_polygon ((tags->'name:en'))
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_admin_name_es
ON planet_osm_polygon ((tags->'name:es'))
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_admin_boundaries_way
ON planet_osm_polygon USING GIST (way)
WHERE boundary = 'administrative' AND admin_level IS NOT NULL;

-- ============================================================
-- 4. Transport indexes
-- ============================================================

-- GIST index on geography for point spatial queries
CREATE INDEX IF NOT EXISTS idx_planet_osm_point_way_geog
ON planet_osm_point USING GIST (way_geog);

-- Partial index for transport stations only (most efficient)
CREATE INDEX IF NOT EXISTS idx_planet_osm_point_transport_geog
ON planet_osm_point USING GIST (way_geog)
WHERE (
    (railway IN ('station', 'halt', 'tram_stop'))
    OR highway = 'bus_stop'
    OR public_transport IN ('platform', 'stop_position', 'station')
);

-- Full GIST index for lines
CREATE INDEX IF NOT EXISTS idx_planet_osm_line_way_geog
ON planet_osm_line USING GIST (way_geog);

-- B-tree indexes for filtering
CREATE INDEX IF NOT EXISTS idx_planet_osm_point_railway
ON planet_osm_point (railway) WHERE railway IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_planet_osm_point_highway
ON planet_osm_point (highway) WHERE highway IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_planet_osm_point_public_transport
ON planet_osm_point (public_transport) WHERE public_transport IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_planet_osm_point_name
ON planet_osm_point (name) WHERE name IS NOT NULL AND name != '';

CREATE INDEX IF NOT EXISTS idx_planet_osm_line_route
ON planet_osm_line (route) WHERE route IS NOT NULL;

-- osm_id indexes for fast lookups
CREATE INDEX IF NOT EXISTS idx_planet_osm_point_osm_id
ON planet_osm_point (osm_id);

CREATE INDEX IF NOT EXISTS idx_planet_osm_line_osm_id
ON planet_osm_line (osm_id);

-- ============================================================
-- 5. Update statistics
-- ============================================================
ANALYZE planet_osm_point;
ANALYZE planet_osm_line;
ANALYZE planet_osm_polygon;
