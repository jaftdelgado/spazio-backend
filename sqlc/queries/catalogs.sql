-- name: ListModalities :many
SELECT
    modality_id,
    name
FROM modalities
ORDER BY modality_id ASC;

-- name: ListPropertyTypes :many
SELECT
    property_type_id,
    name,
    icon
FROM property_types
WHERE is_deprecated = false
ORDER BY property_type_id ASC;