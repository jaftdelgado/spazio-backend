DROP TABLE IF EXISTS clause_modalities;

ALTER TABLE clauses ADD COLUMN description text NOT NULL DEFAULT '';
ALTER TABLE clauses ADD COLUMN icon varchar(80) NOT NULL DEFAULT '';