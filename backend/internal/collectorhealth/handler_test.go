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

func TestEnrichHeartbeatAddsHealthSignals(t *testing.T) {
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	lastEvent := now.Add(-90 * time.Second)
	lastWrite := now.Add(-80 * time.Second)

	item := Enrich(Heartbeat{
		HostID:        "server-1",
		LastSeenAt:    now.Add(-30 * time.Second),
		LastEventTime: &lastEvent,
		LastWriteAt:   &lastWrite,
		EventsWritten: 10,
	}, now)

	if item.Status != "online" || item.HealthLevel != "healthy" {
		t.Fatalf("expected healthy online collector, got status=%s health=%s", item.Status, item.HealthLevel)
	}
	if item.EventLagSeconds != 90 || item.WriteLagSeconds != 80 {
		t.Fatalf("expected lag seconds to be calculated, got event=%d write=%d", item.EventLagSeconds, item.WriteLagSeconds)
	}
}

func TestEnrichHeartbeatWarnsWhenNoRecentEvents(t *testing.T) {
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	lastEvent := now.Add(-11 * time.Minute)

	item := Enrich(Heartbeat{
		HostID:        "server-1",
		LastSeenAt:    now.Add(-30 * time.Second),
		LastEventTime: &lastEvent,
	}, now)

	if item.Status != "online" || item.HealthLevel != "warning" {
		t.Fatalf("expected warning online collector, got status=%s health=%s", item.Status, item.HealthLevel)
	}
	if !strings.Contains(item.Message, "长时间未收到事件") {
		t.Fatalf("expected stale event message, got %q", item.Message)
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
