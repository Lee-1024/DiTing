# File Lifecycle Enforcement Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extend the current sensitive-file AppArmor enforcement from write/edit protection to configurable read, write, create, delete, rename, chmod, chown, and all-operation protection.

**Architecture:** Keep AppArmor as the enforcement authority for Ubuntu 20.04/Linux 5.4 compatibility, and keep Tetragon as the observer/evidence source. The policy `definition` remains JSON in the existing enforcement policy table; Collector parses `filePaths` plus a new `operations` list and generates deterministic AppArmor rules. The frontend exposes operation selection only for `sensitive_file` policies and stores it in the existing `definition`.

**Tech Stack:** Go 1.26, AppArmor profile generation, existing Collector enforcement syncer, React 18, TypeScript, Ant Design, Vitest.

---

## File Map

- `backend/internal/collector/apparmor.go`: add operation-aware AppArmor profile generation while preserving path validation and deterministic output.
- `backend/internal/collector/apparmor_test.go`: add regression coverage for read/write/delete/metadata operation rule output and validation.
- `backend/internal/collector/enforcement_sync.go`: parse `definition.operations`, normalize defaults, and pass operation-aware protected paths to the profile generator.
- `backend/internal/collector/enforcement_sync_test.go`: verify deployment accepts operation lists, defaults old policies to write protection, and rejects unsupported operation names.
- `backend/internal/collector/tetragon_observer.go`: keep observer policy path-based; no operation-specific observer in this phase unless tests prove existing observer misses AppArmor denial evidence.
- `frontend/src/pages/settings/tetragonPolicy.ts`: add operation types to `PolicyFormValues`, defaults, and preview output.
- `frontend/src/pages/settings/tetragonPolicy.test.ts`: cover operation preview/default behavior.
- `frontend/src/pages/settings/TetragonPolicyPage.tsx`: add operation multi-select for sensitive file policies.
- `docs/superpowers/specs/2026-08-11-enforcement-hardening-design.md`: keep as parent spec; do not rewrite during implementation unless a real design correction is discovered.

## Operation Semantics

Use these stable operation names in `definition.operations`:

- `read`: deny AppArmor read permission `r`.
- `write`: deny write/link/lock permissions `wkl`.
- `create`: deny create/write/link/lock permissions `cwkl`.
- `delete`: deny delete permission `d`.
- `rename`: deny rename-related delete/write/link permissions `dwkl`.
- `chmod`: deny metadata write permission via `m` only if parser validation confirms it is accepted in this profile; otherwise mark the policy `failed` with a clear message.
- `chown`: use the same metadata capability as `chmod`; do not pretend owner-only semantics are available if AppArmor cannot distinguish it.
- `all`: expands to every supported file permission used by this phase.

Implementation note: AppArmor file permissions are coarse. If `chmod` and `chown` cannot be separated safely, represent them as metadata protection in deployment messages and tests instead of claiming exact syscall separation.

---

### Task 1: Operation Normalization and Profile Generation

**Files:**
- Modify: `backend/internal/collector/apparmor.go`
- Modify: `backend/internal/collector/apparmor_test.go`

- [ ] **Step 1: Add failing tests for operation-aware profile output**

Add tests that call the desired API before implementing it:

```go
func TestGenerateAppArmorSudoProfileUsesSelectedOperations(t *testing.T) {
	profile, err := GenerateAppArmorSudoProfile([]AppArmorPathRule{{
		Path:       "/etc/docker/daemon.json",
		Operations: []string{"read", "write", "delete"},
	}})
	if err != nil {
		t.Fatalf("generate profile: %v", err)
	}
	for _, expected := range []string{
		`audit deny "/etc/docker/daemon.json" rwkld,`,
		`audit deny "/etc/docker/daemon.json/**" rwkld,`,
	} {
		if !strings.Contains(profile, expected) {
			t.Fatalf("expected profile to contain %q:\n%s", expected, profile)
		}
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

func TestNormalizeAppArmorOperationsRejectsUnknownNames(t *testing.T) {
	if _, err := normalizeAppArmorOperations([]string{"network"}); err == nil {
		t.Fatal("expected unknown operation to be rejected")
	}
}
```

- [ ] **Step 2: Run test to verify RED**

Run: `go test ./internal/collector -run 'TestGenerateAppArmorSudoProfileUsesSelectedOperations|TestNormalizeAppArmorOperations' -count=1`

Expected: build fails because `AppArmorPathRule` and `normalizeAppArmorOperations` do not exist.

- [ ] **Step 3: Implement operation-aware generation**

Change `GenerateAppArmorSudoProfile` to accept `[]AppArmorPathRule` and add a legacy adapter if needed:

```go
type AppArmorPathRule struct {
	Path       string
	Operations []string
}

func GenerateAppArmorSudoProfile(rules []AppArmorPathRule) (string, error) {
	normalized, err := normalizeAppArmorPathRules(rules)
	if err != nil {
		return "", err
	}
	// Existing profile header stays the same.
	for _, rule := range normalized {
		fmt.Fprintf(&profile, "\n  audit deny %q %s,\n", rule.Path, rule.Permissions)
		fmt.Fprintf(&profile, "  audit deny %q %s,\n", strings.TrimSuffix(rule.Path, "/")+"/**", rule.Permissions)
	}
}
```

Keep permission output deterministic by sorting paths and sorting permission characters into a stable order.

- [ ] **Step 4: Preserve path validation**

Move existing path validation into `normalizeAppArmorPathRules`. The function must still reject relative paths, root `/`, NUL/newline, AppArmor glob characters, quotes, backslashes, and empty path lists.

- [ ] **Step 5: Run focused tests**

Run: `go test ./internal/collector -run 'TestGenerateAppArmorSudoProfile|TestNormalizeAppArmorOperations' -count=1`

Expected: all AppArmor profile generation tests pass.

### Task 2: Collector Policy Parsing and Deployment Results

**Files:**
- Modify: `backend/internal/collector/enforcement_sync.go`
- Modify: `backend/internal/collector/enforcement_sync_test.go`

- [ ] **Step 1: Add failing tests for `definition.operations`**

Add tests for these cases:

```go
func TestBuildAppArmorDeploymentUsesSensitiveFileOperations(t *testing.T) {
	policy := EnforcementPolicy{
		ID: "policy-1", Template: "sensitive_file", Mode: "enforce", Enabled: true,
		Definition: json.RawMessage(`{"filePaths":["/etc/docker/daemon.json"],"operations":["read","delete"]}`),
	}
	profile, _, results := buildAppArmorDeployment([]EnforcementPolicy{policy})
	if results["policy-1"].Status != "deployed" {
		t.Fatalf("expected deployed result, got %#v", results["policy-1"])
	}
	if !strings.Contains(profile, `audit deny "/etc/docker/daemon.json" rd,`) {
		t.Fatalf("expected read/delete permissions in profile:\n%s", profile)
	}
}

func TestBuildAppArmorDeploymentDefaultsLegacySensitiveFileToWrite(t *testing.T) {
	policy := EnforcementPolicy{
		ID: "policy-1", Template: "sensitive_file", Mode: "enforce", Enabled: true,
		Definition: json.RawMessage(`{"filePaths":["/etc/docker/daemon.json"]}`),
	}
	profile, _, results := buildAppArmorDeployment([]EnforcementPolicy{policy})
	if results["policy-1"].Status != "deployed" || !strings.Contains(profile, `wkl`) {
		t.Fatalf("expected legacy write protection, result=%#v profile=%s", results["policy-1"], profile)
	}
}
```

- [ ] **Step 2: Run test to verify RED**

Run: `go test ./internal/collector -run 'TestBuildAppArmorDeploymentUsesSensitiveFileOperations|TestBuildAppArmorDeploymentDefaultsLegacySensitiveFileToWrite' -count=1`

Expected: fails because operations are ignored or `GenerateAppArmorSudoProfile` signature changed.

- [ ] **Step 3: Extend definition parsing**

Update `sensitiveFileDefinition`:

```go
type sensitiveFileDefinition struct {
	FilePaths     []string `json:"filePaths"`
	Operations    []string `json:"operations"`
	UserMatchMode string   `json:"userMatchMode"`
}
```

Convert each file path into `AppArmorPathRule{Path: path, Operations: definition.Operations}`.

- [ ] **Step 4: Reject unsupported operation names per policy**

If operation normalization fails for one policy, set that policy result to:

```go
appArmorDeploymentResult{Status: "failed", Message: "策略 operations 无效: " + err.Error()}
```

Do not include its paths in the generated profile.

- [ ] **Step 5: Run collector tests**

Run: `go test ./internal/collector -count=1`

Expected: collector package passes.

### Task 3: Frontend Operation Selection

**Files:**
- Modify: `frontend/src/pages/settings/tetragonPolicy.ts`
- Modify: `frontend/src/pages/settings/tetragonPolicy.test.ts`
- Modify: `frontend/src/pages/settings/TetragonPolicyPage.tsx`

- [ ] **Step 1: Add failing frontend tests**

Add tests for default and preview output:

```ts
it('includes sensitive file operations in preview', () => {
  const preview = generatePolicy({
    template: 'sensitive_file',
    mode: 'enforce',
    name: 'protect',
    filePaths: ['/etc/docker/daemon.json'],
    operations: ['read', 'write', 'delete'],
  });
  expect(preview).toContain('operations:');
  expect(preview).toContain('  - read');
  expect(preview).toContain('  - delete');
});

it('defaults sensitive file operations to write', () => {
  const preview = generatePolicy({
    template: 'sensitive_file',
    mode: 'enforce',
    name: 'protect',
    filePaths: ['/etc/docker/daemon.json'],
  });
  expect(preview).toContain('  - write');
});
```

- [ ] **Step 2: Run test to verify RED**

Run: `npx vitest run src/pages/settings/tetragonPolicy.test.ts`

Expected: fails because preview does not include `operations`.

- [ ] **Step 3: Extend TypeScript types and defaults**

In `tetragonPolicy.ts`, add:

```ts
export type SensitiveFileOperation = 'read' | 'write' | 'create' | 'delete' | 'rename' | 'chmod' | 'chown' | 'all';

export interface PolicyFormValues {
  operations?: SensitiveFileOperation[];
}
```

Default `operations` to `['write']` for legacy compatibility.

- [ ] **Step 4: Add operation picker in the form**

In `TetragonPolicyPage.tsx`, add `operations` to `defaultValues`, `Form.useWatch`, `policy`, and the sensitive file form block:

```tsx
<Form.Item name="operations" label="保护操作" rules={[{ required: true, message: '请选择至少一种保护操作' }]}>
  <Select
    mode="multiple"
    options={[
      { value: 'read', label: '读取' },
      { value: 'write', label: '写入/编辑' },
      { value: 'create', label: '创建' },
      { value: 'delete', label: '删除' },
      { value: 'rename', label: '重命名/移动' },
      { value: 'chmod', label: '权限变更' },
      { value: 'chown', label: '属主变更' },
      { value: 'all', label: '全部保护' },
    ]}
  />
</Form.Item>
```

- [ ] **Step 5: Run frontend tests**

Run: `npx vitest run src/pages/settings/tetragonPolicy.test.ts`

Expected: all tetragon policy tests pass.

### Task 4: Deployment Status and Operator Feedback

**Files:**
- Modify: `backend/internal/collector/enforcement_sync.go`
- Modify: `backend/internal/collector/enforcement_sync_test.go`
- Modify: `frontend/src/pages/settings/TetragonPolicyPage.tsx`

- [ ] **Step 1: Add tests for unsupported metadata operations if needed**

If AppArmor parser validation or local reasoning shows `chmod/chown` cannot be represented distinctly, add a test that expects a failed deployment message for policies using only `chmod` or `chown`.

- [ ] **Step 2: Make deployment message operation-aware**

For deployed sensitive-file policies, use a message like:

```go
results[policy.ID] = appArmorDeploymentResult{
	Status:  "deployed",
	Message: "AppArmor 策略已加载，保护操作: " + strings.Join(normalizedOperations, ", "),
}
```

- [ ] **Step 3: Surface message in frontend without new layout**

Keep the existing deployment table and make sure the Collector message is visible through the existing `message` column. Do not add a new page or hero section for this phase.

- [ ] **Step 4: Run targeted tests**

Run: `go test ./internal/collector -count=1`

Expected: collector package passes and messages include operation names.

### Task 5: AppArmor Denial Evidence and Notification Check

**Files:**
- Inspect: `backend/internal/collector/apparmor_audit.go`
- Inspect: `backend/internal/collector/apparmor_audit_test.go`
- Inspect: `backend/internal/notification/notification.go`
- Modify only if evidence shows AppArmor denial events are not tagged for notification.

- [ ] **Step 1: Verify denial parser tags AppArmor enforcement events**

Run: `go test ./internal/collector -run AppArmor -count=1`

Expected: AppArmor audit tests pass and parsed denial events include `diting-enforcement` or a tag consumed by notification publication.

- [ ] **Step 2: Add failing test if tag is missing**

If parser output lacks the notification tag, add a test asserting:

```go
if !slices.Contains(event.Tags, "diting-enforcement") {
	t.Fatalf("expected AppArmor denial to create enforcement notification, tags=%#v", event.Tags)
}
```

- [ ] **Step 3: Add the minimal parser tag fix**

Ensure AppArmor denial events carry `diting-enforcement` and enough detail for notification text: `EventID`, `Severity`, `FilePath`, `ProcessName`, `Cmdline`, and user identity.

- [ ] **Step 4: Run collector notification path tests**

Run: `go test ./internal/collector ./cmd/audit-server -run 'AppArmor|EnforcementNotification' -count=1`

Expected: AppArmor parsing and notification publication tests pass.

### Task 6: End-to-End Verification

**Files:**
- Modify: `docs/superpowers/specs/2026-08-11-enforcement-hardening-design.md` only if implementation uncovers a real design correction.

- [ ] **Step 1: Format Go**

Run: `gofmt -w backend/internal/collector/apparmor.go backend/internal/collector/apparmor_test.go backend/internal/collector/enforcement_sync.go backend/internal/collector/enforcement_sync_test.go`

Expected: command exits 0.

- [ ] **Step 2: Run backend tests**

Run: `go test ./...` from `backend`.

Expected: all packages pass.

- [ ] **Step 3: Run frontend tests**

Run: `npm test` and `npx vitest run src/pages/settings/tetragonPolicy.test.ts` from `frontend`.

Expected: all tests pass.

- [ ] **Step 4: Run frontend build**

Run: `npm run build` from `frontend`.

Expected: TypeScript and Vite build succeed. Existing bundle-size warnings may remain.

- [ ] **Step 5: Diff hygiene**

Run: `git diff --check` and `git status --short`.

Expected: no whitespace errors; changed files match the planned scope.

## Manual Host Validation

Run these on a test Ubuntu host with AppArmor enabled and Collector sync active:

- Read test: `sudo cat /etc/docker/daemon.json` should be denied when `read` is selected.
- Write test: `sudo sh -c 'echo test >> /etc/docker/daemon.json'` should be denied when `write` is selected.
- Delete test: `sudo rm /path/to/protected-file` should be denied when `delete` is selected.
- Permission test: `sudo chmod 777 /path/to/protected-file` should be denied or reported unsupported according to final AppArmor permission support.
- Direct root control: directly logged-in root behavior should match the existing root-exclusion design.
- Notification check: denied operations should appear in audit events and create a notification center item.
