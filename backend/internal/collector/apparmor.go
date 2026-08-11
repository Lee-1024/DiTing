package collector

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const appArmorProfileName = "diting-sudo"
const appArmorEnabledPath = "/sys/module/apparmor/parameters/enabled"

type appArmorCommandRunner func(context.Context, string, ...string) error

type AppArmorManager struct {
	policyDir string
	parser    string
	run       appArmorCommandRunner
}

type AppArmorPathRule struct {
	Path       string
	Operations []string
}

type normalizedAppArmorPathRule struct {
	Path       string
	Permission string
}

func NewAppArmorManager(policyDir string, parser string) *AppArmorManager {
	return &AppArmorManager{
		policyDir: policyDir,
		parser:    parser,
		run:       runAppArmorCommand,
	}
}

func discoverAppArmorManager(policyDir string) (*AppArmorManager, error) {
	enabled, err := os.ReadFile(appArmorEnabledPath)
	if err != nil {
		return nil, fmt.Errorf("read AppArmor kernel status: %w", err)
	}
	if !appArmorKernelEnabled(enabled) {
		return nil, fmt.Errorf("AppArmor is disabled in the kernel")
	}
	parser, err := exec.LookPath("apparmor_parser")
	if err != nil {
		return nil, fmt.Errorf("apparmor_parser command not found")
	}
	return NewAppArmorManager(policyDir, parser), nil
}

func appArmorKernelEnabled(value []byte) bool {
	return strings.EqualFold(strings.TrimSpace(string(value)), "Y")
}

func (m *AppArmorManager) Apply(ctx context.Context, profile string) (bool, error) {
	if err := os.MkdirAll(m.policyDir, 0o700); err != nil {
		return false, fmt.Errorf("create AppArmor policy directory: %w", err)
	}
	profilePath := filepath.Join(m.policyDir, appArmorProfileName)
	previous, readErr := os.ReadFile(profilePath)
	if readErr == nil && bytes.Equal(previous, []byte(profile)) {
		return false, nil
	}
	if readErr != nil && !os.IsNotExist(readErr) {
		return false, fmt.Errorf("read active AppArmor profile: %w", readErr)
	}

	nextPath := profilePath + ".next"
	if err := os.WriteFile(nextPath, []byte(profile), 0o600); err != nil {
		return false, fmt.Errorf("write candidate AppArmor profile: %w", err)
	}
	defer os.Remove(nextPath)

	if err := m.run(ctx, m.parser, "-Q", nextPath); err != nil {
		return false, fmt.Errorf("validate AppArmor profile: %w", err)
	}
	if err := m.run(ctx, m.parser, "-r", nextPath); err != nil {
		if readErr == nil {
			_ = m.run(ctx, m.parser, "-r", profilePath)
		}
		return false, fmt.Errorf("activate AppArmor profile: %w", err)
	}
	if err := replaceAppArmorProfile(nextPath, profilePath); err != nil {
		if readErr == nil {
			_ = m.run(ctx, m.parser, "-r", profilePath)
		}
		return false, fmt.Errorf("promote AppArmor profile: %w", err)
	}
	return true, nil
}

func (m *AppArmorManager) Remove(ctx context.Context) (bool, error) {
	profilePath := filepath.Join(m.policyDir, appArmorProfileName)
	if _, err := os.Stat(profilePath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("stat managed AppArmor profile: %w", err)
	}
	if err := m.run(ctx, m.parser, "-R", profilePath); err != nil {
		return false, fmt.Errorf("unload AppArmor profile: %w", err)
	}
	if err := os.Remove(profilePath); err != nil {
		return false, fmt.Errorf("remove managed AppArmor profile: %w", err)
	}
	return true, nil
}

func replaceAppArmorProfile(nextPath string, profilePath string) error {
	if err := os.Rename(nextPath, profilePath); err == nil {
		return nil
	}
	if err := os.Remove(profilePath); err != nil && !os.IsNotExist(err) {
		return err
	}
	return os.Rename(nextPath, profilePath)
}

func runAppArmorCommand(ctx context.Context, name string, args ...string) error {
	output, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("%w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

// GenerateAppArmorSudoProfile builds the single AppArmor profile managed by DiTing.
func GenerateAppArmorSudoProfile(rules []AppArmorPathRule) (string, error) {
	normalized, err := normalizeAppArmorPathRules(rules)
	if err != nil {
		return "", err
	}

	var profile strings.Builder
	profile.WriteString(`#include <tunables/global>

profile diting-sudo /{usr/,}bin/sudo flags=(attach_disconnected,mediate_deleted) {
  #include <abstractions/base>

  file,
  capability,
  network,
  mount,
  remount,
  umount,
  pivot_root,
  signal,
  ptrace,
  dbus,
  unix,

  /** ix,
`)
	for _, rule := range normalized {
		fmt.Fprintf(&profile, "\n  audit deny %q %s,\n", rule.Path, rule.Permission)
		fmt.Fprintf(&profile, "  audit deny %q %s,\n", strings.TrimSuffix(rule.Path, "/")+"/**", rule.Permission)
	}
	profile.WriteString("}\n")
	return profile.String(), nil
}

func normalizeAppArmorPathRules(rules []AppArmorPathRule) ([]normalizedAppArmorPathRule, error) {
	unique := make(map[string]normalizedAppArmorPathRule, len(rules))
	for _, rule := range rules {
		path, err := normalizeAppArmorPath(rule.Path)
		if err != nil {
			return nil, err
		}
		permission, err := normalizeAppArmorOperations(rule.Operations)
		if err != nil {
			return nil, err
		}
		if existing, ok := unique[path]; ok {
			permission = mergeAppArmorPermissions(existing.Permission, permission)
		}
		unique[path] = normalizedAppArmorPathRule{Path: path, Permission: permission}
	}
	if len(unique) == 0 {
		return nil, fmt.Errorf("at least one protected path is required")
	}

	result := make([]normalizedAppArmorPathRule, 0, len(unique))
	for _, rule := range unique {
		result = append(result, rule)
	}
	sort.Slice(result, func(left, right int) bool {
		return result[left].Path < result[right].Path
	})
	return result, nil
}

func mergeAppArmorPermissions(values ...string) string {
	permissions := make(map[rune]struct{})
	for _, value := range values {
		for _, permission := range value {
			permissions[permission] = struct{}{}
		}
	}
	var result strings.Builder
	for _, permission := range "rwkldcm" {
		if _, ok := permissions[permission]; ok {
			result.WriteRune(permission)
		}
	}
	return result.String()
}

func normalizeAppArmorPaths(paths []string) ([]string, error) {
	pathRules := make([]AppArmorPathRule, 0, len(paths))
	for _, path := range paths {
		pathRules = append(pathRules, AppArmorPathRule{Path: path})
	}
	rules, err := normalizeAppArmorPathRules(pathRules)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(rules))
	for _, rule := range rules {
		result = append(result, rule.Path)
	}
	return result, nil
}

func normalizeAppArmorPath(raw string) (string, error) {
	path := strings.TrimSpace(raw)
	if path == "" || !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("AppArmor protected path must be absolute: %q", raw)
	}
	if strings.ContainsAny(path, "\x00\r\n*?[]{}@^\"\\") {
		return "", fmt.Errorf("AppArmor protected path contains unsupported characters: %q", raw)
	}
	path = cleanAppArmorPath(path)
	if path == "/" {
		return "", fmt.Errorf("refusing to protect filesystem root")
	}
	return path, nil
}

func normalizeAppArmorOperations(operations []string) (string, error) {
	if len(operations) == 0 {
		operations = []string{"write"}
	}

	permissions := make(map[rune]struct{})
	for _, raw := range operations {
		switch strings.TrimSpace(strings.ToLower(raw)) {
		case "read":
			permissions['r'] = struct{}{}
		case "write":
			permissions['w'] = struct{}{}
			permissions['k'] = struct{}{}
			permissions['l'] = struct{}{}
		case "create":
			permissions['c'] = struct{}{}
			permissions['w'] = struct{}{}
			permissions['k'] = struct{}{}
			permissions['l'] = struct{}{}
		case "delete":
			permissions['d'] = struct{}{}
		case "rename":
			permissions['d'] = struct{}{}
			permissions['w'] = struct{}{}
			permissions['k'] = struct{}{}
			permissions['l'] = struct{}{}
		case "chmod", "chown":
			permissions['m'] = struct{}{}
		case "all":
			for _, permission := range "rwkldcm" {
				permissions[permission] = struct{}{}
			}
		default:
			return "", fmt.Errorf("unsupported AppArmor operation: %q", raw)
		}
	}

	var result strings.Builder
	for _, permission := range "rwkldcm" {
		if _, ok := permissions[permission]; ok {
			result.WriteRune(permission)
		}
	}
	return result.String(), nil
}

func cleanAppArmorPath(value string) string {
	return path.Clean(value)
}
