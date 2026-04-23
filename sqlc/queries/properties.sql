-- name: CreateProperty :one
INSERT INTO properties (
    owner_id,
    title,
    description,
    property_type_id,
    modality_id,
    status_id,
    cover_photo_url
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING property_id, title, created_at;