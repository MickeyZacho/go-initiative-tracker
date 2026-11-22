-- +goose Up
-- Imported monster templates (e.g. the 5e SRD via cmd/import-srd).
-- details keeps the full source record (actions, speed, CR, senses, ...) as
-- JSONB so nothing is lost before the app has UI for it.
-- source/source_key identify where a row came from ('srd-5.2', 'goblin') so a
-- re-import updates rows instead of duplicating them. Hand-made templates leave
-- both NULL, and the partial unique index ignores them, so a user can still
-- have their own "Goblin" next to the SRD one.
ALTER TABLE npc_templates ADD COLUMN IF NOT EXISTS details JSONB;
ALTER TABLE npc_templates ADD COLUMN IF NOT EXISTS source TEXT;
ALTER TABLE npc_templates ADD COLUMN IF NOT EXISTS source_key TEXT;
CREATE UNIQUE INDEX IF NOT EXISTS npc_templates_source_key_idx
    ON npc_templates (source, source_key)
    WHERE source IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS npc_templates_source_key_idx;
ALTER TABLE npc_templates DROP COLUMN IF EXISTS source_key;
ALTER TABLE npc_templates DROP COLUMN IF EXISTS source;
ALTER TABLE npc_templates DROP COLUMN IF EXISTS details;
