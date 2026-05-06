-- 000008_catalogs_data.up.sql

INSERT INTO property_status (status_id, name) VALUES
(1, 'Reserved'),
(2, 'Available'),
(3, 'Sold'),
(4, 'Rented')
ON CONFLICT (status_id) DO NOTHING;

INSERT INTO transaction_status (status_id, name) VALUES
(1, 'Pending'),
(2, 'In Progress'),
(3, 'Closed'),
(4, 'Cancelled')
ON CONFLICT (status_id) DO NOTHING;

INSERT INTO contract_status (status_id, name) VALUES
(1, 'Draft'),
(2, 'Active'),
(3, 'Expired'),
(4, 'Terminated')
ON CONFLICT (status_id) DO NOTHING;

INSERT INTO visit_status (status_id, name) VALUES
(1, 'Scheduled'),
(2, 'Confirmed'),
(3, 'Completed'),
(4, 'Cancelled')
ON CONFLICT (status_id) DO NOTHING;
