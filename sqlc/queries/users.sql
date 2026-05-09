
-- name: CreateUser :one
INSERT INTO users (
    user_uuid, 
    role_id, 
    first_name, 
    last_name, 
    email, 
    phone, 
    profile_picture_url, 
    status_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users 
WHERE email = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetUserByUUID :one
SELECT * FROM users 
WHERE user_uuid = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: UpdateUserStatus :exec
UPDATE users 
SET status_id = $2, updated_at = now() 
WHERE user_id = $1;

-- name: UpdateUserByUUID :one
UPDATE users 
SET 
    first_name = $2, 
    last_name = $3, 
    phone = $4, 
    profile_picture_url = $5,
    updated_at = NOW()
WHERE user_uuid = $1
  AND deleted_at IS NULL
RETURNING *;

-- name: DeleteUserByUUID :execrows
UPDATE users
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE user_uuid = $1
  AND deleted_at IS NULL;

-- name: DeleteUserByUUIDOrEmail :execrows
UPDATE users
SET
    deleted_at = NOW(),
    updated_at = NOW()
WHERE deleted_at IS NULL
  AND (user_uuid = $1 OR email = $2);
