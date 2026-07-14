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

func TestAuditRepositoryQueriesEventsAsJSON(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body := string(data)
		bodies = append(bodies, body)
		if strings.Contains(body, "count() AS total") {
			_, _ = w.Write([]byte(`{"total":"42"}` + "\n"))
			return
		}
		_, _ = w.Write([]byte(`{"event_id":"evt-1","event_time":"2026-07-09 13:00:00.000","event_type":"process_exec","severity":"info","cmdline":"id","tags":[],"rule_ids":["rule-1"],"rule_names":["Download and execute"]}` + "\n"))
	}))
	defer server.Close()

	repository := NewAuditRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}))
	events, total, err := repository.ListEvents(context.Background(), audit.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Page:      1,
		PageSize:  50,
	})
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}

	joinedBodies := strings.Join(bodies, "\n---\n")
	if !strings.Contains(joinedBodies, "FROM diting.audit_events") {
		t.Fatalf("expected table in query, got %s", joinedBodies)
	}
	if !strings.Contains(joinedBodies, "count() AS total") {
		t.Fatalf("expected count query, got %s", joinedBodies)
	}
	if len(events) != 1 || total != 42 {
		t.Fatalf("expected one event, got len=%d total=%d", len(events), total)
	}
	if events[0].EventID != "evt-1" {
		t.Fatalf("expected evt-1, got %q", events[0].EventID)
	}
	if len(events[0].RuleNames) != 1 || events[0].RuleNames[0] != "Download and execute" {
		t.Fatalf("expected rule names to be decoded, got %#v", events[0].RuleNames)
	}
}

func TestAuditRepositoryFiltersCommandDetails(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"event_id":"evt-1","event_time":"2026-07-09 13:00:00.000","event_type":"process_exec","username":"root","login_username":"root","process_name":"whoami","cmdline":"/usr/bin/whoami","tags":[]}` + "\n"))
	}))
	defer server.Close()

	repository := NewAuditRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}))
	_, _, err := repository.ListEvents(context.Background(), audit.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		EventType: "process_exec",
		Username:  "root",
		Cmdline:   "/usr/bin/whoami",
		Page:      1,
		PageSize:  100,
	})
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}

	if !strings.Contains(body, "event_type = 'process_exec'") || !strings.Contains(body, "cmdline = '/usr/bin/whoami'") || !strings.Contains(body, "(username = 'root' OR login_username = 'root')") {
		t.Fatalf("expected command detail filters, got %s", body)
	}
}

func TestAuditRepositoryFiltersMultipleSeverities(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(""))
	}))
	defer server.Close()

	repository := NewAuditRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}))
	_, _, err := repository.ListEvents(context.Background(), audit.Query{
		StartTime:  time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		SeverityIn: []string{"high", "critical"},
		Page:       1,
		PageSize:   50,
	})
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}

	if !strings.Contains(body, "severity IN ('high', 'critical')") {
		t.Fatalf("expected severity IN filter, got %s", body)
	}
}

func TestAuditRepositoryFiltersKeywordInListAndCountQueries(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body := string(data)
		bodies = append(bodies, body)
		if strings.Contains(body, "count() AS total") {
			_, _ = w.Write([]byte(`{"total":"0"}` + "\n"))
			return
		}
		_, _ = w.Write([]byte(""))
	}))
	defer server.Close()

	repository := NewAuditRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}))
	_, _, err := repository.ListEvents(context.Background(), audit.Query{
		StartTime:  time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		EventType:  "process_exec",
		SeverityIn: []string{"high", "critical"},
		Keyword:    "wget",
		Page:       1,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}

	if len(bodies) != 2 {
		t.Fatalf("expected list and count queries, got %d: %v", len(bodies), bodies)
	}
	for _, body := range bodies {
		if !strings.Contains(body, "event_type = 'process_exec'") ||
			!strings.Contains(body, "severity IN ('high', 'critical')") ||
			!strings.Contains(body, "positionCaseInsensitive(cmdline, 'wget')") ||
			!strings.Contains(body, "positionCaseInsensitive(process_name, 'wget')") {
			t.Fatalf("expected keyword filters in query, got %s", body)
		}
	}
}

func TestAuditRepositoryDropsRowsThatDoNotMatchQuery(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		body := string(data)
		if strings.Contains(body, "count() AS total") {
			_, _ = w.Write([]byte(`{"total":"1"}` + "\n"))
			return
		}
		_, _ = w.Write([]byte(`{"event_id":"evt-1","event_time":"2026-07-09 13:00:00.000","event_type":"process_exec","severity":"high","process_name":"docker","cmdline":"/usr/bin/docker stats","tags":[]}` + "\n"))
	}))
	defer server.Close()

	repository := NewAuditRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}))
	events, _, err := repository.ListEvents(context.Background(), audit.Query{
		StartTime:  time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:    time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		EventType:  "process_exec",
		SeverityIn: []string{"critical"},
		Keyword:    "wget",
		Page:       1,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("expected non-matching row to be dropped, got %#v", events)
	}
}

func TestAuditRepositoryFiltersHostName(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(""))
	}))
	defer server.Close()

	repository := NewAuditRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}))
	_, _, err := repository.ListEvents(context.Background(), audit.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		HostName:  "node-1",
		Page:      1,
		PageSize:  50,
	})
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}

	if !strings.Contains(body, "(host_id = 'node-1' OR node_name = 'node-1' OR host_name = 'node-1')") {
		t.Fatalf("expected host filter, got %s", body)
	}
}
