package main

import (
	"encoding/json"
	"fmt"
	"go-initiative-tracker/dao"
	"net/http"
	"strconv"
	"strings"
)

const (
	defaultTemplateSearchLimit = 25
	maxTemplateSearchLimit     = 100
)

func apiNpcTemplatesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	// GET /npcs/templates?q=&type=&size=&min_cr=&max_cr=&source=&sort=&limit=&offset=
	// All optional; one page of results, app-made templates first.
	qs := r.URL.Query()
	f := dao.TemplateFilter{
		Query: strings.TrimSpace(qs.Get("q")),
		Type:  strings.TrimSpace(qs.Get("type")),
		Size:  strings.TrimSpace(qs.Get("size")),
		MinCR: parseOptFloat(qs.Get("min_cr")),
		MaxCR: parseOptFloat(qs.Get("max_cr")),
		Limit: defaultTemplateSearchLimit,
	}
	if n, err := strconv.Atoi(qs.Get("limit")); err == nil && n > 0 {
		f.Limit = min(n, maxTemplateSearchLimit)
	}
	if n, err := strconv.Atoi(qs.Get("offset")); err == nil && n > 0 {
		f.Offset = n
	}
	switch s := qs.Get("source"); s {
	case "", "custom", "srd":
		f.Source = s
	default:
		writeJSONError(w, http.StatusBadRequest, "source must be custom or srd")
		return
	}
	switch s := qs.Get("sort"); s {
	case "", "name", "cr":
		f.SortBy = s
	default:
		writeJSONError(w, http.StatusBadRequest, "sort must be name or cr")
		return
	}
	templates, err := npcTemplateDAO.Search(f)
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
		// Count is how many copies to add (default 1).
		Count int `json:"count"`
		dao.Staging
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
	if req.Count == 0 {
		req.Count = 1
	}
	if req.Count < 1 || req.Count > dao.MaxTemplateCopies {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("Count must be between 1 and %d", dao.MaxTemplateCopies))
		return
	}
	staging, err := req.Staging.Normalize()
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	// Adding monsters is the DM's call.
	if !requireEncounterOwner(w, r, req.EncounterID) {
		return
	}

	characters, err := npcTemplateDAO.AddCharactersToEncounterFromTemplate(req.TemplateID, req.EncounterID, req.Count, staging)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to create character from template")
		return
	}
	events.publish(req.EncounterID, "character")
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "characters": characters})
}

// parseOptFloat parses an optional numeric query value; blank or invalid is nil.
func parseOptFloat(s string) *float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return nil
	}
	return &v
}

// GET /npcs/templates/facets: distinct creature types and sizes, for filters.
func apiNpcTemplateFacetsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	facets, err := npcTemplateDAO.Facets()
	if err != nil {
		fmt.Println(err)
		writeJSONError(w, http.StatusInternalServerError, "Failed to fetch npc template filters")
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(facets)
}
