package main

import (
	"encoding/json"
	"go-initiative-tracker/dao"
	"net/http"
	"strconv"
	"strings"
)

func apiEncountersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	discordID := getDiscordIDFromRequest(r)
	var data []dao.Encounter
	var err error
	if discordID != "" {
		// Include encounters the user owns and any they are a shared-edit member of.
		data, err = encounterDAO.GetAccessibleEncounters(discordID)
	} else {
		data, err = encounterDAO.GetAllEncounters()
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to fetch encounters")
		return
	}
	if data == nil {
		data = []dao.Encounter{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func apiSaveEncounterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	var enc dao.Encounter
	if err := json.NewDecoder(r.Body).Decode(&enc); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if strings.TrimSpace(enc.Name) == "" {
		writeJSONError(w, http.StatusBadRequest, "Encounter name is required")
		return
	}
	if enc.OwnerID == "" {
		enc.OwnerID = getDiscordIDFromRequest(r)
	}
	newID, err := encounterDAO.CreateEncounter(enc)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create encounter")
		return
	}
	enc.ID = newID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "encounter": enc})
}

func apiDeleteEncounterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.ID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid encounter id")
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	deleted, err := encounterDAO.DeleteEncounterByOwner(req.ID, discordID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to delete encounter")
		return
	}
	if !deleted {
		writeJSONError(w, http.StatusForbidden, "Encounter not found or not owned by you")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func apiEncounterLedgerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	encounterIDRaw := r.URL.Query().Get("encounter_id")
	encounterID, err := strconv.Atoi(encounterIDRaw)
	if err != nil || encounterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid encounter id")
		return
	}
	if !requireEncounterAccess(w, r, encounterID) {
		return
	}

	entries, err := encounterLedgerDAO.ListByEncounterID(encounterID, 50)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load encounter log")
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"status": "success", "entries": entries})
}

func apiAddEncounterLedgerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var req struct {
		EncounterID int    `json:"encounter_id"`
		ActorID     int    `json:"actor_id"`
		TargetID    int    `json:"target_id"`
		ActionType  string `json:"action_type"`
		HPChange    int    `json:"hp_change"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.EncounterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid encounter id")
		return
	}
	if req.ActorID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Actor is required")
		return
	}
	if !requireEncounterAccess(w, r, req.EncounterID) {
		return
	}
	actionType := strings.TrimSpace(req.ActionType)
	if actionType == "" {
		actionType = "note"
	}

	entry, err := encounterLedgerDAO.Create(dao.EncounterLedgerInsert{
		EncounterID: req.EncounterID,
		ActorID:     req.ActorID,
		TargetID:    req.TargetID,
		ActionType:  actionType,
		HPChange:    req.HPChange,
		Description: strings.TrimSpace(req.Description),
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to add combat log entry")
		return
	}

	events.publish(req.EncounterID, "ledger")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "entry": entry})
}
