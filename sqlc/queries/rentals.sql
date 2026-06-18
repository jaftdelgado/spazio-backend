-- name: GetRentalPropertyByUUID :one
SELECT
  property_id,
  property_uuid,
  property_type_id,
  modality_id,
  status_id
FROM properties
WHERE property_uuid = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetAllowedRentalPeriods :many
SELECT period_id
FROM property_type_periods
WHERE property_type_id = $1
ORDER BY period_id ASC;

-- name: ListRentalActivePrices :many
SELECT
  rp.period_id,
  rper.name AS period_name,
  rp.rent_price,
  rp.deposit,
  rp.currency,
  rp.is_negotiable
FROM rent_prices rp
JOIN rent_periods rper ON rper.period_id = rp.period_id
WHERE rp.property_id = $1
  AND rp.is_current = true
ORDER BY rp.period_id ASC;

-- name: ListRentalBlockedDates :many
SELECT
  exception_date,
  reason
FROM property_exceptions
WHERE property_id = $1
  AND exception_date >= $2
  AND exception_date <= $3
ORDER BY exception_date ASC;

-- name: GetPrimaryRentalAgentForProperty :one
SELECT agent_id
FROM properties
WHERE property_id = $1
  AND deleted_at IS NULL
  AND agent_id IS NOT NULL
LIMIT 1;

-- name: CreateRentalTransaction :one
INSERT INTO transactions (
  property_id,
  client_id,
  agent_id,
  transaction_type,
  status_id,
  final_amount,
  closing_date
) VALUES (
  $1,
  $2,
  $3,
  'rent',
  $4,
  $5,
  $6
) RETURNING *;

-- name: UpdateRentalPropertyStatus :exec
UPDATE properties
SET status_id = $2,
    updated_at = NOW()
WHERE property_id = $1;

-- name: CreateRentalPropertyStatusHistory :exec
INSERT INTO property_status_history (
  property_id,
  previous_status_id,
  new_status_id,
  changed_by_user_id
) VALUES (
  $1,
  $2,
  $3,
  $4
);

-- name: UpdateRentalTransactionStatus :exec
UPDATE transactions
SET status_id = $2
WHERE transaction_id = $1;
