package collectorhealth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStatusMarksOfflineAfterTwoMinutes(t *testing.T) {
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	if Status(now.Add(-time.Minute), now) != "online" {
		t.Fatal("expected recent heartbeat to be online")
	}
	if Status(now.Add(-3*time.Minute), now) != "offline" {
		t.Fatal("expected stale heartbeat to be offline")
	}
}

func TestHandlerListsCollectorHealth(t *testing.T) {
	repository := NewMemoryRepository()
	_ = repository.Upsert(context.Background(), HeartbeatUpdate{HostID: "server-1", HostName: "server-1", LastSeenAt: time.Now().UTC(), EventsWritten: 3})
	handler := NewHandler(repository)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/collectors/health", nil)

	handler.List(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"hostId":"server-1"`) || !strings.Contains(rec.Body.String(), `"eventsWritten":3`) {
		t.Fatalf("unexpected body %s", rec.Body.String())
	}
}
