-- Rollback: Convert BIGINT IDs back to UUID

-- Admin Boundaries
ALTER TABLE admin_boundaries DROP CONSTRAINT IF EXISTS admin_boundaries_pkey CASCADE;
ALTER TABLE admin_boundaries DROP COLUMN id;
ALTER TABLE admin_boundaries ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE admin_boundaries DROP COLUMN IF EXISTS parent_id;
ALTER TABLE admin_boundaries ADD COLUMN parent_id UUID REFERENCES admin_boundaries(id);

-- Transport Stations
ALTER TABLE transport_stations DROP CONSTRAINT IF EXISTS transport_stations_pkey CASCADE;
ALTER TABLE transport_stations DROP COLUMN id;
ALTER TABLE transport_stations ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE transport_stations DROP COLUMN IF EXISTS line_ids;
ALTER TABLE transport_stations ADD COLUMN line_ids UUID[] DEFAULT '{}';

-- Transport Lines
ALTER TABLE transport_lines DROP CONSTRAINT IF EXISTS transport_lines_pkey CASCADE;
ALTER TABLE transport_lines DROP COLUMN id;
ALTER TABLE transport_lines ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE transport_lines DROP COLUMN IF EXISTS station_ids;
ALTER TABLE transport_lines ADD COLUMN station_ids UUID[] DEFAULT '{}';

-- POI Categories
ALTER TABLE poi_categories DROP CONSTRAINT IF EXISTS poi_categories_pkey CASCADE;
ALTER TABLE poi_categories DROP COLUMN id;
ALTER TABLE poi_categories ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();

-- POI Subcategories
ALTER TABLE poi_subcategories DROP CONSTRAINT IF EXISTS poi_subcategories_pkey CASCADE;
ALTER TABLE poi_subcategories DROP COLUMN id;
ALTER TABLE poi_subcategories ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();
ALTER TABLE poi_subcategories DROP COLUMN IF EXISTS category_id;
ALTER TABLE poi_subcategories ADD COLUMN category_id UUID NOT NULL REFERENCES poi_categories(id) ON DELETE CASCADE;

-- POIs
ALTER TABLE pois DROP CONSTRAINT IF EXISTS pois_pkey CASCADE;
ALTER TABLE pois DROP COLUMN id;
ALTER TABLE pois ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();

-- Green Spaces
ALTER TABLE green_spaces DROP CONSTRAINT IF EXISTS green_spaces_pkey CASCADE;
ALTER TABLE green_spaces DROP COLUMN id;
ALTER TABLE green_spaces ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();

-- Water Bodies
ALTER TABLE water_bodies DROP CONSTRAINT IF EXISTS water_bodies_pkey CASCADE;
ALTER TABLE water_bodies DROP COLUMN id;
ALTER TABLE water_bodies ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();

-- Beaches
ALTER TABLE beaches DROP CONSTRAINT IF EXISTS beaches_pkey CASCADE;
ALTER TABLE beaches DROP COLUMN id;
ALTER TABLE beaches ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();

-- Noise Sources
ALTER TABLE noise_sources DROP CONSTRAINT IF EXISTS noise_sources_pkey CASCADE;
ALTER TABLE noise_sources DROP COLUMN id;
ALTER TABLE noise_sources ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();

-- Tourist Zones
ALTER TABLE tourist_zones DROP CONSTRAINT IF EXISTS tourist_zones_pkey CASCADE;
ALTER TABLE tourist_zones DROP COLUMN id;
ALTER TABLE tourist_zones ADD COLUMN id UUID PRIMARY KEY DEFAULT gen_random_uuid();

-- Recreate indexes
CREATE INDEX IF NOT EXISTS idx_poi_subcategories_category ON poi_subcategories (category_id);
CREATE INDEX IF NOT EXISTS idx_admin_boundaries_parent ON admin_boundaries (parent_id);
CREATE INDEX IF NOT EXISTS idx_transport_stations_line_ids ON transport_stations USING GIN (line_ids);
CREATE INDEX IF NOT EXISTS idx_transport_lines_station_ids ON transport_lines USING GIN (station_ids);
