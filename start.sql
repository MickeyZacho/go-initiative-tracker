

DROP TABLE IF EXISTS encounter_characters;

DROP TABLE IF EXISTS encounter_users;

DROP TABLE IF EXISTS encounter_ledger;
DROP TABLE IF EXISTS encounters;
DROP TABLE IF EXISTS characters;
DROP TABLE IF EXISTS monster_templates;
-- Ability Scores Composite Type
DROP TYPE IF EXISTS stat_block;
CREATE TYPE stat_block AS (
	strength     INTEGER,
	dexterity    INTEGER,
	constitution INTEGER,
	intelligence INTEGER,
	wisdom       INTEGER,
	charisma     INTEGER
);

CREATE TABLE monster_templates (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	description TEXT,
	base_stats stat_block, -- Default stats for this monster/NPC type
	armor_class INTEGER DEFAULT 10,
	max_hp INTEGER DEFAULT 10
);
CREATE TABLE characters (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    owner_id TEXT, -- Discord user ID for PCs, NULL for NPCs/monsters
    type TEXT NOT NULL DEFAULT 'pc', -- 'pc' or 'npc'
    class TEXT, -- For PCs
	stats stat_block, -- Ability scores
	armor_class INTEGER DEFAULT 10, -- AC for the character
	to_hit_modifier INTEGER DEFAULT 0, -- Attack roll modifier
	max_hp INTEGER DEFAULT 10, -- Maximum hit points
	monster_template_id INTEGER REFERENCES monster_templates(id) ON DELETE SET NULL -- For NPCs/monsters
);
  
-- Encounters
CREATE TABLE encounters (
	id SERIAL PRIMARY KEY,
	name TEXT NOT NULL,
	owner_id TEXT, -- Discord user ID of DM
	description TEXT
);
CREATE TABLE encounter_ledger (
	id SERIAL PRIMARY KEY,
	encounter_id INTEGER REFERENCES encounters(id),
	actor_id INTEGER REFERENCES characters(id),   -- who performed the action
	target_id INTEGER REFERENCES characters(id),  -- who received the action
	action_type TEXT,
	hp_change INTEGER,
	description TEXT,
	created_at TIMESTAMP DEFAULT now()
);



CREATE TABLE encounter_characters (
	encounter_id INTEGER REFERENCES encounters(id) ON DELETE CASCADE,
	character_id INTEGER REFERENCES characters(id) ON DELETE CASCADE,
	initiative INTEGER,
	current_hp INTEGER,
	is_active BOOLEAN DEFAULT FALSE,
	PRIMARY KEY (encounter_id, character_id)
);

-- Table to link users (players) to encounters
CREATE TABLE encounter_users (
	encounter_id INTEGER REFERENCES encounters(id) ON DELETE CASCADE,
	user_id TEXT NOT NULL, -- Discord user ID
	PRIMARY KEY (encounter_id, user_id)
);

-- Example Inserts


-- Monster Template: Goblin
INSERT INTO monster_templates (name, description, base_stats, armor_class, max_hp)
VALUES ('Goblin', 'Small, sneaky humanoid', ROW(8, 14, 10, 10, 8, 8)::stat_block, 14, 7);

-- Monster Template: Orc
INSERT INTO monster_templates (name, description, base_stats, armor_class, max_hp)
VALUES ('Orc', 'Brutish warrior', ROW(16, 12, 16, 7, 11, 10)::stat_block, 13, 15);

-- Encounter: Goblin Ambush
INSERT INTO encounters (name, owner_id, description)
VALUES ('Goblin Ambush', 'dm1', 'A group of goblins attack the party.');

-- Player Character: Aragorn
INSERT INTO characters (name, owner_id, type, class, stats, max_hp)
VALUES ('Aragorn', 'user1', 'pc', 'Ranger', ROW(16, 14, 14, 12, 13, 12)::stat_block, 38);

-- NPC: Goblin (from template)
INSERT INTO characters (name, type, stats, max_hp, monster_template_id)
VALUES ('Goblin', 'npc', ROW(8, 14, 10, 10, 8, 8)::stat_block, 7, 1);

INSERT INTO encounter_characters (encounter_id, character_id, initiative, current_hp, is_active)
VALUES (1, 1, 15, 38, TRUE), -- Aragorn
		 (1, 2, 14, 7, FALSE); -- Goblin




-- Example ledger entries for Goblin Ambush (encounter_id = 1)
-- Aragorn attacks Goblin
INSERT INTO encounter_ledger (encounter_id, actor_id, target_id, action_type, hp_change, description)
VALUES (1, 1, 2, 'attack', -5, 'Aragorn slashes Goblin for 5 damage');

-- Goblin attacks Aragorn
INSERT INTO encounter_ledger (encounter_id, actor_id, target_id, action_type, hp_change, description)
VALUES (1, 2, 1, 'attack', -3, 'Goblin stabs Aragorn for 3 damage');

-- Aragorn heals self
INSERT INTO encounter_ledger (encounter_id, actor_id, target_id, action_type, hp_change, description)
VALUES (1, 1, 1, 'heal', 2, 'Aragorn uses second wind and heals 2 HP');