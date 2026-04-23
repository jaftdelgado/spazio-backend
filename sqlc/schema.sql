-- Minimal schema for sqlc code generation.

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
