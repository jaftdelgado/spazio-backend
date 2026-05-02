-- 000008_property_subtype.up.sql

ALTER TABLE property_types
  ADD COLUMN subtype VARCHAR(20) NOT NULL DEFAULT 'other';

UPDATE property_types SET subtype = 'residential' WHERE property_type_id IN (1, 2);
UPDATE property_types SET subtype = 'commercial'  WHERE property_type_id = 3;

ALTER TABLE properties DROP COLUMN category;