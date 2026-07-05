package main

import (
	"encoding/json"
	"fmt"
	"go-initiative-tracker/dao"
	"net/http"
	"strings"
)

func apiNpcTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	templates, err := npcTemplateDAO.GetAll()
	if err != nil {
		fmt.Println(err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to fetch npc templates")
		return
	}
	if templates == nil {
		templates = []dao.NpcTemplate{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(templates)
}

// POST /api/npcs/templates/save
func apiSaveNpcTemplateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	var nt dao.NpcTemplate
	if err := json.NewDecoder(r.Body).Decode(&nt); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if strings.TrimSpace(nt.Name) == "" {
		writeJSONError(w, http.StatusBadRequest, "NPC name is required")
		return
	}

	// Ownership is always the authenticated caller; never trust a body value.
	discordID := getDiscordIDFromRequest(r)
	nt.OwnerID = discordID

	if nt.ID > 0 {
		// Update, but only if the caller owns the template.
		updated, err := npcTemplateDAO.UpdateByOwner(nt, discordID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to update npc template")
			return
		}
		if !updated {
			writeJSONError(w, http.StatusForbidden, "NPC template not found or not owned by you")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "success", "npc": nt})
		return
	}

	// Create new template
	newID, err := npcTemplateDAO.Create(nt)
	if err != nil {
		fmt.Println(nt)
		fmt.Println(err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to create npc template")
		return
	}
	nt.ID = newID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "npc": nt})
}

// POST /api/npcs/templates/delete
func apiDeleteNpcTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
		writeJSONError(w, http.StatusBadRequest, "Invalid npc template id")
		return
	}
	deleted, err := npcTemplateDAO.DeleteByOwner(req.ID, getDiscordIDFromRequest(r))
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to delete npc template")
		return
	}
	if !deleted {
		writeJSONError(w, http.StatusForbidden, "NPC template not found or not owned by you")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// POST /api/npcs/templates/create-character
func apiCreateCharacterFromTemplateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	var req struct {
		TemplateID  int `json:"npc_template_id"`
		EncounterID int `json:"encounter_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.TemplateID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid template id")
		return
	}
	if req.EncounterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "No encounter selected")
		return
	}
	if !requireEncounterOwner(w, r, req.EncounterID) {
		return
	}

	character, err := npcTemplateDAO.AddCharacterToEncounterFromTemplate(req.TemplateID, req.EncounterID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create character from template")
		return
	}
	// Optionally, insert the character into the characters table here
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "character": character})
}
