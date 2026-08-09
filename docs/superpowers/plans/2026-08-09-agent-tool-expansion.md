# Agent Tool Expansion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add 16 agent tools (14→30) closing the Schedule/WorkLog/Timer/Analytics/Task/Settings CRUD gaps that caused the "mark schedule complete → misuse update_task" failure.

**Architecture:** Each new tool mirrors the existing pattern in `backend/internal/agent/tools/` (`Schema`/`Execute`/`Preview`, `agent.ValidateArgs`, required-field guard, narrow per-domain service interface satisfied by the concrete service, in-package test mocks). Reads use `PermRead` (auto-exec), writes `PermWrite` (confirm), the irreversible schedule-DB rewrite `PermDangerous`. Hybrid granularity: parametric reads (`get_analytics`, `get_work_report`, `control_pomodoro`), distinct semantic writes. A WS-driven eval layer (`eval/`) verifies real routing + confirmation lifecycle + post-action DB state.

**Tech Stack:** Go 1.21 (standard `testing`, manual mocks), TickTask module path `ticktask`. Eval: Node `.mjs` + `ws`, real LLM.

**Spec:** `docs/superpowers/specs/2026-08-09-agent-tool-expansion-design.md`

**Branch:** `evolve/agent-tool-expansion` (spec already committed at `242b100`).

---

## File Structure

**Modify (Go tools):**
- `backend/internal/agent/tools/schedule.go` — +4 tools, extend `ScheduleService` iface
- `backend/internal/agent/tools/timer.go` — +2 tools, extend `TimerService` iface
- `backend/internal/agent/tools/worklog.go` — +7 tools, +3 new ifaces (`WorkLogReadSvc`/`WorkLogReportSvc`/`WorkLogWriteSvc`)
- `backend/internal/agent/tools/insight.go` — +1 tool, extend `AnalyticsService` iface
- `backend/internal/agent/tools/task.go` — +1 tool, extend `TaskService` iface
- `backend/internal/agent/tools/register.go` — register all, extend `Deps` (+`Settings`, widen `WorkLog`), fix doc comment
- `backend/cmd/server/main.go` — pass `Settings: settingRepo` into `tools.Deps`
- `backend/internal/agent/prompts.go` — id-domain guardrail + capability sentence

**Create (Go tools):**
- `backend/internal/agent/tools/settings.go` — `get_settings` + `SettingsReader` iface

**Modify (Go tests — co-located):**
- `schedule_test.go`, `timer_test.go`, `worklog_test.go`, `insight_test.go`, `task_test.go` — extend mocks + add tests
- `register_test.go` — widen `mockWorkLogSvc` embeds; `insight_test.go` `wantNames` list

**Create (Go tests):** `backend/internal/agent/tools/settings_test.go`

**Modify (eval, Layer 2–3):**
- `eval/cases.mjs` — add new write tools to `WRITE_TOOLS`
- `eval/themes/schedule-lifecycle.mjs` (new) — routing + confirm/dbVerify + bug-regression cases
- `eval/themes/index.mjs` — register the new theme
- `eval/seed.mjs` — seed a known schedule + today's work log
- `eval/multiturn-test.mjs` — add a `create_schedule` follow-up scenario
- `eval/README.md` — re-baseline

---

## Canonical patterns (reference, do not skip when writing code)

Every tool implements `agent.Tool` (`Schema()`, `Execute(ctx, args)`, `Preview(ctx, args)`). The body shape used throughout:

```go
func (t *FooTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct{ ... }
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.RequiredField == "" {
		return nil, fmt.Errorf("schema: missing required field required_field")
	}
	// ... call t.Svc ...
	return map[string]any{...}, nil
}
```

- **PermRead** tools: `Preview` returns `t.Execute(ctx, args)` (mirror, safe).
- **PermWrite/Dangerous** tools: `Preview` returns a *plan map* and MUST NOT call the service (no side effects). Decode args best-effort with `_ = json.Unmarshal(...)`.
- Extend a domain interface ⇒ every mock implementing it (in the same `package tools` test files) must gain the new methods or the package won't compile. Add them in the same task.
- `ValidateArgs` enforces `type` + `required`; it does NOT enforce enum values — validate enums in `Execute` and return `fmt.Errorf("schema: ...")`.

---

## Task 1: P0 — Schedule tools `update_schedule` + `create_schedule`

This is the direct fix for the originating bug. Backed by existing `ScheduleService.UpdateSchedule` / `CreateScheduleEvent`.

**Files:**
- Modify: `backend/internal/agent/tools/schedule.go` (extend iface + add 2 tools)
- Modify: `backend/internal/agent/tools/schedule_test.go` (extend `mockScheduleSvc` + add tests)
- Modify: `backend/internal/agent/tools/register.go` (register 2 tools + fix doc comment)

- [ ] **Step 1: Extend the `ScheduleService` interface** in `schedule.go` (add 2 methods to the existing `type ScheduleService interface { ... }` block, after `DeleteSchedule`):

```go
type ScheduleService interface {
	GetSchedules(start, end time.Time) ([]service.ScheduleEvent, error)
	GenerateSchedule(startTime, endTime string) ([]service.ScheduleEvent, string, error)
	DeleteSchedule(id string) error
	UpdateSchedule(id string, dto *service.UpdateScheduleDTO) error
	CreateScheduleEvent(dto *service.CreateScheduleDTO) (*service.ScheduleEvent, error)
}
```

- [ ] **Step 2: Extend `mockScheduleSvc`** in `schedule_test.go` — add fields + 2 methods (place after the existing `DeleteSchedule` method, ~line 41):

```go
type mockScheduleSvc struct {
	listEvents []service.ScheduleEvent
	listErr    error
	listStart  time.Time
	listEnd    time.Time
	listCalls  int

	genEvents  []service.ScheduleEvent
	genSummary string
	genErr     error
	genStart   string
	genEnd     string
	genCalls   int

	delErr   error
	delCalls int
	delIDs   []string

	updErr    error
	updCalls  int
	updIDs    []string
	updDTOs   []*service.UpdateScheduleDTO

	createEvt *service.ScheduleEvent
	createErr error
	createDTOs []*service.CreateScheduleDTO
}

func (m *mockScheduleSvc) UpdateSchedule(id string, dto *service.UpdateScheduleDTO) error {
	m.updCalls++
	m.updIDs = append(m.updIDs, id)
	m.updDTOs = append(m.updDTOs, dto)
	return m.updErr
}

func (m *mockScheduleSvc) CreateScheduleEvent(dto *service.CreateScheduleDTO) (*service.ScheduleEvent, error) {
	m.createDTOs = append(m.createDTOs, dto)
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.createEvt != nil {
		return m.createEvt, nil
	}
	return &service.ScheduleEvent{ID: "new-1", Title: dto.Title, Start: dto.StartTime, End: dto.EndTime, Type: dto.Type}, nil
}
```

- [ ] **Step 3: Write failing tests** — append to `schedule_test.go`:

```go
// --- UpdateScheduleTool ---

func TestUpdateSchedule_DelegatesAndRequiresID(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &UpdateScheduleTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"schedule_id":"s-1","status":"completed"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.updCalls != 1 || len(svc.updIDs) != 1 || svc.updIDs[0] != "s-1" {
		t.Errorf("UpdateSchedule calls = %+v", svc.updIDs)
	}
	if svc.updDTOs[0].Status != "completed" {
		t.Errorf("dto.Status = %q, want completed", svc.updDTOs[0].Status)
	}
	m, _ := res.(map[string]any)
	if m["updated"] != true || m["schedule_id"] != "s-1" {
		t.Errorf("result = %+v", res)
	}
	// missing id fails BEFORE touching the service
	svc.updCalls = 0
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"status":"completed"}`)); err == nil {
		t.Error("expected error for missing schedule_id")
	}
	if svc.updCalls != 0 {
		t.Errorf("service called on invalid args")
	}
}

func TestUpdateSchedule_StatusEnumValidation(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &UpdateScheduleTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"schedule_id":"s-1","status":"bogus"}`))
	if err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("expected status enum error, got %v", err)
	}
	if svc.updCalls != 0 {
		t.Errorf("service must not be called on bad enum")
	}
}

func TestUpdateSchedule_PreviewNoSideEffect(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &UpdateScheduleTool{Svc: svc}
	pv, err := tool.Preview(context.Background(), json.RawMessage(`{"schedule_id":"s-1","status":"completed"}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(pv)
	if !strings.Contains(string(m), `"update_schedule"`) || svc.updCalls != 0 {
		t.Fatalf("preview must echo plan and not call service: %s calls=%d", m, svc.updCalls)
	}
}

// --- CreateScheduleTool ---

func TestCreateSchedule_DelegatesAndReturnsEvent(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &CreateScheduleTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"title":"和 Bob 1:1","start":"2026-08-10T15:00:00Z","end":"2026-08-10T15:30:00Z"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(svc.createDTOs) != 1 || svc.createDTOs[0].Title != "和 Bob 1:1" {
		t.Errorf("create DTO = %+v", svc.createDTOs)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), `"new-1"`) {
		t.Errorf("result should include created event id: %s", m)
	}
}

func TestCreateSchedule_RequiresTitleAndTimes(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &CreateScheduleTool{Svc: svc}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"start":"x","end":"y"}`)); err == nil {
		t.Error("expected error for missing title")
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"title":"t","start":"x"}`)); err == nil {
		t.Error("expected error for missing end")
	}
	if len(svc.createDTOs) != 0 {
		t.Errorf("service must not be called on invalid args")
	}
}

func TestCreateSchedule_PreviewNoSideEffect(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &CreateScheduleTool{Svc: svc}
	pv, err := tool.Preview(context.Background(), json.RawMessage(`{"title":"t","start":"s","end":"e"}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(pv)
	if !strings.Contains(string(m), `"create_schedule"`) || len(svc.createDTOs) != 0 {
		t.Fatalf("preview must echo plan and not call service: %s", m)
	}
}
```

- [ ] **Step 4: Run tests to verify they fail** (tools undefined)

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestUpdateSchedule|TestCreateSchedule' -v`
Expected: COMPILE ERROR `undefined: UpdateScheduleTool` / `CreateScheduleTool`.

- [ ] **Step 5: Implement the two tools** — append to `schedule.go`:

```go
// =====================================================================
// update_schedule
// =====================================================================

// UpdateScheduleTool partially updates a schedule event (from list_schedule):
// title/description/start/end/status/color. Only provided fields change. To mark
// a schedule finished set status="completed". PermWrite: requires confirmation.
// NOTE: use schedule ids from list_schedule; do NOT pass task ids here (use
// update_task for tasks) — crossing id domains was the original failure mode.
type UpdateScheduleTool struct {
	Svc ScheduleService
}

var scheduleStatuses = map[string]bool{
	"planned": true, "in_progress": true, "completed": true, "cancelled": true,
}

func (t *UpdateScheduleTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "update_schedule",
		Function: agent.FunctionSpec{
			Name:        "update_schedule",
			Description: "Partially update a schedule event (from list_schedule). Only provided fields change. Set status='completed' to mark a schedule done. Use update_task for tasks, not this.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"schedule_id":  map[string]any{"type": "string", "description": "id from list_schedule"},
					"title":        map[string]any{"type": "string"},
					"description":  map[string]any{"type": "string"},
					"start":        map[string]any{"type": "string", "description": "RFC3339"},
					"end":          map[string]any{"type": "string", "description": "RFC3339"},
					"status":       map[string]any{"type": "string", "description": "planned|in_progress|completed|cancelled"},
					"color":        map[string]any{"type": "string"},
				},
				"required": []any{"schedule_id"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *UpdateScheduleTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		ScheduleID  string  `json:"schedule_id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Start       *string `json:"start"`
		End         *string `json:"end"`
		Status      *string `json:"status"`
		Color       *string `json:"color"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.ScheduleID == "" {
		return nil, fmt.Errorf("schema: missing required field schedule_id")
	}
	if in.Status != nil && *in.Status != "" && !scheduleStatuses[*in.Status] {
		return nil, fmt.Errorf("schema: status must be planned|in_progress|completed|cancelled, got %q", *in.Status)
	}
	dto := &service.UpdateScheduleDTO{}
	if in.Title != nil {
		dto.Title = *in.Title
	}
	if in.Description != nil {
		dto.Description = *in.Description
	}
	if in.Start != nil {
		dto.StartTime = *in.Start
	}
	if in.End != nil {
		dto.EndTime = *in.End
	}
	if in.Status != nil {
		dto.Status = *in.Status
	}
	if in.Color != nil {
		dto.Color = *in.Color
	}
	if err := t.Svc.UpdateSchedule(in.ScheduleID, dto); err != nil {
		return nil, err
	}
	return map[string]any{"schedule_id": in.ScheduleID, "updated": true}, nil
}

func (t *UpdateScheduleTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		ScheduleID string `json:"schedule_id"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "update_schedule", "schedule_id": in.ScheduleID, "args": json.RawMessage(args)}, nil
}

// =====================================================================
// create_schedule
// =====================================================================

// CreateScheduleTool creates a single ad-hoc schedule event (e.g. "加个会，明天
// 下午3点"). PermWrite: requires confirmation. Distinct from generate_schedule,
// which AI-generates a whole day.
type CreateScheduleTool struct {
	Svc ScheduleService
}

var scheduleTypes = map[string]bool{
	"task": true, "pomodoro": true, "break": true, "custom": true,
}

func (t *CreateScheduleTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "create_schedule",
		Function: agent.FunctionSpec{
			Name:        "create_schedule",
			Description: "Create a single ad-hoc schedule event with explicit start/end times (RFC3339). Use generate_schedule for AI bulk scheduling.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":       map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"start":       map[string]any{"type": "string", "description": "RFC3339"},
					"end":         map[string]any{"type": "string", "description": "RFC3339"},
					"type":        map[string]any{"type": "string", "description": "task|pomodoro|break|custom (default task)"},
					"color":       map[string]any{"type": "string"},
				},
				"required": []any{"title", "start", "end"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *CreateScheduleTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
		Start       string  `json:"start"`
		End         string  `json:"end"`
		Type        *string `json:"type"`
		Color       *string `json:"color"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.Title == "" {
		return nil, fmt.Errorf("schema: missing required field title")
	}
	if in.Start == "" {
		return nil, fmt.Errorf("schema: missing required field start")
	}
	if in.End == "" {
		return nil, fmt.Errorf("schema: missing required field end")
	}
	dto := &service.CreateScheduleDTO{
		Title:     in.Title,
		StartTime: in.Start,
		EndTime:   in.End,
	}
	if in.Description != nil {
		dto.Description = *in.Description
	}
	if in.Type != nil && *in.Type != "" {
		if !scheduleTypes[*in.Type] {
			return nil, fmt.Errorf("schema: type must be task|pomodoro|break|custom, got %q", *in.Type)
		}
		dto.Type = *in.Type
	} else {
		dto.Type = "task"
	}
	if in.Color != nil {
		dto.Color = *in.Color
	}
	return t.Svc.CreateScheduleEvent(dto)
}

func (t *CreateScheduleTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		Title string `json:"title"`
		Start string `json:"start"`
		End   string `json:"end"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "create_schedule", "title": in.Title, "start": in.Start, "end": in.End}, nil
}
```

- [ ] **Step 6: Register the two tools** in `register.go`. In `RegisterAll`, after the `DeleteScheduleTool` line:

```go
	reg.MustRegister(&UpdateScheduleTool{Svc: deps.Schedule})
	reg.MustRegister(&CreateScheduleTool{Svc: deps.Schedule})
```

Also fix the stale doc comment above `RegisterAll`: change `13 tools: 5 task tools + 3 timer tools + 2 schedule tools + 1 insight tool + 2 work-log tools.` to `16 tools: 5 task + 3 timer + 5 schedule + 1 insight + 2 work-log (grows across later tasks).` (Update the running total again in the final task.)

- [ ] **Step 7: Run tests to verify they pass**

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestUpdateSchedule|TestCreateSchedule' -v`
Expected: PASS (all 6 tests). Then `go build ./...` → compiles.

- [ ] **Step 8: Commit**

```bash
git add backend/internal/agent/tools/schedule.go backend/internal/agent/tools/schedule_test.go backend/internal/agent/tools/register.go
git commit -m "feat(agent): add update_schedule + create_schedule tools (P0)

Closes the schedule-CRUD gap behind the 'mark schedule complete ->
misuse update_task' failure. update_schedule covers status/move/rename;
create_schedule adds ad-hoc events."

# (Co-Authored-By trailer is auto-added by your commit hook if present;
#  otherwise append a blank line then: Co-Authored-By: Claude <noreply@anthropic.com>)
```

---

## Task 2: P1 — Schedule revise (`revise_schedule` + `apply_schedule_revision`)

Two-step, file-mediated (`schedule.ics`). `revise_schedule` returns a diff (PermWrite); `apply_schedule_revision` writes the DB (PermDangerous).

**Files:** Modify `schedule.go`, `schedule_test.go`, `register.go`.

- [ ] **Step 1: Extend the `ScheduleService` interface** (add 2 methods):

```go
	ReviseSchedule(prompt string) (*service.ReviseResponse, error)
	ApplyRevision() ([]service.ScheduleEvent, error)
```

- [ ] **Step 2: Extend `mockScheduleSvc`** — add fields + 2 methods:

```go
	// add fields
	revResp    *service.ReviseResponse
	revErr     error
	revPrompts []string
	applyEvts  []service.ScheduleEvent
	applyErr   error
	applyCalls int
```

```go
func (m *mockScheduleSvc) ReviseSchedule(prompt string) (*service.ReviseResponse, error) {
	m.revPrompts = append(m.revPrompts, prompt)
	if m.revErr != nil {
		return nil, m.revErr
	}
	return m.revResp, nil
}

func (m *mockScheduleSvc) ApplyRevision() ([]service.ScheduleEvent, error) {
	m.applyCalls++
	if m.applyErr != nil {
		return nil, m.applyErr
	}
	return m.applyEvts, nil
}
```

- [ ] **Step 3: Write failing tests** — append to `schedule_test.go`:

```go
func TestReviseSchedule_DelegatesAndReturnsDiff(t *testing.T) {
	svc := &mockScheduleSvc{revResp: &service.ReviseResponse{
		Summary: "moved 2", Applied: false,
		Changes: []service.RevisionChange{{Type: "moved", Title: "审查 PR-1234"}},
	}}
	tool := &ReviseScheduleTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"prompt":"把下午的会都往后推一小时"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(svc.revPrompts) != 1 || svc.revPrompts[0] != "把下午的会都往后推一小时" {
		t.Errorf("prompt forwarded = %+v", svc.revPrompts)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), "moved 2") || !strings.Contains(string(m), `"applied":false`) {
		t.Errorf("result should expose diff preview, got %s", m)
	}
	// missing prompt fails before service
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Error("expected error for missing prompt")
	}
}

func TestApplyScheduleRevision_DelegatesAndRequiresPriorRevise(t *testing.T) {
	svc := &mockScheduleSvc{applyEvts: []service.ScheduleEvent{{ID: "s-1"}}}
	tool := &ApplyScheduleRevisionTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.applyCalls != 1 {
		t.Fatalf("ApplyRevision calls = %d, want 1", svc.applyCalls)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), `"applied":true`) {
		t.Errorf("result should mark applied, got %s", m)
	}
}
```

- [ ] **Step 4: Run tests → fail** (`undefined: ReviseScheduleTool`).

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestReviseSchedule|TestApplyScheduleRevision' -v`

- [ ] **Step 5: Implement the two tools** — append to `schedule.go`:

```go
// =====================================================================
// revise_schedule + apply_schedule_revision
// =====================================================================

// ReviseScheduleTool runs an AI schedule revision and returns a diff preview
// WITHOUT writing the DB (it writes a baseline + revised schedule.ics on disk).
// The user reviews the diff, then apply_schedule_revision persists it. PermWrite:
// it calls the external AI skill and writes the ICS file, so it confirms.
//
// CONSTRAINT: apply_schedule_revision must follow in the same conversation
// without an intervening generate_schedule (which overwrites schedule.ics).
type ReviseScheduleTool struct {
	Svc ScheduleService
}

func (t *ReviseScheduleTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "revise_schedule",
		Function: agent.FunctionSpec{
			Name:        "revise_schedule",
			Description: "Ask the AI to revise the schedule per a natural-language prompt and return a diff preview (moved/added/removed). Does NOT persist — call apply_schedule_revision after the user agrees.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"prompt": map[string]any{"type": "string", "description": "revision instruction, e.g. '把下午的会都往后推一小时'"},
				},
				"required": []any{"prompt"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *ReviseScheduleTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Prompt string `json:"prompt"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.Prompt == "" {
		return nil, fmt.Errorf("schema: missing required field prompt")
	}
	resp, err := t.Svc.ReviseSchedule(in.Prompt)
	if err != nil {
		return nil, err
	}
	return resp, nil
}

func (t *ReviseScheduleTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		Prompt string `json:"prompt"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "revise_schedule", "prompt": in.Prompt}, nil
}

// ApplyScheduleRevisionTool persists the most recent AI revision to the DB
// (deletes old task schedules, writes new). PermDangerous: irreversible DB rewrite.
// Must be preceded by revise_schedule in the same conversation.
type ApplyScheduleRevisionTool struct {
	Svc ScheduleService
}

func (t *ApplyScheduleRevisionTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "apply_schedule_revision",
		Function: agent.FunctionSpec{
			Name:        "apply_schedule_revision",
			Description: "Apply the schedule revision produced by the last revise_schedule call to the database. Irreversible. Only call after revise_schedule and user agreement.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		Permission: agent.PermDangerous,
	}
}

func (t *ApplyScheduleRevisionTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	events, err := t.Svc.ApplyRevision()
	if err != nil {
		return nil, err
	}
	return map[string]any{"applied": true, "events": events, "count": len(events)}, nil
}

func (t *ApplyScheduleRevisionTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return map[string]any{"action": "apply_schedule_revision", "warning": "irreversibly rewrites schedule DB"}, nil
}
```

- [ ] **Step 6: Register** in `register.go` after the schedule tools:

```go
	reg.MustRegister(&ReviseScheduleTool{Svc: deps.Schedule})
	reg.MustRegister(&ApplyScheduleRevisionTool{Svc: deps.Schedule})
```

- [ ] **Step 7: Run tests → pass**, then `go build ./...`.

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestReviseSchedule|TestApplyScheduleRevision' -v`

- [ ] **Step 8: Commit**

```bash
git add backend/internal/agent/tools/schedule.go backend/internal/agent/tools/schedule_test.go backend/internal/agent/tools/register.go
git commit -m "feat(agent): add revise_schedule + apply_schedule_revision (P1)"
```

---

## Task 3: P1 — Timer tools `control_pomodoro` + `get_pomodoro_stats`

Closes the "can pause but not resume/complete" gap. `control_pomodoro(action)` maps to the unified control path; `get_pomodoro_stats` returns today's per-task time + recent sessions.

**Files:** Modify `timer.go`, `timer_test.go`, `register.go`.

- [ ] **Step 1: Extend `TimerService` interface** in `timer.go`:

```go
type TimerService interface {
	StartSession(req service.CreateSessionRequest) (*model.PomodoroSession, error)
	PauseSession(sessionID string) error
	GetActiveSession() (*model.PomodoroSession, error)
	ResumeSession(sessionID string) error
	CompleteSession(sessionID string) error
	AbandonSession(sessionID string, interruptReason string) error
	GetTodayTaskStats() ([]service.TaskTimeStats, error)
	GetRecentSessions(limit int) ([]model.PomodoroSession, error)
}
```

- [ ] **Step 2: Extend `mockTimerSvc`** in `timer_test.go` — add fields + 5 methods. (Read the existing `mockTimerSvc` struct first to preserve its fields.)

```go
	// add fields
	resumeErr   error
	resumeCalls int
	completeErr error
	completeIDs []string
	abandonErr  error
	abandonIDs  []string
	abandonWhy  []string
	todayStats  []service.TaskTimeStats
	todayErr    error
	recent      []model.PomodoroSession
	recentErr   error
	recentLimit int
```

```go
func (m *mockTimerSvc) ResumeSession(sessionID string) error {
	m.resumeCalls++
	return m.resumeErr
}
func (m *mockTimerSvc) CompleteSession(sessionID string) error {
	m.completeIDs = append(m.completeIDs, sessionID)
	return m.completeErr
}
func (m *mockTimerSvc) AbandonSession(sessionID string, interruptReason string) error {
	m.abandonIDs = append(m.abandonIDs, sessionID)
	m.abandonWhy = append(m.abandonWhy, interruptReason)
	return m.abandonErr
}
func (m *mockTimerSvc) GetTodayTaskStats() ([]service.TaskTimeStats, error) {
	if m.todayErr != nil {
		return nil, m.todayErr
	}
	return m.todayStats, nil
}
func (m *mockTimerSvc) GetRecentSessions(limit int) ([]model.PomodoroSession, error) {
	m.recentLimit = limit
	if m.recentErr != nil {
		return nil, m.recentErr
	}
	return m.recent, nil
}
```

- [ ] **Step 3: Write failing tests** — append to `timer_test.go`:

```go
func TestControlPomodoro_RoutesActionToSession(t *testing.T) {
	cases := []struct {
		action   string
		permErr  error
	}{
		{"resume", nil},
		{"complete", nil},
		{"abandon", nil},
	}
	for _, c := range cases {
		svc := &mockTimerSvc{}
		tool := &ControlPomodoroTool{Svc: svc}
		_, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"`+c.action+`"}`))
		if err != nil {
			t.Errorf("action %s: %v", c.action, err)
		}
	}
	// unknown action rejected before touching the service
	svc := &mockTimerSvc{}
	tool := &ControlPomodoroTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"bogus"}`))
	if err == nil || !strings.Contains(err.Error(), "action") {
		t.Fatalf("expected action enum error, got %v", err)
	}
	if svc.resumeCalls != 0 && len(svc.completeIDs) != 0 && len(svc.abandonIDs) != 0 {
		t.Errorf("service must not be called on bad action")
	}
}

func TestControlPomodoro_RequiresActiveSession(t *testing.T) {
	svc := &mockTimerSvc{active: nil} // no active session → GetActiveSession returns nil
	tool := &ControlPomodoroTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"resume"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := res.(map[string]any)
	if m["active"] != false {
		t.Errorf("expected active=false when no session, got %+v", res)
	}
}

func TestGetPomodoroStats_Aggregates(t *testing.T) {
	svc := &mockTimerSvc{
		todayStats: []service.TaskTimeStats{{TaskID: "t1", TaskTitle: "写文档", SessionCount: 2, TotalTime: 1800}},
		recent:     []model.PomodoroSession{{ID: "s1"}},
	}
	tool := &GetPomodoroStatsTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), "写文档") || !strings.Contains(string(m), `"s1"`) {
		t.Errorf("result should include today stats + recent sessions: %s", m)
	}
}
```

> Note: `TestControlPomodoro_RequiresActiveSession` references `svc.active` — confirm the existing `mockTimerSvc` field name for the active-session return value by reading `timer_test.go` first; if the existing field is named differently (e.g. `activeSess`), use that name. The mock's `GetActiveSession()` already exists (it backs `get_timer_status`).

- [ ] **Step 4: Run tests → fail** (`undefined: ControlPomodoroTool`).

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestControlPomodoro|TestGetPomodoroStats' -v`

- [ ] **Step 5: Implement the two tools** — append to `timer.go`:

```go
// =====================================================================
// control_pomodoro
// =====================================================================

// ControlPomodoroTool resumes / completes / abandons the active pomodoro session.
// (Pause stays on stop_pomodoro.) PermWrite: mutating timer state.
type ControlPomodoroTool struct {
	Svc TimerService
}

var controlActions = map[string]bool{"resume": true, "complete": true, "abandon": true}

func (t *ControlPomodoroTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "control_pomodoro",
		Function: agent.FunctionSpec{
			Name:        "control_pomodoro",
			Description: "Control the active pomodoro session: resume (after stop_pomodoro/暂停), complete (mark done early), or abandon (give up, optional reason). To pause, use stop_pomodoro.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{"type": "string", "description": "resume|complete|abandon"},
					"reason": map[string]any{"type": "string", "description": "optional interruption reason (abandon)"},
				},
				"required": []any{"action"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *ControlPomodoroTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Action string  `json:"action"`
		Reason *string `json:"reason"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if !controlActions[in.Action] {
		return nil, fmt.Errorf("schema: action must be resume|complete|abandon, got %q", in.Action)
	}
	active, err := t.Svc.GetActiveSession()
	if err != nil {
		return nil, err
	}
	if active == nil {
		return map[string]any{"active": false}, nil
	}
	switch in.Action {
	case "resume":
		if err := t.Svc.ResumeSession(active.ID); err != nil {
			return nil, err
		}
	case "complete":
		if err := t.Svc.CompleteSession(active.ID); err != nil {
			return nil, err
		}
	case "abandon":
		reason := ""
		if in.Reason != nil {
			reason = *in.Reason
		}
		if err := t.Svc.AbandonSession(active.ID, reason); err != nil {
			return nil, err
		}
	}
	return map[string]any{"action": in.Action, "session_id": active.ID, "ok": true}, nil
}

func (t *ControlPomodoroTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		Action string `json:"action"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "control_pomodoro", "control": in.Action}, nil
}

// =====================================================================
// get_pomodoro_stats
// =====================================================================

// GetPomodoroStatsTool returns today's per-task time investment plus recent
// sessions. PermRead: auto-executed.
type GetPomodoroStatsTool struct {
	Svc TimerService
}

func (t *GetPomodoroStatsTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "get_pomodoro_stats",
		Function: agent.FunctionSpec{
			Name:        "get_pomodoro_stats",
			Description: "Today's focus time per task plus recent pomodoro sessions. Answers '今天专注了多久' / '最近记录'.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"recent_limit": map[string]any{"type": "integer", "description": "number of recent sessions (default 10)"},
				},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *GetPomodoroStatsTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		RecentLimit *int `json:"recent_limit"`
	}
	_ = json.Unmarshal(args, &in)
	limit := 10
	if in.RecentLimit != nil && *in.RecentLimit > 0 {
		limit = *in.RecentLimit
	}
	today, err := t.Svc.GetTodayTaskStats()
	if err != nil {
		return nil, err
	}
	recent, err := t.Svc.GetRecentSessions(limit)
	if err != nil {
		return nil, err
	}
	return map[string]any{"today": today, "recent": recent}, nil
}

func (t *GetPomodoroStatsTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}
```

- [ ] **Step 6: Register** in `register.go` after the timer tools:

```go
	reg.MustRegister(&ControlPomodoroTool{Svc: deps.Timer})
	reg.MustRegister(&GetPomodoroStatsTool{Svc: deps.Timer})
```

- [ ] **Step 7: Run tests → pass**, then `go build ./...`.

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestControlPomodoro|TestGetPomodoroStats' -v`

- [ ] **Step 8: Commit**

```bash
git add backend/internal/agent/tools/timer.go backend/internal/agent/tools/timer_test.go backend/internal/agent/tools/register.go
git commit -m "feat(agent): add control_pomodoro + get_pomodoro_stats (P1)"
```

---

## Task 4: P1 — WorkLog reads + report (`get_worklog`, `list_worklogs`, `generate_work_report`, `get_work_report`)

**Files:** Modify `worklog.go`, `worklog_test.go`, `register.go` (widen `Deps.WorkLog`), `register_test.go` (widen `mockWorkLogSvc`).

- [ ] **Step 1: Add new interfaces** to `worklog.go` (after `WorkLogSaveSvc`):

```go
// WorkLogReadSvc covers reading daily logs.
type WorkLogReadSvc interface {
	GetWorkLog(date string) (*model.WorkLog, error)
	ListWorkLogs(from, to string) ([]*model.WorkLog, error)
}

// WorkLogReportSvc covers period reports (weekly/monthly/halfyear/yearly).
type WorkLogReportSvc interface {
	GenerateReport(input service.GenerateReportInput) (*model.WorkReport, error)
	GetReport(t model.WorkReportType, periodKey string) (*model.WorkReport, error)
	ListReports(t model.WorkReportType) ([]*model.WorkReport, error)
}
```

- [ ] **Step 2: Widen `Deps.WorkLog`** in `register.go`:

```go
	WorkLog interface {
		WorkLogStructureSvc
		WorkLogSaveSvc
		WorkLogReadSvc
		WorkLogReportSvc
		WorkLogWriteSvc // added in Task 5
	}
```

> `WorkLogWriteSvc` is defined in Task 5. To keep this task compiling, either (a) do Tasks 4+5 together, or (b) drop the `WorkLogWriteSvc` line here and add it in Task 5. **Choose (b):** in this task, union only `Structure+Save+Read+Report`; add `WorkLogWriteSvc` in Task 5's Step 1.

So in this task the union is:

```go
	WorkLog interface {
		WorkLogStructureSvc
		WorkLogSaveSvc
		WorkLogReadSvc
		WorkLogReportSvc
	}
```

- [ ] **Step 3: Add mocks** to `worklog_test.go` (alongside existing mocks):

```go
type mockWorkLogReadSvc struct {
	getLog    *model.WorkLog
	getErr    error
	getDates  []string
	listLogs  []*model.WorkLog
	listErr   error
	listCalls int
}

func (m *mockWorkLogReadSvc) GetWorkLog(date string) (*model.WorkLog, error) {
	m.getDates = append(m.getDates, date)
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getLog, nil
}
func (m *mockWorkLogReadSvc) ListWorkLogs(from, to string) ([]*model.WorkLog, error) {
	m.listCalls++
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listLogs, nil
}

type mockWorkLogReportSvc struct {
	genIn     *service.GenerateReportInput
	genOut    *model.WorkReport
	genErr    error
	getOut    *model.WorkReport
	getErr    error
	listOut   []*model.WorkReport
	listErr   error
}

func (m *mockWorkLogReportSvc) GenerateReport(input service.GenerateReportInput) (*model.WorkReport, error) {
	m.genIn = &input
	if m.genErr != nil {
		return nil, m.genErr
	}
	return m.genOut, nil
}
func (m *mockWorkLogReportSvc) GetReport(t model.WorkReportType, periodKey string) (*model.WorkReport, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}
	return m.getOut, nil
}
func (m *mockWorkLogReportSvc) ListReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listOut, nil
}
```

- [ ] **Step 4: Widen `mockWorkLogSvc`** in `register_test.go` (embed the two new mocks so the combined mock still satisfies `Deps.WorkLog`):

```go
type mockWorkLogSvc struct {
	mockWorkLogStructureSvc
	mockWorkLogSaveSvc
	mockWorkLogReadSvc
	mockWorkLogReportSvc
}
```

Add compile-time guards next to the existing ones:

```go
var (
	_ WorkLogStructureSvc = (*mockWorkLogSvc)(nil)
	_ WorkLogSaveSvc      = (*mockWorkLogSvc)(nil)
	_ WorkLogReadSvc      = (*mockWorkLogSvc)(nil)
	_ WorkLogReportSvc    = (*mockWorkLogSvc)(nil)
)
```

- [ ] **Step 5: Write failing tests** — append to `worklog_test.go`:

```go
func TestGetWorkLog_DelegatesByDate(t *testing.T) {
	svc := &mockWorkLogReadSvc{getLog: &model.WorkLog{Date: "2026-08-08"}}
	tool := &GetWorklogTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"2026-08-08"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(svc.getDates) != 1 || svc.getDates[0] != "2026-08-08" {
		t.Errorf("dates = %+v", svc.getDates)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), "2026-08-08") {
		t.Errorf("result should include the log: %s", m)
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Error("expected error for missing date")
	}
}

func TestListWorklogs_Range(t *testing.T) {
	svc := &mockWorkLogReadSvc{listLogs: []*model.WorkLog{{Date: "2026-08-08"}}}
	tool := &ListWorklogsTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"from":"2026-08-01","to":"2026-08-31"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.listCalls != 1 {
		t.Errorf("ListWorkLogs calls = %d", svc.listCalls)
	}
}

func TestGenerateWorkReport_Delegates(t *testing.T) {
	svc := &mockWorkLogReportSvc{genOut: &model.WorkReport{Type: model.ReportWeekly}}
	tool := &GenerateWorkReportTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"type":"weekly","period_key":"2026-W32"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.genIn == nil || svc.genIn.Type != model.ReportWeekly || svc.genIn.PeriodKey != "2026-W32" {
		t.Errorf("GenerateReport input = %+v", svc.genIn)
	}
	// bad type rejected
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"type":"bogus","period_key":"x"}`)); err == nil {
		t.Error("expected error for bad report type")
	}
}

func TestGetWorkReport_ListWhenNoPeriod(t *testing.T) {
	svc := &mockWorkLogReportSvc{listOut: []*model.WorkReport{{Type: model.ReportWeekly}}}
	tool := &GetWorkReportTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"type":"weekly"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), `"list":`) {
		t.Errorf("omitting period_key should list reports: %s", m)
	}
}
```

- [ ] **Step 6: Run tests → fail** (`undefined: GetWorklogTool` etc.).

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestGetWorkLog|TestListWorklogs|TestGenerateWorkReport|TestGetWorkReport' -v`

- [ ] **Step 7: Implement the four tools** — append to `worklog.go`:

```go
var reportTypes = map[string]model.WorkReportType{
	"weekly": model.ReportWeekly, "monthly": model.ReportMonthly,
	"halfyear": model.ReportHalfYear, "yearly": model.ReportYearly,
}

// =====================================================================
// get_worklog
// =====================================================================

type GetWorklogTool struct{ Svc WorkLogReadSvc }

func (t *GetWorklogTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "get_worklog",
		Function: agent.FunctionSpec{
			Name:        "get_worklog",
			Description: "Get the work log for a single date (YYYY-MM-DD).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"date": map[string]any{"type": "string", "description": "YYYY-MM-DD"},
				},
				"required": []any{"date"},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *GetWorklogTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Date string `json:"date"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.Date == "" {
		return nil, fmt.Errorf("schema: missing required field date")
	}
	if _, err := time.Parse("2006-01-02", in.Date); err != nil {
		return nil, fmt.Errorf("invalid date %q (expected YYYY-MM-DD): %w", in.Date, err)
	}
	return t.Svc.GetWorkLog(in.Date)
}

func (t *GetWorklogTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}

// =====================================================================
// list_worklogs
// =====================================================================

type ListWorklogsTool struct{ Svc WorkLogReadSvc }

func (t *ListWorklogsTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "list_worklogs",
		Function: agent.FunctionSpec{
			Name:        "list_worklogs",
			Description: "List work logs in a YYYY-MM-DD date range (inclusive).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"from": map[string]any{"type": "string", "description": "YYYY-MM-DD inclusive"},
					"to":   map[string]any{"type": "string", "description": "YYYY-MM-DD inclusive"},
				},
				"required": []any{"from", "to"},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *ListWorklogsTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.From == "" {
		return nil, fmt.Errorf("schema: missing required field from")
	}
	if in.To == "" {
		return nil, fmt.Errorf("schema: missing required field to")
	}
	logs, err := t.Svc.ListWorkLogs(in.From, in.To)
	if err != nil {
		return nil, err
	}
	return map[string]any{"from": in.From, "to": in.To, "logs": logs, "count": len(logs)}, nil
}

func (t *ListWorklogsTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}

// =====================================================================
// generate_work_report
// =====================================================================

type GenerateWorkReportTool struct{ Svc WorkLogReportSvc }

func (t *GenerateWorkReportTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "generate_work_report",
		Function: agent.FunctionSpec{
			Name:        "generate_work_report",
			Description: "Generate a period work report (weekly|monthly|halfyear|yearly) via AI. period_key e.g. 2026-W32 / 2026-07 / 2026-H1 / 2026.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type":       map[string]any{"type": "string", "description": "weekly|monthly|halfyear|yearly"},
					"period_key": map[string]any{"type": "string"},
				},
				"required": []any{"type", "period_key"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *GenerateWorkReportTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Type      string `json:"type"`
		PeriodKey string `json:"period_key"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	rt, ok := reportTypes[in.Type]
	if !ok {
		return nil, fmt.Errorf("schema: type must be weekly|monthly|halfyear|yearly, got %q", in.Type)
	}
	if in.PeriodKey == "" {
		return nil, fmt.Errorf("schema: missing required field period_key")
	}
	return t.Svc.GenerateReport(service.GenerateReportInput{Type: rt, PeriodKey: in.PeriodKey})
}

func (t *GenerateWorkReportTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		Type      string `json:"type"`
		PeriodKey string `json:"period_key"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "generate_work_report", "type": in.Type, "period_key": in.PeriodKey}, nil
}

// =====================================================================
// get_work_report
// =====================================================================

type GetWorkReportTool struct{ Svc WorkLogReportSvc }

func (t *GetWorkReportTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "get_work_report",
		Function: agent.FunctionSpec{
			Name:        "get_work_report",
			Description: "Get one period report by type+period_key, or (when period_key omitted) list all reports of that type.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"type":       map[string]any{"type": "string", "description": "weekly|monthly|halfyear|yearly"},
					"period_key": map[string]any{"type": "string", "description": "omit to list all of this type"},
				},
				"required": []any{"type"},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *GetWorkReportTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Type      string `json:"type"`
		PeriodKey string `json:"period_key"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	rt, ok := reportTypes[in.Type]
	if !ok {
		return nil, fmt.Errorf("schema: type must be weekly|monthly|halfyear|yearly, got %q", in.Type)
	}
	if in.PeriodKey == "" {
		list, err := t.Svc.ListReports(rt)
		if err != nil {
			return nil, err
		}
		return map[string]any{"list": list, "count": len(list)}, nil
	}
	return t.Svc.GetReport(rt, in.PeriodKey)
}

func (t *GetWorkReportTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}
```

> `worklog.go` currently imports `"ticktask/internal/model"`. `GetWorklogTool`/`ListWorklogsTool` need `time` for date parsing — add `"time"` to the import block if not present (it is NOT currently imported in worklog.go; the report tools don't need it but get_worklog validates the date). Add it.

- [ ] **Step 8: Register** in `register.go` after the work-log tools:

```go
	reg.MustRegister(&GetWorklogTool{Svc: deps.WorkLog})
	reg.MustRegister(&ListWorklogsTool{Svc: deps.WorkLog})
	reg.MustRegister(&GenerateWorkReportTool{Svc: deps.WorkLog})
	reg.MustRegister(&GetWorkReportTool{Svc: deps.WorkLog})
```

- [ ] **Step 9: Run tests → pass**, then `go build ./...`.

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestGetWorkLog|TestListWorklogs|TestGenerateWorkReport|TestGetWorkReport' -v`

- [ ] **Step 10: Commit**

```bash
git add backend/internal/agent/tools/worklog.go backend/internal/agent/tools/worklog_test.go backend/internal/agent/tools/register.go backend/internal/agent/tools/register_test.go
git commit -m "feat(agent): add worklog reads + report tools (P1)"
```

---

## Task 5: P2 — WorkLog writes (`update_worklog`, `update_worklog_summary`, `add_worklog_entry`)

**Files:** Modify `worklog.go`, `worklog_test.go`, `register.go`, `register_test.go`.

- [ ] **Step 1: Add `WorkLogWriteSvc` interface** to `worklog.go`:

```go
// WorkLogWriteSvc covers editing existing daily logs.
type WorkLogWriteSvc interface {
	UpdateWorkLog(input service.SaveWorkLogInput) (*model.WorkLog, error)
	UpdateSummary(date string, summary string) error
	AddQuickEntry(date string, in service.CreateQuickEntryInput) (*model.WorkItem, error)
}
```

Then add it to the `Deps.WorkLog` union in `register.go`:

```go
	WorkLog interface {
		WorkLogStructureSvc
		WorkLogSaveSvc
		WorkLogReadSvc
		WorkLogReportSvc
		WorkLogWriteSvc
	}
```

- [ ] **Step 2: Add mock + widen `mockWorkLogSvc`** in `worklog_test.go` / `register_test.go`:

```go
// in worklog_test.go
type mockWorkLogWriteSvc struct {
	updIn     *service.SaveWorkLogInput
	updOut    *model.WorkLog
	updErr    error
	sumDates  []string
	sumVals   []string
	sumErr    error
	qAddOut   *model.WorkItem
	qAddErr   error
	qAddDates []string
}

func (m *mockWorkLogWriteSvc) UpdateWorkLog(input service.SaveWorkLogInput) (*model.WorkLog, error) {
	m.updIn = &input
	if m.updErr != nil {
		return nil, m.updErr
	}
	return m.updOut, nil
}
func (m *mockWorkLogWriteSvc) UpdateSummary(date string, summary string) error {
	m.sumDates = append(m.sumDates, date)
	m.sumVals = append(m.sumVals, summary)
	return m.sumErr
}
func (m *mockWorkLogWriteSvc) AddQuickEntry(date string, in service.CreateQuickEntryInput) (*model.WorkItem, error) {
	m.qAddDates = append(m.qAddDates, date)
	if m.qAddErr != nil {
		return nil, m.qAddErr
	}
	return m.qAddOut, nil
}
```

```go
// in register_test.go: embed + guard
type mockWorkLogSvc struct {
	mockWorkLogStructureSvc
	mockWorkLogSaveSvc
	mockWorkLogReadSvc
	mockWorkLogReportSvc
	mockWorkLogWriteSvc
}
// add to the var(...) block:
//   _ WorkLogWriteSvc = (*mockWorkLogSvc)(nil)
```

- [ ] **Step 3: Write failing tests** — append to `worklog_test.go`:

```go
func TestUpdateWorklogSummary_Delegates(t *testing.T) {
	svc := &mockWorkLogWriteSvc{}
	tool := &UpdateWorklogSummaryTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"2026-08-08","summary":"今天主要搞了 agent 工具"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(svc.sumDates) != 1 || svc.sumVals[0] != "今天主要搞了 agent 工具" {
		t.Errorf("summary forwarded = %+v / %+v", svc.sumDates, svc.sumVals)
	}
}

func TestAddWorklogEntry_Delegates(t *testing.T) {
	svc := &mockWorkLogWriteSvc{qAddOut: &model.WorkItem{ID: "wi-1"}}
	tool := &AddWorklogEntryTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"2026-08-09","activity":"写文档","start_time":"2026-08-09T10:00:00Z","end_time":"2026-08-09T11:00:00Z"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(svc.qAddDates) != 1 || svc.qAddDates[0] != "2026-08-09" {
		t.Errorf("dates = %+v", svc.qAddDates)
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"2026-08-09"}`)); err == nil {
		t.Error("expected error for missing activity")
	}
}

func TestUpdateWorklog_Delegates(t *testing.T) {
	svc := &mockWorkLogWriteSvc{updOut: &model.WorkLog{Date: "2026-08-08"}}
	tool := &UpdateWorklogTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"2026-08-08","items":[]}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.updIn == nil || svc.updIn.Date != "2026-08-08" {
		t.Errorf("UpdateWorkLog input = %+v", svc.updIn)
	}
}
```

- [ ] **Step 4: Run tests → fail** (`undefined: UpdateWorklogSummaryTool` etc.).

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestUpdateWorklog|TestAddWorklogEntry' -v`

- [ ] **Step 5: Implement the three tools** — append to `worklog.go`:

```go
// =====================================================================
// update_worklog (full upsert)
// =====================================================================

type UpdateWorklogTool struct{ Svc WorkLogWriteSvc }

func (t *UpdateWorklogTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "update_worklog",
		Function: agent.FunctionSpec{
			Name:        "update_worklog",
			Description: "Upsert a work log for a date (replaces items). Use update_worklog_summary to edit only the summary without touching items.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"date":    map[string]any{"type": "string", "description": "YYYY-MM-DD"},
					"summary": map[string]any{"type": "string"},
					"items":   map[string]any{"type": "array", "items": map[string]any{"type": "object"}},
				},
				"required": []any{"date"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *UpdateWorklogTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Date    string         `json:"date"`
		Summary string         `json:"summary"`
		Items   []map[string]any `json:"items"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.Date == "" {
		return nil, fmt.Errorf("schema: missing required field date")
	}
	// Re-marshal items into []service.SaveItemInput to keep types honest.
	raw, _ := json.Marshal(in.Items)
	var items []service.SaveItemInput
	if len(in.Items) > 0 {
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, fmt.Errorf("parse items: %w", err)
		}
	}
	return t.Svc.UpdateWorkLog(service.SaveWorkLogInput{Date: in.Date, Summary: in.Summary, Items: items})
}

func (t *UpdateWorklogTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		Date string `json:"date"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "update_worklog", "date": in.Date}, nil
}

// =====================================================================
// update_worklog_summary
// =====================================================================

type UpdateWorklogSummaryTool struct{ Svc WorkLogWriteSvc }

func (t *UpdateWorklogSummaryTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "update_worklog_summary",
		Function: agent.FunctionSpec{
			Name:        "update_worklog_summary",
			Description: "Edit only the summary line of an existing work log (items untouched).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"date":    map[string]any{"type": "string", "description": "YYYY-MM-DD"},
					"summary": map[string]any{"type": "string"},
				},
				"required": []any{"date", "summary"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *UpdateWorklogSummaryTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Date    string `json:"date"`
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.Date == "" {
		return nil, fmt.Errorf("schema: missing required field date")
	}
	if in.Summary == "" {
		return nil, fmt.Errorf("schema: missing required field summary")
	}
	if err := t.Svc.UpdateSummary(in.Date, in.Summary); err != nil {
		return nil, err
	}
	return map[string]any{"date": in.Date, "updated": true}, nil
}

func (t *UpdateWorklogSummaryTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		Date string `json:"date"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "update_worklog_summary", "date": in.Date}, nil
}

// =====================================================================
// add_worklog_entry
// =====================================================================

type AddWorklogEntryTool struct{ Svc WorkLogWriteSvc }

func (t *AddWorklogEntryTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "add_worklog_entry",
		Function: agent.FunctionSpec{
			Name:        "add_worklog_entry",
			Description: "Append one manual entry to a date's work log (creates the log if needed). activity is required; start_time/end_time are RFC3339.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"date":          map[string]any{"type": "string", "description": "YYYY-MM-DD"},
					"activity":      map[string]any{"type": "string"},
					"start_time":    map[string]any{"type": "string", "description": "RFC3339"},
					"end_time":      map[string]any{"type": "string", "description": "RFC3339"},
					"quadrant":      map[string]any{"type": "integer", "description": "1-4"},
					"content":       map[string]any{"type": "string"},
					"problem_solved": map[string]any{"type": "string"},
					"result":        map[string]any{"type": "string"},
					"impact":        map[string]any{"type": "string"},
				},
				"required": []any{"date", "activity"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *AddWorklogEntryTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Date          string  `json:"date"`
		Activity      string  `json:"activity"`
		StartTime     *string `json:"start_time"`
		EndTime       *string `json:"end_time"`
		Quadrant      *int    `json:"quadrant"`
		Content       *string `json:"content"`
		ProblemSolved *string `json:"problem_solved"`
		Result        *string `json:"result"`
		Impact        *string `json:"impact"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.Date == "" {
		return nil, fmt.Errorf("schema: missing required field date")
	}
	if in.Activity == "" {
		return nil, fmt.Errorf("schema: missing required field activity")
	}
	ci := service.CreateQuickEntryInput{Activity: in.Activity}
	if in.StartTime != nil {
		ci.StartTime = *in.StartTime
	}
	if in.EndTime != nil {
		ci.EndTime = *in.EndTime
	}
	if in.Quadrant != nil {
		ci.Quadrant = *in.Quadrant
	}
	if in.Content != nil {
		ci.Content = *in.Content
	}
	if in.ProblemSolved != nil {
		ci.ProblemSolved = *in.ProblemSolved
	}
	if in.Result != nil {
		ci.Result = *in.Result
	}
	if in.Impact != nil {
		ci.Impact = *in.Impact
	}
	return t.Svc.AddQuickEntry(in.Date, ci)
}

func (t *AddWorklogEntryTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		Date     string `json:"date"`
		Activity string `json:"activity"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "add_worklog_entry", "date": in.Date, "activity": in.Activity}, nil
}
```

- [ ] **Step 6: Register** in `register.go`:

```go
	reg.MustRegister(&UpdateWorklogTool{Svc: deps.WorkLog})
	reg.MustRegister(&UpdateWorklogSummaryTool{Svc: deps.WorkLog})
	reg.MustRegister(&AddWorklogEntryTool{Svc: deps.WorkLog})
```

- [ ] **Step 7: Run tests → pass**, then `go build ./...`.

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestUpdateWorklog|TestAddWorklogEntry' -v`

- [ ] **Step 8: Commit**

```bash
git add backend/internal/agent/tools/worklog.go backend/internal/agent/tools/worklog_test.go backend/internal/agent/tools/register.go backend/internal/agent/tools/register_test.go
git commit -m "feat(agent): add worklog write tools (P2)"
```

---

## Task 6: P1/P2 — Analytics `get_analytics`

One parametric tool dispatching to `GetTrend/GetDistribution/GetPomodoroByTask/GetPomodoroTrends`.

**Files:** Modify `insight.go`, `insight_test.go`, `register.go`.

- [ ] **Step 1: Extend `AnalyticsService` interface** in `insight.go`:

```go
type AnalyticsService interface {
	GetSummary(date time.Time) (*service.DailySummary, error)
	GetTrend(days int) (*service.TrendData, error)
	GetDistribution(start, end time.Time) (*service.DistributionStats, error)
	GetPomodoroByTask(period string) (*service.PomodoroByTaskResult, error)
	GetPomodoroTrends(period string) (*service.PomodoroTrendsResult, error)
}
```

- [ ] **Step 2: Extend `mockAnalyticsSvc`** in `insight_test.go` — add fields + 4 methods (read the existing struct first to preserve fields):

```go
	// add fields
	trendOut   *service.TrendData
	trendErr   error
	trendDays  int
	distOut    *service.DistributionStats
	distErr    error
	byTaskOut  *service.PomodoroByTaskResult
	byTaskErr  error
	pTrendsOut *service.PomodoroTrendsResult
	pTrendsErr error
```

```go
func (m *mockAnalyticsSvc) GetTrend(days int) (*service.TrendData, error) {
	m.trendDays = days
	return m.trendOut, m.trendErr
}
func (m *mockAnalyticsSvc) GetDistribution(start, end time.Time) (*service.DistributionStats, error) {
	return m.distOut, m.distErr
}
func (m *mockAnalyticsSvc) GetPomodoroByTask(period string) (*service.PomodoroByTaskResult, error) {
	return m.byTaskOut, m.byTaskErr
}
func (m *mockAnalyticsSvc) GetPomodoroTrends(period string) (*service.PomodoroTrendsResult, error) {
	return m.pTrendsOut, m.pTrendsErr
}
```

- [ ] **Step 3: Write failing test** — append to `insight_test.go`:

```go
func TestGetAnalytics_DispatchesByMetric(t *testing.T) {
	cases := []struct {
		args   string
		metric string
	}{
		{`{"metric":"trend","days":7}`, "trend"},
		{`{"metric":"distribution","from":"2026-08-01","to":"2026-08-09"}`, "distribution"},
		{`{"metric":"pomodoro_by_task","period":"week"}`, "pomodoro_by_task"},
		{`{"metric":"pomodoro_trends","period":"week"}`, "pomodoro_trends"},
	}
	for _, c := range cases {
		svc := &mockAnalyticsSvc{
			trendOut: &service.TrendData{}, distOut: &service.DistributionStats{},
			byTaskOut: &service.PomodoroByTaskResult{}, pTrendsOut: &service.PomodoroTrendsResult{},
		}
		tool := &GetAnalyticsTool{Svc: svc}
		res, err := tool.Execute(context.Background(), json.RawMessage(c.args))
		if err != nil {
			t.Errorf("metric %s: %v", c.metric, err)
		}
		m, _ := json.Marshal(res)
		if !strings.Contains(string(m), `"metric":"`+c.metric+`"`) {
			t.Errorf("metric %s: result should echo metric, got %s", c.metric, m)
		}
	}
	// bad metric rejected
	svc := &mockAnalyticsSvc{}
	tool := &GetAnalyticsTool{Svc: svc}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"metric":"bogus"}`)); err == nil {
		t.Error("expected error for bad metric")
	}
}
```

- [ ] **Step 4: Run test → fail** (`undefined: GetAnalyticsTool`).

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestGetAnalytics' -v`

- [ ] **Step 5: Implement** — append to `insight.go`:

```go
// =====================================================================
// get_analytics
// =====================================================================

// GetAnalyticsTool dispatches to one of four analytics queries by `metric`.
// PermRead. (get_daily_insights stays the dedicated daily-summary tool.)
type GetAnalyticsTool struct{ Svc AnalyticsService }

func (t *GetAnalyticsTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "get_analytics",
		Function: agent.FunctionSpec{
			Name:        "get_analytics",
			Description: "Query analytics: metric=trend (days N), distribution (from/to YYYY-MM-DD), pomodoro_by_task (period week|month), or pomodoro_trends (period week|month).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"metric": map[string]any{"type": "string", "description": "trend|distribution|pomodoro_by_task|pomodoro_trends"},
					"days":   map[string]any{"type": "integer", "description": "trend: number of days"},
					"from":   map[string]any{"type": "string", "description": "distribution: YYYY-MM-DD"},
					"to":     map[string]any{"type": "string", "description": "distribution: YYYY-MM-DD"},
					"period": map[string]any{"type": "string", "description": "pomodoro_by_task/pomodoro_trends: week|month"},
				},
				"required": []any{"metric"},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *GetAnalyticsTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Metric string  `json:"metric"`
		Days   *int    `json:"days"`
		From   *string `json:"from"`
		To     *string `json:"to"`
		Period *string `json:"period"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	loc := time.Now().Location()
	switch in.Metric {
	case "trend":
		days := 7
		if in.Days != nil && *in.Days > 0 {
			days = *in.Days
		}
		out, err := t.Svc.GetTrend(days)
		if err != nil {
			return nil, err
		}
		return map[string]any{"metric": "trend", "data": out}, nil
	case "distribution":
		from, to, err := parseRange(in.From, in.To, loc)
		if err != nil {
			return nil, err
		}
		out, err := t.Svc.GetDistribution(from, to)
		if err != nil {
			return nil, err
		}
		return map[string]any{"metric": "distribution", "data": out}, nil
	case "pomodoro_by_task":
		out, err := t.Svc.GetPomodoroByTask(periodOrDefault(in.Period))
		if err != nil {
			return nil, err
		}
		return map[string]any{"metric": "pomodoro_by_task", "data": out}, nil
	case "pomodoro_trends":
		out, err := t.Svc.GetPomodoroTrends(periodOrDefault(in.Period))
		if err != nil {
			return nil, err
		}
		return map[string]any{"metric": "pomodoro_trends", "data": out}, nil
	default:
		return nil, fmt.Errorf("schema: metric must be trend|distribution|pomodoro_by_task|pomodoro_trends, got %q", in.Metric)
	}
}

func (t *GetAnalyticsTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}

// parseRange resolves distribution from/to (defaults: current week Mon..Sun).
func parseRange(from, to *string, loc *time.Location) (time.Time, time.Time, error) {
	parse := func(s string) (time.Time, error) {
		return time.ParseInLocation("2006-01-02", s, loc)
	}
	if from != nil && *from != "" && to != nil && *to != "" {
		f, err := parse(*from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from %q: %w", *from, err)
		}
		tt, err := parse(*to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to %q: %w", *to, err)
		}
		return f, tt, nil
	}
	now := time.Now()
	day := (int(now.Weekday()) + 6) % 7
	mon := time.Date(now.Year(), now.Month(), now.Day()-day, 0, 0, 0, 0, loc)
	return mon, mon.AddDate(0, 0, 7), nil
}

func periodOrDefault(p *string) string {
	if p != nil && *p != "" {
		return *p
	}
	return "week"
}
```

- [ ] **Step 6: Register** in `register.go` after the insight tool:

```go
	reg.MustRegister(&GetAnalyticsTool{Svc: deps.Analytics})
```

- [ ] **Step 7: Run test → pass**, then `go build ./...`.

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestGetAnalytics' -v`

- [ ] **Step 8: Commit**

```bash
git add backend/internal/agent/tools/insight.go backend/internal/agent/tools/insight_test.go backend/internal/agent/tools/register.go
git commit -m "feat(agent): add get_analytics parametric tool (P1/P2)"
```

---

## Task 7: P2 — Task `move_task`

**Files:** Modify `task.go`, `task_test.go`, `register.go`.

- [ ] **Step 1: Extend `TaskService` interface** in `task.go` (add after `GetAllTasks`):

```go
	MoveTask(id string, targetQuadrant model.Quadrant) error
```

- [ ] **Step 2: Extend `mockTaskSvc`** in `task_test.go` — add fields + method (read existing struct first):

```go
	// add fields
	moveErr   error
	moveIDs   []string
	moveQuads []model.Quadrant
```

```go
func (m *mockTaskSvc) MoveTask(id string, targetQuadrant model.Quadrant) error {
	m.moveIDs = append(m.moveIDs, id)
	m.moveQuads = append(m.moveQuads, targetQuadrant)
	return m.moveErr
}
```

- [ ] **Step 3: Write failing test** — append to `task_test.go`:

```go
func TestMoveTask_Delegates(t *testing.T) {
	svc := &mockTaskSvc{}
	tool := &MoveTaskTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"t-1","quadrant":2}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(svc.moveIDs) != 1 || svc.moveIDs[0] != "t-1" || svc.moveQuads[0] != model.Quadrant2 {
		t.Errorf("MoveTask args = %+v / %+v", svc.moveIDs, svc.moveQuads)
	}
	// out-of-range quadrant rejected
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"t-1","quadrant":9}`)); err == nil {
		t.Error("expected error for quadrant 9")
	}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"quadrant":2}`)); err == nil {
		t.Error("expected error for missing task_id")
	}
}
```

- [ ] **Step 4: Run test → fail** (`undefined: MoveTaskTool`).

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestMoveTask' -v`

- [ ] **Step 5: Implement** — append to `task.go`:

```go
// =====================================================================
// move_task
// =====================================================================

// MoveTaskTool moves a task to an Eisenhower quadrant (auto-syncs
// IsImportant/IsUrgent on the service side). PermWrite. Overlaps with
// update_task's priority field but exposes the cleaner "移到第二象限" semantic.
type MoveTaskTool struct{ Svc TaskService }

func (t *MoveTaskTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "move_task",
		Function: agent.FunctionSpec{
			Name:        "move_task",
			Description: "Move a task to a different Eisenhower quadrant (1 important+urgent … 4 neither).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task_id":   map[string]any{"type": "string"},
					"quadrant":  map[string]any{"type": "integer", "description": "1-4"},
				},
				"required": []any{"task_id", "quadrant"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *MoveTaskTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		TaskID    string `json:"task_id"`
		Quadrant  int    `json:"quadrant"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.TaskID == "" {
		return nil, fmt.Errorf("schema: missing required field task_id")
	}
	if in.Quadrant < 1 || in.Quadrant > 4 {
		return nil, fmt.Errorf("schema: quadrant must be 1-4, got %d", in.Quadrant)
	}
	if err := t.Svc.MoveTask(in.TaskID, model.Quadrant(in.Quadrant)); err != nil {
		return nil, err
	}
	return map[string]any{"task_id": in.TaskID, "quadrant": in.Quadrant, "moved": true}, nil
}

func (t *MoveTaskTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		TaskID   string `json:"task_id"`
		Quadrant int    `json:"quadrant"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "move_task", "task_id": in.TaskID, "quadrant": in.Quadrant}, nil
}
```

- [ ] **Step 6: Register** in `register.go` after the task tools:

```go
	reg.MustRegister(&MoveTaskTool{Svc: deps.Tasks})
```

- [ ] **Step 7: Run test → pass**, then `go build ./...`.

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestMoveTask' -v`

- [ ] **Step 8: Commit**

```bash
git add backend/internal/agent/tools/task.go backend/internal/agent/tools/task_test.go backend/internal/agent/tools/register.go
git commit -m "feat(agent): add move_task tool (P2)"
```

---

## Task 8: P2 — Settings `get_settings` (new file + Deps + main.go wiring)

**Files:** Create `settings.go`, `settings_test.go`; modify `register.go`, `register_test.go`, `cmd/server/main.go`.

- [ ] **Step 1: Write `settings.go`**:

```go
package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"ticktask/internal/agent"
	"ticktask/internal/model"
)

// SettingsReader is the subset of repository.SettingRepository methods the
// settings tool needs. The production repository.SettingRepository satisfies
// this implicitly (it has GetPomodoroSettings); main.go wires the concrete
// settingRepo into Deps.Settings.
type SettingsReader interface {
	GetPomodoroSettings() (*model.PomodoroSettings, error)
}

// GetSettingsTool returns the user's pomodoro settings (durations, breaks,
// automation toggles). PermRead. (AI settings are deliberately NOT exposed.)
type GetSettingsTool struct{ Svc SettingsReader }

func (t *GetSettingsTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "get_settings",
		Function: agent.FunctionSpec{
			Name:        "get_settings",
			Description: "Read the user's pomodoro settings (work/break durations, long-break cadence, auto-start toggles, sound).",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *GetSettingsTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	return t.Svc.GetPomodoroSettings()
}

func (t *GetSettingsTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}
```

- [ ] **Step 2: Write `settings_test.go`**:

```go
package tools

import (
	"context"
	"errors"
	"strings"
	"testing"

	"ticktask/internal/model"
)

type mockSettingsReader struct {
	out *model.PomodoroSettings
	err error
}

func (m *mockSettingsReader) GetPomodoroSettings() (*model.PomodoroSettings, error) {
	return m.out, m.err
}

func TestGetSettings_Delegates(t *testing.T) {
	svc := &mockSettingsReader{out: &model.PomodoroSettings{WorkDuration: 25, ShortBreakDuration: 5}}
	tool := &GetSettingsTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), `"WorkDuration":25`) {
		t.Errorf("result should include pomodoro settings: %s", m)
	}
}

func TestGetSettings_PreviewMirrorsExecute(t *testing.T) {
	svc := &mockSettingsReader{out: &model.PomodoroSettings{}}
	tool := &GetSettingsTool{Svc: svc}
	pv, err := tool.Preview(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if pv == nil {
		t.Fatal("preview should mirror execute for read tool")
	}
}

func TestGetSettings_ServiceError(t *testing.T) {
	svc := &mockSettingsReader{err: errors.New("db locked")}
	tool := &GetSettingsTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "db locked") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}
```

- [ ] **Step 3: Add `Settings` to `Deps`** in `register.go`:

```go
type Deps struct {
	Tasks     TaskService
	Timer     TimerService
	Schedule  ScheduleService
	Analytics AnalyticsService
	LLM       LLMClient
	WorkLog   interface {
		WorkLogStructureSvc
		WorkLogSaveSvc
		WorkLogReadSvc
		WorkLogReportSvc
		WorkLogWriteSvc
	}
	Settings SettingsReader
}
```

Register in `RegisterAll` (after work-log tools):

```go
	reg.MustRegister(&GetSettingsTool{Svc: deps.Settings})
```

- [ ] **Step 4: Wire `main.go`** — at `cmd/server/main.go` ~line 132, add `Settings: settingRepo,` to the `tools.Deps{...}` literal:

```go
	tools.RegisterAll(registry, tools.Deps{
		Tasks:     taskService,
		Timer:     timerService,
		Schedule:  scheduleService,
		Analytics: analyticsService,
		WorkLog:   workLogService,
		LLM:       llm,
		Settings:  settingRepo,
	})
```

> `settingRepo` is `repository.SettingRepository`, which has `GetPomodoroSettings() (*model.PomodoroSettings, error)` — satisfies `SettingsReader`.

- [ ] **Step 5: Update `mockWorkLogSvc`-style wiring** — `register_test.go`'s `newTestRegistry` must now pass `Settings`. Add a mock + pass it:

```go
// add near the other mock definitions in register_test.go
// (mockSettingsReader already lives in settings_test.go, same package — reuse it)

// in newTestRegistry, add to the Deps literal:
		Settings:  &mockSettingsReader{},
```

- [ ] **Step 6: Run tests → pass**, then `go build ./...` (this validates main.go wiring compiles).

Run: `cd backend && go test ./internal/agent/tools/ -run 'TestGetSettings' -v && go build ./...`

- [ ] **Step 7: Commit**

```bash
git add backend/internal/agent/tools/settings.go backend/internal/agent/tools/settings_test.go backend/internal/agent/tools/register.go backend/internal/agent/tools/register_test.go backend/cmd/server/main.go
git commit -m "feat(agent): add get_settings tool + wire settingRepo into Deps (P2)"
```

---

## Task 9: System prompt — id-domain guardrail + capability sentence

**Files:** Modify `backend/internal/agent/prompts.go`.

- [ ] **Step 1: Edit the `DefaultSystemPrompt`** — append a guardrail sentence before the closing backtick. Change the line at prompts.go:10 from:

```
When the user asks for an action you have a tool for (create/update/delete task or schedule, start/stop pomodoro, generate schedule, save worklog), CALL THE TOOL DIRECTLY.
```

to:

```
When the user asks for an action you have a tool for (create/update/delete task or schedule, start/resume/complete pomodoro, generate/revise schedule, worklog read/write/report, analytics), CALL THE TOOL DIRECTLY.
```

And insert this new paragraph before the final "Never claim..." paragraph:

```
Entity ids are domain-scoped: schedule ids come from list_schedule, task ids from list_tasks, worklog items from get_worklog. Never pass one domain's id to another domain's tool (e.g. do not call update_task with a schedule id). If unsure which entity the user means, list the relevant domain first to get the right id.
```

- [ ] **Step 2: Run the prompt test** (if `service_systemprompt_test.go` asserts prompt content, update its expectation; otherwise just ensure compile).

Run: `cd backend && go test ./internal/agent/ -run 'SystemPrompt' -v`
Expected: PASS (or update the assertion to match the new sentence).

- [ ] **Step 3: Commit**

```bash
git add backend/internal/agent/prompts.go backend/internal/agent/service_systemprompt_test.go
git commit -m "feat(agent): harden prompt with id-domain guardrail + new tool inventory"
```

---

## Task 10: Eval Layer 2 — WS real-behavior cases

**Files:** Modify `eval/cases.mjs`, create `eval/themes/schedule-lifecycle.mjs`, modify `eval/themes/index.mjs`, `eval/seed.mjs`.

- [ ] **Step 1: Add new write tools to `WRITE_TOOLS`** in `eval/cases.mjs`:

```js
const WRITE_TOOLS = new Set([
  'create_task', 'update_task', 'delete_task', 'delete_schedule',
  'update_schedule', 'create_schedule', 'revise_schedule', 'apply_schedule_revision',
  'start_pomodoro', 'stop_pomodoro', 'control_pomodoro', 'generate_schedule',
  'generate_work_report', 'update_worklog', 'update_worklog_summary', 'add_worklog_entry',
  'save_worklog', 'classify_task', 'move_task',
]);
```

- [ ] **Step 2: Seed a known schedule + worklog** in `eval/seed.mjs` (append to the seeding logic; use today's date). Add a schedule with a stable title the routing/dbVerify cases target:

```js
// seed a schedule for today so update_schedule mark-complete can dbVerify it
const todayStr = new Date().toISOString().slice(0, 10);
await fetch(`${BASE}/api/schedules`, {
  method: 'POST', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({
    title: '审查 PR-1234',
    start_time: `${todayStr}T09:00:00Z`,
    end_time:   `${todayStr}T10:00:00Z`,
    type: 'task',
  }),
});
```

- [ ] **Step 3: Create `eval/themes/schedule-lifecycle.mjs`**:

```js
import { called, tools, txt, notFabricated, today } from '../lib/helpers.mjs';

// Real-behavior cases for the new schedule/timer/worklog/analytics tools.
// Drives the live backend; uses confirm + dbVerify where noted.
export const CASES = [
  // --- routing ---
  { cat: 'schedule-lifecycle', prompt: '把审查 PR-1234 那个日程标记完成',
    check: r => [called(r, 'update_schedule'), 'update_schedule'] },
  { cat: 'schedule-lifecycle', prompt: '明天下午3点加个会和产品对一下',
    check: r => [called(r, 'create_schedule'), 'create_schedule'] },
  { cat: 'schedule-lifecycle', prompt: '继续番茄钟',
    check: r => [called(r, 'control_pomodoro'), 'control_pomodoro'] },
  { cat: 'schedule-lifecycle', prompt: '今天我专注了多久了',
    check: r => [called(r, 'get_pomodoro_stats') || called(r, 'get_daily_insights'), 'focus stats'] },
  { cat: 'schedule-lifecycle', prompt: '看看我昨天的工作日志',
    check: r => [called(r, 'get_worklog') || called(r, 'list_worklogs'), 'worklog read'] },
  { cat: 'schedule-lifecycle', prompt: '帮我生成本周的周报',
    check: r => [called(r, 'generate_work_report'), 'generate_work_report'] },
  { cat: 'schedule-lifecycle', prompt: '最近一周专注趋势怎么样',
    check: r => [called(r, 'get_analytics') || called(r, 'get_daily_insights'), 'analytics'] },

  // --- ORIGINAL-BUG REGRESSION GUARD ---
  // Referring to a schedule + "已完成" must route to update_schedule and NOT
  // misuse update_task (the failure that motivated this whole change).
  { cat: 'schedule-lifecycle', prompt: '审查 PR-1234 这个我已经完成了',
    check: r => {
      const usedUpdateSchedule = called(r, 'update_schedule');
      const misusedUpdateTask = tools(r).some(t => t.name === 'update_task');
      const ok = usedUpdateSchedule && !misusedUpdateTask;
      return [ok, ok ? 'routed to update_schedule, not update_task' : 'regressed: misused update_task or missed update_schedule'];
    },
    note: 'original-bug regression guard' },

  // --- confirm + dbVerify (real post-action DB state) ---
  { cat: 'schedule-lifecycle', prompt: '把审查 PR-1234 标记为已完成',
    confirm: 'approve', dbVerify: 'schedules',
    check: (r, ctx) => {
      const scheds = (ctx && ctx.dbState) || [];
      const target = scheds.find(s => (s.title || '').includes('审查 PR-1234'));
      const ok = !!target && target.status === 'completed';
      return [ok, ok ? 'schedule status=completed in DB' : `status=${target && target.status}`];
    },
    note: 'dbVerify: update_schedule → completed' },
  { cat: 'schedule-lifecycle', prompt: '把审查 PR-1234 标记为已完成',
    confirm: 'reject', dbVerify: 'schedules',
    check: (r, ctx) => {
      const scheds = (ctx && ctx.dbState) || [];
      const target = scheds.find(s => (s.title || '').includes('审查 PR-1234'));
      const ok = !!target && target.status !== 'completed';
      return [ok, ok ? 'rejected: status unchanged' : 'regressed: reject still applied'];
    },
    note: 'dbVerify: reject leaves DB unchanged' },

  // --- revise two-step (multi-turn) ---
  { cat: 'schedule-lifecycle',
    turns: ['优化一下今天的安排', '就按这个改吧'],
    confirm: 'approve', dbVerify: 'schedules',
    check: (r, ctx) => {
      const ok = called(r, 'revise_schedule') && called(r, 'apply_schedule_revision');
      return [ok, ok ? 'revised + applied' : 'did not complete revise→apply'];
    },
    note: 'two-step revise→apply (turns)' },
];
```

> **Confirm the `check(r, ctx)` signature.** `run-cases.mjs` executes each case and builds `{ r, ctx }` with `ctx.dbState` set when `dbVerify` is present. Verify how `check` is invoked (it may be `check(r, ctx)` or the dbState may be merged into `r`) by reading the `runCase` function in `eval/run-cases.mjs` (~line 61–100) before finalizing the case bodies — adapt the argument access to match. This is the one place to verify against the runner, not assume.

- [ ] **Step 4: Register the theme** in `eval/themes/index.mjs` (follow how existing themes are exported/combined — add `schedule-lifecycle` to the aggregated list).

- [ ] **Step 5: Run the eval** (live backend + real LLM, ~minutes for this subset):

```bash
cd eval && npm install
AGENT_BASE_URL=http://localhost:8080 node seed.mjs
# run only the new theme (filter by cat if the runner supports it; else full run)
AGENT_BASE_URL=http://localhost:8080 npm run cases
```
Expected: routing cases pass; the bug-regression case passes (`update_schedule`, not `update_task`); dbVerify cases show `status=completed` after approve and unchanged after reject.

- [ ] **Step 6: Commit**

```bash
git add eval/cases.mjs eval/themes/schedule-lifecycle.mjs eval/themes/index.mjs eval/seed.mjs
git commit -m "test(eval): add schedule-lifecycle real-behavior cases (routing+confirm+dbVerify+bug guard)"
```

---

## Task 11: Eval Layer 3 — multiturn guard for a new write tool

**Files:** Modify `eval/multiturn-test.mjs`.

- [ ] **Step 1: Read the existing `multiturn-test.mjs`** to mirror its structure (it creates a conversation, calls a write tool, `/confirm` approves, then sends a follow-up and asserts the reply mentions the entity). Add a parallel scenario using `create_schedule`:

```js
// Append a create_schedule scenario alongside the existing create_task one.
// 1. create a conversation
// 2. prompt: '明天下午3点加个会叫「产品评审」'  → expect pending_confirmation on create_schedule
// 3. POST /confirm approve
// 4. follow-up in SAME conversation: '这个会几点来着？' → assert reply mentions 15:00 / 下午3点 / 产品评审
```

Implement it by copying the existing scenario function, swapping the prompt and the assertion substring. The assertion must confirm the agent can still answer coherently after the new write tool fired (guards the historical "multi-turn breakage after a write" regression).

- [ ] **Step 2: Run it**:

```bash
cd eval && AGENT_BASE_URL=http://localhost:8080 node multiturn-test.mjs
```
Expected: the create_schedule scenario passes (agent answers the follow-up coherently).

- [ ] **Step 3: Commit**

```bash
git add eval/multiturn-test.mjs
git commit -m "test(eval): add create_schedule multi-turn guard"
```

---

## Task 12: Final verification + README re-baseline

- [ ] **Step 1: Full backend test + build**

```bash
cd backend && go test ./... && go build ./...
```
Expected: all PASS, compiles.

- [ ] **Step 2: Confirm the registry count** — update the `register.go` doc comment to the final tally: `30 tools: 5 task + 3 timer + 5 schedule + 1 insight + 1 analytics + 3 worklog-read/report... ` — actually restate accurately as `30 tools total (5 task, 3 timer, 6 schedule [generate/list/delete/update/create/revise/apply — note: 7], ...)`. **Count carefully and write the exact per-domain breakdown** so the comment is not stale again. (Schedule = generate/list/delete/update/create/revise/apply = 7; total 14+16=30.)

- [ ] **Step 3: Re-baseline `eval/README.md`** — update the runnable/total counts and the baseline pass-rate after the full eval run.

- [ ] **Step 4: E2E in the running app** — restart backend per the restart rule, open the agent, and reproduce the originating request positively: refer to a schedule and say "我已经完成了" → confirm → schedule status flips to completed (no `record not found`, no `update_task`).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/agent/tools/register.go eval/README.md
git commit -m "docs: finalize tool-count comment + re-baseline eval README"
```

---

## Self-Review (completed during authoring)

**1. Spec coverage:** Every tool in the spec's inventory table maps to a task — Schedule (T1/T2), Timer (T3), WorkLog reads+report (T4) + writes (T5), Analytics (T6), Task (T7), Settings (T8); prompt guardrail (T9); eval Layers 2 (T10) and 3 (T11); verification (T12). All 16 tools present. ✓

**2. Placeholder scan:** One intentional "verify against the runner" note in T10 Step 3 (the `check(r, ctx)` signature) — this is a real verify-then-adapt instruction, not a placeholder; the case bodies are complete. No "TBD"/"implement later". Code blocks present for every implementation step. ✓

**3. Type consistency:** `model.Quadrant` / `Quadrant1-4` (T7), `model.WorkReportType` + `ReportWeekly/Monthly/HalfYear/Yearly` via `reportTypes` map (T4/T5), `service.GenerateReportInput{Type,PeriodKey,Force}`, `service.CreateQuickEntryInput` fields, `service.SaveWorkLogInput`, `service.TaskTimeStats`, `service.ScheduleEvent`, `service.UpdateScheduleDTO`/`CreateScheduleDTO`/`ReviseResponse` — all match the audited signatures. Mock names match existing (`mockScheduleSvc`, `mockTimerSvc`, `mockTaskSvc`, `mockAnalyticsSvc`, `mockWorkLogStructureSvc/Save` + new `Read/Report/Write`, `mockWorkLogSvc`). `Deps.WorkLog` union grows consistently across T4→T5. ✓

**Known follow-up (not blockers):** the `check(r, ctx)` signature in the runner (T10) and the exact `mockTimerSvc`/`mockTaskSvc` field names (T3/T7) must be confirmed by reading the respective `_test.go` files at execution time — both are flagged in-task.
