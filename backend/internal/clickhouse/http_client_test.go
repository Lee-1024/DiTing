package clickhouse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"diting/backend/internal/audit"
)

func TestHTTPClientWritesJSONEachRow(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(bodyBytes)
		body = string(bodyBytes)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"})
	err := client.WriteEvents(context.Background(), []audit.Event{{
		EventID: "evt-1", EventTime: time.Unix(1, 0).UTC(), EventDate: time.Unix(0, 0).UTC(), IngestTime: time.Unix(2, 0).UTC(),
		EventType: "process_exec", Action: "exec", Severity: "info", HostID: "machine-1", HostName: "app-01", ProcessName: "bash", Cmdline: "bash -c id",
	}})
	if err != nil {
		t.Fatalf("WriteEvents returned error: %v", err)
	}

	if !strings.Contains(body, "INSERT INTO diting.audit_events FORMAT JSONEachRow") {
		t.Fatalf("expected insert statement in body, got %s", body)
	}
	if !strings.Contains(body, `"event_id":"evt-1"`) {
		t.Fatalf("expected event json in body, got %s", body)
	}
	if !strings.Contains(body, `"event_time":"1970-01-01 00:00:01.000"`) {
		t.Fatalf("expected ClickHouse datetime string, got %s", body)
	}
	if !strings.Contains(body, `"host_id":"machine-1"`) {
		t.Fatalf("expected host id in event json, got %s", body)
	}
	if strings.Contains(body, `"tags":null`) {
		t.Fatalf("expected empty tags array instead of null, got %s", body)
	}
}

func TestHTTPURLFromNativeAddressUsesHTTPPort(t *testing.T) {
	got := HTTPURLFromAddress("10.40.0.184:9002")
	if got != "http://10.40.0.184:8123" {
		t.Fatalf("expected http://10.40.0.184:8123, got %q", got)
	}
}
