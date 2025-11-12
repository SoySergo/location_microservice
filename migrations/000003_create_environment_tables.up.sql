-- Green Spaces
CREATE TABLE green_spaces (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    osm_id              BIGINT NOT NULL UNIQUE,
    type                VARCHAR(50) NOT NULL,
    name                VARCHAR(255),
    name_en             VARCHAR(255),
    area_sq_m           DOUBLE PRECISION NOT NULL,
    geometry            GEOMETRY(MULTIPOLYGON, 4326) NOT NULL,
    center_lat          DOUBLE PRECISION NOT NULL,
    center_lon          DOUBLE PRECISION NOT NULL,
    access              VARCHAR(50),
    tags                JSONB,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_green_spaces_geom ON green_spaces USING GIST (geometry);
CREATE INDEX idx_green_spaces_center ON green_spaces USING GIST (ST_MakePoint(center_lon, center_lat));
CREATE INDEX idx_green_spaces_type ON green_spaces (type);
CREATE INDEX idx_green_spaces_area ON green_spaces (area_sq_m DESC);

-- Water Bodies
CREATE TABLE water_bodies (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    osm_id              BIGINT NOT NULL UNIQUE,
    type                VARCHAR(50) NOT NULL,
    name                VARCHAR(255),
    name_en             VARCHAR(255),
    geometry            GEOMETRY(GEOMETRY, 4326) NOT NULL,
    length              DOUBLE PRECISION,
    area_sq_m           DOUBLE PRECISION,
    tags                JSONB,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_water_bodies_geom ON water_bodies USING GIST (geometry);
CREATE INDEX idx_water_bodies_type ON water_bodies (type);

-- Beaches
CREATE TABLE beaches (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    osm_id              BIGINT NOT NULL UNIQUE,
    name                VARCHAR(255),
    name_en             VARCHAR(255),
    surface             VARCHAR(50) NOT NULL CHECK (surface IN ('sand', 'pebbles', 'unknown')),
    lat                 DOUBLE PRECISION NOT NULL,
    lon                 DOUBLE PRECISION NOT NULL,
    geometry            GEOMETRY(POLYGON, 4326) NOT NULL,
    length              DOUBLE PRECISION,
    blue_flag           BOOLEAN,
    tags                JSONB,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_beaches_geom ON beaches USING GIST (geometry);
CREATE INDEX idx_beaches_location ON beaches USING SPGIST (ST_MakePoint(lon, lat));
CREATE INDEX idx_beaches_surface ON beaches (surface);

-- Noise Sources
CREATE TABLE noise_sources (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    osm_id              BIGINT NOT NULL UNIQUE,
    type                VARCHAR(50) NOT NULL CHECK (type IN ('airport', 'railway', 'industrial', 'nightclub', 'construction')),
    name                VARCHAR(255),
    lat                 DOUBLE PRECISION NOT NULL,
    lon                 DOUBLE PRECISION NOT NULL,
    geometry            GEOMETRY(GEOMETRY, 4326) NOT NULL,
    intensity           VARCHAR(20) CHECK (intensity IN ('high', 'medium', 'low')),
    tags                JSONB,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_noise_sources_geom ON noise_sources USING GIST (geometry);
CREATE INDEX idx_noise_sources_location ON noise_sources USING SPGIST (ST_MakePoint(lon, lat));
CREATE INDEX idx_noise_sources_type ON noise_sources (type);
