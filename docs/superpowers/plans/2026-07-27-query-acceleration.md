# Query Acceleration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a unified query acceleration layer for ClickHouse-backed audit investigation pages, host audit pages, dashboards, and related statistics so large datasets return faster while lowering CPU load.

**Architecture:** Keep raw event detail in ClickHouse `audit_events`, move recurring list/group/stat queries to dedicated query services, hourly aggregate tables, and Redis response caching. Use a mixed hot/cold strategy: recent hot windows read raw events, historical windows read aggregate tables, and cross-window requests are split and merged by the backend.

**Tech Stack:** Go HTTP API, ClickHouse MergeTree and materialized views, PostgreSQL for configuration, Redis for short TTL response caching and singleflight-style duplicate request suppression, React/Ant Design frontend.

---

## Current Bottlenecks

- `frontend/src/pages/audit-events/AuditEventsPage.tsx` shows "操作日志调查" but reads ClickHouse audit events through `/audit/events`.
- The page defaults to seven days and multiplies page size by `10` so it can group raw events in the browser.
- `backend/internal/clickhouse/audit_repository.go` executes a list query plus a `count()` query for every `/audit/events` request.
- Host audit, user audit, command audit, rule hit, dashboard, and behavior pages query raw `audit_events` with repeated `GROUP BY`, `ORDER BY`, `uniqExact`, and string search operations.
- Opening a host drawer triggers multiple parallel heavy queries.
- Redis is not currently integrated, so repeated default page loads always hit ClickHouse.

## Performance Targets

- Dashboard and top-level aggregate pages: P95 under `1s`.
- Operation investigation list: P95 under `1s` for default filters.
- Host detail profile drawer: P95 under `2s`.
- Raw event detail drilldown first page: P95 under `2s`.
- Redis cache hit latency: usually under `100ms`.
- Large exports and deep pagination must not run as synchronous online queries.

## File Structure

- Create `backend/internal/queryguard/queryguard.go`
  - Owns query limits, time range validation, timeout wrapping, and include-total rules.
- Create `backend/internal/cache/cache.go`
  - Defines small cache interfaces used by repositories without coupling business code to Redis.
- Create `backend/internal/cache/memory.go`
  - Provides an in-memory fallback for tests and deployments without Redis.
- Create `backend/internal/cache/redis.go`
  - Provides Redis implementation after Redis configuration is added.
- Create `backend/internal/cache/singleflight.go`
  - Suppresses duplicate concurrent requests for the same cache key.
- Modify `backend/internal/config/config.go`
  - Add Redis and query acceleration configuration.
- Modify `backend/configs/config.example.yaml`
  - Document Redis and query acceleration defaults.
- Modify `backend/cmd/audit-server/main.go`
  - Wire cache, query guard, and repositories.
- Modify `backend/internal/audit/query.go`
  - Add `include_total`, cursor fields if needed, and enforce query limits.
- Modify `backend/internal/audit/handler.go`
  - Return `hasMore`, optional `total`, and query metadata.
- Modify `backend/internal/clickhouse/audit_repository.go`
  - Split raw event list and count paths, add lightweight list fields, and support operation grouping queries.
- Create `backend/internal/audit/operation.go`
  - Defines operation group request and response models.
- Create `backend/internal/audit/operation_handler.go`
  - Adds `/api/v1/audit/operations` endpoint.
- Create `backend/internal/clickhouse/operation_repository.go`
  - Queries grouped operations from ClickHouse.
- Modify `backend/internal/server/server.go`
  - Register operation grouping and profile endpoints.
- Create `backend/migrations/clickhouse/003_query_acceleration.sql`
  - Adds projections, skip indexes, and aggregate tables/materialized views.
- Modify `backend/internal/clickhouse/stats_repository.go`
  - Route common stats to aggregate tables and fall back to raw table for hot windows.
- Create `backend/internal/clickhouse/accelerated_stats_repository.go`
  - Keeps aggregate-query SQL isolated from raw stats SQL.
- Modify `frontend/src/api/audit.ts`
  - Add operation-group API.
- Modify `frontend/src/types/audit.ts`
  - Add operation-group types and pagination metadata.
- Modify `frontend/src/pages/audit-events/AuditEventsPage.tsx`
  - Stop multiplying raw page size, load grouped operations from backend, and drill down to raw events only when needed.
- Modify `frontend/src/pages/hosts/HostAuditPage.tsx`
  - Reduce drawer fan-out by calling a consolidated profile endpoint when available.
- Modify or add tests under `backend/internal/...`
  - Cover query guard, cache keys, audit list without total, operation grouping SQL, and aggregate stats routing.

## Phase 1: Query Governance and Raw Event List Speed

### Task 1: Add Query Guard

- [ ] Create `backend/internal/queryguard/queryguard.go`.
- [ ] Implement a `Limits` struct with `MaxRange`, `DefaultTimeout`, `MaxPageSize`, `MaxExportRows`, and `HotWindow`.
- [ ] Add functions:
  - `NormalizePage(page int) int`
  - `NormalizePageSize(pageSize int, max int) int`
  - `ValidateTimeRange(start, end time.Time, maxRange time.Duration) error`
  - `WithTimeout(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc)`
- [ ] Unit test invalid time ranges, oversized page sizes, and timeout wrapping.

### Task 2: Make Audit Event Count Optional

- [ ] Modify `backend/internal/audit/query.go` to parse `include_total=true`.
- [ ] Add `IncludeTotal bool` to `audit.Query`.
- [ ] Modify `backend/internal/clickhouse/audit_repository.go` so `ListEvents` skips `countEvents` unless `IncludeTotal` is true.
- [ ] Return `hasMore` by querying `page_size + 1` rows and trimming the extra row.
- [ ] Update `backend/internal/audit/handler.go` response to include `hasMore`.
- [ ] Update tests to verify default `/audit/events` does not emit `SELECT count()`.

### Task 3: Reduce Raw Event Payload

- [ ] Keep list queries using `auditEventListSelectFields()` without `rule_matches` and `raw_event`.
- [ ] Verify detail drawer still calls `GetEvent` for full fields when needed.
- [ ] Add a regression test that list SQL uses `'' AS rule_matches, '' AS raw_event`.

### Task 4: Add ClickHouse Detail Query Improvements

- [ ] Create `backend/migrations/clickhouse/003_query_acceleration.sql`.
- [ ] Add projections or secondary indexes for common filters:
  - host identity plus `event_time`
  - login user plus `event_time`
  - exec user plus `event_time`
  - destination IP/port plus `event_time`
  - file path plus `event_time`
- [ ] Add token/bloom skip indexes only where useful for `cmdline`, `process_name`, and `file_path` keyword searches.
- [ ] Run ClickHouse migration in a staging dataset before production.

## Phase 2: Backend Operation Grouping for "操作日志调查"

### Task 5: Add Operation Group Models

- [ ] Create `backend/internal/audit/operation.go`.
- [ ] Define `OperationGroup` with:
  - `GroupID string`
  - `Representative Event`
  - `EventCount uint64`
  - `EventTypes []string`
  - `FilePaths []string`
  - `Tags []string`
  - `MaxSeverity string`
  - `FirstSeen time.Time`
  - `LastSeen time.Time`
- [ ] Define `OperationQuery` using the same filters as `audit.Query`.

### Task 6: Add ClickHouse Operation Repository

- [ ] Create `backend/internal/clickhouse/operation_repository.go`.
- [ ] Group by the current frontend grouping semantics:
  - second-level event time bucket
  - stable host identity
  - namespace
  - pod name
  - login user
  - execution user
  - process name
  - command line
- [ ] Use `argMax` to select representative fields by latest `event_time`.
- [ ] Query `LIMIT page_size + 1` and return `hasMore`.
- [ ] Avoid exact total by default.
- [ ] Add SQL-building tests for filters, grouping keys, and no-count default.

### Task 7: Add Operation Group API

- [ ] Create `backend/internal/audit/operation_handler.go`.
- [ ] Register `GET /api/v1/audit/operations` in `backend/internal/server/server.go`.
- [ ] Return:
  - `items`
  - `page`
  - `pageSize`
  - `hasMore`
  - optional `total` only when requested
- [ ] Add handler tests for successful list, invalid time range, and repository errors.

### Task 8: Update Operation Investigation Frontend

- [ ] Modify `frontend/src/api/audit.ts` to add `queryAuditOperations`.
- [ ] Modify `frontend/src/types/audit.ts` to add `AuditOperationGroup`.
- [ ] Modify `frontend/src/pages/audit-events/AuditEventsPage.tsx`:
  - remove `rawPageMultiplier`
  - call `/audit/operations` for the table
  - show `eventCount` as "明细数"
  - use representative event for columns
  - fetch related raw events only when a row is expanded or detail drawer opens
- [ ] Keep export using raw `/audit/events/export`, but apply export limits or async export in a later phase.
- [ ] Build frontend and verify no type errors.

## Phase 3: Aggregate Tables for Statistics Pages

### Task 9: Create Hourly Aggregate Tables

- [ ] Extend `backend/migrations/clickhouse/003_query_acceleration.sql`.
- [ ] Add:
  - `audit_overview_hourly`
  - `audit_host_stats_hourly`
  - `audit_user_stats_hourly`
  - `audit_command_stats_hourly`
  - `audit_rule_hit_stats_hourly`
  - `audit_host_behavior_hourly`
- [ ] Use `AggregatingMergeTree` where uniqueness and min/max states are needed.
- [ ] Partition by month and order by dimensions plus hour.
- [ ] Add materialized views from `audit_events` into each aggregate table.

### Task 10: Add Aggregate Stats Repository

- [ ] Create `backend/internal/clickhouse/accelerated_stats_repository.go`.
- [ ] Implement aggregate-table versions for:
  - overview
  - event trend
  - top commands
  - top hosts
  - top namespaces
  - command stats
  - user audits
  - host audits
  - host users
  - host behavior
  - rule hits
- [ ] For requests crossing the hot window, split into:
  - cold part from aggregate tables
  - hot part from raw `audit_events`
  - merge in Go by dimension key
- [ ] Add tests for cold-only, hot-only, and mixed-window routing.

### Task 11: Consolidate Host Detail Queries

- [ ] Add a backend endpoint such as `GET /api/v1/stats/hosts/profile`.
- [ ] Return high-risk commands, risk timeline, host users, and host behavior in one response.
- [ ] Reuse aggregate tables for behavior and user distribution.
- [ ] Query raw event details only for small top-N timelines.
- [ ] Modify `frontend/src/pages/hosts/HostAuditPage.tsx` to use the consolidated endpoint.

## Phase 4: Redis Cache and Duplicate Request Suppression

### Task 12: Add Cache Interfaces

- [ ] Create `backend/internal/cache/cache.go`.
- [ ] Define `Cache` with `Get`, `Set`, and `DeletePrefix`.
- [ ] Define stable query cache key helpers using route name, normalized query, user scope if needed, and response version.
- [ ] Unit test that logically identical query parameter ordering generates the same cache key.

### Task 13: Add Redis Implementation

- [ ] Add Redis configuration in `backend/internal/config/config.go`.
- [ ] Add example config in `backend/configs/config.example.yaml`.
- [ ] Implement Redis client in `backend/internal/cache/redis.go`.
- [ ] If Redis is disabled or unavailable at startup, fall back to no-op or memory cache depending on config.
- [ ] Add tests for disabled cache behavior and serialization.

### Task 14: Cache Query Responses

- [ ] Wrap `/stats/*`, `/audit/operations`, and first-page `/audit/events` responses.
- [ ] TTL defaults:
  - Dashboard overview/trend/top: `10s`
  - operation groups: `15s`
  - host/user/command/rule stats: `30s`
  - host profile: `30s`
  - raw event first page: `5s`
- [ ] Do not cache exports.
- [ ] Do not cache requests with very large `page` or unsupported deep pagination.

### Task 15: Add Singleflight

- [ ] Create `backend/internal/cache/singleflight.go`.
- [ ] Ensure concurrent identical cache misses share one ClickHouse request.
- [ ] Add tests with multiple goroutines and verify the underlying loader runs once.

## Phase 5: UX and Operational Controls

### Task 16: Frontend Query Limits and Feedback

- [ ] Change default ranges on heavy pages to `24h` where appropriate.
- [ ] Keep user-selectable seven-day or custom ranges.
- [ ] Add clear messages when a query exceeds online limits.
- [ ] Add `hasMore`-based pagination display for endpoints without exact totals.

### Task 17: Slow Query Observability

- [ ] Log query route, normalized time range, cache hit/miss, duration, row count, and whether aggregate/raw/mixed path was used.
- [ ] Add warning logs for queries above `2s`.
- [ ] Add counters for cache hit rate and slow query rate if metrics infrastructure exists.

### Task 18: Backfill and Deployment Runbook

- [ ] Document ClickHouse migration order.
- [ ] Backfill aggregate tables from existing `audit_events` in monthly or daily batches.
- [ ] Verify row counts for aggregate tables against raw table for selected days.
- [ ] Roll out repository routing behind a config flag.
- [ ] Enable Redis cache after aggregate queries are validated.

## Verification Plan

- [ ] Run backend unit tests with `go test ./...`.
- [ ] Run frontend build with `npm run build` from `frontend`.
- [ ] On staging data, compare old and new stats results for a fixed one-day range.
- [ ] Measure these endpoints before and after:
  - `/api/v1/audit/operations`
  - `/api/v1/audit/events`
  - `/api/v1/stats/overview`
  - `/api/v1/stats/hosts`
  - `/api/v1/stats/hosts/profile`
  - `/api/v1/stats/commands`
  - `/api/v1/stats/users`
  - `/api/v1/stats/rules`
- [ ] Confirm CPU drops on repeated page loads and Redis hit latency is visible in logs.

## Rollout Order

1. Ship query guard and optional totals.
2. Ship `/audit/operations` and update the "操作日志调查" page.
3. Add ClickHouse aggregate tables and materialized views.
4. Route stats pages to aggregate tables behind a config flag.
5. Add Redis response caching and duplicate request suppression.
6. Consolidate host detail drawer queries.
7. Enable async export or strict export limits for large ranges.

## Risks

- Aggregate-table schema mistakes can create incorrect counts. Mitigate by comparing old and new query results on fixed staging windows.
- Materialized views do not backfill old rows automatically. Mitigate with explicit backfill scripts.
- Redis can hide fresh writes briefly. Keep TTL short and use raw hot-window reads for near-real-time visibility.
- Query split/merge can produce duplicate dimension rows. Normalize merge keys in one helper and unit test each stat type.
- ClickHouse skip indexes help only when query patterns match the index. Validate with `EXPLAIN indexes = 1` on production-like data.

