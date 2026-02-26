# Testing Plan: Coverage + TUI E2E

## Short Summary

Increase confidence in `hours` in small, reviewable increments: first improve unit/integration coverage in `cmd` and `internal/ui`, then add deterministic TUI journey tests, and finally add one Linux-only PTY smoke test plus CI wiring for long-term stability.

## Scope and Constraints

- Interactive E2E runs on Linux CI only.
- Prefer deterministic tests (fixed time provider, seeded DB, isolated temp dirs).
- Keep PRs small (1-3 backlog items each).
- Optional items must not block core delivery.

## Backlog IDs

- `T-001` Coverage baseline
- `T-002` Test matrix definition
- `T-010` root pre-run and env/flag precedence
- `T-011` theme command error paths
- `T-012` error output behavior
- `T-020` navigation/view transitions
- `T-021` task/tracking model flows
- `T-022` task log operations
- `T-023` resize/viewport edge cases
- `T-030` task log renderer coverage
- `T-031` report renderer coverage
- `T-032` stats/active renderer coverage
- `T-040` journey test harness
- `T-041` journey: create -> track -> stop
- `T-042` journey: edit -> move -> lifecycle
- `T-043` optional journey snapshots
- `T-050` PTY smoke harness (Linux only)
- `T-051` PTY core user flow (Linux only)
- `T-060` Linux CI job for interactive tests
- `T-061` stability controls
- `T-070` testing guide docs
- `T-071` optional coverage trend/gate

## Baseline Coverage (T-001)

Captured on: 2026-02-26

| Package | Coverage | Status |
|---------|----------|--------|
| `github.com/dhth/hours` | 0.0% | No tests |
| `github.com/dhth/hours/cmd` | 1.4% | Critical hotspot |
| `github.com/dhth/hours/internal/common` | No test files | Hotspot |
| `github.com/dhth/hours/internal/persistence` | 54.6% | Moderate |
| `github.com/dhth/hours/internal/types` | 68.2% | Moderate |
| `github.com/dhth/hours/internal/ui` | 14.6% | Critical hotspot |
| `github.com/dhth/hours/internal/ui/theme` | 66.4% | Moderate |
| `github.com/dhth/hours/internal/utils` | 100.0% | Excellent |
| `github.com/dhth/hours/tests/cli` | 0.0% | Test infrastructure |
| `github.com/dhth/hours/tests/cli/themes` | No statements | Theme validation tests |

### Hotspots (Priority Order)

1. **`cmd` package (1.4%)** - All command handling, flag/env parsing, error output
2. **`internal/ui` package (14.6%)** - All TUI state management, view rendering, user interactions
3. **`internal/common` (no tests)** - Shared utilities and types
4. **Root package (0.0%)** - Main entry point

## Test Matrix (T-002)

| Test Layer | Location | Command | Environment | Notes |
|------------|----------|---------|-------------|-------|
| Unit/Integration | `*/...` | `go test ./...` | All platforms | Standard Go tests |
| Command Coverage | `cmd/*_test.go` | `go test ./cmd/...` | All platforms | Flag/env precedence, error paths |
| UI State Tests | `internal/ui/*_test.go` | `go test ./internal/ui/...` | All platforms | Deterministic with mock time/DB |
| Renderer Tests | `internal/ui/*_test.go` | `go test ./internal/ui/...` | All platforms | Fixed DB fixtures |
| Journey Tests | `internal/ui/journey_test.go` | `go test ./internal/ui/...` | All platforms | In-process E2E, deterministic |
| PTY Smoke Tests | `tests/cli/tui_smoke_test.go` | `go test ./tests/cli/...` | **Linux only** | True interactive TUI testing |

### Linux-Only Interactive Test Policy

- PTY smoke tests (`T-050`, `T-051`) require pseudo-terminal support
- These tests use `github.com/creack/pty` or similar which requires Linux
- All other test layers run on all platforms (macOS, Linux, Windows)
- CI will have a dedicated Linux job for interactive tests

## Execution Plan (Agent-Oriented)

### Phase 1 - Baseline and Guardrails

1. `T-001` Capture current coverage
   - Run: `go test -cover ./...`
   - Record package-level coverage in this file.
   - Output: baseline table + hotspot list.
2. `T-002` Finalize test matrix
   - Define where each test layer runs.
   - Confirm Linux-only scope for interactive tests.

Done criteria:
- Baseline and matrix are committed and reproducible.

### Phase 2 - Command Layer Coverage

3. `T-010` Add `cmd/root.go` behavior tests
   - Cases: invalid db extension, env/flag precedence, period validation paths.
   - Target files: `cmd/root_test.go` (new/expand).
4. `T-011` Add `cmd/themes.go` error path tests
   - Cases: invalid theme names, filesystem write/create failures.
   - Target files: `cmd/themes_test.go` (new/expand).
5. `T-012` Add `cmd/errors.go` message tests
   - Assert user-facing stderr for known typed errors.
   - Target files: `cmd/errors_test.go` (new).

Done criteria:
- Main error and precedence branches in `cmd` are covered.

### Phase 3 - UI State and Action Coverage

6. `T-020` Navigation and view transitions
   - Cases: `1/2/3`, tab navigation, `?` help toggle, back actions (`esc/q`).
   - Target files: `internal/ui/update_test.go` and/or `internal/ui/handle_test.go`.
7. `T-023` Resize and viewport edge cases
   - Cases: below min dims, recovery from insufficient dims, viewport scroll guards.
8. `T-021` Task and tracking flow tests
   - Cases: create/update task request flow, start/stop tracking, quick switch.
9. `T-022` Task log operation tests
   - Cases: delete TL, move TL, deactivate/reactivate task, key failures.

Done criteria:
- Key state transitions and high-risk mutations are explicitly asserted.

### Phase 4 - Report/Log/Stats Integration Coverage

10. `T-030` Test `RenderTaskLog` / `getTaskLog`
    - Cases: empty state, non-empty state, plain/styled, interactive day limit guard.
    - Target files: `internal/ui/log_test.go` (new).
11. `T-031` Test `RenderReport` / `getReport` / `getReportAgg`
    - Cases: no entries, multi-day entries, aggregate and non-aggregate totals.
    - Target files: `internal/ui/report_test.go` (new).
12. `T-032` Test `RenderStats` / `getStats` / `ShowActiveTask`
    - Cases: all/range modes, interactive constraints, active template substitution.
    - Target files: `internal/ui/stats_test.go`, `internal/ui/active_test.go` (new).

Done criteria:
- Renderers are covered with deterministic DB fixtures and stable assertions.

### Phase 5 - Deterministic Journey Tests (In-Process E2E)

13. `T-040` Build journey harness
    - Helper capabilities:
      - seed test DB
      - initialize model with fixed time provider
      - inject key message sequence
      - assert model + DB end state
    - Target file: `internal/ui/journey_test.go`.
14. `T-041` Journey flow A
    - Flow: create task -> start tracking -> stop/save -> verify log + task secs.
15. `T-042` Journey flow B
    - Flow: edit log -> move log -> deactivate/reactivate -> verify totals/state.
16. `T-043` Optional minimal journey snapshots
    - Keep snapshot count intentionally low.

Done criteria:
- At least two high-value journeys pass deterministically in normal `go test`.

### Phase 6 - True Interactive Smoke (Linux Only)

17. `T-050` PTY harness
    - Launch built binary in pseudo-terminal on Linux.
    - Send controlled key sequence and verify clean exit.
    - Target file: `tests/cli/tui_smoke_test.go`.
18. `T-051` PTY core flow
    - Extend harness with one meaningful interaction before quit.

Done criteria:
- Linux-only smoke test is stable and skipped on non-Linux.

### Phase 7 - CI Integration and Stability

19. `T-060` Add dedicated Linux CI job
    - Run journey tests + PTY smoke tests in separate Linux job.
    - Target files: `.github/workflows/pr.yml`, `.github/workflows/main.yml`.
20. `T-061` Stability controls
    - Add explicit timeouts and useful failure diagnostics/artifacts.

Done criteria:
- Interactive test failures are diagnosable and isolated from baseline unit suite.

### Phase 8 - Docs and Follow-Up

21. `T-070` Add testing guide
    - Document local commands and when to run each layer.
22. `T-071` Optional coverage trend/gate
    - Start non-blocking trend output; only gate later if stable.

Done criteria:
- Contributors can run and debug all test layers with minimal ramp-up.

## PR Sequence

1. PR-1: `T-001`, `T-002`
2. PR-2: `T-010`, `T-011`, `T-012`
3. PR-3: `T-020`, `T-023`
4. PR-4: `T-021`
5. PR-5: `T-022`
6. PR-6: `T-030`, `T-031`, `T-032`
7. PR-7: `T-040`
8. PR-8: `T-041`, `T-042` (+ optional `T-043`)
9. PR-9: `T-050`, `T-051`
10. PR-10: `T-060`, `T-061`
11. PR-11: `T-070` (+ optional `T-071`)

## Tracking Table

| ID | Task | Status | PR | Notes |
|---|---|---|---|---|
| T-001 | Coverage baseline | **done** | PR-1 | Captured in "Baseline Coverage" section above |
| T-002 | Test matrix | **done** | PR-1 | Defined in "Test Matrix" section above |
| T-010 | root pre-run/env/flag | **done** | PR-2 | Added tests for invalid DB extension, env/flag precedence, default values, subcommands |
| T-011 | themes error paths | todo | PR-2 | |
| T-012 | error output behavior | todo | PR-2 | |
| T-020 | navigation transitions | todo | PR-3 | |
| T-023 | resize/viewport edge cases | todo | PR-3 | |
| T-021 | task/tracking flows | todo | PR-4 | |
| T-022 | task log ops | todo | PR-5 | |
| T-030 | RenderTaskLog/getTaskLog | todo | PR-6 | |
| T-031 | RenderReport/getReport/getReportAgg | todo | PR-6 | |
| T-032 | RenderStats/getStats/ShowActiveTask | todo | PR-6 | |
| T-040 | journey harness | todo | PR-7 | |
| T-041 | journey flow A | todo | PR-8 | |
| T-042 | journey flow B | todo | PR-8 | |
| T-043 | optional journey snapshots | todo | PR-8 | optional |
| T-050 | PTY harness (Linux only) | todo | PR-9 | |
| T-051 | PTY core flow (Linux only) | todo | PR-9 | |
| T-060 | Linux CI job | todo | PR-10 | |
| T-061 | stability controls | todo | PR-10 | |
| T-070 | testing guide docs | todo | PR-11 | |
| T-071 | optional coverage trend/gate | todo | PR-11 | optional |
