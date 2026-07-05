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
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	templates, err := npcTemplateDAO.GetAll()
	if err != nil {
		fmt.Println(err)
		http.Error(w, "Failed to fetch npc templates", http.StatusInternalServerError)
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
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var nt dao.NpcTemplate
	if err := json.NewDecoder(r.Body).Decode(&nt); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(nt.Name) == "" {
		http.Error(w, "NPC name is required", http.StatusBadRequest)
		return
	}

	if nt.ID > 0 {
		// Update
		err := npcTemplateDAO.Update(nt)
		if err != nil {
			http.Error(w, "Failed to update npc template", http.StatusInternalServerError)
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
		http.Error(w, "Failed to create npc template", http.StatusInternalServerError)
		return
	}
	nt.ID = newID
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "npc": nt})
}

// POST /api/npcs/templates/delete
func apiDeleteNpcTemplateHandler(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "Invalid npc template id", http.StatusBadRequest)
		return
	}
	if err := npcTemplateDAO.Delete(req.ID); err != nil {
		http.Error(w, "Failed to delete npc template", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// POST /api/npcs/templates/create-character
func apiCreateCharacterFromTemplateHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TemplateID  int `json:"npc_template_id"`
		EncounterID int `json:"encounter_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request payload", http.StatusBadRequest)
		return
	}
	if req.TemplateID <= 0 {
		http.Error(w, "Invalid template id", http.StatusBadRequest)
		return
	}
	if req.EncounterID <= 0 {
		http.Error(w, "No encounter selected", http.StatusBadRequest)
		return
	}

	character, err := npcTemplateDAO.AddCharacterToEncounterFromTemplate(req.TemplateID, req.EncounterID)
	if err != nil {
		http.Error(w, "Failed to create character from template", http.StatusInternalServerError)
		return
	}
	// Optionally, insert the character into the characters table here
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "character": character})
}
