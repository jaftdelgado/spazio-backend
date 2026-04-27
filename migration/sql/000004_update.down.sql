ALTER TABLE properties DROP COLUMN IF EXISTS category;
DROP TYPE IF EXISTS property_category;

ALTER TABLE contracts DROP COLUMN IF EXISTS parent_contract_id;

DROP INDEX IF EXISTS idx_single_property_cover;

ALTER TABLE properties ALTER COLUMN cover_photo_url SET NOT NULL;
ALTER TABLE users ALTER COLUMN profile_picture_url SET NOT NULL;

ALTER TABLE agent_schedules DROP CONSTRAINT IF EXISTS exclude_overlap;
DROP EXTENSION IF EXISTS btree_gist;

ALTER TABLE property_exceptions DROP COLUMN IF EXISTS start_time;
ALTER TABLE property_exceptions DROP COLUMN IF EXISTS end_time;

ALTER TABLE locations DROP COLUMN IF EXISTS city_id;
ALTER TABLE locations ADD COLUMN IF NOT EXISTS country_id int REFERENCES countries(country_id);
ALTER TABLE locations ADD COLUMN IF NOT EXISTS state_id int REFERENCES states(state_id);
ALTER TABLE locations ADD COLUMN IF NOT EXISTS city varchar(60);

DROP TABLE IF EXISTS cities;

ALTER TABLE properties ADD COLUMN IF NOT EXISTS current_resident_id int REFERENCES users(user_id);
ALTER TABLE commercial_properties ADD COLUMN IF NOT EXISTS lot_area decimal(12,2);
ALTER TABLE residential_properties ADD COLUMN IF NOT EXISTS lot_area decimal(12,2);
ALTER TABLE properties DROP COLUMN IF EXISTS lot_area;

ALTER TABLE payments ADD COLUMN IF NOT EXISTS client_id int REFERENCES users(user_id);
ALTER TABLE payments ALTER COLUMN gateway_id SET NOT NULL;

ALTER TABLE prices DROP CONSTRAINT IF EXISTS chk_sale_period;
ALTER TABLE prices DROP CONSTRAINT IF EXISTS chk_price_positive;
ALTER TABLE prices DROP COLUMN IF EXISTS is_current;
ALTER TABLE prices ALTER COLUMN period_id SET NOT NULL;

DROP TABLE IF EXISTS clause_modalities;

ALTER TABLE clauses ADD COLUMN IF NOT EXISTS description text;
ALTER TABLE clauses ADD COLUMN IF NOT EXISTS icon varchar(80);

ALTER TABLE users ADD COLUMN IF NOT EXISTS password_hash varchar(255) NOT NULL DEFAULT '';
ALTER TABLE users ALTER COLUMN user_uuid SET DEFAULT gen_random_uuid();