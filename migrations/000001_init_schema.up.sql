-- Enable PostGIS extensions
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_topology;
CREATE EXTENSION IF NOT EXISTS fuzzystrmatch;
CREATE EXTENSION IF NOT EXISTS pg_trgm;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Admin Boundaries
CREATE TABLE admin_boundaries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    osm_id              BIGINT NOT NULL UNIQUE,
    name                VARCHAR(255) NOT NULL,
    name_en             VARCHAR(255),
    name_es             VARCHAR(255),
    name_ca             VARCHAR(255),
    name_ru             VARCHAR(255),
    name_uk             VARCHAR(255),
    name_fr             VARCHAR(255),
    name_pt             VARCHAR(255),
    name_it             VARCHAR(255),
    name_de             VARCHAR(255),
    type                VARCHAR(50) NOT NULL,
    admin_level         INTEGER NOT NULL CHECK (admin_level IN (2, 4, 6, 8, 9)),
    center_lat          DOUBLE PRECISION NOT NULL,
    center_lon          DOUBLE PRECISION NOT NULL,
    geometry            GEOMETRY(MULTIPOLYGON, 4326) NOT NULL,
    parent_id           UUID REFERENCES admin_boundaries(id),
    population          INTEGER,
    area_sq_km          DOUBLE PRECISION,
    search_vector       TSVECTOR,
    tags                JSONB,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_admin_boundaries_geom ON admin_boundaries USING GIST (geometry);
CREATE INDEX idx_admin_boundaries_center ON admin_boundaries USING GIST (ST_MakePoint(center_lon, center_lat));
CREATE INDEX idx_admin_boundaries_search ON admin_boundaries USING GIN (search_vector);
CREATE INDEX idx_admin_boundaries_admin_level ON admin_boundaries (admin_level);
CREATE INDEX idx_admin_boundaries_parent ON admin_boundaries (parent_id);
CREATE INDEX idx_admin_boundaries_tags ON admin_boundaries USING GIN (tags);

-- Transport Stations
CREATE TABLE transport_stations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    osm_id              BIGINT NOT NULL UNIQUE,
    name                VARCHAR(255) NOT NULL,
    name_en             VARCHAR(255),
    type                VARCHAR(50) NOT NULL CHECK (type IN ('metro', 'train', 'tram', 'bus')),
    lat                 DOUBLE PRECISION NOT NULL,
    lon                 DOUBLE PRECISION NOT NULL,
    geometry            GEOMETRY(POINT, 4326) NOT NULL,
    line_ids            UUID[] DEFAULT '{}',
    operator            VARCHAR(255),
    network             VARCHAR(255),
    wheelchair          BOOLEAN,
    tags                JSONB,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_transport_stations_geom ON transport_stations USING SPGIST (geometry);
CREATE INDEX idx_transport_stations_type ON transport_stations (type);
CREATE INDEX idx_transport_stations_line_ids ON transport_stations USING GIN (line_ids);
CREATE INDEX idx_transport_stations_tags ON transport_stations USING GIN (tags);

-- Transport Lines
CREATE TABLE transport_lines (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    osm_id              BIGINT NOT NULL UNIQUE,
    name                VARCHAR(255) NOT NULL,
    ref                 VARCHAR(50) NOT NULL,
    type                VARCHAR(50) NOT NULL CHECK (type IN ('metro', 'train', 'tram', 'bus')),
    color               VARCHAR(7),
    text_color          VARCHAR(7),
    operator            VARCHAR(255),
    network             VARCHAR(255),
    from_station        VARCHAR(255),
    to_station          VARCHAR(255),
    geometry            GEOMETRY(MULTILINESTRING, 4326) NOT NULL,
    station_ids         UUID[] DEFAULT '{}',
    tags                JSONB,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_transport_lines_geom ON transport_lines USING GIST (geometry);
CREATE INDEX idx_transport_lines_type ON transport_lines (type);
CREATE INDEX idx_transport_lines_ref ON transport_lines (ref);
CREATE INDEX idx_transport_lines_station_ids ON transport_lines USING GIN (station_ids);
