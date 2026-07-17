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
	if !strings.Contains(body, "uniqExact(if(host_name != '', host_name, if(host_id != '', host_id, node_name))) AS active_hosts") {
		t.Fatalf("expected active hosts to use stable host identity, got %s", body)
	}
	if overview.TotalEvents != 10 || overview.ActiveRules != 4 {
		t.Fatalf("unexpected overview %#v", overview)
	}
}

func TestStatsRepositoryEventTrendUsesShanghaiTimezone(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"time":"2026-07-14 11:00:00","count":"5"}` + "\n"))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), nil)
	points, err := repository.EventTrend(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 14, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("EventTrend returned error: %v", err)
	}

	if !strings.Contains(body, "toTimeZone(event_time, 'Asia/Shanghai')") {
		t.Fatalf("expected event trend to use Asia/Shanghai timezone, got %s", body)
	}
	if len(points) != 1 || points[0].Time != "2026-07-14 11:00:00" || points[0].Count != 5 {
		t.Fatalf("unexpected trend points %#v", points)
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

func TestStatsRepositoryTopHostsUsesStableHostIdentity(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"name":"server-1","count":"9"}` + "\n"))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), nil)
	items, err := repository.TopHosts(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("TopHosts returned error: %v", err)
	}

	if !strings.Contains(body, "if(host_name != '', host_name, if(host_id != '', host_id, node_name)) AS name") || !strings.Contains(body, "name != ''") {
		t.Fatalf("expected stable host identity in query, got %s", body)
	}
	if len(items) != 1 || items[0].Name != "server-1" || items[0].Count != 9 {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestStatsRepositoryTopNamespacesFiltersEmptyNamespace(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"name":"default","count":"7"}` + "\n"))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), nil)
	items, err := repository.TopNamespaces(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Limit:     5,
	})
	if err != nil {
		t.Fatalf("TopNamespaces returned error: %v", err)
	}

	if !strings.Contains(body, "namespace AS name") || !strings.Contains(body, "namespace != ''") {
		t.Fatalf("expected namespace query to skip empty values, got %s", body)
	}
	if len(items) != 1 || items[0].Name != "default" || items[0].Count != 7 {
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
		HostName:  "host-001",
	})
	if err != nil {
		t.Fatalf("CommandStats returned error: %v", err)
	}

	if !strings.Contains(body, "event_type = 'process_exec'") || !strings.Contains(body, "positionCaseInsensitive(cmdline, 'whoami')") || !strings.Contains(body, "username = 'root'") {
		t.Fatalf("expected command filters in query, got %s", body)
	}
	if !strings.Contains(body, "(host_id = 'host-001' OR node_name = 'host-001' OR host_name = 'host-001')") {
		t.Fatalf("expected command host filter, got %s", body)
	}
	if !strings.Contains(body, "event_time >= parseDateTime64BestEffort('2026-07-09 00:00:00.000', 3)") || !strings.Contains(body, "event_time <= parseDateTime64BestEffort('2026-07-10 00:00:00.000', 3)") {
		t.Fatalf("expected command stats to apply time range, got %s", body)
	}
	if strings.Contains(body, "toTimeZone(min(event_time)") || strings.Contains(body, "toTimeZone(max(event_time)") {
		t.Fatalf("expected command first/last seen to keep the same raw event_time timezone as audit details, got %s", body)
	}
	if !strings.Contains(body, "min(event_time) AS first_seen") || !strings.Contains(body, "max(event_time) AS last_seen") {
		t.Fatalf("expected command first/last seen to use raw event_time aggregates, got %s", body)
	}
	if len(items) != 1 || items[0].ProcessName != "whoami" || items[0].Username != "root" {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestStatsRepositoryCommandStatsIncludesEventsWithoutProcessName(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"process_name":"","cmdline":"id","username":"ubuntu","login_username":"ubuntu","latest_host_id":"host-001","latest_host_name":"prod-web-01","latest_node_name":"node-1","host_count":"1","command_count":"1","first_seen":"2026-07-15 09:10:00","last_seen":"2026-07-15 09:10:00"}` + "\n"))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), nil)
	items, err := repository.CommandStats(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 16, 0, 0, 0, 0, time.UTC),
		Limit:     50,
	})
	if err != nil {
		t.Fatalf("CommandStats returned error: %v", err)
	}

	if strings.Contains(body, "process_name != ''") {
		t.Fatalf("expected command stats to include rows with empty process_name, got %s", body)
	}
	if !strings.Contains(body, "cmdline != ''") {
		t.Fatalf("expected command stats to require cmdline, got %s", body)
	}
	if !strings.Contains(body, "count() AS command_count") {
		t.Fatalf("expected command stats to avoid count alias conflicts, got %s", body)
	}
	for _, expected := range []string{
		"argMax(host_id, event_time) AS latest_host_id",
		"argMax(host_name, event_time) AS latest_host_name",
		"argMax(node_name, event_time) AS latest_node_name",
	} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected latest host field %q, got %s", expected, body)
		}
	}
	if !strings.Contains(body, "max(event_time) AS last_seen_sort") || !strings.Contains(body, "ORDER BY last_seen_sort DESC, command_count DESC") {
		t.Fatalf("expected command stats to order by newest execution first, got %s", body)
	}
	if len(items) != 1 || items[0].Cmdline != "id" || items[0].Username != "ubuntu" || items[0].HostID != "host-001" || items[0].HostCount != 1 {
		t.Fatalf("unexpected command stats %#v", items)
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

func TestStatsRepositoryUserAuditsFiltersHostName(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(""))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), nil)
	_, err := repository.UserAudits(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		HostName:  "host-001",
	})
	if err != nil {
		t.Fatalf("UserAudits returned error: %v", err)
	}

	if !strings.Contains(body, "(host_id = 'host-001' OR node_name = 'host-001' OR host_name = 'host-001')") {
		t.Fatalf("expected host filter, got %s", body)
	}
}

func TestStatsRepositoryHostAuditsAggregatesHosts(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"host_id":"host-001","host_name":"prod-web-01","node_name":"node-1","command_count":"12","active_users":"2","high_risk_events":"3","first_seen":"2026-07-10 02:13:38.363","last_seen":"2026-07-10 02:14:25.564"}` + "\n"))
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

	if !strings.Contains(body, "event_type = 'process_exec'") || !strings.Contains(body, "if(host_id != '', host_id, if(node_name != '', node_name, host_name)) AS audit_host_key") || !strings.Contains(body, "positionCaseInsensitive(audit_host, 'node')") {
		t.Fatalf("expected host audit query filters, got %s", body)
	}
	if len(items) != 1 || items[0].HostID != "host-001" || items[0].HostName != "prod-web-01" || items[0].NodeName != "node-1" || items[0].CommandCount != 12 {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestStatsRepositoryHostUsersAggregatesUsersForHost(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"username":"ubuntu","command_count":"6","high_risk_events":"1","first_seen":"2026-07-10 02:13:38.363","last_seen":"2026-07-10 02:14:25.564"}` + "\n"))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), nil)
	items, err := repository.HostUsers(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		HostName:  "host-001",
		Limit:     20,
	})
	if err != nil {
		t.Fatalf("HostUsers returned error: %v", err)
	}

	if !strings.Contains(body, "event_type = 'process_exec'") ||
		!strings.Contains(body, "(host_id = 'host-001' OR node_name = 'host-001' OR host_name = 'host-001')") ||
		!strings.Contains(body, "if(login_username != '', login_username, username) AS audit_user") {
		t.Fatalf("expected host user query filters, got %s", body)
	}
	if len(items) != 1 || items[0].Username != "ubuntu" || items[0].CommandCount != 6 || items[0].HighRiskEvents != 1 {
		t.Fatalf("unexpected items %#v", items)
	}
}

func TestStatsRepositoryRuleHitsAggregatesRules(t *testing.T) {
	var body string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(data)
		body = string(data)
		_, _ = w.Write([]byte(`{"rule_name":"反弹 Shell 命令","hit_count":"5","active_hosts":"2","active_users":"1","first_seen":"2026-07-10 02:13:38.363","last_seen":"2026-07-10 02:14:25.564"}` + "\n"))
	}))
	defer server.Close()

	repository := NewStatsRepository(NewHTTPClient(HTTPConfig{URL: server.URL, Database: "diting"}), nil)
	items, err := repository.RuleHits(context.Background(), stats.Query{
		StartTime: time.Date(2026, 7, 9, 0, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
		Keyword:   "反弹",
		Limit:     20,
	})
	if err != nil {
		t.Fatalf("RuleHits returned error: %v", err)
	}

	if !strings.Contains(body, "arrayJoin(rule_names) AS rule_name") ||
		!strings.Contains(body, "positionCaseInsensitive(rule_name, '反弹')") ||
		!strings.Contains(body, "uniqExact(audit_host) AS active_hosts") {
		t.Fatalf("expected rule hit query, got %s", body)
	}
	if len(items) != 1 || items[0].RuleName != "反弹 Shell 命令" || items[0].HitCount != 5 || items[0].ActiveHosts != 2 {
		t.Fatalf("unexpected items %#v", items)
	}
}

type fakeRuleCounter struct {
	count uint64
}

func (f fakeRuleCounter) CountEnabledRules(context.Context) (uint64, error) {
	return f.count, nil
}
