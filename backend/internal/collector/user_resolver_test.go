package collector

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"diting/backend/internal/audit"
)

func TestPasswdUserResolver(t *testing.T) {
	path := filepath.Join(t.TempDir(), "passwd")
	if err := os.WriteFile(path, []byte("root:x:0:0:root:/root:/bin/bash\nalice:x:1000:1000:Alice:/home/alice:/bin/bash\n"), 0o600); err != nil {
		t.Fatalf("write passwd: %v", err)
	}

	resolver, err := NewPasswdUserResolver(path)
	if err != nil {
		t.Fatalf("NewPasswdUserResolver returned error: %v", err)
	}

	if got := resolver.Username(1000); got != "alice" {
		t.Fatalf("expected alice, got %q", got)
	}
}

func TestIdentityWriterEnrichesLinuxUsernames(t *testing.T) {
	sink := &captureWriter{}
	resolver := &PasswdUserResolver{users: map[uint32]string{0: "root", 1000: "alice"}}
	writer := NewIdentityWriter(resolver, sink)

	err := writer.Write(context.Background(), []audit.Event{{UID: 0, AUID: 1000}})
	if err != nil {
		t.Fatalf("Write returned error: %v", err)
	}

	event := sink.events[0]
	if event.Username != "root" {
		t.Fatalf("expected username root, got %q", event.Username)
	}
	if event.LoginUsername != "alice" {
		t.Fatalf("expected login username alice, got %q", event.LoginUsername)
	}
}

type captureWriter struct {
	events []audit.Event
}

func (w *captureWriter) Write(_ context.Context, events []audit.Event) error {
	w.events = append(w.events, events...)
	return nil
}
