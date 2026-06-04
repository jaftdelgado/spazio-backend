ALTER TABLE users ADD COLUMN password varchar(255) NOT NULL DEFAULT '';

INSERT INTO user_status (status_id, name) VALUES (3, 'Pending')
ON CONFLICT (status_id) DO NOTHING;

CREATE TABLE refresh_tokens (
  token_id    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     integer NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  token_hash  varchar(255) NOT NULL UNIQUE,
  expires_at  timestamptz NOT NULL,
  revoked_at  timestamptz,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);

CREATE TABLE email_verification_codes (
  code_id    serial PRIMARY KEY,
  user_id    integer NOT NULL REFERENCES users(user_id) ON DELETE CASCADE,
  code_hash  varchar(255) NOT NULL,
  expires_at timestamptz NOT NULL,
  used_at    timestamptz,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_email_verification_codes_user_id ON email_verification_codes(user_id);
