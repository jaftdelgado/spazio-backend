-- 000006_create_property_type_periods.up.sql

CREATE TABLE property_type_periods (
    property_type_id  INT  NOT NULL REFERENCES property_types(property_type_id),
    period_id         INT  NOT NULL REFERENCES rent_periods(period_id),
    PRIMARY KEY (property_type_id, period_id)
);