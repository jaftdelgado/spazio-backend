# Spazio Data Model (PostgreSQL)

---

## 1. USERS, SECURITY & RBAC

Granular access control and user profile management.

### [Users]

- `user_id` (SERIAL, PK)
- `user_uuid` (UUID, UNIQUE, DEFAULT gen_random_uuid())
- `role_id` (INT, FK → Roles)
- `first_name` (VARCHAR 80)
- `last_name` (VARCHAR 80)
- `email` (VARCHAR 150, UNIQUE)
- `password_hash` (VARCHAR 255)
- `phone` (VARCHAR 20)
- `profile_picture_url` (VARCHAR 255)
- `status_id` (INT, FK → UserStatus)
- `created_at` (TIMESTAMPTZ, DEFAULT NOW())
- `updated_at` (TIMESTAMPTZ, DEFAULT NOW())
- `deleted_at` (TIMESTAMPTZ, Nullable)

### [Roles]

- `role_id` (SERIAL, PK)
- `name` (VARCHAR 50) -- e.g., 'Admin', 'Agent', 'Client'

### [UserStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 30) -- e.g., 'Active', 'Inactive', 'Suspended'

### [Permissions]

- `permission_id` (SERIAL, PK)
- `code` (VARCHAR 60, UNIQUE) -- e.g., 'PROPERTIES_EDIT', 'CONTRACTS_SIGN'
- `description` (TEXT)

### [RolePermissions]

- `role_id` (INT, FK → Roles)
- `permission_id` (INT, FK → Permissions)
- PRIMARY KEY (`role_id`, `permission_id`)

---

## 2. PROPERTIES (CORE)

Base entities for real estate management.

### [Properties]

- `property_id` (SERIAL, PK)
- `property_uuid` (UUID, UNIQUE, DEFAULT gen_random_uuid())
- `owner_id` (INT, FK → Users) -- The legal owner/landlord
- `current_resident_id` (INT, FK → Users, Nullable) -- Current Tenant or Buyer
- `title` (VARCHAR 128)
- `description` (TEXT)
- `property_type_id` (INT, FK → PropertyTypes)
- `modality_id` (INT, FK → Modalities)
- `status_id` (INT, FK → PropertyStatus)
- `cover_photo_url` (VARCHAR 255)
- `is_featured` (BOOLEAN, DEFAULT false)
- `published_at` (TIMESTAMPTZ, Nullable)
- `updated_at` (TIMESTAMPTZ, Nullable)
- `created_at` (TIMESTAMPTZ, DEFAULT NOW())
- `deleted_at` (TIMESTAMPTZ, Nullable)

### [PropertyTypes]

- `property_type_id` (SERIAL, PK)
- `name` (VARCHAR 50) -- e.g., 'House', 'Apartment', 'Commercial'
- `icon` (VARCHAR 80)
- `is_deprecated` (BOOLEAN, DEFAULT false)

### [Modalities]

- `modality_id` (SERIAL, PK)
- `name` (VARCHAR 50) -- e.g., 'Sale', 'Rent', 'Both'

### [PropertyStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50) -- e.g., 'Available', 'Reserved', 'Sold', 'Rented'

---

## 3. SPECIALIZATION (INHERITANCE)

Specific attributes for different property types.

### [ResidentialProperties]

- `property_id` (INT, PK, FK → Properties)
- `bedrooms` (SMALLINT)
- `bathrooms` (SMALLINT)
- `beds` (SMALLINT)
- `floors` (SMALLINT)
- `parking_spots` (SMALLINT)
- `built_area` (DECIMAL 12,2)
- `lot_area` (DECIMAL 12,2)
- `construction_year` (SMALLINT)
- `orientation_id` (INT, FK → Orientations)
- `is_furnished` (BOOLEAN, DEFAULT false)

### [CommercialProperties]

- `property_id` (INT, PK, FK → Properties)
- `ceiling_height` (DECIMAL 5,2)
- `loading_docks` (SMALLINT)
- `internal_offices` (SMALLINT)
- `three_phase_power` (BOOLEAN)
- `lot_area` (DECIMAL 12,2)
- `land_use` (VARCHAR 100)

### [Orientations]

- `orientation_id` (SERIAL, PK)
- `name` (VARCHAR 30) -- e.g., 'North', 'South', 'East', 'West'

---

## 4. LOCATION & GEOGRAPHY

Hierarchical geo-management with PostGIS integration.

### [Locations]

- `location_id` (SERIAL, PK)
- `property_id` (INT, FK → Properties)
- `country_id` (INT, FK → Countries)
- `state_id` (INT, FK → States)
- `zone_id` (INT, FK → Zones, Nullable)
- `city` (VARCHAR 60)
- `neighborhood` (VARCHAR 60)
- `street` (VARCHAR 120)
- `exterior_number` (VARCHAR 20)
- `interior_number` (VARCHAR 20, Nullable)
- `postal_code` (VARCHAR 10)
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
- `zone_type` (VARCHAR 30) -- e.g., 'County', 'District', 'Sector'
- `name` (VARCHAR 60)
- `description` (VARCHAR 255, Nullable)
- `is_active` (BOOLEAN, DEFAULT true)

---

## 5. FINANCIALS (APPEND-ONLY PRICES)

Immutable snapshots for property pricing.

### [Prices]

- `price_id` (SERIAL, PK)
- `property_id` (INT, FK → Properties)
- `sale_price` (DECIMAL 15,2, Nullable)
- `rent_price` (DECIMAL 15,2, Nullable)
- `deposit` (DECIMAL 15,2, Nullable)
- `currency` (CHAR 3, DEFAULT 'MXN')
- `period_id` (INT, FK → RentPeriods)
- `is_negotiable` (BOOLEAN, DEFAULT false)
- `valid_from` (TIMESTAMPTZ, DEFAULT NOW())
- `valid_until` (TIMESTAMPTZ, Nullable) -- NULL means 'currently active'
- `change_reason` (VARCHAR 100, Nullable)
- `changed_by_user_id` (INT, FK → Users)

### [RentPeriods]

- `period_id` (SERIAL, PK)
- `name` (VARCHAR 50) -- e.g., 'Monthly', 'Weekly'

---

## 6. MULTIMEDIA & ANALYTICS

High-volume tracking optimized for performance with partitioning.

### [PropertyPhotos]

- `photo_id` (SERIAL, PK)
- `property_id` (INT, FK → Properties)
- `storage_key` (VARCHAR 255, NOT NULL)
- `mime_type` (VARCHAR 30)
- `sort_order` (SMALLINT, DEFAULT 0)
- `is_cover` (BOOLEAN, DEFAULT false)
- `label` (VARCHAR 60, Nullable)
- `alt_text` (VARCHAR 255, Nullable) -- Accessibility Requirement (WCAG)
- `created_at` (TIMESTAMPTZ, DEFAULT NOW())

### [PropertyEvents]

- `event_id` (BIGSERIAL)
- `property_id` (INT, FK → Properties)
- `user_id` (INT, FK → Users, Nullable)
- `event_type` (VARCHAR 30) -- e.g., 'View', 'Save', 'Call'
- `occurred_at` (TIMESTAMPTZ, DEFAULT NOW())
- PRIMARY KEY (`event_id`, `occurred_at`)
- **Note:** Partitioned by range (occurred_at) monthly.

---

## 7. SERVICES & CLAUSES (METADATA)

Dynamic property features with versioning.

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
- `description` (TEXT)
- `value_type_id` (INT, FK → ClauseValueTypes)
- `icon` (VARCHAR 80)
- `is_active` (BOOLEAN, DEFAULT true)
- `is_deprecated` (BOOLEAN, DEFAULT false)
- `sort_order` (INT)

### [ClauseValueTypes]

- `value_type_id` (SERIAL, PK)
- `code` (VARCHAR 40) -- boolean, integer, range
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

---

## 8. CRM & LOGISTICS

Lead management, agent assignments, and visit scheduling.

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
- `name` (VARCHAR 50) -- e.g., 'New', 'Interested', 'Closed'

### [PropertyAgents]

- `property_id` (INT, FK → Properties)
- `agent_id` (INT, FK → Users)
- `is_primary` (BOOLEAN, DEFAULT true)
- `assigned_at` (TIMESTAMPTZ, DEFAULT NOW())
- PRIMARY KEY (`property_id`, `agent_id`)

### [AgentSchedules]

- `schedule_id` (SERIAL, PK)
- `agent_id` (INT, FK → Users)
- `day_of_week` (SMALLINT) -- 0-6
- `start_time` (TIME)
- `end_time` (TIME)
- `is_active` (BOOLEAN, DEFAULT true)
- **Constraint:** UNIQUE (agent_id, day_of_week, start_time, end_time)

### [PropertyExceptions]

- `exception_id` (SERIAL, PK)
- `property_id` (INT, FK → Properties)
- `exception_date` (DATE)
- `reason` (VARCHAR 100)

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
- `name` (VARCHAR 50) -- e.g., 'Pending', 'Completed', 'Canceled'

---

## 9. OPERATIONS, CONTRACTS & PAYMENTS

Business logic for deal closing and digital transactions.

### [Transactions]

- `transaction_id` (SERIAL, PK)
- `transaction_uuid` (UUID, UNIQUE, DEFAULT gen_random_uuid())
- `property_id` (INT, FK → Properties)
- `client_id` (INT, FK → Users)
- `agent_id` (INT, FK → Users)
- `transaction_type` (ENUM: 'sale', 'rent')
- `status_id` (INT, FK → TransactionStatus)
- `final_amount` (DECIMAL 15,2)
- `closing_date` (DATE)

### [TransactionStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50) -- e.g., 'In Progress', 'Finalized', 'Canceled'

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

### [ContractStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 50) -- e.g., 'Draft', 'Signed', 'Expired'

### [Payments]

- `payment_id` (SERIAL, PK)
- `payment_uuid` (UUID, UNIQUE, DEFAULT gen_random_uuid())
- `contract_id` (INT, FK → Contracts)
- `client_id` (INT, FK → Users)
- `billing_period` (DATE) -- CHECK (Day = 1)
- `due_date` (DATE)
- `amount` (DECIMAL 15,2)
- `payment_method_id` (INT, FK → PaymentMethods)
- `gateway_id` (INT, FK → PaymentGateways)
- `gateway_payment_id` (VARCHAR 100, Nullable)
- `gateway_order_id` (VARCHAR 100, Nullable)
- `status_id` (INT, FK → PaymentStatus)
- `gateway_status` (VARCHAR 50, Nullable)
- `payment_date` (TIMESTAMPTZ, Nullable)
- `metadata` (JSONB, Nullable)

### [PaymentGateways]

- `gateway_id` (SERIAL, PK)
- `name` (VARCHAR 50) -- e.g., 'Stripe', 'MercadoPago'
- `is_active` (BOOLEAN, DEFAULT true)

### [PaymentMethods]

- `method_id` (SERIAL, PK)
- `name` (VARCHAR 50) -- e.g., 'Credit Card', 'PayPal'
- `is_active` (BOOLEAN, DEFAULT true)

### [PaymentStatus]

- `status_id` (SERIAL, PK)
- `name` (VARCHAR 30) -- e.g., 'Paid', 'Pending', 'Overdue'
- `is_active` (BOOLEAN, DEFAULT true)

---

## 10. AUDIT & HISTORY

Consistent status history pattern applied to all operational entities.

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
