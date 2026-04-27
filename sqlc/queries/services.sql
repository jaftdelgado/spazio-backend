-- name: ListPopularServices :many
SELECT
    s.service_id,
    s.code,
    s.icon,
    sc.code AS category_code,
    COUNT(*) OVER() AS total_count
FROM services AS s
JOIN service_categories AS sc ON sc.category_id = s.category_id
LEFT JOIN property_services AS ps ON ps.service_id = s.service_id
WHERE s.is_active = true
  AND s.is_deprecated = false
GROUP BY s.service_id, s.code, s.icon, sc.code, s.sort_order
ORDER BY COUNT(ps.property_id) DESC, s.sort_order ASC
LIMIT $1;

-- name: SearchServices :many
SELECT
    s.service_id,
    s.code,
    s.icon,
    sc.code AS category_code,
    COUNT(*) OVER() AS total_count
FROM services AS s
JOIN service_categories AS sc ON sc.category_id = s.category_id
WHERE s.is_active = true
  AND s.is_deprecated = false
  AND s.code ILIKE '%' || sqlc.arg(query) || '%'
ORDER BY s.sort_order ASC, s.service_id ASC
LIMIT sqlc.arg(search_limit);
