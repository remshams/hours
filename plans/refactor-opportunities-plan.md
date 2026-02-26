# Refactor Opportunities Plan

This document captures sensible refactors identified after reviewing the current codebase, with an emphasis on simplification, duplication reduction, and safer boundaries.

## Goals

- Reduce complexity in UI state handling and command wiring.
- Eliminate repeated rendering and form-handling logic.
- Improve maintainability of persistence queries and theme validation.
- Keep behavior stable by sequencing changes from low-risk to high-risk.

## Top Opportunities (Ranked)

1. **Decompose UI state machine in `internal/ui/update.go`**
   - Current `Model.Update` is large and mixes: global key handling, form key handling, view transitions, and message handling.
   - Proposed split:
     - `handleGlobalKeys` (quit, resize, global shortcuts)
     - `handleFormKeys` (submit/escape/tab/time-shifts)
     - `handleListKeys` (task/log/inactive/move/help interactions)
     - `handleMsg` (typed message handlers)
   - Expected benefit: easier reasoning about keybindings and lower regression risk when adding new views.

2. **Unify report rendering paths in `internal/ui/report.go`**
   - `getReport` and `getReportAgg` are structurally very similar.
   - Proposed approach:
     - Introduce one shared report grid renderer.
     - Inject data fetch function and row adapter for non-aggregated vs aggregated rows.
   - Expected benefit: less duplication and fewer inconsistencies over time.

3. **Extract shared table factory used by report/log/stats**
   - Table writer configuration is repeated in:
     - `internal/ui/report.go`
     - `internal/ui/log.go`
     - `internal/ui/stats.go`
   - Proposed helper:
     - `newRecordsTable(buffer, styles, headers, footer)` with shared `tablewriter.Config` and renderer setup.
   - Expected benefit: single place to change rendering behavior.

4. **Break up `NewRootCommand` in `cmd/root.go`**
   - Current function mixes DB setup, env behavior, command construction, and flag wiring.
   - Proposed split:
     - `newGenerateCmd`, `newReportCmd`, `newLogCmd`, `newStatsCmd`, `newActiveCmd`, `newThemesCmd`
     - shared pre-run helper and shared flag registration helpers.
   - Expected benefit: much clearer command boundaries and easier extension.

5. **Remove repeated flag registration logic in `cmd/root.go`**
   - Repeated `dbpath`, `theme`, and `task-status` flag registrations.
   - Proposed helpers:
     - `addDBPathFlag(cmd, &dbPath, defaultDBPath)`
     - `addThemeFlag(cmd, &themeName, defaultThemeName, usage)`
     - `addTaskStatusFlag(cmd, &taskStatusStr)`
   - Expected benefit: fewer copy/paste mistakes and easier updates.

6. **Consolidate task-log form workflows in `internal/ui/handle.go` + `internal/ui/view.go`**
   - Repeated comment extraction (`textarea -> *string`) and repeated focus/setup logic.
   - Proposed helpers:
     - `commentPtrFromInput(textarea.Model) *string`
     - setup/focus helpers for `editActiveTL`, `finishActiveTL`, `manualTasklogEntry`, `editSavedTL`.
   - Expected benefit: less branching and lower cognitive load.

7. **Refactor repeated list guard/cast patterns in UI handlers**
   - Repeated patterns around:
     - filtered list checks
     - selected item casting
     - generic error message setting
   - Proposed helpers for safe selection per list type.
   - Expected benefit: less boilerplate and more consistent errors.

8. **Make theme color validation data-driven in `internal/ui/theme/theme.go`**
   - `getInvalidColors` is a long chain of manual checks.
   - Proposed approach:
     - map-like list of `{name, value}` and loop.
   - Expected benefit: easier to maintain and harder to forget fields.

9. **Reduce duplication in persistence query scanning and tx wrappers**
   - Repeated scan loops and very similar transaction wrappers in `internal/persistence/queries.go`.
   - Proposed approach:
     - shared row scan helpers for task/tasklog/report rows.
     - one generic transaction helper replacing `runInTxAndReturnID`, `runInTx`, `runInTxAndReturnA`.
   - Expected benefit: fewer subtle error-handling differences.

10. **Consolidate list model setup in `internal/ui/initial.go`**
    - Active/inactive/tasklog/target list setup repeats title/help/keymap/style wiring.
    - Proposed helper builder to initialize list models with shared defaults and small overrides.
    - Expected benefit: cleaner initialization path.

## Suggested Execution Plan (PR-Sized)

### PR 1: Safety Net

- Add characterization tests around high-risk behavior:
  - key navigation + submit behavior in UI forms.
  - persistence accounting around deleting/editing/moving logs.
- Goal: preserve behavior while refactoring internals.

### PR 2: Quick Wins (Low Risk)

- Extract comment parsing helper.
- Extract list guard/cast helpers.
- Extract list initialization helper in `initial.go`.
- No functional changes expected.

### PR 3: Rendering Unification

- Add shared table factory for report/log/stats.
- Refactor `getReport` + `getReportAgg` to one shared renderer pipeline.
- Keep snapshot tests updated only if formatting changes are intentional.

### PR 4: CLI Command Modularization

- Split `NewRootCommand` into subcommand builder functions.
- Introduce shared flag helpers and central pre-run behavior.
- Preserve existing command UX/flags exactly.

### PR 5: Persistence Simplification

- Introduce reusable scan helpers.
- Consolidate transaction helpers.
- Keep SQL semantics unchanged.

### PR 6: UI Update Decomposition (Highest Risk)

- Split `Model.Update` into focused handlers by concern.
- Verify keybindings and view transitions comprehensively.

### PR 7: Theme Validation Cleanup

- Convert `getInvalidColors` to data-driven field validation.
- Keep error messages and field names stable.

## Caution Areas

- UI snapshots are sensitive to rendering/layout changes (`internal/ui/view_test.go`).
- `secs_spent` invariants in persistence are critical; refactors must preserve exact accounting.
- CLI snapshots depend on exact stderr/stdout wording.
- `MoveTaskLog` path should get explicit behavior coverage before deep persistence refactors.

## Definition of Done (Per Refactor PR)

- All existing tests pass.
- No user-visible behavior change unless explicitly intended.
- New helpers reduce net duplication.
- Public CLI contract (flags, errors, outputs) remains stable.
