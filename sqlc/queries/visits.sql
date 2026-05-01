-- name: GetPrimaryAgentForProperty :one
SELECT agent_id 
FROM property_agents 
WHERE property_id = $1 AND is_primary = true
LIMIT 1;

-- name: GetAgentSchedule :many
SELECT day_of_week, start_time, end_time 
FROM agent_schedules 
WHERE agent_id = $1 AND is_active = true;

-- name: GetPropertyExceptions :many
SELECT exception_date, start_time, end_time, reason 
FROM property_exceptions 
WHERE property_id = $1 AND exception_date >= $2 AND exception_date <= $3;

-- name: GetOccupiedVisits :many
SELECT visit_date 
FROM visits 
WHERE agent_id = $1 
AND visit_date >= $2 
AND visit_date <= $3
AND status_id != 5; -- Excluir canceladas (StatusCancelled = 5)

-- name: CreateVisit :one
INSERT INTO visits (
    property_id,
    client_id,
    agent_id, 
    visit_date,
    status_id
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetVisitByUUID :one
SELECT * FROM visits WHERE visit_uuid = $1;

-- name: GetUserRole :one
SELECT role_id FROM users WHERE user_id = $1;

-- name: GetPropertyStatusAndCheckDeleted :one
SELECT status_id, deleted_at FROM properties WHERE property_id = $1;

-- name: CheckUserActive :one
SELECT user_id FROM users WHERE user_id = $1 AND deleted_at IS NULL;

-- name: ListVisits :many
SELECT 
    v.*, 
    p.title as property_title,
    s.name as status_name,
    u_client.first_name || ' ' || u_client.last_name as client_name,
    u_client.phone as client_phone,
    u_agent.first_name || ' ' || u_agent.last_name as agent_name,
    u_agent.phone as agent_phone,
    c.name as city_name,
    l.street || ' ' || l.exterior_number || ', ' || l.neighborhood as address
FROM visits v
JOIN properties p ON v.property_id = p.property_id AND p.deleted_at IS NULL
JOIN visit_status s ON v.status_id = s.status_id
JOIN users u_client ON v.client_id = u_client.user_id AND u_client.deleted_at IS NULL
LEFT JOIN users u_agent ON v.agent_id = u_agent.user_id AND (u_agent.deleted_at IS NULL OR v.agent_id IS NULL)
LEFT JOIN locations l ON p.property_id = l.property_id
LEFT JOIN cities c ON l.city_id = c.city_id
WHERE 
    (sqlc.narg('client_id')::int IS NULL OR v.client_id = sqlc.narg('client_id')) AND
    (sqlc.narg('agent_id')::int IS NULL OR v.agent_id = sqlc.narg('agent_id')) AND
    (sqlc.narg('status_id')::int IS NULL OR v.status_id = sqlc.narg('status_id')) AND
    (sqlc.narg('property_id')::int IS NULL OR v.property_id = sqlc.narg('property_id')) AND
    (sqlc.narg('visit_date')::date IS NULL OR v.visit_date::date = sqlc.narg('visit_date')::date) AND
    (v.deleted_at IS NULL)
ORDER BY v.visit_date DESC;

-- name: UpdateVisitStatus :exec
UPDATE visits SET status_id = $2 WHERE visit_id = $1;

-- name: CreateVisitStatusHistory :exec
INSERT INTO visit_status_history (
    visit_id, 
    previous_status_id, 
    new_status_id, 
    changed_by_user_id
) VALUES (
    $1, $2, $3, $4
);
