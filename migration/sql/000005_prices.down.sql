BEGIN;

DROP TABLE IF EXISTS rent_prices;

ALTER TABLE sale_prices
    ADD COLUMN rent_price  DECIMAL(15, 2) NULL,
    ADD COLUMN deposit     DECIMAL(15, 2) NULL,
    ADD COLUMN period_id   INT            NULL
        REFERENCES rent_periods (period_id);

ALTER TABLE sale_prices
    ALTER COLUMN sale_price DROP NOT NULL;

ALTER TABLE sale_prices RENAME TO prices;

ALTER SEQUENCE IF EXISTS sale_prices_price_id_seq
    RENAME TO prices_price_id_seq;

ALTER INDEX IF EXISTS sale_prices_pkey
    RENAME TO prices_pkey;

DROP INDEX IF EXISTS idx_sale_prices_property_id;
DROP INDEX IF EXISTS idx_sale_prices_current;

COMMIT;