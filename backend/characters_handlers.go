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
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	encounterID := 0
	if encounterIDRaw := r.URL.Query().Get("encounter_id"); encounterIDRaw != "" {
		id, err := strconv.Atoi(encounterIDRaw)
		if err != nil || id <= 0 {
			writeJSONError(w, http.StatusBadRequest, "Invalid encounter id")
			return
		}
		encounterID = id
	}
	characters, err := fetchCharacters(r, encounterID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to fetch characters")
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
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
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
		writeJSONError(w, http.StatusInternalServerError, "Failed to fetch characters")
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
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	var char dao.Character
	if err := json.NewDecoder(r.Body).Decode(&char); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if strings.TrimSpace(char.Name) == "" {
		writeJSONError(w, http.StatusBadRequest, "Character name is required")
		return
	}
	if char.MaxHP < 1 {
		writeJSONError(w, http.StatusBadRequest, "Invalid max HP value")
		return
	}
	// Ownership is always the authenticated caller; never trust an owner_id
	// supplied in the request body.
	discordID := getDiscordIDFromRequest(r)
	char.OwnerID = discordID
	if char.ID == 0 {
		newID, err := characterDAO.CreateCharacter(char)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to create character")
			return
		}
		char.ID = newID
	} else {
		updated, err := characterDAO.UpdateCharacterByOwner(char, discordID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to update character")
			return
		}
		if !updated {
			writeJSONError(w, http.StatusForbidden, "Character not found or not owned by you")
			return
		}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "character": char})
}

func apiDeleteLibraryCharacterHandler(w http.ResponseWriter, r *http.Request) {
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
		writeJSONError(w, http.StatusBadRequest, "Invalid character id")
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		writeJSONError(w, http.StatusUnauthorized, "Unauthorized")
		return
	}
	deleted, err := characterDAO.DeleteCharacterByOwner(req.ID, discordID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to delete character")
		return
	}
	if !deleted {
		writeJSONError(w, http.StatusForbidden, "Character not found or not owned by you")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func saveCharacterHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("Saving character...")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}

	var payload struct {
		dao.Character
		EncounterID int `json:"encounter_id"`
	}
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	char := payload.Character

	if char.MaxHP < 1 {
		writeJSONError(w, http.StatusBadRequest, "Invalid max HP value")
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
			writeJSONError(w, http.StatusInternalServerError, "Failed to create character")
			return
		}
		char.ID = newID
	} else {
		// Update existing character, but only if the caller owns it.
		updated, err := characterDAO.UpdateCharacterByOwner(char, discordID)
		if err != nil {
			log.Printf("Error updating character: %v", err)
			writeJSONError(w, http.StatusInternalServerError, "Failed to update character")
			return
		}
		if !updated {
			writeJSONError(w, http.StatusForbidden, "Character not found or not owned by you")
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
			writeJSONError(w, http.StatusInternalServerError, "Failed to save encounter values")
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
	if req.CharacterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid character id")
		return
	}
	encounterID := req.EncounterID
	if encounterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "encounter_id is required")
		return
	}
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}
	err := encounterDAO.AddCharacterToEncounter(encounterID, req.CharacterID)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "duplicate key") {
			writeJSONError(w, http.StatusConflict, "Character is already in this encounter")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to add character to encounter")
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// Handler to remove a character from the selected encounter
func removeCharacterFromEncounterHandler(w http.ResponseWriter, r *http.Request) {
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
	if req.CharacterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid character id")
		return
	}
	encounterID := req.EncounterID
	if encounterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "encounter_id is required")
		return
	}
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}
	err := encounterDAO.RemoveCharacterFromEncounter(encounterID, req.CharacterID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to remove character from encounter")
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
