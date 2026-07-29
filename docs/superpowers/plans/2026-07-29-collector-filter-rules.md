# Collector Filter Rules Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Change collector filtering from one flat ignore list into multiple enabled rules, where all conditions inside a rule must match and any matching rule drops the event unless its severity is protected.

**Architecture:** Keep the existing `collector_filter` system config key and add a `rules` array to the JSON payload. Backend normalization preserves old fields for backward compatibility and converts them into a legacy-compatible rule set; the collector writer evaluates rules after audit rule enrichment. The frontend replaces flat tag inputs with a `Form.List` rule editor.

**Tech Stack:** Go, Postgres JSONB system configs, React, Ant Design, TypeScript.

---

### Task 1: Backend Config Model

**Files:**
- Modify: `backend/internal/systemconfig/repository.go`
- Modify: `backend/internal/systemconfig/handler.go`
- Test: `backend/internal/systemconfig/repository_test.go`
- Test: `backend/internal/systemconfig/handler_test.go`

- [ ] Add `CollectorFilterRule` and `CollectorFilterCondition` types with `id`, `name`, `enabled`, `conditions`, `field`, `op`, `value`, and `values`.
- [ ] Add `Rules []CollectorFilterRule` to `CollectorFilterConfig`.
- [ ] Normalize nil `Rules` to an empty slice and keep old fields normalized for API compatibility.
- [ ] Validate condition fields and operators in the handler.
- [ ] Update tests to cover saving and returning multi-rule configs.

### Task 2: Collector Rule Evaluation

**Files:**
- Create: `backend/cmd/audit-server/collector_filter.go`
- Modify: `backend/cmd/audit-server/main.go`
- Test: `backend/cmd/audit-server/collector_writer_test.go`

- [ ] Move collector filter matching logic into `collector_filter.go`.
- [ ] Implement `ShouldDrop` as: disabled filter keeps everything; protected severity keeps everything; any enabled rule with all conditions matching drops the event.
- [ ] Preserve old `ignoreProcessNames`, `ignoreCommandKeywords`, and `ignoreUsers` behavior by evaluating them when `rules` is empty.
- [ ] Add tests for rule-internal AND, rule-level OR, disabled rule, and protected severity.

### Task 3: Frontend Rule Editor

**Files:**
- Modify: `frontend/src/types/systemConfig.ts`
- Modify: `frontend/src/pages/settings/CollectorConfigPage.tsx`

- [ ] Add TypeScript types for `CollectorFilterRule` and `CollectorFilterCondition`.
- [ ] Replace the three flat ignore selectors with a rule list editor.
- [ ] Each rule supports name, enabled switch, and condition rows.
- [ ] Each condition supports field, operator, and tag values.
- [ ] Save payload includes `rules` plus existing `keepSeverities`.

### Task 4: Verification

**Commands:**
- `cd backend; $env:GOCACHE='E:\goProject\DiTing\.cache\go-build'; go test ./...`
- `cd frontend; npm run build`

- [ ] Run backend tests and fix any failures.
- [ ] Run frontend build and fix any TypeScript errors.
- [ ] Restore `frontend/tsconfig.tsbuildinfo` after frontend build.
