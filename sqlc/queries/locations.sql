-- name: ListCountries :many
SELECT country_id, iso2_code, name
FROM countries
WHERE is_active = true
ORDER BY name ASC;

-- name: ListStates :many
SELECT state_id, iso_code, name
FROM states
WHERE is_active = true
  AND country_id = $1
ORDER BY name ASC;

-- name: ListCities :many
SELECT
    city_id,
    name,
    COUNT(*) OVER() AS total_count
FROM cities
WHERE state_id = $1
ORDER BY name ASC
LIMIT $2 OFFSET $3;
