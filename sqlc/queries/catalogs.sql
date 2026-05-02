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

-- name: ListOrientations :many
SELECT
    orientation_id,
    name
FROM orientations
ORDER BY name ASC;

-- name: ListRentPeriodsByPropertyType :many
SELECT rp.period_id, rp.name
FROM rent_periods rp
INNER JOIN property_type_periods ptp ON ptp.period_id = rp.period_id
WHERE ptp.property_type_id = $1
ORDER BY rp.period_id ASC;
