package collector

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnforcementSyncerApplyPoliciesWritesAndPrunesManagedFiles(t *testing.T) {
	dir := t.TempDir()
	stale := filepath.Join(dir, "diting-old.yaml")
	if err := os.WriteFile(stale, []byte("old"), 0o644); err != nil {
		t.Fatalf("write stale policy: %v", err)
	}
	other := filepath.Join(dir, "custom-policy.yaml")
	if err := os.WriteFile(other, []byte("keep"), 0o644); err != nil {
		t.Fatalf("write custom policy: %v", err)
	}
	syncer := NewEnforcementSyncer("http://api/api/v1/ingest/events", "", "host-1", "host", dir, "")

	changed, err := syncer.applyPolicies([]EnforcementPolicy{{ID: "p1", Name: "敏感文件", YAML: "kind: TracingPolicy"}})

	if err != nil {
		t.Fatalf("apply policies: %v", err)
	}
	if !changed {
		t.Fatalf("expected changed")
	}
	if _, err := os.Stat(filepath.Join(dir, "diting-policy-p1.yaml")); err != nil {
		t.Fatalf("expected new policy file: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatalf("expected stale managed file removed, got %v", err)
	}
	if _, err := os.Stat(other); err != nil {
		t.Fatalf("expected custom policy kept: %v", err)
	}
}

func TestEnforcementSyncerApplyPoliciesDoesNotChangeSameContent(t *testing.T) {
	dir := t.TempDir()
	syncer := NewEnforcementSyncer("http://api/api/v1/ingest/events", "", "host-1", "host", dir, "")
	policies := []EnforcementPolicy{{ID: "p1", Name: "diting-sensitive", YAML: "kind: TracingPolicy"}}
	if _, err := syncer.applyPolicies(policies); err != nil {
		t.Fatalf("first apply policies: %v", err)
	}

	changed, err := syncer.applyPolicies(policies)

	if err != nil {
		t.Fatalf("second apply policies: %v", err)
	}
	if changed {
		t.Fatalf("expected unchanged policies")
	}
}
