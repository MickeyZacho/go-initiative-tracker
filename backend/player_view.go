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

// redactLedgerForPlayers hides exact damage/healing amounts on NPCs, which
// would otherwise reveal their HP. The entry ("Goblin 2 took damage") stays.
func redactLedgerForPlayers(entries []dao.EncounterLedgerEntry) {
	for i := range entries {
		if entries[i].ActorType == "npc" {
			entries[i].HPChange = 0
		}
	}
}
