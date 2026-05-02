-- name: GetModalityName :one
SELECT name
FROM modalities
WHERE modality_id = $1;

-- name: CreateProperty :one
INSERT INTO properties (
  property_uuid, owner_id, category, title, description,
  property_type_id, modality_id, status_id, lot_area, is_featured,
  created_at, updated_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, 2, $8, $9, NOW(), NOW()
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
  $1, $2, $3, $4, $5, $6, $7,
  ST_SetSRID(ST_MakePoint($8, $9), 4326),
  $10
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
