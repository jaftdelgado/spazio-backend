ALTER TABLE clauses DROP COLUMN description;
ALTER TABLE clauses DROP COLUMN icon;

CREATE TABLE clause_modalities (
    clause_id   INT NOT NULL REFERENCES clauses (clause_id),
    modality_id INT NOT NULL REFERENCES modalities (modality_id),
    PRIMARY KEY (clause_id, modality_id)
);