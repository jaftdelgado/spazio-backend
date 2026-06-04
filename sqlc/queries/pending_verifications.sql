-- name: CreatePendingVerification :one
INSERT INTO pending_verifications (email, code_hash, expires_at)
VALUES ($1, $2, $3)
RETURNING verification_id, created_at;

-- name: GetLatestPendingVerification :one
SELECT verification_id, email, code_hash, expires_at, verified_at
FROM pending_verifications
WHERE email = $1
  AND verified_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: MarkPendingVerificationVerified :exec
UPDATE pending_verifications
SET verified_at = now()
WHERE verification_id = $1;

-- name: DeleteExpiredPendingVerifications :exec
DELETE FROM pending_verifications
WHERE expires_at < now();
