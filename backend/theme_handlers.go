package main

import (
	"bytes"
	"encoding/json"
	"net/http"
)

// maxThemeBytes caps a saved theme. Tokens are well under 1 KB; the rest is
// headroom for custom CSS.
const maxThemeBytes = 32 << 10

// apiUserThemeHandler reads (GET) or saves (POST) the caller's personal theme.
// GET works logged out too and returns a null theme, so the frontend can make
// one call on load. The theme is stored as an opaque JSON object; the frontend
// validates every field when it applies it.
func apiUserThemeHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	discordID := getDiscordIDFromRequest(r)
	switch r.Method {
	case http.MethodGet:
		var theme json.RawMessage
		if discordID != "" {
			raw, err := userDAO.GetTheme(discordID)
			if err != nil {
				writeJSONError(w, http.StatusInternalServerError, "Failed to load theme")
				return
			}
			theme = raw
		}
		if theme == nil {
			theme = json.RawMessage("null")
		}
		json.NewEncoder(w).Encode(map[string]any{"loggedIn": discordID != "", "theme": theme})
	case http.MethodPost:
		if discordID == "" {
			writeJSONError(w, http.StatusUnauthorized, "Log in to save your theme")
			return
		}
		var req struct {
			Theme json.RawMessage `json:"theme"`
		}
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxThemeBytes)).Decode(&req); err != nil {
			writeJSONError(w, http.StatusBadRequest, "Invalid theme payload")
			return
		}
		theme := bytes.TrimSpace(req.Theme)
		if len(theme) == 0 || bytes.Equal(theme, []byte("null")) {
			theme = nil
		} else if theme[0] != '{' {
			writeJSONError(w, http.StatusBadRequest, "Theme must be a JSON object")
			return
		}
		ok, err := userDAO.SetTheme(discordID, theme)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to save theme")
			return
		}
		if !ok {
			writeJSONError(w, http.StatusNotFound, "User not found")
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
	}
}
