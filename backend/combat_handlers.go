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
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}

	encounterID, err := encounterIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}

	activeCharacterID, err := encounterCharacterDAO.StartCombat(encounterID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no characters") {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Encounter has no characters"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to start combat"})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"status": "success", "active_character_id": activeCharacterID})
}

func apiResetCombatHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}

	encounterID, err := encounterIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}

	if err := encounterCharacterDAO.ResetCombat(encounterID); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to reset combat"})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func apiNextTurnHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}

	encounterID, err := encounterIDFromRequest(r)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}

	activeCharacterID, err := encounterCharacterDAO.AdvanceTurn(encounterID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "no characters") {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Encounter has no characters"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to advance turn"})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"status": "success", "active_character_id": activeCharacterID})
}
