package main

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	if !requireEncounterAccess(w, r, encounterID) {
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
	if !requireEncounterAccess(w, r, req.EncounterID) {
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
	if !requireEncounterAccess(w, r, encounterID) {
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

	events.publish(encounterID, "combat")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "active_character_id": activeCharacterID})
}
