-- 000012_flexible_rent_periods.down.sql

ALTER TABLE contracts DROP COLUMN IF EXISTS period_id;

-- Restoration of the constraint might fail if non-monthly data exists
-- so we leave it commented or just skip it.
-- ALTER TABLE payments ADD CONSTRAINT payments_billing_period_check CHECK (EXTRACT(DAY FROM billing_period) = 1);
