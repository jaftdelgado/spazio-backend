-- name: GetSalePropertyByUUID :one
SELECT
  property_id,
  property_uuid,
  modality_id,
  status_id
FROM properties
WHERE property_uuid = $1
  AND deleted_at IS NULL
LIMIT 1;

-- name: GetCurrentSalePriceByPropertyID :one
SELECT
  sale_price,
  currency
FROM sale_prices
WHERE property_id = $1
  AND is_current = true
LIMIT 1;

-- name: CreateSaleTransaction :one
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
  NULL,
  $2,
  'sale',
  $3,
  $4,
  CURRENT_DATE
) RETURNING
  transaction_id,
  transaction_uuid,
  property_id,
  agent_id,
  transaction_type,
  status_id,
  final_amount,
  closing_date;

-- name: CreateSalePropertyStatusHistory :exec
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
