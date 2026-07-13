package server

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"diting/backend/internal/auth"
)

func TestHealthzReturnsOK(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, nil, nil, nil)
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

func TestAuditEventsRouteReturnsListEnvelope(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/audit/events", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
}

func TestProtectedRouteRequiresAuthWhenAuthServiceConfigured(t *testing.T) {
	service := auth.NewService(nil, auth.Config{Secret: "test-secret", ExpiresHours: 1})
	router := NewRouter(nil, nil, nil, service, nil, nil, nil)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d", rec.Code)
	}
}

func TestRuleDetailRouteAllowsGet(t *testing.T) {
	router := NewRouter(nil, nil, nil, nil, nil, nil, nil)
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
