-- Minimal schema for sqlc code generation.
CREATE TABLE IF NOT EXISTS roles (
    role_id serial PRIMARY KEY,
    name varchar(50) NOT NULL
);

CREATE TABLE IF NOT EXISTS user_status (
    status_id serial PRIMARY KEY,
    name varchar(30) NOT NULL
);

CREATE TABLE users (
	user_id serial PRIMARY KEY,
    user_uuid uuid NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    role_id int NOT NULL REFERENCES roles(role_id),
    first_name varchar(80) NOT NULL,
    last_name varchar(80) NOT NULL,
    email varchar(150) NOT NULL UNIQUE,
    --password_hash varchar(255) NOT NULL,
    phone varchar(20) NOT NULL,
    profile_picture_url varchar(255) NOT NULL,
    status_id int NOT NULL REFERENCES user_status(status_id),
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    deleted_at timestamptz
);

CREATE TABLE property_types (
    property_type_id serial PRIMARY KEY,
    name varchar(50) NOT NULL,
    icon varchar(80),
    is_deprecated boolean NOT NULL DEFAULT false
);

CREATE TABLE modalities (
    modality_id serial PRIMARY KEY,
    name varchar(50) NOT NULL
);

CREATE TABLE property_status (
	status_id serial PRIMARY KEY
);

CREATE TABLE properties (
	property_id serial PRIMARY KEY,
	owner_id int NOT NULL REFERENCES users(user_id),
	current_resident_id int REFERENCES users(user_id),
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

CREATE TABLE service_categories (
	category_id serial PRIMARY KEY,
	code varchar(40) NOT NULL UNIQUE,
	name varchar(80) NOT NULL
);

CREATE TABLE services (
	service_id serial PRIMARY KEY,
	code varchar(40) NOT NULL UNIQUE,
	icon varchar(80) NOT NULL,
	category_id int NOT NULL REFERENCES service_categories(category_id),
	is_active boolean NOT NULL DEFAULT true,
	is_deprecated boolean NOT NULL DEFAULT false,
	sort_order int NOT NULL
);

CREATE TABLE property_services (
	property_id int NOT NULL REFERENCES properties(property_id),
	service_id int NOT NULL REFERENCES services(service_id),
	assigned_at timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (property_id, service_id)
);

CREATE TABLE clause_value_types (
	value_type_id serial PRIMARY KEY,
	code varchar(40) NOT NULL UNIQUE,
	name varchar(80) NOT NULL
);

CREATE TABLE clauses (
	clause_id serial PRIMARY KEY,
	code varchar(40) NOT NULL UNIQUE,
	name varchar(100) NOT NULL,
	value_type_id int NOT NULL REFERENCES clause_value_types(value_type_id),
	is_active boolean NOT NULL DEFAULT true,
	is_deprecated boolean NOT NULL DEFAULT false,
	sort_order int NOT NULL,
	search_tags jsonb
);

CREATE TABLE clause_modalities (
	clause_id int NOT NULL REFERENCES clauses(clause_id),
	modality_id int NOT NULL REFERENCES modalities(modality_id),
	PRIMARY KEY (clause_id, modality_id)
);

CREATE TABLE countries (
	country_id serial PRIMARY KEY,
	iso2_code varchar(2) NOT NULL,
	name varchar(100) NOT NULL,
	is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE states (
	state_id serial PRIMARY KEY,
	country_id int NOT NULL REFERENCES countries(country_id),
	iso_code varchar(10),
	name varchar(100) NOT NULL,
	is_active boolean NOT NULL DEFAULT true
);

CREATE TABLE cities (
	city_id serial PRIMARY KEY,
	state_id int NOT NULL REFERENCES states(state_id),
	name varchar(100) NOT NULL
);