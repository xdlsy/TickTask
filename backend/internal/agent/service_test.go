package agent

import (
	"context"
	"testing"

	"ticktask/internal/ai"
	"ticktask/internal/model"
	"ticktask/internal/repository"

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
