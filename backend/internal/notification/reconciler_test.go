package notification

import (
	"context"
	"testing"
	"time"

	"diting/backend/internal/collectorhealth"
)

func TestReconcileHealthOpensAndResolvesCollectorOfflineNotification(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	health := collectorhealth.NewMemoryRepository()
	if err := health.Upsert(ctx, collectorhealth.HeartbeatUpdate{HostID: "host-1", HostName: "node-1", LastSeenAt: now.Add(-3 * time.Minute)}); err != nil {
		t.Fatal(err)
	}
	notifications := NewMemoryRepository()

	if err := ReconcileHealth(ctx, notifications, health, now); err != nil {
		t.Fatal(err)
	}
	opened, _ := notifications.List(ctx, "user-1", "all", 20)
	if len(opened.Items) != 1 || opened.Items[0].Type != TypeCollector || opened.Items[0].Status != StatusOpen {
		t.Fatalf("offline notification not opened: %#v", opened)
	}

	lastEvent := now.Add(time.Minute)
	lastWrite := now.Add(time.Minute)
	if err := health.Upsert(ctx, collectorhealth.HeartbeatUpdate{HostID: "host-1", HostName: "node-1", LastSeenAt: now.Add(time.Minute), LastEventTime: &lastEvent, LastWriteAt: &lastWrite, ClearError: true}); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileHealth(ctx, notifications, health, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	resolved, _ := notifications.List(ctx, "user-1", "all", 20)
	if resolved.Items[0].Status != StatusResolved || resolved.Items[0].ResolvedAt == nil {
		t.Fatalf("offline notification not resolved: %#v", resolved)
	}
}

func TestReconcileHealthTracksTetragonFailureSeparately(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	health := collectorhealth.NewMemoryRepository()
	lastEvent := now
	lastWrite := now
	if err := health.Upsert(ctx, collectorhealth.HeartbeatUpdate{HostID: "host-1", HostName: "node-1", LastSeenAt: now, LastEventTime: &lastEvent, LastWriteAt: &lastWrite, LastError: "Tetragon gRPC unavailable"}); err != nil {
		t.Fatal(err)
	}
	notifications := NewMemoryRepository()

	if err := ReconcileHealth(ctx, notifications, health, now); err != nil {
		t.Fatal(err)
	}
	result, _ := notifications.List(ctx, "user-1", "all", 20)
	if len(result.Items) != 1 || result.Items[0].Type != TypeTetragon {
		t.Fatalf("tetragon notification not opened: %#v", result)
	}
}

func TestReconcileHealthTracksOnlineCollectorRPCFailureAndRecovery(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 8, 10, 10, 0, 0, 0, time.UTC)
	health := collectorhealth.NewMemoryRepository()
	lastEvent := now
	lastWrite := now
	if err := health.Upsert(ctx, collectorhealth.HeartbeatUpdate{
		HostID: "server-002", HostName: "diting-test-113", LastSeenAt: now,
		LastEventTime: &lastEvent, LastWriteAt: &lastWrite,
		LastError: "rpc error: code = Unavailable desc = connection error",
	}); err != nil {
		t.Fatal(err)
	}
	notifications := NewMemoryRepository()

	if err := ReconcileHealth(ctx, notifications, health, now); err != nil {
		t.Fatal(err)
	}
	opened, _ := notifications.List(ctx, "user-1", "all", 20)
	if len(opened.Items) != 1 || opened.Items[0].Type != TypeCollector || opened.Items[0].Status != StatusOpen {
		t.Fatalf("online collector RPC warning not opened: %#v", opened)
	}

	if err := health.Upsert(ctx, collectorhealth.HeartbeatUpdate{
		HostID: "server-002", HostName: "diting-test-113", LastSeenAt: now.Add(time.Minute),
		LastEventTime: &lastEvent, LastWriteAt: &lastWrite, ClearError: true,
	}); err != nil {
		t.Fatal(err)
	}
	if err := ReconcileHealth(ctx, notifications, health, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	resolved, _ := notifications.List(ctx, "user-1", "all", 20)
	if resolved.Items[0].Status != StatusResolved || resolved.Items[0].ResolvedAt == nil {
		t.Fatalf("collector RPC warning not resolved after recovery: %#v", resolved)
	}
}
