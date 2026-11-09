package main

import (
	"go-initiative-tracker/dao"
	"log"
)

// logCombatEvent appends an entry to the encounter ledger that powers the
// automatic combat log. Combat log entries are a side effect of the real
// action (a turn advancing, HP changing, a condition applied), so a failure to
// record one is logged but never surfaced — it must not fail the action that
// triggered it. Callers should invoke this before publishing the encounter
// event so the live refetch observes the new entry.
func logCombatEvent(entry dao.EncounterLedgerInsert) {
	if encounterLedgerDAO == nil {
		return
	}
	if _, err := encounterLedgerDAO.Create(entry); err != nil {
		log.Printf("combat log: failed to record %q event for encounter %d: %v",
			entry.ActionType, entry.EncounterID, err)
	}
}
