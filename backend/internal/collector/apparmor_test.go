package collector

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGenerateAppArmorSudoProfileProtectsFilesAndChildren(t *testing.T) {
	profile, err := GenerateAppArmorSudoProfile([]AppArmorPathRule{
		{Path: "/etc/docker/daemon.json"},
		{Path: "/opt/diting/protected"},
	})
	if err != nil {
		t.Fatalf("generate profile: %v", err)
	}

	for _, expected := range []string{
		"profile diting-sudo /{usr/,}bin/sudo",
		`audit deny "/etc/docker/daemon.json" wkl,`,
		`audit deny "/etc/docker/daemon.json/**" wkl,`,
		`audit deny "/opt/diting/protected" wkl,`,
		`audit deny "/opt/diting/protected/**" wkl,`,
		"/** ix,",
	} {
		if !strings.Contains(profile, expected) {
			t.Fatalf("expected profile to contain %q:\n%s", expected, profile)
		}
	}
}

func TestGenerateAppArmorSudoProfileUsesSelectedOperations(t *testing.T) {
	profile, err := GenerateAppArmorSudoProfile([]AppArmorPathRule{{
		Path:       "/etc/docker/daemon.json",
		Operations: []string{"read", "write", "delete"},
	}})
	if err != nil {
		t.Fatalf("generate profile: %v", err)
	}

	for _, expected := range []string{
		`audit deny "/etc/docker/daemon.json" rwkl,`,
		`audit deny "/etc/docker/daemon.json/**" rwkl,`,
	} {
		if !strings.Contains(profile, expected) {
			t.Fatalf("expected profile to contain %q:\n%s", expected, profile)
		}
	}
}

func TestGenerateAppArmorSudoProfileUsesParserSupportedModesForLifecycleOperations(t *testing.T) {
	profile, err := GenerateAppArmorSudoProfile([]AppArmorPathRule{{
		Path:       "/tmp/diting-enforce-test",
		Operations: []string{"create", "delete", "rename"},
	}})
	if err != nil {
		t.Fatalf("generate profile: %v", err)
	}

	for _, forbidden := range []string{" wklc,", " wkld,", " wklcd,"} {
		if strings.Contains(profile, forbidden) {
			t.Fatalf("expected profile to avoid unsupported AppArmor mode %q:\n%s", forbidden, profile)
		}
	}
	if !strings.Contains(profile, `audit deny "/tmp/diting-enforce-test" wkl,`) {
		t.Fatalf("expected lifecycle operations to use parser-supported wkl mode:\n%s", profile)
	}
}

func TestGenerateAppArmorSudoProfileUsesWriteModesForMetadataOperations(t *testing.T) {
	profile, err := GenerateAppArmorSudoProfile([]AppArmorPathRule{{
		Path:       "/tmp/diting-enforce-test/secret.txt",
		Operations: []string{"chmod", "chown"},
	}})
	if err != nil {
		t.Fatalf("generate profile: %v", err)
	}

	if strings.Contains(profile, `audit deny "/tmp/diting-enforce-test/secret.txt" m,`) {
		t.Fatalf("expected metadata operations not to rely on mmap-only mode:\n%s", profile)
	}
	if !strings.Contains(profile, `audit deny "/tmp/diting-enforce-test/secret.txt" wkl,`) {
		t.Fatalf("expected metadata operations to use write-class modes:\n%s", profile)
	}
}

func TestNormalizeAppArmorOperationsDefaultsToWrite(t *testing.T) {
	permissions, err := normalizeAppArmorOperations(nil)
	if err != nil {
		t.Fatalf("normalize operations: %v", err)
	}
	if permissions != "wkl" {
		t.Fatalf("expected legacy default wkl, got %q", permissions)
	}
}

func TestNormalizeAppArmorOperationsUsesStablePermissions(t *testing.T) {
	tests := []struct {
		name       string
		operation  []string
		permission string
	}{
		{name: "read", operation: []string{"read"}, permission: "r"},
		{name: "write", operation: []string{"write"}, permission: "wkl"},
		{name: "create", operation: []string{"create"}, permission: "wkl"},
		{name: "delete", operation: []string{"delete"}, permission: "wkl"},
		{name: "rename", operation: []string{"rename"}, permission: "wkl"},
		{name: "chmod", operation: []string{"chmod"}, permission: "wkl"},
		{name: "chown", operation: []string{"chown"}, permission: "wkl"},
		{name: "all", operation: []string{"all"}, permission: "rwkl"},
		{name: "deduplicated", operation: []string{"delete", "read", "delete"}, permission: "rwkl"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			permission, err := normalizeAppArmorOperations(test.operation)
			if err != nil {
				t.Fatalf("normalize operations: %v", err)
			}
			if permission != test.permission {
				t.Fatalf("expected %q, got %q", test.permission, permission)
			}
		})
	}
}

func TestNormalizeAppArmorOperationsRejectsUnknownNames(t *testing.T) {
	if _, err := normalizeAppArmorOperations([]string{"network"}); err == nil {
		t.Fatal("expected unknown operation to be rejected")
	}
}

func TestNormalizeAppArmorPathRulesMergesDuplicatePathPermissions(t *testing.T) {
	rules, err := normalizeAppArmorPathRules([]AppArmorPathRule{
		{Path: "/etc/docker/daemon.json", Operations: []string{"read"}},
		{Path: "/etc/docker/daemon.json", Operations: []string{"delete"}},
	})
	if err != nil {
		t.Fatalf("normalize path rules: %v", err)
	}
	if len(rules) != 1 || rules[0].Permission != "rwkl" {
		t.Fatalf("expected duplicate path permissions to merge, got %#v", rules)
	}
}

func TestAppArmorKernelEnabledRequiresY(t *testing.T) {
	if !appArmorKernelEnabled([]byte("Y\n")) {
		t.Fatal("expected Y to enable AppArmor")
	}
	if appArmorKernelEnabled([]byte("N\n")) {
		t.Fatal("expected N to disable AppArmor")
	}
}

func TestGenerateAppArmorSudoProfileRejectsUnsafePaths(t *testing.T) {
	tests := []string{
		"etc/passwd",
		"/etc/{passwd,shadow}",
		"/etc/*",
		"/etc/passwd\n#include <tunables/global>",
	}
	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			if _, err := GenerateAppArmorSudoProfile([]AppArmorPathRule{{Path: path}}); err == nil {
				t.Fatalf("expected path %q to be rejected", path)
			}
		})
	}
}

func TestGenerateAppArmorSudoProfileIsDeterministic(t *testing.T) {
	first, err := GenerateAppArmorSudoProfile([]AppArmorPathRule{
		{Path: "/var/lib/diting"},
		{Path: "/etc/docker/daemon.json"},
		{Path: "/var/lib/diting"},
	})
	if err != nil {
		t.Fatalf("generate first profile: %v", err)
	}
	second, err := GenerateAppArmorSudoProfile([]AppArmorPathRule{
		{Path: "/etc/docker/daemon.json"},
		{Path: "/var/lib/diting"},
	})
	if err != nil {
		t.Fatalf("generate second profile: %v", err)
	}
	if first != second {
		t.Fatalf("expected deterministic profile output\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestAppArmorManagerApplyValidatesAndLoadsProfile(t *testing.T) {
	dir := t.TempDir()
	var calls [][]string
	manager := NewAppArmorManager(dir, "apparmor_parser")
	manager.run = func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	}

	changed, err := manager.Apply(context.Background(), "profile-content")
	if err != nil {
		t.Fatalf("apply profile: %v", err)
	}
	if !changed {
		t.Fatal("expected profile to change")
	}
	if len(calls) != 2 || calls[0][1] != "-Q" || calls[1][1] != "-r" {
		t.Fatalf("expected validate then replace calls, got %#v", calls)
	}
	content, err := os.ReadFile(filepath.Join(dir, appArmorProfileName))
	if err != nil {
		t.Fatalf("read active profile: %v", err)
	}
	if string(content) != "profile-content" {
		t.Fatalf("unexpected active profile %q", content)
	}
}

func TestAppArmorManagerApplySkipsUnchangedProfile(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, appArmorProfileName)
	if err := os.WriteFile(profilePath, []byte("same"), 0o600); err != nil {
		t.Fatalf("write existing profile: %v", err)
	}
	manager := NewAppArmorManager(dir, "apparmor_parser")
	manager.run = func(context.Context, string, ...string) error {
		t.Fatal("parser must not run for unchanged profile")
		return nil
	}

	changed, err := manager.Apply(context.Background(), "same")
	if err != nil {
		t.Fatalf("apply unchanged profile: %v", err)
	}
	if changed {
		t.Fatal("expected unchanged profile")
	}
}

func TestAppArmorManagerApplyReloadsPreviousProfileAfterActivationFailure(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, appArmorProfileName)
	if err := os.WriteFile(profilePath, []byte("previous"), 0o600); err != nil {
		t.Fatalf("write previous profile: %v", err)
	}
	var calls [][]string
	manager := NewAppArmorManager(dir, "apparmor_parser")
	manager.run = func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		if len(calls) == 2 {
			return errors.New("activation failed")
		}
		return nil
	}

	if _, err := manager.Apply(context.Background(), "next"); err == nil {
		t.Fatal("expected activation failure")
	}
	if len(calls) != 3 || calls[2][1] != "-r" || calls[2][2] != profilePath {
		t.Fatalf("expected previous profile reload, got %#v", calls)
	}
	content, err := os.ReadFile(profilePath)
	if err != nil {
		t.Fatalf("read previous profile: %v", err)
	}
	if string(content) != "previous" {
		t.Fatalf("expected previous profile preserved, got %q", content)
	}
}

func TestAppArmorManagerRemoveUnloadsManagedProfile(t *testing.T) {
	dir := t.TempDir()
	profilePath := filepath.Join(dir, appArmorProfileName)
	if err := os.WriteFile(profilePath, []byte("active"), 0o600); err != nil {
		t.Fatalf("write active profile: %v", err)
	}
	var calls [][]string
	manager := NewAppArmorManager(dir, "apparmor_parser")
	manager.run = func(_ context.Context, name string, args ...string) error {
		calls = append(calls, append([]string{name}, args...))
		return nil
	}

	changed, err := manager.Remove(context.Background())
	if err != nil {
		t.Fatalf("remove profile: %v", err)
	}
	if !changed {
		t.Fatal("expected profile removal")
	}
	if len(calls) != 1 || calls[0][1] != "-R" || calls[0][2] != profilePath {
		t.Fatalf("expected parser remove call, got %#v", calls)
	}
	if _, err := os.Stat(profilePath); !os.IsNotExist(err) {
		t.Fatalf("expected managed profile removed, got %v", err)
	}
}
