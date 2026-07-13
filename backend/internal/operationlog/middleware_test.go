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
