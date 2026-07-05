package main

import (
	"encoding/json"
	"go-initiative-tracker/dao"
	"log"
	"net/http"
	"strconv"
	"strings"
)

// fetchCharacters loads characters for the given request, scoped to encounterID
// when it is > 0 and to the signed-in Discord user when a cookie is present.
// It holds no server-side state, so concurrent requests never see each other's
// selection.
func fetchCharacters(r *http.Request, encounterID int) ([]dao.Character, error) {
	discordID := getDiscordIDFromRequest(r)
	if encounterID > 0 {
		if discordID != "" {
			return characterDAO.GetCharactersByEncounterIDAndOwner(encounterID, discordID)
		}
		return characterDAO.GetCharactersByEncounterID(encounterID)
	}
	if discordID != "" {
		return characterDAO.GetAllCharactersByOwner(discordID)
	}
	return characterDAO.GetAllCharacters()
}

func apiCharactersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	encounterID := 0
	if encounterIDRaw := r.URL.Query().Get("encounter_id"); encounterIDRaw != "" {
		id, err := strconv.Atoi(encounterIDRaw)
		if err != nil || id <= 0 {
			http.Error(w, "Invalid encounter id", http.StatusBadRequest)
			return
		}
		encounterID = id
	}
	characters, err := fetchCharacters(r, encounterID)
	if err != nil {
		http.Error(w, "Failed to fetch characters", http.StatusInternalServerError)
		return
	}
	if characters == nil {
		characters = []dao.Character{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(characters)
}

func apiLibraryCharactersHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	discordID := getDiscordIDFromRequest(r)
	var data []dao.Character
	var err error
	if discordID != "" {
		data, err = characterDAO.GetAllCharactersByOwner(discordID)
	} else {
		data, err = characterDAO.GetAllCharacters()
	}
	if err != nil {
		http.Error(w, "Failed to fetch characters", http.StatusInternalServerError)
		return
	}
	if data == nil {
		data = []dao.Character{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func apiSaveLibraryCharacterHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var char dao.Character
	if err := json.NewDecoder(r.Body).Decode(&char); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(char.Name) == "" {
		http.Error(w, "Character name is required", http.StatusBadRequest)
		return
	}
	if char.MaxHP < 1 {
		http.Error(w, "Invalid max HP value", http.StatusBadRequest)
		return
	}
	// Ownership is always the authenticated caller; never trust an owner_id
	// supplied in the request body.
	discordID := getDiscordIDFromRequest(r)
	char.OwnerID = discordID
	if char.ID == 0 {
		newID, err := characterDAO.CreateCharacter(char)
		if err != nil {
			http.Error(w, "Failed to create character", http.StatusInternalServerError)
			return
		}
		char.ID = newID
	} else {
		updated, err := characterDAO.UpdateCharacterByOwner(char, discordID)
		if err != nil {
			http.Error(w, "Failed to update character", http.StatusInternalServerError)
			return
		}
		if !updated {
			http.Error(w, "Character not found or not owned by you", http.StatusForbidden)
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "character": char})
}

func apiDeleteLibraryCharacterHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Invalid character id", http.StatusBadRequest)
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}
	deleted, err := characterDAO.DeleteCharacterByOwner(req.ID, discordID)
	if err != nil {
		http.Error(w, "Failed to delete character", http.StatusInternalServerError)
		return
	}
	if !deleted {
		http.Error(w, "Character not found or not owned by you", http.StatusForbidden)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func saveCharacterHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Saving character...")
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}

	var payload struct {
		dao.Character
		EncounterID int `json:"encounter_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	char := payload.Character

	if char.MaxHP < 1 {
		http.Error(w, "Invalid max HP value", http.StatusBadRequest)
		return
	}
	if char.CurrentHP < 0 {
		char.CurrentHP = 0
	}
	if char.CurrentHP > char.MaxHP {
		char.CurrentHP = char.MaxHP
	}
	// Ownership is always the authenticated caller; never trust an owner_id
	// supplied in the request body.
	discordID := getDiscordIDFromRequest(r)
	char.OwnerID = discordID

	if char.ID == 0 {
		// New character, insert into DB
		newID, err := characterDAO.CreateCharacter(char)
		if err != nil {
			log.Printf("Error creating character: %v", err)
			http.Error(w, "Failed to create character", http.StatusInternalServerError)
			return
		}
		char.ID = newID
	} else {
		// Update existing character, but only if the caller owns it.
		updated, err := characterDAO.UpdateCharacterByOwner(char, discordID)
		if err != nil {
			log.Printf("Error updating character: %v", err)
			http.Error(w, "Failed to update character", http.StatusInternalServerError)
			return
		}
		if !updated {
			http.Error(w, "Character not found or not owned by you", http.StatusForbidden)
			return
		}
	}

	if payload.EncounterID > 0 {
		encChar := dao.EncounterCharacter{
			EncounterID: payload.EncounterID,
			CharacterID: char.ID,
			Initiative:  char.Initiative,
			CurrentHP:   char.CurrentHP,
			IsActive:    char.IsActive,
		}
		err = encounterCharacterDAO.Upsert(encChar)
		if err != nil {
			log.Printf("Error upserting encounter character: %v", err)
			http.Error(w, "Failed to save encounter values", http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "success",
		"character": char,
	})
}

// Handler to add a character to the selected encounter
func addCharacterToEncounterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}
	var req struct {
		EncounterID int `json:"encounter_id"`
		CharacterID int `json:"character_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}
	if req.CharacterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid character id"})
		return
	}
	encounterID := req.EncounterID
	if encounterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "encounter_id is required"})
		return
	}
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}
	err := encounterDAO.AddCharacterToEncounter(encounterID, req.CharacterID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Character is already in this encounter"})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to add character to encounter"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Handler to remove a character from the selected encounter
func removeCharacterFromEncounterHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request method"})
		return
	}
	var req struct {
		EncounterID int `json:"encounter_id"`
		CharacterID int `json:"character_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid request payload"})
		return
	}
	if req.CharacterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Invalid character id"})
		return
	}
	encounterID := req.EncounterID
	if encounterID <= 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "encounter_id is required"})
		return
	}
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}
	err := encounterDAO.RemoveCharacterFromEncounter(encounterID, req.CharacterID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": "Failed to remove character from encounter"})
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
