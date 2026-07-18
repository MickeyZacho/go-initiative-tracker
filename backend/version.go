package main

import (
	"encoding/json"
	"net/http"
	"runtime/debug"
)

// version identifies the running backend binary. It defaults to "dev" and is
// stamped at build time via ldflags (see backend/Dockerfile):
//
//	go build -ldflags "-X main.version=<short-sha>" -o server .
//
// When it isn't stamped we fall back to the VCS revision the Go toolchain
// embeds automatically, but only when the build had access to the .git dir
// (it doesn't inside the Docker image, which is why ldflags is the primary
// path). Kept in one place so the /version endpoint is the single source of
// truth for "which commit is this server running".
var version = "dev"

// resolveVersion returns the stamped version, or the embedded VCS revision as a
// fallback, or "dev" when neither is available.
func resolveVersion() string {
	if version != "" && version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				rev := s.Value
				if len(rev) > 7 {
					rev = rev[:7]
				}
				return rev
			}
		}
	}
	return version
}

// apiVersionHandler reports the backend build version as JSON: {"version": ...}.
// Public and unauthenticated by design — it exposes only a short commit hash.
func apiVersionHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Invalid request method", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"version": resolveVersion(),
	})
}
