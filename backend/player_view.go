package main

import "go-initiative-tracker/dao"

// The player view: shared-encounter members (players) see turn order and how
// hurt each enemy looks, but not the numbers behind it. Redaction happens here,
// on the server, so the hidden stats never reach a player's browser.

// healthStatus describes HP the way a DM would at the table.
func healthStatus(current, max int) string {
	switch {
	case current <= 0:
		return "Down"
	case max > 0 && current*2 <= max:
		return "Bloodied"
	case current < max:
		return "Wounded"
	default:
		return "Healthy"
	}
}

// redactNpcsForPlayers hides NPC combat numbers (AC, HP, to-hit) in place,
// replacing HP with a coarse HealthStatus. PCs are left untouched: the party
// can see each other's sheets.
func redactNpcsForPlayers(chars []dao.Character) {
	for i := range chars {
		c := &chars[i]
		if c.Type != "npc" {
			continue
		}
		c.HealthStatus = healthStatus(c.CurrentHP, c.MaxHP)
		c.Redacted = true
		c.ArmorClass = 0
		c.ToHitModifier = 0
		c.CurrentHP = 0
		c.MaxHP = 0
		c.TempHP = 0
	}
}

// dropHiddenForPlayers removes combatants the DM has staged as hidden. Players
// must not learn they exist at all, so they are filtered out rather than
// redacted.
func dropHiddenForPlayers(chars []dao.Character) []dao.Character {
	visible := chars[:0]
	for _, c := range chars {
		if !c.Hidden {
			visible = append(visible, c)
		}
	}
	return visible
}

// redactLedgerForPlayers hides exact damage/healing amounts on NPCs, which
// would otherwise reveal their HP. The entry ("Goblin 2 took damage") stays.
// Entries whose actor is currently hidden are dropped entirely.
func redactLedgerForPlayers(entries []dao.EncounterLedgerEntry) []dao.EncounterLedgerEntry {
	kept := entries[:0]
	for _, e := range entries {
		if e.ActorHidden {
			continue
		}
		if e.ActorType == "npc" {
			e.HPChange = 0
		}
		kept = append(kept, e)
	}
	return kept
}
