package main

import (
	"encoding/json"
	"go-initiative-tracker/dao"
	"net/http"
	"strings"
)

// apiAddConditionHandler applies a status condition to a character in an
// encounter. Access mirrors the combat endpoints (requireEncounterAccess), and
// a successful change publishes a "combat" event so every viewer live-refreshes.
func apiAddConditionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var req struct {
		EncounterID    int    `json:"encounter_id"`
		CharacterID    int    `json:"character_id"`
		Condition      string `json:"condition"`
		DurationRounds *int   `json:"duration_rounds"`
		Note           string `json:"note"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.EncounterID <= 0 || req.CharacterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid encounter or character id")
		return
	}
	req.Condition = strings.TrimSpace(req.Condition)
	if !dao.IsValidCondition(req.Condition) {
		writeJSONError(w, http.StatusBadRequest, "Unknown condition")
		return
	}
	// A duration, when provided, must be positive; "until removed" is expressed
	// by omitting it (nil), not by a zero/negative value.
	if req.DurationRounds != nil && *req.DurationRounds <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Duration must be greater than 0")
		return
	}
	if !requireEncounterAccess(w, r, req.EncounterID) {
		return
	}

	if err := encounterConditionDAO.Add(dao.Condition{
		EncounterID:    req.EncounterID,
		CharacterID:    req.CharacterID,
		Condition:      req.Condition,
		DurationRounds: req.DurationRounds,
		Note:           strings.TrimSpace(req.Note),
	}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to add condition")
		return
	}

	events.publish(req.EncounterID, "combat")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// apiRemoveConditionHandler clears a single condition by id, scoped to its
// encounter so a stale id cannot delete a condition from another fight.
func apiRemoveConditionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var req struct {
		EncounterID int `json:"encounter_id"`
		ConditionID int `json:"condition_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.EncounterID <= 0 || req.ConditionID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid encounter or condition id")
		return
	}
	if !requireEncounterAccess(w, r, req.EncounterID) {
		return
	}

	removed, err := encounterConditionDAO.Remove(req.ConditionID, req.EncounterID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to remove condition")
		return
	}
	if !removed {
		writeJSONError(w, http.StatusNotFound, "Condition not found")
		return
	}

	events.publish(req.EncounterID, "combat")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// apiConditionCatalogHandler returns the canonical set of condition names so the
// frontend can populate its picker without hardcoding the list.
func apiConditionCatalogHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(dao.ValidConditions)
}
