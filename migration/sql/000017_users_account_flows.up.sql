CREATE TABLE IF NOT EXISTS user_verification_challenges (
    challenge_id SERIAL PRIMARY KEY,
    user_id INT REFERENCES users(user_id) ON DELETE CASCADE,
    email VARCHAR(150) NOT NULL,
    purpose VARCHAR(40) NOT NULL,
    code_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    verified_at TIMESTAMPTZ,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_user_verification_challenges_email_purpose
    ON user_verification_challenges(email, purpose, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_user_verification_challenges_user_id_purpose
    ON user_verification_challenges(user_id, purpose, created_at DESC);
