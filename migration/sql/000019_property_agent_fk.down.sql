CREATE TABLE property_agents (
    property_id integer NOT NULL REFERENCES properties(property_id),
    agent_id integer NOT NULL REFERENCES users(user_id),
    is_primary boolean DEFAULT true NOT NULL,
    assigned_at timestamp with time zone DEFAULT now() NOT NULL,
    PRIMARY KEY (property_id, agent_id)
);

CREATE INDEX idx_property_agents_agent_id ON property_agents(agent_id);

INSERT INTO property_agents (
    property_id,
    agent_id,
    is_primary
)
SELECT
    property_id,
    agent_id,
    true
FROM properties
WHERE agent_id IS NOT NULL;

DROP INDEX IF EXISTS idx_properties_agent_id;

ALTER TABLE properties
DROP COLUMN agent_id;
