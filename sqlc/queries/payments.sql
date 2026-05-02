-- name: GetContractForPaymentWithLock :one
SELECT c.contract_id, t.client_id, c.agreed_amount, c.status_id
FROM contracts c
JOIN transactions t ON c.transaction_id = t.transaction_id
WHERE c.contract_id = $1
FOR UPDATE;

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
SELECT c.contract_id, t.client_id, c.agreed_amount
FROM contracts c
JOIN transactions t ON c.transaction_id = t.transaction_id
WHERE c.contract_id = $1;

-- name: GetPaymentByUUID :one
SELECT * FROM payments WHERE payment_uuid = $1;

-- name: UpdatePaymentStatus :exec
UPDATE payments 
SET status_id = $2, 
    gateway_payment_id = $3, 
    gateway_status = $4, 
    payment_date = $5 
WHERE payment_id = $1;
