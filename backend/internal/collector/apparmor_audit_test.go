package collector

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestParseAppArmorDenialCreatesEnforcementEvent(t *testing.T) {
	line := `type=AVC msg=audit(1786060201.123:456): apparmor="DENIED" operation="open" profile="diting-sudo" name="/etc/docker/daemon.json" pid=321 comm="vim" requested_mask="w" denied_mask="w" fsuid=0 ouid=0 auid=1000`

	event, err := ParseAppArmorAuditEvent(line)
	if err != nil {
		t.Fatalf("parse AppArmor denial: %v", err)
	}
	if event.EventType != "file_access" || event.Action != "open" {
		t.Fatalf("unexpected event type/action: %#v", event)
	}
	if event.FilePath != "/etc/docker/daemon.json" || event.FileOperation != "w" {
		t.Fatalf("unexpected file context: %#v", event)
	}
	if event.ProcessName != "vim" || event.PID != 321 {
		t.Fatalf("unexpected process context: %#v", event)
	}
	if event.UID != 0 || event.AUID != 1000 {
		t.Fatalf("expected sudo uid and login auid, got uid=%d auid=%d", event.UID, event.AUID)
	}
	if !containsAuditTag(event.Tags, "diting-enforcement") {
		t.Fatalf("expected enforcement tag, got %#v", event.Tags)
	}
}

func TestParseAppArmorAuditEventIgnoresOtherProfiles(t *testing.T) {
	line := `type=AVC msg=audit(1786060201.123:456): apparmor="DENIED" operation="open" profile="snap.foo" name="/tmp/file" pid=321 comm="foo"`

	if _, err := ParseAppArmorAuditEvent(line); err != ErrUnsupportedEvent {
		t.Fatalf("expected unsupported event, got %v", err)
	}
}

func TestAppArmorAuditCollectorWritesOnlyManagedDenials(t *testing.T) {
	path := filepath.Join(t.TempDir(), "audit.log")
	content := "type=AVC apparmor=\"DENIED\" operation=\"open\" profile=\"snap.foo\" name=\"/tmp/file\" pid=1 comm=\"foo\"\n" +
		"type=AVC msg=audit(1786060201.123:456): apparmor=\"DENIED\" operation=\"open\" profile=\"diting-sudo\" name=\"/etc/docker/daemon.json\" pid=321 comm=\"vim\" denied_mask=\"w\" fsuid=0 auid=1000\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write audit log: %v", err)
	}
	writer := &recordingWriter{}
	collector := NewAppArmorAuditCollector(path, 10, writer)

	if err := collector.RunOnce(context.Background()); err != nil {
		t.Fatalf("collect AppArmor audit log: %v", err)
	}
	if len(writer.batches) != 1 || len(writer.batches[0]) != 1 || writer.batches[0][0].ProcessName != "vim" {
		t.Fatalf("expected one managed denial, got %#v", writer.batches)
	}
}

func containsAuditTag(tags []string, expected string) bool {
	for _, tag := range tags {
		if tag == expected {
			return true
		}
	}
	return false
}
