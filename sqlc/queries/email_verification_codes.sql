-- name: CreateVerificationCode :one
INSERT INTO email_verification_codes (user_id, code_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING code_id, created_at;

-- name: GetLatestVerificationCode :one
SELECT code_id, user_id, code_hash, expires_at, used_at
FROM email_verification_codes
WHERE user_id = $1 AND used_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkVerificationCodeUsed :exec
UPDATE email_verification_codes
SET used_at = now()
WHERE code_id = $1;
