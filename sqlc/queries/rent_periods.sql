-- name: ListRentPeriods :many
SELECT
    period_id,
    name
FROM rent_periods
ORDER BY period_id ASC;
