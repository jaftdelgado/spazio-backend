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
