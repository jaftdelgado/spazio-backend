CREATE TABLE IF NOT EXISTS health_check (
    id         SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW()
);