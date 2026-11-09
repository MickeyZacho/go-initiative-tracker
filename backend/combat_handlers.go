package main

import (
	"encoding/json"
	"fmt"
	"go-initiative-tracker/dao"
	"net/http"
	"strconv"
	"strings"
)

func encounterIDFromRequest(r *http.Request) (int, error) {
	var req struct {
		EncounterID int `json:"encounter_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return 0, err
	}
	if req.EncounterID <= 0 {
		return 0, fmt.Errorf("invalid encounter id")
	}
	return req.EncounterID, nil
}

func apiStartCombatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	encounterID, err := encounterIDFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	// Starting the fight is the DM's call; players can only end their turn.
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}

	activeCharacterID, err := encounterCharacterDAO.StartCombat(encounterID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no characters") {
			writeJSONError(w, http.StatusBadRequest, "Encounter has no characters")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to start combat")
		return
	}

	logCombatEvent(dao.EncounterLedgerInsert{EncounterID: encounterID, ActorID: activeCharacterID, ActionType: "turn"})
	events.publish(encounterID, "combat")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "active_character_id": activeCharacterID})
}

func apiSetActiveHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var req struct {
		EncounterID int `json:"encounter_id"`
		CharacterID int `json:"character_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.EncounterID <= 0 || req.CharacterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid encounter or character id")
		return
	}
	// Handing the turn to an arbitrary creature is a DM override.
	if !requireEncounterOwner(w, r, req.EncounterID) {
		return
	}

	if err := encounterCharacterDAO.SetActiveCharacter(req.EncounterID, req.CharacterID); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not in encounter") {
			writeJSONError(w, http.StatusNotFound, "Character not in encounter")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to set active character")
		return
	}

	events.publish(req.EncounterID, "combat")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "active_character_id": req.CharacterID})
}

// apiSetReactionHandler toggles whether a character has spent its reaction this
// round. Access mirrors the other combat endpoints; a change publishes a "combat"
// event so every viewer live-refreshes. The turn-advance logic clears the flag on
// its own when a creature's turn begins, so this only covers the manual toggle.
func apiSetReactionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var req struct {
		EncounterID  int  `json:"encounter_id"`
		CharacterID  int  `json:"character_id"`
		ReactionUsed bool `json:"reaction_used"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.EncounterID <= 0 || req.CharacterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid encounter or character id")
		return
	}
	if !requireEncounterAccess(w, r, req.EncounterID) {
		return
	}

	if err := encounterCharacterDAO.SetReactionUsed(req.EncounterID, req.CharacterID, req.ReactionUsed); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "not in encounter") {
			writeJSONError(w, http.StatusNotFound, "Character not in encounter")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to update reaction")
		return
	}

	// Only spending a reaction is worth a log line; restoring it (a correction)
	// is not an in-fiction event.
	if req.ReactionUsed {
		logCombatEvent(dao.EncounterLedgerInsert{EncounterID: req.EncounterID, ActorID: req.CharacterID, ActionType: "reaction"})
	}
	events.publish(req.EncounterID, "combat")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "reaction_used": req.ReactionUsed})
}

func apiResetCombatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	encounterID, err := encounterIDFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	// Ending the fight is the DM's call.
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}

	if err := encounterCharacterDAO.ResetCombat(encounterID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to reset combat")
		return
	}

	events.publish(encounterID, "combat")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func apiNextTurnHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	encounterID, err := encounterIDFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if !requireEncounterAccess(w, r, encounterID) {
		return
	}

	activeCharacterID, err := encounterCharacterDAO.AdvanceTurn(encounterID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no characters") {
			writeJSONError(w, http.StatusBadRequest, "Encounter has no characters")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to advance turn")
		return
	}

	logCombatEvent(dao.EncounterLedgerInsert{EncounterID: encounterID, ActorID: activeCharacterID, ActionType: "turn"})
	events.publish(encounterID, "combat")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "active_character_id": activeCharacterID})
}

// apiCombatStateHandler returns the encounter-level combat state that isn't
// carried on the character rows — currently the round (0 = not started).
// GET /encounters/combat/state?encounter_id=N
func apiCombatStateHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	encounterID, err := strconv.Atoi(r.URL.Query().Get("encounter_id"))
	if err != nil || encounterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid encounter id")
		return
	}
	if !requireEncounterAccess(w, r, encounterID) {
		return
	}
	round, err := encounterCharacterDAO.GetRound(encounterID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load combat state")
		return
	}
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "round": round})
}

// apiRollNpcInitiativeHandler rolls initiative for every NPC in the encounter
// (d20 + DEX modifier from its template), so the DM only types the players'
// rolls. POST /encounters/combat/roll-npc-initiative { encounter_id }
func apiRollNpcInitiativeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	encounterID, err := encounterIDFromRequest(r)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	// NPC initiative is the DM's roll.
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}
	rolled, err := encounterCharacterDAO.RollNpcInitiative(encounterID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to roll initiative")
		return
	}
	events.publish(encounterID, "combat")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "rolled": rolled})
}
