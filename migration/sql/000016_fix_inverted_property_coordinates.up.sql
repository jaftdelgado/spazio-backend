-- Corrects clearly inverted property coordinates.
-- Only swaps rows whose current latitude is impossible and whose current
-- longitude looks like a latitude. This avoids touching ambiguous rows.
WITH candidates AS (
    SELECT
        location_id,
        ST_X(coordinates) AS current_longitude,
        ST_Y(coordinates) AS current_latitude
    FROM locations
    WHERE ST_Y(coordinates) NOT BETWEEN -90 AND 90
      AND ST_X(coordinates) BETWEEN -90 AND 90
)
UPDATE locations AS l
SET coordinates = ST_SetSRID(
    ST_MakePoint(c.current_latitude, c.current_longitude),
    4326
)
FROM candidates AS c
WHERE l.location_id = c.location_id;
