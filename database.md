# Spazio Data Model (PostgreSQL)

---

## 1. USERS, SECURITY & RBAC

### [Users]

- `user_id` (SERIAL, PK)
- `user_uuid` (UUID, UNIQUE)
- `role_id` (INT, FK → Roles)
- `first_name` (VARCHAR 80)
- `last_name` (VARCHAR 80)
- `email` (VARCHAR 150, UNIQUE)
- `phone` (VARCHAR 20)
- `profile_picture_url` (VARCHAR 255, Nullable)
- `status_id` (INT, FK → UserStatus)
- `created_at` (TIMESTAMPTZ, DEFAULT NOW())
- `updated_at` (TIMESTAMPTZ, DEFAULT NOW())
- `deleted_at` (TIMESTAMPTZ, Nullable)

### [Roles]

- `role_id` (SERIAL, PK)
- `name` (VARCHAR 50)

### [UserStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 30)

### [Permissions]

- `permission_id` (SERIAL, PK)
- `code` (VARCHAR 60, UNIQUE)
- `description` (TEXT)

### [RolePermissions]

- `role_id` (INT, FK → Roles)
- `permission_id` (INT, FK → Permissions)
- PRIMARY KEY (`role_id`, `permission_id`)

---

## 2. PROPERTIES (CORE)

### [Properties]

- `property_id` (SERIAL, PK)
- `property_uuid` (UUID, UNIQUE, DEFAULT gen_random_uuid())
- `owner_id` (INT, FK → Users)
- `title` (VARCHAR 128)
- `description` (TEXT)
- `property_type_id` (INT, FK → PropertyTypes)
- `modality_id` (INT, FK → Modalities)
- `status_id` (INT, FK → PropertyStatus)
- `cover_photo_url` (VARCHAR 255, Nullable)
- `is_featured` (BOOLEAN, DEFAULT false)
- `published_at` (TIMESTAMPTZ, Nullable)
- `updated_at` (TIMESTAMPTZ, Nullable)
- `created_at` (TIMESTAMPTZ, DEFAULT NOW())
- `deleted_at` (TIMESTAMPTZ, Nullable)
- `lot_area` (DECIMAL 12,2, Nullable)
- `category` (property_category NOT NULL)

### [PropertyTypes]

- `property_type_id` (SERIAL, PK)
- `name` (VARCHAR 50)
- `icon` (VARCHAR 80)
- `is_deprecated` (BOOLEAN, DEFAULT false)

### [Modalities]

- `modality_id` (SERIAL, PK)
- `name` (VARCHAR 50)

### [PropertyStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50)

---

## 3. SPECIALIZATION (INHERITANCE)

### [ResidentialProperties]

- `property_id` (INT, PK, FK → Properties)
- `bedrooms` (SMALLINT)
- `bathrooms` (SMALLINT)
- `beds` (SMALLINT)
- `floors` (SMALLINT)
- `parking_spots` (SMALLINT)
- `built_area` (DECIMAL 12,2)
- `construction_year` (SMALLINT)
- `orientation_id` (INT, FK → Orientations)
- `is_furnished` (BOOLEAN, DEFAULT false)

### [CommercialProperties]

- `property_id` (INT, PK, FK → Properties)
- `ceiling_height` (DECIMAL 5,2)
- `loading_docks` (SMALLINT)
- `internal_offices` (SMALLINT)
- `three_phase_power` (BOOLEAN)
- `land_use` (VARCHAR 100)

### [Orientations]

- `orientation_id` (SERIAL, PK)
- `name` (VARCHAR 30)

---

## 4. LOCATION & GEOGRAPHY

### [Cities]

- `city_id` (SERIAL, PK)
- `state_id` (INT NOT NULL, FK → States)
- `name` (VARCHAR 80) NOT NULL

### [Locations]

- `location_id` (SERIAL, PK)
- `property_id` (INT, FK → Properties)
- `city_id` (INT NOT NULL, FK → Cities)
- `zone_id` (INT, FK → Zones, Nullable)
- `neighborhood` (VARCHAR 60)
- `street` (VARCHAR 120)
- `exterior_number` (VARCHAR 20)
- `interior_number` (VARCHAR 20, Nullable)
- `postal_code` (VARCHAR 10)
- `postal_code_id` (INT, FK → PostalCodes, Nullable)
- `coordinates` (GEOMETRY(Point, 4326), NOT NULL)
- `is_public_address` (BOOLEAN, DEFAULT true)

### [Countries]

- `country_id` (SERIAL, PK)
- `iso2_code` (CHAR 2, UNIQUE)
- `name` (VARCHAR 60)
- `is_active` (BOOLEAN, DEFAULT true)

### [States]

- `state_id` (SERIAL, PK)
- `country_id` (INT, FK → Countries)
- `iso_code` (VARCHAR 10)
- `name` (VARCHAR 60)
- `is_active` (BOOLEAN, DEFAULT true)

### [Zones]

- `zone_id` (SERIAL, PK)
- `state_id` (INT, FK → States)
- `parent_zone_id` (INT, FK → Zones, Nullable)
- `zone_type` (VARCHAR 30)
- `name` (VARCHAR 60)
- `description` (VARCHAR 255, Nullable)
- `is_active` (BOOLEAN, DEFAULT true)
- `postal_code_id` (INT, FK → PostalCodes, Nullable)

---

### [PostalCodes]

- `postal_code_id` (SERIAL, PK)
- `code` (VARCHAR 10)
- `city_id` (INT, FK → Cities)
- `state_id` (INT, FK → States)
- `source` (VARCHAR 30, DEFAULT 'manual')
- `created_at` (TIMESTAMPTZ, DEFAULT NOW())
- `updated_at` (TIMESTAMPTZ, DEFAULT NOW())

### [PostalCodeZones]

- `postal_code_id` (INT, FK → PostalCodes)
- `zone_id` (INT, FK → Zones)
- PRIMARY KEY (`postal_code_id`, `zone_id`)

## 5. FINANCIALS (APPEND-ONLY PRICES)

### [Prices]

- `price_id` (SERIAL, PK)
- `property_id` (INT, FK → Properties)
- `sale_price` (DECIMAL 15,2, Nullable)
- `rent_price` (DECIMAL 15,2, Nullable)
- `deposit` (DECIMAL 15,2, Nullable)
- `currency` (CHAR 3, DEFAULT 'MXN')
- `period_id` (INT, FK → RentPeriods, Nullable)
- `is_negotiable` (BOOLEAN, DEFAULT false)
- `is_current` (BOOLEAN, DEFAULT true)
- `valid_from` (TIMESTAMPTZ, DEFAULT NOW())
- `valid_until` (TIMESTAMPTZ, Nullable)
- `change_reason` (VARCHAR 100, Nullable)
- `changed_by_user_id` (INT, FK → Users)

### [RentPeriods]

- `period_id` (SERIAL, PK)
- `name` (VARCHAR 50)

---

## 6. MULTIMEDIA & ANALYTICS

### [PropertyPhotos]

- `photo_id` (SERIAL, PK)
- `property_id` (INT, FK → Properties)
- `storage_key` (VARCHAR 255, NOT NULL)
- `mime_type` (VARCHAR 30)
- `sort_order` (SMALLINT, DEFAULT 0)
- `is_cover` (BOOLEAN, DEFAULT false)
- `label` (VARCHAR 60, Nullable)
- `alt_text` (VARCHAR 255, Nullable)
- `created_at` (TIMESTAMPTZ, DEFAULT NOW())

### [PropertyEvents]

- `event_id` (BIGSERIAL)
- `property_id` (INT, FK → Properties)
- `user_id` (INT, FK → Users, Nullable)
- `event_type` (VARCHAR 30)
- `occurred_at` (TIMESTAMPTZ, DEFAULT NOW())
- PRIMARY KEY (`event_id`, `occurred_at`)

---

## 7. SERVICES & CLAUSES (METADATA)

### [Services]

- `service_id` (SERIAL, PK)
- `code` (VARCHAR 40, UNIQUE)
- `icon` (VARCHAR 80)
- `category_id` (INT, FK → ServiceCategories)
- `is_active` (BOOLEAN, DEFAULT true)
- `is_deprecated` (BOOLEAN, DEFAULT false)
- `sort_order` (INT)

### [ServiceCategories]

- `category_id` (SERIAL, PK)
- `code` (VARCHAR 40, UNIQUE)
- `name` (VARCHAR 80)

### [PropertyServices]

- `property_id` (INT, FK → Properties)
- `service_id` (INT, FK → Services)
- `assigned_at` (TIMESTAMPTZ, DEFAULT NOW())
- PRIMARY KEY (`property_id`, `service_id`)

### [Clauses]

- `clause_id` (SERIAL, PK)
- `code` (VARCHAR 40, UNIQUE)
- `name` (VARCHAR 100)
- `value_type_id` (INT, FK → ClauseValueTypes)
- `is_active` (BOOLEAN, DEFAULT true)
- `is_deprecated` (BOOLEAN, DEFAULT false)
- `sort_order` (INT)

### [ClauseValueTypes]

- `value_type_id` (SERIAL, PK)
- `code` (VARCHAR 40)
- `name` (VARCHAR 80)

### [PropertyClauses]

- `property_clause_id` (SERIAL, PK)
- `property_id` (INT, FK → Properties)
- `clause_id` (INT, FK → Clauses)
- `boolean_value` (BOOLEAN, Nullable)
- `integer_value` (INT, Nullable)
- `min_value` (DECIMAL 12,2, Nullable)
- `max_value` (DECIMAL 12,2, Nullable)
- `assigned_at` (TIMESTAMPTZ, DEFAULT NOW())

### [ClauseModalities]

- `clause_id` (INT, FK → Clauses)
- `modality_id` (INT, FK → Modalities)
- PRIMARY KEY (`clause_id`, `modality_id`)

---

## 8. CRM & LOGISTICS

### [Inquiries]

- `inquiry_id` (SERIAL, PK)
- `inquiry_uuid` (UUID, UNIQUE, DEFAULT gen_random_uuid())
- `property_id` (INT, FK → Properties)
- `user_id` (INT, FK → Users, Nullable)
- `message` (TEXT)
- `follow_up_status_id` (INT, FK → FollowUpStatus)
- `created_at` (TIMESTAMPTZ, DEFAULT NOW())

### [FollowUpStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50)

### [PropertyAgents]

- `property_id` (INT, FK → Properties)
- `agent_id` (INT, FK → Users)
- `is_primary` (BOOLEAN, DEFAULT true)
- `assigned_at` (TIMESTAMPTZ, DEFAULT NOW())
- PRIMARY KEY (`property_id`, `agent_id`)

### [AgentSchedules]

- `schedule_id` (SERIAL, PK)
- `agent_id` (INT, FK → Users)
- `day_of_week` (SMALLINT)
- `start_time` (TIME)
- `end_time` (TIME)
- `is_active` (BOOLEAN, DEFAULT true)

### [PropertyExceptions]

- `exception_id` (SERIAL, PK)
- `property_id` (INT, FK → Properties)
- `exception_date` (DATE)
- `reason` (VARCHAR 100)
- `start_time` (TIME, Nullable)
- `end_time` (TIME, Nullable)

### [Visits]

- `visit_id` (SERIAL, PK)
- `visit_uuid` (UUID, UNIQUE, DEFAULT gen_random_uuid())
- `property_id` (INT, FK → Properties)
- `client_id` (INT, FK → Users)
- `agent_id` (INT, FK → Users, Nullable)
- `visit_date` (TIMESTAMPTZ)
- `status_id` (INT, FK → VisitStatus)
- `created_at` (TIMESTAMPTZ, DEFAULT NOW())

### [VisitStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50)

---

## 9. OPERATIONS, CONTRACTS & PAYMENTS

### [Transactions]

- `transaction_id` (SERIAL, PK)
- `transaction_uuid` (UUID, UNIQUE, DEFAULT gen_random_uuid())
- `property_id` (INT, FK → Properties)
- `client_id` (INT, FK → Users)
- `agent_id` (INT, FK → Users)
- `transaction_type` (ENUM)
- `status_id` (INT, FK → TransactionStatus)
- `final_amount` (DECIMAL 15,2)
- `closing_date` (DATE)

### [TransactionStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50)

### [Contracts]

- `contract_id` (SERIAL, PK)
- `contract_uuid` (UUID, UNIQUE, DEFAULT gen_random_uuid())
- `transaction_id` (INT, FK → Transactions)
- `currency` (CHAR 3)
- `agreed_amount` (DECIMAL 15,2)
- `storage_key` (VARCHAR 255, NOT NULL)
- `start_date` (DATE)
- `end_date` (DATE, Nullable)
- `status_id` (INT, FK → ContractStatus)
- `created_at` (TIMESTAMPTZ, DEFAULT NOW())
- `updated_at` (TIMESTAMPTZ, DEFAULT NOW())
- `deleted_at` (TIMESTAMPTZ, Nullable)
- `parent_contract_id` (INT, FK → Contracts, Nullable)

### [ContractStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50)

### [Payments]

- `payment_id` (SERIAL, PK)
- `payment_uuid` (UUID, UNIQUE, DEFAULT gen_random_uuid())
- `contract_id` (INT, FK → Contracts)
- `billing_period` (DATE)
- `due_date` (DATE)
- `amount` (DECIMAL 15,2)
- `payment_method_id` (INT, FK → PaymentMethods)
- `gateway_id` (INT, FK → PaymentGateways, Nullable)
- `gateway_payment_id` (VARCHAR 100, Nullable)
- `gateway_order_id` (VARCHAR 100, Nullable)
- `status_id` (INT, FK → PaymentStatus)
- `gateway_status` (VARCHAR 50, Nullable)
- `payment_date` (TIMESTAMPTZ, Nullable)
- `metadata` (JSONB, Nullable)

### [PaymentGateways]

- `gateway_id` (SERIAL, PK)
- `name` (VARCHAR 50)
- `is_active` (BOOLEAN, DEFAULT true)

### [PaymentMethods]

- `method_id` (SERIAL, PK)
- `name` (VARCHAR 50)
- `is_active` (BOOLEAN, DEFAULT true)

### [PaymentStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 30)
- `is_active` (BOOLEAN, DEFAULT true)

---

## 10. AUDIT & HISTORY

### [PropertyStatusHistory]

- `history_id` (SERIAL, PK)
- `property_id` (INT, FK → Properties)
- `previous_status_id` (INT, FK → PropertyStatus)
- `new_status_id` (INT, FK → PropertyStatus)
- `changed_by_user_id` (INT, FK → Users)
- `changed_at` (TIMESTAMPTZ, DEFAULT NOW())

### [ContractStatusHistory]

- `history_id` (SERIAL, PK)
- `contract_id` (INT, FK → Contracts)
- `previous_status_id` (INT, FK → ContractStatus)
- `new_status_id` (INT, FK → ContractStatus)
- `changed_by_user_id` (INT, FK → Users)
- `changed_at` (TIMESTAMPTZ, DEFAULT NOW())

### [VisitStatusHistory]

- `history_id` (SERIAL, PK)
- `visit_id` (INT, FK → Visits)
- `previous_status_id` (INT, FK → VisitStatus)
- `new_status_id` (INT, FK → VisitStatus)
- `changed_by_user_id` (INT, FK → Users)
- `changed_at` (TIMESTAMPTZ, DEFAULT NOW())

### [TransactionStatusHistory]

- `history_id` (SERIAL, PK)
- `transaction_id` (INT, FK → Transactions)
- `previous_status_id` (INT, FK → TransactionStatus)
- `new_status_id` (INT, FK → TransactionStatus)
- `changed_by_user_id` (INT, FK → Users)
- `changed_at` (TIMESTAMPTZ, DEFAULT NOW())
