-- name: ListClauses :many
SELECT
    c.clause_id,
    c.code,
    cvt.code AS value_type_code,
    c.sort_order,
    COUNT(*) OVER() AS total_count
FROM clauses AS c
JOIN clause_value_types AS cvt ON cvt.value_type_id = c.value_type_id
WHERE c.is_active = true
  AND c.is_deprecated = false
  AND c.clause_id IN (
    SELECT cm.clause_id
    FROM clause_modalities AS cm
    WHERE cm.modality_id = sqlc.arg(modality_id)
  )
ORDER BY c.sort_order ASC
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: SearchClauses :many
SELECT
    c.clause_id,
    c.code,
    cvt.code AS value_type_code,
    c.sort_order,
    COUNT(*) OVER() AS total_count
FROM clauses AS c
JOIN clause_value_types AS cvt ON cvt.value_type_id = c.value_type_id
WHERE c.is_active = true
  AND c.is_deprecated = false
  AND c.clause_id IN (
    SELECT cm.clause_id
    FROM clause_modalities AS cm
    WHERE cm.modality_id = sqlc.arg(modality_id)
  )
  AND c.search_tags IS NOT NULL
  AND jsonb_typeof(c.search_tags) = 'array'
  AND (
    c.search_tags @> jsonb_build_array(sqlc.arg(query)::text)
    OR EXISTS (
      SELECT 1
      FROM jsonb_array_elements_text(c.search_tags) AS tag(value)
      WHERE tag.value ILIKE '%' || sqlc.arg(query) || '%'
    )
  )
ORDER BY c.sort_order ASC
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);
