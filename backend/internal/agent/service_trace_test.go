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
