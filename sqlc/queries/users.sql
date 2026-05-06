
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
WHERE email = $1 LIMIT 1;

-- name: GetUserByUUID :one
SELECT * FROM users 
WHERE user_uuid = $1 LIMIT 1;

-- name: UpdateUserStatus :exec
UPDATE users 
SET status_id = $2, updated_at = now() 
WHERE user_id = $1;