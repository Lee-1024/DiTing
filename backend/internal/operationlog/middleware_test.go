package operationlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"diting/backend/internal/auth"
)

type fakeRepository struct {
	entries []Entry
}

func (f *fakeRepository) Create(_ context.Context, entry Entry) error {
	f.entries = append(f.entries, entry)
	return nil
}

func (f *fakeRepository) List(_ context.Context, query Query) ([]Entry, int, error) {
	return f.entries, len(f.entries), nil
}

func TestMiddlewareRecordsAuthenticatedOperation(t *testing.T) {
	repository := &fakeRepository{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	claims := auth.Claims{UserID: "user-1", Username: "admin"}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/rules", nil)
	req = req.WithContext(auth.ContextWithClaims(req.Context(), claims))
	rec := httptest.NewRecorder()

	Middleware(repository)(next).ServeHTTP(rec, req)

	if len(repository.entries) != 1 {
		t.Fatalf("expected one operation log, got %d", len(repository.entries))
	}
	entry := repository.entries[0]
	if entry.Username != "admin" || entry.Method != http.MethodPost || entry.Path != "/api/v1/rules" || entry.Status != http.StatusCreated {
		t.Fatalf("unexpected entry %#v", entry)
	}
}

func TestMiddlewareRecordsForwardedClientIP(t *testing.T) {
	repository := &fakeRepository{}
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	claims := auth.Claims{UserID: "user-1", Username: "admin"}
	req := httptest.NewRequest(http.MethodPut, "/api/v1/settings", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 127.0.0.1")
	req = req.WithContext(auth.ContextWithClaims(req.Context(), claims))
	rec := httptest.NewRecorder()

	Middleware(repository)(next).ServeHTTP(rec, req)

	if len(repository.entries) != 1 {
		t.Fatalf("expected one operation log, got %d", len(repository.entries))
	}
	if repository.entries[0].IP != "203.0.113.10" {
		t.Fatalf("expected forwarded client IP, got %q", repository.entries[0].IP)
	}
}

func TestClientIPFallsBackToRealIPAndRemoteAddr(t *testing.T) {
	realIPReq := httptest.NewRequest(http.MethodGet, "/", nil)
	realIPReq.Header.Set("X-Real-IP", "198.51.100.20")
	if got := clientIP(realIPReq); got != "198.51.100.20" {
		t.Fatalf("expected X-Real-IP, got %q", got)
	}

	remoteReq := httptest.NewRequest(http.MethodGet, "/", nil)
	remoteReq.RemoteAddr = "192.0.2.30:53120"
	if got := clientIP(remoteReq); got != "192.0.2.30" {
		t.Fatalf("expected remote host without port, got %q", got)
	}
}
