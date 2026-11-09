-- +goose Up
-- Track the combat round on the encounter so the DM (and players) can see it and
-- so "N rounds" condition durations have a visible clock. 0 means combat has not
-- started; StartCombat sets 1, AdvanceTurn increments it each time initiative
-- wraps back to the top, and ResetCombat returns it to 0.
ALTER TABLE encounters
    ADD COLUMN IF NOT EXISTS round INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE encounters
    DROP COLUMN IF EXISTS round;
