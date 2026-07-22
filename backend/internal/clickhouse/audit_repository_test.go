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
		_, _ = w.Write([]byte(`{"event_id":"evt-1","event_time":"2026-07-09 13:00:00.000","event_type":"process_exec","severity":"info","cmdline":"id","tags":[],"rule_ids":["rule-1"],"rule_names":["Download and execute"],"rule_matches":"[{\"ruleId\":\"rule-1\",\"ruleName\":\"Download and execute\",\"field\":\"cmdline\",\"operator\":\"contains\",\"value\":\"id\",\"actual\":\"id\"}]"}` + "\n"))
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
	if strings.Contains(joinedBodies, "LIMIT 1 BY") {
		t.Fatalf("expected list query to avoid expensive ClickHouse LIMIT BY, got %s", joinedBodies)
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
	if len(events[0].RuleMatches) != 1 || events[0].RuleMatches[0].Field != "cmdline" || events[0].RuleMatches[0].Value != "id" {
		t.Fatalf("expected rule matches to be decoded, got %#v", events[0].RuleMatches)
	}
}

func TestCollapseDuplicateListEvents(t *testing.T) {
	eventTime := time.Date(2026, 7, 22, 14, 34, 6, 0, time.UTC)
	events := []audit.Event{
		{EventID: "evt-1", EventTime: eventTime, EventType: "file_access", Action: "sys_unlink", HostID: "host-1", LoginUsername: "ubuntu", Username: "ubuntu", ProcessName: "rm", Cmdline: "/usr/bin/rm test", FilePath: "test"},
		{EventID: "evt-2", EventTime: eventTime.Add(100 * time.Millisecond), EventType: "file_access", Action: "sys_unlink", HostID: "host-1", LoginUsername: "ubuntu", Username: "ubuntu", ProcessName: "rm", Cmdline: "/usr/bin/rm test", FilePath: "test"},
		{EventID: "evt-3", EventTime: eventTime, EventType: "process_exit", Action: "exit", HostID: "host-1", LoginUsername: "ubuntu", Username: "ubuntu", ProcessName: "rm", Cmdline: "/usr/bin/rm test"},
	}

	collapsed := collapseDuplicateListEvents(events)

	if len(collapsed) != 2 {
		t.Fatalf("expected duplicate file events to collapse while keeping process exit, got %#v", collapsed)
	}
	if collapsed[0].EventID != "evt-1" || collapsed[1].EventID != "evt-3" {
		t.Fatalf("unexpected collapsed events %#v", collapsed)
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

func TestAuditRepositoryGetsEventByID(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"event_id":"evt-1","event_time":"2026-07-09 13:00:00.000","event_type":"process_exec","username":"root","process_name":"id","cmdline":"/usr/bin/id","tags":[],"rule_ids":[],"rule_names":[],"raw_event":"{\"process_exec\":{}}"}` + "\n"))
	}))
	defer server.Close()

	repository := NewAuditRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}))
	event, err := repository.GetEvent(context.Background(), "evt-1")
	if err != nil {
		t.Fatalf("GetEvent returned error: %v", err)
	}

	if !strings.Contains(body, "FROM diting.audit_events") || !strings.Contains(body, "WHERE event_id = 'evt-1'") || !strings.Contains(body, "LIMIT 1") {
		t.Fatalf("expected event detail query by id, got %s", body)
	}
	if event.EventID != "evt-1" || event.Cmdline != "/usr/bin/id" {
		t.Fatalf("unexpected event detail %#v", event)
	}
}

func TestAuditRepositoryReturnsNotFoundWhenEventMissing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(""))
	}))
	defer server.Close()

	repository := NewAuditRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}))
	_, err := repository.GetEvent(context.Background(), "missing")
	if err != audit.ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
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
			!strings.Contains(body, "positionCaseInsensitive(process_name, 'wget')") ||
			!strings.Contains(body, "positionCaseInsensitive(file_path, 'wget')") ||
			!strings.Contains(body, "positionCaseInsensitive(dst_ip, 'wget')") {
			t.Fatalf("expected keyword filters in query, got %s", body)
		}
	}
}

func TestAuditRepositoryReturnsFileAndNetworkFields(t *testing.T) {
	var bodies []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body := string(data)
		bodies = append(bodies, body)
		if strings.Contains(body, "count() AS total") {
			_, _ = w.Write([]byte(`{"total":"1"}` + "\n"))
			return
		}
		_, _ = w.Write([]byte(`{"event_id":"evt-file","event_time":"2026-07-09 13:00:00.000","event_type":"file_access","severity":"info","process_name":"cat","cmdline":"cat /etc/passwd","file_path":"/etc/passwd","file_operation":"open","src_ip":"10.0.0.1","src_port":42000,"dst_ip":"93.184.216.34","dst_port":443,"protocol":"tcp","domain":"example.com","tags":[]}` + "\n"))
	}))
	defer server.Close()

	repository := NewAuditRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}))
	events, _, err := repository.ListEvents(context.Background(), audit.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Keyword:   "/etc/passwd",
		Page:      1,
		PageSize:  10,
	})
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}

	joinedBodies := strings.Join(bodies, "\n---\n")
	for _, expected := range []string{"file_path", "file_operation", "dst_ip", "dst_port", "protocol", "domain"} {
		if !strings.Contains(joinedBodies, expected) {
			t.Fatalf("expected %q to be selected, got %s", expected, joinedBodies)
		}
	}
	if len(events) != 1 {
		t.Fatalf("expected one event, got %#v", events)
	}
	event := events[0]
	if event.FilePath != "/etc/passwd" || event.FileOperation != "open" || event.DstIP != "93.184.216.34" || event.DstPort != 443 || event.Protocol != "tcp" || event.Domain != "example.com" {
		t.Fatalf("expected file and network fields to be decoded, got %#v", event)
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

func TestAuditRepositoryFiltersNamespacePodAndSeparateUsers(t *testing.T) {
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
		StartTime:     time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:       time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Namespace:     "default",
		PodName:       "api-0",
		LoginUsername: "ubuntu",
		ExecUsername:  "root",
		Page:          1,
		PageSize:      50,
	})
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}

	for _, expected := range []string{
		"namespace = 'default'",
		"pod_name = 'api-0'",
		"login_username = 'ubuntu'",
		"username = 'root'",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in query, got %s", expected, body)
		}
	}
}

func TestAuditRepositoryFiltersNetworkTarget(t *testing.T) {
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
		StartTime: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC),
		EventType: "network_connect",
		HostName:  "host-001",
		DstIP:     "110.242.68.4",
		DstPort:   443,
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}

	for _, expected := range []string{
		"event_type = 'network_connect'",
		"(host_id = 'host-001' OR node_name = 'host-001' OR host_name = 'host-001')",
		"dst_ip = '110.242.68.4'",
		"dst_port = 443",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in query, got %s", expected, body)
		}
	}
}

func TestAuditRepositoryFiltersFilePath(t *testing.T) {
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
		StartTime: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 21, 0, 0, 0, 0, time.UTC),
		EventType: "file_access",
		HostName:  "host-001",
		FilePath:  "/etc/passwd",
		Page:      1,
		PageSize:  20,
	})
	if err != nil {
		t.Fatalf("ListEvents returned error: %v", err)
	}

	for _, expected := range []string{
		"event_type = 'file_access'",
		"(host_id = 'host-001' OR node_name = 'host-001' OR host_name = 'host-001')",
		"file_path = '/etc/passwd'",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %q in query, got %s", expected, body)
		}
	}
}
