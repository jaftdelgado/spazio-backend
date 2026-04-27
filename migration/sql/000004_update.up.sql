ALTER TABLE prices ALTER COLUMN period_id DROP NOT NULL;
ALTER TABLE prices ADD COLUMN is_current BOOLEAN DEFAULT true;
ALTER TABLE prices ADD CONSTRAINT chk_price_positive CHECK (sale_price > 0 OR rent_price > 0);
ALTER TABLE prices ADD CONSTRAINT chk_sale_period CHECK (
  (sale_price IS NOT NULL AND period_id IS NULL) OR
  (rent_price IS NOT NULL AND period_id IS NOT NULL)
);

ALTER TABLE payments ALTER COLUMN gateway_id DROP NOT NULL;
ALTER TABLE payments DROP COLUMN client_id;

ALTER TABLE properties ADD COLUMN lot_area DECIMAL(12,2);
ALTER TABLE residential_properties DROP COLUMN lot_area;
ALTER TABLE commercial_properties DROP COLUMN lot_area;
ALTER TABLE properties DROP COLUMN current_resident_id;

CREATE TABLE cities (
  city_id SERIAL PRIMARY KEY,
  state_id INT NOT NULL REFERENCES states(state_id),
  name VARCHAR(80) NOT NULL
);

ALTER TABLE locations DROP COLUMN country_id;
ALTER TABLE locations DROP COLUMN state_id;
ALTER TABLE locations DROP COLUMN city;
ALTER TABLE locations ADD COLUMN city_id INT NOT NULL REFERENCES cities(city_id);

ALTER TABLE property_exceptions ADD COLUMN start_time TIME;
ALTER TABLE property_exceptions ADD COLUMN end_time TIME;

CREATE EXTENSION IF NOT EXISTS btree_gist;

ALTER TABLE agent_schedules ADD CONSTRAINT exclude_overlap
EXCLUDE USING gist (
  agent_id WITH =,
  day_of_week WITH =,
  tsrange(
    ('2000-01-01'::date + start_time)::timestamp,
    ('2000-01-01'::date + end_time)::timestamp
  ) WITH &&
);

ALTER TABLE users ALTER COLUMN profile_picture_url DROP NOT NULL;
ALTER TABLE properties ALTER COLUMN cover_photo_url DROP NOT NULL;

CREATE UNIQUE INDEX idx_single_property_cover ON property_photos(property_id) WHERE (is_cover = true);

ALTER TABLE contracts ADD COLUMN parent_contract_id INT REFERENCES contracts(contract_id);

CREATE TYPE property_category AS ENUM ('residential', 'commercial', 'land', 'other');
ALTER TABLE properties ADD COLUMN category property_category NOT NULL DEFAULT 'residential';
ALTER TABLE properties ALTER COLUMN category DROP DEFAULT;

ALTER TABLE users DROP COLUMN password_hash;
ALTER TABLE users ALTER COLUMN user_uuid DROP DEFAULT;

-- Clauses: search_tags + clause_modalities
ALTER TABLE clauses DROP COLUMN IF EXISTS description;
ALTER TABLE clauses DROP COLUMN IF EXISTS icon;
ALTER TABLE clauses ADD COLUMN search_tags JSONB NULL;

CREATE INDEX IF NOT EXISTS idx_clauses_search_tags ON clauses USING GIN (search_tags);

CREATE TABLE IF NOT EXISTS clause_modalities (
  clause_id   INT NOT NULL REFERENCES clauses(clause_id),
  modality_id INT NOT NULL REFERENCES modalities(modality_id),
  PRIMARY KEY (clause_id, modality_id)
);

-- Services: search_tags
ALTER TABLE services ADD COLUMN search_tags JSONB NULL;

CREATE INDEX IF NOT EXISTS idx_services_search_tags ON services USING GIN (search_tags);