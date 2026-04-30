-- name: GetPropertyIDByUUID :one
SELECT property_id
FROM properties
WHERE property_uuid = $1 AND deleted_at IS NULL;

-- name: InsertPropertyPhoto :one
INSERT INTO property_photos (
    property_id,
    storage_key,
    mime_type,
    sort_order,
    is_cover,
    label,
    alt_text
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING photo_id;

-- name: ClearPropertyPhotoCover :exec
UPDATE property_photos
SET is_cover = false
WHERE property_id = $1
    AND is_cover = true;

-- name: UpdatePropertyCoverPhoto :exec
UPDATE properties
SET cover_photo_url = $2,
    updated_at = NOW()
WHERE property_id = $1;
