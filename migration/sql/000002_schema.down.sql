-- ## 10. audit & history
DROP TABLE IF EXISTS transaction_status_history CASCADE;
DROP TABLE IF EXISTS visit_status_history CASCADE;
DROP TABLE IF EXISTS contract_status_history CASCADE;
DROP TABLE IF EXISTS property_status_history CASCADE;

-- ## 9. operations, contracts & payments
DROP TABLE IF EXISTS payments CASCADE;
DROP TABLE IF EXISTS contracts CASCADE;
DROP TABLE IF EXISTS transactions CASCADE;
DROP TABLE IF EXISTS payment_status CASCADE;
DROP TABLE IF EXISTS payment_methods CASCADE;
DROP TABLE IF EXISTS payment_gateways CASCADE;
DROP TABLE IF EXISTS contract_status CASCADE;
DROP TABLE IF EXISTS transaction_status CASCADE;
DROP TYPE IF EXISTS transaction_type;

-- ## 8. crm & logistics
DROP TABLE IF EXISTS visits CASCADE;
DROP TABLE IF EXISTS property_exceptions CASCADE;
DROP TABLE IF EXISTS agent_schedules CASCADE;
DROP TABLE IF EXISTS property_agents CASCADE;
DROP TABLE IF EXISTS inquiries CASCADE;
DROP TABLE IF EXISTS visit_status CASCADE;
DROP TABLE IF EXISTS follow_up_status CASCADE;

-- ## 7. services & clauses (metadata)
DROP TABLE IF EXISTS property_clauses CASCADE;
DROP TABLE IF EXISTS property_services CASCADE;
DROP TABLE IF EXISTS clauses CASCADE;
DROP TABLE IF EXISTS clause_value_types CASCADE;
DROP TABLE IF EXISTS services CASCADE;
DROP TABLE IF EXISTS service_categories CASCADE;

-- ## 6. multimedia & analytics
DROP TABLE IF EXISTS property_events_2026_04 CASCADE;
DROP TABLE IF EXISTS property_events CASCADE;
DROP TABLE IF EXISTS property_photos CASCADE;

-- ## 5. financials (append-only prices)
DROP TABLE IF EXISTS prices CASCADE;
DROP TABLE IF EXISTS rent_periods CASCADE;

-- ## 4. location & geography
DROP TABLE IF EXISTS locations CASCADE;
DROP TABLE IF EXISTS zones CASCADE;
DROP TABLE IF EXISTS states CASCADE;
DROP TABLE IF EXISTS countries CASCADE;

-- ## 3. specialization (inheritance)
DROP TABLE IF EXISTS commercial_properties CASCADE;
DROP TABLE IF EXISTS residential_properties CASCADE;
DROP TABLE IF EXISTS orientations CASCADE;

-- ## 2. properties (core)
DROP TABLE IF EXISTS properties CASCADE;
DROP TABLE IF EXISTS property_status CASCADE;
DROP TABLE IF EXISTS modalities CASCADE;
DROP TABLE IF EXISTS property_types CASCADE;

-- ## 1. users, security & rbac
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS user_status CASCADE;
DROP TABLE IF EXISTS roles CASCADE;

DROP EXTENSION IF EXISTS "postgis";
DROP EXTENSION IF EXISTS "pgcrypto";
