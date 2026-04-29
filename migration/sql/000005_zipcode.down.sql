ALTER TABLE locations
    DROP COLUMN IF EXISTS postal_code_id;

DROP TABLE IF EXISTS postal_code_zones;

DROP INDEX IF EXISTS idx_zones_postal_code_id;
ALTER TABLE zones
    DROP COLUMN IF EXISTS postal_code_id;

DROP INDEX IF EXISTS idx_locations_postal_code_id;
DROP INDEX IF EXISTS idx_postal_codes_city_id;
DROP INDEX IF EXISTS idx_postal_codes_code;
DROP TABLE IF EXISTS postal_codes;

DROP FUNCTION IF EXISTS postal_codes_state_consistency() CASCADE;