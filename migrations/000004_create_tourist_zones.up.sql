CREATE TABLE tourist_zones (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    osm_id              BIGINT NOT NULL UNIQUE,
    type                VARCHAR(50) NOT NULL,
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
    lat                 DOUBLE PRECISION NOT NULL,
    lon                 DOUBLE PRECISION NOT NULL,
    geometry            GEOMETRY(GEOMETRY, 4326) NOT NULL,
    visitors_per_year   INTEGER,
    fee                 BOOLEAN,
    opening_hours       TEXT,
    website             TEXT,
    tags                JSONB,
    search_vector       TSVECTOR,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_tourist_zones_geom ON tourist_zones USING GIST (geometry);
CREATE INDEX idx_tourist_zones_location ON tourist_zones USING SPGIST (ST_MakePoint(lon, lat));
CREATE INDEX idx_tourist_zones_type ON tourist_zones (type);
CREATE INDEX idx_tourist_zones_search ON tourist_zones USING GIN (search_vector);
