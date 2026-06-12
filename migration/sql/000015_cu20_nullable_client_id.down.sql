UPDATE transactions SET client_id = 0 WHERE client_id IS NULL;

ALTER TABLE transactions
    ALTER COLUMN client_id SET NOT NULL;