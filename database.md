# Spazio Data Model (PostgreSQL)

## 1. Módulo de Seguridad y Usuarios (RBAC)

**Clase: roles**

- `role_id` (SERIAL, PK)
- `name` (VARCHAR 50, UNIQUE)

**Clase: user_status**

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 30)

**Clase: permissions**

- `permission_id` (SERIAL, PK)
- `code` (VARCHAR 60, UNIQUE)
- `description` (TEXT)

**Clase: role_permissions**

- `role_id` (INT, PK, FK -> roles)
- `permission_id` (INT, PK, FK -> permissions)

**Clase: users**

- `user_id` (SERIAL, PK)
- `user_uuid` (UUID, UNIQUE) -- UID Sincronizado con Supabase
- `role_id` (INT, FK -> roles)
- `first_name` (VARCHAR 80)
- `last_name` (VARCHAR 80)
- `email` (VARCHAR 150, UNIQUE)
- `phone` (VARCHAR 20)
- `profile_picture_url` (VARCHAR 255, NULL)
- `status_id` (INT, FK -> user_status)
- `created_at` (TIMESTAMPTZ)
- `updated_at` (TIMESTAMPTZ)
- `deleted_at` (TIMESTAMPTZ, NULL)

---

## 2. Módulo Geográfico Normalizado

**Clase: countries**

- `country_id` (SERIAL, PK)
- `iso2_code` (CHAR 2, UNIQUE)
- `name` (VARCHAR 60)
- `is_active` (BOOLEAN)

**Clase: states**

- `state_id` (SERIAL, PK)
- `country_id` (INT, FK -> countries)
- `iso_code` (VARCHAR 10)
- `name` (VARCHAR 60)
- `is_active` (BOOLEAN)

**Clase: cities**

- `city_id` (SERIAL, PK)
- `state_id` (INT, FK -> states)
- `name` (VARCHAR 80)

---

## 3. Módulo de Propiedades (Core y Herencia)

**Clase: property_types**

- `property_type_id` (SERIAL, PK)
- `name` (VARCHAR 50)
- `icon` (VARCHAR 80)
- `is_deprecated` (BOOLEAN)

**Clase: modalities**

- `modality_id` (SERIAL, PK)
- `name` (VARCHAR 50)

**Clase: property_status**

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50)

**Clase: properties**

- `property_id` (SERIAL, PK)
- `property_uuid` (UUID, UNIQUE)
- `owner_id` (INT, FK -> users)
- `category` (ENUM: residential, commercial, land, other)
- `title` (VARCHAR 128)
- `description` (TEXT)
- `property_type_id` (INT, FK -> property_types)
- `modality_id` (INT, FK -> modalities)
- `status_id` (INT, FK -> property_status)
- `cover_photo_url` (VARCHAR 255, NULL)
- `lot_area` (DECIMAL 12,2)
- `is_featured` (BOOLEAN)
- `published_at` (TIMESTAMPTZ, NULL)
- `created_at` (TIMESTAMPTZ)
- `updated_at` (TIMESTAMPTZ)
- `deleted_at` (TIMESTAMPTZ, NULL)

**Clase: locations**

- `location_id` (SERIAL, PK)
- `property_id` (INT, UNIQUE, FK -> properties)
- `city_id` (INT, FK -> cities)
- `zone_id` (INT, FK -> zones, NULL)
- `postal_code_id` (INT, FK -> postal_codes, NULL)
- `neighborhood` (VARCHAR 60)
- `street` (VARCHAR 120)
- `exterior_number` (VARCHAR 20)
- `interior_number` (VARCHAR 20, NULL)
- `postal_code` (VARCHAR 10) -- Texto redundante para compatibilidad
- `coordinates` (GEOMETRY Point)
- `is_public_address` (BOOLEAN)

**Clase: orientations**

- `orientation_id` (SERIAL, PK)
- `name` (VARCHAR 30)

**Clase: residential_properties**

- `property_id` (INT, PK, FK -> properties)
- `bedrooms` (SMALLINT)
- `bathrooms` (SMALLINT)
- `beds` (SMALLINT)
- `floors` (SMALLINT)
- `parking_spots` (SMALLINT)
- `built_area` (DECIMAL 12,2)
- `construction_year` (SMALLINT)
- `orientation_id` (INT, FK -> orientations)
- `is_furnished` (BOOLEAN)

**Clase: commercial_properties**

- `property_id` (INT, PK, FK -> properties)
- `ceiling_height` (DECIMAL 5,2)
- `loading_docks` (SMALLINT)
- `internal_offices` (SMALLINT)
- `three_phase_power` (BOOLEAN)
- `land_use` (VARCHAR 100)

---

## 4. Módulo Financiero y Multimedia

**Clase: rent_periods**

- `period_id` (SERIAL, PK)
- `name` (VARCHAR 50)

**Clase: sale_prices**

- `price_id` (SERIAL, PK)
- `property_id` (INT, FK -> properties)
- `sale_price` (DECIMAL 15,2)
- `currency` (CHAR 3)
- `is_negotiable` (BOOLEAN)
- `is_current` (BOOLEAN)
- `valid_from` (TIMESTAMPTZ)
- `valid_until` (TIMESTAMPTZ, NULL)
- `change_reason` (VARCHAR 100, NULL)
- `changed_by_user_id` (INT, FK -> users)

**Clase: rent_prices**

- `rent_price_id` (SERIAL, PK)
- `property_id` (INT, FK -> properties)
- `period_id` (INT, FK -> rent_periods)
- `rent_price` (DECIMAL 15,2)
- `deposit` (DECIMAL 15,2, NULL)
- `currency` (CHAR 3)
- `is_negotiable` (BOOLEAN)
- `is_current` (BOOLEAN)
- `valid_from` (TIMESTAMPTZ)
- `valid_until` (TIMESTAMPTZ, NULL)
- `change_reason` (VARCHAR 100, NULL)
- `changed_by_user_id` (INT, FK -> users)

**Clase: property_photos**

- `photo_id` (SERIAL, PK)
- `property_id` (INT, FK -> properties)
- `storage_key` (VARCHAR 255)
- `mime_type` (VARCHAR 30)
- `sort_order` (SMALLINT)
- `is_cover` (BOOLEAN)
- `label` (VARCHAR 60, NULL)
- `alt_text` (VARCHAR 255, NULL)

---

## 5. Módulo de Servicios y Cláusulas

**Clase: service_categories**

- `category_id` (SERIAL, PK)
- `code` (VARCHAR 40)
- `name` (VARCHAR 80)

**Clase: services**

- `service_id` (SERIAL, PK)
- `code` (VARCHAR 40)
- `icon` (VARCHAR 80)
- `category_id` (INT, FK -> service_categories)
- `is_active` (BOOLEAN)
- `is_deprecated` (BOOLEAN)
- `sort_order` (INT)

**Clase: clause_value_types**

- `value_type_id` (SERIAL, PK)
- `code` (VARCHAR 40)
- `name` (VARCHAR 80)

**Clase: clauses**

- `clause_id` (SERIAL, PK)
- `code` (VARCHAR 40)
- `name` (VARCHAR 100)
- `description` (TEXT)
- `value_type_id` (INT, FK -> clause_value_types)
- `icon` (VARCHAR 80)
- `is_active` (BOOLEAN)

**Clase: clause_modalities**

- `clause_id` (INT, PK, FK -> clauses)
- `modality_id` (INT, PK, FK -> modalities)

**Clase: property_services**

- `property_id` (INT, PK, FK -> properties)
- `service_id` (INT, PK, FK -> services)

**Clase: property_clauses**

- `property_clause_id` (SERIAL, PK)
- `property_id` (INT, FK -> properties)
- `clause_id` (INT, FK -> clauses)
- `boolean_value` (BOOLEAN, NULL)
- `integer_value` (INT, NULL)
- `min_value` (DECIMAL 12,2, NULL)
- `max_value` (DECIMAL 12,2, NULL)

---

## 6. Módulo CRM y Logística

**Clase: follow_up_status**

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50)

**Clase: visit_status**

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50)

**Clase: inquiries**

- `inquiry_id` (SERIAL, PK)
- `property_id` (INT, FK -> properties)
- `user_id` (INT, FK -> users, NULL)
- `message` (TEXT)
- `follow_up_status_id` (INT, FK -> follow_up_status)

**Clase: property_agents**

- `property_id` (INT, PK, FK -> properties)
- `agent_id` (INT, PK, FK -> users)
- `is_primary` (BOOLEAN)

**Clase: agent_schedules**

- `schedule_id` (SERIAL, PK)
- `agent_id` (INT, FK -> users)
- `day_of_week` (SMALLINT)
- `start_time` (TIME)
- `end_time` (TIME)
- `is_active` (BOOLEAN)

**Clase: property_exceptions**

- `exception_id` (SERIAL, PK)
- `property_id` (INT, FK -> properties)
- `exception_date` (DATE)
- `reason` (VARCHAR 100)

**Clase: visits**

- `visit_id` (SERIAL, PK)
- `property_id` (INT, FK -> properties)
- `client_id` (INT, FK -> users)
- `agent_id` (INT, FK -> users, NULL)
- `visit_date` (TIMESTAMPTZ)
- `status_id` (INT, FK -> visit_status)

---

## 7. Módulo de Operaciones y Auditoría

**Clase: transaction_status**

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50)

**Clase: contract_status**

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50)

**Clase: payment_gateways**

- `gateway_id` (SERIAL, PK)
- `name` (VARCHAR 50)

**Clase: payment_methods**

- `method_id` (SERIAL, PK)
- `name` (VARCHAR 50)

**Clase: payment_status**

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 30)

**Clase: transactions**

- `transaction_id` (SERIAL, PK)
- `property_id` (INT, FK -> properties)
- `client_id` (INT, FK -> users)
- `agent_id` (INT, FK -> users)
- `transaction_type` (ENUM: sale, rent)
- `status_id` (INT, FK -> transaction_status)
- `final_amount` (DECIMAL 15,2)
- `closing_date` (DATE)

**Clase: contracts**

- `contract_id` (SERIAL, PK)
- `transaction_id` (INT, FK -> transactions)
- `parent_contract_id` (INT, FK -> contracts, NULL)
- `currency` (CHAR 3)
- `agreed_amount` (DECIMAL 15,2)
- `storage_key` (VARCHAR 255)
- `start_date` (DATE)
- `end_date` (DATE, NULL)
- `status_id` (INT, FK -> contract_status)

**Clase: payments**

- `payment_id` (SERIAL, PK)
- `contract_id` (INT, FK -> contracts)
- `billing_period` (DATE)
- `due_date` (DATE)
- `amount` (DECIMAL 15,2)
- `payment_method_id` (INT, FK -> payment_methods)
- `gateway_id` (INT, FK -> payment_gateways, NULL)
- `status_id` (INT, FK -> payment_status)
- `payment_date` (TIMESTAMPTZ, NULL)

**Clase: status_history**

- `history_id` (SERIAL, PK)
- `property_id` (INT, FK -> properties)
- `previous_status_id` (INT, FK -> property_status)
- `new_status_id` (INT, FK -> property_status)
- `changed_by_user_id` (INT, FK -> users)
- `changed_at` (TIMESTAMPTZ)
- `entity_type` (VARCHAR 30) -- property, contract, visit, transaction
