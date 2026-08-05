# Tetragon Enforcement Rebuild Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Rebuild DiTing's Tetragon policy layer around official runtime enforcement capabilities so the system uses Tetragon as its real foundation rather than a command blacklist generator.

**Architecture:** Split policy generation into a tested capability-driven module, add explicit Tetragon capability profiles, generate enforcement policies from path/syscall/ancestry hooks, parse enforcement events consistently, and surface operator-facing deployment warnings. The UI will present only capability-backed templates and will label unsupported or environment-dependent behavior clearly.

**Tech Stack:** Tetragon TracingPolicy YAML, Go backend collector/API, React + Ant Design frontend, TypeScript policy generator tests, Go collector/rule tests.

---

## File Structure

- Modify `frontend/src/pages/settings/tetragonPolicy.ts`: policy generation becomes capability-driven; remove sudo argv matching; add official hook builders.
- Modify `frontend/src/pages/settings/tetragonPolicy.test.ts`: executable generator assertions for command, sensitive file, delete protection, permission change, and sudo ancestry cases.
- Modify `frontend/src/pages/settings/TetragonPolicyPage.tsx`: UI exposes capability-backed templates, environment dependency hints, and enforcement modes by template.
- Modify `frontend/src/types/enforcement.ts`: add capability metadata fields if needed for warnings.
- Modify `backend/internal/collector/parser.go`: ensure file-mode `process_kprobe` parses path/file/syscall enforcement events into audit events.
- Modify `backend/internal/collector/parser_test.go`: cover `security_path_unlink`, `security_path_rmdir`, `security_file_open`, and `sys_execve` enforcement events.
- Modify `backend/internal/collector/grpc_parser.go`: ensure gRPC parsing preserves tags, paths, parent/process context, and event type for enforcement events.
- Modify `backend/internal/collector/grpc_parser_test.go`: mirror file-mode enforcement cases.
- Modify `backend/internal/rule/matcher.go`: keep tag matching and add any missing fields needed by enforcement rules.
- Modify `backend/internal/postgres/production_baseline.go` and migrations if default enforcement alert rules are persisted.
- Modify `backend/configs/config.example.yaml`: document `parents_map_enabled` dependency in Tetragon runtime notes or collector comments.
- Modify deployment docs `docs/production-deployment.md`: add Tetragon runtime prerequisites.
- Create `docs/tetragon-enforcement-capability.md`: official capability matrix and DiTing template mapping.

---

### Task 1: Document The Official Capability Matrix

**Files:**
- Create: `docs/tetragon-enforcement-capability.md`

- [ ] **Step 1: Write the capability document**

Create `docs/tetragon-enforcement-capability.md` with this content:

```markdown
# Tetragon Enforcement Capability Matrix

DiTing uses Tetragon as its runtime observability and enforcement layer. Policy templates must map to official Tetragon capabilities and must not promise behavior that depends on unverified command-line patterns.

## Runtime Prerequisites

- Tetragon must run privileged or with required BPF permissions.
- `/sys/fs/bpf` must be mounted and writable by Tetragon.
- BTF must be available through `/sys/kernel/btf/vmlinux` or a compatible fallback.
- Parent process matching requires Tetragon parent tracking. Deployments using `matchParentBinaries` with `followChildren: true` must start Tetragon with parent map support enabled.
- `Override` requires kernel support for BPF override. For `security_` hooks, Linux 5.7+ is required by Tetragon documentation.

## Template Capability Mapping

| DiTing Template | Strong Enforcement Hooks | Primary Use | Boundary |
|---|---|---|---|
| Dangerous command | `sys_execve` | Block direct execution of known dangerous binaries | Complex shell semantics are not guaranteed by argv matching |
| Sensitive file | `security_file_open`, optional `fd_install` | Block opening sensitive files by process/user/ancestry context | Write-mode precision depends on available hook arguments |
| Delete protection | `security_path_unlink`, `security_path_rmdir` | Block deletion of files/directories by path | Preferred over `rm` command matching |
| Permission change | `sys_chmod`, `sys_fchmodat`, `sys_chown`, `sys_fchownat`, path/security hooks when available | Block chmod/chown on protected paths | Path resolution differs by syscall/hook |
| Sudo ancestry | `matchParentBinaries` with `followChildren: true` | Distinguish sudo-root process chains from direct root sessions | Requires parent tracking and verified binary paths |

## Enforcement Actions

- `Sigkill` kills the current process and is useful as a stop signal.
- `Override` should be preferred where supported because it returns an error from the probed hook/syscall before the operation continues.
- DiTing strong-enforcement templates should emit `Override` when supported and `Sigkill` as a fallback action.

## Product Rules

- Do not implement strong enforcement by matching `sudo` command arguments.
- Do not label a template strong enforcement unless it targets the actual kernel behavior being protected.
- All enforce-mode policies must include `diting-enforcement` and `diting-blocked-command` tags for notification and audit correlation.
- Every generated template must have a documented manual verification command.
```

- [ ] **Step 2: Commit**

```bash
git add docs/tetragon-enforcement-capability.md
git commit -m "docs: define tetragon enforcement capability matrix"
```

---

### Task 2: Replace Sudo Argv Matching With Parent-Ancestry Matching

**Files:**
- Modify: `frontend/src/pages/settings/tetragonPolicy.ts`
- Modify: `frontend/src/pages/settings/tetragonPolicy.test.ts`

- [ ] **Step 1: Write failing tests**

In `frontend/src/pages/settings/tetragonPolicy.test.ts`, assert that generated sudo-aware policies contain official parent binary matching and do not contain sudo argv matching:

```ts
function assertNotIncludes(text: string, unexpected: string) {
  if (text.includes(unexpected)) {
    throw new Error(`expected YAML not to include ${unexpected}`);
  }
}

const sudoSensitiveYaml = generatePolicy({
  template: 'sensitive_file',
  mode: 'enforce',
  name: 'block-sudo-docker-config',
  enabled: true,
  filePaths: ['/etc/docker/daemon.json'],
  processNames: ['vim'],
  userMatchMode: 'exclude_root',
});

assertIncludes(sudoSensitiveYaml, 'matchParentBinaries:');
assertIncludes(sudoSensitiveYaml, 'followChildren: true');
assertIncludes(sudoSensitiveYaml, '- "/usr/bin/sudo"');
assertIncludes(sudoSensitiveYaml, '- "/bin/sudo"');
assertNotIncludes(sudoSensitiveYaml, '- "diting-sudo-pre-escalation"\n    - "process_exec"');
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
& 'E:\goProject\DiTing\frontend\node_modules\.bin\esbuild.cmd' 'E:\goProject\DiTing\frontend\src\pages\settings\tetragonPolicy.test.ts' --bundle --platform=node --format=esm --outfile='E:\goProject\DiTing\frontend\dist\tetragonPolicy.test.mjs'
node 'E:\goProject\DiTing\frontend\dist\tetragonPolicy.test.mjs'
```

Expected: FAIL if old sudo argv matching is still generated.

- [ ] **Step 3: Implement parent ancestry helper**

In `frontend/src/pages/settings/tetragonPolicy.ts`, use this helper:

```ts
function matchSudoAncestry() {
  return `
      matchParentBinaries:
      - operator: In
        values:
        - "/usr/bin/sudo"
        - "/bin/sudo"
        followChildren: true`;
}
```

Update sudo-aware selectors to append `matchSudoAncestry()` to the actual protected hook selector, not to a separate `sys_execve sudo` selector.

- [ ] **Step 4: Remove sudo argv helper functions**

Delete any functions that generate selectors matching:

```yaml
index: 0
values:
- "sudo"
```

The remaining sudo support must be ancestry-based.

- [ ] **Step 5: Run tests**

Run:

```powershell
& 'E:\goProject\DiTing\frontend\node_modules\.bin\esbuild.cmd' 'E:\goProject\DiTing\frontend\src\pages\settings\tetragonPolicy.test.ts' --bundle --platform=node --format=esm --outfile='E:\goProject\DiTing\frontend\dist\tetragonPolicy.test.mjs'
node 'E:\goProject\DiTing\frontend\dist\tetragonPolicy.test.mjs'
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/pages/settings/tetragonPolicy.ts frontend/src/pages/settings/tetragonPolicy.test.ts
git commit -m "feat: use tetragon parent ancestry for sudo-aware policies"
```

---

### Task 3: Rebuild Delete Protection As Strong Enforcement

**Files:**
- Modify: `frontend/src/pages/settings/tetragonPolicy.ts`
- Modify: `frontend/src/pages/settings/tetragonPolicy.test.ts`
- Modify: `frontend/src/pages/settings/TetragonPolicyPage.tsx`

- [ ] **Step 1: Write failing generator test**

Add this to `frontend/src/pages/settings/tetragonPolicy.test.ts`:

```ts
const deleteYaml = generatePolicy({
  template: 'delete_behavior',
  mode: 'enforce',
  name: 'block-delete-test',
  enabled: true,
  filePaths: ['/home/ubuntu/test'],
  userMatchMode: 'all',
});

assertIncludes(deleteYaml, 'call: "security_path_unlink"');
assertIncludes(deleteYaml, 'call: "security_path_rmdir"');
assertIncludes(deleteYaml, 'type: "path"');
assertIncludes(deleteYaml, 'operator: Equal');
assertIncludes(deleteYaml, '- "/home/ubuntu/test"');
assertIncludes(deleteYaml, 'action: Override');
assertIncludes(deleteYaml, 'argError: -1');
assertIncludes(deleteYaml, 'action: Sigkill');
assertIncludes(deleteYaml, '- "diting-enforcement"');
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```powershell
& 'E:\goProject\DiTing\frontend\node_modules\.bin\esbuild.cmd' 'E:\goProject\DiTing\frontend\src\pages\settings\tetragonPolicy.test.ts' --bundle --platform=node --format=esm --outfile='E:\goProject\DiTing\frontend\dist\tetragonPolicy.test.mjs'
node 'E:\goProject\DiTing\frontend\dist\tetragonPolicy.test.mjs'
```

Expected: FAIL because delete behavior currently does not generate strong enforcement.

- [ ] **Step 3: Implement delete protection block**

Replace `deleteBehaviorBlock` with a strong delete protection generator:

```ts
function deleteBehaviorBlock(paths: string[], processNames: string[], user: UserMatcher | null, mode: PolicyMode) {
  const values = (paths.filter(Boolean).length ? paths.filter(Boolean) : ['/']).map((path) => `            - "${escapeYaml(path)}"`).join('\n');
  const actionBlock = mode === 'enforce' ? enforcementActions(mode) : '';
  return `  kprobes:
  - call: "security_path_unlink"
    syscall: false
    return: false
    args:
    - index: 0
      type: "path"
${uidDataBlock(user)}
    tags:
    - "delete-protection"
    - "file_access"
${enforcementTags(mode)}
    selectors:
    - matchArgs:
      - index: 0
        operator: Equal
        values:
${values}${matchBinaries(processNames)}${matchUser(user)}${actionBlock}
  - call: "security_path_rmdir"
    syscall: false
    return: false
    args:
    - index: 0
      type: "path"
${uidDataBlock(user)}
    tags:
    - "delete-protection"
    - "file_access"
${enforcementTags(mode)}
    selectors:
    - matchArgs:
      - index: 0
        operator: Equal
        values:
${values}${matchBinaries(processNames)}${matchUser(user)}${actionBlock}`;
}
```

Add:

```ts
function enforcementActions(mode: PolicyMode) {
  if (mode !== 'enforce') {
    return '';
  }
  return `
      matchActions:
      - action: Override
        argError: -1
      - action: Sigkill`;
}
```

- [ ] **Step 4: Enable enforce mode in UI for delete behavior**

In `TetragonPolicyPage.tsx`, remove the special case that disables enforce for `delete_behavior`.

Change:

```tsx
{ value: 'enforce', label: '拦截', disabled: template === 'delete_behavior' },
```

to:

```tsx
{ value: 'enforce', label: '拦截' },
```

Update the delete behavior alert copy to:

```tsx
message={mode === 'enforce' ? '删除保护会阻断指定路径删除' : '删除行为模板可审计指定路径删除'}
description={mode === 'enforce'
  ? '强拦截使用 security_path_unlink/security_path_rmdir，并优先通过 Override 返回错误。需要目标内核支持 Tetragon Override。'
  : '建议先审计观察命中路径，确认后再切换为拦截。'}
```

- [ ] **Step 5: Run tests and build**

Run:

```powershell
& 'E:\goProject\DiTing\frontend\node_modules\.bin\esbuild.cmd' 'E:\goProject\DiTing\frontend\src\pages\settings\tetragonPolicy.test.ts' --bundle --platform=node --format=esm --outfile='E:\goProject\DiTing\frontend\dist\tetragonPolicy.test.mjs'
node 'E:\goProject\DiTing\frontend\dist\tetragonPolicy.test.mjs'
npm run build
```

Expected: generator test PASS; build PASS.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/pages/settings/tetragonPolicy.ts frontend/src/pages/settings/tetragonPolicy.test.ts frontend/src/pages/settings/TetragonPolicyPage.tsx
git commit -m "feat: add strong tetragon delete protection template"
```

---

### Task 4: Add Enforcement Event Parsing Coverage

**Files:**
- Modify: `backend/internal/collector/parser.go`
- Modify: `backend/internal/collector/parser_test.go`
- Modify: `backend/internal/collector/grpc_parser.go`
- Modify: `backend/internal/collector/grpc_parser_test.go`

- [ ] **Step 1: Add file-mode delete enforcement test**

In `backend/internal/collector/parser_test.go`, add:

```go
func TestParseTetragonProcessKprobeDeleteProtectionEvent(t *testing.T) {
	line := `{"process_kprobe":{"function_name":"security_path_unlink","policy_name":"block-delete","tags":["delete-protection","diting-enforcement"],"process":{"pid":10,"uid":1000,"binary":"/usr/bin/rm","arguments":"/home/ubuntu/test","process_credentials":{"uid":1000,"gid":1000,"euid":1000,"egid":1000},"pod":{}},"parent":{"pid":1,"binary":"/bin/bash","arguments":"-l"},"args":[{"path_arg":{"path":"/home/ubuntu/test"}}]},"node_name":"server-1","time":"2026-08-05T07:08:20.928822560Z"}`

	event, err := ParseTetragonEvent([]byte(line))
	if err != nil {
		t.Fatalf("ParseTetragonEvent returned error: %v", err)
	}
	if event.EventType != "file_access" || event.Action != "security_path_unlink" {
		t.Fatalf("expected file_access security_path_unlink, got type=%s action=%s", event.EventType, event.Action)
	}
	if event.FilePath != "/home/ubuntu/test" {
		t.Fatalf("expected protected path, got %q", event.FilePath)
	}
	if len(event.Tags) == 0 || event.Tags[0] != "delete-protection" {
		t.Fatalf("expected delete protection tags, got %#v", event.Tags)
	}
}
```

- [ ] **Step 2: Run collector tests to verify failure if parser misses path context**

Run:

```bash
go test ./internal/collector
```

Expected: FAIL if file-mode kprobe parser does not populate `FilePath` or `file_access`.

- [ ] **Step 3: Implement file-mode path context**

In `backend/internal/collector/parser.go`, add helper:

```go
func processKprobePathContext(event *processKprobeEvent) (string, string) {
	for _, arg := range append(event.Args, event.Data...) {
		if arg.PathArg.Path != "" {
			return arg.PathArg.Path, firstNonEmpty(arg.PathArg.Permission, arg.PathArg.Flags, event.FunctionName)
		}
		if arg.FileArg.Path != "" {
			return arg.FileArg.Path, firstNonEmpty(arg.FileArg.Permission, arg.FileArg.Flags, event.FunctionName)
		}
		if arg.SockaddrU.Path != "" {
			return arg.SockaddrU.Path, event.FunctionName
		}
		if arg.StringArg != "" && isFileSyscall(event.FunctionName) {
			return arg.StringArg, event.FunctionName
		}
	}
	return "", ""
}
```

Update `parseProcessKprobe`:

```go
filePath, fileOperation := processKprobePathContext(envelope.ProcessKprobe)
eventType := "process_kprobe"
if filePath != "" {
	eventType = "file_access"
}
```

Populate:

```go
EventType: eventType,
FilePath: filePath,
FileOperation: fileOperation,
```

- [ ] **Step 4: Add gRPC test for delete protection event**

In `backend/internal/collector/grpc_parser_test.go`, add a `ProcessKprobe` case using `security_path_unlink`, `Tags: []string{"delete-protection", "diting-enforcement"}`, and `KprobeArgument_PathArg{Path: "/home/ubuntu/test"}`. Assert `EventType == "file_access"`, `Action == "security_path_unlink"`, `FilePath == "/home/ubuntu/test"`.

- [ ] **Step 5: Run collector tests**

Run:

```bash
go test ./internal/collector
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add backend/internal/collector/parser.go backend/internal/collector/parser_test.go backend/internal/collector/grpc_parser.go backend/internal/collector/grpc_parser_test.go
git commit -m "feat: parse tetragon enforcement path events"
```

---

### Task 5: Add Runtime Prerequisite Checks To UI

**Files:**
- Modify: `frontend/src/pages/settings/TetragonPolicyPage.tsx`
- Modify: `backend/configs/config.example.yaml`
- Modify: `docs/production-deployment.md`

- [ ] **Step 1: Add UI alert for enforcement prerequisites**

In `TetragonPolicyPage.tsx`, add an `Alert` near the top of the form when `mode === 'enforce'`:

```tsx
{mode === 'enforce' && (
  <Alert
    type="warning"
    showIcon
    style={{ marginBottom: 16 }}
    message="强拦截依赖 Tetragon 与内核能力"
    description="Override 需要内核支持；sudo 链路识别依赖 parent map。请确认 Tetragon 启动参数包含 parents map 支持，并在目标主机实测策略加载日志。"
  />
)}
```

- [ ] **Step 2: Update example config comments**

In `backend/configs/config.example.yaml`, add comments above enforcement fields:

```yaml
  # Strong enforcement policies depend on the target Tetragon runtime.
  # Sudo ancestry matching requires Tetragon parent map support.
  # Override actions require compatible kernel support.
```

- [ ] **Step 3: Update deployment docs**

In `docs/production-deployment.md`, add:

```markdown
## Tetragon Enforcement Runtime Checks

Before enabling DiTing strong enforcement templates:

1. Confirm bpffs is mounted: `mount | grep /sys/fs/bpf`.
2. Confirm Tetragon starts without policy load errors.
3. Confirm parent ancestry matching is enabled before using sudo-aware policies.
4. Confirm Override support on the target kernel before relying on delete/path protection.
5. Verify generated policy tags include `diting-enforcement`.
```

- [ ] **Step 4: Build frontend**

Run:

```bash
npm run build
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/pages/settings/TetragonPolicyPage.tsx backend/configs/config.example.yaml docs/production-deployment.md
git commit -m "docs: surface tetragon enforcement prerequisites"
```

---

### Task 6: Add Default Enforcement Alert Rule

**Files:**
- Modify: `backend/internal/rule/matcher.go`
- Modify: `backend/internal/rule/matcher_test.go`
- Modify: `backend/internal/postgres/production_baseline.go`
- Add migration: `backend/migrations/postgres/016_default_enforcement_alert_rules.sql`

- [ ] **Step 1: Confirm tag matching test exists**

Ensure `backend/internal/rule/matcher_test.go` contains:

```go
func TestMatcherContainsTag(t *testing.T) {
	event := audit.Event{Tags: []string{"diting-enforcement", "delete-protection"}}
	expr := Expression{
		Operator: "and",
		Conditions: []Condition{
			{Field: "tags", Op: "contains", Value: "diting-enforcement"},
		},
	}
	if !Match(expr, event) {
		t.Fatal("expected tags contains rule to match")
	}
}
```

- [ ] **Step 2: Add default rule migration**

Create `backend/migrations/postgres/016_default_enforcement_alert_rules.sql`:

```sql
INSERT INTO diting_audit_rules (id, name, description, event_type, enabled, severity, risk_score, match_expr, tags, created_at, updated_at)
VALUES (
    '00000000-0000-0000-0000-000000000016',
    'Tetragon 强拦截触发',
    '识别 DiTing 下发的 Tetragon enforcement 策略触发事件。',
    'file_access',
    TRUE,
    'critical',
    95,
    '{"operator":"and","conditions":[{"field":"tags","op":"contains","value":"diting-enforcement"}]}'::jsonb,
    '["tetragon","enforcement","blocked"]'::jsonb,
    NOW(),
    NOW()
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    event_type = EXCLUDED.event_type,
    enabled = EXCLUDED.enabled,
    severity = EXCLUDED.severity,
    risk_score = EXCLUDED.risk_score,
    match_expr = EXCLUDED.match_expr,
    tags = EXCLUDED.tags,
    updated_at = NOW();
```

- [ ] **Step 3: Add production baseline entry**

Mirror the same insert in `backend/internal/postgres/production_baseline.go` if this project embeds baseline SQL there.

- [ ] **Step 4: Run backend tests**

Run:

```bash
go test ./internal/rule ./internal/postgres
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/rule/matcher.go backend/internal/rule/matcher_test.go backend/internal/postgres/production_baseline.go backend/migrations/postgres/016_default_enforcement_alert_rules.sql
git commit -m "feat: add default tetragon enforcement alert rule"
```

---

### Task 7: End-To-End Manual Verification Matrix

**Files:**
- Create: `docs/tetragon-enforcement-verification.md`

- [ ] **Step 1: Add verification doc**

Create `docs/tetragon-enforcement-verification.md`:

```markdown
# Tetragon Enforcement Manual Verification

Run these checks on a Linux host with the updated DiTing collector and Tetragon policy sync enabled.

## Common Checks

```bash
mount | grep /sys/fs/bpf
docker logs tetragon --tail 200 | grep -i -E 'error|failed|policy|override|parent'
grep -R 'diting-enforcement' /home/diting/tetragon/policies
```

## Dangerous Command

Policy: block `reboot` in enforce mode.

```bash
reboot
```

Expected: process killed or syscall blocked; DiTing notification appears with `diting-enforcement`.

## Sensitive File Direct Access

Policy: protect `/etc/docker/daemon.json`, process `vim`, exclude root actual UID.

```bash
vim /etc/docker/daemon.json
```

Expected for non-root user: blocked.

## Sensitive File Sudo Ancestry

Policy: same as above with sudo ancestry enabled.

```bash
sudo vim /etc/docker/daemon.json
```

Expected: blocked if Tetragon parent tracking is enabled.

## Direct Root Session

```bash
sudo -i
vim /etc/docker/daemon.json
```

Expected: allowed if policy excludes root and no sudo ancestry selector is intended to match the direct root shell. If ancestry still matches due to shell being spawned by sudo, log this as expected environment behavior and use true root SSH session for direct-root verification.

## Delete Protection

Policy: protect `/home/ubuntu/test`.

```bash
touch /home/ubuntu/test
rm /home/ubuntu/test
python3 -c 'import os; os.unlink("/home/ubuntu/test")'
```

Expected: file remains; DiTing notification appears.
```

- [ ] **Step 2: Commit**

```bash
git add docs/tetragon-enforcement-verification.md
git commit -m "docs: add tetragon enforcement verification matrix"
```

---

## Self-Review

- Spec coverage: The plan covers official capability documentation, sudo ancestry, delete protection, parser support, runtime prerequisites, default alerting, and manual verification.
- Placeholder scan: No `TBD` or vague implementation-only steps remain; each task has concrete files and commands.
- Type consistency: `PolicyFormValues`, `PolicyTemplate`, `PolicyMode`, `UserMatcher`, and helper names are defined before use or refer to existing files.
- Known risk: `matchParentBinaries` requires runtime support and exact sudo binary paths; deployment checks must verify this before claiming sudo-aware enforcement works.

