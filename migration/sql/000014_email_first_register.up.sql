CREATE TABLE pending_verifications (
  verification_id  serial PRIMARY KEY,
  email            varchar(150) NOT NULL,
  code_hash        varchar(255) NOT NULL,
  expires_at       timestamptz NOT NULL,
  verified_at      timestamptz,
  created_at       timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_pending_verifications_email
  ON pending_verifications(email);
