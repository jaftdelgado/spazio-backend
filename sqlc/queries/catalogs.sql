-- name: ListModalities :many
SELECT
    modality_id,
    name
FROM modalities
ORDER BY modality_id ASC;