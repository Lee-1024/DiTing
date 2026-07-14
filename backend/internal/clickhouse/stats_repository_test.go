package clickhouse

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"diting/backend/internal/stats"
)

func TestStatsRepositoryOverviewQueriesClickHouse(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"total_events":"10","high_risk_events":"2","active_hosts":"3"}` + "\n"))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), fakeRuleCounter{count: 4})
	overview, err := repository.Overview(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("Overview returned error: %v", err)
	}

	if !strings.Contains(body, "count() AS total_events") {
		t.Fatalf("expected overview query, got %s", body)
	}
	if overview.TotalEvents != 10 || overview.ActiveRules != 4 {
		t.Fatalf("unexpected overview %#v", overview)
	}
}

func TestStatsRepositoryTopCommandsOnlyCountsProcessExec(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"name":"whoami","count":"4"}` + "\n"))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), nil)
	items, err := repository.TopCommands(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Limit:     50,
	})
	if err != nil {
		t.Fatalf("TopCommands returned error: %v", err)
	}

	if !strings.Contains(body, "event_type = 'process_exec'") {
		t.Fatalf("expected process_exec filter in query, got %s", body)
	}
	if len(items) != 1 || items[0].Name != "whoami" {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestStatsRepositoryCommandStatsFiltersByKeywordAndUser(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"process_name":"whoami","cmdline":"/usr/bin/whoami","username":"root","login_username":"root","count":"4","first_seen":"2026-07-10 02:13:38.363","last_seen":"2026-07-10 02:13:38.365"}` + "\n"))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), nil)
	items, err := repository.CommandStats(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Limit:     50,
		Keyword:   "whoami",
		Username:  "root",
	})
	if err != nil {
		t.Fatalf("CommandStats returned error: %v", err)
	}

	if !strings.Contains(body, "event_type = 'process_exec'") || !strings.Contains(body, "positionCaseInsensitive(cmdline, 'whoami')") || !strings.Contains(body, "username = 'root'") {
		t.Fatalf("expected command filters in query, got %s", body)
	}
	if len(items) != 1 || items[0].ProcessName != "whoami" || items[0].Username != "root" {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestStatsRepositoryUserAuditsAggregatesLinuxUsers(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"username":"root","command_count":"8","active_hosts":"1","high_risk_events":"2","first_seen":"2026-07-10 02:13:38.363","last_seen":"2026-07-10 02:14:25.564"}` + "\n"))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), nil)
	items, err := repository.UserAudits(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Limit:     20,
		Keyword:   "root",
	})
	if err != nil {
		t.Fatalf("UserAudits returned error: %v", err)
	}

	if !strings.Contains(body, "event_type = 'process_exec'") || !strings.Contains(body, "if(login_username != '', login_username, username) AS audit_user") || !strings.Contains(body, "positionCaseInsensitive(audit_user, 'root')") {
		t.Fatalf("expected user audit query filters, got %s", body)
	}
	if len(items) != 1 || items[0].Username != "root" || items[0].CommandCount != 8 {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestStatsRepositoryHostAuditsAggregatesHosts(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"host_name":"node-1","command_count":"12","active_users":"2","high_risk_events":"3","first_seen":"2026-07-10 02:13:38.363","last_seen":"2026-07-10 02:14:25.564"}` + "\n"))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), nil)
	items, err := repository.HostAudits(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Limit:     20,
		Keyword:   "node",
	})
	if err != nil {
		t.Fatalf("HostAudits returned error: %v", err)
	}

	if !strings.Contains(body, "event_type = 'process_exec'") || !strings.Contains(body, "if(host_name != '', host_name, if(host_id != '', host_id, node_name)) AS audit_host") || !strings.Contains(body, "positionCaseInsensitive(audit_host, 'node')") {
		t.Fatalf("expected host audit query filters, got %s", body)
	}
	if len(items) != 1 || items[0].HostName != "node-1" || items[0].CommandCount != 12 {
		t.Fatalf("unexpected items %#v", items)
	}
}

type fakeRuleCounter struct {
	count uint64
}

func (f fakeRuleCounter) CountEnabledRules(context.Context) (uint64, error) {
	return f.count, nil
}
