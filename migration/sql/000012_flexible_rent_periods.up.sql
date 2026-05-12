-- 000012_flexible_rent_periods.up.sql

-- Add period_id to contracts to know the frequency of rent payments
ALTER TABLE contracts ADD COLUMN period_id int REFERENCES rent_periods(period_id);

-- Populate rent_periods if they are empty
INSERT INTO rent_periods (period_id, name) VALUES
(1, 'Daily'),
(2, 'Weekly'),
(3, 'Monthly'),
(4, 'Yearly')
ON CONFLICT (period_id) DO NOTHING;

-- Default existing rent contracts to Monthly (3)
UPDATE contracts c
SET period_id = 3
FROM transactions t
WHERE c.transaction_id = t.transaction_id
AND t.transaction_type = 'rent'
AND c.period_id IS NULL;

-- Remove the monthly-only constraint from payments
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'payments_billing_period_check') THEN
        ALTER TABLE payments DROP CONSTRAINT payments_billing_period_check;
    END IF;
END $$;
