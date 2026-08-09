# Agent Tool Expansion — Design

**Date:** 2026-08-09
**Status:** Approved (brainstorming) → pending implementation plan
**Scope:** P0 + P1 + P2 tool-coverage gaps (excludes P3 "do not expose")

## Context

A user told the agent "我已经完成了" referring to a *schedule* event. The agent
had no way to mutate a schedule's status, so it misused `update_task` with the
schedule's id → `record not found`. This is not an isolated bug but a **systemic
gap**: for each entity with a status lifecycle (Task / Schedule / WorkLog /
Session), the agent needs full CRUD coverage, or the LLM reaches for the wrong
domain's tool. Task already has full CRUD; Schedule/WorkLog/Timer/Analytics do
not. The backend *services* already implement these capabilities — they are
simply not exposed as tools.

A three-agent audit confirmed: **53 backend routes vs 14 agent tools**. This
spec closes the gap by adding 16 tools (14 → 30), prioritized by natural-language
frequency and the "complete the CRUD surface" principle, while deliberately
*not* exposing dangerous/infra operations (data import/clear, AI-settings,
conversation management).

## Design decisions

1. **Hybrid granularity** (chosen over one-per-action and max-consolidate):
   consolidate same-family **reads** into parametric tools; keep **writes** as
   distinct semantic tools. Balances tool-count/token cost against naming
   clarity and consistency with the existing one-per-action convention.
2. **`revise_schedule` two-step, file-mediated.** `ReviseSchedule` writes a
   baseline ICS, calls the AI to rewrite `schedule.ics`, parses both, returns a
   diff (no DB write). `ApplyRevision` reads the rewritten `schedule.ics` and
   applies it to the DB. State between the two is the on-disk `schedule.ics`.
   Rejected alternative: a single `PermDangerous` tool whose `Preview()` runs the
   AI — rejected because `Preview()` must stay cheap and side-effect-free (all
   existing tools' previews only echo args); running external AI in preview is
   slow, costly, and wasted on rejection.
3. **Permission tiers follow convention:** reads = `PermRead` (auto-exec),
  writes = `PermWrite` (confirm), irreversible DB rewrite = `PermDangerous`.
4. **Each domain keeps its narrow tools-package interface** (e.g. `ScheduleService`,
   `AnalyticsService`); new service methods are added to those interfaces, and
   test mocks implement them — mirroring the established pattern.

## Tool inventory (16 new, 14 → 30)

| Domain | Tool | Wraps (service method) | Permission | Priority |
|---|---|---|---|---|
| Schedule | `update_schedule` | `UpdateSchedule` (covers move/status) | Write | P0 |
| Schedule | `create_schedule` | `CreateScheduleEvent` | Write | P0 |
| Schedule | `revise_schedule(prompt)` | `ReviseSchedule` → diff | Write | P1 |
| Schedule | `apply_schedule_revision()` | `ApplyRevision` → DB write | **Dangerous** | P1 |
| Timer | `control_pomodoro(action, reason?)` | `Resume/Complete/AbandonSession` | Write | P1 |
| Timer | `get_pomodoro_stats` | `GetTodayTaskStats` + `GetRecentSessions` | Read | P1 |
| WorkLog | `get_worklog(date)` | `GetWorkLog` | Read | P1 |
| WorkLog | `list_worklogs(from, to)` | `ListWorkLogs` | Read | P1 |
| WorkLog | `generate_work_report(type, period_key)` | `GenerateReport` | Write | P1 |
| WorkLog | `get_work_report(type, period_key?)` | `GetReport` / `ListReports` (merged) | Read | P1 |
| WorkLog | `update_worklog` | `UpdateWorkLog` (full upsert) | Write | P2 |
| WorkLog | `update_worklog_summary` | `UpdateSummary` | Write | P2 |
| WorkLog | `add_worklog_entry` | `AddQuickEntry` | Write | P2 |
| Analytics | `get_analytics(metric, …)` | `GetTrend/GetDistribution/GetPomodoroByTask/GetPomodoroTrends` | Read | P1/P2 |
| Task | `move_task` | `MoveTask` | Write | P2 |
| Settings | `get_settings` | `GetPomodoroSettings` | Read | P2 |

Notes:
- `stop_pomodoro` (pause) is unchanged; `control_pomodoro` covers
  resume/complete/abandon. (`action ∈ {resume, complete, abandon}`.)
- `get_daily_insights` is unchanged; `get_analytics` handles the other four
  metrics (`metric ∈ {trend, distribution, pomodoro_by_task, pomodoro_trends}`).
- `move_task` overlaps with `update_task`'s priority field; kept because "移到第二
  象限" is a cleaner semantic and `MoveTask` auto-syncs important/urgent.

## File changes & interfaces

All new tools mirror the existing pattern (`Schema`/`Execute`/`Preview`,
`agent.ValidateArgs`, required-field guard, nil-safe `Svc`). Each domain's
narrow tools-package interface gains the methods it needs; test mocks implement
them.

- `backend/internal/agent/tools/schedule.go` — +4 tools; `ScheduleService`
  interface += `UpdateSchedule`, `CreateScheduleEvent`, `ReviseSchedule`,
  `ApplyRevision`.
- `backend/internal/agent/tools/timer.go` — +2 tools; `TimerService` interface
  += `ResumeSession`, `CompleteSession`, `AbandonSession`, `GetTodayTaskStats`,
  `GetRecentSessions`.
- `backend/internal/agent/tools/worklog.go` — +7 tools; the combined `WorkLog`
  interface += `GetWorkLog`, `ListWorkLogs`, `GenerateReport`, `GetReport`,
  `ListReports`, `UpdateWorkLog`, `UpdateSummary`, `AddQuickEntry`.
- `backend/internal/agent/tools/insight.go` — +1 tool (`get_analytics`);
  `AnalyticsService` interface += `GetTrend`, `GetDistribution`,
  `GetPomodoroByTask`, `GetPomodoroTrends`.
- `backend/internal/agent/tools/task.go` — +1 tool (`move_task`); `TaskService`
  interface += `MoveTask`.
- `backend/internal/agent/tools/settings.go` **(new)** — +1 tool (`get_settings`)
  + `SettingsReader` interface (`GetPomodoroSettings`).
- `backend/internal/agent/tools/register.go` — register 16 tools; add `Settings`
  field to `Deps`; fix the stale tool-count doc comment.
- `backend/cmd/server/main.go` (~line 132) — pass `Settings: settingRepo` into
  `tools.Deps{...}` (no `SettingService` exists; `settingRepo` is already
  constructed and threaded into `agent.AgentDeps`).
- `backend/internal/agent/prompts.go` — add an **id-domain guardrail** (schedule
  ids come from `list_schedule`, task ids from `list_tasks`; never cross them)
  and update the capability sentence.

## Testing — three layers

**Layer 1 — Unit (`go test`, CI gate, fast/mocked).** Table-driven per-tool
tests: delegation, missing-required, schema validation, service-error propagation,
and **preview-has-no-side-effects** (the invariant the rejected revise design
relied on). Mocks implement the new interface methods. Only layer run by
`make test`.

**Layer 2 — WS real-behavior eval (`eval/`, manual/regression, ~20 min, real
LLM).** Drives the **live** backend agent loop (`collect()` → POST `/chat` + WS
events) and asserts what mocks cannot: real tool routing, the real confirmation
lifecycle, and **real post-action DB state**. Additions:

- Add all new write tools to `WRITE_TOOLS` in `eval/cases.mjs`: `update_schedule,
  create_schedule, revise_schedule, apply_schedule_revision, control_pomodoro,
  generate_work_report, update_worklog, update_worklog_summary,
  add_worklog_entry, move_task`.
- New theme `eval/themes/schedule-lifecycle.mjs`:
  - *Routing:* "把审查那个日程标记完成"→`update_schedule`; "明天下午3点加个会"
    →`create_schedule`; "继续番茄钟"→`control_pomodoro`; "今天专注了多久"
    →`get_pomodoro_stats`; "看看昨天的日报"→`get_worklog`; "生成本周周报"
    →`generate_work_report`; "最近一周专注趋势"→`get_analytics`.
  - **Original-bug regression guard:** refer to a *schedule* and say "已完成"
    → assert `called(update_schedule)` **and not** `called(update_task)`.
  - *Confirmation lifecycle + dbVerify:* `update_schedule` mark-complete with
    `confirm:'approve', dbVerify:'schedules'` → target `status==='completed'`;
    a `confirm:'reject'` variant → DB unchanged. `control_pomodoro(complete)` +
    `dbVerify:'sessions'` → session completed. `create_schedule` +
    `dbVerify:'schedules'` → new event present.
  - *Revise two-step (`turns`):* `turns:['优化一下今天的安排','就按这个改吧']`,
    `confirm:'approve'` → `dbVerify:'schedules'` shows the diff applied.
- `eval/seed.mjs`: seed a known schedule (stable id/title) for the
  mark-complete dbVerify, plus today's work log, etc.
- `eval/README.md`: re-baseline counts after a run.

**Layer 3 — Multi-turn guard (`eval/multiturn-test.mjs`).** A scenario that
calls a new write tool (e.g. `create_schedule`) → `/confirm` approve → **keeps
chatting in the same conversation** → assert coherent reply. Guards the
historical "multi-turn tool-call breakage after a write" regression for the new
tools.

> **Gating:** Layers 2–3 are scoring/manual (real LLM, ~20 min); they are **not**
> a CI gate and `make test` does not invoke them. Layer 1 remains the automated
> gate; Layers 2–3 provide the real-behavior confidence mocks cannot.

## Verification (end-to-end)

1. `cd backend && go test ./internal/agent/...` — Layer 1 green.
2. `cd backend && go build ./...` — compiles (interface/mock wiring correct).
3. Restart backend per the restart rule, then in the running app reproduce the
   original failure positively: "把审查那个日程标记完成" routes to
   `update_schedule` and the schedule's status flips to completed.
4. `cd eval && AGENT_BASE_URL=http://localhost:8080 npm run cases` — Layers 2–3
   green against the live backend; original-bug regression case passes.

## Known tradeoffs / out of scope

- **30 tools raises per-turn token cost** (full schema sent each turn) and
  slightly increases mis-selection risk. Acceptable short-term (schemas are
  small); a future two-tier tool-loading scheme (core always-loaded + extended
  on-demand) is noted but **out of scope** here.
- **Out of scope (P3, deliberately not exposed):** data export/import/clear
  (`ClearAll` catastrophic, complex merge policies, dedicated UI);
  `update_ai_settings` (agent must not reconfigure its own brain / API key);
  `update_pomodoro_settings` (low frequency, better in Settings UI);
  conversation management (agent infrastructure, not a domain tool);
  single-item `get_schedule`/`get_task` (list tools already return everything).
