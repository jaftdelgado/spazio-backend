-- name: CreateContract :one
INSERT INTO contracts (
    contract_uuid,
    transaction_id,
    currency,
    agreed_amount,
    storage_key,
    start_date,
    end_date,
    status_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetContractDataByTransactionID :one
SELECT 
    t.transaction_id,
    t.transaction_type,
    t.final_amount,
    t.closing_date,
    t.status_id AS transaction_status_id,
    p.property_id,
    p.owner_id,
    p.status_id AS property_status_id,
    t.client_id,
    p.title AS property_title,
    l.street,
    l.exterior_number,
    l.neighborhood,
    ct.name AS city_name,
    st.name AS state_name,
    u_owner.first_name AS owner_first_name,
    u_owner.last_name AS owner_last_name,
    u_owner.email AS owner_email,
    u_client.first_name AS client_first_name,
    u_client.last_name AS client_last_name,
    u_client.email AS client_email
FROM transactions t
JOIN properties p ON t.property_id = p.property_id
JOIN locations l ON p.property_id = l.property_id
JOIN cities ct ON l.city_id = ct.city_id
JOIN states st ON ct.state_id = st.state_id
JOIN users u_owner ON p.owner_id = u_owner.user_id
JOIN users u_client ON t.client_id = u_client.user_id
WHERE t.transaction_id = $1 LIMIT 1;

-- name: CheckContractExistsByTransactionID :one
SELECT EXISTS (
    SELECT 1 FROM contracts WHERE transaction_id = $1 AND deleted_at IS NULL
);

-- name: GetPropertyClausesByTransactionID :many
SELECT 
    c.name AS clause_name,
    c.description AS clause_description,
    pc.boolean_value,
    pc.integer_value,
    pc.min_value,
    pc.max_value,
    cvt.code AS value_type_code
FROM transactions t
JOIN property_clauses pc ON t.property_id = pc.property_id
JOIN clauses c ON pc.clause_id = c.clause_id
JOIN clause_value_types cvt ON c.value_type_id = cvt.value_type_id
WHERE t.transaction_id = $1
ORDER BY c.sort_order ASC;

-- name: ListContracts :many
SELECT 
    c.contract_id,
    c.contract_uuid,
    c.currency,
    c.agreed_amount,
    c.start_date,
    c.end_date,
    c.status_id,
    cs.name AS status_name,
    c.created_at,
    t.transaction_type,
    p.title AS property_title,
    p.owner_id,
    u_client.first_name || ' ' || u_client.last_name as client_name
FROM contracts c
JOIN transactions t ON c.transaction_id = t.transaction_id
JOIN properties p ON t.property_id = p.property_id
JOIN contract_status cs ON c.status_id = cs.status_id
JOIN users u_client ON t.client_id = u_client.user_id
WHERE c.deleted_at IS NULL
    AND (sqlc.narg('owner_id')::int IS NULL OR p.owner_id = sqlc.narg('owner_id'))
    AND (sqlc.narg('transaction_type')::transaction_type IS NULL OR t.transaction_type = sqlc.narg('transaction_type'))
    AND (sqlc.narg('status_id')::int IS NULL OR c.status_id = sqlc.narg('status_id'))
    AND (sqlc.narg('start_date')::timestamptz IS NULL OR c.created_at >= sqlc.narg('start_date'))
    AND (sqlc.narg('end_date')::timestamptz IS NULL OR c.created_at <= sqlc.narg('end_date'))
    AND (sqlc.narg('search')::text IS NULL OR 
         p.title ILIKE '%' || sqlc.narg('search') || '%' OR 
         u_client.first_name ILIKE '%' || sqlc.narg('search') || '%' OR 
         u_client.last_name ILIKE '%' || sqlc.narg('search') || '%')
ORDER BY c.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetContractByUUID :one
SELECT 
    c.contract_id,
    c.contract_uuid,
    c.currency,
    c.agreed_amount,
    c.storage_key,
    c.start_date,
    c.end_date,
    c.status_id,
    cs.name AS status_name,
    c.created_at,
    t.transaction_type,
    p.property_id,
    p.owner_id,
    p.title AS property_title,
    l.street,
    l.exterior_number,
    l.neighborhood,
    ct.name AS city_name,
    st.name AS state_name,
    u_owner.first_name AS owner_first_name,
    u_owner.last_name AS owner_last_name,
    u_owner.email AS owner_email,
    u_client.first_name AS client_first_name,
    u_client.last_name AS client_last_name,
    u_client.email AS client_email
FROM contracts c
JOIN transactions t ON c.transaction_id = t.transaction_id
JOIN properties p ON t.property_id = p.property_id
JOIN contract_status cs ON c.status_id = cs.status_id
JOIN locations l ON p.property_id = l.property_id
JOIN cities ct ON l.city_id = ct.city_id
JOIN states st ON ct.state_id = st.state_id
JOIN users u_owner ON p.owner_id = u_owner.user_id
JOIN users u_client ON t.client_id = u_client.user_id
WHERE c.contract_uuid = $1 AND c.deleted_at IS NULL LIMIT 1;
