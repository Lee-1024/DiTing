package collector

import (
	"context"
	"net/http"
	"net/http/httptest"
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
