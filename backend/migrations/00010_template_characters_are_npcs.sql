-- +goose Up
-- A character created from an NPC template is an NPC. Some databases have rows
-- where type drifted to 'pc' (e.g. the seed Goblin, flipped via the Characters
-- page), which keeps a dead monster in the turn order and leaks its stats in
-- the player view. Put them back; a no-op where the data is already right.
UPDATE characters SET type = 'npc'
WHERE npc_template_id IS NOT NULL AND type <> 'npc';

-- +goose Down
-- Data fix only; the previous (wrong) types are not recoverable or worth restoring.
SELECT 1;
