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

func TestMemoryRepositoryKeepsLastErrorUntilCleared(t *testing.T) {
	repository := NewMemoryRepository()
	now := time.Date(2026, 7, 16, 10, 0, 0, 0, time.UTC)
	if err := repository.Upsert(context.Background(), HeartbeatUpdate{HostID: "server-1", LastSeenAt: now, LastError: "grpc unavailable"}); err != nil {
		t.Fatalf("Upsert error heartbeat: %v", err)
	}
	if err := repository.Upsert(context.Background(), HeartbeatUpdate{HostID: "server-1", LastSeenAt: now.Add(time.Second)}); err != nil {
		t.Fatalf("Upsert heartbeat: %v", err)
	}
	items, err := repository.List(context.Background(), now.Add(2*time.Second))
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if items[0].LastError != "grpc unavailable" {
		t.Fatalf("expected heartbeat to keep last error, got %q", items[0].LastError)
	}
	if err := repository.Upsert(context.Background(), HeartbeatUpdate{HostID: "server-1", LastSeenAt: now.Add(3 * time.Second), ClearError: true}); err != nil {
		t.Fatalf("Upsert clear heartbeat: %v", err)
	}
	items, err = repository.List(context.Background(), now.Add(4*time.Second))
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if items[0].LastError != "" {
		t.Fatalf("expected successful write to clear last error, got %q", items[0].LastError)
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

func TestHandlerReportsAuthorizedHeartbeat(t *testing.T) {
	repository := NewMemoryRepository()
	handler := NewHandlerWithToken(repository, "secret-token")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/heartbeat", strings.NewReader(`{"hostId":"server-1","hostName":"server-1","inputMode":"grpc","eventsWritten":5,"bufferedEvents":2,"droppedEvents":1}`))
	req.Header.Set("Authorization", "Bearer secret-token")

	handler.Report(rec, req)

	if rec.Code != http.StatusAccepted {
		t.Fatalf("expected 202, got %d body=%s", rec.Code, rec.Body.String())
	}
	items, err := repository.List(context.Background(), time.Now().UTC())
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(items) != 1 || items[0].HostID != "server-1" || items[0].InputMode != "grpc" || items[0].EventsWritten != 5 || items[0].BufferedEvents != 2 || items[0].DroppedEvents != 1 {
		t.Fatalf("unexpected heartbeat items: %#v", items)
	}
}

func TestHandlerRejectsHeartbeatWithoutToken(t *testing.T) {
	handler := NewHandlerWithToken(NewMemoryRepository(), "secret-token")
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ingest/heartbeat", strings.NewReader(`{"hostId":"server-1"}`))

	handler.Report(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}
