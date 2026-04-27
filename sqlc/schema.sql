CREATE TABLE users (
	user_id serial PRIMARY KEY
);

CREATE TABLE property_types (
	property_type_id serial PRIMARY KEY
);

CREATE TABLE modalities (
	modality_id serial PRIMARY KEY
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
