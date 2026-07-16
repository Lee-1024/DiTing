package operationlog

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerListsOperationLogs(t *testing.T) {
	repository := &fakeRepository{entries: []Entry{{ID: "log-1", Username: "admin", Method: http.MethodPost, Path: "/api/v1/rules", Status: 201}}}
	handler := NewHandler(repository)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/operation-logs?page_size=20&username=admin", nil)

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"username":"admin"`) || !strings.Contains(rec.Body.String(), `"total":1`) {
		t.Fatalf("unexpected body %s", rec.Body.String())
	}
}

func TestParseQueryRejectsInvalidOperationLogTime(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/v1/operation-logs?start_time=bad", nil)

	_, err := ParseQuery(req)

	if err == nil {
		t.Fatal("expected invalid start_time error")
	}
}

func TestFakeRepositorySatisfiesRepository(t *testing.T) {
	var _ Repository = (*fakeRepository)(nil)
	_, _, _ = (&fakeRepository{}).List(context.Background(), Query{})
}
