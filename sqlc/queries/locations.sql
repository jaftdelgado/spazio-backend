-- name: ListCountries :many
SELECT country_id, iso2_code, name
FROM countries
WHERE is_active = true
ORDER BY name ASC;

-- name: ListStates :many
SELECT state_id, iso_code, name
FROM states
WHERE is_active = true
  AND country_id = sqlc.arg(country_id)
  AND (sqlc.arg(search) = '' OR name ILIKE '%' || sqlc.arg(search) || '%')
ORDER BY name ASC;

-- name: ListCities :many
SELECT
    city_id,
    name,
    COUNT(*) OVER() AS total_count
FROM cities
WHERE state_id = sqlc.arg(state_id)
  AND (sqlc.arg(search) = '' OR name ILIKE '%' || sqlc.arg(search) || '%')
ORDER BY name ASC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(row_offset);
