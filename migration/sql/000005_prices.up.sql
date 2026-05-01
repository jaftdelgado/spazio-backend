BEGIN;

ALTER TABLE prices RENAME TO sale_prices;

ALTER TABLE sale_prices
    DROP COLUMN IF EXISTS rent_price,
    DROP COLUMN IF EXISTS deposit,
    DROP COLUMN IF EXISTS period_id;

ALTER TABLE sale_prices
    ALTER COLUMN sale_price SET NOT NULL;

ALTER SEQUENCE IF EXISTS prices_price_id_seq
    RENAME TO sale_prices_price_id_seq;

ALTER INDEX IF EXISTS prices_pkey
    RENAME TO sale_prices_pkey;

CREATE TABLE rent_prices (
    rent_price_id       SERIAL          PRIMARY KEY,
    property_id         INT             NOT NULL
                                              REFERENCES properties (property_id)
                                              ON DELETE CASCADE,
    period_id           INT             NOT NULL
                                              REFERENCES rent_periods (period_id),
    rent_price          DECIMAL(15, 2)  NOT NULL,
    deposit             DECIMAL(15, 2)  NULL,
    currency            CHAR(3)         NOT NULL DEFAULT 'MXN',
    is_negotiable       BOOLEAN         NOT NULL DEFAULT false,
    is_current          BOOLEAN         NOT NULL DEFAULT true,
    valid_from          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    valid_until         TIMESTAMPTZ     NULL,
    change_reason       VARCHAR(100)    NULL,
    changed_by_user_id  INT             NOT NULL
                                              REFERENCES users (user_id)
);

CREATE UNIQUE INDEX uq_rent_price_current
    ON rent_prices (property_id, period_id)
    WHERE is_current = true;

CREATE INDEX idx_rent_prices_property_id
    ON rent_prices (property_id);

CREATE INDEX idx_rent_prices_current
    ON rent_prices (property_id, period_id)
    WHERE is_current = true;

CREATE INDEX idx_rent_prices_changed_by
    ON rent_prices (changed_by_user_id);

CREATE INDEX IF NOT EXISTS idx_sale_prices_property_id
    ON sale_prices (property_id);

CREATE INDEX IF NOT EXISTS idx_sale_prices_current
    ON sale_prices (property_id)
    WHERE is_current = true;

COMMIT;