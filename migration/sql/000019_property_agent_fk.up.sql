ALTER TABLE properties
ADD COLUMN agent_id int REFERENCES users(user_id);

CREATE INDEX IF NOT EXISTS idx_properties_agent_id ON properties(agent_id);

WITH ranked_assignments AS (
    SELECT
        property_id,
        agent_id,
        ROW_NUMBER() OVER (
            PARTITION BY property_id
            ORDER BY is_primary DESC, assigned_at ASC, agent_id ASC
        ) AS row_num
    FROM property_agents
)
UPDATE properties AS p
SET agent_id = ranked.agent_id
FROM ranked_assignments AS ranked
WHERE p.property_id = ranked.property_id
  AND ranked.row_num = 1;

DROP TABLE property_agents;
