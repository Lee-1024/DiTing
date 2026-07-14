package collector

import (
	"context"
	"testing"

	"diting/backend/internal/audit"
)

func TestHostMetadataWriterAppliesStableHostIdentity(t *testing.T) {
	next := &captureWriter{}
	writer := NewHostMetadataWriter(HostMetadata{ID: "machine-1", Name: "app-01"}, next)

	if err := writer.Write(context.Background(), []audit.Event{{NodeName: "tetragon-container"}}); err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	if len(next.events) != 1 {
		t.Fatalf("expected one event, got %d", len(next.events))
	}
	if next.events[0].HostID != "machine-1" {
		t.Fatalf("expected host id machine-1, got %q", next.events[0].HostID)
	}
	if next.events[0].HostName != "app-01" {
		t.Fatalf("expected host name app-01, got %q", next.events[0].HostName)
	}
	if next.events[0].NodeName != "tetragon-container" {
		t.Fatalf("expected original node name to be preserved, got %q", next.events[0].NodeName)
	}
}
