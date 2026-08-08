package agent

import (
	"context"
	"strings"
	"testing"
	"time"

	"ticktask/internal/ai"
)

// captureMsgsLLM records the messages handed to ChatWithTools so tests can
// assert on the system prompt. It drains a queued response list like mockLLM.
type captureMsgsLLM struct {
	responses []ai.ToolResponse
	lastMsgs  []ai.Message
}

func (m *captureMsgsLLM) ChatCompletion(context.Context, string) (string, error) {
	return "", nil
}

func (m *captureMsgsLLM) ChatWithTools(_ context.Context, msgs []ai.Message, _ []ai.ToolSpec) (ai.ToolResponse, error) {
	m.lastMsgs = msgs
	if len(m.responses) > 0 {
		r := m.responses[0]
		m.responses = m.responses[1:]
		return r, nil
	}
	return ai.ToolResponse{FinishReason: "stop"}, nil
}

// Guards the date-injection fix: the model must be told today's date so it can
// route date-relative queries (e.g. "today's schedule") to the right tool args.
func TestAgentService_SystemPromptIncludesTodayDate(t *testing.T) {
	reg := NewToolRegistry()
	llm := &captureMsgsLLM{responses: []ai.ToolResponse{{Content: "ok", FinishReason: "stop"}}}
	hub := &mockHub{}
	repo := newInMemoryRepo(t)
	svc := NewAgentService(AgentDeps{
		Repo: repo, LLMFactory: func() ai.LLMClient { return llm }, Registry: reg, Hub: hub,
	})
	conv, _ := repo.CreateConversation()

	if err := svc.SendMessage(context.Background(), conv.ID, "我一会有啥安排吗？"); err != nil {
		t.Fatal(err)
	}
	if len(llm.lastMsgs) == 0 || llm.lastMsgs[0].Role != "system" {
		t.Fatalf("no system message captured: %+v", llm.lastMsgs)
	}
	today := time.Now().Format("2006-01-02")
	if !strings.Contains(llm.lastMsgs[0].Content, today) {
		t.Errorf("system prompt = %q; want it to contain today's date %s", llm.lastMsgs[0].Content, today)
	}
}
