# Persistent Notifications Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a PostgreSQL-backed, per-user notification center with manual enforcement disposition and automatic service-alert recovery.

**Architecture:** The `notification` package owns domain types, PostgreSQL/memory repositories, HTTP handlers, event creation, and collector-health reconciliation. Audit writes publish enforcement notifications; a background reconciler opens and resolves Collector/Tetragon incidents. React consumes notification APIs and renders unread, pending, and all views.

**Tech Stack:** Go 1.26, pgx/PostgreSQL, net/http, React 18, TypeScript, Ant Design, Vitest.

---

### Task 1: Notification Persistence
- [ ] Add failing lifecycle tests for permanent enforcement dedupe and state recurrence.
- [ ] Add PostgreSQL migrations and repository implementation.
- [ ] Run `go test ./internal/notification`.

### Task 2: Notification HTTP API
- [ ] Add failing handler tests for list, read, read-all, and disposition.
- [ ] Implement authenticated handlers and server routes.
- [ ] Run notification and server tests.

### Task 3: Notification Producers
- [ ] Add failing tests for enforcement publication and health reconciliation.
- [ ] Publish enforcement notifications after successful writes.
- [ ] Start Collector/Tetragon reconciliation and verify open/resolve behavior.

### Task 4: Frontend Notification Center
- [ ] Add failing presentation tests.
- [ ] Implement notification API/types and the three-view panel.
- [ ] Implement read, read-all, disposition, and event deep-link actions.

### Task 5: Verification
- [ ] Run `gofmt -w`, `go test ./...`, `npm test`, and `npm run build`.
- [ ] Run `git diff --check` and inspect `git status --short`.