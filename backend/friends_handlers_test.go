package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestApiFriendsHandlerRequiresLogin(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/friends", nil)
	rr := httptest.NewRecorder()

	apiFriendsHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("apiFriendsHandler status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestApiSendFriendRequestHandlerWrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/friends/request", nil)
	rr := httptest.NewRecorder()

	apiSendFriendRequestHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("apiSendFriendRequestHandler status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestApiSendFriendRequestHandlerRequiresLogin(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"username": "someone"})
	req := httptest.NewRequest(http.MethodPost, "/friends/request", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	apiSendFriendRequestHandler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Errorf("apiSendFriendRequestHandler status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestApiAddEncounterMemberHandlerRequiresFields(t *testing.T) {
	body, _ := json.Marshal(map[string]any{"encounter_id": 0, "user_id": ""})
	req := httptest.NewRequest(http.MethodPost, "/encounters/members/add", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	apiAddEncounterMemberHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("apiAddEncounterMemberHandler status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
