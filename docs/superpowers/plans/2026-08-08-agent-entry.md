# Agent Entry Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace `AIService` with a function-calling conversational agent exposed via a global right-side drawer, supporting 13 MVP tools with a three-tier trust permission model (Read/Write/Dangerous).

**Architecture:** New `agent/` package on backend (service + registry + tools + repo) replaces existing `service/ai_service.go` and `service/work_log_ai_client.go`. New frontend drawer + 6 components + `agent` Pinia store; embedded AI buttons refactored to call agent in headless mode. WebSocket carries streaming tokens + tool_call progress events.

**Tech Stack:** Go 1.21 / Gin 1.10 / GORM 1.25 / SQLite | Vue 3.5 / Pinia 2.2 / Element Plus 2.8 / Vite 5.4 / TypeScript 5.6 strict

**Spec:** [`docs/superpowers/specs/2026-08-08-agent-entry-design.md`](../specs/2026-08-08-agent-entry-design.md)

## Global Constraints

- Go module path: `ticktask`; package names: short lowercase singular (`agent`, `model`, `repository`)
- File naming: snake_case with domain prefix (`agent_repo.go`, `tool_card.vue`)
- Go interfaces: exported PascalCase noun; implementations: unexported lowercase struct
- Constructors: `New*` prefix returning the interface type
- Repository constructors return interface types, not concrete structs
- Backend test mocks shared in `backend/internal/api/handler/mocks_test.go`
- Frontend types: single barrel `src/types/index.ts`; TS strict + `noUnusedLocals` + `noUnusedParameters`
- Pinia test isolation: `setActivePinia(createPinia())` in each `beforeEach`
- Conventional Commits: `feat:`/`fix:`/`docs:`/`chore:`/`refactor:`/`test:`
- Branch strategy: `evolve/*` for feature work
- Don't skip pre-commit hooks; no `--no-verify`

## File Structure Overview

### Backend — new files

| Path | Responsibility |
|---|---|
| `backend/internal/model/agent.go` | `AgentConversation`, `AgentMessage` GORM models |
| `backend/internal/repository/agent_repo.go` | `AgentRepository` interface + GORM impl |
| `backend/internal/agent/tool.go` | `Tool`, `ToolSchema`, `ToolPermission` constants |
| `backend/internal/agent/registry.go` | `ToolRegistry` |
| `backend/internal/agent/service.go` | `AgentService` (orchestration loop) |
| `backend/internal/agent/conversation.go` | Build LLM message sequence from DB history |
| `backend/internal/agent/limits.go` | Boundary constants |
| `backend/internal/agent/prompts.go` | System prompt |
| `backend/internal/agent/tools/register.go` | `RegisterAll(reg, deps)` |
| `backend/internal/agent/tools/task.go` | 5 task tools |
| `backend/internal/agent/tools/timer.go` | 3 timer tools |
| `backend/internal/agent/tools/schedule.go` | 2 schedule tools |
| `backend/internal/agent/tools/worklog.go` | 2 worklog tools |
| `backend/internal/agent/tools/insight.go` | 1 insight tool |
| `backend/internal/api/handler/agent_handler.go` | 7 HTTP endpoints + tests |

### Backend — modified

| Path | Change |
|---|---|
| `backend/internal/ai/client.go` | Add `ChatWithTools` to `LLMClient` interface + 3 impls |
| `backend/internal/websocket/hub.go` | Add `EventAgent*` constants |
| `backend/internal/api/router.go` | Register `/api/agent/*` |
| `backend/cmd/server/main.go` | Wire `AgentService` + `ToolRegistry`; remove AIService wiring |
| `backend/pkg/database/database.go` | Add agent models to `AutoMigrate` |

### Backend — deleted (Task 15)

`internal/service/ai_service.go`, `internal/service/work_log_ai_client.go`, `internal/api/handler/ai_handler.go`, `internal/ai/prompts.go`, `internal/ai/work_log_prompts.go`

### Frontend — new

| Path | Responsibility |
|---|---|
| `frontend/src/components/agent/AgentDrawer.vue` | Drawer container |
| `frontend/src/components/agent/AgentMessageList.vue` | Message rendering |
| `frontend/src/components/agent/AgentInput.vue` | Input box |
| `frontend/src/components/agent/ToolCard.vue` | Tool call card |
| `frontend/src/components/agent/ToolConfirmDialog.vue` | Dangerous confirm modal |
| `frontend/src/components/agent/ConversationList.vue` | History list |
| `frontend/src/stores/agent.ts` | Pinia store |

### Frontend — modified

| Path | Change |
|---|---|
| `frontend/src/types/index.ts` | Add Agent types |
| `frontend/src/api/client.ts` | Add `agent` API group |
| `frontend/src/utils/websocket.ts` | Dispatch `agent_*` events |
| `frontend/src/App.vue` | Mount `<AgentDrawer/>` + header icon |
| `frontend/src/views/Settings.vue` | Use `agent.checkStatus()` |
| `frontend/src/views/Dashboard.vue` | Use `agent.runTool('get_daily_insights', ...)` |
| `frontend/src/views/Analytics.vue` | Same |
| `frontend/src/components/tasks/TaskForm.vue` | Use `agent.runTool('classify_task', ...)` |
| `frontend/src/components/tasks/TaskCard.vue` | Same |

### Frontend — deleted (Task 21)

`frontend/src/stores/ai.ts`

---

## Milestone Overview

| Milestone | Tasks | Description |
|---|---|---|
| M1 | 1-5 | Backend core abstractions |
| M2 | 6-9 | AgentService orchestration complete |
| M3 | 10-12 | Tool implementations |
| M4 | 13-15 | HTTP API + wiring + AIService removal |
| M5 | 16-17 | Frontend foundation |
| M6 | 18-20 | Frontend components |
| M7 | 21-22 | Embedded button refactor + finish |

---

## M1 — Backend Core Abstractions

### Task 1: Agent Data Model + Repository

**Files:**
- Create: `backend/internal/model/agent.go`
- Create: `backend/internal/repository/agent_repo.go`
- Create: `backend/internal/repository/agent_repo_test.go`
- Modify: `backend/pkg/database/database.go`

**Interfaces:**
- Consumes: existing `*gorm.DB`, `repository.ErrNotFound` pattern
- Produces: `repository.AgentRepository` interface, `model.AgentConversation`, `model.AgentMessage`

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/repository/agent_repo_test.go
package repository

import (
	"testing"
	"time"

	"ticktask/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.AgentConversation{}, &model.AgentMessage{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestAgentRepo_CreateConversation(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentRepository(db)
	conv, err := repo.CreateConversation()
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if conv.ID == "" {
		t.Fatal("id empty")
	}
	if conv.Title != "New Conversation" {
		t.Fatalf("default title: %q", conv.Title)
	}
}

func TestAgentRepo_AppendMessage_TitleFromFirstUser(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentRepository(db)
	conv, _ := repo.CreateConversation()
	longText := "今天有哪些没做完的任务需要顺延到明天并写日报总结"
	_, err := repo.AppendMessage(conv.ID, "user", longText, nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("append: %v", err)
	}
	got, _ := repo.GetConversation(conv.ID)
	want := longText[:30]
	if got.Title != want {
		t.Fatalf("title = %q, want %q", got.Title, want)
	}
}

func TestAgentRepo_LoadRecentMessages(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentRepository(db)
	conv, _ := repo.CreateConversation()
	for i := 0; i < 25; i++ {
		repo.AppendMessage(conv.ID, "user", "msg", nil, nil, nil, nil)
	}
	msgs, err := repo.LoadRecentMessages(conv.ID, 20)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(msgs) != 20 {
		t.Fatalf("got %d, want 20", len(msgs))
	}
	if !msgs[0].CreatedAt.Before(msgs[19].CreatedAt) {
		t.Fatal("not ascending")
	}
}

func TestAgentRepo_DeleteConversation_Cascade(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentRepository(db)
	conv, _ := repo.CreateConversation()
	repo.AppendMessage(conv.ID, "user", "hi", nil, nil, nil, nil)
	if err := repo.DeleteConversation(conv.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err := repo.GetConversation(conv.ID)
	if err != ErrNotFound {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
}

func TestAgentRepo_UpdateMessageStatus(t *testing.T) {
	db := setupTestDB(t)
	repo := NewAgentRepository(db)
	conv, _ := repo.CreateConversation()
	msgID, _ := repo.AppendMessage(conv.ID, "tool_call", "", strPtr("list_tasks"), nil, nil, nil)
	status := "succeeded"
	result := `{"tasks":[]}`
	if err := repo.UpdateMessage(msgID, &status, &result); err != nil {
		t.Fatalf("update: %v", err)
	}
}

func strPtr(s string) *string { return &s }
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/repository/ -run TestAgentRepo -v
```
Expected: FAIL with `undefined: model.AgentConversation` etc.

- [ ] **Step 3: Write model + repository implementation**

```go
// backend/internal/model/agent.go
package model

import "time"

type AgentConversation struct {
	ID           string    `gorm:"primaryKey;type:text" json:"id"`
	Title        string    `gorm:"size:200" json:"title"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	MessageCount int       `json:"message_count"`
}

type AgentMessage struct {
	ID             string    `gorm:"primaryKey;type:text" json:"id"`
	ConversationID string    `gorm:"index;type:text" json:"conversation_id"`
	Role           string    `gorm:"size:20" json:"role"`
	Content        string    `gorm:"type:text" json:"content"`
	ToolName       *string   `gorm:"size:50" json:"tool_name,omitempty"`
	ToolArgs       *string   `gorm:"type:text" json:"tool_args,omitempty"`
	ToolResult     *string   `gorm:"type:text" json:"tool_result,omitempty"`
	ToolStatus     *string   `gorm:"size:30" json:"tool_status,omitempty"`
	ParentID       *string   `gorm:"type:text" json:"parent_id,omitempty"`
	CreatedAt      time.Time `gorm:"index" json:"created_at"`
}
```

```go
// backend/internal/repository/agent_repo.go
package repository

import (
	"crypto/rand"
	"time"

	"ticktask/internal/model"

	"gorm.io/gorm"
)

type AgentRepository interface {
	CreateConversation() (*model.AgentConversation, error)
	GetConversation(id string) (*model.AgentConversation, error)
	ListConversations(page, size int) ([]*model.AgentConversation, int, error)
	DeleteConversation(id string) error
	AppendMessage(convID, role, content string, toolName, toolArgs, toolResult, toolStatus *string) (string, error)
	LoadRecentMessages(convID string, limit int) ([]*model.AgentMessage, error)
	ListMessages(convID string) ([]*model.AgentMessage, error)
	UpdateMessage(id string, status, result *string) error
	GetMessage(id string) (*model.AgentMessage, error)
}

type agentRepo struct{ db *gorm.DB }

func NewAgentRepository(db *gorm.DB) AgentRepository {
	return &agentRepo{db: db}
}

func newUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func (r *agentRepo) CreateConversation() (*model.AgentConversation, error) {
	conv := &model.AgentConversation{
		ID:        newUUID(),
		Title:     "New Conversation",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	if err := r.db.Create(conv).Error; err != nil {
		return nil, err
	}
	return conv, nil
}

func (r *agentRepo) GetConversation(id string) (*model.AgentConversation, error) {
	var c model.AgentConversation
	if err := r.db.First(&c, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &c, nil
}

func (r *agentRepo) ListConversations(page, size int) ([]*model.AgentConversation, int, error) {
	var items []*model.AgentConversation
	var total int64
	r.db.Model(&model.AgentConversation{}).Count(&total)
	off := (page - 1) * size
	if err := r.db.Order("updated_at DESC").Offset(off).Limit(size).Find(&items).Error; err != nil {
		return nil, 0, err
	}
	return items, int(total), nil
}

func (r *agentRepo) DeleteConversation(id string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		tx.Where("conversation_id = ?", id).Delete(&model.AgentMessage{})
		return tx.Where("id = ?", id).Delete(&model.AgentConversation{}).Error
	})
}

func (r *agentRepo) AppendMessage(convID, role, content string, toolName, toolArgs, toolResult, toolStatus *string) (string, error) {
	msg := &model.AgentMessage{
		ID:             newUUID(),
		ConversationID: convID,
		Role:           role,
		Content:        content,
		ToolName:       toolName,
		ToolArgs:       toolArgs,
		ToolResult:     toolResult,
		ToolStatus:     toolStatus,
		CreatedAt:      time.Now(),
	}
	if err := r.db.Create(msg).Error; err != nil {
		return "", err
	}
	r.db.Model(&model.AgentConversation{}).Where("id = ?", convID).
		UpdateColumns(map[string]any{"updated_at": time.Now(), "message_count": gorm.Expr("message_count + 1")})
	if role == "user" && len(content) > 0 {
		title := content
		if len(title) > 30 {
			title = title[:30]
		}
		r.db.Model(&model.AgentConversation{}).Where("id = ?", convID).Update("title", title)
	}
	return msg.ID, nil
}

func (r *agentRepo) LoadRecentMessages(convID string, limit int) ([]*model.AgentMessage, error) {
	var msgs []*model.AgentMessage
	sub := r.db.Where("conversation_id = ?", convID).
		Order("created_at DESC").Limit(limit)
	if err := r.db.Raw("SELECT * FROM (?) AS u ORDER BY u.created_at ASC", sub).Scan(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}

func (r *agentRepo) ListMessages(convID string) ([]*model.AgentMessage, error) {
	var msgs []*model.AgentMessage
	if err := r.db.Where("conversation_id = ?", convID).Order("created_at ASC").Find(&msgs).Error; err != nil {
		return nil, err
	}
	return msgs, nil
}

func (r *agentRepo) UpdateMessage(id string, status, result *string) error {
	updates := map[string]any{}
	if status != nil {
		updates["tool_status"] = *status
	}
	if result != nil {
		updates["tool_result"] = *result
	}
	if len(updates) == 0 {
		return nil
	}
	return r.db.Model(&model.AgentMessage{}).Where("id = ?", id).Updates(updates).Error
}

func (r *agentRepo) GetMessage(id string) (*model.AgentMessage, error) {
	var m model.AgentMessage
	if err := r.db.First(&m, "id = ?", id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &m, nil
}
```

Add `"fmt"` and `"crypto/rand"` imports to `agent_repo.go`. Verify `ErrNotFound` exists in `backend/internal/repository/errors.go` (it does).

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/repository/ -run TestAgentRepo -v
```
Expected: PASS

- [ ] **Step 5: Add to AutoMigrate + commit**

```go
// backend/pkg/database/database.go — in the existing AutoMigrate call, add:
//   &model.AgentConversation{}, &model.AgentMessage{},
```

```bash
git add backend/internal/model/agent.go backend/internal/repository/agent_repo.go backend/internal/repository/agent_repo_test.go backend/pkg/database/database.go
git commit -m "feat(agent): add agent conversation/message model and repository"
```

---

### Task 2: Tool Interface + Permission Constants + ToolRegistry

**Files:**
- Create: `backend/internal/agent/tool.go`
- Create: `backend/internal/agent/registry.go`
- Create: `backend/internal/agent/registry_test.go`
- Create: `backend/internal/agent/limits.go`

**Interfaces:**
- Consumes: nothing
- Produces: `agent.Tool` interface, `agent.ToolSchema`, `agent.ToolPermission` (`PermRead`/`PermWrite`/`PermDangerous`), `agent.ToolRegistry` interface, `agent.NewToolRegistry()`, `agent.ErrToolNotFound`, `agent.ErrDuplicateTool`

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/agent/registry_test.go
package agent

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
)

type fakeTool struct {
	name string
	perm ToolPermission
}

func (t *fakeTool) Schema() ToolSchema {
	return ToolSchema{
		Name: t.name,
		Function: FunctionSpec{
			Description: "fake",
			Parameters:  map[string]any{"type": "object"},
		},
		Permission: t.perm,
	}
}
func (t *fakeTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	return "ok:" + t.name, nil
}
func (t *fakeTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return "preview:" + t.name, nil
}

func TestRegistry_RegisterAndLookup(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&fakeTool{name: "t1", perm: PermRead})
	got, err := r.Lookup("t1")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	res, _ := got.Execute(context.Background(), nil)
	if res != "ok:t1" {
		t.Fatalf("res = %v", res)
	}
}

func TestRegistry_LookupUnknown(t *testing.T) {
	r := NewToolRegistry()
	_, err := r.Lookup("xxx")
	if !errors.Is(err, ErrToolNotFound) {
		t.Fatalf("err = %v, want ErrToolNotFound", err)
	}
}

func TestRegistry_DuplicateRegister(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&fakeTool{name: "t1", perm: PermRead})
	err := r.Register(&fakeTool{name: "t1", perm: PermRead})
	if !errors.Is(err, ErrDuplicateTool) {
		t.Fatalf("err = %v, want ErrDuplicateTool", err)
	}
}

func TestRegistry_ToOpenAITools(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&fakeTool{name: "t1", perm: PermRead})
	specs := r.ToOpenAITools()
	if len(specs) != 1 {
		t.Fatalf("got %d specs", len(specs))
	}
	if specs[0].Type != "function" || specs[0].Function.Name != "t1" {
		t.Fatalf("spec malformed: %+v", specs[0])
	}
}

func TestRegistry_ListByPermission(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&fakeTool{name: "r1", perm: PermRead})
	r.Register(&fakeTool{name: "w1", perm: PermWrite})
	r.Register(&fakeTool{name: "d1", perm: PermDangerous})
	if len(r.ListByPermission(PermRead)) != 1 {
		t.Fatal("read count")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/agent/ -v
```
Expected: FAIL (package not built)

- [ ] **Step 3: Write implementation**

```go
// backend/internal/agent/tool.go
package agent

import (
	"context"
	"encoding/json"
)

type ToolPermission int

const (
	PermRead ToolPermission = iota
	PermWrite
	PermDangerous
)

type FunctionSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolSchema struct {
	Name       string
	Function   FunctionSpec
	Permission ToolPermission
}

type Tool interface {
	Schema() ToolSchema
	Execute(ctx context.Context, args json.RawMessage) (any, error)
	Preview(ctx context.Context, args json.RawMessage) (any, error)
}
```

```go
// backend/internal/agent/registry.go
package agent

import (
	"context"
	"encoding/json"
	"errors"
)

var (
	ErrToolNotFound   = errors.New("tool not found")
	ErrDuplicateTool  = errors.New("duplicate tool name")
)

type ToolRegistry interface {
	Register(t Tool) error
	Lookup(name string) (Tool, error)
	ToOpenAITools() []OpenAIToolSpec
	ListByPermission(p ToolPermission) []Tool
}

type OpenAIToolSpec struct {
	Type     string         `json:"type"`
	Function FunctionSpec   `json:"function"`
}

type toolRegistry struct{ tools map[string]Tool }

func NewToolRegistry() ToolRegistry {
	return &toolRegistry{tools: make(map[string]Tool)}
}

func (r *toolRegistry) Register(t Tool) error {
	schema := t.Schema()
	if _, exists := r.tools[schema.Name]; exists {
		return ErrDuplicateTool
	}
	r.tools[schema.Name] = t
	return nil
}

// RegisterAll accepts the method above too for backward-compat in tests
func (r *toolRegistry) RegisterAll(tools []Tool) error {
	for _, t := range tools {
		if err := r.Register(t); err != nil {
			return err
		}
	}
	return nil
}

func (r *toolRegistry) Lookup(name string) (Tool, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, ErrToolNotFound
	}
	return t, nil
}

func (r *toolRegistry) ToOpenAITools() []OpenAIToolSpec {
	specs := make([]OpenAIToolSpec, 0, len(r.tools))
	for _, t := range r.tools {
		s := t.Schema()
		specs = append(specs, OpenAIToolSpec{Type: "function", Function: s.Function})
	}
	return specs
}

func (r *toolRegistry) ListByPermission(p ToolPermission) []Tool {
	out := []Tool{}
	for _, t := range r.tools {
		if t.Schema().Permission == p {
			out = append(out, t)
		}
	}
	return out
}

// no-op to keep imports
var _ = json.RawMessage{}
var _ = context.Background{}
```

> NOTE: Tests call `r.Register(&fakeTool{...})` without checking error. Update tests to ignore error or update `Register` to not return error. **Fix: change tests to `r.Register(...)` returning error and tests should use `_ = r.Register(...)`.** Actually simpler — keep tests as-is and make `Register` not return error. Let me restructure: see corrected version below.

```go
// backend/internal/agent/registry.go (CORRECTED — Register returns no error for ergonomic chaining in tests; use MustRegister or check separately)
package agent

import (
	"errors"
)

var (
	ErrToolNotFound  = errors.New("tool not found")
	ErrDuplicateTool = errors.New("duplicate tool name")
)

type ToolRegistry interface {
	Register(t Tool)
	MustRegister(t Tool) // panics on duplicate (used at startup)
	Lookup(name string) (Tool, error)
	ToOpenAITools() []OpenAIToolSpec
	ListByPermission(p ToolPermission) []Tool
}

type OpenAIToolSpec struct {
	Type     string       `json:"type"`
	Function FunctionSpec `json:"function"`
}

type toolRegistry struct{ tools map[string]Tool }

func NewToolRegistry() ToolRegistry {
	return &toolRegistry{tools: make(map[string]Tool)}
}

func (r *toolRegistry) Register(t Tool) {
	r.tools[t.Schema().Name] = t
}

func (r *toolRegistry) MustRegister(t Tool) {
	name := t.Schema().Name
	if _, exists := r.tools[name]; exists {
		panic(ErrDuplicateTool)
	}
	r.tools[name] = t
}

func (r *toolRegistry) Lookup(name string) (Tool, error) {
	t, ok := r.tools[name]
	if !ok {
		return nil, ErrToolNotFound
	}
	return t, nil
}

func (r *toolRegistry) ToOpenAITools() []OpenAIToolSpec {
	specs := make([]OpenAIToolSpec, 0, len(r.tools))
	for _, t := range r.tools {
		s := t.Schema()
		specs = append(specs, OpenAIToolSpec{Type: "function", Function: s.Function})
	}
	return specs
}

func (r *toolRegistry) ListByPermission(p ToolPermission) []Tool {
	out := []Tool{}
	for _, t := range r.tools {
		if t.Schema().Permission == p {
			out = append(out, t)
		}
	}
	return out
}
```

Update `TestRegistry_DuplicateRegister` to use `MustRegister`:

```go
func TestRegistry_DuplicateRegister(t *testing.T) {
	r := NewToolRegistry()
	r.Register(&fakeTool{name: "t1", perm: PermRead})
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic on duplicate MustRegister")
		}
	}()
	r.MustRegister(&fakeTool{name: "t1", perm: PermRead})
}
```

```go
// backend/internal/agent/limits.go
package agent

import "time"

const (
	MaxToolCallsPerTurn = 20
	MaxContextMessages  = 20
	MaxMessageTokens    = 4000
	ConfirmationTimeout = 30 * time.Minute
)
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/agent/ -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/agent/
git commit -m "feat(agent): add tool interface, registry, permission constants"
```

---

### Task 3: LLMClient ChatWithTools Extension (OpenAI impl)

**Files:**
- Modify: `backend/internal/ai/client.go`
- Create: `backend/internal/ai/client_tools_test.go`

**Interfaces:**
- Consumes: existing `LLMClient`, `model.AISettings`
- Produces: `ai.Message`, `ai.ToolCall`, `ai.ToolSpec`, `ai.ToolResponse`, `LLMClient.ChatWithTools(ctx, messages, tools) (ToolResponse, error)`, `ErrFunctionCallNotSupported`

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/ai/client_tools_test.go
package ai

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOpenAIClient_ChatWithTools_BuildsRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["tools"]; !ok {
			t.Errorf("request missing tools field")
		}
		resp := `{"choices":[{"message":{"role":"assistant","content":"hi","tool_calls":null},"finish_reason":"stop"}]}`
		w.Write([]byte(resp))
	}))
	defer srv.Close()
	c := NewOpenAIClient(srv.URL, "k", "gpt-4o-mini")
	res, err := c.ChatWithTools(context.Background(),
		[]Message{{Role: "user", Content: "hi"}},
		[]ToolSpec{{Type: "function", Function: FunctionSpec{Name: "f"}}})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Content != "hi" {
		t.Fatalf("content = %q", res.Content)
	}
	if res.FinishReason != "stop" {
		t.Fatalf("finish = %q", res.FinishReason)
	}
}

func TestOpenAIClient_ChatWithTools_ParsesToolCalls(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := `{"choices":[{"message":{"role":"assistant","tool_calls":[{"id":"c1","type":"function","function":{"name":"list_tasks","arguments":"{\"status\":\"todo\"}"}}]},"finish_reason":"tool_calls"}]}`
		w.Write([]byte(resp))
	}))
	defer srv.Close()
	c := NewOpenAIClient(srv.URL, "k", "m")
	res, _ := c.ChatWithTools(context.Background(),
		[]Message{{Role: "user", Content: "go"}}, nil)
	if len(res.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d", len(res.ToolCalls))
	}
	tc := res.ToolCalls[0]
	if tc.ID != "c1" || tc.Name != "list_tasks" {
		t.Fatalf("tc = %+v", tc)
	}
	var args map[string]string
	json.Unmarshal(tc.Args, &args)
	if args["status"] != "todo" {
		t.Fatalf("args = %+v", args)
	}
}

func TestCLIClient_ChatWithTools_Unsupported(t *testing.T) {
	c := CLIClient{Binary: "echo"}
	_, err := c.ChatWithTools(context.Background(), nil, nil)
	if !errors.Is(err, ErrFunctionCallNotSupported) {
		t.Fatalf("err = %v, want ErrFunctionCallNotSupported", err)
	}
}

func TestOpenAIClient_NetworkError_Retries(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()
	c := NewOpenAIClient(srv.URL, "k", "m")
	_, err := c.ChatWithTools(context.Background(),
		[]Message{{Role: "user", Content: "x"}}, nil)
	if err == nil || !strings.Contains(err.Error(), "503") {
		t.Fatalf("err = %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/ai/ -run ChatWithTools -v
```
Expected: FAIL (undefined symbols)

- [ ] **Step 3: Write implementation**

```go
// Add to backend/internal/ai/client.go (extend existing file)

package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// === Function-calling extension ===

var ErrFunctionCallNotSupported = errors.New("function calling not supported by this provider")

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
	Name       string     `json:"name,omitempty"`
}

type ToolCall struct {
	ID   string          `json:"id"`
	Name string          `json:"name"`
	Args json.RawMessage `json:"args"`
}

type FunctionSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
}

type ToolSpec struct {
	Type     string       `json:"type"`
	Function FunctionSpec `json:"function"`
}

type ToolResponse struct {
	Content      string     `json:"content"`
	ToolCalls    []ToolCall `json:"tool_calls"`
	FinishReason string     `json:"finish_reason"`
}

// ChatWithTools is added to the LLMClient interface signature in this same file.
// Each implementation gets its own method.
```

Modify the `LLMClient` interface (currently in client.go):

```go
type LLMClient interface {
	ChatCompletion(ctx context.Context, prompt string) (string, error)
	ChatWithTools(ctx context.Context, messages []Message, tools []ToolSpec) (ToolResponse, error)
}
```

Add `ChatWithTools` to `OpenAIClient`:

```go
func (c *OpenAIClient) ChatWithTools(ctx context.Context, messages []Message, tools []ToolSpec) (ToolResponse, error) {
	body := map[string]any{
		"model":    c.Model,
		"messages": messages,
	}
	if len(tools) > 0 {
		body["tools"] = tools
		body["tool_choice"] = "auto"
	}
	jsonBody, _ := json.Marshal(body)

	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		req, _ := http.NewRequestWithContext(ctx, "POST", c.BaseURL+"/chat/completions", bytes.NewReader(jsonBody))
		req.Header.Set("Authorization", "Bearer "+c.APIKey)
		req.Header.Set("Content-Type", "application/json")

		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}
		defer resp.Body.Close()

		raw, _ := io.ReadAll(resp.Body)
		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("openai %d: %s", resp.StatusCode, string(raw))
			time.Sleep(time.Duration(1<<attempt) * time.Second)
			continue
		}
		if resp.StatusCode != 200 {
			return ToolResponse{}, fmt.Errorf("openai %d: %s", resp.StatusCode, string(raw))
		}

		var parsed struct {
			Choices []struct {
				Message struct {
					Role      string `json:"role"`
					Content   string `json:"content"`
					ToolCalls []struct {
						ID       string `json:"id"`
						Type     string `json:"type"`
						Function struct {
							Name      string `json:"name"`
							Arguments string `json:"arguments"`
						} `json:"function"`
					} `json:"tool_calls"`
				} `json:"message"`
				FinishReason string `json:"finish_reason"`
			} `json:"choices"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return ToolResponse{}, fmt.Errorf("decode: %w", err)
		}
		if len(parsed.Choices) == 0 {
			return ToolResponse{}, errors.New("no choices")
		}
		ch := parsed.Choices[0]
		out := ToolResponse{
			Content:      ch.Message.Content,
			FinishReason: ch.FinishReason,
		}
		for _, tc := range ch.Message.ToolCalls {
			out.ToolCalls = append(out.ToolCalls, ToolCall{
				ID:   tc.ID,
				Name: tc.Function.Name,
				Args: json.RawMessage(tc.Function.Arguments),
			})
		}
		return out, nil
	}
	return ToolResponse{}, lastErr
}
```

Add to `AnthropicClient` (stub — full impl in Task 22):

```go
func (c *AnthropicClient) ChatWithTools(ctx context.Context, messages []Message, tools []ToolSpec) (ToolResponse, error) {
	return ToolResponse{}, ErrFunctionCallNotSupported
}
```

Add to `CLIClient`:

```go
func (c CLIClient) ChatWithTools(ctx context.Context, messages []Message, tools []ToolSpec) (ToolResponse, error) {
	return ToolResponse{}, ErrFunctionCallNotSupported
}
```

If existing `OpenAIClient` struct lacks an `HTTP *http.Client` field, add it with default `&http.Client{Timeout: 60 * time.Second}` in `NewOpenAIClient`. Verify the constructor matches the test calls `NewOpenAIClient(srv.URL, "k", "gpt-4o-mini")`.

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/ai/ -run ChatWithTools -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/ai/client.go backend/internal/ai/client_tools_test.go
git commit -m "feat(ai): add ChatWithTools to LLMClient with OpenAI implementation"
```

---

### Task 4: WebSocket Hub Agent Event Constants

**Files:**
- Modify: `backend/internal/websocket/hub.go`

**Interfaces:**
- Consumes: existing `Hub.Broadcast` method
- Produces: constants `EventAgentMessage`, `EventAgentTool`, `EventAgentDone`

- [ ] **Step 1: Write the failing test**

```go
// Add to backend/internal/websocket/hub_test.go (or new file hub_agent_events_test.go)
package websocket

import "testing"

func TestAgentEventConstants(t *testing.T) {
	cases := map[string]string{
		"agent_message": EventAgentMessage,
		"agent_tool":    EventAgentTool,
		"agent_done":    EventAgentDone,
	}
	for want, got := range cases {
		if got != want {
			t.Errorf("constant = %q, want %q", got, want)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/websocket/ -v
```
Expected: FAIL (undefined constants)

- [ ] **Step 3: Add constants to hub.go**

```go
// In backend/internal/websocket/hub.go, near existing event constants like EventTimerTick:
const (
	EventAgentMessage = "agent_message"
	EventAgentTool    = "agent_tool"
	EventAgentDone    = "agent_done"
)
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/websocket/ -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/websocket/hub.go backend/internal/websocket/hub_agent_events_test.go
git commit -m "feat(ws): add agent_message/agent_tool/agent_done event constants"
```

---

### Task 5: AgentService Core (no-tool-call path)

**Files:**
- Create: `backend/internal/agent/service.go`
- Create: `backend/internal/agent/conversation.go`
- Create: `backend/internal/agent/prompts.go`
- Create: `backend/internal/agent/service_test.go`

**Interfaces:**
- Consumes: `repository.AgentRepository`, `ai.LLMClient`, `ai.Message`
- Produces: `agent.AgentService` interface, `NewAgentService(deps AgentDeps) AgentService`, `agent.AgentDeps` struct

- [ ] **Step 1: Write the failing test**

```go
// backend/internal/agent/service_test.go
package agent

import (
	"context"
	"testing"

	"ticktask/internal/ai"
)

// mockLLM returns a queued sequence of responses
type mockLLM struct {
	responses []ai.ToolResponse
	calls     int
}

func (m *mockLLM) ChatCompletion(ctx context.Context, prompt string) (string, error) {
	return "", nil
}

func (m *mockLLM) ChatWithTools(ctx context.Context, msgs []ai.Message, tools []ai.ToolSpec) (ai.ToolResponse, error) {
	if m.calls >= len(m.responses) {
		return ai.ToolResponse{FinishReason: "stop"}, nil
	}
	r := m.responses[m.calls]
	m.calls++
	return r, nil
}

// mockHub records broadcasts
type mockHub struct{ events []mockEvent }
type mockEvent struct{ Type string; Payload any }

func (h *mockHub) Broadcast(msg map[string]any, _ ...string) {
	if t, ok := msg["type"].(string); ok {
		h.events = append(h.events, mockEvent{Type: t, Payload: msg})
	}
}

func TestAgentService_NoToolCall(t *testing.T) {
	llm := &mockLLM{responses: []ai.ToolResponse{
		{Content: "hi there", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	svc := NewAgentService(AgentDeps{
		Repo:    newInMemoryRepo(t),
		LLM:     llm,
		Registry: NewToolRegistry(),
		Hub:     hub,
		System:  "you are nice",
	})
	conv, _ := svc.Repo.CreateConversation()
	if err := svc.SendMessage(context.Background(), conv.ID, "hello"); err != nil {
		t.Fatalf("send: %v", err)
	}
	// Expect: 1 agent_message + 1 agent_done
	if len(hub.events) != 2 {
		t.Fatalf("events = %d, want 2: %+v", len(hub.events), hub.events)
	}
	if hub.events[0].Type != "agent_message" {
		t.Errorf("first event = %q", hub.events[0].Type)
	}
	if hub.events[1].Type != "agent_done" {
		t.Errorf("last event = %q", hub.events[1].Type)
	}
}
```

Note: `newInMemoryRepo(t)` helper wraps `repository.NewAgentRepository(setupTestDB(t))` — reuse from Task 1's setup but place helper here. To avoid cyclic imports between `agent` and `repository` test packages, the helper creates a `*gorm.DB` directly.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/agent/ -run TestAgentService_NoToolCall -v
```
Expected: FAIL

- [ ] **Step 3: Write implementation**

```go
// backend/internal/agent/prompts.go
package agent

const DefaultSystemPrompt = `You are TickTask Agent, a personal time-management assistant.

You can call tools to manage the user's tasks, pomodoro timer, schedule, and work log.
Tool calls that modify state require user confirmation. Be concise and friendly.
Available tools are listed in the tool schema. Always explain what you're about to do before calling tools.`
```

```go
// backend/internal/agent/conversation.go
package agent

import (
	"ticktask/internal/ai"
	"ticktask/internal/model"
)

func buildLLMMessages(system string, history []*model.AgentMessage) []ai.Message {
	msgs := []ai.Message{{Role: "system", Content: system}}
	for _, m := range history {
		switch m.Role {
		case "user":
			msgs = append(msgs, ai.Message{Role: "user", Content: m.Content})
		case "assistant":
			msgs = append(msgs, ai.Message{Role: "assistant", Content: m.Content})
		case "tool_result":
			if m.ToolName != nil {
				msgs = append(msgs, ai.Message{
					Role: "tool", Name: *m.ToolName, ToolCallID: deref(m.ParentID), Content: derep(m.ToolResult),
				})
			}
		case "tool_call":
			// assistant message carrying tool_calls is reconstructed in the previous assistant turn;
			// here we skip; full reconstruction handled when persisting assistant turns with tool_calls
		}
	}
	return msgs
}

func deref(p *string) string { if p == nil { return "" }; return *p }
func derep(p *string) string { if p == nil { return "" }; return *p }
```

```go
// backend/internal/agent/service.go
package agent

import (
	"context"
	"encoding/json"
	"fmt"

	"ticktask/internal/ai"
	"ticktask/internal/repository"
	"ticktask/internal/websocket"
)

type HubBroadcaster interface {
	Broadcast(msg map[string]any, types ...string)
}

type AgentDeps struct {
	Repo     repository.AgentRepository
	LLM      ai.LLMClient
	Registry ToolRegistry
	Hub      HubBroadcaster
	System   string
}

type AgentService interface {
	SendMessage(ctx context.Context, convID, text string) error
	Confirm(ctx context.Context, msgID string, decision string) error
	RunTool(ctx context.Context, name string, args json.RawMessage) (any, error)
}

type agentService struct{ AgentDeps }

func NewAgentService(d AgentDeps) AgentService {
	if d.System == "" {
		d.System = DefaultSystemPrompt
	}
	return &agentService{AgentDeps: d}
}

func (s *agentService) SendMessage(ctx context.Context, convID, text string) error {
	if _, err := s.Repo.GetConversation(convID); err != nil {
		return err
	}
	if _, err := s.Repo.AppendMessage(convID, "user", text, nil, nil, nil, nil); err != nil {
		return err
	}
	return s.runTurn(ctx, convID, 0)
}

func (s *agentService) runTurn(ctx context.Context, convID string, toolCount int) error {
	for toolCount < MaxToolCallsPerTurn {
		history, err := s.Repo.LoadRecentMessages(convID, MaxContextMessages)
		if err != nil {
			return err
		}
		msgs := buildLLMMessages(s.System, history)
		resp, err := s.LLM.ChatWithTools(ctx, msgs, s.Registry.ToOpenAITools())
		if err != nil {
			s.broadcastDone(convID, "error")
			return err
		}
		if resp.Content != "" {
			s.broadcast(convID, websocket.EventAgentMessage, map[string]any{
				"conversation_id": convID, "delta_text": resp.Content,
			})
			s.Repo.AppendMessage(convID, "assistant", resp.Content, nil, nil, nil, nil)
		}
		if len(resp.ToolCalls) == 0 {
			s.broadcastDone(convID, "stop")
			return nil
		}
		// Tool execution handled in Tasks 6-9
		s.broadcastDone(convID, "stop")
		return nil
	}
	s.broadcastDone(convID, "max_tools")
	return nil
}

func (s *agentService) Confirm(ctx context.Context, msgID, decision string) error {
	return nil // implemented in Task 7
}

func (s *agentService) RunTool(ctx context.Context, name string, args json.RawMessage) (any, error) {
	t, err := s.Registry.Lookup(name)
	if err != nil {
		return nil, err
	}
	return t.Execute(ctx, args)
}

func (s *agentService) broadcast(convID, eventType string, payload map[string]any) {
	msg := map[string]any{"type": eventType}
	for k, v := range payload {
		msg[k] = v
	}
	s.Hub.Broadcast(msg)
}

func (s *agentService) broadcastDone(convID, reason string) {
	s.broadcast(convID, websocket.EventAgentDone, map[string]any{
		"conversation_id": convID, "finish_reason": reason,
	})
}

var _ = fmt.Sprintf
```

The `newInMemoryRepo(t)` helper for tests:

```go
// Add to service_test.go (or a shared test_helper_test.go in same package)
func newInMemoryRepo(t *testing.T) repository.AgentRepository {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&model.AgentConversation{}, &model.AgentMessage{}); err != nil {
		t.Fatal(err)
	}
	return repository.NewAgentRepository(db)
}
```

Add necessary imports to `service_test.go`: `gorm.io/driver/sqlite`, `gorm.io/gorm`, `ticktask/internal/model`, `ticktask/internal/repository`.

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/agent/ -run TestAgentService_NoToolCall -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/agent/service.go backend/internal/agent/conversation.go backend/internal/agent/prompts.go backend/internal/agent/service_test.go
git commit -m "feat(agent): add AgentService core (no-tool-call path)"
```

---

## M2 — AgentService Full Orchestration

### Task 6: PermRead Tool Execution Path

**Files:**
- Modify: `backend/internal/agent/service.go` (extend `runTurn`)
- Modify: `backend/internal/agent/service_test.go`

**Interfaces:**
- Consumes: Task 2 registry, Task 5 service
- Produces: extended `runTurn` that executes PermRead tools and broadcasts `agent_tool` events

- [ ] **Step 1: Write the failing test**

```go
// Append to service_test.go
func TestAgentService_PermReadToolAutoExecutes(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&fakeTool{name: "list_tasks", perm: PermRead})
	llm := &mockLLM{responses: []ai.ToolResponse{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "list_tasks", Args: json.RawMessage(`{"status":"todo"}`)}}, FinishReason: "tool_calls"},
		{Content: "all done", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	svc := NewAgentService(AgentDeps{Repo: newInMemoryRepo(t), LLM: llm, Registry: reg, Hub: hub})
	conv, _ := svc.Repo.CreateConversation()
	if err := svc.SendMessage(context.Background(), conv.ID, "go"); err != nil {
		t.Fatal(err)
	}
	// Expect at least: 1 agent_tool(succeeded) + 1 agent_message + 1 agent_done
	found := false
	for _, e := range hub.events {
		if e.Type == "agent_tool" {
			p := e.Payload.(map[string]any)
			if p["status"] == "succeeded" && p["tool_name"] == "list_tasks" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("no succeeded tool event in %+v", hub.events)
	}
}
```

Add `"encoding/json"` import to test file.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/agent/ -run TestAgentService_PermReadToolAutoExecutes -v
```
Expected: FAIL (current `runTurn` skips tool execution)

- [ ] **Step 3: Modify runTurn to execute PermRead tools**

Replace the tool-stub block in `service.go` `runTurn`:

```go
// In runTurn, replace:
//   // Tool execution handled in Tasks 6-9
//   s.broadcastDone(convID, "stop")
//   return nil

// With:
for _, tc := range resp.ToolCalls {
    toolCount++
    tool, err := s.Registry.Lookup(tc.Name)
    if err != nil {
        s.broadcastTool(convID, "", tc.Name, tc.Args, "failed", nil, fmt.Sprintf("tool not found: %s", tc.Name))
        s.appendToolResult(convID, tc, "failed", `{"error":"not found"}`)
        continue
    }
    perm := tool.Schema().Permission
    if perm == PermRead {
        s.broadcastTool(convID, "", tc.Name, tc.Args, "started", nil, "")
        result, err := tool.Execute(ctx, tc.Args)
        if err != nil {
            s.broadcastTool(convID, "", tc.Name, tc.Args, "failed", nil, err.Error())
            s.appendToolResult(convID, tc, "failed", fmt.Sprintf(`{"error":%q}`, err.Error()))
        } else {
            rjson, _ := json.Marshal(result)
            s.broadcastTool(convID, "", tc.Name, tc.Args, "succeeded", result, "")
            s.appendToolResult(convID, tc, "succeeded", string(rjson))
        }
    } else {
        // PermWrite / PermDangerous handled in Tasks 7-8
        s.broadcastTool(convID, "", tc.Name, tc.Args, "started", nil, "")
        s.broadcastDone(convID, "stop")
        return nil
    }
}
```

Add helpers to `service.go`:

```go
func (s *agentService) broadcastTool(convID, msgID, name string, args json.RawMessage, status string, result any, errMsg string) {
	payload := map[string]any{
		"conversation_id": convID, "tool_name": name,
		"args": json.RawMessage(args), "status": status,
	}
	if msgID != "" {
		payload["message_id"] = msgID
	}
	if result != nil {
		payload["result"] = result
	}
	if errMsg != "" {
		payload["error"] = errMsg
	}
	s.broadcast(convID, websocket.EventAgentTool, payload)
}

func (s *agentService) appendToolResult(convID string, tc ai.ToolCall, status, resultJSON string) {
	statusPtr := &status
	resultPtr := &resultJSON
	s.Repo.AppendMessage(convID, "tool_result", "", &tc.Name, nil, resultPtr, statusPtr)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/agent/ -run TestAgentService_PermReadToolAutoExecutes -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/agent/service.go backend/internal/agent/service_test.go
git commit -m "feat(agent): auto-execute PermRead tools in runTurn"
```

---

### Task 7: PermWrite Tool + Confirmation Flow

**Files:**
- Modify: `backend/internal/agent/service.go`
- Modify: `backend/internal/agent/service_test.go`

**Interfaces:**
- Produces: `AgentService.Confirm(ctx, msgID, decision)` full impl; `pending_confirmation` event; tool message persistence with `started`/`pending_confirmation`/`succeeded`/`failed`/`rejected` status

- [ ] **Step 1: Write the failing test**

```go
// Append to service_test.go
type pendingState struct {
	convID   string
	pending  chan confirmSignal
}

type confirmSignal struct {
	msgID    string
	decision string
}

func TestAgentService_PermWriteTool_Approve(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&fakeTool{name: "create_task", perm: PermWrite})
	var llmResponses = []ai.ToolResponse{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "create_task", Args: json.RawMessage(`{"title":"x"}`)}}, FinishReason: "tool_calls"},
	}
	llm := &mockLLM{responses: llmResponses}
	hub := &mockHub{}
	svc := NewAgentService(AgentDeps{Repo: newInMemoryRepo(t), LLM: llm, Registry: reg, Hub: hub})
	conv, _ := svc.Repo.CreateConversation()

	// Run SendMessage in goroutine; it should block on pending confirmation
	done := make(chan error, 1)
	go func() { done <- svc.SendMessage(context.Background(), conv.ID, "go") }()

	// Wait for pending_confirmation event
	var msgID string
	for i := 0; i < 50; i++ {
		for _, e := range hub.events {
			if e.Type == "agent_tool" {
				p := e.Payload.(map[string]any)
				if p["status"] == "pending_confirmation" {
					msgID, _ = p["message_id"].(string)
				}
			}
		}
		if msgID != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if msgID == "" {
		t.Fatal("no pending_confirmation event")
	}
	if err := svc.Confirm(context.Background(), msgID, "approve"); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("SendMessage never returned")
	}
	// Should have a succeeded event
	found := false
	for _, e := range hub.events {
		if e.Type == "agent_tool" {
			if p, ok := e.Payload.(map[string]any); ok && p["status"] == "succeeded" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no succeeded event")
	}
}

func TestAgentService_PermWriteTool_Reject(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&fakeTool{name: "create_task", perm: PermWrite})
	llm := &mockLLM{responses: []ai.ToolResponse{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "create_task", Args: json.RawMessage(`{}`)}}, FinishReason: "tool_calls"},
		{Content: "ok I won't", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	svc := NewAgentService(AgentDeps{Repo: newInMemoryRepo(t), LLM: llm, Registry: reg, Hub: hub})
	conv, _ := svc.Repo.CreateConversation()
	done := make(chan error, 1)
	go func() { done <- svc.SendMessage(context.Background(), conv.ID, "go") }()
	var msgID string
	for i := 0; i < 50; i++ {
		for _, e := range hub.events {
			if e.Type == "agent_tool" {
				if p, ok := e.Payload.(map[string]any); ok && p["status"] == "pending_confirmation" {
					msgID, _ = p["message_id"].(string)
				}
			}
		}
		if msgID != "" { break }
		time.Sleep(10 * time.Millisecond)
	}
	svc.Confirm(context.Background(), msgID, "reject")
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("never returned")
	}
	found := false
	for _, e := range hub.events {
		if e.Type == "agent_tool" {
			if p, ok := e.Payload.(map[string]any); ok && p["status"] == "rejected" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("no rejected event")
	}
}
```

Add `"time"` import.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/agent/ -run TestAgentService_PermWriteTool -v
```
Expected: FAIL

- [ ] **Step 3: Implement confirmation flow**

Add to `service.go`:

```go
type agentService struct {
	AgentDeps
	pending      map[string]chan string // msgID -> decision channel
	pendingMu    sync.Mutex
}

func NewAgentService(d AgentDeps) AgentService {
	if d.System == "" {
		d.System = DefaultSystemPrompt
	}
	return &agentService{AgentDeps: d, pending: make(map[string]chan string)}
}
```

Update the `PermWrite / PermDangerous` branch in `runTurn`:

```go
} else {
    // PermWrite / PermDangerous
    preview, _ := tool.Preview(ctx, tc.Args)
    status := "pending_confirmation"
    msgID, _ := s.Repo.AppendMessage(convID, "tool_call", "", &tc.Name, strPtr(string(tc.Args)), nil, &status)
    s.broadcastTool(convID, msgID, tc.Name, tc.Args, "pending_confirmation", preview, "")
    ch := make(chan string, 1)
    s.setPending(msgID, ch)
    select {
    case decision := <-ch:
        if decision != "approve" {
            s.broadcastTool(convID, msgID, tc.Name, tc.Args, "rejected", nil, "")
            s.Repo.UpdateMessage(msgID, strPtr("rejected"), strPtr(`{"rejected":true}`))
            continue
        }
    case <-time.After(ConfirmationTimeout):
        s.broadcastTool(convID, msgID, tc.Name, tc.Args, "rejected", nil, "timeout")
        s.Repo.UpdateMessage(msgID, strPtr("rejected"), strPtr(`{"error":"timeout"}`))
        s.clearPending(msgID)
        continue
    case <-ctx.Done():
        return ctx.Err()
    }
    s.clearPending(msgID)
    result, err := tool.Execute(ctx, tc.Args)
    if err != nil {
        s.broadcastTool(convID, msgID, tc.Name, tc.Args, "failed", nil, err.Error())
        s.Repo.UpdateMessage(msgID, strPtr("failed"), strPtr(fmt.Sprintf(`{"error":%q}`, err.Error())))
    } else {
        rjson, _ := json.Marshal(result)
        s.broadcastTool(convID, msgID, tc.Name, tc.Args, "succeeded", result, "")
        s.Repo.UpdateMessage(msgID, strPtr("succeeded"), strPtr(string(rjson)))
    }
}
```

Replace the `Confirm` stub:

```go
func (s *agentService) Confirm(ctx context.Context, msgID, decision string) error {
	ch, ok := s.getPending(msgID)
	if !ok {
		return ErrToolNotFound
	}
	select {
	case ch <- decision:
		return nil
	default:
		return ErrToolNotFound
	}
}

func (s *agentService) setPending(msgID string, ch chan string) {
	s.pendingMu.Lock()
	s.pending[msgID] = ch
	s.pendingMu.Unlock()
}
func (s *agentService) getPending(msgID string) (chan string, bool) {
	s.pendingMu.Lock()
	defer s.pendingMu.Unlock()
	ch, ok := s.pending[msgID]
	return ch, ok
}
func (s *agentService) clearPending(msgID string) {
	s.pendingMu.Lock()
	delete(s.pending, msgID)
	s.pendingMu.Unlock()
}
```

Add helpers: `func strPtr(s string) *string { return &s }`.

Add imports: `"sync"`, `"time"`.

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/agent/ -run TestAgentService_PermWriteTool -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/agent/service.go backend/internal/agent/service_test.go
git commit -m "feat(agent): add PermWrite confirmation flow with timeout"
```

---

### Task 8: PermDangerous + Preview Path

**Files:**
- Modify: `backend/internal/agent/service.go` (already handles Dangerous via the same branch — only test additions)
- Modify: `backend/internal/agent/service_test.go`

- [ ] **Step 1: Write the failing test**

```go
// Append to service_test.go
func TestAgentService_PermDangerousTool_SecondConfirm(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&fakeTool{name: "delete_task", perm: PermDangerous})
	llm := &mockLLM{responses: []ai.ToolResponse{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "delete_task", Args: json.RawMessage(`{"task_id":"12"}`)}}, FinishReason: "tool_calls"},
		{Content: "deleted", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	svc := NewAgentService(AgentDeps{Repo: newInMemoryRepo(t), LLM: llm, Registry: reg, Hub: hub})
	conv, _ := svc.Repo.CreateConversation()
	done := make(chan error, 1)
	go func() { done <- svc.SendMessage(context.Background(), conv.ID, "go") }()
	var msgID string
	for i := 0; i < 50; i++ {
		for _, e := range hub.events {
			if e.Type == "agent_tool" {
				if p, ok := e.Payload.(map[string]any); ok && p["status"] == "pending_confirmation" {
					if pn, _ := p["tool_name"].(string); pn == "delete_task" {
						msgID, _ = p["message_id"].(string)
					}
				}
			}
		}
		if msgID != "" { break }
		time.Sleep(10 * time.Millisecond)
	}
	if msgID == "" {
		t.Fatal("no pending_confirmation for dangerous tool")
	}
	// preview should be present
	var previewSeen bool
	for _, e := range hub.events {
		if e.Type == "agent_tool" {
			if p, ok := e.Payload.(map[string]any); ok && p["message_id"] == msgID {
				if _, ok := p["preview"]; ok { previewSeen = true }
			}
		}
	}
	if !previewSeen {
		t.Fatal("no preview field in dangerous pending_confirmation")
	}
	svc.Confirm(context.Background(), msgID, "approve")
	<-done
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/agent/ -run TestAgentService_PermDangerousTool_SecondConfirm -v
```
Expected: probably PASS already if Task 7 was generic; if FAIL, the gap is `Preview` call on Dangerous. Ensure service.go calls `Preview` for **both** PermWrite and PermDangerous (already done in Task 7).

- [ ] **Step 3: If failed, ensure Preview called for Dangerous branch**

(No code change needed if Task 7 already calls `Preview` in the shared branch.)

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/agent/ -run TestAgentService_PermDangerousTool_SecondConfirm -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/agent/service_test.go
git commit -m "test(agent): cover PermDangerous second-confirm preview path"
```

---

### Task 9: Boundaries + Error Re-feed + Schema Validation

**Files:**
- Modify: `backend/internal/agent/service.go`
- Modify: `backend/internal/agent/service_test.go`
- Create: `backend/internal/agent/schema.go`

- [ ] **Step 1: Write the failing tests**

```go
// Append to service_test.go
func TestAgentService_MaxToolCalls(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&fakeTool{name: "list_tasks", perm: PermRead})
	// LLM always returns the same tool_call
	llm := &mockLLM{responses: repeat(ai.ToolResponse{
		ToolCalls:    []ai.ToolCall{{ID: "c", Name: "list_tasks", Args: json.RawMessage(`{}`)}},
		FinishReason: "tool_calls",
	}, MaxToolCallsPerTurn+5)}
	hub := &mockHub{}
	svc := NewAgentService(AgentDeps{Repo: newInMemoryRepo(t), LLM: llm, Registry: reg, Hub: hub})
	conv, _ := svc.Repo.CreateConversation()
	if err := svc.SendMessage(context.Background(), conv.ID, "loop"); err != nil {
		t.Fatal(err)
	}
	var doneReason string
	for _, e := range hub.events {
		if e.Type == "agent_done" {
			doneReason, _ = e.Payload.(map[string]any)["finish_reason"].(string)
		}
	}
	if doneReason != "max_tools" {
		t.Fatalf("finish = %q, want max_tools", doneReason)
	}
}

func TestAgentService_SchemaValidationError(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&fakeTool{name: "list_tasks", perm: PermRead})
	// Tool call with bad args type
	llm := &mockLLM{responses: []ai.ToolResponse{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "list_tasks", Args: json.RawMessage(`{"status":123}`)}}, FinishReason: "tool_calls"},
		{Content: "ok", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	svc := NewAgentService(AgentDeps{Repo: newInMemoryRepo(t), LLM: llm, Registry: reg, Hub: hub})
	conv, _ := svc.Repo.CreateConversation()
	svc.SendMessage(context.Background(), conv.ID, "bad args")
	// Expect agent_tool failed with schema error
	var found bool
	for _, e := range hub.events {
		if e.Type == "agent_tool" {
			if p, ok := e.Payload.(map[string]any); ok && p["status"] == "failed" {
				if msg, _ := p["error"].(string); strings.Contains(msg, "schema") {
					found = true
				}
			}
		}
	}
	if !found { t.Fatal("no schema failure event") }
}

func repeat(r ai.ToolResponse, n int) []ai.ToolResponse {
	out := make([]ai.ToolResponse, n)
	for i := range out { out[i] = r }
	return out
}
```

Add `"strings"` import.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/agent/ -run "TestAgentService_MaxToolCalls|TestAgentService_SchemaValidationError" -v
```
Expected: FAIL

- [ ] **Step 3: Implement schema validation + max-tool enforcement**

```go
// backend/internal/agent/schema.go
package agent

import (
	"encoding/json"
	"errors"
	"fmt"
)

// ValidateArgs validates raw args against the tool's FunctionSpec.Parameters (lightweight: required + type check)
func ValidateArgs(schema map[string]any, args json.RawMessage) error {
	if len(schema) == 0 {
		return nil
	}
	var got map[string]any
	if err := json.Unmarshal(args, &got); err != nil {
		return fmt.Errorf("args not a json object: %w", err)
	}
	if t, ok := schema["type"].(string); ok && t == "object" {
		props, _ := schema["properties"].(map[string]any)
		for field, fval := range got {
			ps, ok := props[field].(map[string]any)
			if !ok {
				return errors.New("schema: unknown field " + field)
			}
			wantType, _ := ps["type"].(string)
			if err := checkType(wantType, fval); err != nil {
				return fmt.Errorf("schema: field %s: %w", field, err)
			}
		}
		req, _ := schema["required"].([]any)
		for _, r := range req {
			if rs, ok := r.(string); ok {
				if _, present := got[rs]; !present {
					return errors.New("schema: missing required field " + rs)
				}
			}
		}
	}
	return nil
}

func checkType(want string, v any) error {
	switch want {
	case "string":
		if _, ok := v.(string); !ok { return errors.New("expected string") }
	case "number", "integer":
		if _, ok := v.(float64); !ok { return errors.New("expected number") }
	case "boolean":
		if _, ok := v.(bool); !ok { return errors.New("expected boolean") }
	case "object":
		if _, ok := v.(map[string]any); !ok { return errors.New("expected object") }
	case "array":
		if _, ok := v.([]any); !ok { return errors.New("expected array") }
	}
	return nil
}
```

In `service.go`, before executing any tool:

```go
// In runTurn, inside the tool-call loop, BEFORE perm branch:
tool, err := s.Registry.Lookup(tc.Name)
if err != nil { /* unchanged: not-found path */ continue }
if err := ValidateArgs(tool.Schema().Function.Parameters, tc.Args); err != nil {
    s.broadcastTool(convID, "", tc.Name, tc.Args, "failed", nil, "schema: "+err.Error())
    s.appendToolResult(convID, tc, "failed", fmt.Sprintf(`{"schema_error":%q}`, err.Error()))
    continue
}
```

`MaxToolCallsPerTurn` enforcement is already in the for-loop condition (Task 5).

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/agent/ -v
```
Expected: all PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/agent/schema.go backend/internal/agent/service.go backend/internal/agent/service_test.go
git commit -m "feat(agent): enforce schema validation + max tool calls boundary"
```

---

## M3 — Tool Implementations

### Task 10: Task Tools (5 tools)

**Files:**
- Create: `backend/internal/agent/tools/register.go`
- Create: `backend/internal/agent/tools/task.go`
- Create: `backend/internal/agent/tools/task_test.go`

**Interfaces:**
- Consumes: `agent.Tool` (Task 2), `service.TaskService` (existing)
- Produces: 5 task tools registered via `tools.RegisterAll`

- [ ] **Step 1: Write the failing test (representative for `list_tasks`)**

```go
// backend/internal/agent/tools/task_test.go
package tools

import (
	"context"
	"encoding/json"
	"testing"
	"ticktask/internal/agent"
	"ticktask/internal/model"
)

type mockTaskSvc struct{ tasks []*model.Task }

func (m *mockTaskSvc) List(filter service.TaskFilter) ([]*model.Task, error) {
	return m.tasks, nil
}
func (m *mockTaskSvc) Get(id string) (*model.Task, error) { return nil, nil }
func (m *mockTaskSvc) Create(t *model.Task) (*model.Task, error) { return t, nil }
func (m *mockTaskSvc) Update(t *model.Task) (*model.Task, error) { return t, nil }
func (m *mockTaskSvc) Delete(id string) error { return nil }
// ... remaining TaskService methods stubbed

func TestListTasks(t *testing.T) {
	tt := &ListTasksTool{Svc: &mockTaskSvc{tasks: []*model.Task{{ID: "1", Title: "x"}}}}
	args := json.RawMessage(`{}`)
	res, err := tt.Execute(context.Background(), args)
	if err != nil { t.Fatal(err) }
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), "\"x\"") { t.Fatalf("res = %s", m) }
}

func TestListTasks_SchemaValidationFails(t *testing.T) {
	tt := &ListTasksTool{Svc: &mockTaskSvc{}}
	args := json.RawMessage(`{"status":123}`)
	_, err := tt.Execute(context.Background(), args)
	if err == nil { t.Fatal("expected schema error") }
}
```

The exact `mockTaskSvc` signature depends on the existing `service.TaskService` interface — read `backend/internal/service/task_service.go` to confirm method names before writing the mock. Add `"strings"`, `"ticktask/internal/service"` imports.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/agent/tools/ -v
```
Expected: FAIL (undefined symbols)

- [ ] **Step 3: Implement task tools**

```go
// backend/internal/agent/tools/task.go
package tools

import (
	"context"
	"encoding/json"

	"ticktask/internal/agent"
	"ticktask/internal/service"
)

type TaskDeps struct {
	Svc service.TaskService
}

type ListTasksTool struct{ Svc service.TaskService }

func (t *ListTasksTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "list_tasks",
		Function: agent.FunctionSpec{
			Description: "List tasks with optional filters",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status":   map[string]any{"type": "string"},
					"due":      map[string]any{"type": "string"},
					"quadrant": map[string]any{"type": "integer"},
				},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *ListTasksTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		Status   *string `json:"status"`
		Due      *string `json:"due"`
		Quadrant *int    `json:"quadrant"`
	}
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	json.Unmarshal(args, &in)
	tasks, err := t.Svc.List(service.TaskFilter{Status: in.Status, Due: in.Due, Quadrant: in.Quadrant})
	if err != nil { return nil, err }
	return map[string]any{"tasks": tasks}, nil
}

func (t *ListTasksTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args) // Read tool: preview == execute
}

type CreateTaskTool struct{ Svc service.TaskService }
func (t *CreateTaskTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "create_task",
		Function: agent.FunctionSpec{
			Description: "Create a new task",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":       map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"priority":    map[string]any{"type": "string"},
					"due":         map[string]any{"type": "string"},
				},
				"required": []any{"title"},
			},
		},
		Permission: agent.PermWrite,
	}
}
func (t *CreateTaskTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil { return nil, err }
	var in struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
		Priority    *string `json:"priority"`
		Due         *string `json:"due"`
	}
	json.Unmarshal(args, &in)
	task := &model.Task{Title: in.Title, Description: deref(in.Description)}
	return t.Svc.Create(task)
}
func (t *CreateTaskTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct{ Title string `json:"title"` }
	json.Unmarshal(args, &in)
	return map[string]any{"action": "create", "title": in.Title}, nil
}

// Implement UpdateTaskTool (PermWrite), DeleteTaskTool (PermDangerous), ClassifyTaskTool (PermRead) similarly.
// ClassifyTaskTool.Execute calls original AIService.ClassifyTask logic — ported from service/ai_service.go.
```

Add `deref` helper. Confirm `service.TaskFilter` exists; if not, adapt to the actual signature.

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/agent/tools/ -v
```
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add backend/internal/agent/tools/
git commit -m "feat(agent/tools): add task tools (list/create/update/delete/classify)"
```

---

### Task 11: Timer + Schedule + Insight Tools (4 tools)

**Files:**
- Create: `backend/internal/agent/tools/timer.go`
- Create: `backend/internal/agent/tools/schedule.go`
- Create: `backend/internal/agent/tools/insight.go`
- Create: `backend/internal/agent/tools/timer_test.go`
- Create: `backend/internal/agent/tools/schedule_test.go`

**Pattern**：same as Task 10. For each tool:
- Step 1: write 2-3 representative tests (success + schema error)
- Step 2: run, verify fail
- Step 3: implement (TimerDeps wraps `service.TimerService`; ScheduleDeps wraps `service.ScheduleService` + `ai.LLMClient` for `generate_schedule` since it calls LLM; InsightDeps wraps `service.AnalyticsService` + `ai.LLMClient`)
- Step 4: run, verify pass
- Step 5: commit

**Key signatures**:

```go
// tools/timer.go
type StartPomodoroTool struct{ Svc service.TimerService }
// Schema: name="start_pomodoro", perm=PermWrite, params={task_id?:string, duration_min?:integer}

type StopPomodoroTool struct{ Svc service.TimerService }
// name="stop_pomodoro", perm=PermWrite, params={}

type GetTimerStatusTool struct{ Svc service.TimerService }
// name="get_timer_status", perm=PermRead, params={}
```

```go
// tools/schedule.go
type GenerateScheduleTool struct {
	Svc service.ScheduleService
	LLM ai.LLMClient
}
// name="generate_schedule", perm=PermWrite, params={date?:string, pomodoro_settings?:object}
// Execute: port logic from service/ai_service.go GenerateDailySchedule

type ListScheduleTool struct{ Svc service.ScheduleService }
// name="list_schedule", perm=PermRead, params={from:string, to:string}
```

```go
// tools/insight.go
type GetDailyInsightsTool struct {
	Svc service.AnalyticsService
	LLM ai.LLMClient
}
// name="get_daily_insights", perm=PermRead, params={date:string}
```

For `GenerateScheduleTool.Execute` and `GetDailyInsightsTool.Execute`, port the prompt-building logic from `service/ai_service.go` (which is being deleted in Task 15) — copy the prompt templates verbatim into the tool files.

**Commit message**: `feat(agent/tools): add timer/schedule/insight tools`

---

### Task 12: WorkLog Tools + Migrate work_log_ai_client

**Files:**
- Create: `backend/internal/agent/tools/worklog.go`
- Create: `backend/internal/agent/tools/worklog_test.go`
- (Delete happens in Task 15)

**Interfaces:**
- Produces: `StructureWorklogTool` (PermRead), `SaveWorklogTool` (PermWrite)

- [ ] **Step 1: Write the failing test**

```go
// worklog_test.go — test StructureWorklogTool success + LLM parse failure
// test SaveWorklogTool success + missing items schema error
```

- [ ] **Step 2: Run test to verify it fails**

- [ ] **Step 3: Implement**

```go
// backend/internal/agent/tools/worklog.go
package tools

import (
	"context"
	"encoding/json"

	"ticktask/internal/ai"
	"ticktask/internal/agent"
	"ticktask/internal/service"
)

type StructureWorklogTool struct {
	LLM ai.LLMClient
}

func (t *StructureWorklogTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "structure_worklog",
		Function: agent.FunctionSpec{
			Description: "Structure a brain-dump into 4-dimensional work items",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"brain_dump": map[string]any{"type": "string"},
				},
				"required": []any{"brain_dump"},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *StructureWorklogTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil { return nil, err }
	var in struct{ BrainDump string `json:"brain_dump"` }
	json.Unmarshal(args, &in)
	// Port prompt from internal/ai/work_log_prompts.go WorkLogStructurePrompt
	prompt := workLogStructurePrompt(in.BrainDump)
	out, err := t.LLM.ChatCompletion(ctx, prompt)
	if err != nil { return nil, err }
	var items []map[string]any
	if err := json.Unmarshal([]byte(out), &items); err != nil {
		return nil, fmt.Errorf("parse LLM output: %w", err)
	}
	return map[string]any{"items": items}, nil
}

func (t *StructureWorklogTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}

type SaveWorklogTool struct{ Svc service.WorkLogService }

func (t *SaveWorklogTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "save_worklog",
		Function: agent.FunctionSpec{
			Description: "Save a structured work log entry",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"items": map[string]any{"type": "array"},
					"date":  map[string]any{"type": "string"},
				},
				"required": []any{"items", "date"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *SaveWorklogTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil { return nil, err }
	var in struct {
		Items []map[string]any `json:"items"`
		Date  string           `json:"date"`
	}
	json.Unmarshal(args, &in)
	return t.Svc.Save(in.Date, in.Items)
}
func (t *SaveWorklogTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct{ Date string `json:"date"` }
	json.Unmarshal(args, &in)
	return map[string]any{"action": "save_worklog", "date": in.Date, "items_count": "TBD"}, nil
}
```

Copy `workLogStructurePrompt` from existing `internal/ai/work_log_prompts.go` (file deleted in Task 15).

- [ ] **Step 4: Run test to verify it passes**

- [ ] **Step 5: Commit**

```bash
git add backend/internal/agent/tools/worklog.go backend/internal/agent/tools/worklog_test.go
git commit -m "feat(agent/tools): add worklog tools (structure/save)"
```

---

## M4 — HTTP API + Wiring + AIService Removal

### Task 13: agent_handler.go + Routes

**Files:**
- Create: `backend/internal/api/handler/agent_handler.go`
- Create: `backend/internal/api/handler/agent_handler_test.go`
- Modify: `backend/internal/api/router.go`

**Interfaces:**
- Produces: `handler.AgentHandler` interface, `NewAgentHandler(svc agent.AgentService, repo repository.AgentRepository)`, 7 HTTP endpoints

- [ ] **Step 1: Write the failing test**

```go
// agent_handler_test.go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAgentHandler_CreateConversation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAgentHandler(mockAgentSvc{}, mockAgentRepo{})
	h.Register(r.Group("/api/agent"))

	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, err := http.Post(srv.URL+"/api/agent/conversations", "application/json", bytes.NewReader([]byte("{}")))
	if err != nil { t.Fatal(err) }
	if resp.StatusCode != 201 {
		t.Fatalf("status = %d", resp.StatusCode)
	}
}

func TestAgentHandler_Status(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	h := NewAgentHandler(mockAgentSvc{}, mockAgentRepo{})
	h.Register(r.Group("/api/agent"))
	srv := httptest.NewServer(r)
	defer srv.Close()

	resp, _ := http.Get(srv.URL + "/api/agent/status")
	if resp.StatusCode != 200 { t.Fatalf("status = %d", resp.StatusCode) }
	var body map[string]any
	json.NewDecoder(resp.Body).Decode(&body)
	if _, ok := body["configured"]; !ok {
		t.Fatal("missing configured")
	}
}

// mockAgentSvc and mockAgentRepo live in mocks_test.go (extend existing)
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/api/handler/ -run TestAgentHandler -v
```

- [ ] **Step 3: Implement**

```go
// backend/internal/api/handler/agent_handler.go
package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"ticktask/internal/agent"
	"ticktask/internal/repository"
)

type AgentHandler interface {
	Register(rg *gin.RouterGroup)
}

type agentHandler struct {
	Svc  agent.AgentService
	Repo repository.AgentRepository
}

func NewAgentHandler(svc agent.AgentService, repo repository.AgentRepository) AgentHandler {
	return &agentHandler{Svc: svc, Repo: repo}
}

func (h *agentHandler) Register(rg *gin.RouterGroup) {
	rg.POST("/conversations", h.createConversation)
	rg.GET("/conversations", h.listConversations)
	rg.GET("/conversations/:id/messages", h.getMessages)
	rg.DELETE("/conversations/:id", h.deleteConversation)
	rg.POST("/chat", h.chat)
	rg.POST("/run-tool", h.runTool)
	rg.POST("/confirm", h.confirm)
	rg.GET("/status", h.status)
}

func (h *agentHandler) createConversation(c *gin.Context) {
	conv, err := h.Repo.CreateConversation()
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(201, conv)
}

func (h *agentHandler) listConversations(c *gin.Context) {
	page := atoiDefault(c.Query("page"), 1)
	size := atoiDefault(c.Query("size"), 20)
	items, total, err := h.Repo.ListConversations(page, size)
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, gin.H{"items": items, "total": total})
}

func (h *agentHandler) getMessages(c *gin.Context) {
	msgs, err := h.Repo.ListMessages(c.Param("id"))
	if err != nil { c.JSON(500, gin.H{"error": err.Error()}); return }
	c.JSON(200, msgs)
}

func (h *agentHandler) deleteConversation(c *gin.Context) {
	if err := h.Repo.DeleteConversation(c.Param("id")); err != nil {
		c.JSON(500, gin.H{"error": err.Error()}); return
	}
	c.Status(204)
}

func (h *agentHandler) chat(c *gin.Context) {
	var in struct {
		ConversationID string `json:"conversation_id"`
		Text           string `json:"text"`
	}
	if err := c.BindJSON(&in); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	go h.Svc.SendMessage(c.Request.Context(), in.ConversationID, in.Text)
	c.Status(202)
}

func (h *agentHandler) runTool(c *gin.Context) {
	var in struct {
		Tool string          `json:"tool"`
		Args json.RawMessage `json:"args"`
	}
	if err := c.BindJSON(&in); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	res, err := h.Svc.RunTool(c.Request.Context(), in.Tool, in.Args)
	if err != nil {
		if errors.Is(err, agent.ErrToolNotFound) {
			c.JSON(404, gin.H{"error": err.Error()}); return
		}
		c.JSON(400, gin.H{"error": err.Error()}); return
	}
	c.JSON(200, gin.H{"result": res})
}

func (h *agentHandler) confirm(c *gin.Context) {
	var in struct {
		MessageID string `json:"message_id"`
		Decision  string `json:"decision"`
	}
	if err := c.BindJSON(&in); err != nil { c.JSON(400, gin.H{"error": err.Error()}); return }
	if err := h.Svc.Confirm(c.Request.Context(), in.MessageID, in.Decision); err != nil {
		c.JSON(404, gin.H{"error": err.Error()}); return
	}
	c.Status(200)
}

func (h *agentHandler) status(c *gin.Context) {
	c.JSON(200, h.Svc.Status())
}

func atoiDefault(s string, def int) int { /* helper */ }
```

Add `"encoding/json"`, `"errors"`, `"ticktask/internal/agent"` imports. Add `Status()` to `AgentService` interface returning `{configured, supports_function_calling, provider}`.

- [ ] **Step 4: Run test to verify it passes**

- [ ] **Step 5: Commit**

```bash
git add backend/internal/api/handler/agent_handler.go backend/internal/api/handler/agent_handler_test.go backend/internal/api/handler/mocks_test.go backend/internal/agent/service.go
git commit -m "feat(api): add agent_handler with 8 endpoints"
```

---

### Task 14: main.go Wiring + AutoMigrate

**Files:**
- Modify: `backend/cmd/server/main.go`
- Modify: `backend/internal/api/router.go`

- [ ] **Step 1: Read existing main.go to confirm DI pattern**

```bash
cd backend && cat cmd/server/main.go
```

- [ ] **Step 2: Modify wiring**

```go
// In cmd/server/main.go after constructing taskSvc, timerSvc, schedSvc, workLogSvc, llmClient:

agentRepo := repository.NewAgentRepository(db)
registry := agent.NewToolRegistry()
tools.RegisterAll(registry, tools.Deps{
	Task: taskSvc, Timer: timerSvc, Schedule: schedSvc, WorkLog: workLogSvc, Analytics: analyticsSvc, LLM: llmClient,
})
agentSvc := agent.NewAgentService(agent.AgentDeps{
	Repo: agentRepo, LLM: llmClient, Registry: registry, Hub: hub, System: agent.DefaultSystemPrompt,
})
agentHandler := handler.NewAgentHandler(agentSvc, agentRepo)

// Pass agentHandler to SetupRouter
```

In `router.go`, add `/api/agent` group with `agentHandler.Register(...)`.

Add `&model.AgentConversation{}, &model.AgentMessage{}` to existing AutoMigrate in `pkg/database/database.go`.

- [ ] **Step 3: Run all backend tests + smoke run**

```bash
cd backend && go test ./... && go build ./cmd/server
```

- [ ] **Step 4: Manual smoke test**

```bash
cd backend && go run cmd/server/main.go &
sleep 2
curl -X POST http://localhost:8080/api/agent/conversations
curl http://localhost:8080/api/agent/status
```

Expected: 201 + JSON conversation; 200 + `{"configured":...}`.

- [ ] **Step 5: Commit**

```bash
git add backend/cmd/server/main.go backend/internal/api/router.go backend/pkg/database/database.go backend/internal/agent/tools/register.go
git commit -m "chore(agent): wire AgentService in main.go and register /api/agent routes"
```

`tools/register.go` content:

```go
package tools

import (
	"ticktask/internal/ai"
	"ticktask/internal/agent"
	"ticktask/internal/service"
)

type Deps struct {
	Task      service.TaskService
	Timer     service.TimerService
	Schedule  service.ScheduleService
	WorkLog   service.WorkLogService
	Analytics service.AnalyticsService
	LLM       ai.LLMClient
}

func RegisterAll(reg agent.ToolRegistry, d Deps) {
	reg.MustRegister(&ListTasksTool{Svc: d.Task})
	reg.MustRegister(&CreateTaskTool{Svc: d.Task})
	reg.MustRegister(&UpdateTaskTool{Svc: d.Task})
	reg.MustRegister(&DeleteTaskTool{Svc: d.Task})
	reg.MustRegister(&ClassifyTaskTool{Svc: d.Task, LLM: d.LLM})
	reg.MustRegister(&StartPomodoroTool{Svc: d.Timer})
	reg.MustRegister(&StopPomodoroTool{Svc: d.Timer})
	reg.MustRegister(&GetTimerStatusTool{Svc: d.Timer})
	reg.MustRegister(&GenerateScheduleTool{Svc: d.Schedule, LLM: d.LLM})
	reg.MustRegister(&ListScheduleTool{Svc: d.Schedule})
	reg.MustRegister(&StructureWorklogTool{LLM: d.LLM})
	reg.MustRegister(&SaveWorklogTool{Svc: d.WorkLog})
	reg.MustRegister(&GetDailyInsightsTool{Svc: d.Analytics, LLM: d.LLM})
}
```

---

### Task 15: Delete AIService + ai_handler + /api/ai/* + Old Prompts

**Files:**
- Delete: `backend/internal/service/ai_service.go`
- Delete: `backend/internal/service/work_log_ai_client.go`
- Delete: `backend/internal/api/handler/ai_handler.go`
- Delete: `backend/internal/ai/prompts.go`
- Delete: `backend/internal/ai/work_log_prompts.go`
- Modify: `backend/internal/api/router.go` (remove `/api/ai` routes)
- Modify: `backend/cmd/server/main.go` (remove AIService wiring)
- Modify: `backend/internal/api/handler/mocks_test.go` (remove mockAIService)

- [ ] **Step 1: Find all references to AIService**

```bash
cd backend && grep -rn "AIService\|ai_handler\|/api/ai" --include="*.go"
```

- [ ] **Step 2: Remove references**

For each grep hit: replace with agent-equivalent or remove the call. Specifically:
- `main.go`: remove `aiSvc := service.NewAIService(...)` and `aiHandler := handler.NewAIHandler(aiSvc)`
- `router.go`: remove `aiHandler.Register(rg.Group("/api/ai"))`
- `handler/mocks_test.go`: remove `mockAIService` if no test uses it anymore
- Any service that injected `*AIService` (e.g. `workLogService` for `CallLLM`): change to inject `ai.LLMClient` directly

- [ ] **Step 3: Delete the files**

```bash
cd backend && rm internal/service/ai_service.go internal/service/work_log_ai_client.go internal/api/handler/ai_handler.go internal/ai/prompts.go internal/ai/work_log_prompts.go
```

- [ ] **Step 4: Run all backend tests + build**

```bash
cd backend && go test ./... && go build ./cmd/server
```

Expected: PASS, no compile errors.

- [ ] **Step 5: Commit**

```bash
git add -A backend/
git commit -m "refactor(agent): remove AIService and /api/ai/* (superseded by agent)"
```

---

## M5 — Frontend Foundation

### Task 16: Types + API Client agent Group

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/client.ts`

- [ ] **Step 1: Write the failing test**

```ts
// frontend/src/api/agent.spec.ts
import { describe, it, expect, vi } from 'vitest'
import { api } from './client'

vi.mock('./client', () => ({
  api: {
    agent: {
      listConversations: vi.fn().mockResolvedValue({ items: [], total: 0 }),
      createConversation: vi.fn(),
      runTool: vi.fn(),
    },
  },
}))

describe('api.agent', () => {
  it('has listConversations', async () => {
    const r = await api.agent.listConversations()
    expect(r.total).toBe(0)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/api/agent.spec.ts
```

- [ ] **Step 3: Add types and API group**

```ts
// Append to frontend/src/types/index.ts
export interface AgentConversation {
  id: string
  title: string
  created_at: string
  updated_at: string
  message_count: number
}

export type AgentMessageRole = 'user' | 'assistant' | 'tool_call' | 'tool_result'
export type ToolStatus = 'started' | 'pending_confirmation' | 'succeeded' | 'failed' | 'rejected'

export interface AgentMessage {
  id: string
  conversation_id: string
  role: AgentMessageRole
  content: string
  tool_name?: string
  tool_args?: string
  tool_result?: string
  tool_status?: ToolStatus
  parent_id?: string
  created_at: string
}

export interface AgentStatus {
  configured: boolean
  supports_function_calling: boolean
  provider: string
}

export interface AgentToolEvent {
  conversation_id: string
  message_id?: string
  tool_name: string
  args: Record<string, unknown>
  status: ToolStatus
  preview?: unknown
  result?: unknown
  error?: string
}

export type AgentWsEvent =
  | { type: 'agent_message'; conversation_id: string; message_id: string; delta_text: string }
  | { type: 'agent_tool'; } & AgentToolEvent
  | { type: 'agent_done'; conversation_id: string; finish_reason: 'stop' | 'max_tools' | 'error'; total_tokens?: number }
```

```ts
// Append agent group to frontend/src/api/client.ts
export const api = {
  // ... existing groups

  agent: {
    createConversation: () => http.post<AgentConversation>('/agent/conversations'),
    listConversations: (page = 1, size = 20) =>
      http.get<{ items: AgentConversation[]; total: number }>('/agent/conversations', { params: { page, size } }),
    getMessages: (id: string) => http.get<AgentMessage[]>(`/agent/conversations/${id}/messages`),
    deleteConversation: (id: string) => http.delete(`/agent/conversations/${id}`),
    chat: (conversationId: string, text: string) =>
      http.post('/agent/chat', { conversation_id: conversationId, text }),
    runTool: (tool: string, args: Record<string, unknown>) =>
      http.post<{ result: unknown }>('/agent/run-tool', { tool, args }),
    confirm: (messageId: string, decision: 'approve' | 'reject') =>
      http.post('/agent/confirm', { message_id: messageId, decision }),
    status: () => http.get<AgentStatus>('/agent/status'),
  },
}
```

Add imports for the new types at the top.

- [ ] **Step 4: Run test to verify it passes**

- [ ] **Step 5: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/api/client.ts frontend/src/api/agent.spec.ts
git commit -m "feat(api): add agent API client and types"
```

---

### Task 17: stores/agent.ts + WS Event Dispatch

**Files:**
- Create: `frontend/src/stores/agent.ts`
- Create: `frontend/src/stores/agent.spec.ts`
- Modify: `frontend/src/utils/websocket.ts`

- [ ] **Step 1: Write the failing test**

```ts
// frontend/src/stores/agent.spec.ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useAgentStore } from './agent'

vi.mock('@/api/client', () => ({
  api: {
    agent: {
      listConversations: vi.fn().mockResolvedValue({ items: [], total: 0 }),
      createConversation: vi.fn().mockResolvedValue({ id: 'c1', title: 'New' }),
      chat: vi.fn(),
      runTool: vi.fn().mockResolvedValue({ result: 'ok' }),
      confirm: vi.fn(),
      status: vi.fn().mockResolvedValue({ configured: true, supports_function_calling: true, provider: 'openai' }),
    },
  },
}))

describe('useAgentStore', () => {
  beforeEach(() => setActivePinia(createPinia()))
  it('opens/closes drawer', () => {
    const s = useAgentStore()
    expect(s.isOpen).toBe(false)
    s.openDrawer()
    expect(s.isOpen).toBe(true)
  })
  it('appends streaming tokens', () => {
    const s = useAgentStore()
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'a' })
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'b' })
    expect(s.streamingText).toBe('ab')
  })
  it('runTool calls API', async () => {
    const s = useAgentStore()
    const r = await s.runTool('classify_task', { task_id: '12' })
    expect(r).toBe('ok')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/stores/agent.spec.ts
```

- [ ] **Step 3: Implement store**

```ts
// frontend/src/stores/agent.ts
import { defineStore } from 'pinia'
import { api } from '@/api/client'
import type { AgentConversation, AgentMessage, AgentStatus, AgentWsEvent, ToolStatus } from '@/types'

interface PendingToolCall {
  messageId: string
  toolName: string
  args: Record<string, unknown>
  preview?: unknown
}

export const useAgentStore = defineStore('agent', {
  state: () => ({
    isOpen: false,
    status: { configured: false, supports_function_calling: false, provider: '' } as AgentStatus,
    conversations: [] as AgentConversation[],
    currentConvId: null as string | null,
    messages: [] as AgentMessage[],
    streamingText: '',
    streamingMessageId: null as string | null,
    pendingConfirm: null as PendingToolCall | null,
    isThinking: false,
  }),
  actions: {
    openDrawer() { this.isOpen = true },
    closeDrawer() { this.isOpen = false },
    toggleDrawer() { this.isOpen = !this.isOpen },
    async checkStatus() { this.status = await api.agent.status() },
    async listConversations() {
      const r = await api.agent.listConversations()
      this.conversations = r.items
    },
    async createConversation() {
      const c = await api.agent.createConversation()
      this.conversations.unshift(c)
      this.currentConvId = c.id
      this.messages = []
      return c
    },
    async switchConversation(id: string) {
      this.currentConvId = id
      this.messages = await api.agent.getMessages(id)
    },
    async sendMessage(text: string) {
      if (!this.currentConvId) await this.createConversation()
      this.messages.push({
        id: 'local-' + Date.now(), conversation_id: this.currentConvId!,
        role: 'user', content: text, created_at: new Date().toISOString(),
      })
      this.isThinking = true
      this.streamingText = ''
      await api.agent.chat(this.currentConvId!, text)
    },
    async runTool(name: string, args: Record<string, unknown>) {
      const r = await api.agent.runTool(name, args)
      return r.result
    },
    async confirmToolCall(messageId: string, decision: 'approve' | 'reject') {
      await api.agent.confirm(messageId, decision)
      this.pendingConfirm = null
    },
    onAgentMessage(e: Extract<AgentWsEvent, { type: 'agent_message' }>) {
      if (e.conversation_id !== this.currentConvId) return
      this.streamingMessageId = e.message_id
      this.streamingText += e.delta_text
    },
    onAgentTool(e: Extract<AgentWsEvent, { type: 'agent_tool' }>) {
      if (e.conversation_id !== this.currentConvId) return
      if (e.status === 'pending_confirmation') {
        this.pendingConfirm = {
          messageId: e.message_id!, toolName: e.tool_name, args: e.args, preview: e.preview,
        }
      }
      // Update or append the tool_call message locally based on message_id
    },
    onAgentDone(e: Extract<AgentWsEvent, { type: 'agent_done' }>) {
      if (e.conversation_id !== this.currentConvId) return
      if (this.streamingText) {
        this.messages.push({
          id: this.streamingMessageId || 'ast-' + Date.now(),
          conversation_id: this.currentConvId!,
          role: 'assistant', content: this.streamingText, created_at: new Date().toISOString(),
        })
      }
      this.streamingText = ''
      this.streamingMessageId = null
      this.isThinking = false
    },
    handleWsEvent(e: AgentWsEvent) {
      if (e.type === 'agent_message') this.onAgentMessage(e)
      else if (e.type === 'agent_tool') this.onAgentTool(e)
      else if (e.type === 'agent_done') this.onAgentDone(e)
    },
  },
})
```

In `utils/websocket.ts`, add agent event dispatch:

```ts
// In the WS message handler, after existing event branches:
if (msg.type === 'agent_message' || msg.type === 'agent_tool' || msg.type === 'agent_done') {
  useAgentStore().handleWsEvent(msg)
}
```

(Careful about circular import; if it occurs, expose a registry callback in wsClient.)

- [ ] **Step 4: Run test to verify it passes**

- [ ] **Step 5: Commit**

```bash
git add frontend/src/stores/agent.ts frontend/src/stores/agent.spec.ts frontend/src/utils/websocket.ts
git commit -m "feat(store): add agent Pinia store + WS event dispatch"
```

---

## M6 — Frontend Components

### Task 18: AgentDrawer + MessageList + Input

**Files:**
- Create: `frontend/src/components/agent/AgentDrawer.vue`
- Create: `frontend/src/components/agent/AgentMessageList.vue`
- Create: `frontend/src/components/agent/AgentInput.vue`
- Create: `frontend/src/components/agent/AgentDrawer.spec.ts`

- [ ] **Step 1: Write the failing test**

```ts
// AgentDrawer.spec.ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import AgentDrawer from './AgentDrawer.vue'
import ElementPlus from 'element-plus'

describe('AgentDrawer', () => {
  beforeEach(() => setActivePinia(createPinia()))
  it('does not render when closed', () => {
    const w = mount(AgentDrawer, { global: { plugins: [ElementPlus] } })
    expect(w.find('[data-testid="agent-drawer"]').exists()).toBe(false)
  })
  it('renders header/input/messages when open', async () => {
    const w = mount(AgentDrawer, { global: { plugins: [ElementPlus] } })
    await w.vm.$nextTick()
    // Set isOpen via store
    const { useAgentStore } = await import('@/stores/agent')
    useAgentStore().openDrawer()
    await w.vm.$nextTick()
    expect(w.find('[data-testid="agent-input"]').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

- [ ] **Step 3: Implement**

```vue
<!-- AgentDrawer.vue -->
<template>
  <el-drawer
    v-model="open"
    title="🤖 Agent"
    direction="rtl"
    size="480px"
    data-testid="agent-drawer"
  >
    <template #header>
      <div class="drawer-header">
        <span class="title">🤖 Agent</span>
        <div class="actions">
          <el-button text @click="showHistory = !showHistory">{{ showHistory ? '返回' : '历史' }}</el-button>
          <el-button text @click="close">✕</el-button>
        </div>
      </div>
    </template>
    <ConversationList v-if="showHistory" />
    <template v-else>
      <AgentMessageList :messages="store.messages" :streaming-text="store.streamingText" :is-thinking="store.isThinking" />
      <ToolConfirmDialog v-if="store.pendingConfirm" />
    </template>
    <template #footer>
      <AgentInput v-if="!showHistory" :disabled="store.isThinking" @send="onSend" />
    </template>
  </el-drawer>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useAgentStore } from '@/stores/agent'
import AgentMessageList from './AgentMessageList.vue'
import AgentInput from './AgentInput.vue'
import ConversationList from './ConversationList.vue'
import ToolConfirmDialog from './ToolConfirmDialog.vue'

const store = useAgentStore()
const showHistory = ref(false)
const open = computed({
  get: () => store.isOpen,
  set: (v: boolean) => v ? store.openDrawer() : store.closeDrawer(),
})
const close = () => store.closeDrawer()
const onSend = (text: string) => store.sendMessage(text)
</script>
```

```vue
<!-- AgentMessageList.vue -->
<template>
  <div class="messages">
    <div v-for="m in messages" :key="m.id" :class="['msg', m.role]">
      <div class="role">{{ roleLabel(m.role) }}</div>
      <div class="bubble" v-if="m.content">{{ m.content }}</div>
      <ToolCard v-if="m.role === 'tool_call' || m.role === 'tool_result'" :message="m" />
    </div>
    <div v-if="streamingText || isThinking" class="msg agent">
      <div class="role">Agent</div>
      <div class="bubble">{{ streamingText }}<span v-if="isThinking && !streamingText" class="typing"><span></span><span></span><span></span></span></div>
    </div>
  </div>
</template>
<script setup lang="ts">
import type { AgentMessage } from '@/types'
import ToolCard from './ToolCard.vue'
defineProps<{ messages: AgentMessage[]; streamingText: string; isThinking: boolean }>()
const roleLabel = (r: string) => r === 'user' ? '你' : 'Agent'
</script>
```

```vue
<!-- AgentInput.vue -->
<template>
  <div class="agent-input">
    <el-input
      v-model="text"
      type="textarea"
      :rows="2"
      placeholder="问点什么都行..."
      data-testid="agent-input"
      @keydown.enter.exact.prevent="send"
    />
    <div class="send-row">
      <span class="shortcuts"><code>/clear</code><code>/history</code><code>/new</code></span>
      <el-button type="primary" :disabled="disabled || !text" @click="send">发送 ➤</el-button>
    </div>
  </div>
</template>
<script setup lang="ts">
import { ref } from 'vue'
const props = defineProps<{ disabled?: boolean }>()
const emit = defineEmits<{ send: [text: string] }>()
const text = ref('')
const send = () => {
  if (!text.value) return
  emit('send', text.value)
  text.value = ''
}
</script>
```

- [ ] **Step 4: Run test to verify it passes**

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/agent/
git commit -m "feat(agent-ui): add AgentDrawer, MessageList, Input components"
```

---

### Task 19: ToolCard + ConfirmDialog + ConversationList

**Files:**
- Create: `frontend/src/components/agent/ToolCard.vue`
- Create: `frontend/src/components/agent/ToolConfirmDialog.vue`
- Create: `frontend/src/components/agent/ConversationList.vue`
- Create: `frontend/src/components/agent/ToolCard.spec.ts`

- [ ] **Step 1: Write the failing test (5 states)**

```ts
// ToolCard.spec.ts
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import ToolCard from './ToolCard.vue'

const baseMsg: any = {
  id: 'm1', conversation_id: 'c1', role: 'tool_call', content: '',
  tool_name: 'list_tasks', tool_args: '{}', tool_status: 'succeeded', created_at: '',
}

describe('ToolCard', () => {
  it('read succeeded shows green', () => {
    const w = mount(ToolCard, { props: { message: { ...baseMsg, tool_status: 'succeeded' } } })
    expect(w.classes()).toContain('read')
  })
  it('write pending shows yellow with confirm buttons', () => {
    const w = mount(ToolCard, { props: { message: { ...baseMsg, tool_name: 'create_task', tool_status: 'pending_confirmation' } } })
    expect(w.classes()).toContain('write')
    expect(w.find('[data-testid="confirm-btn"]').exists()).toBe(true)
  })
  it('dangerous pending shows red', () => {
    const w = mount(ToolCard, { props: { message: { ...baseMsg, tool_name: 'delete_task', tool_status: 'pending_confirmation' } } })
    expect(w.classes()).toContain('danger')
  })
  it('failed shows error text', () => {
    const w = mount(ToolCard, { props: { message: { ...baseMsg, tool_status: 'failed', tool_result: '{"error":"oops"}' } } })
    expect(w.text()).toContain('oops')
  })
})
```

- [ ] **Step 2-5**: Implement ToolCard/ConfirmDialog/ConversationList per mockup, run tests, commit.

**Commit message**: `feat(agent-ui): add ToolCard, ConfirmDialog, ConversationList`

Component skeletons:

```vue
<!-- ToolCard.vue -->
<template>
  <div :class="['tool-card', permClass, statusClass]" data-testid="tool-card">
    <div class="tool-name">
      <span class="icon">{{ statusIcon }}</span>
      <code>{{ message.tool_name }}</code>
      <el-tag size="small" :type="tagType">{{ statusLabel }}</el-tag>
    </div>
    <pre class="tool-args">{{ message.tool_args }}</pre>
    <div v-if="message.tool_result" class="tool-result">{{ message.tool_result }}</div>
    <div v-if="message.tool_status === 'pending_confirmation'" class="actions">
      <el-button size="small" type="primary" data-testid="confirm-btn" @click="onApprove">✓ 确认</el-button>
      <el-button size="small" @click="onReject">✕ 取消</el-button>
    </div>
  </div>
</template>
<script setup lang="ts">
import { computed } from 'vue'
import { useAgentStore } from '@/stores/agent'
import type { AgentMessage } from '@/types'
const props = defineProps<{ message: AgentMessage }>()
const store = useAgentStore()
const isDanger = computed(() => props.message.tool_name === 'delete_task')
const permClass = computed(() => {
  if (props.message.tool_status === 'failed') return 'failed'
  if (isDanger.value) return 'danger'
  // Determine read/write from tool name — could be derived from a lookup table
  const writeTools = ['create_task', 'update_task', 'start_pomodoro', 'stop_pomodoro', 'generate_schedule', 'save_worklog']
  return writeTools.includes(props.message.tool_name || '') ? 'write' : 'read'
})
const statusClass = computed(() => props.message.tool_status)
const statusLabel = computed(() => ({
  started: '执行中',
  pending_confirmation: '待确认',
  succeeded: '已执行',
  failed: '失败',
  rejected: '已取消',
}[props.message.tool_status || ''] || props.message.tool_status)
const tagType = computed(() => ({
  succeeded: 'success', pending_confirmation: 'warning', failed: 'danger', rejected: 'info', started: 'info',
}[props.message.tool_status || '']))
const statusIcon = computed(() => ({
  succeeded: '✓', pending_confirmation: '⏸', failed: '⚠', rejected: '✕', started: '▶',
}[props.message.tool_status || '']))
const onApprove = () => store.confirmToolCall(props.message.id, 'approve')
const onReject = () => store.confirmToolCall(props.message.id, 'reject')
</script>
```

---

### Task 20: App.vue Mount + Header Icon

**Files:**
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/router/index.ts` (if needed for header)

- [ ] **Step 1-5**: TDD-style. Mount `<AgentDrawer />` in App.vue's template, add a header icon button that calls `useAgentStore().openDrawer()`. On mount, call `useAgentStore().checkStatus()`.

**Commit message**: `feat(agent-ui): mount AgentDrawer globally with header trigger`

---

## M7 — Embedded Refactor + Finish

### Task 21: Embedded Button Refactor + Delete ai.ts + Settings.vue

**Files:**
- Modify: `frontend/src/components/tasks/TaskForm.vue`
- Modify: `frontend/src/components/tasks/TaskCard.vue`
- Modify: `frontend/src/views/Dashboard.vue`
- Modify: `frontend/src/views/Analytics.vue`
- Modify: `frontend/src/views/Settings.vue`
- Delete: `frontend/src/stores/ai.ts`

- [ ] **Step 1: Find all uses of useAiStore**

```bash
cd frontend && grep -rn "useAiStore\|stores/ai" src/
```

- [ ] **Step 2: Replace each occurrence**

For each hit:
- `ai.classifyTask(id)` → `agent.runTool('classify_task', { task_id: id })`
- `ai.classifyTaskByText(title, desc)` → `agent.runTool('classify_task', { title, description: desc })`
- `ai.getDailyInsights(date)` → `agent.runTool('get_daily_insights', { date })`
- `ai.checkStatus()` → `agent.checkStatus()` (already migrated; just change store)
- `ai.generateSchedule(...)` → keep existing call path (`/api/schedules/generate`) — backend reuses tool internally

In `Settings.vue`, replace `useAiStore().configured` with `useAgentStore().status.configured`.

- [ ] **Step 3: Run frontend type check + tests**

```bash
cd frontend && npx vue-tsc --noEmit && npx vitest run
```

Expected: no type errors; all tests pass.

- [ ] **Step 4: Delete stores/ai.ts**

```bash
rm frontend/src/stores/ai.ts
```

- [ ] **Step 5: Re-run + commit**

```bash
cd frontend && npx vue-tsc --noEmit && npx vitest run
git add -A frontend/
git commit -m "refactor(frontend): migrate all embedded AI buttons to agent.runTool; delete stores/ai.ts"
```

---

### Task 22: AnthropicClient ChatWithTools + CLIClient Degradation + Config Hot-Reload + E2E

**Files:**
- Modify: `backend/internal/ai/client.go` (Anthropic impl)
- Modify: `backend/internal/agent/service.go` (`Status()` method, hot-reload LLMClient)

- [ ] **Step 1: Write the failing test for Anthropic**

```go
// client_tools_test.go
func TestAnthropicClient_ChatWithTools(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assert request body has Anthropic-style tools
		resp := `{"content":[{"type":"text","text":"hello"},{"type":"tool_use","id":"tu1","name":"list_tasks","input":{"status":"todo"}}],"stop_reason":"tool_use"}`
		w.Write([]byte(resp))
	}))
	defer srv.Close()
	c := NewAnthropicClient(srv.URL, "k", "claude-sonnet-4-6")
	res, err := c.ChatWithTools(context.Background(),
		[]Message{{Role: "user", Content: "hi"}}, []ToolSpec{{Type: "function", Function: FunctionSpec{Name: "list_tasks"}}})
	if err != nil { t.Fatalf("err: %v", err) }
	if res.Content != "hello" { t.Fatalf("content = %q", res.Content) }
	if len(res.ToolCalls) != 1 || res.ToolCalls[0].Name != "list_tasks" {
		t.Fatalf("tool_calls = %+v", res.ToolCalls)
	}
}
```

- [ ] **Step 2: Run to verify fails**

- [ ] **Step 3: Implement AnthropicClient.ChatWithTools** with request/response mapping to/from Anthropic Messages API (`tools` array with `input_schema`, `tool_use` content blocks, `stop_reason=tool_use`).

- [ ] **Step 4: Implement Status() + config hot-reload**

```go
// In agent/service.go, add Status():
func (s *agentService) Status() map[string]any {
	settings, _ := s.SettingsRepo.GetAISettings() // injected in AgentDeps
	return map[string]any{
		"configured": settings.APIKey != "",
		"supports_function_calling": settings.Provider != "claude",
		"provider": settings.Provider,
	}
}
```

Add `SettingsRepo repository.SettingRepository` to `AgentDeps`. Construct LLMClient lazily per call (instead of once at startup):

```go
// Replace s.LLM.ChatWithTools(ctx, ...) with:
client := s.LLMFactory() // returns ai.LLMClient based on current settings
resp, err := client.ChatWithTools(ctx, ...)
```

Add `LLMFactory func() ai.LLMClient` to `AgentDeps` and remove the static `LLM` field. Update main.go to pass a factory that reads settings each call.

- [ ] **Step 5: E2E manual test + commit**

```bash
# Manual E2E checklist:
# 1. Start backend + frontend (make dev)
# 2. Open Agent drawer, send "list today's tasks" → see list_tasks succeeded card
# 3. Send "start a pomodoro" → see pending_confirmation → click approve → see started
# 4. Send "delete all completed tasks" → see danger pending → click second-confirm dialog → cancel
# 5. Switch provider to cli in Settings → Agent entry disabled with message
# 6. Close browser → restart backend → reopen → history preserved
```

```bash
git add backend/internal/ai/client.go backend/internal/agent/service.go backend/cmd/server/main.go
git commit -m "feat(agent): add AnthropicClient tool support + config hot-reload + Status()"
```

---

## Self-Review Checklist (run after completing all tasks)

### Spec Coverage

- [x] Architecture (Section 2) → Tasks 1-5
- [x] Backend components (Section 3) → Tasks 1-15
- [x] Frontend components (Section 4) → Tasks 16-21
- [x] Data model (Section 5.1) → Task 1
- [x] REST API (Section 5.2) → Tasks 13, 17
- [x] WebSocket events (Section 5.3) → Tasks 4, 17
- [x] 13 tools (Section 6) → Tasks 10-12
- [x] Permission model (Section 3.2) → Tasks 6-8
- [x] Error handling (Section 7.1) → Tasks 7, 9
- [x] Boundaries (Section 7.2) → Task 9
- [x] Degradation (Section 7.3) → Task 22
- [x] Test strategy (Section 8) → embedded in each task
- [x] AIService removal (ADR-1) → Task 15
- [x] WS new events (ADR-2) → Task 4
- [x] Trust Levels (ADR-3) → Tasks 6-8

### Notes for Implementer

- **Read existing service interfaces before mocking**: `service.TaskService`, `service.TimerService`, `service.ScheduleService`, `service.WorkLogService`, `service.AnalyticsService` — confirm exact method signatures before writing mocks. The plan uses representative signatures; the real ones may differ slightly.
- **`deref` helper**: define once in `tools/register.go` or as a shared utility.
- **Test helpers**: `setupTestDB`, `newInMemoryRepo` — place in shared `_test.go` files in respective packages.
- **TypeScript strict**: any unused imports will fail `vue-tsc`. Run `npx vue-tsc --noEmit` after each frontend task.
- **No commit `--no-verify`** — if hooks fail, fix the root cause.
