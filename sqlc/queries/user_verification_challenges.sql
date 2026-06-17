-- name: CreateUserVerificationChallenge :one
INSERT INTO user_verification_challenges (
    user_id,
    email,
    purpose,
    code_hash,
    expires_at
) VALUES ($1, $2, $3, $4, $5)
RETURNING challenge_id, created_at;

-- name: GetLatestUserVerificationChallenge :one
SELECT challenge_id, user_id, email, purpose, code_hash, expires_at, verified_at, consumed_at, created_at
FROM user_verification_challenges
WHERE email = $1
  AND purpose = $2
  AND consumed_at IS NULL
ORDER BY created_at DESC
LIMIT 1;

-- name: GetUserVerificationChallengeByID :one
SELECT challenge_id, user_id, email, purpose, code_hash, expires_at, verified_at, consumed_at, created_at
FROM user_verification_challenges
WHERE challenge_id = $1;

-- name: MarkUserVerificationChallengeVerified :exec
UPDATE user_verification_challenges
SET verified_at = now()
WHERE challenge_id = $1;

-- name: ConsumeUserVerificationChallenge :exec
UPDATE user_verification_challenges
SET consumed_at = now()
WHERE challenge_id = $1;
