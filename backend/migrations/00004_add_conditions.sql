-- +goose Up
-- Per-encounter status conditions (D&D 5e: poisoned, prone, stunned, ...).
-- A condition attaches to a character *within an encounter* (the
-- encounter_characters join), not to the global characters row, so the same
-- monster template can be poisoned in one fight and unaffected in another.
-- duration_rounds NULL means "until removed" (manual); a value counts down at
-- the start of the affected creature's turn and is deleted when it hits 0.
-- The composite FK cascades, so removing a character from an encounter clears
-- its conditions; the UNIQUE key prevents stacking the same condition twice.
CREATE TABLE IF NOT EXISTS encounter_character_conditions (
    id              SERIAL PRIMARY KEY,
    encounter_id    INTEGER NOT NULL,
    character_id    INTEGER NOT NULL,
    condition       TEXT NOT NULL,   -- validated against the 5e set in Go
    duration_rounds INTEGER,         -- NULL = until removed
    note            TEXT,
    created_at      TIMESTAMP DEFAULT now(),
    FOREIGN KEY (encounter_id, character_id)
        REFERENCES encounter_characters(encounter_id, character_id) ON DELETE CASCADE,
    UNIQUE (encounter_id, character_id, condition)
);

-- +goose Down
DROP TABLE IF EXISTS encounter_character_conditions;
