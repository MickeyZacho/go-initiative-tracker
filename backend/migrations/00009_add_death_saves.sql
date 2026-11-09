-- +goose Up
-- Death saving throws (5e) for a PC at 0 HP: three successes = stable, three
-- failures = dead. Per-encounter combat state; both reset to 0 whenever the
-- character is back above 0 HP (see the encounter_characters upsert).
ALTER TABLE encounter_characters
    ADD COLUMN IF NOT EXISTS death_save_successes INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS death_save_failures INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE encounter_characters
    DROP COLUMN IF EXISTS death_save_successes,
    DROP COLUMN IF EXISTS death_save_failures;
