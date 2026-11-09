-- +goose Up
-- Temporary hit points (5e): a buffer that absorbs damage before current HP and
-- doesn't stack or heal. Per-encounter state, alongside current_hp.
ALTER TABLE encounter_characters
    ADD COLUMN IF NOT EXISTS temp_hp INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE encounter_characters
    DROP COLUMN IF EXISTS temp_hp;
