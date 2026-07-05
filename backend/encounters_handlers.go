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
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	discordID := getDiscordIDFromRequest(r)
	var data []dao.Encounter
	var err error
	if discordID != "" {
		data, err = encounterDAO.GetEncountersByOwnerDiscordID(discordID)
	} else {
		data, err = encounterDAO.GetAllEncounters()
	}
	if err != nil {
		http.Error(w, "Failed to fetch encounters", http.StatusInternalServerError)
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
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var enc dao.Encounter
	if err := json.NewDecoder(r.Body).Decode(&enc); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(enc.Name) == "" {
		http.Error(w, "Encounter name is required", http.StatusBadRequest)
		return
	}
	if enc.OwnerID == "" {
		enc.OwnerID = getDiscordIDFromRequest(r)
	}
	newID, err := encounterDAO.CreateEncounter(enc)
	if err != nil {
		http.Error(w, "Failed to create encounter", http.StatusInternalServerError)
		return
	}
	enc.ID = newID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "encounter": enc})
}

func apiDeleteEncounterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		ID int `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if req.ID <= 0 {
		http.Error(w, "Invalid encounter id", http.StatusBadRequest)
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	deleted, err := encounterDAO.DeleteEncounterByOwner(req.ID, discordID)
	if err != nil {
		http.Error(w, "Failed to delete encounter", http.StatusInternalServerError)
		return
	}
	if !deleted {
		http.Error(w, "Encounter not found or not owned by you", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func apiEncounterLedgerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}

	encounterIDRaw := r.URL.Query().Get("encounter_id")
	encounterID, err := strconv.Atoi(encounterIDRaw)
	if err != nil || encounterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid encounter id"})
		return
	}
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}

	entries, err := encounterLedgerDAO.ListByEncounterID(encounterID, 50)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to load encounter log"})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"status": "success", "entries": entries})
}

func apiAddEncounterLedgerHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
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
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}
	if req.EncounterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid encounter id"})
		return
	}
	if req.ActorID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Actor is required"})
		return
	}
	if !requireEncounterOwner(w, r, req.EncounterID) {
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
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to add combat log entry"})
		return
	}

	json.NewEncoder(w).Encode(map[string]any{"status": "success", "entry": entry})
}
