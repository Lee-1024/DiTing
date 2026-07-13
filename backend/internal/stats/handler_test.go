package stats

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeRepository struct{}

func (fakeRepository) Overview(context.Context, Query) (Overview, error) {
	return Overview{TotalEvents: 10, HighRiskEvents: 2, ActiveHosts: 3, ActiveRules: 4}, nil
}

func (fakeRepository) EventTrend(context.Context, Query) ([]TrendPoint, error) {
	return []TrendPoint{{Time: "2026-07-09 10:00:00", Count: 5}}, nil
}

func (fakeRepository) TopCommands(context.Context, Query) ([]TopItem, error) {
	return []TopItem{{Name: "bash", Count: 6}}, nil
}

func (fakeRepository) CommandStats(_ context.Context, query Query) ([]CommandItem, error) {
	return []CommandItem{{ProcessName: "whoami", Cmdline: "/usr/bin/whoami", Username: query.Username, Count: 4}}, nil
}

func (fakeRepository) UserAudits(_ context.Context, query Query) ([]UserAuditItem, error) {
	return []UserAuditItem{{Username: query.Keyword, CommandCount: 8, ActiveHosts: 1}}, nil
}

func (fakeRepository) HostAudits(_ context.Context, query Query) ([]HostAuditItem, error) {
	return []HostAuditItem{{HostName: query.Keyword, CommandCount: 12, ActiveUsers: 2}}, nil
}

func TestOverviewHandlerReturnsOverview(t *testing.T) {
	handler := NewHandler(fakeRepository{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/overview", nil)

	handler.Overview(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"totalEvents":10`) {
		t.Fatalf("unexpected body %s", rec.Body.String())
	}
}

func TestTopCommandsHandlerReturnsItems(t *testing.T) {
	handler := NewHandler(fakeRepository{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/top-commands?limit=5", nil)

	handler.TopCommands(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"name":"bash"`) {
		t.Fatalf("unexpected body %s", rec.Body.String())
	}
}

func TestCommandStatsHandlerPassesFilters(t *testing.T) {
	handler := NewHandler(fakeRepository{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/commands?limit=50&keyword=whoami&username=root", nil)

	handler.CommandStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"processName":"whoami"`) || !strings.Contains(body, `"username":"root"`) {
		t.Fatalf("unexpected body %s", body)
	}
}

func TestExportCommandStatsHandlerReturnsCSV(t *testing.T) {
	handler := NewHandler(fakeRepository{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/commands/export?keyword=whoami", nil)

	handler.ExportCommandStats(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Header().Get("Content-Type") != "text/csv; charset=utf-8" {
		t.Fatalf("unexpected content type %q", rec.Header().Get("Content-Type"))
	}
	body := rec.Body.String()
	if !strings.Contains(body, "process_name,cmdline,login_username,username,count,first_seen,last_seen") || !strings.Contains(body, "whoami") {
		t.Fatalf("unexpected body %s", body)
	}
}

func TestUserAuditsHandlerPassesFilters(t *testing.T) {
	handler := NewHandler(fakeRepository{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/users?limit=20&keyword=root", nil)

	handler.UserAudits(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"username":"root"`) || !strings.Contains(body, `"commandCount":8`) {
		t.Fatalf("unexpected body %s", body)
	}
}

func TestHostAuditsHandlerPassesFilters(t *testing.T) {
	handler := NewHandler(fakeRepository{})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/hosts?limit=20&keyword=node-1", nil)

	handler.HostAudits(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, `"hostName":"node-1"`) || !strings.Contains(body, `"commandCount":12`) {
		t.Fatalf("unexpected body %s", body)
	}
}
