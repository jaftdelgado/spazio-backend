CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "postgis";

-- ## 1. users, security & rbac
CREATE TABLE IF NOT EXISTS roles (
    role_id serial PRIMARY KEY,
    name varchar(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS user_status (
    status_id serial PRIMARY KEY,
    name varchar(30) NOT NULL
);

CREATE TABLE IF NOT EXISTS permissions (
    permission_id serial PRIMARY KEY,
    code varchar(60) NOT NULL UNIQUE,
    description text NOT NULL
);

CREATE TABLE IF NOT EXISTS users (
    user_id serial PRIMARY KEY,
    user_uuid uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    role_id int NOT NULL REFERENCES roles(role_id),
    first_name varchar(80) NOT NULL,
    last_name varchar(80) NOT NULL,
    email varchar(150) NOT NULL UNIQUE,
    password_hash varchar(255) NOT NULL,
    phone varchar(20) NOT NULL,
    profile_picture_url varchar(255) NOT NULL,
    status_id int NOT NULL REFERENCES user_status(status_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE IF NOT EXISTS role_permissions (
    role_id int NOT NULL REFERENCES roles(role_id),
    permission_id int NOT NULL REFERENCES permissions(permission_id),
    PRIMARY KEY (role_id, permission_id)
);

CREATE INDEX IF NOT EXISTS idx_users_role_id ON users(role_id);
CREATE INDEX IF NOT EXISTS idx_users_status_id ON users(status_id);
CREATE INDEX IF NOT EXISTS idx_role_permissions_permission_id ON role_permissions(permission_id);

-- ## 2. properties (core)
CREATE TABLE IF NOT EXISTS property_types (
    property_type_id serial PRIMARY KEY,
    name varchar(50) NOT NULL,
    icon varchar(80) NOT NULL,
    is_deprecated boolean NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS modalities (
    modality_id serial PRIMARY KEY,
    name varchar(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS property_status (
    status_id serial PRIMARY KEY,
    name varchar(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS properties (
    property_id serial PRIMARY KEY,
    property_uuid uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    owner_id int NOT NULL REFERENCES users(user_id),
    current_resident_id int references users(user_id),
    title varchar(128) NOT NULL,
    description text NOT NULL,
    property_type_id int NOT NULL REFERENCES property_types(property_type_id),
    modality_id int NOT NULL REFERENCES modalities(modality_id),
    status_id int NOT NULL REFERENCES property_status(status_id),
    cover_photo_url varchar(255) NOT NULL,
    is_featured boolean NOT NULL DEFAULT false,
    published_at timestamptz,
    updated_at timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_properties_owner_id ON properties(owner_id);
CREATE INDEX IF NOT EXISTS idx_properties_current_resident_id ON properties(current_resident_id);
CREATE INDEX IF NOT EXISTS idx_properties_property_type_id ON properties(property_type_id);
CREATE INDEX IF NOT EXISTS idx_properties_modality_id ON properties(modality_id);
CREATE INDEX IF NOT EXISTS idx_properties_status_id ON properties(status_id);
CREATE INDEX IF NOT EXISTS idx_properties_is_featured ON properties(is_featured);
CREATE INDEX IF NOT EXISTS idx_properties_published_at ON properties(published_at);

-- ## 3. specialization (inheritance)
CREATE TABLE IF NOT EXISTS orientations (
    orientation_id serial PRIMARY KEY,
    name varchar(30) NOT NULL
);

CREATE TABLE IF NOT EXISTS residential_properties (
    property_id int PRIMARY KEY REFERENCES properties(property_id),
    bedrooms smallint NOT NULL,
    bathrooms smallint NOT NULL,
    beds smallint NOT NULL,
    floors smallint NOT NULL,
    parking_spots smallint NOT NULL,
    built_area decimal(12,2) NOT NULL,
    lot_area decimal(12,2) NOT NULL,
    construction_year smallint NOT NULL,
    orientation_id int NOT NULL REFERENCES orientations(orientation_id),
    is_furnished boolean NOT NULL DEFAULT false
);

CREATE TABLE IF NOT EXISTS commercial_properties (
    property_id int PRIMARY KEY REFERENCES properties(property_id),
    ceiling_height decimal(5,2) NOT NULL,
    loading_docks smallint NOT NULL,
    internal_offices smallint NOT NULL,
    three_phase_power boolean NOT NULL,
    lot_area decimal(12,2) NOT NULL,
    land_use varchar(100) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_residential_properties_orientation_id ON residential_properties(orientation_id);

-- ## 4. location & geography
CREATE TABLE IF NOT EXISTS countries (
    country_id serial PRIMARY KEY,
    iso2_code char(2) NOT NULL UNIQUE,
    name varchar(60) NOT NULL,
    is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS states (
    state_id serial PRIMARY KEY,
    country_id int NOT NULL REFERENCES countries(country_id),
    iso_code varchar(10) NOT NULL,
    name varchar(60) NOT NULL,
    is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS zones (
    zone_id serial PRIMARY KEY,
    state_id int NOT NULL REFERENCES states(state_id),
    parent_zone_id int references zones(zone_id),
    zone_type varchar(30) NOT NULL,
    name varchar(60) NOT NULL,
    description varchar(255),
    is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS locations (
    location_id serial PRIMARY KEY,
    property_id int NOT NULL REFERENCES properties(property_id),
    country_id int NOT NULL REFERENCES countries(country_id),
    state_id int NOT NULL REFERENCES states(state_id),
    zone_id int references zones(zone_id),
    city varchar(60) NOT NULL,
    neighborhood varchar(60) NOT NULL,
    street varchar(120) NOT NULL,
    exterior_number varchar(20) NOT NULL,
    interior_number varchar(20),
    postal_code varchar(10) NOT NULL,
    coordinates geometry(point, 4326) NOT NULL,
    is_public_address boolean NOT NULL DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_states_country_id ON states(country_id);
CREATE INDEX IF NOT EXISTS idx_zones_state_id ON zones(state_id);
CREATE INDEX IF NOT EXISTS idx_zones_parent_zone_id ON zones(parent_zone_id);
CREATE INDEX IF NOT EXISTS idx_locations_property_id ON locations(property_id);
CREATE INDEX IF NOT EXISTS idx_locations_country_id ON locations(country_id);
CREATE INDEX IF NOT EXISTS idx_locations_state_id ON locations(state_id);
CREATE INDEX IF NOT EXISTS idx_locations_zone_id ON locations(zone_id);
CREATE INDEX IF NOT EXISTS idx_locations_coordinates ON locations USING gist(coordinates);

-- ## 5. financials (append-only prices)
CREATE TABLE IF NOT EXISTS rent_periods (
    period_id serial PRIMARY KEY,
    name varchar(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS prices (
    price_id serial PRIMARY KEY,
    property_id int NOT NULL REFERENCES properties(property_id),
    sale_price decimal(15,2),
    rent_price decimal(15,2),
    deposit decimal(15,2),
    currency char(3) NOT NULL DEFAULT 'MXN',
    period_id int NOT NULL REFERENCES rent_periods(period_id),
    is_negotiable boolean NOT NULL DEFAULT false,
    valid_from timestamptz NOT NULL DEFAULT now(),
    valid_until timestamptz,
    change_reason varchar(100),
    changed_by_user_id int NOT NULL REFERENCES users(user_id)
);

CREATE INDEX IF NOT EXISTS idx_prices_property_id ON prices(property_id);
CREATE INDEX IF NOT EXISTS idx_prices_period_id ON prices(period_id);
CREATE INDEX IF NOT EXISTS idx_prices_changed_by_user_id ON prices(changed_by_user_id);

-- ## 6. multimedia & analytics
CREATE TABLE IF NOT EXISTS property_photos (
    photo_id serial PRIMARY KEY,
    property_id int NOT NULL REFERENCES properties(property_id),
    storage_key varchar(255) NOT NULL,
    mime_type varchar(30) NOT NULL,
    sort_order smallint NOT NULL DEFAULT 0,
    is_cover boolean NOT NULL DEFAULT false,
    label varchar(60),
    alt_text varchar(255),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS property_events (
    event_id bigserial NOT NULL,
    property_id int NOT NULL REFERENCES properties(property_id),
    user_id int references users(user_id),
    event_type varchar(30) NOT NULL,
    occurred_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (event_id, occurred_at)
) PARTITION BY RANGE (occurred_at);

CREATE TABLE IF NOT EXISTS property_events_2026_04
    PARTITION OF property_events
    FOR VALUES FROM ('2026-04-01 00:00:00+00') TO ('2026-05-01 00:00:00+00');

CREATE INDEX IF NOT EXISTS idx_property_photos_property_id ON property_photos(property_id);
CREATE INDEX IF NOT EXISTS idx_property_events_property_id ON property_events(property_id);
CREATE INDEX IF NOT EXISTS idx_property_events_user_id ON property_events(user_id);
CREATE INDEX IF NOT EXISTS idx_property_events_occurred_at ON property_events(occurred_at);

-- ## 7. services & clauses (metadata)
CREATE TABLE IF NOT EXISTS service_categories (
    category_id serial PRIMARY KEY,
    code varchar(40) NOT NULL UNIQUE,
    name varchar(80) NOT NULL
);

CREATE TABLE IF NOT EXISTS services (
    service_id serial PRIMARY KEY,
    code varchar(40) NOT NULL UNIQUE,
    icon varchar(80) NOT NULL,
    category_id int NOT NULL REFERENCES service_categories(category_id),
    is_active boolean NOT NULL DEFAULT true,
    is_deprecated boolean NOT NULL DEFAULT false,
    sort_order int NOT NULL
);

CREATE TABLE IF NOT EXISTS clause_value_types (
    value_type_id serial PRIMARY KEY,
    code varchar(40) NOT NULL,
    name varchar(80) NOT NULL
);

CREATE TABLE IF NOT EXISTS clauses (
    clause_id serial PRIMARY KEY,
    code varchar(40) NOT NULL UNIQUE,
    name varchar(100) NOT NULL,
    description text NOT NULL,
    value_type_id int NOT NULL REFERENCES clause_value_types(value_type_id),
    icon varchar(80) NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    is_deprecated boolean NOT NULL DEFAULT false,
    sort_order int NOT NULL
);

CREATE TABLE IF NOT EXISTS property_services (
    property_id int NOT NULL REFERENCES properties(property_id),
    service_id int NOT NULL REFERENCES services(service_id),
    assigned_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (property_id, service_id)
);

CREATE TABLE IF NOT EXISTS property_clauses (
    property_clause_id serial PRIMARY KEY,
    property_id int NOT NULL REFERENCES properties(property_id),
    clause_id int NOT NULL REFERENCES clauses(clause_id),
    boolean_value boolean,
    integer_value int,
    min_value decimal(12,2),
    max_value decimal(12,2),
    assigned_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_services_category_id ON services(category_id);
CREATE INDEX IF NOT EXISTS idx_clauses_value_type_id ON clauses(value_type_id);
CREATE INDEX IF NOT EXISTS idx_property_services_service_id ON property_services(service_id);
CREATE INDEX IF NOT EXISTS idx_property_clauses_property_id ON property_clauses(property_id);
CREATE INDEX IF NOT EXISTS idx_property_clauses_clause_id ON property_clauses(clause_id);

-- ## 8. crm & logistics
CREATE TABLE IF NOT EXISTS follow_up_status (
    status_id serial PRIMARY KEY,
    name varchar(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS visit_status (
    status_id serial PRIMARY KEY,
    name varchar(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS inquiries (
    inquiry_id serial PRIMARY KEY,
    inquiry_uuid uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    property_id int NOT NULL REFERENCES properties(property_id),
    user_id int references users(user_id),
    message text NOT NULL,
    follow_up_status_id int NOT NULL REFERENCES follow_up_status(status_id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS property_agents (
    property_id int NOT NULL REFERENCES properties(property_id),
    agent_id int NOT NULL REFERENCES users(user_id),
    is_primary boolean NOT NULL DEFAULT true,
    assigned_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (property_id, agent_id)
);

CREATE TABLE IF NOT EXISTS agent_schedules (
    schedule_id serial PRIMARY KEY,
    agent_id int NOT NULL REFERENCES users(user_id),
    day_of_week smallint NOT NULL,
    start_time time NOT NULL,
    end_time time NOT NULL,
    is_active boolean NOT NULL DEFAULT true,
    UNIQUE (agent_id, day_of_week, start_time, end_time)
);

CREATE TABLE IF NOT EXISTS property_exceptions (
    exception_id serial PRIMARY KEY,
    property_id int NOT NULL REFERENCES properties(property_id),
    exception_date date NOT NULL,
    reason varchar(100) NOT NULL
);

CREATE TABLE IF NOT EXISTS visits (
    visit_id serial PRIMARY KEY,
    visit_uuid uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    property_id int NOT NULL REFERENCES properties(property_id),
    client_id int NOT NULL REFERENCES users(user_id),
    agent_id int references users(user_id),
    visit_date timestamptz NOT NULL,
    status_id int NOT NULL REFERENCES visit_status(status_id),
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_inquiries_property_id ON inquiries(property_id);
CREATE INDEX IF NOT EXISTS idx_inquiries_user_id ON inquiries(user_id);
CREATE INDEX IF NOT EXISTS idx_inquiries_follow_up_status_id ON inquiries(follow_up_status_id);
CREATE INDEX IF NOT EXISTS idx_property_agents_agent_id ON property_agents(agent_id);
CREATE INDEX IF NOT EXISTS idx_property_exceptions_property_id ON property_exceptions(property_id);
CREATE INDEX IF NOT EXISTS idx_visits_property_id ON visits(property_id);
CREATE INDEX IF NOT EXISTS idx_visits_client_id ON visits(client_id);
CREATE INDEX IF NOT EXISTS idx_visits_agent_id ON visits(agent_id);
CREATE INDEX IF NOT EXISTS idx_visits_status_id ON visits(status_id);

-- ## 9. operations, contracts & payments
CREATE TYPE transaction_type AS ENUM ('sale', 'rent');

CREATE TABLE IF NOT EXISTS transaction_status (
    status_id serial PRIMARY KEY,
    name varchar(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS contract_status (
    status_id serial PRIMARY KEY,
    name varchar(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS payment_gateways (
    gateway_id serial PRIMARY KEY,
    name varchar(50) NOT NULL,
    is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS payment_methods (
    method_id serial PRIMARY KEY,
    name varchar(50) NOT NULL,
    is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS payment_status (
    status_id serial PRIMARY KEY,
    name varchar(30) NOT NULL,
    is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE IF NOT EXISTS transactions (
    transaction_id serial PRIMARY KEY,
    transaction_uuid uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    property_id int NOT NULL REFERENCES properties(property_id),
    client_id int NOT NULL REFERENCES users(user_id),
    agent_id int NOT NULL REFERENCES users(user_id),
    transaction_type transaction_type NOT NULL,
    status_id int NOT NULL REFERENCES transaction_status(status_id),
    final_amount decimal(15,2) NOT NULL,
    closing_date date NOT NULL
);

CREATE TABLE IF NOT EXISTS contracts (
    contract_id serial PRIMARY KEY,
    contract_uuid uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    transaction_id int NOT NULL REFERENCES transactions(transaction_id),
    currency char(3) NOT NULL,
    agreed_amount decimal(15,2) NOT NULL,
    storage_key varchar(255) NOT NULL,
    start_date date NOT NULL,
    end_date date,
    status_id int NOT NULL REFERENCES contract_status(status_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE IF NOT EXISTS payments (
    payment_id serial PRIMARY KEY,
    payment_uuid uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    contract_id int NOT NULL REFERENCES contracts(contract_id),
    client_id int NOT NULL REFERENCES users(user_id),
    billing_period date NOT NULL,
    due_date date NOT NULL,
    amount decimal(15,2) NOT NULL,
    payment_method_id int NOT NULL REFERENCES payment_methods(method_id),
    gateway_id int NOT NULL REFERENCES payment_gateways(gateway_id),
    gateway_payment_id varchar(100),
    gateway_order_id varchar(100),
    status_id int NOT NULL REFERENCES payment_status(status_id),
    gateway_status varchar(50),
    payment_date timestamptz,
    metadata jsonb,
    CHECK (EXTRACT(DAY FROM billing_period) = 1)
);

CREATE INDEX IF NOT EXISTS idx_transactions_property_id ON transactions(property_id);
CREATE INDEX IF NOT EXISTS idx_transactions_client_id ON transactions(client_id);
CREATE INDEX IF NOT EXISTS idx_transactions_agent_id ON transactions(agent_id);
CREATE INDEX IF NOT EXISTS idx_transactions_status_id ON transactions(status_id);
CREATE INDEX IF NOT EXISTS idx_contracts_transaction_id ON contracts(transaction_id);
CREATE INDEX IF NOT EXISTS idx_contracts_status_id ON contracts(status_id);
CREATE INDEX IF NOT EXISTS idx_payments_contract_id ON payments(contract_id);
CREATE INDEX IF NOT EXISTS idx_payments_client_id ON payments(client_id);
CREATE INDEX IF NOT EXISTS idx_payments_payment_method_id ON payments(payment_method_id);
CREATE INDEX IF NOT EXISTS idx_payments_gateway_id ON payments(gateway_id);
CREATE INDEX IF NOT EXISTS idx_payments_status_id ON payments(status_id);
CREATE INDEX IF NOT EXISTS idx_payments_billing_period ON payments(billing_period);

-- ## 10. audit & history
CREATE TABLE IF NOT EXISTS property_status_history (
    history_id serial PRIMARY KEY,
    property_id int NOT NULL REFERENCES properties(property_id),
    previous_status_id int NOT NULL REFERENCES property_status(status_id),
    new_status_id int NOT NULL REFERENCES property_status(status_id),
    changed_by_user_id int NOT NULL REFERENCES users(user_id),
    changed_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS contract_status_history (
    history_id serial PRIMARY KEY,
    contract_id int NOT NULL REFERENCES contracts(contract_id),
    previous_status_id int NOT NULL REFERENCES contract_status(status_id),
    new_status_id int NOT NULL REFERENCES contract_status(status_id),
    changed_by_user_id int NOT NULL REFERENCES users(user_id),
    changed_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS visit_status_history (
    history_id serial PRIMARY KEY,
    visit_id int NOT NULL REFERENCES visits(visit_id),
    previous_status_id int NOT NULL REFERENCES visit_status(status_id),
    new_status_id int NOT NULL REFERENCES visit_status(status_id),
    changed_by_user_id int NOT NULL REFERENCES users(user_id),
    changed_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS transaction_status_history (
    history_id serial PRIMARY KEY,
    transaction_id int NOT NULL REFERENCES transactions(transaction_id),
    previous_status_id int NOT NULL REFERENCES transaction_status(status_id),
    new_status_id int NOT NULL REFERENCES transaction_status(status_id),
    changed_by_user_id int NOT NULL REFERENCES users(user_id),
    changed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_property_status_history_property_id ON property_status_history(property_id);
CREATE INDEX IF NOT EXISTS idx_property_status_history_previous_status_id ON property_status_history(previous_status_id);
CREATE INDEX IF NOT EXISTS idx_property_status_history_new_status_id ON property_status_history(new_status_id);
CREATE INDEX IF NOT EXISTS idx_property_status_history_changed_by_user_id ON property_status_history(changed_by_user_id);
CREATE INDEX IF NOT EXISTS idx_contract_status_history_contract_id ON contract_status_history(contract_id);
CREATE INDEX IF NOT EXISTS idx_contract_status_history_previous_status_id ON contract_status_history(previous_status_id);
CREATE INDEX IF NOT EXISTS idx_contract_status_history_new_status_id ON contract_status_history(new_status_id);
CREATE INDEX IF NOT EXISTS idx_contract_status_history_changed_by_user_id ON contract_status_history(changed_by_user_id);
CREATE INDEX IF NOT EXISTS idx_visit_status_history_visit_id ON visit_status_history(visit_id);
CREATE INDEX IF NOT EXISTS idx_visit_status_history_previous_status_id ON visit_status_history(previous_status_id);
CREATE INDEX IF NOT EXISTS idx_visit_status_history_new_status_id ON visit_status_history(new_status_id);
CREATE INDEX IF NOT EXISTS idx_visit_status_history_changed_by_user_id ON visit_status_history(changed_by_user_id);
CREATE INDEX IF NOT EXISTS idx_transaction_status_history_transaction_id ON transaction_status_history(transaction_id);
CREATE INDEX IF NOT EXISTS idx_transaction_status_history_previous_status_id ON transaction_status_history(previous_status_id);
CREATE INDEX IF NOT EXISTS idx_transaction_status_history_new_status_id ON transaction_status_history(new_status_id);
CREATE INDEX IF NOT EXISTS idx_transaction_status_history_changed_by_user_id ON transaction_status_history(changed_by_user_id);
