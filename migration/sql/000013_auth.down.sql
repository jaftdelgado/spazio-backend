DROP INDEX IF EXISTS idx_email_verification_codes_user_id;
DROP TABLE IF EXISTS email_verification_codes;

DROP INDEX IF EXISTS idx_refresh_tokens_user_id;
DROP TABLE IF EXISTS refresh_tokens;

DELETE FROM user_status WHERE status_id = 3 AND name = 'Pending';

ALTER TABLE users DROP COLUMN IF EXISTS password;
