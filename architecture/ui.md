# Architecture

## 1. Full Application

The application is a single Go binary with two modes: an **interactive TUI** (default) and **non-interactive CLI subcommands** (`report`, `log`, `stats`, `active`). All data is persisted in a local SQLite file.

### Package Dependency Tree

```
main → cmd → ui → persistence → types → utils
                → types
             → common
             → ui/theme
```

No circular dependencies. `utils` and `common` are leaves with no internal deps.

### Database Schema

Three tables in SQLite (`modernc.org/sqlite`, pure-Go driver):

| Table | Key Columns | Notes |
|---|---|---|
| `task` | `id`, `summary`, `secs_spent`, `active` | `secs_spent` is a **denormalized total** kept in sync on every write |
| `task_log` | `id`, `task_id`, `begin_ts`, `end_ts`, `secs_spent`, `comment`, `active` | `end_ts` is NULL while a session is open; `active=1` means currently tracking |
| `db_versions` | `id`, `version`, `created_at` | Append-only migration log |

A DB trigger (`prevent_duplicate_active_insert`) enforces that only **one** `task_log` row can have `active=1` at any time.

### Diagram

```mermaid
graph TD
    subgraph User
        CLI["CLI invocation\n$ hours [subcommand]"]
    end

    subgraph cmd["cmd/ (cobra)"]
        root["root command\nLaunches interactive TUI"]
        report["report / log / stats\nNon-interactive output"]
        active["active\nPrints current task"]
        gen["gen\nSeeds fake data"]
        themes["themes\nadd / list / show-config"]
    end

    subgraph ui["internal/ui (BubbleTea)"]
        Model["Model\nFull interactive TUI"]
        RecordsModel["recordsModel\nPaginated report viewer"]
        Renderers["Standalone renderers\nreport.go / log.go / stats.go"]
        Theme["theme/\nJSON theme loading\n8 built-in palettes"]
    end

    subgraph types["internal/types"]
        DomainTypes["Task, TaskLogEntry\nActiveTaskDetails\nTaskReportEntry\nDateRange, TimeProvider"]
        TimeParsing["Time parsing\nperiod → DateRange\nshiftTime, HumanizeDuration"]
    end

    subgraph persistence["internal/persistence"]
        queries["queries.go\n20+ CRUD functions"]
        migrations["migrations.go\nVersion-tracked migrations"]
        opener["open.go\nGetDB() — single connection"]
    end

    subgraph db["SQLite file (~/.local/share/hours/hours.db)"]
        task_tbl[("task")]
        task_log_tbl[("task_log")]
        db_versions_tbl[("db_versions")]
        trigger["TRIGGER\nprevent_duplicate_active_insert"]
    end

    subgraph utils["internal/utils"]
        strhelpers["String helpers\nRightPadTrim, Trim…"]
    end

    CLI --> cmd
    root --> Model
    report & active & gen --> RecordsModel & Renderers
    themes --> Theme

    Model --> queries
    RecordsModel --> queries
    Renderers --> queries

    queries --> task_tbl & task_log_tbl
    migrations --> db_versions_tbl
    trigger -.->|enforces single active| task_log_tbl
    opener --> db

    Model & RecordsModel & Renderers --> DomainTypes & TimeParsing
    queries --> DomainTypes
    DomainTypes --> strhelpers
```

---

# UI Architecture

The `internal/ui` package implements two BubbleTea models:

- **`Model`** — the main interactive TUI launched by `RenderUI()`
- **`recordsModel`** — a lightweight pagination shell for the non-interactive `report`/`log`/`stats` subcommands

## File Overview

| File | Purpose |
|---|---|
| `model.go` | `Model` struct, all enums/types, `Init()` |
| `initial.go` | `InitialModel()`, `initialRecordsModel()` constructors |
| `msgs.go` | All 17 `tea.Msg` types |
| `cmds.go` | DB → `tea.Cmd` factories |
| `handle.go` | All handler & `getCmdTo…` methods (966 lines) |
| `update.go` | `Update()` dispatch only (508 lines) |
| `view.go` | `View()` rendering only (404 lines) |
| `styles.go` | `Style` struct, `NewStyle()` |
| `help.go` | `getHelpText()` |
| `report.go` / `log.go` / `stats.go` | Standalone table renderers |
| `theme/` | `Theme` struct + 8 built-in palettes |

**Naming conventions in `handle.go`:**
- `getCmdTo…` — validates input, returns a `tea.Cmd` (issues a DB command)
- `handleRequest…` / `handle…` — mutates model state, no DB interaction

---

## Diagrams

### 2. UI Package Architecture

```mermaid
graph TD
    subgraph Entry["Entry Points"]
        RenderUI["RenderUI()"]
        RenderReport["RenderReport()"]
        RenderTaskLog["RenderTaskLog()"]
        RenderStats["RenderStats()"]
    end

    subgraph Models["BubbleTea Models"]
        Model["Model\n(main interactive TUI)"]
        RecordsModel["recordsModel\n(report/log/stats pagination)"]
    end

    subgraph Core["Core Files"]
        model_go["model.go\nstruct + enums + Init()"]
        msgs_go["msgs.go\n17 message types"]
        cmds_go["cmds.go\nDB → tea.Cmd"]
        handle_go["handle.go\nAll handler logic"]
        update_go["update.go\nUpdate() dispatch"]
        view_go["view.go\nView() rendering"]
    end

    subgraph Components["Embedded Bubbles Components"]
        list["list.Model ×4\nactive tasks, inactive tasks\ntask log, move target"]
        viewport["viewport.Model ×2\nhelp, TL details"]
        inputs["textinput ×3 + textarea ×1\nbegin TS, end TS, summary, comment"]
    end

    subgraph Renderers["Standalone Renderers"]
        report_go["report.go"]
        log_go["log.go"]
        stats_go["stats.go"]
    end

    RenderUI --> Model
    RenderReport & RenderTaskLog & RenderStats --> RecordsModel

    Model --> model_go & update_go & view_go & handle_go
    RecordsModel --> model_go & update_go & view_go

    handle_go --> cmds_go --> msgs_go
    update_go --> handle_go & msgs_go
    Model --> list & viewport & inputs

    RenderReport --> report_go
    RenderTaskLog --> log_go
    RenderStats --> stats_go
```

---

### 3. Update Dispatch (7 Passes)

```mermaid
flowchart TD
    MSG([tea.Msg]) --> P1

    subgraph P1["Pass 1: Early intercepts"]
        WS{WindowSizeMsg?} -->|yes| HWR[handleWindowResizing]
        WS -->|no| CTRLC{ctrl+c?}
        CTRLC -->|yes| QUIT([Quit])
        CTRLC -->|no| INSUF{insufficientDimensionsView?}
        INSUF -->|q/esc| QUIT
        INSUF -->|other| NOOP([no-op])
        INSUF -->|no| P2
    end

    P2["Pass 2: Decrement message.framesLeft\nClear message when it hits 0"] --> P3

    subgraph P3["Pass 3: Filter intercept"]
        FILT{List in filter mode?} -->|yes| FWDLIST[Forward to list → return early]
        FILT -->|no| P4
    end

    subgraph P4["Pass 4: Form keys"]
        FORMKEY -->|enter / ctrl+s| SUBMIT[getCmdTo… form]
        FORMKEY -->|esc| ESC[handleEscapeInForms]
        FORMKEY -->|tab / shift+tab| TABNAV[goForward/BackwardInView]
        FORMKEY -->|k j K J h l| SHIFT[shiftTime / cursor nav]
    end

    P4 --> P5

    subgraph P5["Pass 5: Form input forwarding"]
        FORMINPUT{Form view active?} -->|taskInputView| FWDTASK[Forward to taskInputs → return early]
        FORMINPUT -->|TL form views| FWDTL[Forward to tLInputs + tLCommentInput → return early]
        FORMINPUT -->|no| P6
    end

    subgraph P6["Pass 6: Navigation + async messages"]
        NAVKEYS["Key → view switch or action\n(q/esc, 1/2/3, ctrl+r, s, S, a, u,\nctrl+d, ctrl+s, ctrl+x, f, d, m, A, ?)"]
        ASYNCMSGS["Async msg → handleXxxMsg()\n(17 message types)"]
    end

    P6 --> P7["Pass 7: Forward msg to active\nlist or viewport sub-model"]
    P7 --> RETURN([return m, cmds])
```

---

### 4. View Rendering

```mermaid
graph TD
    VIEW["View()"] --> CTX["Compute shared context:\n• status bar (message.value)\n• active tracking msg\n• form duration validity\n• submit help text"]

    CTX --> SWITCH

    subgraph SWITCH["Switch on activeView"]
        TLV["taskListView → activeTasksList.View()"]
        TLOGV["taskLogView → taskLogList.View()"]
        TLOGDETV["taskLogDetailsView → tLDetailsVP.View()"]
        INACTV["inactiveTaskListView → inactiveTasksList.View()"]
        FORMS["taskInputView\nfinishActiveTLView\neditActiveTLView\nmanualTasklogEntryView\neditSavedTLView\n→ inline fmt.Sprintf form layout"]
        MOVEV["moveTaskLogView → targetTasksList.View()"]
        HELPV["helpView → helpVP.View()"]
        INSUFDIMV["insufficientDimensionsView → plain text, return early"]
    end

    SWITCH --> FOOTER["Footer: tool name + help indicator + active tracking info"]
    FOOTER --> JOIN["lipgloss.JoinVertical(content, statusBar, footer)"]
    JOIN --> RETURN(["return string"])
```

---

### 5. Async Command / Message Cycle

```mermaid
sequenceDiagram
    participant U as Update()
    participant H as handle.go
    participant C as cmds.go
    participant DB as Database

    U->>H: getCmdToXxx()
    H->>C: returns tea.Cmd
    C->>DB: SQL query (goroutine)
    DB-->>C: result
    C-->>U: XxxMsg via BubbleTea runtime
    U->>H: handleXxxMsg(msg)
    H-->>U: []tea.Cmd (follow-up cmds)
    Note over U,H: e.g. trackingToggledMsg<br/>→ updateTaskRep + fetchTLS
```

---

### 6. View State Transitions

```mermaid
stateDiagram-v2
    [*] --> taskListView : Init()

    taskListView --> taskInputView : a / u
    taskListView --> editActiveTLView : ctrl+s (tracking on)
    taskListView --> manualTasklogEntryView : ctrl+s (tracking off)
    taskListView --> finishActiveTLView : s (stop tracking)

    taskLogView --> taskLogDetailsView : d
    taskLogView --> editSavedTLView : ctrl+s / u
    taskLogView --> moveTaskLogView : m

    taskListView --> taskLogView : 2
    taskListView --> inactiveTaskListView : 3
    taskLogView --> taskListView : 1
    taskLogView --> inactiveTaskListView : 3
    inactiveTaskListView --> taskListView : 1
    inactiveTaskListView --> taskLogView : 2

    taskListView --> helpView : ?
    taskLogView --> helpView : ?
    inactiveTaskListView --> helpView : ?

    taskInputView --> taskListView : esc / submit
    editActiveTLView --> taskListView : esc / submit
    finishActiveTLView --> taskListView : esc / submit
    manualTasklogEntryView --> taskListView : esc / submit
    editSavedTLView --> taskLogView : esc / submit
    moveTaskLogView --> taskLogView : esc / submit
    taskLogDetailsView --> taskLogView : q / esc
    helpView --> taskListView : q / esc (restores lastView)

    taskListView --> insufficientDimensionsView : terminal too small
    insufficientDimensionsView --> taskListView : terminal resized
```
