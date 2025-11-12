-- POI Categories
CREATE TABLE poi_categories (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                VARCHAR(50) NOT NULL UNIQUE,
    name_en             VARCHAR(100) NOT NULL,
    name_es             VARCHAR(100) NOT NULL,
    name_ca             VARCHAR(100) NOT NULL,
    name_ru             VARCHAR(100) NOT NULL,
    name_uk             VARCHAR(100) NOT NULL,
    name_fr             VARCHAR(100) NOT NULL,
    name_pt             VARCHAR(100) NOT NULL,
    name_it             VARCHAR(100) NOT NULL,
    name_de             VARCHAR(100) NOT NULL,
    icon                VARCHAR(100),
    color               VARCHAR(7),
    sort_order          INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_poi_categories_code ON poi_categories (code);

-- POI Subcategories
CREATE TABLE poi_subcategories (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category_id         UUID NOT NULL REFERENCES poi_categories(id) ON DELETE CASCADE,
    code                VARCHAR(50) NOT NULL,
    name_en             VARCHAR(100) NOT NULL,
    name_es             VARCHAR(100) NOT NULL,
    name_ca             VARCHAR(100) NOT NULL,
    name_ru             VARCHAR(100) NOT NULL,
    name_uk             VARCHAR(100) NOT NULL,
    name_fr             VARCHAR(100) NOT NULL,
    name_pt             VARCHAR(100) NOT NULL,
    name_it             VARCHAR(100) NOT NULL,
    name_de             VARCHAR(100) NOT NULL,
    icon                VARCHAR(100),
    sort_order          INTEGER NOT NULL DEFAULT 0,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(category_id, code)
);

CREATE INDEX idx_poi_subcategories_category ON poi_subcategories (category_id);

-- POIs
CREATE TABLE pois (
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
    category            VARCHAR(50) NOT NULL,
    subcategory         VARCHAR(50) NOT NULL,
    lat                 DOUBLE PRECISION NOT NULL,
    lon                 DOUBLE PRECISION NOT NULL,
    geometry            GEOMETRY(POINT, 4326) NOT NULL,
    address             TEXT,
    phone               VARCHAR(50),
    website             TEXT,
    opening_hours       TEXT,
    wheelchair          BOOLEAN,
    tags                JSONB,
    search_vector       TSVECTOR,
    created_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at          TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_pois_geom ON pois USING SPGIST (geometry);
CREATE INDEX idx_pois_category ON pois (category, subcategory);
CREATE INDEX idx_pois_search ON pois USING GIN (search_vector);
CREATE INDEX idx_pois_tags ON pois USING GIN (tags);
