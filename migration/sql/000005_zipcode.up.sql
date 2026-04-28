CREATE TABLE IF NOT EXISTS postal_codes (
    postal_code_id  SERIAL          PRIMARY KEY,
    code            VARCHAR(10)     NOT NULL,
    city_id         INT             NOT NULL  REFERENCES cities(city_id),
    state_id        INT             NOT NULL  REFERENCES states(state_id),
    source          VARCHAR(30)     NOT NULL  DEFAULT 'manual'
                                             CHECK (source IN ('seed', 'zippopotam', 'manual')),
    created_at      TIMESTAMPTZ     NOT NULL  DEFAULT NOW(),
    updated_at      TIMESTAMPTZ     NOT NULL  DEFAULT NOW(),

    CONSTRAINT uq_postal_codes_code_city UNIQUE (code, city_id)
);

CREATE INDEX idx_postal_codes_code    ON postal_codes(code);
CREATE INDEX idx_postal_codes_city_id ON postal_codes(city_id);

ALTER TABLE zones
    ADD COLUMN IF NOT EXISTS postal_code_id INT
        REFERENCES postal_codes(postal_code_id)
        ON DELETE SET NULL;

CREATE INDEX idx_zones_postal_code_id ON zones(postal_code_id)
    WHERE postal_code_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS postal_code_zones (
    postal_code_id  INT  NOT NULL  REFERENCES postal_codes(postal_code_id) ON DELETE CASCADE,
    zone_id         INT  NOT NULL  REFERENCES zones(zone_id)               ON DELETE CASCADE,
    PRIMARY KEY (postal_code_id, zone_id)
);

CREATE INDEX idx_pcz_zone_id ON postal_code_zones(zone_id);

ALTER TABLE locations
    ADD COLUMN IF NOT EXISTS postal_code_id INT
        REFERENCES postal_codes(postal_code_id)
        ON DELETE SET NULL;

CREATE INDEX idx_locations_postal_code_id ON locations(postal_code_id)
    WHERE postal_code_id IS NOT NULL;

CREATE OR REPLACE FUNCTION postal_codes_state_consistency()
RETURNS trigger AS $$
BEGIN
    IF (SELECT state_id FROM cities WHERE city_id = NEW.city_id) IS DISTINCT FROM NEW.state_id THEN
        RAISE EXCEPTION 'postal_codes: city (% ) state_id mismatch with provided state_id (% )', NEW.city_id, NEW.state_id;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_postal_codes_state_consistency
BEFORE INSERT OR UPDATE ON postal_codes
FOR EACH ROW
EXECUTE FUNCTION postal_codes_state_consistency();