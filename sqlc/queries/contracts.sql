-- name: UpdateTransactionStatus :exec
UPDATE transactions
SET status_id = $2
WHERE transaction_id = $1;

-- name: UpdatePropertyStatus :exec
UPDATE properties
SET status_id = $2, updated_at = now()
WHERE property_id = $1;

-- name: CreateContract :one
INSERT INTO contracts (
    contract_uuid,
    transaction_id,
    parent_contract_id,
    period_id,
    currency,
    agreed_amount,
    security_deposit,
    storage_key,
    start_date,
    end_date,
    status_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
) RETURNING *;

-- name: FindLatestContractByPropertyAndClient :one
SELECT c.contract_id
FROM contracts c
JOIN transactions t ON c.transaction_id = t.transaction_id
WHERE t.property_id = $1 
  AND t.client_id = $2
  AND c.deleted_at IS NULL
ORDER BY c.created_at DESC
LIMIT 1;

-- name: GetContractDataByTransactionID :one
SELECT 
    t.transaction_id,
    t.transaction_type,
    t.final_amount,
    t.closing_date,
    t.status_id AS transaction_status_id,
    t.agent_id,
    p.property_id,
    p.owner_id,
    p.status_id AS property_status_id,
    COALESCE(t.client_id, 0) AS client_id,
    p.title AS property_title,
    p.description AS property_description,
    pt.name AS property_type_name,
    p.lot_area,
    rp_res.bedrooms,
    rp_res.bathrooms,
    rp_res.floors,
    rp_res.built_area,
    l.street,
    l.exterior_number,
    l.neighborhood,
    ct.name AS city_name,
    st.name AS state_name,
    u_owner.first_name AS owner_first_name,
    u_owner.last_name AS owner_last_name,
    u_owner.email AS owner_email,
    COALESCE(u_client.first_name, '') AS client_first_name,
    COALESCE(u_client.last_name, '') AS client_last_name,
    COALESCE(u_client.email, '') AS client_email,
    rper.name AS period_name
FROM transactions t
JOIN properties p ON t.property_id = p.property_id
JOIN property_types pt ON p.property_type_id = pt.property_type_id
LEFT JOIN residential_properties rp_res ON p.property_id = rp_res.property_id
JOIN locations l ON p.property_id = l.property_id
JOIN cities ct ON l.city_id = ct.city_id
JOIN states st ON ct.state_id = st.state_id
JOIN users u_owner ON p.owner_id = u_owner.user_id
LEFT JOIN users u_client ON t.client_id = u_client.user_id
LEFT JOIN rent_prices rpr ON p.property_id = rpr.property_id AND rpr.is_current = true
LEFT JOIN rent_periods rper ON rpr.period_id = rper.period_id
WHERE t.transaction_id = $1 LIMIT 1;

-- name: GetPropertyServicesByTransactionID :many
SELECT s.code
FROM transactions t
JOIN property_services ps ON t.property_id = ps.property_id
JOIN services s ON ps.service_id = s.service_id
WHERE t.transaction_id = $1;

-- name: CheckContractExistsByTransactionID :one
SELECT EXISTS (
    SELECT 1 FROM contracts WHERE transaction_id = $1 AND deleted_at IS NULL
);

-- name: GetPropertyClausesByTransactionID :many
SELECT 
    c.name AS clause_name,
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
    c.security_deposit,
    c.start_date,
    c.end_date,
    c.status_id,
    cs.name AS status_name,
    c.created_at,
    t.transaction_type,
    p.title AS property_title,
    p.owner_id,
    COALESCE(t.client_id, 0) AS client_id,
    TRIM(COALESCE(u_client.first_name, '') || ' ' || COALESCE(u_client.last_name, '')) AS client_name
FROM contracts c
JOIN transactions t ON c.transaction_id = t.transaction_id
JOIN properties p ON t.property_id = p.property_id
JOIN contract_status cs ON c.status_id = cs.status_id
LEFT JOIN users u_client ON t.client_id = u_client.user_id
WHERE c.deleted_at IS NULL
    AND (
        sqlc.narg('filter_user_id')::int IS NULL OR 
        p.owner_id = sqlc.narg('filter_user_id') OR 
        t.client_id = sqlc.narg('filter_user_id')
    )
    AND (sqlc.narg('transaction_type')::transaction_type IS NULL OR t.transaction_type = sqlc.narg('transaction_type'))
    AND (sqlc.narg('status_id')::int IS NULL OR c.status_id = sqlc.narg('status_id'))
    AND (sqlc.narg('start_date')::timestamptz IS NULL OR c.created_at >= sqlc.narg('start_date'))
    AND (sqlc.narg('end_date')::timestamptz IS NULL OR c.created_at <= sqlc.narg('end_date'))
    AND (
        sqlc.narg('search')::text IS NULL OR 
        p.title ILIKE '%' || sqlc.narg('search') || '%' OR 
        u_client.first_name ILIKE '%' || sqlc.narg('search') || '%' OR 
        u_client.last_name ILIKE '%' || sqlc.narg('search') || '%'
    )
ORDER BY c.created_at DESC
LIMIT $1 OFFSET $2;

-- name: GetContractByUUID :one
SELECT 
    c.contract_id,
    c.contract_uuid,
    c.currency,
    c.agreed_amount,
    c.security_deposit,
    c.storage_key,
    c.start_date,
    c.end_date,
    c.status_id,
    cs.name AS status_name,
    c.created_at,
    t.transaction_type,
    p.property_id,
    p.owner_id,
    COALESCE(t.client_id, 0) AS client_id,
    p.title AS property_title,
    l.street,
    l.exterior_number,
    l.neighborhood,
    ct.name AS city_name,
    st.name AS state_name,
    u_owner.first_name AS owner_first_name,
    u_owner.last_name AS owner_last_name,
    u_owner.email AS owner_email,
    COALESCE(u_client.first_name, '') AS client_first_name,
    COALESCE(u_client.last_name, '') AS client_last_name,
    COALESCE(u_client.email, '') AS client_email,
    rper.name AS period_name
FROM contracts c
JOIN transactions t ON c.transaction_id = t.transaction_id
JOIN properties p ON t.property_id = p.property_id
JOIN contract_status cs ON c.status_id = cs.status_id
JOIN locations l ON p.property_id = l.property_id
JOIN cities ct ON l.city_id = ct.city_id
JOIN states st ON ct.state_id = st.state_id
JOIN users u_owner ON p.owner_id = u_owner.user_id
LEFT JOIN users u_client ON t.client_id = u_client.user_id
LEFT JOIN rent_periods rper ON c.period_id = rper.period_id
WHERE c.contract_uuid = $1 AND c.deleted_at IS NULL LIMIT 1;