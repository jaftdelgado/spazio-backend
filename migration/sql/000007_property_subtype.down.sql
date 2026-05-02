-- 000008_property_subtype.down.sql

ALTER TABLE property_types DROP COLUMN subtype;

ALTER TABLE properties
  ADD COLUMN category property_category NOT NULL DEFAULT 'other';