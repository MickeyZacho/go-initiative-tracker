package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
)

// writeJSONError writes a { status: "error", message } envelope with the given
// status code, matching the shape the mutation endpoints already return.
func writeJSONError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"status": "error", "message": message})
}

// requireEncounterOwner reports whether the authenticated caller owns the
// encounter, writing the appropriate error response (404/403/500) and returning
// false when they do not. Callers should `return` immediately when it is false.
// Ownership is a match between the encounter's owner_id and the signed
// discord_id cookie, so logged-out callers ("") only own logged-out encounters.
func requireEncounterOwner(w http.ResponseWriter, r *http.Request, encounterID int) bool {
	enc, err := encounterDAO.GetByID(encounterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "Encounter not found")
			return false
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to load encounter")
		return false
	}
	if enc.OwnerID != getDiscordIDFromRequest(r) {
		writeJSONError(w, http.StatusForbidden, "You do not own this encounter")
		return false
	}
	return true
}

// requireEncounterAccess reports whether the authenticated caller may edit the
// encounter's shared surface (combat, ledger, roster) — true when they own it or
// have been added as a shared-edit member (encounter_users). It writes the
// appropriate error response and returns false otherwise; callers should
// `return` immediately when it is false. Logged-out callers ("") only match
// logged-out encounters and are never members.
func requireEncounterAccess(w http.ResponseWriter, r *http.Request, encounterID int) bool {
	enc, err := encounterDAO.GetByID(encounterID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "Encounter not found")
			return false
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to load encounter")
		return false
	}
	discordID := getDiscordIDFromRequest(r)
	if enc.OwnerID == discordID {
		return true
	}
	if discordID != "" {
		isMember, err := encounterDAO.IsMember(encounterID, discordID)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, "Failed to check encounter access")
			return false
		}
		if isMember {
			return true
		}
	}
	writeJSONError(w, http.StatusForbidden, "You do not have access to this encounter")
	return false
}
