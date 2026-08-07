# Audit Filter Selects Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace manual audit user and host filter inputs with searchable dropdowns populated by user-audit and host-audit data.

**Architecture:** A focused `AuditEntitySelect` module owns option normalization, async loading, stale-request protection, and the two public user/host select components. Existing pages retain their forms and query builders, passing the active time range into the shared selectors so backend query semantics do not change.

**Tech Stack:** React 18, TypeScript, Ant Design 5, Axios, Day.js, Vite, Vitest.

---

### Task 1: Testable Audit Option Mapping

**Files:**
- Modify: `frontend/package.json`
- Modify: `frontend/package-lock.json`
- Create: `frontend/src/components/auditEntityOptions.test.ts`
- Create: `frontend/src/components/auditEntityOptions.ts`

- [ ] **Step 1: Add the Vitest test command and dependency**

Add `"test": "vitest run"` to scripts and `"vitest": "^3.2.4"` to dev dependencies, then run `npm install` in `frontend` so the lockfile records the exact dependency tree.

- [ ] **Step 2: Write failing option mapping tests**

Create tests that call `buildUserOptions` with empty and repeated usernames and expect only unique non-empty `{ value, label }` entries. Add host cases proving value fallback order `hostId -> nodeName -> hostName`, distinct identity labels, duplicate submitted-value removal, and empty-record removal.

```ts
expect(buildUserOptions([{ username: 'root' }, { username: '' }, { username: 'root' }])).toEqual([
  { value: 'root', label: 'root' },
]);
expect(buildHostOptions([
  { hostId: 'id-1', hostName: 'web-1', nodeName: 'node-1' },
  { hostName: 'web-2', nodeName: 'node-2' },
  { hostName: 'web-3' },
])).toEqual([
  { value: 'id-1', label: 'web-1 / node-1 / id-1' },
  { value: 'node-2', label: 'web-2 / node-2' },
  { value: 'web-3', label: 'web-3' },
]);
```

- [ ] **Step 3: Run the tests and verify RED**

Run `npm test -- src/components/auditEntityOptions.test.ts` from `frontend`. Expected: FAIL because `auditEntityOptions.ts` or its exports do not exist.

- [ ] **Step 4: Implement minimal pure mapping helpers**

Export `AuditSelectOption`, `buildUserOptions`, and `buildHostOptions`. Accept only the identity fields required from `UserAuditItem` and `HostAuditItem`; trim values, preserve response order, and deduplicate with `Set`.

- [ ] **Step 5: Run the tests and verify GREEN**

Run `npm test -- src/components/auditEntityOptions.test.ts`. Expected: all mapping tests PASS.

### Task 2: Shared Async User and Host Selects

**Files:**
- Create: `frontend/src/components/AuditEntitySelect.tsx`
- Modify: `frontend/src/api/stats.ts`

- [ ] **Step 1: Add option-loading API helpers**

Add `getAuditUserOptions(startTime?, endTime?)` and `getAuditHostOptions(startTime?, endTime?)` that call existing audit endpoints with `limit: 100` and map results using Task 1 helpers. Do not add backend routes.

- [ ] **Step 2: Implement the shared selector state**

Implement `AuditUserSelect` and `AuditHostSelect` around a private loader component. Props extend the Ant Design string `Select` props and add optional `startTime`/`endTime`. Use `useEffect` plus a monotonically increasing request sequence to ignore stale responses. Set `showSearch`, `optionFilterProp="label"`, `allowClear`, loading state, and `notFoundContent` messages for loading, empty, and failure states. Never enable `tags` or free-text values.

- [ ] **Step 3: Type-check the shared components**

Run `npm run build` from `frontend`. Expected: TypeScript and Vite build complete successfully.

### Task 3: Replace Page-Level Inputs

**Files:**
- Modify: `frontend/src/pages/audit-events/AuditEventsPage.tsx`
- Modify: `frontend/src/pages/commands/CommandStatsPage.tsx`
- Modify: `frontend/src/pages/risks/RiskEventsPage.tsx`
- Modify: `frontend/src/pages/users/UserAuditPage.tsx`
- Modify: `frontend/src/pages/settings/CollectorDebugPage.tsx`

- [ ] **Step 1: Watch each form's effective time range**

Use `Form.useWatch('timeRange', form) ?? defaultRange` in each page and convert its boundaries to ISO strings for selector props. Keep existing query builders unchanged.

- [ ] **Step 2: Replace audit event filters**

Replace host with `AuditHostSelect`, and login/execution users with `AuditUserSelect`, retaining `filter-control-compact` and existing field names.

- [ ] **Step 3: Replace command and risk filters**

Replace their user and host `Input` controls with the matching shared selectors. Preserve CSS classes and clear behavior.

- [ ] **Step 4: Replace user-audit and collector-debug host filters**

Replace the page-level user-audit host input and collector-debug host input with `AuditHostSelect`. Preserve the one-hour default in collector debug and seven-day default in user audit.

- [ ] **Step 5: Build after page integration**

Run `npm run build` from `frontend`. Expected: build succeeds without TypeScript errors.

### Task 4: Replace Detail Filters

**Files:**
- Modify: `frontend/src/pages/users/UserAuditPage.tsx`
- Modify: `frontend/src/pages/hosts/HostAuditPage.tsx`

- [ ] **Step 1: Replace the user-audit detail host input**

Use `AuditHostSelect` with the selected user's effective detail time range. Keep controlled `value` and update `detailFilters.hostName` directly from the select value.

- [ ] **Step 2: Keep host-detail users constrained to host audit data**

Retain the existing `Select` backed by `getHostUsers`, add `showSearch`, `optionFilterProp="label"`, an explicit empty message, and unique non-empty username options. This source is more specific than the global user list and already comes from host audit.

- [ ] **Step 3: Run complete frontend tests and build**

Run `npm test` and `npm run build` from `frontend`. Expected: all tests pass and the production build succeeds.

### Task 5: Regression and Browser Verification

**Files:**
- Modify only if verification reveals a scoped defect in the files above.

- [ ] **Step 1: Run backend regression tests**

Run `go test ./...` from `backend`. Expected: all packages pass, confirming existing `/stats/users` and `/stats/hosts` behavior remains intact.

- [ ] **Step 2: Start the frontend development server**

Run `npm run dev -- --host 127.0.0.1` from `frontend`, selecting another available port if the default is occupied.

- [ ] **Step 3: Verify affected routes in the browser**

Check `/audit/events`, `/audit/commands`, `/audit/risks`, `/audit/users`, `/audit/hosts`, and `/settings/collector-debug`. Verify controls render as selects, open without overlap, show loading/empty/error states correctly, search locally, select only listed values, clear successfully, and retain layout at desktop and mobile widths.

- [ ] **Step 4: Final verification**

Run `git diff --check`, `npm test`, `npm run build`, and `go test ./...` again after any verification fix. Expected: no whitespace errors and every command exits successfully.
