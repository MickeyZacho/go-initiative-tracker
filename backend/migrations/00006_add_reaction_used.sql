-- +goose Up
-- A creature has a single reaction that refreshes at the start of its turn. Track
-- whether it has been spent this round as per-encounter state alongside the other
-- combat fields (initiative, current_hp, is_active), since the same library
-- character can be mid-reaction in one encounter and untouched in another. Defaults
-- to FALSE (reaction available); the turn-advance logic resets it to FALSE when a
-- creature's turn begins.
ALTER TABLE encounter_characters
    ADD COLUMN IF NOT EXISTS reaction_used BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose Down
ALTER TABLE encounter_characters
    DROP COLUMN IF EXISTS reaction_used;
