package collector

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBuildAppArmorDeploymentUsesSensitiveFileDefinitions(t *testing.T) {
	policies := []EnforcementPolicy{
		{
			ID:         "policy-1",
			Name:       "docker config",
			Template:   "sensitive_file",
			Mode:       "enforce",
			Enabled:    true,
			Definition: json.RawMessage("{\"filePaths\":[\"/etc/docker/daemon.json\"],\"userMatchMode\":\"exclude_root\"}"),
		},
		{
			ID:         "policy-2",
			Name:       "diting data",
			Template:   "sensitive_file",
			Mode:       "enforce",
			Enabled:    true,
			Definition: json.RawMessage("{\"filePaths\":[\"/var/lib/diting\"],\"userMatchMode\":\"exclude_root\"}"),
		},
	}

	profile, _, results := buildAppArmorDeployment(policies)

	if !strings.Contains(profile, "audit deny \"/etc/docker/daemon.json\" wkl,") {
		t.Fatalf("expected docker path in profile:\n%s", profile)
	}
	if !strings.Contains(profile, "audit deny \"/var/lib/diting\" wkl,") {
		t.Fatalf("expected diting path in profile:\n%s", profile)
	}
	for _, id := range []string{"policy-1", "policy-2"} {
		if results[id].Status != "deployed" {
			t.Fatalf("expected %s deployable, got %#v", id, results[id])
		}
	}
}

func TestBuildAppArmorDeploymentUsesSensitiveFileOperations(t *testing.T) {
	policy := EnforcementPolicy{
		ID:         "policy-1",
		Template:   "sensitive_file",
		Mode:       "enforce",
		Enabled:    true,
		Definition: json.RawMessage(`{"filePaths":["/etc/docker/daemon.json"],"operations":["read","delete"]}`),
	}

	profile, _, results := buildAppArmorDeployment([]EnforcementPolicy{policy})

	if results["policy-1"].Status != "deployed" {
		t.Fatalf("expected deployed result, got %#v", results["policy-1"])
	}
	if !strings.Contains(profile, `audit deny "/etc/docker/daemon.json" rd,`) {
		t.Fatalf("expected read/delete permissions in profile:\n%s", profile)
	}
	if !strings.Contains(results["policy-1"].Message, "read, delete") {
		t.Fatalf("expected deployment message to include operations, got %#v", results["policy-1"])
	}
}

func TestBuildAppArmorDeploymentDefaultsLegacySensitiveFileToWrite(t *testing.T) {
	policy := EnforcementPolicy{
		ID:         "policy-1",
		Template:   "sensitive_file",
		Mode:       "enforce",
		Enabled:    true,
		Definition: json.RawMessage(`{"filePaths":["/etc/docker/daemon.json"]}`),
	}

	profile, _, results := buildAppArmorDeployment([]EnforcementPolicy{policy})

	if results["policy-1"].Status != "deployed" || !strings.Contains(profile, `wkl`) {
		t.Fatalf("expected legacy write protection, result=%#v profile=%s", results["policy-1"], profile)
	}
}

func TestBuildAppArmorDeploymentRejectsInvalidSensitiveFileOperation(t *testing.T) {
	policy := EnforcementPolicy{
		ID:         "policy-1",
		Template:   "sensitive_file",
		Mode:       "enforce",
		Enabled:    true,
		Definition: json.RawMessage(`{"filePaths":["/etc/docker/daemon.json"],"operations":["network"]}`),
	}

	profile, _, results := buildAppArmorDeployment([]EnforcementPolicy{policy})

	if results["policy-1"].Status != "failed" {
		t.Fatalf("expected failed result, got %#v", results["policy-1"])
	}
	if !strings.HasPrefix(results["policy-1"].Message, "策略 operations 无效: ") {
		t.Fatalf("expected invalid operations message, got %#v", results["policy-1"])
	}
	if strings.Contains(profile, "/etc/docker/daemon.json") {
		t.Fatalf("expected invalid policy path not to be deployed:\n%s", profile)
	}
}

func TestBuildAppArmorDeploymentRejectsUnsupportedTemplates(t *testing.T) {
	policies := []EnforcementPolicy{{
		ID:         "policy-1",
		Name:       "legacy command",
		Template:   "dangerous_command",
		Mode:       "enforce",
		Enabled:    true,
		Definition: json.RawMessage("{\"commands\":[\"reboot\"]}"),
	}}

	profile, _, results := buildAppArmorDeployment(policies)

	if profile != "" {
		t.Fatalf("expected no profile for unsupported policy, got %q", profile)
	}
	if results["policy-1"].Status != "failed" || !strings.Contains(results["policy-1"].Message, "不支持") {
		t.Fatalf("expected explicit unsupported result, got %#v", results["policy-1"])
	}
}

func TestBuildAppArmorDeploymentRejectsUnsafePolicyWithoutDroppingValidPolicy(t *testing.T) {
	policies := []EnforcementPolicy{
		{
			ID:         "valid",
			Template:   "sensitive_file",
			Mode:       "enforce",
			Enabled:    true,
			Definition: json.RawMessage("{\"filePaths\":[\"/etc/docker/daemon.json\"]}"),
		},
		{
			ID:         "invalid",
			Template:   "sensitive_file",
			Mode:       "enforce",
			Enabled:    true,
			Definition: json.RawMessage("{\"filePaths\":[\"/etc/*\"]}"),
		},
	}

	profile, _, results := buildAppArmorDeployment(policies)

	if !strings.Contains(profile, "/etc/docker/daemon.json") {
		t.Fatalf("expected valid policy to remain deployable:\n%s", profile)
	}
	if results["valid"].Status != "deployed" {
		t.Fatalf("expected valid result, got %#v", results["valid"])
	}
	if results["invalid"].Status != "failed" {
		t.Fatalf("expected invalid result, got %#v", results["invalid"])
	}
}
