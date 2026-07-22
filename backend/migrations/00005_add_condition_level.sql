-- +goose Up
-- Exhaustion is not a binary condition in 5e: it has six cumulative levels, each
-- strictly worse than the last (level 6 is death). Store that as a nullable
-- `level` alongside the condition name rather than as six distinct condition
-- names, so the existing UNIQUE (encounter_id, character_id, condition) key still
-- holds one exhaustion row per creature and raising/lowering a level is an upsert
-- instead of a delete + insert. NULL means "this condition has no levels"
-- (every condition except Exhaustion); Go validates the range.
ALTER TABLE encounter_character_conditions
    ADD COLUMN IF NOT EXISTS level INTEGER;

-- Existing exhaustion rows predate levels; treat them as level 1 rather than
-- leaving them NULL, which the API would reject as malformed on read-modify-write.
UPDATE encounter_character_conditions
SET level = 1
WHERE condition = 'Exhaustion' AND level IS NULL;

-- +goose Down
ALTER TABLE encounter_character_conditions
    DROP COLUMN IF EXISTS level;
