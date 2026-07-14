package main

import (
	"bufio"
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestEventHubDeliversToSubscriber(t *testing.T) {
	h := newEventHub()
	ch := h.subscribe(7)
	defer h.unsubscribe(7, ch)

	h.publish(7, "combat")

	select {
	case msg := <-ch:
		if msg != "combat" {
			t.Errorf("got %q, want %q", msg, "combat")
		}
	case <-time.After(time.Second):
		t.Fatal("subscriber did not receive the published message")
	}
}

func TestEventHubIsolatesEncounters(t *testing.T) {
	h := newEventHub()
	ch := h.subscribe(7)
	defer h.unsubscribe(7, ch)

	// A publish to a different encounter must not reach this subscriber.
	h.publish(8, "combat")

	select {
	case msg := <-ch:
		t.Fatalf("received unexpected message for another encounter: %q", msg)
	case <-time.After(50 * time.Millisecond):
		// expected: nothing delivered
	}
}

func TestEventHubPublishDoesNotBlockOnFullSubscriber(t *testing.T) {
	h := newEventHub()
	ch := h.subscribe(7)
	defer h.unsubscribe(7, ch)

	// Overfill well past the channel buffer; the non-blocking send must drop
	// extras rather than deadlock the publisher.
	done := make(chan struct{})
	go func() {
		for range 1000 {
			h.publish(7, "spam")
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("publish blocked on a full subscriber channel")
	}
}

func TestEventHubUnsubscribeStopsDelivery(t *testing.T) {
	h := newEventHub()
	ch := h.subscribe(7)
	h.unsubscribe(7, ch)

	h.publish(7, "combat")

	select {
	case msg, ok := <-ch:
		if ok {
			t.Fatalf("received message after unsubscribe: %q", msg)
		}
	case <-time.After(50 * time.Millisecond):
		// expected: no delivery to an unsubscribed channel
	}
}

// TestApiEncounterEventsHandlerStreamsPublishedEvents drives the handler over a
// real HTTP server: it connects, waits for the initial comment, publishes an
// event, and asserts the client receives it as an SSE data frame. This exercises
// the streaming path (flush + response controller) end to end.
func TestApiEncounterEventsHandlerStreamsPublishedEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(apiEncounterEventsHandler))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, srv.URL+"/encounters/events?encounter_id=4242", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want text/event-stream", ct)
	}

	reader := bufio.NewReader(resp.Body)
	// First line is the initial ": connected" comment, written before subscribe
	// completes — read it so the next publish is guaranteed to be delivered.
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatalf("failed to read initial stream line: %v", err)
	}

	// The handler subscribes to encounter 4242 on the global hub; publish there.
	// Retry briefly to remove any race between the read above and subscription.
	got := make(chan string, 1)
	go func() {
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			if after, ok := strings.CutPrefix(line, "data: "); ok {
				got <- strings.TrimSpace(after)
				return
			}
		}
	}()

	deadline := time.After(3 * time.Second)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case msg := <-got:
			if msg != "combat" {
				t.Fatalf("received %q, want %q", msg, "combat")
			}
			return
		case <-tick.C:
			events.publish(4242, "combat")
		case <-deadline:
			t.Fatal("did not receive the published event over the stream")
		}
	}
}

func TestApiEncounterEventsHandlerWrongMethod(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/encounters/events?encounter_id=1", nil)
	rr := httptest.NewRecorder()

	apiEncounterEventsHandler(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestApiEncounterEventsHandlerRequiresEncounterID(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/encounters/events", nil)
	rr := httptest.NewRecorder()

	apiEncounterEventsHandler(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
}
