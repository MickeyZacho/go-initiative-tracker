package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"go-initiative-tracker/dao"
	"net/http"
	"strconv"
	"strings"
)

// friendActionRequest is the shared body for accept/decline/remove: the other
// user's Discord id.
type friendActionRequest struct {
	DiscordID string `json:"discord_id"`
}

// apiFriendsHandler returns the caller's accepted friends.
func apiFriendsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		writeJSONError(w, http.StatusUnauthorized, "You must be logged in")
		return
	}
	friends, err := friendshipDAO.ListFriends(discordID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load friends")
		return
	}
	if friends == nil {
		friends = []dao.Friend{}
	}
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "friends": friends})
}

// apiFriendRequestsHandler returns the caller's pending incoming and outgoing
// friend requests.
func apiFriendRequestsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		writeJSONError(w, http.StatusUnauthorized, "You must be logged in")
		return
	}
	incoming, err := friendshipDAO.ListIncoming(discordID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load requests")
		return
	}
	outgoing, err := friendshipDAO.ListOutgoing(discordID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load requests")
		return
	}
	if incoming == nil {
		incoming = []dao.Friend{}
	}
	if outgoing == nil {
		outgoing = []dao.Friend{}
	}
	json.NewEncoder(w).Encode(map[string]any{
		"status":   "success",
		"incoming": incoming,
		"outgoing": outgoing,
	})
}

// apiSendFriendRequestHandler sends a friend request by Discord username. The
// target must have logged into this app at least once (exist in users).
func apiSendFriendRequestHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		writeJSONError(w, http.StatusUnauthorized, "You must be logged in")
		return
	}
	var req struct {
		Username string `json:"username"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		writeJSONError(w, http.StatusBadRequest, "Username is required")
		return
	}
	target, err := userDAO.GetUserByUsername(username)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSONError(w, http.StatusNotFound, "No user with that username has signed in to this app")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to look up user")
		return
	}
	if target.DiscordID == discordID {
		writeJSONError(w, http.StatusBadRequest, "You cannot friend yourself")
		return
	}
	if err := friendshipDAO.SendRequest(discordID, target.DiscordID); err != nil {
		if errors.Is(err, dao.ErrFriendshipExists) {
			writeJSONError(w, http.StatusConflict, "You are already friends or have a pending request with this user")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "Failed to send friend request")
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// apiAcceptFriendHandler accepts a pending incoming request from the given user.
func apiAcceptFriendHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		writeJSONError(w, http.StatusUnauthorized, "You must be logged in")
		return
	}
	var req friendActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.DiscordID) == "" {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	accepted, err := friendshipDAO.Accept(discordID, req.DiscordID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to accept request")
		return
	}
	if !accepted {
		writeJSONError(w, http.StatusNotFound, "No pending request from this user")
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// apiRemoveFriendHandler removes a friendship or request in either direction. It
// backs decline (incoming), cancel (outgoing), and unfriend.
func apiRemoveFriendHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		writeJSONError(w, http.StatusUnauthorized, "You must be logged in")
		return
	}
	var req friendActionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || strings.TrimSpace(req.DiscordID) == "" {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if _, err := friendshipDAO.Remove(discordID, req.DiscordID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to remove friend")
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// apiEncounterMembersHandler lists the shared-edit members of an encounter.
// Owner-only, since it exposes who the encounter is shared with.
func apiEncounterMembersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	encounterID, err := strconv.Atoi(r.URL.Query().Get("encounter_id"))
	if err != nil || encounterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid encounter id")
		return
	}
	if !requireEncounterOwner(w, r, encounterID) {
		return
	}
	members, err := encounterDAO.ListMembers(encounterID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to load members")
		return
	}
	if members == nil {
		members = []dao.Friend{}
	}
	json.NewEncoder(w).Encode(map[string]any{"status": "success", "members": members})
}

// apiAddEncounterMemberHandler shares an encounter with a friend, granting them
// shared-edit access. Owner-only, and the target must be an accepted friend.
func apiAddEncounterMemberHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	var req struct {
		EncounterID int    `json:"encounter_id"`
		UserID      string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.EncounterID <= 0 || strings.TrimSpace(req.UserID) == "" {
		writeJSONError(w, http.StatusBadRequest, "encounter_id and user_id are required")
		return
	}
	discordID := getDiscordIDFromRequest(r)
	if discordID == "" {
		writeJSONError(w, http.StatusUnauthorized, "You must be logged in")
		return
	}
	if !requireEncounterOwner(w, r, req.EncounterID) {
		return
	}
	// Only accepted friends can be added, so an encounter can't be shared with an
	// arbitrary Discord id.
	friends, err := friendshipDAO.AreFriends(discordID, req.UserID)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to verify friendship")
		return
	}
	if !friends {
		writeJSONError(w, http.StatusForbidden, "You can only share encounters with your friends")
		return
	}
	if err := encounterDAO.AddMember(req.EncounterID, req.UserID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to add member")
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// apiRemoveEncounterMemberHandler revokes a member's shared-edit access.
// Owner-only.
func apiRemoveEncounterMemberHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	var req struct {
		EncounterID int    `json:"encounter_id"`
		UserID      string `json:"user_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if req.EncounterID <= 0 || strings.TrimSpace(req.UserID) == "" {
		writeJSONError(w, http.StatusBadRequest, "encounter_id and user_id are required")
		return
	}
	if !requireEncounterOwner(w, r, req.EncounterID) {
		return
	}
	if err := encounterDAO.RemoveMember(req.EncounterID, req.UserID); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Failed to remove member")
		return
	}
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}
