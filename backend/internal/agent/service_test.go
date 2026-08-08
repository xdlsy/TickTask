package agent

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"ticktask/internal/ai"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/internal/websocket"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
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

// mockHub records broadcasts. It is safe for concurrent use because PermWrite
// confirmation tests run SendMessage in a goroutine while the main test
// goroutine inspects events.
type mockHub struct {
	mu     sync.Mutex
	events []mockEvent
}

type mockEvent struct {
	Type    string
	Payload any
}

func (h *mockHub) Broadcast(msg interface{}) {
	if m, ok := msg.(map[string]any); ok {
		if t, ok := m["type"].(string); ok {
			h.mu.Lock()
			h.events = append(h.events, mockEvent{Type: t, Payload: m})
			h.mu.Unlock()
		}
	}
}

// snapshot returns a copy of the recorded events under the lock, safe to
// iterate without racing against concurrent Broadcast calls.
func (h *mockHub) snapshot() []mockEvent {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]mockEvent, len(h.events))
	copy(out, h.events)
	return out
}

// newInMemoryRepo builds an in-memory SQLite AgentRepository for service tests.
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

// failingOnAssistantRepo wraps a real AgentRepository and forces AppendMessage
// to fail once it has been called for an assistant message (the second
// AppendMessage in a normal no-tool-call turn: user first, assistant second).
type failingOnAssistantRepo struct {
	repository.AgentRepository
	appendCalls int
}

func (f *failingOnAssistantRepo) AppendMessage(convID, role, content string, toolName, toolArgs, toolResult, toolStatus *string) (string, error) {
	f.appendCalls++
	if role == "assistant" {
		return "", errAssistantAppendFailed
	}
	return f.AgentRepository.AppendMessage(convID, role, content, toolName, toolArgs, toolResult, toolStatus)
}

// errAssistantAppendFailed is the sentinel error returned by failingOnAssistantRepo.
var errAssistantAppendFailed = errors.New("simulated assistant append failure")

func TestAgentService_NoToolCall(t *testing.T) {
	llm := &mockLLM{responses: []ai.ToolResponse{
		{Content: "hi there", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	repo := newInMemoryRepo(t)
	svc := NewAgentService(AgentDeps{
		Repo:     repo,
		LLMFactory: func() ai.LLMClient { return llm },
		Registry: NewToolRegistry(),
		Hub:      hub,
		System:   "you are nice",
	})
	conv, _ := repo.CreateConversation()
	if err := svc.SendMessage(context.Background(), conv.ID, "hello"); err != nil {
		t.Fatalf("send: %v", err)
	}
	// Expect: 1 agent_message + 1 agent_done
	events := hub.snapshot()
	if len(events) != 2 {
		t.Fatalf("events = %d, want 2: %+v", len(events), events)
	}
	if events[0].Type != "agent_message" {
		t.Errorf("first event = %q", events[0].Type)
	}
	// agent_message payload must include a non-empty message_id alongside
	// conversation_id and delta_text (spec contract for the WS event).
	msgPayload, ok := events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("agent_message payload not map[string]any: %T", events[0].Payload)
	}
	if msgID, _ := msgPayload["message_id"].(string); msgID == "" {
		t.Errorf("agent_message.message_id = %q, want non-empty", msgID)
	}
	if got := msgPayload["delta_text"]; got != "hi there" {
		t.Errorf("agent_message.delta_text = %v, want %q", got, "hi there")
	}
	if got := msgPayload["conversation_id"]; got != conv.ID {
		t.Errorf("agent_message.conversation_id = %v, want %q", got, conv.ID)
	}
	if events[1].Type != "agent_done" {
		t.Errorf("last event = %q", events[1].Type)
	}
}

func TestAgentService_NoToolCall_RepoAppendError(t *testing.T) {
	llm := &mockLLM{responses: []ai.ToolResponse{
		{Content: "hi there", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	baseRepo := newInMemoryRepo(t)
	repo := &failingOnAssistantRepo{AgentRepository: baseRepo}
	svc := NewAgentService(AgentDeps{
		Repo:     repo,
		LLMFactory: func() ai.LLMClient { return llm },
		Registry: NewToolRegistry(),
		Hub:      hub,
		System:   "you are nice",
	})
	conv, _ := baseRepo.CreateConversation()

	// SendMessage should propagate the assistant-append error.
	err := svc.SendMessage(context.Background(), conv.ID, "hello")
	if !errors.Is(err, errAssistantAppendFailed) {
		t.Fatalf("expected errAssistantAppendFailed, got %v", err)
	}

	// The repo should have seen exactly two AppendMessage calls:
	// the user message (succeeds) and the assistant message (fails).
	if repo.appendCalls != 2 {
		t.Fatalf("appendCalls = %d, want 2", repo.appendCalls)
	}

	// With persist-first ordering (AppendMessage runs before the
	// agent_message broadcast), a failed assistant-append short-circuits
	// BEFORE any agent_message is emitted. The only expected broadcast is a
	// single agent_done with finish_reason="error".
	events := hub.snapshot()
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1: %+v", len(events), events)
	}
	if events[0].Type != websocket.EventAgentDone {
		t.Errorf("only event = %q, want %q", events[0].Type, websocket.EventAgentDone)
	}
	payload, ok := events[0].Payload.(map[string]any)
	if !ok {
		t.Fatalf("agent_done payload not map[string]any: %T", events[0].Payload)
	}
	if got := payload["finish_reason"]; got != "error" {
		t.Errorf("finish_reason = %v, want %q", got, "error")
	}
}

func TestAgentService_PermReadToolAutoExecutes(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&fakeTool{name: "list_tasks", perm: PermRead})
	llm := &mockLLM{responses: []ai.ToolResponse{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "list_tasks", Args: json.RawMessage(`{"status":"todo"}`)}}, FinishReason: "tool_calls"},
		{Content: "all done", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	repo := newInMemoryRepo(t)
	svc := NewAgentService(AgentDeps{Repo: repo, LLMFactory: func() ai.LLMClient { return llm }, Registry: reg, Hub: hub})
	conv, _ := repo.CreateConversation()
	if err := svc.SendMessage(context.Background(), conv.ID, "go"); err != nil {
		t.Fatal(err)
	}
	// Expect at least: 1 agent_tool(succeeded) + 1 agent_message + 1 agent_done
	found := false
	for _, e := range hub.snapshot() {
		if e.Type == "agent_tool" {
			p := e.Payload.(map[string]any)
			if p["status"] == "succeeded" && p["tool_name"] == "list_tasks" {
				found = true
			}
		}
	}
	if !found {
		t.Fatalf("no succeeded tool event in %+v", hub.snapshot())
	}
}

func TestAgentService_PermWriteTool_Approve(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&fakeTool{name: "create_task", perm: PermWrite})
	var llmResponses = []ai.ToolResponse{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "create_task", Args: json.RawMessage(`{"title":"x"}`)}}, FinishReason: "tool_calls"},
	}
	llm := &mockLLM{responses: llmResponses}
	hub := &mockHub{}
	repo := newInMemoryRepo(t)
	svc := NewAgentService(AgentDeps{Repo: repo, LLMFactory: func() ai.LLMClient { return llm }, Registry: reg, Hub: hub})
	conv, _ := repo.CreateConversation()

	// Run SendMessage in goroutine; it should block on pending confirmation
	done := make(chan error, 1)
	go func() { done <- svc.SendMessage(context.Background(), conv.ID, "go") }()

	// Wait for pending_confirmation event
	var msgID string
	for i := 0; i < 50; i++ {
		for _, e := range hub.snapshot() {
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
	for _, e := range hub.snapshot() {
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
	repo := newInMemoryRepo(t)
	svc := NewAgentService(AgentDeps{Repo: repo, LLMFactory: func() ai.LLMClient { return llm }, Registry: reg, Hub: hub})
	conv, _ := repo.CreateConversation()
	done := make(chan error, 1)
	go func() { done <- svc.SendMessage(context.Background(), conv.ID, "go") }()
	var msgID string
	for i := 0; i < 50; i++ {
		for _, e := range hub.snapshot() {
			if e.Type == "agent_tool" {
				if p, ok := e.Payload.(map[string]any); ok && p["status"] == "pending_confirmation" {
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
	if err := svc.Confirm(context.Background(), msgID, "reject"); err != nil {
		t.Fatalf("confirm: %v", err)
	}
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("never returned")
	}
	found := false
	for _, e := range hub.snapshot() {
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

func TestAgentService_PermDangerousTool_SecondConfirm(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&fakeTool{name: "delete_task", perm: PermDangerous})
	llm := &mockLLM{responses: []ai.ToolResponse{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "delete_task", Args: json.RawMessage(`{"task_id":"12"}`)}}, FinishReason: "tool_calls"},
		{Content: "deleted", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	repo := newInMemoryRepo(t)
	svc := NewAgentService(AgentDeps{Repo: repo, LLMFactory: func() ai.LLMClient { return llm }, Registry: reg, Hub: hub})
	conv, _ := repo.CreateConversation()
	done := make(chan error, 1)
	go func() { done <- svc.SendMessage(context.Background(), conv.ID, "go") }()
	var msgID string
	for i := 0; i < 50; i++ {
		for _, e := range hub.snapshot() {
			if e.Type == "agent_tool" {
				if p, ok := e.Payload.(map[string]any); ok && p["status"] == "pending_confirmation" {
					if pn, _ := p["tool_name"].(string); pn == "delete_task" {
						msgID, _ = p["message_id"].(string)
					}
				}
			}
		}
		if msgID != "" {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if msgID == "" {
		t.Fatal("no pending_confirmation for dangerous tool")
	}
	// preview should be present
	var previewSeen bool
	for _, e := range hub.snapshot() {
		if e.Type == "agent_tool" {
			if p, ok := e.Payload.(map[string]any); ok && p["message_id"] == msgID {
				if _, ok := p["preview"]; ok {
					previewSeen = true
				}
			}
		}
	}
	if !previewSeen {
		t.Fatal("no preview field in dangerous pending_confirmation")
	}
	svc.Confirm(context.Background(), msgID, "approve")
	<-done
}

// typedFakeTool declares a real schema so ValidateArgs has something to check.
// (fakeTool declares only {"type":"object"} with no properties, which is treated
// leniently to keep that test-double usable as a no-arg tool.)
type typedFakeTool struct{ name string }

func (t *typedFakeTool) Schema() ToolSchema {
	return ToolSchema{
		Name: t.name,
		Function: FunctionSpec{
			Description: "typed fake",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status": map[string]any{"type": "string"},
				},
			},
		},
		Permission: PermRead,
	}
}
func (t *typedFakeTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	return "ok:" + t.name, nil
}
func (t *typedFakeTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return "preview:" + t.name, nil
}

func repeat(r ai.ToolResponse, n int) []ai.ToolResponse {
	out := make([]ai.ToolResponse, n)
	for i := range out {
		out[i] = r
	}
	return out
}

// TestAgentService_MaxToolCalls verifies the runTurn outer loop terminates at
// MaxToolCallsPerTurn even when the LLM keeps returning tool calls forever.
func TestAgentService_MaxToolCalls(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&fakeTool{name: "list_tasks", perm: PermRead})
	// LLM always returns the same tool_call
	llm := &mockLLM{responses: repeat(ai.ToolResponse{
		ToolCalls:    []ai.ToolCall{{ID: "c", Name: "list_tasks", Args: json.RawMessage(`{}`)}},
		FinishReason: "tool_calls",
	}, MaxToolCallsPerTurn+5)}
	hub := &mockHub{}
	repo := newInMemoryRepo(t)
	svc := NewAgentService(AgentDeps{Repo: repo, LLMFactory: func() ai.LLMClient { return llm }, Registry: reg, Hub: hub})
	conv, _ := repo.CreateConversation()
	if err := svc.SendMessage(context.Background(), conv.ID, "loop"); err != nil {
		t.Fatal(err)
	}
	var doneReason string
	for _, e := range hub.snapshot() {
		if e.Type == "agent_done" {
			doneReason, _ = e.Payload.(map[string]any)["finish_reason"].(string)
		}
	}
	if doneReason != "max_tools" {
		t.Fatalf("finish = %q, want max_tools", doneReason)
	}
}

// TestAgentService_SchemaValidationError verifies that a tool call whose args
// fail JSON-schema validation is short-circuited: a failed agent_tool event is
// broadcast and the turn continues without executing the tool.
func TestAgentService_SchemaValidationError(t *testing.T) {
	reg := NewToolRegistry()
	reg.Register(&typedFakeTool{name: "list_tasks"})
	// Tool call with bad args type — status is declared as string but we pass a number.
	llm := &mockLLM{responses: []ai.ToolResponse{
		{ToolCalls: []ai.ToolCall{{ID: "c1", Name: "list_tasks", Args: json.RawMessage(`{"status":123}`)}}, FinishReason: "tool_calls"},
		{Content: "ok", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	repo := newInMemoryRepo(t)
	svc := NewAgentService(AgentDeps{Repo: repo, LLMFactory: func() ai.LLMClient { return llm }, Registry: reg, Hub: hub})
	conv, _ := repo.CreateConversation()
	if err := svc.SendMessage(context.Background(), conv.ID, "bad args"); err != nil {
		t.Fatal(err)
	}
	// Expect agent_tool failed with a "schema: ..." error message.
	var found bool
	for _, e := range hub.snapshot() {
		if e.Type == "agent_tool" {
			if p, ok := e.Payload.(map[string]any); ok && p["status"] == "failed" {
				if msg, _ := p["error"].(string); strings.Contains(msg, "schema") {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("no schema failure event")
	}
}

// fakeSettingsRepo is a minimal SettingRepository stub for Status() tests.
// It implements only the methods Status() touches; the rest panic so any
// accidental call surfaces loudly. The interface is small enough that filling
// in the unused methods with nil/error returns would just be noise.
type fakeSettingsRepo struct {
	settings *model.AISettings
}

func (f *fakeSettingsRepo) Get(string) (*model.Setting, error)              { return nil, repository.ErrNotFound }
func (f *fakeSettingsRepo) Set(string, string) error                        { return nil }
func (f *fakeSettingsRepo) GetPomodoroSettings() (*model.PomodoroSettings, error) {
	return model.DefaultPomodoroSettings(), nil
}
func (f *fakeSettingsRepo) UpdatePomodoroSettings(*model.PomodoroSettings) error { return nil }
func (f *fakeSettingsRepo) GetAISettings() (*model.AISettings, error)           { return f.settings, nil }
func (f *fakeSettingsRepo) UpdateAISettings(*model.AISettings) error            { return nil }
func (f *fakeSettingsRepo) MigrateLegacyAPIKey() error                          { return nil }

// TestAgentService_Status verifies the Task 22 real Status() implementation
// against the four meaningful shape cases:
//   - nil SettingsRepo → conservative all-false zero (handler still serves)
//   - openai + key     → Configured + SupportsFunctionCalling
//   - cli / claude     → Configured but NOT SupportsFunctionCalling
//   - openai no key    → not Configured, but SupportsFunctionCalling is true
//     (provider capability is independent of credential presence)
func TestAgentService_Status(t *testing.T) {
	llm := &mockLLM{}
	factory := func() ai.LLMClient { return llm }

	// Case 1: nil SettingsRepo returns zero value.
	svcNil := NewAgentService(AgentDeps{
		Repo:        newInMemoryRepo(t),
		LLMFactory:  factory,
		Registry:    NewToolRegistry(),
		Hub:         &mockHub{},
	}).(*agentService)
	if got := svcNil.Status(); got.Configured || got.SupportsFunctionCalling || got.Provider != "" {
		t.Fatalf("nil SettingsRepo = %+v, want zero", got)
	}

	cases := []struct {
		name              string
		settings          *model.AISettings
		wantConfigured    bool
		wantSupportsTools bool
		wantProvider      string
	}{
		{
			name:              "openai with key",
			settings:          &model.AISettings{Provider: "openai", APIKey: "sk-x"},
			wantConfigured:    true,
			wantSupportsTools: true,
			wantProvider:      "openai",
		},
		{
			name:              "anthropic with key",
			settings:          &model.AISettings{Provider: "anthropic", APIKey: "sk-x"},
			wantConfigured:    true,
			wantSupportsTools: true,
			wantProvider:      "anthropic",
		},
		{
			name:              "cli provider (subprocess, no tools)",
			settings:          &model.AISettings{Provider: "cli"},
			wantConfigured:    true,
			wantSupportsTools: false,
			wantProvider:      "cli",
		},
		{
			name:              "claude provider alias (subprocess)",
			settings:          &model.AISettings{Provider: "claude"},
			wantConfigured:    true,
			wantSupportsTools: false,
			wantProvider:      "claude",
		},
		{
			name:              "openai without key",
			settings:          &model.AISettings{Provider: "openai", APIKey: ""},
			wantConfigured:    false,
			wantSupportsTools: true,
			wantProvider:      "openai",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewAgentService(AgentDeps{
				Repo:         newInMemoryRepo(t),
				LLMFactory:   factory,
				SettingsRepo: &fakeSettingsRepo{settings: tc.settings},
				Registry:     NewToolRegistry(),
				Hub:          &mockHub{},
			}).(*agentService)
			got := svc.Status()
			if got.Configured != tc.wantConfigured {
				t.Errorf("Configured = %v, want %v", got.Configured, tc.wantConfigured)
			}
			if got.SupportsFunctionCalling != tc.wantSupportsTools {
				t.Errorf("SupportsFunctionCalling = %v, want %v", got.SupportsFunctionCalling, tc.wantSupportsTools)
			}
			if got.Provider != tc.wantProvider {
				t.Errorf("Provider = %q, want %q", got.Provider, tc.wantProvider)
			}
		})
	}
}

// TestAgentService_TestConnection_DBPath exercises the settings==nil branch:
// the service falls back to its LLMFactory and reads provider/model from the
// SettingsRepo. With a working mock LLM, OK should be true and the provider
// echoed from the saved settings.
func TestAgentService_TestConnection_DBPath(t *testing.T) {
	llm := &mockLLM{}
	factoryCalls := 0
	factory := func() ai.LLMClient {
		factoryCalls++
		return llm
	}
	savedSettings := &model.AISettings{Provider: "openai", APIKey: "sk-saved", Model: "gpt-4o-mini"}
	svc := NewAgentService(AgentDeps{
		Repo:         newInMemoryRepo(t),
		LLMFactory:   factory,
		SettingsRepo: &fakeSettingsRepo{settings: savedSettings},
		Registry:     NewToolRegistry(),
		Hub:          &mockHub{},
	}).(*agentService)

	result := svc.TestConnection(context.Background(), nil)
	if !result.OK {
		t.Fatalf("OK = false, want true; err=%s", result.Error)
	}
	if result.Provider != "openai" {
		t.Errorf("Provider = %q, want openai (from DB settings)", result.Provider)
	}
	if result.Model != "gpt-4o-mini" {
		t.Errorf("Model = %q, want gpt-4o-mini (from DB settings)", result.Model)
	}
	if factoryCalls != 1 {
		t.Errorf("LLMFactory called %d times, want 1 (DB path uses factory)", factoryCalls)
	}
}

// TestAgentService_TestConnection_DBPath_NilClient verifies the nil-client
// guard on the DB path: when LLMFactory returns nil, TestConnection short-
// circuits with a descriptive error rather than panicking on a nil dereference.
func TestAgentService_TestConnection_DBPath_NilClient(t *testing.T) {
	svc := NewAgentService(AgentDeps{
		Repo:         newInMemoryRepo(t),
		LLMFactory:   func() ai.LLMClient { return nil },
		SettingsRepo: &fakeSettingsRepo{settings: &model.AISettings{Provider: "openai"}},
		Registry:     NewToolRegistry(),
		Hub:          &mockHub{},
	}).(*agentService)

	result := svc.TestConnection(context.Background(), nil)
	if result.OK {
		t.Fatal("OK = true, want false")
	}
	if result.Error == "" {
		t.Fatal("Error empty, want descriptive 'AI not configured' message")
	}
}

// TestAgentService_TestConnection_WithTempSettings exercises the settings!=nil
// branch. Two observable contract points:
//
//  1. Provider/Model on the returned TestResult come from the temp settings,
//     NOT from the DB-stored settings (we set them to different values to make
//     the assertion meaningful).
//  2. The temp path bypasses LLMFactory entirely — verified by giving the
//     factory a panic-on-call sentinel and observing that the test does not
//     panic. When temp settings lack an API key for an HTTP provider,
//     ai.NewClientFromSettings returns nil, which the service surfaces as the
//     "AI 未配置" error rather than calling the factory.
//
// This is the closest behavioral proxy for "temp key was forwarded, DB key was
// not used" available without making NewClientFromSettings injectable. The
// finer-grained assertion that the underlying HTTP client receives the temp
// api_key lives implicitly in ai.NewClientFromSettings's own contract (which
// copies settings.APIKey into the concrete client struct).
func TestAgentService_TestConnection_WithTempSettings(t *testing.T) {
	// Sentinel factory that panics if invoked: the temp path must NOT call it.
	// (If the service ignores the temp settings and falls back to LLMFactory,
	// this test fails loudly instead of silently returning a passing result.)
	factory := func() ai.LLMClient {
		t.Fatal("LLMFactory must not be called when temp settings are provided")
		return nil
	}
	// DB-stored settings differ from temp settings on every field, so any
	// leakage from DB into the result is observable.
	savedSettings := &model.AISettings{Provider: "anthropic", APIKey: "sk-from-db", Model: "claude-from-db"}
	svc := NewAgentService(AgentDeps{
		Repo:         newInMemoryRepo(t),
		LLMFactory:   factory,
		SettingsRepo: &fakeSettingsRepo{settings: savedSettings},
		Registry:     NewToolRegistry(),
		Hub:          &mockHub{},
	}).(*agentService)

	// Temp settings with no API key on an HTTP provider: NewClientFromSettings
	// returns nil → service must surface the "not configured" error WITHOUT
	// calling LLMFactory.
	tempSettings := &model.AISettings{Provider: "openai", APIKey: "", Model: "gpt-4o-mini-temp"}
	result := svc.TestConnection(context.Background(), tempSettings)
	if result.OK {
		t.Fatal("OK = true, want false (no API key on openai provider → nil client)")
	}
	if result.Provider != "openai" {
		t.Errorf("Provider = %q, want openai (from temp settings, not DB)", result.Provider)
	}
	if result.Model != "gpt-4o-mini-temp" {
		t.Errorf("Model = %q, want gpt-4o-mini-temp (from temp settings, not DB)", result.Model)
	}
}
