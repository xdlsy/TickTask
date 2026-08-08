# Agent 端到端验证（P1）实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 TickTask 对话式 Agent 建立可重复验证「自然语言 → 正确工具」的能力，先交付 P1：内部 `TraceRecorder` 录点 + Promptfoo eval 骨架（自定义 WS provider + seed + 12 条意图 case）跑通 canonical 用例。

**Architecture:** 在 `agent.runTurn` 注入一个 nil 安全的 `TraceRecorder`（与现有 `HubBroadcaster` 同样的注入模式），把每轮 `TurnTrace` 落到 JSONL（L3 可观测 + 将来 VCR cassette 源）。新建顶层 `eval/` 目录，用 Promptfoo 自定义 provider 黑盒驱动 `/api/agent/*`，从 WS 事件重建 `{tool_calls, assistant_text}` 做结构化断言。L2 不进默认 `go test`，改 prompt/工具时手动跑。

**Tech Stack:** Go 1.21（`testing` + 内存 SQLite，复用 `service_test.go` 的 `mockLLM`/`mockHub`/`newInMemoryRepo`/`fakeTool`）｜Node（`promptfoo` + `ws`，Node 内置 `http`/`node:test`/`fetch`）

**Spec:** `docs/superpowers/specs/2026-08-09-agent-e2e-verification-design.md`（分支 `evolve/agent-e2e-verification`）

**P2（不在本计划）：** `langfuseTracer` + `docker-compose.langfuse.yml` + gated Playwright 纵向 spec，另起计划。

---

## 文件结构（P1）

**新建（Go）：**
- `backend/internal/agent/trace.go` — `TraceRecorder` 接口、`TurnTrace`/`TraceStep` 类型、`noopTracer`、`jsonTracer`、`selectTracer`
- `backend/internal/agent/trace_test.go` — `selectTracer` 与 `jsonTracer` 测试
- `backend/internal/agent/service_trace_test.go` — L1 录制回归测试（注入 capture tracer，断言 `TurnTrace` 字段）

**修改（Go）：**
- `backend/internal/agent/service.go` — `AgentDeps` 加 `Tracer` 字段；`NewAgentService` 注入 noop 默认；`runTurn` 线程化 `userText`、构建 trace、`defer RecordTurn`、`traceStep` 辅助函数、在各终态 append；`SendMessage` 透传 `text`
- `backend/cmd/server/main.go:140-145` — `AgentDeps` 注入 `selectTracer(os.Getenv("TICKTASK_TRACE_DIR"))`

**新建（eval/）：**
- `eval/package.json` — `promptfoo`、`ws` 依赖
- `eval/provider.mjs` — 自定义 WS provider（`callApi` + 可测的 `collect`）
- `eval/provider.test.mjs` — provider 元测试（mock http+ws，断言事件重建）
- `eval/seed.mjs` — 幂等 seed 今天的日程 + 任务
- `eval/promptfooconfig.yaml` — provider + 12 条 case + 断言
- `eval/README.md` — 如何运行

**修改：**
- `.gitignore` — 忽略 `backend/internal/agent/testdata/traces/*.jsonl`、`eval/node_modules/`、`eval/.promptfoo/`

---

## Task 1: `TraceRecorder` 类型 + noop + `selectTracer`

**Files:**
- Create: `backend/internal/agent/trace.go`
- Test: `backend/internal/agent/trace_test.go`

- [ ] **Step 1: 写 `selectTracer` 的失败测试**

Create `backend/internal/agent/trace_test.go`:

```go
package agent

import (
	"os"
	"testing"
)

func TestSelectTracer_DefaultsToNoop(t *testing.T) {
	got := selectTracer("")
	if _, ok := got.(noopTracer); !ok {
		t.Fatalf("selectTracer(\"\") = %T, want noopTracer", got)
	}
}

func TestSelectTracer_NonEmptyReturnsJsonTracer(t *testing.T) {
	got := selectTracer(t.TempDir())
	jt, ok := got.(*jsonTracer)
	if !ok {
		t.Fatalf("selectTracer(dir) = %T, want *jsonTracer", got)
	}
	if jt.dir == "" {
		t.Fatal("jsonTracer.dir is empty")
	}
}

// noopTracer.RecordTurn must be a no-op (no panic, no file writes).
func TestNoopTracer_DoesNotPanic(t *testing.T) {
	noopTracer{}.RecordTurn("any", TurnTrace{UserText: "x"})
	// 无文件应被创建：用 selectTracer("") 后调 RecordTurn，断言目录为空
	dir := t.TempDir()
	selectTracer("").(noopTracer).RecordTurn("c", TurnTrace{})
	if ents, _ := os.ReadDir(dir); len(ents) != 0 {
		t.Fatalf("noop wrote files: %v", ents)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `cd backend && go test -run TestSelectTracer -v ./internal/agent/`
Expected: 编译失败 / FAIL —— `selectTracer`、`noopTracer`、`jsonTracer`、`TurnTrace` 未定义。

- [ ] **Step 3: 写最小实现**

Create `backend/internal/agent/trace.go`:

```go
package agent

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"

	"ticktask/internal/ai"
)

// TraceRecorder observes a completed agent turn as a structured TurnTrace.
// Implementations must be safe for concurrent use across conversations.
// It is a nil-safe byproduct injected alongside HubBroadcaster: the noop
// default records nothing, so production and existing tests are unaffected
// unless a real tracer is wired in.
type TraceRecorder interface {
	RecordTurn(convID string, trace TurnTrace)
}

// TraceStep captures one tool-call decision plus its execution outcome. The
// ToolCall (model-requested name+args) is the decision evidence L2 asserts on;
// Status/Result/Error record what happened when it ran.
type TraceStep struct {
	ToolCall   ai.ToolCall
	Permission ToolPermission
	Status     string // started | pending_confirmation | rejected | succeeded | failed
	Result     any
	Error      string
}

// TurnTrace is the full record of one runTurn invocation: the user message
// that triggered it, every assistant text chunk concatenated, and one step per
// tool call. JSONL-serialized by jsonTracer; also the future VCR cassette shape.
type TurnTrace struct {
	ConversationID string
	UserText       string
	AssistantText  string
	Steps          []TraceStep
}

// noopTracer discards everything. NewAgentService injects it as the default so
// the field is never nil and existing call sites need no nil checks.
type noopTracer struct{}

func (noopTracer) RecordTurn(string, TurnTrace) {}

// jsonTracer appends one JSONL line per turn to <dir>/<conversation_id>.jsonl.
// Phase-1 observability sink + future record/replay cassette source. A mutex
// serializes appends to the same conversation file across concurrent turns.
type jsonTracer struct {
	dir string
	mu  sync.Mutex
}

func (j *jsonTracer) RecordTurn(convID string, trace TurnTrace) {
	trace.ConversationID = convID
	data, err := json.Marshal(trace)
	if err != nil {
		return
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	path := filepath.Join(j.dir, convID+".jsonl")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = f.Write(append(data, '\n'))
}

// selectTracer picks a tracer by configuration: empty dir -> noop (default),
// non-empty -> jsonTracer writing to that directory. Wired in main.go from the
// TICKTASK_TRACE_DIR env var. (Phase 2 will add a langfuseTracer branch here.)
func selectTracer(dir string) TraceRecorder {
	if dir == "" {
		return noopTracer{}
	}
	return &jsonTracer{dir: dir}
}
```

- [ ] **Step 4: 运行测试，确认通过**

Run: `cd backend && go test -run 'TestSelectTracer|TestNoopTracer' -v ./internal/agent/`
Expected: PASS（3 个测试全绿）。

- [ ] **Step 5: 提交**

```bash
git add backend/internal/agent/trace.go backend/internal/agent/trace_test.go
git commit -m "feat(agent): add TraceRecorder with noop + json tracer"
```

---

## Task 2: `jsonTracer` 序列化测试

**Files:**
- Modify: `backend/internal/agent/trace_test.go`（追加测试）

- [ ] **Step 1: 写 `jsonTracer` 落盘的失败测试**

Append to `backend/internal/agent/trace_test.go`:

```go
func TestJsonTracer_AppendsOneLinePerTurn(t *testing.T) {
	dir := t.TempDir()
	jt := selectTracer(dir).(*jsonTracer)

	jt.RecordTurn("conv-1", TurnTrace{
		UserText:       "我一会有啥安排吗？",
		AssistantText:  "你今天有 3 个安排",
		Steps:          []TraceStep{{ToolCall: ai.ToolCall{ID: "c1", Name: "list_schedule", Args: json.RawMessage(`{"from":"2026-08-09","to":"2026-08-09"}`)}, Permission: PermRead, Status: "succeeded"}},
	})
	jt.RecordTurn("conv-1", TurnTrace{UserText: "第二问"})

	data, err := os.ReadFile(filepath.Join(dir, "conv-1.jsonl"))
	if err != nil {
		t.Fatalf("read trace file: %v", err)
	}
	lines := splitLines(string(data))
	if len(lines) != 2 {
		t.Fatalf("want 2 JSONL lines, got %d: %q", len(lines), string(data))
	}

	var first TurnTrace
	if err := json.Unmarshal([]byte(lines[0]), &first); err != nil {
		t.Fatalf("unmarshal line 1: %v", err)
	}
	if first.UserText != "我一会有啥安排吗？" || first.AssistantText != "你今天有 3 个安排" {
		t.Errorf("line 1 trace = %+v", first)
	}
	if len(first.Steps) != 1 || first.Steps[0].ToolCall.Name != "list_schedule" || first.Steps[0].Status != "succeeded" {
		t.Errorf("line 1 steps = %+v", first.Steps)
	}
	if first.ConversationID != "conv-1" {
		t.Errorf("ConversationID = %q, want conv-1", first.ConversationID)
	}
}

// separate conversations write separate files
func TestJsonTracer_PerConversationFile(t *testing.T) {
	dir := t.TempDir()
	jt := selectTracer(dir).(*jsonTracer)
	jt.RecordTurn("a", TurnTrace{UserText: "A"})
	jt.RecordTurn("b", TurnTrace{UserText: "B"})
	for _, id := range []string{"a", "b"} {
		if _, err := os.Stat(filepath.Join(dir, id+".jsonl")); err != nil {
			t.Errorf("missing %s.jsonl: %v", id, err)
		}
	}
}

func splitLines(s string) []string {
	var out []string
	cur := ""
	for _, r := range s {
		if r == '\n' {
			if cur != "" {
				out = append(out, cur)
			}
			cur = ""
			continue
		}
		cur += string(r)
	}
	if cur != "" {
		out = append(out, cur)
	}
	return out
}
```

Ensure the import block at the top of `trace_test.go` contains `encoding/json`, `os`, `path/filepath`, `testing`, and `ticktask/internal/ai` (add any that are missing).

- [ ] **Step 2: 运行测试，确认通过**

Run: `cd backend && go test -run TestJsonTracer -v ./internal/agent/`
Expected: PASS（jsonTracer 已在 Task 1 实现，这里验证其序列化行为）。

- [ ] **Step 3: 确认整个包仍编译通过**

Run: `cd backend && go build ./internal/agent/`
Expected: 无输出（成功）。

- [ ] **Step 4: 提交**

```bash
git add backend/internal/agent/trace_test.go
git commit -m "test(agent): cover jsonTracer JSONL serialization"
```

---

## Task 3: 把 `TraceRecorder` 接进 `runTurn` + L1 录制回归测试

**Files:**
- Create: `backend/internal/agent/service_trace_test.go`
- Modify: `backend/internal/agent/service.go`

- [ ] **Step 1: 写录制的失败测试（红）**

Create `backend/internal/agent/service_trace_test.go`:

```go
package agent

import (
	"context"
	"encoding/json"
	"testing"

	"ticktask/internal/ai"
)

// captureTracer stores the last recorded turn for assertion.
type captureTracer struct {
	got  TurnTrace
	seen bool
}

func (c *captureTracer) RecordTurn(_ string, t TurnTrace) { c.got = t; c.seen = true }

// Mirrors the setup of TestAgentService_PermReadToolAutoExecutes
// (service_test.go:198) but injects a capture tracer and asserts the trace.
func TestAgentService_RecordsTurnTrace(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&fakeTool{name: "list_tasks", perm: PermRead})
	llm := &mockLLM{responses: []ai.ToolResponse{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "list_tasks", Args: json.RawMessage(`{"status":"todo"}`)}}, FinishReason: "tool_calls"},
		{Content: "all done", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	repo := newInMemoryRepo(t)
	tr := &captureTracer{}
	svc := NewAgentService(AgentDeps{Repo: repo, LLMFactory: func() ai.LLMClient { return llm }, Registry: reg, Hub: hub, Tracer: tr})
	conv, _ := repo.CreateConversation()

	if err := svc.SendMessage(context.Background(), conv.ID, "go"); err != nil {
		t.Fatal(err)
	}

	if !tr.seen {
		t.Fatal("RecordTurn was never called")
	}
	if tr.got.UserText != "go" {
		t.Errorf("UserText = %q, want %q", tr.got.UserText, "go")
	}
	if tr.got.AssistantText != "all done" {
		t.Errorf("AssistantText = %q, want %q", tr.got.AssistantText, "all done")
	}
	if len(tr.got.Steps) != 1 {
		t.Fatalf("Steps len = %d, want 1: %+v", len(tr.got.Steps), tr.got.Steps)
	}
	s := tr.got.Steps[0]
	if s.ToolCall.Name != "list_tasks" {
		t.Errorf("step tool = %q, want list_tasks", s.ToolCall.Name)
	}
	if s.Permission != PermRead {
		t.Errorf("step permission = %v, want PermRead", s.Permission)
	}
	if s.Status != "succeeded" {
		t.Errorf("step status = %q, want succeeded", s.Status)
	}
}
```

- [ ] **Step 2: 运行测试，确认失败**

Run: `cd backend && go test -run TestAgentService_RecordsTurnTrace -v ./internal/agent/`
Expected: FAIL —— `AgentDeps` 无 `Tracer` 字段（编译错误），或 `RecordTurn never called`。

- [ ] **Step 3: 给 `AgentDeps` 加 `Tracer` 字段 + noop 默认**

In `backend/internal/agent/service.go`, find the `AgentDeps` struct (around line 23) and add a `Tracer` field as the last field:

```go
type AgentDeps struct {
	Repo         repository.AgentRepository
	LLMFactory   func() ai.LLMClient
	SettingsRepo repository.SettingRepository
	Registry     ToolRegistry
	Hub          HubBroadcaster
	System       string
	Tracer       TraceRecorder
}
```

Then in `NewAgentService` (around line 68), add the noop default so the field is never nil:

```go
func NewAgentService(d AgentDeps) AgentService {
	if d.System == "" {
		d.System = DefaultSystemPrompt
	}
	if d.Tracer == nil {
		d.Tracer = noopTracer{}
	}
	return &agentService{AgentDeps: d, pending: make(map[string]chan string)}
}
```

- [ ] **Step 4: 改 `runTurn` 签名线程化 `userText`，并更新唯一调用方**

Change the `runTurn` signature (line 85) and the call site in `SendMessage` (line 82). `runTurn` has exactly one caller (verified: only `SendMessage` at line 82).

In `SendMessage` (around line 75-83), change the call:

```go
func (s *agentService) SendMessage(ctx context.Context, convID, text string) error {
	if _, err := s.Repo.GetConversation(convID); err != nil {
		return err
	}
	if _, err := s.Repo.AppendMessage(convID, "user", text, nil, nil, nil, nil); err != nil {
		return err
	}
	return s.runTurn(ctx, convID, text, 0)
}
```

Change the `runTurn` signature to accept `userText`:

```go
func (s *agentService) runTurn(ctx context.Context, convID, userText string, toolCount int) error {
```

- [ ] **Step 5: 在 `runTurn` 顶部构建 trace + defer 落盘**

At the very top of `runTurn`'s body (right after the opening `{`, before `client := s.LLMFactory()`), insert:

```go
	trace := TurnTrace{ConversationID: convID, UserText: userText}
	defer func() { s.Tracer.RecordTurn(convID, trace) }()
```

> 关键：用闭包 `defer func(){...}()` 而非 `defer s.Tracer.RecordTurn(convID, trace)`——后者会在 defer 语句处就求值 `trace`（零值），拿不到循环里累积的最终值。

- [ ] **Step 6: 累积 `AssistantText`**

In the `if resp.Content != ""` block (around line 105-114), add accumulation right after the broadcast. Find:

```go
			s.broadcast(convID, websocket.EventAgentMessage, map[string]any{
				"conversation_id": convID, "message_id": msgID, "delta_text": resp.Content,
			})
```

Add immediately after that statement:

```go
			trace.AssistantText += resp.Content
```

- [ ] **Step 7: 加 `traceStep` 辅助函数**

Add this helper method anywhere in `service.go` (e.g. right after `runTurn`, before `Confirm`):

```go
// traceStep builds a TraceStep from the in-scope tool-call decision and its
// terminal outcome, keeping each append site in runTurn a one-liner.
func traceStep(tc ai.ToolCall, perm ToolPermission, status string, result any, errMsg string) TraceStep {
	return TraceStep{ToolCall: tc, Permission: perm, Status: status, Result: result, Error: errMsg}
}
```

- [ ] **Step 8: 在每个终态 append step**

In the `for _, tc := range resp.ToolCalls` loop, add ONE append at each terminal point. `tc` and `perm` are in scope (perm computed at the `perm := tool.Schema().Permission` line). For the two early branches (not-found, schema-fail) `perm` is not yet computed — pass `PermRead` (informational only).

**(a) tool not found** — find the block around line 122-125:

```go
			tool, err := s.Registry.Lookup(tc.Name)
			if err != nil {
				s.broadcastTool(convID, "", tc.Name, tc.Args, "failed", nil, nil, fmt.Sprintf("tool not found: %s", tc.Name))
				s.appendToolResult(convID, tc, "failed", `{"error":"not found"}`)
				continue
			}
```

Add before `continue`:

```go
				trace.Steps = append(trace.Steps, traceStep(tc, PermRead, "failed", nil, fmt.Sprintf("tool not found: %s", tc.Name)))
```

**(b) schema invalid** — find the block around line 132-136:

```go
			if err := ValidateArgs(tool.Schema().Function.Parameters, tc.Args); err != nil {
				s.broadcastTool(convID, "", tc.Name, tc.Args, "failed", nil, nil, err.Error())
				s.appendToolResult(convID, tc, "failed", fmt.Sprintf(`{"schema_error":%q}`, err.Error()))
				continue
			}
```

Add before `continue`:

```go
				trace.Steps = append(trace.Steps, traceStep(tc, PermRead, "failed", nil, err.Error()))
```

**(c) PermRead execute error** — find the block around line 141-143:

```go
				if err != nil {
					s.broadcastTool(convID, "", tc.Name, tc.Args, "failed", nil, nil, err.Error())
					s.appendToolResult(convID, tc, "failed", fmt.Sprintf(`{"error":%q}`, err.Error()))
```

Add after the `s.appendToolResult(...)` line (still inside `if err != nil`):

```go
					trace.Steps = append(trace.Steps, traceStep(tc, perm, "failed", nil, err.Error()))
```

**(d) PermRead execute ok** — find the block around line 144-148:

```go
				} else {
					rjson, _ := json.Marshal(result)
					s.broadcastTool(convID, "", tc.Name, tc.Args, "succeeded", result, nil, "")
					s.appendToolResult(convID, tc, "succeeded", string(rjson))
				}
```

Add before the closing `}` of the `else`:

```go
					trace.Steps = append(trace.Steps, traceStep(tc, perm, "succeeded", result, ""))
```

**(e) PermWrite/Dangerous rejected (decision != approve)** — find around line 159-164:

```go
					if decision != "approve" {
						s.broadcastTool(convID, msgID, tc.Name, tc.Args, "rejected", nil, nil, "")
						s.Repo.UpdateMessage(msgID, strPtr("rejected"), strPtr(`{"rejected":true}`))
						s.clearPending(msgID)
						continue
					}
```

Add before `continue`:

```go
						trace.Steps = append(trace.Steps, traceStep(tc, perm, "rejected", nil, ""))
```

**(f) PermWrite/Dangerous confirmation timeout** — find around line 165-169:

```go
				case <-time.After(ConfirmationTimeout):
					s.broadcastTool(convID, msgID, tc.Name, tc.Args, "rejected", nil, nil, "timeout")
					s.Repo.UpdateMessage(msgID, strPtr("rejected"), strPtr(`{"error":"timeout"}`))
					s.clearPending(msgID)
					continue
```

Add before `continue`:

```go
					trace.Steps = append(trace.Steps, traceStep(tc, perm, "rejected", nil, "timeout"))
```

**(g) PermWrite/Dangerous execute error** — find around line 175-177:

```go
				if err != nil {
					s.broadcastTool(convID, msgID, tc.Name, tc.Args, "failed", nil, nil, err.Error())
					s.Repo.UpdateMessage(msgID, strPtr("failed"), strPtr(fmt.Sprintf(`{"error":%q}`, err.Error())))
```

Add after the `s.Repo.UpdateMessage(...)` line (still inside `if err != nil`):

```go
					trace.Steps = append(trace.Steps, traceStep(tc, perm, "failed", nil, err.Error()))
```

**(h) PermWrite/Dangerous execute ok** — find around line 178-182:

```go
				} else {
					rjson, _ := json.Marshal(result)
					s.broadcastTool(convID, msgID, tc.Name, tc.Args, "succeeded", result, nil, "")
					s.Repo.UpdateMessage(msgID, strPtr("succeeded"), strPtr(string(rjson)))
				}
```

Add before the closing `}` of the `else`:

```go
					trace.Steps = append(trace.Steps, traceStep(tc, perm, "succeeded", result, ""))
```

- [ ] **Step 9: 运行录制测试，确认通过**

Run: `cd backend && go test -run TestAgentService_RecordsTurnTrace -v ./internal/agent/`
Expected: PASS。

- [ ] **Step 10: 确认现有 L1 测试零回归**

Run: `cd backend && go test ./internal/agent/...`
Expected: PASS（noop 默认使现有测试行为不变）。

- [ ] **Step 11: 提交**

```bash
git add backend/internal/agent/service.go backend/internal/agent/service_trace_test.go
git commit -m "feat(agent): record TurnTrace in runTurn via injected TraceRecorder"
```

---

## Task 4: `main.go` 按 env 选择 tracer

**Files:**
- Modify: `backend/cmd/server/main.go:140-145`

- [ ] **Step 1: 在 `AgentDeps` 构造处注入 tracer**

Find the block in `backend/cmd/server/main.go` around line 140-145:

```go
	agentSvc := agent.NewAgentService(agent.AgentDeps{
		Repo:         agentRepo,
		LLMFactory:   llmFactory,
		SettingsRepo: settingRepo,
		Registry:     registry,
		Hub:          wsHub,
	})
```

Add the `Tracer` field (last):

```go
	agentSvc := agent.NewAgentService(agent.AgentDeps{
		Repo:         agentRepo,
		LLMFactory:   llmFactory,
		SettingsRepo: settingRepo,
		Registry:     registry,
		Hub:          wsHub,
		Tracer:       agent.SelectTracerFromEnv(os.Getenv),
	})
```

- [ ] **Step 2: 暴露一个 env 驱动的构造入口（可测）**

In `backend/internal/agent/trace.go`, add after `selectTracer`:

```go
// SelectTracerFromEnv reads TICKTASK_TRACE_DIR via the given getter (os.Getenv
// in main, injectable for tests). Empty -> noop, non-empty -> jsonTracer.
func SelectTracerFromEnv(getenv func(string) string) TraceRecorder {
	return selectTracer(getenv("TICKTASK_TRACE_DIR"))
}
```

- [ ] **Step 3: 加可测的 env 测试**

Append to `backend/internal/agent/trace_test.go`:

```go
func TestSelectTracerFromEnv(t *testing.T) {
	if _, ok := SelectTracerFromEnv(func(string) string { return "" }).(noopTracer); !ok {
		t.Fatal("empty env should yield noopTracer")
	}
	if _, ok := SelectTracerFromEnv(func(string) string { return t.TempDir() }).(*jsonTracer); !ok {
		t.Fatal("set env should yield *jsonTracer")
	}
}
```

- [ ] **Step 4: 运行测试 + 编译整个后端**

Run: `cd backend && go test -run 'TestSelectTracerFromEnv' -v ./internal/agent/ && go build ./...`
Expected: 测试 PASS；`go build ./...` 无输出（成功）。

> 若 `main.go` 提示 `os` 未导入，确认 `import "os"` 已在 main.go 的 import 块中（该文件已大量使用 os，通常已导入）。

- [ ] **Step 5: 提交**

```bash
git add backend/internal/agent/trace.go backend/internal/agent/trace_test.go backend/cmd/server/main.go
git commit -m "feat(main): wire TraceRecorder from TICKTASK_TRACE_DIR env"
```

---

## Task 5: `eval/` 骨架 + 自定义 WS provider + 元测试

**Files:**
- Create: `eval/package.json`
- Create: `eval/provider.mjs`
- Create: `eval/provider.test.mjs`

- [ ] **Step 1: 建 `eval/package.json`**

Create `eval/package.json`:

```json
{
  "name": "ticktask-agent-eval",
  "private": true,
  "type": "module",
  "scripts": {
    "seed": "node seed.mjs",
    "eval": "promptfoo eval",
    "test": "node --test provider.test.mjs"
  },
  "devDependencies": {
    "promptfoo": "^0.103.0",
    "ws": "^8.18.0"
  }
}
```

- [ ] **Step 2: 写 provider 元测试（红）**

Create `eval/provider.test.mjs`:

```js
import { test } from 'node:test';
import assert from 'node:assert/strict';
import http from 'node:http';
import { WebSocketServer, WebSocket } from 'ws';
import { collect } from './provider.mjs';

// Mocks the agent REST + WS surface with a scripted event sequence and asserts
// the provider reconstructs {tool_calls, assistant_text} from WS events. This
// is the "verify the verifier" check — it does not call any real LLM.
test('collect reconstructs tool_calls and assistant_text from WS events', async () => {
  const toSend = [
    { type: 'agent_tool', conversation_id: 'conv-1', tool_name: 'list_schedule', args: { from: '2026-08-09', to: '2026-08-09' }, status: 'succeeded' },
    { type: 'agent_message', conversation_id: 'conv-1', delta_text: '你今天有 3 个安排' },
    { type: 'agent_done', conversation_id: 'conv-1', finish_reason: 'stop' },
  ];

  const httpServer = http.createServer((req, res) => {
    if (req.method === 'POST' && req.url === '/api/agent/conversations') {
      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify({ id: 'conv-1' }));
    } else if (req.method === 'POST' && req.url === '/api/agent/chat') {
      res.statusCode = 202;
      res.end();
      // give the WS client a moment to be registered before broadcasting
      setTimeout(() => {
        for (const c of wss.clients) {
          if (c.readyState === WebSocket.OPEN) toSend.forEach((e) => c.send(JSON.stringify(e)));
        }
      }, 50);
    } else {
      res.statusCode = 404;
      res.end();
    }
  });

  const wss = new WebSocketServer({ noServer: true });
  httpServer.on('upgrade', (req, socket, head) => {
    wss.handleUpgrade(req, socket, head, () => {});
  });

  await new Promise((r) => httpServer.listen(0, r));
  const port = httpServer.address().port;

  try {
    const out = await collect({
      baseUrl: `http://localhost:${port}`,
      wsUrl: `ws://localhost:${port}/ws`,
      prompt: '我一会有啥安排吗？',
      timeoutMs: 2000,
    });
    const r = JSON.parse(out.output);
    assert.equal(r.error, undefined, `unexpected error: ${r.error}`);
    assert.equal(r.tool_calls.length, 1);
    assert.equal(r.tool_calls[0].name, 'list_schedule');
    assert.equal(r.tool_calls[0].status, 'succeeded');
    assert.equal(r.assistant_text, '你今天有 3 个安排');
  } finally {
    wss.close();
    httpServer.close();
  }
});

test('collect surfaces a timeout error when agent_done never arrives', async () => {
  const httpServer = http.createServer((req, res) => {
    if (req.method === 'POST' && req.url === '/api/agent/conversations') {
      res.end(JSON.stringify({ id: 'conv-2' }));
    } else if (req.method === 'POST' && req.url === '/api/agent/chat') {
      res.statusCode = 202;
      res.end(); // never broadcast done
    } else {
      res.statusCode = 404;
      res.end();
    }
  });
  const wss = new WebSocketServer({ noServer: true });
  httpServer.on('upgrade', (req, socket, head) => wss.handleUpgrade(req, socket, head, () => {}));
  await new Promise((r) => httpServer.listen(0, r));
  const port = httpServer.address().port;
  try {
    const out = await collect({
      baseUrl: `http://localhost:${port}`,
      wsUrl: `ws://localhost:${port}/ws`,
      prompt: 'x',
      timeoutMs: 200,
    });
    const r = JSON.parse(out.output);
    assert.equal(r.error, 'timeout');
  } finally {
    wss.close();
    httpServer.close();
  }
});
```

- [ ] **Step 3: 运行元测试，确认失败**

Run: `cd eval && npm install && npm test`
Expected: FAIL —— `Cannot find module './provider.mjs'`。

- [ ] **Step 4: 写 `provider.mjs`**

Create `eval/provider.mjs`:

```js
import { WebSocket } from 'ws';

// Default env-driven entry point Promptfoo calls. Returns Promptfoo's
// { output: <json string> } shape; assertions parse output back to JSON.
export async function callApi(prompt) {
  const baseUrl = process.env.AGENT_BASE_URL || 'http://localhost:8080';
  const wsUrl = process.env.AGENT_WS_URL || 'ws://localhost:8080/ws';
  const result = await collect({ baseUrl, wsUrl, prompt, timeoutMs: 60_000 });
  return { output: JSON.stringify(result) };
}

// Core logic, exported for unit testing with mock servers. Creates a
// conversation, opens WS, posts the chat, collects agent_* events for that
// conversation until agent_done (or timeout), and returns the reconstruction.
export function collect({ baseUrl, wsUrl, prompt, timeoutMs }) {
  return (async () => {
    const conv = await fetch(`${baseUrl}/api/agent/conversations`, { method: 'POST' }).then((r) => r.json());
    const convId = conv.id;

    return new Promise((resolve) => {
      const toolCalls = [];
      let assistantText = '';
      let settled = false;

      const finish = (extra) => {
        if (settled) return;
        settled = true;
        clearTimeout(timer);
        try { ws.close(); } catch {}
        resolve({ tool_calls: toolCalls, assistant_text: assistantText, ...(extra || {}) });
      };

      const ws = new WebSocket(wsUrl);
      const timer = setTimeout(() => finish({ error: 'timeout' }), timeoutMs);

      ws.on('open', () => {
        fetch(`${baseUrl}/api/agent/chat`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ conversation_id: convId, text: prompt }),
        }).catch((e) => finish({ error: `chat POST failed: ${e.message}` }));
      });
      ws.on('message', (data) => {
        let m;
        try { m = JSON.parse(data.toString()); } catch { return; }
        if (m.conversation_id !== convId) return;
        if (m.type === 'agent_message') {
          assistantText += m.delta_text || '';
        } else if (m.type === 'agent_tool') {
          toolCalls.push({ name: m.tool_name, args: m.args, status: m.status });
        } else if (m.type === 'agent_done') {
          finish(m.error ? { error: m.error } : {});
        }
      });
      ws.on('error', (e) => finish({ error: `ws error: ${e.message}` }));
    });
  })();
}

export default { callApi };
```

- [ ] **Step 5: 运行元测试，确认通过**

Run: `cd eval && npm test`
Expected: PASS（2 个测试）。

- [ ] **Step 6: 提交**

```bash
git add eval/package.json eval/provider.mjs eval/provider.test.mjs
git commit -m "feat(eval): add custom WS provider + meta test"
```

---

## Task 6: `eval/seed.mjs` 幂等 seed

**Files:**
- Create: `eval/seed.mjs`

- [ ] **Step 1: 写 `seed.mjs`**

Create `eval/seed.mjs`:

```js
// Seeds today's schedules + tasks with distinctive titles so L2 assertions can
// reference stable keywords. Idempotent: clears prior seed data first.
// Run against a live backend:  AGENT_BASE_URL=http://localhost:8080 node seed.mjs
const BASE = process.env.AGENT_BASE_URL || 'http://localhost:8080';
const today = new Date().toISOString().slice(0, 10); // YYYY-MM-DD
const rfc = (hhmm) => `${today}T${hhmm}:00Z`; // time.RFC3339, UTC

const SCHEDULE_TITLES = ['审查 PR-1234', '和 Alice 1:1', '发布 v2.3'];
const TASK_TITLES = ['整理周报', '修复登录 bug'];

const SCHEDULES = [
  { title: '审查 PR-1234', start_time: rfc('09:00'), end_time: rfc('10:00'), type: 'task' },
  { title: '和 Alice 1:1', start_time: rfc('14:00'), end_time: rfc('14:30'), type: 'task' },
  { title: '发布 v2.3', start_time: rfc('17:00'), end_time: rfc('17:30'), type: 'task' },
];
const TASKS = [
  { title: '整理周报', quadrant: 2 },
  { title: '修复登录 bug', quadrant: 1 },
];

async function json(method, path, body) {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(`${method} ${path} -> ${res.status}: ${await res.text().catch(() => '')}`);
  return res.status === 204 ? null : res.json().catch(() => null);
}

async function clearSeed() {
  // Schedules: backend exposes DELETE /api/schedules (DeleteAll) — acceptable
  // for an eval DB we fully control.
  await json('DELETE', '/api/schedules').catch(() => {});
  // Tasks: delete only our seed titles to stay idempotent without nuking all.
  const tasks = await json('GET', '/api/tasks');
  for (const t of tasks || []) {
    if (TASK_TITLES.includes(t.title)) await json('DELETE', `/api/tasks/${t.id}`).catch(() => {});
  }
}

async function main() {
  await clearSeed();
  for (const s of SCHEDULES) {
    const ev = await json('POST', '/api/schedules', s);
    console.log('schedule:', ev && ev.title);
  }
  for (const tk of TASKS) {
    const task = await json('POST', '/api/tasks', tk);
    console.log('task:', task && task.title);
  }
  console.log(`seeded ${SCHEDULES.length} schedules + ${TASKS.length} tasks for ${today}`);
}

main().catch((e) => { console.error(e); process.exit(1); });
```

- [ ] **Step 2: 幂等性手测（需活后端）**

前置：`make dev`（后端 :8080）。然后：

Run: `cd eval && AGENT_BASE_URL=http://localhost:8080 node seed.mjs && AGENT_BASE_URL=http://localhost:8080 node seed.mjs`
Expected: 两次都打印 `seeded 3 schedules + 2 tasks`；第二次不应报「title 重复」错。

验证数据正确：Run: `curl -s http://localhost:8080/api/tasks | head -c 400`
Expected: 含且仅含 2 条 seed task（`整理周报`、`修复登录 bug`），而非 4 条（幂等生效）。

- [ ] **Step 3: 提交**

```bash
git add eval/seed.mjs
git commit -m "feat(eval): idempotent seed for today's schedules + tasks"
```

---

## Task 7: Promptfoo 配置 + 12 条 case + README + .gitignore + 集成跑通

**Files:**
- Create: `eval/promptfooconfig.yaml`
- Create: `eval/README.md`
- Modify: `.gitignore`

- [ ] **Step 1: 写 `.gitignore` 条目**

Append to `.gitignore` (repo root):

```
# Agent eval artifacts
backend/internal/agent/testdata/traces/*.jsonl
eval/node_modules/
eval/.promptfoo/
```

- [ ] **Step 2: 写 `promptfooconfig.yaml`**

Create `eval/promptfooconfig.yaml`:

```yaml
description: TickTask 对话式 Agent 端到端意图验证（L2）

# 自定义 WS provider：黑盒驱动 /api/agent/*，从 WS 重建 {tool_calls, assistant_text}
providers:
  - id: file://./provider.mjs
    config:
      baseUrl: '{{AGENT_BASE_URL|http://localhost:8080}}'

# 共用断言片段（YAML anchor，避免重复）
# toolCalled(<name>): 断言某工具被调用
# answerMentionsSeed: 断言答复命中 ≥1 个 seeded 关键词
# answerNotApology: 断言答复非道歉性

tests:
  - description: 我一会有啥安排吗 → list_schedule 覆盖今天
    vars: { prompt: '我一会有啥安排吗？' }
    assert:
      - type: javascript
        value: |
          const r = JSON.parse(output);
          const ls = (r.tool_calls||[]).find(t => t.name === 'list_schedule');
          if (!ls) return { pass: false, reason: 'list_schedule 未被调用' };
          let a = typeof ls.args === 'string' ? JSON.parse(ls.args) : (ls.args||{});
          const today = new Date().toISOString().slice(0,10);
          const from = (a.from||'').slice(0,10), to = (a.to||'').slice(0,10);
          return { pass: from <= today && today <= to, reason: `from=${from} to=${to} today=${today}` };
      - type: javascript
        value: |
          const r = JSON.parse(output);
          const kw = ['审查 PR-1234','和 Alice 1:1','发布 v2.3'];
          return { pass: kw.some(k => (r.assistant_text||'').includes(k)), reason: r.assistant_text };

  - description: 今天还剩什么安排 → list_schedule
    vars: { prompt: '今天还剩什么安排？' }
    assert:
      - type: javascript
        value: JSON.parse(output).tool_calls.some(t => t.name === 'list_schedule')

  - description: 有哪些没做的任务 → list_tasks
    vars: { prompt: '我有哪些没做的任务？' }
    assert:
      - type: javascript
        value: JSON.parse(output).tool_calls.some(t => t.name === 'list_tasks')

  - description: 下一个任务 → list_tasks
    vars: { prompt: '下一个任务是啥？' }
    assert:
      - type: javascript
        value: JSON.parse(output).tool_calls.some(t => t.name === 'list_tasks')

  - description: 今天专注了多久 → get_daily_insights
    vars: { prompt: '我今天专注了多久？' }
    assert:
      - type: javascript
        value: JSON.parse(output).tool_calls.some(t => t.name === 'get_daily_insights')

  - description: 番茄钟状态 → get_timer_status
    vars: { prompt: '番茄钟现在啥状态？' }
    assert:
      - type: javascript
        value: JSON.parse(output).tool_calls.some(t => t.name === 'get_timer_status')

  - description: 建任务 → create_task 触发确认
    vars: { prompt: '帮我建个任务：周报' }
    assert:
      - type: javascript
        value: |
          const r = JSON.parse(output);
          const c = (r.tool_calls||[]).find(t => t.name === 'create_task');
          return { pass: !!c && c.status === 'pending_confirmation', reason: JSON.stringify(c) };

  - description: 标完成 → update_task 触发确认
    vars: { prompt: '把「整理周报」标完成' }
    assert:
      - type: javascript
        value: |
          const r = JSON.parse(output);
          const c = (r.tool_calls||[]).find(t => t.name === 'update_task');
          return { pass: !!c && c.status === 'pending_confirmation', reason: JSON.stringify(c) };

  - description: 删任务 → delete_task 触发确认
    vars: { prompt: '删掉任务「修复登录 bug」' }
    assert:
      - type: javascript
        value: |
          const r = JSON.parse(output);
          const c = (r.tool_calls||[]).find(t => t.name === 'delete_task');
          return { pass: !!c && c.status === 'pending_confirmation', reason: JSON.stringify(c) };

  - description: 开始番茄钟 → start_pomodoro 触发确认
    vars: { prompt: '开始一个番茄钟' }
    assert:
      - type: javascript
        value: |
          const r = JSON.parse(output);
          const c = (r.tool_calls||[]).find(t => t.name === 'start_pomodoro');
          return { pass: !!c && c.status === 'pending_confirmation', reason: JSON.stringify(c) };

  - description: 今天有啥洞察 → get_daily_insights
    vars: { prompt: '我今天有啥洞察？' }
    assert:
      - type: javascript
        value: JSON.parse(output).tool_calls.some(t => t.name === 'get_daily_insights')

  - description: 记工作日志 → structure_worklog
    vars: { prompt: '帮我把这段工作记成日志：今天修了登录 bug 并发了 v2.3' }
    assert:
      - type: javascript
        value: JSON.parse(output).tool_calls.some(t => t.name === 'structure_worklog')

# 打分制：非二值闸门；建议阈值 0.9，手动/改 prompt 时跑
```

> 注：第 7 条（建任务）第一个 assert 里 `t_status` 引用了下方第二个 assert 才定义的函数——**以第二个 assert 为准**（它自包含 `t_status` 定义）。实现时把第 7 条的两个 assert 合并/对齐成自包含写法，删除第一个冗余断言。最终第 7 条应只有一条自包含 javascript 断言。

- [ ] **Step 3: 写 `eval/README.md`**

Create `eval/README.md`:

```markdown
# TickTask Agent 端到端意图验证（L2 / Promptfoo）

验证「自然语言 → 正确工具」的命中率。打分制，非 CI 闸门；改 prompt 或工具时手动跑。

## 前置

1. 后端运行且 AI 已配 key：`make dev`，在设置页配置并测试连接成功。
2. 安装依赖：`cd eval && npm install`

## 跑一次

```bash
# 1. seed 今天的强特征数据（幂等）
AGENT_BASE_URL=http://localhost:8080 node seed.mjs

# 2. 跑 eval（每条 case 都真打一次 LLM）
AGENT_BASE_URL=http://localhost:8080 npx promptfoo eval

# 3. 看报告
npx promptfoo view
```

## provider 契约

`provider.mjs` 黑盒驱动 `/api/agent/*`：建会话 → 开 WS → POST /chat → 收集该会话的
`agent_message`/`agent_tool`/`agent_done` → 返回 `{ tool_calls, assistant_text }`。
断言据此判断工具路由是否正确（如 `list_schedule` 覆盖今天）。

## 不进默认测试

L2 是打分制（建议通过率 ≥0.9），偶发 miss 不挂 CI。默认 `go test` / `make test` 不触发它。

## 相关

- spec：`docs/superpowers/specs/2026-08-09-agent-e2e-verification-design.md`
- 计划：`docs/superpowers/plans/2026-08-09-agent-e2e-verification-p1.md`
```

- [ ] **Step 4: 手测 provider 元测试仍绿**

Run: `cd eval && npm test`
Expected: PASS（确认 Task 5 的改动未被 config 破坏）。

- [ ] **Step 5: 集成跑通 canonical case（需活后端 + key + seed）**

前置：`make dev` + 设置页配 key + 测试连接成功。然后：

Run: `cd eval && AGENT_BASE_URL=http://localhost:8080 node seed.mjs && AGENT_BASE_URL=http://localhost:8080 npx promptfoo eval -j /tmp/pf.json`
Expected: 至少第 1 条（「我一会有啥安排吗？」）PASS——`list_schedule` 被调用、日期覆盖今天、答复含 seeded 关键词。其余 case 按 LLM 表现有通过/不通过，整体输出 pass-rate。

> 真 LLM 非确定性：单次跑个别 case miss 正常；关注 pass-rate 与 canonical case 是否稳定通过。

- [ ] **Step 6: 提交**

```bash
git add eval/promptfooconfig.yaml eval/README.md .gitignore
git commit -m "feat(eval): promptfoo config with 12 intent cases + README"
```

---

## 完成标准（P1）

- [ ] `cd backend && go test ./...` 全绿（含 Task 3 录制测试、Task 4 env 测试；noop 默认使现有测试零回归）
- [ ] `cd eval && npm test` 绿（provider 元测试，不需 LLM）
- [ ] `TICKTASK_TRACE_DIR=/tmp/traces make dev` + 一次 Agent 对话后，`/tmp/traces/<conv>.jsonl` 有非空 trace 行
- [ ] canonical case「我一会有啥安排吗？」在 `npx promptfoo eval` 中通过（list_schedule 覆盖今天 + 答复引用 seeded 数据）

## P1 完成后

P2（`langfuseTracer` + `docker-compose.langfuse.yml` + gated Playwright 纵向 spec）另起计划。
```
