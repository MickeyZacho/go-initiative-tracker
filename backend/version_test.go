package main

import (
	"encoding/json"
	"net/http"
	"testing"
)

// The /version endpoint returns the resolved build version as JSON. It touches
// no DB or DAO, so it needs no mock.
func TestVersionHandlerReturnsVersion(t *testing.T) {
	prev := version
	version = "abc1234"
	defer func() { version = prev }()

	rr, req := getReq("/version")
	apiVersionHandler(rr, req)

	assertStatus(t, rr, http.StatusOK)
	if ct := rr.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
	var body struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (body: %s)", err, rr.Body.String())
	}
	if body.Version != "abc1234" {
		t.Errorf("version = %q, want %q", body.Version, "abc1234")
	}
}

// When unstamped, resolveVersion falls back to the embedded VCS revision or
// "dev" — never a stamped placeholder that could masquerade as a real commit.
func TestResolveVersionFallsBackWhenUnstamped(t *testing.T) {
	prev := version
	version = "dev"
	defer func() { version = prev }()

	got := resolveVersion()
	// In the test binary there's usually no embedded vcs.revision, so this is
	// "dev"; if the toolchain did embed one it'll be a 7-char short hash. Either
	// way it must never be the empty string.
	if got == "" {
		t.Error("resolveVersion() returned empty string")
	}
}
