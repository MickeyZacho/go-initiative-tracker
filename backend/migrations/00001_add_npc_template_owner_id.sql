-- +goose Up
-- Databases created from an older start.sql are missing npc_templates.owner_id
-- (added to the baseline later). IF NOT EXISTS makes this a no-op on fresh DBs
-- that already have the column while repairing drifted ones.
ALTER TABLE npc_templates ADD COLUMN IF NOT EXISTS owner_id TEXT;

-- +goose Down
ALTER TABLE npc_templates DROP COLUMN IF EXISTS owner_id;
