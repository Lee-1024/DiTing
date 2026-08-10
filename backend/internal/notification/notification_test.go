package notification

import (
	"context"
	"testing"

	"diting/backend/internal/audit"
)

func TestMemoryRepositoryNotificationLifecycle(t *testing.T) {
	repo := NewMemoryRepository()
	event := audit.Event{EventID: "event-1", LoginUsername: "alice", ProcessName: "vim", FilePath: "/etc/docker/daemon.json", Severity: "critical", Tags: []string{"diting-enforcement"}}
	first, err := repo.Upsert(context.Background(), EnforcementInput(event))
	if err != nil {
		t.Fatal(err)
	}
	second, err := repo.Upsert(context.Background(), EnforcementInput(event))
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID {
		t.Fatal("duplicate event created another notification")
	}
	result, err := repo.List(context.Background(), "user-1", "unread", 20)
	if err != nil {
		t.Fatal(err)
	}
	if result.Counts.Unread != 1 || result.Counts.Pending != 1 {
		t.Fatalf("unexpected counts: %#v", result.Counts)
	}
	if err := repo.MarkRead(context.Background(), "user-1", first.ID); err != nil {
		t.Fatal(err)
	}
	userOne, _ := repo.List(context.Background(), "user-1", "unread", 20)
	other, _ := repo.List(context.Background(), "user-2", "unread", 20)
	if len(userOne.Items) != 0 || len(other.Items) != 1 {
		t.Fatal("read state must be per user")
	}
	if err := repo.Handle(context.Background(), first.ID, "confirmed", "admin"); err != nil {
		t.Fatal(err)
	}
	pending, _ := repo.List(context.Background(), "user-1", "pending", 20)
	if len(pending.Items) != 0 {
		t.Fatal("handled notification remained pending")
	}
}

func TestEnforcementNotificationDoesNotReopenAfterHandling(t *testing.T) {
	repo := NewMemoryRepository()
	input := EnforcementInput(audit.Event{EventID: "event-1", Tags: []string{"diting-enforcement"}})
	item, err := repo.Upsert(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Handle(context.Background(), item.ID, "false_positive", "admin"); err != nil {
		t.Fatal(err)
	}
	again, err := repo.Upsert(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if again.ID != item.ID || again.Status != StatusResolved || again.Disposition != "false_positive" {
		t.Fatalf("handled enforcement notification reopened: %#v", again)
	}
}

func TestCollectorNotificationRecurrencePreservesResolvedHistory(t *testing.T) {
	repo := NewMemoryRepository()
	input := Input{Type: TypeCollector, DedupeKey: "collector:offline:host-1", SourceID: "host-1", Title: "采集节点离线"}
	first, err := repo.Upsert(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.MarkRead(context.Background(), "user-1", first.ID); err != nil {
		t.Fatal(err)
	}
	if err := repo.Resolve(context.Background(), input.DedupeKey); err != nil {
		t.Fatal(err)
	}
	second, err := repo.Upsert(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	if second.ID == first.ID {
		t.Fatal("recurrent state alert must create a new history record")
	}
	all, err := repo.List(context.Background(), "user-1", "all", 20)
	if err != nil {
		t.Fatal(err)
	}
	if len(all.Items) != 2 || all.Counts.Unread != 1 {
		t.Fatalf("unexpected recurrence history: %#v", all)
	}
	var resolvedFound bool
	for _, item := range all.Items {
		if item.ID == first.ID && item.Status == StatusResolved && item.ResolvedAt != nil {
			resolvedFound = true
		}
	}
	if !resolvedFound {
		t.Fatal("resolved incident history was not preserved")
	}
}
