-- name: CreateUser :one
INSERT INTO users (
    user_uuid,
    role_id,
    first_name,
    last_name,
    email,
    password,
    phone,
    profile_picture_url,
    status_id
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING user_id, user_uuid, email, created_at;

-- name: GetUserByEmail :one
SELECT u.user_id, u.user_uuid, u.email, u.password, u.role_id, r.name AS role_name, u.status_id, u.created_at
FROM users u
JOIN roles r ON r.role_id = u.role_id
WHERE u.email = $1 AND u.deleted_at IS NULL;

-- name: GetUserByUUID :one
SELECT u.user_id, u.user_uuid, u.email, u.role_id, r.name AS role_name, u.status_id, u.created_at
FROM users u
JOIN roles r ON r.role_id = u.role_id
WHERE u.user_uuid = $1 AND u.deleted_at IS NULL;

-- name: GetUserByID :one
SELECT user_id, user_uuid, email, role_id, status_id, created_at
FROM users
WHERE user_id = $1 AND deleted_at IS NULL;

-- name: GetAuthenticatedUserByUUID :one
SELECT u.user_id, u.user_uuid, u.email, u.role_id, r.name AS role_name
FROM users u
JOIN roles r ON r.role_id = u.role_id
WHERE u.user_uuid = $1 AND u.deleted_at IS NULL
  AND u.status_id = 1;

-- name: GetAuthenticatedUserByID :one
SELECT u.user_id, u.user_uuid, u.email, u.role_id, r.name AS role_name
FROM users u
JOIN roles r ON r.role_id = u.role_id
WHERE u.user_id = $1 AND u.deleted_at IS NULL
  AND u.status_id = 1;

-- name: UpdateUserStatus :exec
UPDATE users
SET status_id = $1, updated_at = now()
WHERE user_id = $2 AND deleted_at IS NULL;

-- name: UpdateUserProfile :one
UPDATE users
SET first_name = $1,
    last_name = $2,
    phone = $3,
    profile_picture_url = $4,
    updated_at = now()
WHERE user_uuid = $5 AND deleted_at IS NULL
RETURNING user_id, user_uuid, email, created_at;

-- name: SoftDeleteUser :execrows
UPDATE users
SET deleted_at = now(), updated_at = now()
WHERE user_uuid = $1 AND deleted_at IS NULL;
