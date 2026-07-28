package audit

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestParseQueryDefaultsToLast24Hours(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events", nil)

	query, err := ParseQuery(req)
	if err != nil {
		t.Fatalf("ParseQuery returned error: %v", err)
	}

	if query.Page != 1 {
		t.Fatalf("expected page 1, got %d", query.Page)
	}
	if query.PageSize != 10 {
		t.Fatalf("expected page size 10, got %d", query.PageSize)
	}
	if query.EndTime.Sub(query.StartTime).Hours() < 23.9 {
		t.Fatalf("expected default range close to 24 hours, got %s", query.EndTime.Sub(query.StartTime))
	}
}

func TestParseQueryCapsPageSizeAt500(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?page_size=999", nil)

	query, err := ParseQuery(req)
	if err != nil {
		t.Fatalf("ParseQuery returned error: %v", err)
	}

	if query.PageSize != 100 {
		t.Fatalf("expected page size capped at 100, got %d", query.PageSize)
	}
}

func TestParseQueryRejectsDeepPagination(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?page=101&page_size=100", nil)

	_, err := ParseQuery(req)
	if err == nil {
		t.Fatal("expected deep pagination to be rejected")
	}
	if !strings.Contains(err.Error(), "deep pagination") {
		t.Fatalf("expected deep pagination error, got %v", err)
	}
}

func TestParseQueryRejectsBroadKeywordSearch(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?start_time=2026-07-01T00:00:00Z&end_time=2026-07-10T00:00:00Z&keyword=wget", nil)

	_, err := ParseQuery(req)
	if err == nil {
		t.Fatal("expected broad keyword search to be rejected")
	}
	if !strings.Contains(err.Error(), "keyword") {
		t.Fatalf("expected keyword governance error, got %v", err)
	}
}

func TestParseQueryAllowsKeywordSearchWithHostFilter(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?start_time=2026-07-01T00:00:00Z&end_time=2026-07-10T00:00:00Z&keyword=wget&host_name=host-001", nil)

	query, err := ParseQuery(req)
	if err != nil {
		t.Fatalf("expected host-scoped keyword search to pass, got %v", err)
	}
	if query.Keyword != "wget" || query.HostName != "host-001" {
		t.Fatalf("unexpected query %#v", query)
	}
}

func TestParseQueryReadsExtendedAuditFilters(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?namespace=default&pod_name=api-0&login_username=ubuntu&exec_username=root&include_total=true", nil)

	query, err := ParseQuery(req)
	if err != nil {
		t.Fatalf("ParseQuery returned error: %v", err)
	}

	if query.Namespace != "default" || query.PodName != "api-0" || query.LoginUsername != "ubuntu" || query.ExecUsername != "root" || !query.IncludeTotal {
		t.Fatalf("unexpected filters: %#v", query)
	}
}

func TestParseQueryRejectsInvalidTime(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?start_time=bad-time", nil)

	_, err := ParseQuery(req)
	if err == nil {
		t.Fatal("expected invalid time error")
	}
	if !strings.Contains(err.Error(), "start_time") {
		t.Fatalf("expected start_time error, got %v", err)
	}
}

func TestParseQueryRejectsOversizedTimeRange(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?start_time=2026-01-01T00:00:00Z&end_time=2026-03-15T00:00:00Z", nil)

	_, err := ParseQuery(req)
	if err == nil {
		t.Fatal("expected oversized time range error")
	}
	if !strings.Contains(err.Error(), "time range") {
		t.Fatalf("expected time range error, got %v", err)
	}
}

type fakeRepository struct {
	events []Event
	groups []OperationGroup
	query  Query
}

func (f *fakeRepository) ListEvents(_ context.Context, query Query) ([]Event, int, bool, error) {
	f.query = query
	return f.events, len(f.events), len(f.events) > query.PageSize, nil
}

func (f *fakeRepository) GetEvent(_ context.Context, eventID string) (Event, error) {
	for _, event := range f.events {
		if event.EventID == eventID {
			return event, nil
		}
	}
	return Event{}, ErrNotFound
}

func (f *fakeRepository) ListOperations(_ context.Context, query Query) ([]OperationGroup, int, bool, error) {
	f.query = query
	return f.groups, len(f.groups), false, nil
}

func TestHandlerReturnsRepositoryEvents(t *testing.T) {
	repository := &fakeRepository{events: []Event{{EventID: "evt-1", EventType: "process_exec"}}}
	handler := NewHandler(repository)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events?page_size=10", nil)
	rec := httptest.NewRecorder()

	handler.ListEvents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"eventId":"evt-1"`) {
		t.Fatalf("expected event in response, got %s", rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"hasMore":false`) {
		t.Fatalf("expected hasMore in response, got %s", rec.Body.String())
	}
	if repository.query.PageSize != 10 {
		t.Fatalf("expected page size 10, got %d", repository.query.PageSize)
	}
}

func TestHandlerReturnsEventDetailByID(t *testing.T) {
	repository := &fakeRepository{events: []Event{{EventID: "evt-1", EventType: "process_exec", Cmdline: "id"}}}
	handler := NewHandler(repository)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events/evt-1", nil)
	req.SetPathValue("event_id", "evt-1")
	rec := httptest.NewRecorder()

	handler.GetEvent(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"eventId":"evt-1"`) || !strings.Contains(rec.Body.String(), `"cmdline":"id"`) {
		t.Fatalf("expected event detail in response, got %s", rec.Body.String())
	}
}

func TestHandlerReturnsOperationGroups(t *testing.T) {
	repository := &fakeRepository{groups: []OperationGroup{{
		GroupID:        "grp-1",
		Representative: Event{EventID: "evt-1", EventType: "process_exec", Cmdline: "id"},
		EventCount:     3,
		EventTypes:     []string{"process_exec", "file_access"},
		MaxSeverity:    "high",
	}}}
	handler := NewHandler(repository)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/operations?page_size=10&include_total=true", nil)
	rec := httptest.NewRecorder()

	handler.ListOperations(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	for _, expected := range []string{`"groupId":"grp-1"`, `"eventCount":3`, `"maxSeverity":"high"`, `"hasMore":false`} {
		if !strings.Contains(body, expected) {
			t.Fatalf("expected %s in response, got %s", expected, body)
		}
	}
	if !repository.query.IncludeTotal {
		t.Fatalf("expected include_total to be parsed")
	}
}

func TestHandlerReturnsNotFoundForMissingEventDetail(t *testing.T) {
	handler := NewHandler(&fakeRepository{})
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events/missing", nil)
	req.SetPathValue("event_id", "missing")
	rec := httptest.NewRecorder()

	handler.GetEvent(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

func TestExportEventsHandlerReturnsCSV(t *testing.T) {
	repository := &fakeRepository{events: []Event{{
		EventID: "evt-1", Severity: "high", LoginUsername: "root", Username: "root", NodeName: "node-1", ProcessName: "bash", Cmdline: "bash -i", RuleNames: []string{"Reverse shell"},
	}}}
	handler := NewHandler(repository)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events/export", nil)
	rec := httptest.NewRecorder()

	handler.ExportEvents(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Fatalf("unexpected content type %q", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, "event_time,severity,login_username,username,host,process,cmdline,rule_names,event_id") || !strings.Contains(body, "Reverse shell") {
		t.Fatalf("unexpected body %s", body)
	}
	if repository.query.PageSize != 5000 {
		t.Fatalf("expected export page size 5000, got %d", repository.query.PageSize)
	}
}
