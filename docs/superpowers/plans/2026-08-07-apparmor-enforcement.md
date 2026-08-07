# AppArmor Enforcement Implementation Plan

**Goal:** Deliver one production-shaped sensitive-file enforcement path using AppArmor from the existing root Collector.

**Architecture:** Reuse enforcement policy `definition` JSON, generate one deterministic managed sudo profile per host, activate it transactionally with `apparmor_parser`, and report real per-policy outcomes. Tetragon enforcement YAML generation is removed from the supported UI path.

**Tech stack:** Go, AppArmor parser, React/TypeScript, existing DiTing enforcement API.

## Tasks

1. Add failing Go tests for capability detection, definition validation, deterministic profile generation, unchanged synchronization, parser failure, and rollback.
2. Implement an AppArmor manager in `backend/internal/collector` with injected command execution for tests.
3. Change collector policy DTO and synchronization to consume template, mode, enabled, and definition fields and report each policy independently.
4. Add collector configuration for enabling AppArmor and selecting its managed data directory; discover `apparmor_parser` from `PATH`.
5. Restrict the frontend enforcement experience to the supported sensitive-file template and replace the Tetragon YAML preview with an AppArmor capability summary.
6. Add AppArmor denial parsing and map denial events to the existing blocked-policy notification tags.
7. Run focused Go tests, full backend tests, frontend tests, and the production frontend build.

## Acceptance

- No Tetragon restart occurs when an enforcement policy changes.
- A malformed or unsupported policy cannot be reported as deployed.
- A failed AppArmor reload retains the previous profile.
- Direct root is not attached to the managed sudo profile.
- sudo and its inherited child processes receive protected-path deny rules.
- Existing audit collection continues when AppArmor is unavailable.

