package main

import (
	"net/http"
	"strconv"
	"sync"
	"time"
)

// eventHub is a tiny in-memory pub/sub keyed by encounter id. Each subscriber
// (an open SSE connection) gets its own buffered channel; publish fans a message
// out to every subscriber of that encounter with a non-blocking send, so a slow
// or stuck client can never block a mutation handler. A single backend process
// runs in production, so in-memory state is sufficient; scaling to multiple
// backends would require Postgres LISTEN/NOTIFY or similar here instead.
type eventHub struct {
	mu   sync.Mutex
	subs map[int]map[chan string]struct{}
}

func newEventHub() *eventHub {
	return &eventHub{subs: make(map[int]map[chan string]struct{})}
}

var events = newEventHub()

func (h *eventHub) subscribe(encounterID int) chan string {
	// Small buffer so a brief scheduling gap between publishes doesn't drop
	// messages; publish still never blocks if this fills up.
	ch := make(chan string, 8)
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.subs[encounterID] == nil {
		h.subs[encounterID] = make(map[chan string]struct{})
	}
	h.subs[encounterID][ch] = struct{}{}
	return ch
}

func (h *eventHub) unsubscribe(encounterID int, ch chan string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if subs := h.subs[encounterID]; subs != nil {
		delete(subs, ch)
		if len(subs) == 0 {
			delete(h.subs, encounterID)
		}
	}
}

// publish delivers msg to every current subscriber of encounterID. The send is
// non-blocking: if a subscriber's buffer is full it is skipped (that client will
// still re-sync on its next received event or heartbeat-driven reconnect).
func (h *eventHub) publish(encounterID int, msg string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.subs[encounterID] {
		select {
		case ch <- msg:
		default:
		}
	}
}

// apiEncounterEventsHandler streams encounter change notifications to a viewer as
// Server-Sent Events. It broadcasts a nudge ("something changed"), not state; the
// client responds by re-fetching characters and the ledger. Access is gated the
// same way as apiCharactersHandler: logged-in callers must own or be a shared-edit
// member of the encounter, while logged-out callers keep the lenient read behavior.
func apiEncounterEventsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "Invalid request method")
		return
	}
	encounterID, err := strconv.Atoi(r.URL.Query().Get("encounter_id"))
	if err != nil || encounterID <= 0 {
		writeJSONError(w, http.StatusBadRequest, "Invalid encounter id")
		return
	}
	if getDiscordIDFromRequest(r) != "" {
		if !requireEncounterAccess(w, r, encounterID) {
			return
		}
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// The server sets a 30s WriteTimeout for slow-client protection; clear the
	// write deadline on this connection so the long-lived stream is not killed.
	rc := http.NewResponseController(w)
	if err := rc.SetWriteDeadline(time.Time{}); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "Streaming unsupported")
		return
	}

	// Subscribe before writing anything so no publish can slip through between
	// opening the stream and registering this connection.
	ch := events.subscribe(encounterID)
	defer events.unsubscribe(encounterID, ch)

	// Flush headers (and an initial comment) immediately so intermediary proxies
	// open the stream to the client right away.
	if _, err := w.Write([]byte(": connected\n\n")); err != nil {
		return
	}
	rc.Flush()

	// Heartbeat below Cloudflare's ~100s idle timeout keeps the connection open
	// through the tunnel when no real events are flowing.
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg := <-ch:
			if _, err := w.Write([]byte("data: " + msg + "\n\n")); err != nil {
				return
			}
			rc.Flush()
		case <-ticker.C:
			if _, err := w.Write([]byte(": ping\n\n")); err != nil {
				return
			}
			rc.Flush()
		}
	}
}
