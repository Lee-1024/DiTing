# DiTing MVP Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first usable DiTing operation audit MVP from Tetragon JSON events to ClickHouse storage, Go APIs, PostgreSQL-backed configuration, and React UI.

**Architecture:** The MVP uses a single Go binary with two modes: `api` and `collector`. The collector tails a Tetragon JSON export file, normalizes events, matches PostgreSQL audit rules, and writes audit events into ClickHouse. The API server reads ClickHouse for audit logs and PostgreSQL for users/rules/configuration, while the React frontend provides login, dashboard, audit event search, event detail, and rule management.

**Tech Stack:** Go, Gin, zap, Viper, pgx/GORM, clickhouse-go/v2, PostgreSQL, ClickHouse, React, TypeScript, Vite, Ant Design, ECharts, Docker Compose.

---

## File Structure

Create this structure:

```text
backend/
  cmd/audit-server/main.go
  internal/app/app.go
  internal/config/config.go
  internal/logger/logger.go
  internal/server/server.go
  internal/auth/
  internal/audit/
  internal/collector/
  internal/rule/
  internal/stats/
  internal/postgres/
  internal/clickhouse/
  migrations/postgres/001_init.sql
  migrations/clickhouse/001_audit_events.sql
  configs/config.example.yaml
  sample-events/process_exec.jsonl
  go.mod

frontend/
  src/
    api/
    app/
    components/
    layouts/
    pages/
    router/
    stores/
    types/
    utils/
  package.json
  vite.config.ts
  tsconfig.json

deploy/
  docker-compose.yaml
```

## Task 1: Backend Skeleton

**Files:**
- Create: `backend/go.mod`
- Create: `backend/cmd/audit-server/main.go`
- Create: `backend/internal/config/config.go`
- Create: `backend/internal/logger/logger.go`
- Create: `backend/internal/server/server.go`
- Create: `backend/configs/config.example.yaml`

- [ ] **Step 1: Write failing config test**

Create `backend/internal/config/config_test.go`:

```go
package config

import "testing"

func TestLoadReadsServerAndDatabaseConfig(t *testing.T) {
	cfg, err := Load("../../configs/config.example.yaml")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("expected server port 8080, got %d", cfg.Server.Port)
	}
	if cfg.ClickHouse.Database != "diting" {
		t.Fatalf("expected ClickHouse database diting, got %q", cfg.ClickHouse.Database)
	}
	if cfg.Postgres.Database != "diting" {
		t.Fatalf("expected PostgreSQL database diting, got %q", cfg.Postgres.Database)
	}
}
```

- [ ] **Step 2: Run test and verify failure**

Run:

```bash
cd backend
go test ./internal/config
```

Expected: FAIL because `Load` and config types do not exist.

- [ ] **Step 3: Implement config loading**

Implement `backend/internal/config/config.go` with structs for server, ClickHouse, PostgreSQL, collector, and JWT settings. Use Viper to read YAML from an explicit path.

- [ ] **Step 4: Add example config**

Create `backend/configs/config.example.yaml`:

```yaml
server:
  port: 8080
  mode: debug

jwt:
  secret: change-me
  expires_hours: 24

postgres:
  host: 127.0.0.1
  port: 5432
  database: diting
  username: diting
  password: diting
  ssl_mode: disable

clickhouse:
  addr: 127.0.0.1:9000
  database: diting
  username: default
  password: ""

collector:
  input_mode: file
  tetragon_log_file: /data/tetragon/logs/tetragon.log
  flush_interval_seconds: 1
  batch_size: 1000
```

- [ ] **Step 5: Run config test and verify pass**

Run:

```bash
cd backend
go test ./internal/config
```

Expected: PASS.

- [ ] **Step 6: Add HTTP server health endpoint**

Create `backend/internal/server/server.go` with Gin router exposing:

```text
GET /healthz
```

Response:

```json
{"status":"ok"}
```

- [ ] **Step 7: Add health endpoint test**

Create `backend/internal/server/server_test.go`:

```go
package server

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzReturnsOK(t *testing.T) {
	router := NewRouter()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}
	if rec.Body.String() != `{"status":"ok"}` {
		t.Fatalf("unexpected body: %s", rec.Body.String())
	}
}
```

- [ ] **Step 8: Run backend tests**

Run:

```bash
cd backend
go test ./...
```

Expected: PASS.

## Task 2: Database Migrations

**Files:**
- Create: `backend/migrations/postgres/001_init.sql`
- Create: `backend/migrations/clickhouse/001_audit_events.sql`
- Create: `deploy/docker-compose.yaml`

- [ ] **Step 1: Add PostgreSQL migration**

Create tables:

- `users`
- `roles`
- `user_roles`
- `audit_rules`
- `system_configs`

Use the schema from `docs/operation-audit-platform-technical-design.md`.

- [ ] **Step 2: Add ClickHouse migration**

Create `audit_events` table using the schema from the technical design document.

- [ ] **Step 3: Add Docker Compose dependencies**

Create `deploy/docker-compose.yaml` with services:

- `postgres`
- `clickhouse`

Expose PostgreSQL `5432` and ClickHouse `8123`, `9000`.

- [ ] **Step 4: Verify compose config**

Run:

```bash
docker compose -f deploy/docker-compose.yaml config
```

Expected: command exits 0 and prints resolved compose configuration.

## Task 3: Audit Event Model and Rule Matching

**Files:**
- Create: `backend/internal/audit/event.go`
- Create: `backend/internal/rule/matcher.go`
- Create: `backend/internal/rule/matcher_test.go`

- [ ] **Step 1: Write failing rule matcher test**

Create tests that assert:

- `contains` matches `cmdline`.
- `eq` matches `event_type`.
- `or` returns true when one condition matches.
- `and` returns false when one condition fails.

- [ ] **Step 2: Run test and verify failure**

Run:

```bash
cd backend
go test ./internal/rule
```

Expected: FAIL because matcher does not exist.

- [ ] **Step 3: Implement AuditEvent**

Define `AuditEvent` with fields matching the ClickHouse table and JSON API response names.

- [ ] **Step 4: Implement matcher**

Implement rule expressions:

- `eq`
- `neq`
- `contains`
- `prefix`
- `suffix`
- `in`
- `regex`
- `and`
- `or`

- [ ] **Step 5: Run matcher tests**

Run:

```bash
cd backend
go test ./internal/rule
```

Expected: PASS.

## Task 4: Tetragon JSON Parser

**Files:**
- Create: `backend/internal/collector/parser.go`
- Create: `backend/internal/collector/parser_test.go`
- Create: `backend/sample-events/process_exec.jsonl`

- [ ] **Step 1: Capture or create sample Tetragon process event**

Use one JSON line from the running Tetragon export file. Store it in:

```text
backend/sample-events/process_exec.jsonl
```

- [ ] **Step 2: Write failing parser test**

Test that parsing the sample returns:

- `event_type = process_exec`
- non-empty `event_id`
- non-empty `event_time`
- expected `process_name`
- expected `cmdline`
- host/container fields default to empty string when absent

- [ ] **Step 3: Run parser test and verify failure**

Run:

```bash
cd backend
go test ./internal/collector
```

Expected: FAIL because parser does not exist.

- [ ] **Step 4: Implement parser**

Parse the Tetragon JSON using typed structs for known fields and `map[string]any` fallback for raw preservation.

- [ ] **Step 5: Run parser test**

Run:

```bash
cd backend
go test ./internal/collector
```

Expected: PASS.

## Task 5: ClickHouse Writer

**Files:**
- Create: `backend/internal/clickhouse/audit_writer.go`
- Create: `backend/internal/clickhouse/audit_writer_test.go`

- [ ] **Step 1: Write batch writer test with fake sink**

Test that writer accepts a slice of `AuditEvent` and builds one insert batch call. Use an interface around ClickHouse batch behavior to avoid requiring live ClickHouse for unit tests.

- [ ] **Step 2: Run test and verify failure**

Run:

```bash
cd backend
go test ./internal/clickhouse
```

Expected: FAIL because writer does not exist.

- [ ] **Step 3: Implement writer**

Implement batch insertion against `audit_events`.

- [ ] **Step 4: Run writer tests**

Run:

```bash
cd backend
go test ./internal/clickhouse
```

Expected: PASS.

## Task 6: Audit Query API

**Files:**
- Create: `backend/internal/audit/query.go`
- Create: `backend/internal/audit/handler.go`
- Create: `backend/internal/audit/handler_test.go`
- Modify: `backend/internal/server/server.go`

- [ ] **Step 1: Write query validation tests**

Assert:

- Missing time range defaults to last 24 hours.
- `page_size` greater than 500 is capped to 500.
- Invalid time returns HTTP 400.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
cd backend
go test ./internal/audit
```

Expected: FAIL because handlers do not exist.

- [ ] **Step 3: Implement query parser and handler**

Expose:

```text
GET /api/v1/audit/events
GET /api/v1/audit/events/:event_id
```

- [ ] **Step 4: Run audit tests**

Run:

```bash
cd backend
go test ./internal/audit
```

Expected: PASS.

## Task 7: Rule CRUD API

**Files:**
- Create: `backend/internal/rule/repository.go`
- Create: `backend/internal/rule/handler.go`
- Create: `backend/internal/rule/handler_test.go`
- Modify: `backend/internal/server/server.go`

- [ ] **Step 1: Write handler tests**

Assert:

- Create rule validates required name.
- Create rule validates severity.
- List returns rules from repository.
- Enable/disable updates status.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
cd backend
go test ./internal/rule
```

Expected: FAIL because CRUD handlers do not exist.

- [ ] **Step 3: Implement repository interface and handlers**

Implement handlers against an interface so unit tests can use an in-memory repository.

- [ ] **Step 4: Run rule tests**

Run:

```bash
cd backend
go test ./internal/rule
```

Expected: PASS.

## Task 8: Collector File Tail Loop

**Files:**
- Create: `backend/internal/collector/file_tail.go`
- Create: `backend/internal/collector/collector.go`
- Create: `backend/internal/collector/collector_test.go`

- [ ] **Step 1: Write collector test**

Test that a file containing two JSON lines is parsed into two `AuditEvent` values and passed to a fake writer.

- [ ] **Step 2: Run test and verify failure**

Run:

```bash
cd backend
go test ./internal/collector
```

Expected: FAIL because collector loop does not exist.

- [ ] **Step 3: Implement file collector**

Implement a file input mode that reads JSON lines and flushes when batch size is reached.

- [ ] **Step 4: Run collector tests**

Run:

```bash
cd backend
go test ./internal/collector
```

Expected: PASS.

## Task 9: Frontend Skeleton

**Files:**
- Create: `frontend/package.json`
- Create: `frontend/vite.config.ts`
- Create: `frontend/tsconfig.json`
- Create: `frontend/src/main.tsx`
- Create: `frontend/src/app/App.tsx`
- Create: `frontend/src/router/index.tsx`
- Create: `frontend/src/layouts/MainLayout.tsx`

- [ ] **Step 1: Initialize React app files**

Use Vite + React + TypeScript with Ant Design.

- [ ] **Step 2: Add basic routes**

Routes:

- `/`
- `/audit/events`
- `/rules`

- [ ] **Step 3: Run frontend build**

Run:

```bash
cd frontend
npm install
npm run build
```

Expected: build exits 0.

## Task 10: Audit Events Frontend

**Files:**
- Create: `frontend/src/api/client.ts`
- Create: `frontend/src/api/audit.ts`
- Create: `frontend/src/pages/audit-events/AuditEventsPage.tsx`
- Create: `frontend/src/pages/audit-events/EventDetailDrawer.tsx`
- Create: `frontend/src/types/audit.ts`

- [ ] **Step 1: Implement API types**

Define `AuditEvent`, `AuditEventQuery`, and paged response types matching backend API.

- [ ] **Step 2: Implement query page**

Add time range, severity, event type, keyword filters and table pagination.

- [ ] **Step 3: Implement detail drawer**

Show event details and raw JSON.

- [ ] **Step 4: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected: build exits 0.

## Task 11: Rule Management Frontend

**Files:**
- Create: `frontend/src/api/rules.ts`
- Create: `frontend/src/pages/rules/RulesPage.tsx`
- Create: `frontend/src/pages/rules/RuleFormDrawer.tsx`
- Create: `frontend/src/types/rule.ts`

- [ ] **Step 1: Implement rule API types**

Define rule request/response types.

- [ ] **Step 2: Implement rules table**

Show name, event type, severity, enabled status, tags, updated time.

- [ ] **Step 3: Implement create/edit drawer**

Use JSON editor textarea for `matchExpr` in MVP.

- [ ] **Step 4: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected: build exits 0.

## Task 12: End-to-End Local Run

**Files:**
- Modify: `deploy/docker-compose.yaml`
- Modify: `backend/configs/config.example.yaml`

- [ ] **Step 1: Start databases**

Run:

```bash
docker compose -f deploy/docker-compose.yaml up -d postgres clickhouse
```

Expected: containers are running.

- [ ] **Step 2: Apply migrations**

Apply PostgreSQL and ClickHouse migrations using local CLI tools or migration runner.

- [ ] **Step 3: Run backend tests**

Run:

```bash
cd backend
go test ./...
```

Expected: PASS.

- [ ] **Step 4: Run frontend build**

Run:

```bash
cd frontend
npm run build
```

Expected: build exits 0.

- [ ] **Step 5: Run collector against sample file**

Run:

```bash
cd backend
go run ./cmd/audit-server collector --config ./configs/config.example.yaml
```

Expected: sample events are inserted into ClickHouse.

- [ ] **Step 6: Run API**

Run:

```bash
cd backend
go run ./cmd/audit-server api --config ./configs/config.example.yaml
```

Expected: `/healthz` returns `{"status":"ok"}` and audit query API returns inserted events.

