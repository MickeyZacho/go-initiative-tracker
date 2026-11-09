-- +goose Up
-- Hidden combatants: the DM can stage creatures in an encounter that players
-- can't see and that don't take turns until revealed (reinforcements, an ambush).
-- reveal_round, when set, auto-reveals the creature as that round begins;
-- NULL means "until the DM reveals it by hand".
ALTER TABLE encounter_characters ADD COLUMN IF NOT EXISTS hidden BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE encounter_characters ADD COLUMN IF NOT EXISTS reveal_round INTEGER;

-- +goose Down
ALTER TABLE encounter_characters DROP COLUMN IF EXISTS reveal_round;
ALTER TABLE encounter_characters DROP COLUMN IF EXISTS hidden;
