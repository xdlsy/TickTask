package agent

import (
	"context"
	"errors"
	"testing"

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

// mockHub records broadcasts
type mockHub struct{ events []mockEvent }

type mockEvent struct {
	Type    string
	Payload any
}

func (h *mockHub) Broadcast(msg interface{}) {
	if m, ok := msg.(map[string]any); ok {
		if t, ok := m["type"].(string); ok {
			h.events = append(h.events, mockEvent{Type: t, Payload: m})
		}
	}
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
		LLM:      llm,
		Registry: NewToolRegistry(),
		Hub:      hub,
		System:   "you are nice",
	})
	conv, _ := repo.CreateConversation()
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

func TestAgentService_NoToolCall_RepoAppendError(t *testing.T) {
	llm := &mockLLM{responses: []ai.ToolResponse{
		{Content: "hi there", FinishReason: "stop"},
	}}
	hub := &mockHub{}
	baseRepo := newInMemoryRepo(t)
	repo := &failingOnAssistantRepo{AgentRepository: baseRepo}
	svc := NewAgentService(AgentDeps{
		Repo:     repo,
		LLM:      llm,
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

	// Expect exactly 2 broadcasts: the agent_message (sent before the failed
	// persistence) and a single agent_done with finish_reason="error".
	if len(hub.events) != 2 {
		t.Fatalf("events = %d, want 2: %+v", len(hub.events), hub.events)
	}
	if hub.events[0].Type != websocket.EventAgentMessage {
		t.Errorf("first event = %q, want %q", hub.events[0].Type, websocket.EventAgentMessage)
	}
	if hub.events[1].Type != websocket.EventAgentDone {
		t.Errorf("last event = %q, want %q", hub.events[1].Type, websocket.EventAgentDone)
	}
	payload, ok := hub.events[1].Payload.(map[string]any)
	if !ok {
		t.Fatalf("agent_done payload not map[string]any: %T", hub.events[1].Payload)
	}
	if got := payload["finish_reason"]; got != "error" {
		t.Errorf("finish_reason = %v, want %q", got, "error")
	}
}
