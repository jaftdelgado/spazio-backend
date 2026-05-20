-- name: GetContractForPaymentWithLock :one
SELECT 
    c.contract_id, 
    t.client_id, 
    c.agreed_amount, 
    c.status_id, 
    c.start_date,
    c.end_date,
    c.currency,
    t.transaction_type,
    c.period_id,
    rp.name as period_name,
    p.status_id as property_status_id
FROM contracts c
JOIN transactions t ON c.transaction_id = t.transaction_id
JOIN properties p ON t.property_id = p.property_id
LEFT JOIN rent_periods rp ON c.period_id = rp.period_id
WHERE c.contract_id = $1
FOR UPDATE OF c;

-- name: UpdateContractStatus :exec
UPDATE contracts
SET status_id = $2
WHERE contract_id = $1;

-- name: GetLastPaidPeriod :one
SELECT billing_period
FROM payments
WHERE contract_id = $1 AND status_id = 2 -- Completed
ORDER BY billing_period DESC
LIMIT 1;

-- name: GetPendingPayments :many
SELECT payment_id, gateway_payment_id
FROM payments
WHERE contract_id = $1 AND status_id = 1; -- Pending

-- name: GetPaymentByContract :many
SELECT * FROM payments 
WHERE contract_id = $1 
AND status_id = $2;

-- name: CreatePayment :one
INSERT INTO payments (
    contract_id,
    client_id,
    billing_period,
    due_date,
    amount,
    payment_method_id,
    gateway_id,
    status_id,
    gateway_payment_id,
    gateway_status,
    payment_date,
    metadata
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
) RETURNING *;

-- name: GetContractForPayment :one
SELECT 
    c.contract_id, 
    t.client_id, 
    c.agreed_amount,
    c.period_id,
    rp.name as period_name,
    t.transaction_type
FROM contracts c
JOIN transactions t ON c.transaction_id = t.transaction_id
LEFT JOIN rent_periods rp ON c.period_id = rp.period_id
WHERE c.contract_id = $1;

-- name: GetPaymentByUUID :one
SELECT p.*, t.client_id
FROM payments p
JOIN contracts c ON p.contract_id = c.contract_id
JOIN transactions t ON c.transaction_id = t.transaction_id
WHERE p.payment_uuid = $1;

-- name: GetPaymentByGatewayID :one
SELECT p.*, t.client_id
FROM payments p
JOIN contracts c ON p.contract_id = c.contract_id
JOIN transactions t ON c.transaction_id = t.transaction_id
WHERE p.gateway_payment_id = $1;

-- name: UpdatePaymentStatus :exec
UPDATE payments 
SET status_id = $2, 
    gateway_payment_id = $3, 
    gateway_status = $4, 
    payment_date = $5 
WHERE payment_id = $1;

-- name: ListPayments :many
SELECT
    p.payment_id,
    p.contract_id,
    t.property_id,
    p.billing_period,
    p.due_date,
    p.amount::text AS amount,
    TRIM(c.currency) AS currency,
    pm.name AS payment_method,
    pg.name AS gateway,
    ps.name AS status,
    p.payment_date,
    COUNT(*) OVER() AS total_count
FROM payments AS p
JOIN contracts AS c ON c.contract_id = p.contract_id AND c.deleted_at IS NULL
JOIN transactions AS t ON t.transaction_id = c.transaction_id
JOIN payment_methods AS pm ON pm.method_id = p.payment_method_id
LEFT JOIN payment_gateways AS pg ON pg.gateway_id = p.gateway_id
JOIN payment_status AS ps ON ps.status_id = p.status_id
WHERE
    (sqlc.narg('property_id')::int IS NULL OR t.property_id = sqlc.narg('property_id')) AND
    (sqlc.narg('status_id')::int IS NULL OR p.status_id = sqlc.narg('status_id')) AND
    (sqlc.narg('date_from')::date IS NULL OR p.due_date >= sqlc.narg('date_from')::date) AND
    (sqlc.narg('date_to')::date IS NULL OR p.due_date <= sqlc.narg('date_to')::date) AND
    (
        sqlc.arg('role_id')::int = 1 OR
        (sqlc.arg('role_id')::int = 2 AND t.agent_id = sqlc.arg('user_id')::int) OR
        (sqlc.arg('role_id')::int = 3 AND t.client_id = sqlc.arg('user_id')::int)
    )
ORDER BY p.due_date DESC, p.payment_id DESC
LIMIT sqlc.arg('page_limit')
OFFSET sqlc.arg('page_offset');

-- name: CountPayments :one
SELECT COUNT(*)
FROM payments AS p
JOIN contracts AS c ON c.contract_id = p.contract_id AND c.deleted_at IS NULL
JOIN transactions AS t ON t.transaction_id = c.transaction_id
WHERE
    (sqlc.narg('property_id')::int IS NULL OR t.property_id = sqlc.narg('property_id')) AND
    (sqlc.narg('status_id')::int IS NULL OR p.status_id = sqlc.narg('status_id')) AND
    (sqlc.narg('date_from')::date IS NULL OR p.due_date >= sqlc.narg('date_from')::date) AND
    (sqlc.narg('date_to')::date IS NULL OR p.due_date <= sqlc.narg('date_to')::date) AND
    (
        sqlc.arg('role_id')::int = 1 OR
        (sqlc.arg('role_id')::int = 2 AND t.agent_id = sqlc.arg('user_id')::int) OR
        (sqlc.arg('role_id')::int = 3 AND t.client_id = sqlc.arg('user_id')::int)
    );

-- name: CountCompletedPaymentsForContract :one
SELECT COUNT(*) FROM payments 
WHERE contract_id = $1 AND status_id = 2;

-- name: UpdateTransactionStatusByContract :exec
UPDATE transactions
SET status_id = $2
WHERE transaction_id = (SELECT transaction_id FROM contracts WHERE contract_id = $1);

-- name: UpdatePropertyStatusByContract :exec
UPDATE properties
SET status_id = $2
WHERE property_id = (
    SELECT t.property_id 
    FROM transactions t
    JOIN contracts c ON t.transaction_id = c.transaction_id
    WHERE c.contract_id = $1
);

-- name: GetPaymentByID :one
SELECT
    p.payment_id,
    p.contract_id,
    t.property_id,
    t.transaction_id,
    t.transaction_type::text AS transaction_type,
    p.billing_period,
    p.due_date,
    c.agreed_amount::text AS agreed_amount,
    p.amount::text AS amount,
    TRIM(c.currency) AS currency,
    pm.name AS payment_method,
    pg.name AS gateway,
    ps.name AS status,
    p.payment_date,
    t.client_id,
    t.agent_id
FROM payments AS p
JOIN contracts AS c ON c.contract_id = p.contract_id AND c.deleted_at IS NULL
JOIN transactions AS t ON t.transaction_id = c.transaction_id
JOIN payment_methods AS pm ON pm.method_id = p.payment_method_id
LEFT JOIN payment_gateways AS pg ON pg.gateway_id = p.gateway_id
JOIN payment_status AS ps ON ps.status_id = p.status_id
WHERE p.payment_id = $1;
