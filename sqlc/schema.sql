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

CREATE FUNCTION ST_MakePoint(double precision, double precision) RETURNS bytea
LANGUAGE SQL
AS $$
	SELECT ''::bytea;
$$;

CREATE FUNCTION ST_SetSRID(bytea, integer) RETURNS bytea
LANGUAGE SQL
AS $$
	SELECT $1;
$$;

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


CREATE TABLE orientations (
	orientation_id serial PRIMARY KEY,
	name varchar(30) NOT NULL
);

CREATE TABLE rent_periods (
	period_id serial PRIMARY KEY,
	name varchar(50) NOT NULL
);

<<<<<<< HEAD

=======
>>>>>>> fa39902 (feat(catalogs): implement property type-specific rent periods retrieval and update API documentation)
CREATE TABLE property_type_periods (
	property_type_id int NOT NULL REFERENCES property_types(property_type_id),
	period_id int NOT NULL REFERENCES rent_periods(period_id),
	PRIMARY KEY (property_type_id, period_id)
);

CREATE TABLE property_status (
	status_id serial PRIMARY KEY,
	name varchar(50) NOT NULL
);

CREATE TABLE properties (
	property_id serial PRIMARY KEY,
	property_uuid uuid NOT NULL UNIQUE,
	owner_id int NOT NULL REFERENCES users(user_id),
	current_resident_id int REFERENCES users(user_id),
	category text NOT NULL,
	title varchar(128) NOT NULL,
	description text NOT NULL,
	property_type_id int NOT NULL REFERENCES property_types(property_type_id),
	modality_id int NOT NULL REFERENCES modalities(modality_id),
	status_id int NOT NULL REFERENCES property_status(status_id),
	cover_photo_url varchar(255) NOT NULL,
	lot_area numeric(12,2) NOT NULL,
	is_featured boolean NOT NULL DEFAULT false,
	published_at timestamptz,
	updated_at timestamptz,
	created_at timestamptz NOT NULL DEFAULT now(),
	deleted_at timestamptz
);

CREATE TABLE residential_properties (
	property_id int PRIMARY KEY REFERENCES properties(property_id),
	bedrooms smallint NOT NULL,
	bathrooms smallint NOT NULL,
	beds smallint NOT NULL,
	floors smallint NOT NULL,
	parking_spots smallint NOT NULL,
	built_area numeric(12,2) NOT NULL,
	construction_year smallint NOT NULL,
	orientation_id int NOT NULL REFERENCES orientations(orientation_id),
	is_furnished boolean NOT NULL
);

CREATE TABLE commercial_properties (
	property_id int PRIMARY KEY REFERENCES properties(property_id),
	ceiling_height numeric(5,2) NOT NULL,
	loading_docks smallint NOT NULL,
	internal_offices smallint NOT NULL,
	three_phase_power boolean NOT NULL,
	land_use varchar(100) NOT NULL
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

CREATE TABLE postal_codes (
	postal_code_id serial PRIMARY KEY
);

CREATE TABLE zones (
	zone_id serial PRIMARY KEY
);

CREATE TABLE locations (
	location_id serial PRIMARY KEY,
	property_id integer NOT NULL UNIQUE REFERENCES properties(property_id),
	city_id integer NOT NULL REFERENCES cities(city_id),
	zone_id integer REFERENCES zones(zone_id),
	postal_code_id integer REFERENCES postal_codes(postal_code_id),
	neighborhood varchar(60) NOT NULL,
	street varchar(120) NOT NULL,
	exterior_number varchar(20) NOT NULL,
	interior_number varchar(20),
	postal_code varchar(10) NOT NULL,
	coordinates bytea NOT NULL,
	is_public_address boolean NOT NULL
);

CREATE TABLE sale_prices (
	price_id serial PRIMARY KEY,
	property_id int NOT NULL REFERENCES properties(property_id),
	sale_price numeric(15,2) NOT NULL,
	currency char(3) NOT NULL,
	is_negotiable boolean NOT NULL,
	is_current boolean NOT NULL,
	valid_from timestamptz NOT NULL,
	valid_until timestamptz,
	change_reason varchar(100),
	changed_by_user_id int NOT NULL REFERENCES users(user_id)
);

CREATE TABLE rent_prices (
	rent_price_id serial PRIMARY KEY,
	property_id int NOT NULL REFERENCES properties(property_id),
	period_id int NOT NULL REFERENCES rent_periods(period_id),
	rent_price numeric(15,2) NOT NULL,
	deposit numeric(15,2),
	currency char(3) NOT NULL,
	is_negotiable boolean NOT NULL,
	is_current boolean NOT NULL,
	valid_from timestamptz NOT NULL,
	valid_until timestamptz,
	change_reason varchar(100),
	changed_by_user_id int NOT NULL REFERENCES users(user_id)
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
	description text,
	value_type_id int NOT NULL REFERENCES clause_value_types(value_type_id),
	icon varchar(80),
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

CREATE TABLE property_clauses (
	property_clause_id serial PRIMARY KEY,
	property_id int NOT NULL REFERENCES properties(property_id),
	clause_id int NOT NULL REFERENCES clauses(clause_id),
	boolean_value boolean,
	integer_value int,
	min_value numeric(12,2),
	max_value numeric(12,2)
);

CREATE TABLE property_photos (
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

CREATE TABLE property_agents (
	property_id integer NOT NULL REFERENCES properties(property_id),
	agent_id integer NOT NULL REFERENCES users(user_id),
	is_primary boolean DEFAULT true NOT NULL,
	assigned_at timestamp with time zone DEFAULT now() NOT NULL,
	PRIMARY KEY (property_id, agent_id)
);

CREATE TABLE agent_schedules (
	schedule_id serial PRIMARY KEY,
	agent_id integer NOT NULL REFERENCES users(user_id),
	day_of_week smallint NOT NULL,
	start_time time without time zone NOT NULL,
	end_time time without time zone NOT NULL,
	is_active boolean DEFAULT true NOT NULL
);

CREATE TABLE property_exceptions (
	exception_id serial PRIMARY KEY,
	property_id integer NOT NULL REFERENCES properties(property_id),
	exception_date date NOT NULL,
	reason character varying(100) NOT NULL,
	start_time time without time zone,
	end_time time without time zone
);

CREATE TABLE visit_status (
	status_id serial PRIMARY KEY,
	name character varying(50) NOT NULL
);

CREATE TABLE visits (
	visit_id serial PRIMARY KEY,
	visit_uuid uuid DEFAULT gen_random_uuid() NOT NULL UNIQUE,
	property_id integer NOT NULL REFERENCES properties(property_id),
	client_id integer NOT NULL REFERENCES users(user_id),
	agent_id integer REFERENCES users(user_id),
	visit_date timestamp with time zone NOT NULL,
	status_id integer NOT NULL REFERENCES visit_status(status_id),
	created_at timestamp with time zone DEFAULT now() NOT NULL,
	deleted_at timestamp with time zone
);

CREATE TABLE visit_status_history (
	history_id serial PRIMARY KEY,
	visit_id integer NOT NULL REFERENCES visits(visit_id),
	previous_status_id integer NOT NULL REFERENCES visit_status(status_id),
	new_status_id integer NOT NULL REFERENCES visit_status(status_id),
	changed_by_user_id integer NOT NULL REFERENCES users(user_id),
	changed_at timestamp with time zone DEFAULT now() NOT NULL
);