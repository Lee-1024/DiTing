package server

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"diting/backend/internal/auth"
	"diting/backend/internal/systemconfig"
)

func TestHealthzReturnsOK(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != `{"status":"ok"}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}

func TestRouterLogsRequests(t *testing.T) {
	var logs bytes.Buffer
	original := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(original) })

	router := NewRouter(nil, nil, nil, nil, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	router.ServeHTTP(rec, req)

	output := logs.String()
	if !strings.Contains(output, "http request") {
		t.Fatalf("expected request log, got %q", output)
	}
	if !strings.Contains(output, "path=/healthz") {
		t.Fatalf("expected request path in log, got %q", output)
	}
}

func TestAuditEventsRouteReturnsListEnvelope(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestProtectedRouteRequiresAuthWhenAuthServiceConfigured(t *testing.T) {
	service := auth.NewService(nil, auth.Config{Secret: "test-secret", ExpiresHours: 1})
	router := NewRouter(nil, nil, nil, service, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestRuleDetailRouteAllowsGet(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, nil, nil, nil, nil)
	createRec := httptest.NewRecorder()
	createBody := bytes.NewBufferString(`{"name":"test","eventType":"process_exec","enabled":true,"severity":"info","riskScore":0,"matchExpr":{"operator":"and","conditions":[]}}`)
	createReq := httptest.NewRequest(http.MethodPost, "/api/v1/rules", createBody)
	router.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("expected create status 201, got %d", createRec.Code)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules/rule-1", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 from rule handler, got %d", rec.Code)
	}
}

func TestCollectorFilterConfigRouteAllowsGetAndPut(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, nil, nil, nil, systemconfig.NewMemoryRepository())

	putRec := httptest.NewRecorder()
	putBody := bytes.NewBufferString(`{"enabled":true,"ignoreProcessNames":["node_exporter"],"ignoreCommandKeywords":["/metrics"],"ignoreUsers":["prometheus"],"keepSeverities":["high","critical"]}`)
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/system-configs/collector-filter", putBody)
	router.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected put status 200, got %d: %s", putRec.Code, putRec.Body.String())
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/system-configs/collector-filter", nil)
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected get status 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	if !strings.Contains(getRec.Body.String(), `"enabled":true`) {
		t.Fatalf("expected saved config, got %s", getRec.Body.String())
	}
}
