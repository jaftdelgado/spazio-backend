-- name: GetModalityName :one
SELECT name
FROM modalities
WHERE modality_id = $1;

-- name: CreateProperty :one
INSERT INTO properties (
  property_uuid, owner_id, title, description,
  property_type_id, modality_id, status_id, lot_area, is_featured,
  created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, 2, $7, $8, NOW(), NOW()
) RETURNING property_id, property_uuid;

-- name: CreateResidentialProperty :exec
INSERT INTO residential_properties (
  property_id, bedrooms, bathrooms, beds, floors, parking_spots,
  built_area, construction_year, orientation_id, is_furnished
) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: CreateCommercialProperty :exec
INSERT INTO commercial_properties (
  property_id, ceiling_height, loading_docks, internal_offices,
  three_phase_power, land_use
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: CreateLocation :exec
INSERT INTO locations (
  property_id, city_id, neighborhood, street, exterior_number,
  interior_number, postal_code, coordinates, is_public_address
) VALUES (
  sqlc.arg(property_id),
  sqlc.arg(city_id),
  sqlc.arg(neighborhood),
  sqlc.arg(street),
  sqlc.arg(exterior_number),
  sqlc.arg(interior_number),
  sqlc.arg(postal_code),
  ST_SetSRID(ST_MakePoint(sqlc.arg(longitude), sqlc.arg(latitude)), 4326),
  sqlc.arg(is_public_address)
);

-- name: CreateSalePrice :exec
INSERT INTO sale_prices (
  property_id, sale_price, currency, is_negotiable,
  is_current, valid_from, changed_by_user_id
) VALUES ($1, $2, $3, $4, true, NOW(), $5);

-- name: CreateRentPrice :exec
INSERT INTO rent_prices (
  property_id, period_id, rent_price, deposit, currency,
  is_negotiable, is_current, valid_from, changed_by_user_id
) VALUES ($1, $2, $3, $4, $5, $6, true, NOW(), $7);

-- name: CreatePropertyService :exec
INSERT INTO property_services (property_id, service_id)
VALUES ($1, $2)
ON CONFLICT DO NOTHING;

-- name: CreatePropertyClause :exec
INSERT INTO property_clauses (
  property_id, clause_id, boolean_value, integer_value, min_value, max_value
) VALUES ($1, $2, $3, $4, $5, $6);

-- name: GetClauseValueTypes :many
SELECT clause_id, value_type_id
FROM clauses
WHERE clause_id = ANY($1::int[]);

-- name: GetAllowedPeriods :many
SELECT period_id
FROM property_type_periods
WHERE property_type_id = $1;

-- name: GetPropertySubtype :one
SELECT subtype
FROM property_types
WHERE property_type_id = $1;

-- name: ListPropertiesCards :many
WITH sale_current AS (
  SELECT
    property_id,
    sale_price,
    currency
  FROM sale_prices
  WHERE is_current = true
    AND (sqlc.arg(min_price)::numeric = 0 OR sale_price >= sqlc.arg(min_price)::numeric)
    AND (sqlc.arg(max_price)::numeric = 0 OR sale_price <= sqlc.arg(max_price)::numeric)
),
ranked_rent_prices AS (
  SELECT
    rp.property_id,
    rp.rent_price,
    rp.currency,
    rper.name AS period_name,
    ROW_NUMBER() OVER (
      PARTITION BY rp.property_id
      ORDER BY
        CASE
          WHEN LOWER(rper.name) LIKE '%month%' OR LOWER(rper.name) LIKE '%mens%' THEN 1
          WHEN LOWER(rper.name) LIKE '%year%' OR LOWER(rper.name) LIKE '%an%' THEN 2
          WHEN LOWER(rper.name) LIKE '%week%' OR LOWER(rper.name) LIKE '%seman%' THEN 3
          WHEN LOWER(rper.name) LIKE '%day%' OR LOWER(rper.name) LIKE '%dia%' THEN 4
          ELSE 100
        END,
        rp.period_id ASC
    ) AS rent_rank
  FROM rent_prices rp
  JOIN rent_periods rper ON rper.period_id = rp.period_id
  WHERE rp.is_current = true
    AND (sqlc.arg(min_price)::numeric = 0 OR rp.rent_price >= sqlc.arg(min_price)::numeric)
    AND (sqlc.arg(max_price)::numeric = 0 OR rp.rent_price <= sqlc.arg(max_price)::numeric)
),
selected_rent AS (
  SELECT
    property_id,
    rent_price,
    currency,
    period_name
  FROM ranked_rent_prices
  WHERE rent_rank = 1
)
SELECT
  p.property_id,
  p.property_uuid,
  p.title,
  p.cover_photo_url,
  p.is_featured,
  pt.property_type_id,
  pt.name AS property_type_name,
  pt.icon AS property_type_icon,
  m.modality_id,
  m.name AS modality_name,
  ps.status_id,
  ps.name AS status_name,
  CAST(
    COALESCE(sc.sale_price, sr.rent_price) AS numeric
  ) AS display_price_amount,
  CAST(
    COALESCE(sc.currency, sr.currency, '') AS text
  ) AS display_price_currency,
  CAST(
    CASE
      WHEN sc.sale_price IS NOT NULL THEN 'sale'
      WHEN sr.rent_price IS NOT NULL THEN 'rent'
      ELSE ''
    END AS text
  ) AS display_price_type,
  CAST(
    CASE
      WHEN sc.sale_price IS NULL THEN COALESCE(sr.period_name, '')
      ELSE ''
    END AS text
  ) AS display_period_name,
  co.country_id,
  COALESCE(co.name, '') AS country_name,
  st.state_id,
  COALESCE(st.name, '') AS state_name,
  ci.city_id,
  COALESCE(ci.name, '') AS city_name,
  COALESCE(l.neighborhood, '') AS neighborhood,
  CAST(
    NULLIF(
      CONCAT_WS(
        ', ',
        NULLIF(
          TRIM(
            CONCAT_WS(
              ' ',
              NULLIF(l.street, ''),
              NULLIF(l.exterior_number, '')
            )
          ),
          ''
        ),
        NULLIF(l.neighborhood, ''),
        NULLIF(ci.name, ''),
        NULLIF(st.name, ''),
        NULLIF(co.name, '')
      ),
      ''
    ) AS text
  ) AS address_summary,
  res.bedrooms,
  res.bathrooms,
  res.parking_spots,
  res.built_area,
  EXISTS (
    SELECT 1
    FROM property_clauses pc
    JOIN clauses cl ON cl.clause_id = pc.clause_id
    WHERE pc.property_id = p.property_id
      AND cl.code = 'pets_allowed'
      AND COALESCE(pc.boolean_value, false) = true
  ) AS pet_friendly,
  COUNT(*) OVER() AS total_count
FROM properties p
JOIN property_types pt ON pt.property_type_id = p.property_type_id
JOIN modalities m ON m.modality_id = p.modality_id
JOIN property_status ps ON ps.status_id = p.status_id
LEFT JOIN residential_properties res ON res.property_id = p.property_id
LEFT JOIN locations l ON l.property_id = p.property_id
LEFT JOIN cities ci ON ci.city_id = l.city_id
LEFT JOIN states st ON st.state_id = ci.state_id
LEFT JOIN countries co ON co.country_id = st.country_id
LEFT JOIN sale_current sc ON sc.property_id = p.property_id
LEFT JOIN selected_rent sr ON sr.property_id = p.property_id
WHERE p.deleted_at IS NULL
  AND (
    sqlc.arg(search_query)::text = ''
    OR p.title ILIKE '%' || sqlc.arg(search_query)::text || '%'
    OR l.street ILIKE '%' || sqlc.arg(search_query)::text || '%'
    OR l.neighborhood ILIKE '%' || sqlc.arg(search_query)::text || '%'
    OR ci.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
    OR st.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
    OR co.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
  )
  AND (
    cardinality(sqlc.arg(status_ids)::int[]) = 0
    OR p.status_id = ANY(sqlc.arg(status_ids)::int[])
  )
  AND (
    sqlc.arg(property_type_id)::int = 0
    OR p.property_type_id = sqlc.arg(property_type_id)::int
  )
  AND (
    sqlc.arg(modality_id)::int = 0
    OR p.modality_id = sqlc.arg(modality_id)::int
  )
  AND (
    sqlc.arg(country_id)::int = 0
    OR co.country_id = sqlc.arg(country_id)::int
  )
  AND (
    sqlc.arg(state_id)::int = 0
    OR st.state_id = sqlc.arg(state_id)::int
  )
  AND (
    sqlc.arg(city_id)::int = 0
    OR ci.city_id = sqlc.arg(city_id)::int
  )
  AND (
    sqlc.arg(min_bedrooms)::int = 0
    OR (pt.subtype = 'residential' AND res.bedrooms >= sqlc.arg(min_bedrooms)::int)
  )
  AND (
    sqlc.narg('is_featured')::boolean IS NULL
    OR p.is_featured = sqlc.narg('is_featured')::boolean
  )
  AND (
    sqlc.arg(min_parking_spots)::int = 0
    OR (pt.subtype = 'residential' AND res.parking_spots >= sqlc.arg(min_parking_spots)::int)
  )
  AND (
    sqlc.arg(pet_friendly)::boolean = false
    OR EXISTS (
      SELECT 1
      FROM property_clauses pc
      JOIN clauses cl ON cl.clause_id = pc.clause_id
      WHERE pc.property_id = p.property_id
        AND cl.code = 'pets_allowed'
        AND COALESCE(pc.boolean_value, false) = true
    )
  )
  AND (
    (sqlc.arg(min_price)::numeric = 0 AND sqlc.arg(max_price)::numeric = 0)
    OR (sc.property_id IS NOT NULL OR sr.property_id IS NOT NULL)
  )
ORDER BY
  CASE WHEN sqlc.arg(sort_field)::text = '' THEN p.is_featured END DESC,
  CASE WHEN sqlc.arg(sort_field)::text = '' THEN p.created_at END DESC,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'created_at' AND sqlc.arg(sort_order)::text = 'asc' THEN p.created_at
  END ASC,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'created_at' AND sqlc.arg(sort_order)::text = 'desc' THEN p.created_at
  END DESC,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'title' AND sqlc.arg(sort_order)::text = 'asc' THEN p.title
  END ASC,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'title' AND sqlc.arg(sort_order)::text = 'desc' THEN p.title
  END DESC,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'price' AND sqlc.arg(sort_order)::text = 'asc' THEN COALESCE(sc.sale_price, sr.rent_price)
  END ASC NULLS LAST,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'price' AND sqlc.arg(sort_order)::text = 'desc' THEN COALESCE(sc.sale_price, sr.rent_price)
  END DESC NULLS LAST,
  p.property_id DESC
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: ListPropertiesCardsForAgent :many
WITH sale_current AS (
  SELECT
    property_id,
    sale_price,
    currency
  FROM sale_prices
  WHERE is_current = true
    AND (sqlc.arg(min_price)::numeric = 0 OR sale_price >= sqlc.arg(min_price)::numeric)
    AND (sqlc.arg(max_price)::numeric = 0 OR sale_price <= sqlc.arg(max_price)::numeric)
),
ranked_rent_prices AS (
  SELECT
    rp.property_id,
    rp.rent_price,
    rp.currency,
    rper.name AS period_name,
    ROW_NUMBER() OVER (
      PARTITION BY rp.property_id
      ORDER BY
        CASE
          WHEN LOWER(rper.name) LIKE '%month%' OR LOWER(rper.name) LIKE '%mens%' THEN 1
          WHEN LOWER(rper.name) LIKE '%year%' OR LOWER(rper.name) LIKE '%an%' THEN 2
          WHEN LOWER(rper.name) LIKE '%week%' OR LOWER(rper.name) LIKE '%seman%' THEN 3
          WHEN LOWER(rper.name) LIKE '%day%' OR LOWER(rper.name) LIKE '%dia%' THEN 4
          ELSE 100
        END,
        rp.period_id ASC
    ) AS rent_rank
  FROM rent_prices rp
  JOIN rent_periods rper ON rper.period_id = rp.period_id
  WHERE rp.is_current = true
    AND (sqlc.arg(min_price)::numeric = 0 OR rp.rent_price >= sqlc.arg(min_price)::numeric)
    AND (sqlc.arg(max_price)::numeric = 0 OR rp.rent_price <= sqlc.arg(max_price)::numeric)
),
selected_rent AS (
  SELECT
    property_id,
    rent_price,
    currency,
    period_name
  FROM ranked_rent_prices
  WHERE rent_rank = 1
)
SELECT
  p.property_id,
  p.property_uuid,
  p.title,
  p.cover_photo_url,
  p.is_featured,
  pt.property_type_id,
  pt.name AS property_type_name,
  pt.icon AS property_type_icon,
  m.modality_id,
  m.name AS modality_name,
  ps.status_id,
  ps.name AS status_name,
  CAST(
    COALESCE(sc.sale_price, sr.rent_price) AS numeric
  ) AS display_price_amount,
  CAST(
    COALESCE(sc.currency, sr.currency, '') AS text
  ) AS display_price_currency,
  CAST(
    CASE
      WHEN sc.sale_price IS NOT NULL THEN 'sale'
      WHEN sr.rent_price IS NOT NULL THEN 'rent'
      ELSE ''
    END AS text
  ) AS display_price_type,
  CAST(
    CASE
      WHEN sc.sale_price IS NULL THEN COALESCE(sr.period_name, '')
      ELSE ''
    END AS text
  ) AS display_period_name,
  co.country_id,
  COALESCE(co.name, '') AS country_name,
  st.state_id,
  COALESCE(st.name, '') AS state_name,
  ci.city_id,
  COALESCE(ci.name, '') AS city_name,
  COALESCE(l.neighborhood, '') AS neighborhood,
  CAST(
    NULLIF(
      CONCAT_WS(
        ', ',
        NULLIF(
          TRIM(
            CONCAT_WS(
              ' ',
              NULLIF(l.street, ''),
              NULLIF(l.exterior_number, '')
            )
          ),
          ''
        ),
        NULLIF(l.neighborhood, ''),
        NULLIF(ci.name, ''),
        NULLIF(st.name, ''),
        NULLIF(co.name, '')
      ),
      ''
    ) AS text
  ) AS address_summary,
  res.bedrooms,
  res.bathrooms,
  res.parking_spots,
  res.built_area,
  EXISTS (
    SELECT 1
    FROM property_clauses pc
    JOIN clauses cl ON cl.clause_id = pc.clause_id
    WHERE pc.property_id = p.property_id
      AND cl.code = 'pets_allowed'
      AND COALESCE(pc.boolean_value, false) = true
  ) AS pet_friendly,
  COUNT(*) OVER() AS total_count
FROM properties p
JOIN property_types pt ON pt.property_type_id = p.property_type_id
JOIN modalities m ON m.modality_id = p.modality_id
JOIN property_status ps ON ps.status_id = p.status_id
JOIN property_agents pa ON pa.property_id = p.property_id
  AND pa.agent_id = sqlc.arg(agent_id)::int
LEFT JOIN residential_properties res ON res.property_id = p.property_id
LEFT JOIN locations l ON l.property_id = p.property_id
LEFT JOIN cities ci ON ci.city_id = l.city_id
LEFT JOIN states st ON st.state_id = ci.state_id
LEFT JOIN countries co ON co.country_id = st.country_id
LEFT JOIN sale_current sc ON sc.property_id = p.property_id
LEFT JOIN selected_rent sr ON sr.property_id = p.property_id
WHERE p.deleted_at IS NULL
  AND (
    sqlc.arg(search_query)::text = ''
    OR p.title ILIKE '%' || sqlc.arg(search_query)::text || '%'
    OR l.street ILIKE '%' || sqlc.arg(search_query)::text || '%'
    OR l.neighborhood ILIKE '%' || sqlc.arg(search_query)::text || '%'
    OR ci.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
    OR st.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
    OR co.name ILIKE '%' || sqlc.arg(search_query)::text || '%'
  )
  AND (
    cardinality(sqlc.arg(status_ids)::int[]) = 0
    OR p.status_id = ANY(sqlc.arg(status_ids)::int[])
  )
  AND (
    sqlc.arg(property_type_id)::int = 0
    OR p.property_type_id = sqlc.arg(property_type_id)::int
  )
  AND (
    sqlc.arg(modality_id)::int = 0
    OR p.modality_id = sqlc.arg(modality_id)::int
  )
  AND (
    sqlc.arg(country_id)::int = 0
    OR co.country_id = sqlc.arg(country_id)::int
  )
  AND (
    sqlc.arg(state_id)::int = 0
    OR st.state_id = sqlc.arg(state_id)::int
  )
  AND (
    sqlc.arg(city_id)::int = 0
    OR ci.city_id = sqlc.arg(city_id)::int
  )
  AND (
    sqlc.arg(min_bedrooms)::int = 0
    OR (pt.subtype = 'residential' AND res.bedrooms >= sqlc.arg(min_bedrooms)::int)
  )
  AND (
    sqlc.narg('is_featured')::boolean IS NULL
    OR p.is_featured = sqlc.narg('is_featured')::boolean
  )
  AND (
    sqlc.arg(min_parking_spots)::int = 0
    OR (pt.subtype = 'residential' AND res.parking_spots >= sqlc.arg(min_parking_spots)::int)
  )
  AND (
    sqlc.arg(pet_friendly)::boolean = false
    OR EXISTS (
      SELECT 1
      FROM property_clauses pc
      JOIN clauses cl ON cl.clause_id = pc.clause_id
      WHERE pc.property_id = p.property_id
        AND cl.code = 'pets_allowed'
        AND COALESCE(pc.boolean_value, false) = true
    )
  )
  AND (
    (sqlc.arg(min_price)::numeric = 0 AND sqlc.arg(max_price)::numeric = 0)
    OR (sc.property_id IS NOT NULL OR sr.property_id IS NOT NULL)
  )
ORDER BY
  CASE WHEN sqlc.arg(sort_field)::text = '' THEN p.is_featured END DESC,
  CASE WHEN sqlc.arg(sort_field)::text = '' THEN p.created_at END DESC,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'created_at' AND sqlc.arg(sort_order)::text = 'asc' THEN p.created_at
  END ASC,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'created_at' AND sqlc.arg(sort_order)::text = 'desc' THEN p.created_at
  END DESC,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'title' AND sqlc.arg(sort_order)::text = 'asc' THEN p.title
  END ASC,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'title' AND sqlc.arg(sort_order)::text = 'desc' THEN p.title
  END DESC,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'price' AND sqlc.arg(sort_order)::text = 'asc' THEN COALESCE(sc.sale_price, sr.rent_price)
  END ASC NULLS LAST,
  CASE
    WHEN sqlc.arg(sort_field)::text = 'price' AND sqlc.arg(sort_order)::text = 'desc' THEN COALESCE(sc.sale_price, sr.rent_price)
  END DESC NULLS LAST,
  p.property_id DESC
LIMIT sqlc.arg(page_size)
OFFSET sqlc.arg(page_offset);

-- name: ListPropertyClauses :many
SELECT
  clause_id,
  boolean_value,
  integer_value,
  min_value,
  max_value
FROM property_clauses
WHERE property_id = $1
ORDER BY property_clause_id ASC;

-- name: DeletePropertyClauses :exec
DELETE FROM property_clauses
WHERE property_id = $1;

-- name: ListPropertyPhotos :many
SELECT
  photo_id,
  storage_key,
  mime_type,
  sort_order,
  is_cover,
  label,
  alt_text
FROM property_photos
WHERE property_id = $1
ORDER BY sort_order ASC, photo_id ASC;

-- name: ListPropertyPhotosByIDs :many
SELECT
  photo_id,
  storage_key,
  mime_type,
  sort_order,
  is_cover,
  label,
  alt_text
FROM property_photos
WHERE property_id = $1
  AND photo_id = ANY($2::int[])
ORDER BY photo_id ASC;

-- name: DeletePropertyPhotos :exec
DELETE FROM property_photos
WHERE property_id = $1;

-- name: DeletePropertyPhotosExceptIDs :exec
DELETE FROM property_photos
WHERE property_id = $1
  AND NOT (photo_id = ANY($2::int[]));

-- name: UpdatePropertyPhotoMetadata :exec
UPDATE property_photos
SET
  sort_order = $3,
  is_cover = $4,
  label = $5,
  alt_text = $6
WHERE property_id = $1
  AND photo_id = $2;

-- name: ListPropertyServiceIDs :many
SELECT service_id
FROM property_services
WHERE property_id = $1
ORDER BY service_id ASC;

-- name: DeletePropertyServices :exec
DELETE FROM property_services
WHERE property_id = $1;

-- name: GetPropertyOwnerID :one
SELECT owner_id
FROM properties
WHERE property_id = $1;

-- name: ListActiveSalePrice :one
SELECT
  sale_price,
  currency,
  is_negotiable
FROM sale_prices
WHERE property_id = $1 AND is_current = true;

-- name: ListActiveRentPrices :many
SELECT
  period_id,
  rent_price,
  deposit,
  currency,
  is_negotiable
FROM rent_prices
WHERE property_id = $1 AND is_current = true
ORDER BY period_id ASC;

-- name: ListPropertyPriceTimeline :many
SELECT
  'sale'::text AS price_type,
  sp.sale_price AS amount,
  sp.currency,
  NULL::text AS period_name,
  sp.is_negotiable,
  NULL::numeric AS deposit,
  sp.valid_from,
  sp.valid_until,
  sp.is_current
FROM sale_prices sp
WHERE sp.property_id = $1

UNION ALL

SELECT
  'rent'::text AS price_type,
  rp.rent_price AS amount,
  rp.currency,
  rper.name AS period_name,
  rp.is_negotiable,
  rp.deposit,
  rp.valid_from,
  rp.valid_until,
  rp.is_current
FROM rent_prices rp
JOIN rent_periods rper ON rper.period_id = rp.period_id
WHERE rp.property_id = $1
ORDER BY valid_from DESC, price_type ASC, period_name ASC NULLS LAST;

-- name: ListActiveRentPriceByPeriod :one
SELECT
  rent_price,
  deposit,
  currency,
  is_negotiable
FROM rent_prices
WHERE property_id = $1 AND period_id = $2 AND is_current = true;

-- name: UpdateSalePriceToInactive :exec
UPDATE sale_prices
SET is_current = false, valid_until = NOW()
WHERE property_id = $1 AND is_current = true;

-- name: CreateSalePriceHistoryRecord :exec
INSERT INTO sale_prices (
  property_id, sale_price, currency, is_negotiable,
  is_current, valid_from, changed_by_user_id
) VALUES ($1, $2, $3, $4, true, NOW(), $5);

-- name: UpdateSalePriceIsNegotiable :exec
UPDATE sale_prices
SET is_negotiable = $2
WHERE property_id = $1 AND is_current = true;

-- name: UpdateRentPriceToInactive :exec
UPDATE rent_prices
SET is_current = false, valid_until = NOW()
WHERE property_id = $1 AND period_id = $2 AND is_current = true;

-- name: CreateRentPriceHistoryRecord :exec
INSERT INTO rent_prices (
  property_id, period_id, rent_price, deposit, currency,
  is_negotiable, is_current, valid_from, changed_by_user_id
) VALUES ($1, $2, $3, $4, $5, $6, true, NOW(), $7);

-- name: UpdateRentPriceIsNegotiableAndDeposit :exec
UPDATE rent_prices
SET is_negotiable = $3, deposit = $4
WHERE property_id = $1 AND period_id = $2 AND is_current = true;

-- name: GetPropertyBaseByUUID :one
SELECT
  p.property_uuid,
  p.property_id,
  p.owner_id,
  p.title,
  p.description,
  p.property_type_id,
  p.modality_id,
  p.status_id,
  p.lot_area,
  p.is_featured,
  pt.subtype,
  u.first_name || ' ' || u.last_name AS registered_by
FROM properties p
JOIN property_types pt ON pt.property_type_id = p.property_type_id
JOIN users u ON u.user_id = p.owner_id
WHERE p.property_uuid = $1 AND p.deleted_at IS NULL;

-- name: IsPropertyAssignedToAgent :one
SELECT EXISTS (
  SELECT 1
  FROM property_agents
  WHERE property_id = $1 AND agent_id = $2
) AS assigned;

-- name: GetResidentialByPropertyID :one
SELECT
  bedrooms,
  bathrooms,
  beds,
  floors,
  parking_spots,
  built_area,
  construction_year,
  orientation_id,
  is_furnished
FROM residential_properties
WHERE property_id = $1;

-- name: GetCommercialByPropertyID :one
SELECT
  ceiling_height,
  loading_docks,
  internal_offices,
  three_phase_power,
  land_use
FROM commercial_properties
WHERE property_id = $1;

-- name: GetLocationByPropertyID :one
SELECT
  co.country_id,
  co.name AS country_name,
  st.state_id,
  st.name AS state_name,
  ci.city_id,
  ci.name AS city_name,
  l.neighborhood,
  l.street,
  l.exterior_number,
  l.interior_number,
  l.postal_code,
  ST_Y(l.coordinates)::float8 AS latitude,
  ST_X(l.coordinates)::float8 AS longitude,
  l.is_public_address
FROM locations l
JOIN cities ci ON ci.city_id = l.city_id
JOIN states st ON st.state_id = ci.state_id
JOIN countries co ON co.country_id = st.country_id
WHERE l.property_id = $1;

-- name: UpdatePropertyBaseByID :exec
UPDATE properties
SET title = $2,
    description = $3,
    lot_area = $4,
    is_featured = $5,
    updated_at = NOW()
WHERE property_id = $1;

-- name: UpdateResidentialPropertyByID :exec
UPDATE residential_properties
SET bedrooms = $2,
    bathrooms = $3,
    beds = $4,
    floors = $5,
    parking_spots = $6,
    built_area = $7,
    construction_year = $8,
    orientation_id = $9,
    is_furnished = $10
WHERE property_id = $1;

-- name: UpdateCommercialPropertyByID :exec
UPDATE commercial_properties
SET ceiling_height = $2,
    loading_docks = $3,
    internal_offices = $4,
    three_phase_power = $5,
    land_use = $6
WHERE property_id = $1;

-- name: UpdateLocationByID :exec
UPDATE locations
SET city_id = sqlc.arg(city_id),
    neighborhood = sqlc.arg(neighborhood),
    street = sqlc.arg(street),
    exterior_number = sqlc.arg(exterior_number),
    interior_number = sqlc.arg(interior_number),
    postal_code = sqlc.arg(postal_code),
    coordinates = ST_SetSRID(ST_MakePoint(sqlc.arg(longitude), sqlc.arg(latitude)), 4326),
    is_public_address = sqlc.arg(is_public_address)
WHERE property_id = sqlc.arg(property_id);

-- name: GetPropertyStorageKeys :many
SELECT storage_key
FROM property_photos
WHERE property_id = $1
ORDER BY photo_id ASC;

-- name: SoftDeleteProperty :exec
UPDATE properties
SET deleted_at = NOW(),
    status_id = 5
WHERE property_id = $1;

-- name: ClearPropertyCoverPhotoURL :exec
UPDATE properties
SET cover_photo_url = NULL
WHERE property_id = $1;

-- name: InsertPropertyStatusHistory :exec
INSERT INTO property_status_history (
  property_id,
  previous_status_id,
  new_status_id,
  changed_by_user_id
) VALUES ($1, $2, $3, $4);

-- name: ListPropertyStatusHistory :many
SELECT 
    psh.history_id,
    p.property_uuid,
    prev.name AS previous_status_name,
    curr.name AS new_status_name,
    u.first_name || ' ' || u.last_name AS changed_by_name,
    psh.changed_at
FROM property_status_history psh
JOIN property_status prev ON psh.previous_status_id = prev.status_id
JOIN property_status curr ON psh.new_status_id = curr.status_id
JOIN users u ON psh.changed_by_user_id = u.user_id
JOIN properties p ON psh.property_id = p.property_id
WHERE p.property_uuid = $1 AND p.deleted_at IS NULL
ORDER BY psh.changed_at ASC;

-- name: GetPropertyPricesHistory :many
SELECT
  'sale'::text AS price_type,
  sp.sale_price AS amount,
  sp.currency,
  NULL::text AS period_name,
  sp.is_negotiable,
  NULL::numeric AS deposit,
  sp.valid_from,
  sp.valid_until,
  sp.is_current
FROM sale_prices sp
JOIN properties p ON p.property_id = sp.property_id
WHERE p.property_uuid = $1 AND p.deleted_at IS NULL

UNION ALL

SELECT
  'rent'::text AS price_type,
  rp.rent_price AS amount,
  rp.currency,
  rper.name AS period_name,
  rp.is_negotiable,
  rp.deposit,
  rp.valid_from,
  rp.valid_until,
  rp.is_current
FROM rent_prices rp
JOIN rent_periods rper ON rper.period_id = rp.period_id
JOIN properties p ON p.property_id = rp.property_id
WHERE p.property_uuid = $1 AND p.deleted_at IS NULL
ORDER BY valid_from DESC, price_type ASC, period_name ASC NULLS LAST;
