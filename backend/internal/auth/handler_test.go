package auth

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestLoginHandlerReturnsToken(t *testing.T) {
	service := NewService(&fakeUserRepository{user: User{
		ID: "user-1", Username: "admin", DisplayName: "Administrator", PasswordHash: HashPassword("admin123", "fixed-salt"), Status: "active",
	}}, Config{Secret: "test-secret", ExpiresHours: 1})
	handler := NewHandler(service)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"username":"admin","password":"admin123"}`))

	handler.Login(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"token"`) {
		t.Fatalf("expected token response, got %s", rec.Body.String())
	}
}

func TestMiddlewareRejectsMissingToken(t *testing.T) {
	service := NewService(&fakeUserRepository{}, Config{Secret: "test-secret", ExpiresHours: 1})
	middleware := Middleware(service)
	nextCalled := false
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		nextCalled = true
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/rules", nil)

	middleware(next).ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
	if nextCalled {
		t.Fatal("expected next handler not to be called")
	}
}
