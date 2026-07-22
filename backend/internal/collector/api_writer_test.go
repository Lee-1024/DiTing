package collector

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"diting/backend/internal/audit"
)

func TestAPIWriterPostsEventsWithCollectorToken(t *testing.T) {
	var authHeader string
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"accepted":1}`))
	}))
	defer server.Close()

	writer := NewAPIWriter(server.URL+"/api/v1/ingest/events", "secret-token")
	err := writer.Write(context.Background(), []audit.Event{{
		EventID: "evt-1", EventTime: time.Unix(1, 0).UTC(), EventType: "process_exec", Severity: "info", ProcessName: "id", Cmdline: "/usr/bin/id",
	}})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if authHeader != "Bearer secret-token" {
		t.Fatalf("expected bearer token, got %q", authHeader)
	}
	if !strings.Contains(body, `"events":[`) || !strings.Contains(body, `"eventId":"evt-1"`) {
		t.Fatalf("expected events payload, got %s", body)
	}
}

func TestAPIWriterPostsHeartbeatWithCollectorToken(t *testing.T) {
	var path string
	var authHeader string
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		authHeader = r.Header.Get("Authorization")
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"accepted":true}`))
	}))
	defer server.Close()

	writer := NewAPIWriter(server.URL+"/api/v1/ingest/events", "secret-token")
	err := writer.WriteHeartbeat(context.Background(), APIHeartbeat{
		HostID:         "server-1",
		HostName:       "server-1",
		InputMode:      "grpc",
		ClearError:     true,
		EventsWritten:  2,
		BufferedEvents: 3,
		DroppedEvents:  1,
	})
	if err != nil {
		t.Fatalf("WriteHeartbeat returned error: %v", err)
	}

	if path != "/api/v1/ingest/heartbeat" {
		t.Fatalf("expected heartbeat path, got %q", path)
	}
	if authHeader != "Bearer secret-token" {
		t.Fatalf("expected bearer token, got %q", authHeader)
	}
	if !strings.Contains(body, `"hostId":"server-1"`) || !strings.Contains(body, `"eventsWritten":2`) || !strings.Contains(body, `"bufferedEvents":3`) || !strings.Contains(body, `"droppedEvents":1`) || !strings.Contains(body, `"clearError":true`) {
		t.Fatalf("expected heartbeat payload, got %s", body)
	}
}

func TestAPIWriterRetriesRetriableStatus(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts == 1 {
			http.Error(w, "temporary failure", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	writer := NewAPIWriter(server.URL+"/api/v1/ingest/events", "secret-token")
	writer.SetRetryPolicy(2, time.Millisecond)

	err := writer.Write(context.Background(), []audit.Event{{EventID: "evt-1", EventType: "process_exec"}})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
}

func TestAPIWriterDoesNotRetryUnauthorized(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		http.Error(w, "unauthorized", http.StatusUnauthorized)
	}))
	defer server.Close()

	writer := NewAPIWriter(server.URL+"/api/v1/ingest/events", "bad-token")
	writer.SetRetryPolicy(3, time.Millisecond)

	err := writer.Write(context.Background(), []audit.Event{{EventID: "evt-1", EventType: "process_exec"}})
	if err == nil {
		t.Fatal("expected Write to return error")
	}
	if attempts != 1 {
		t.Fatalf("expected unauthorized response to skip retry, got %d attempts", attempts)
	}
	if !strings.Contains(err.Error(), strconv.Itoa(http.StatusUnauthorized)) {
		t.Fatalf("expected status in error, got %v", err)
	}
}

func TestAPIWriterBuffersFailedEventsAndFlushesOnNextWrite(t *testing.T) {
	requests := [][]string{}
	failFirst := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Events []audit.Event `json:"events"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		ids := make([]string, 0, len(payload.Events))
		for _, event := range payload.Events {
			ids = append(ids, event.EventID)
		}
		requests = append(requests, ids)
		if failFirst {
			failFirst = false
			http.Error(w, "temporary failure", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	writer := NewAPIWriter(server.URL+"/api/v1/ingest/events", "secret-token")
	writer.SetRetryPolicy(1, 0)
	writer.SetBufferLimit(10)

	if err := writer.Write(context.Background(), []audit.Event{{EventID: "evt-1", EventType: "process_exec"}}); err != nil {
		t.Fatalf("expected failed write to be buffered without returning error, got %v", err)
	}
	if writer.BufferedEvents() != 1 {
		t.Fatalf("expected one buffered event, got %d", writer.BufferedEvents())
	}
	if !writer.LastWriteBuffered() {
		t.Fatal("expected failed write to be marked as buffered")
	}

	if err := writer.Write(context.Background(), []audit.Event{{EventID: "evt-2", EventType: "process_exec"}}); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}
	if writer.LastWriteBuffered() {
		t.Fatal("expected successful write to clear buffered marker")
	}
	if writer.BufferedEvents() != 0 {
		t.Fatalf("expected buffer to flush, got %d buffered events", writer.BufferedEvents())
	}
	if len(requests) != 3 || requests[1][0] != "evt-1" || requests[2][0] != "evt-2" {
		t.Fatalf("expected buffered event to flush before new event, got %#v", requests)
	}
}

func TestAPIWriterBufferLimitKeepsHigherSeverityEvents(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "temporary failure", http.StatusInternalServerError)
	}))
	defer server.Close()

	writer := NewAPIWriter(server.URL+"/api/v1/ingest/events", "secret-token")
	writer.SetRetryPolicy(1, 0)
	writer.SetBufferLimit(2)

	_ = writer.Write(context.Background(), []audit.Event{{EventID: "low-1", EventType: "process_exec", Severity: "info"}})
	_ = writer.Write(context.Background(), []audit.Event{{EventID: "low-2", EventType: "process_exec", Severity: "low"}})
	_ = writer.Write(context.Background(), []audit.Event{{EventID: "high-1", EventType: "process_exec", Severity: "critical"}})

	buffered := writer.BufferedEventIDs()
	if len(buffered) != 2 || buffered[0] != "low-2" || buffered[1] != "high-1" {
		t.Fatalf("expected buffer to drop lowest oldest event, got %#v", buffered)
	}
	if writer.DroppedEvents() != 1 {
		t.Fatalf("expected one dropped event, got %d", writer.DroppedEvents())
	}
}
