package agent

import (
	"encoding/json"
	"testing"

	"ticktask/internal/ai"
	"ticktask/internal/model"
)

// buildLLMMessages must reconstruct a multi-turn tool exchange so the model can
// continue after a tool call: the assistant message carries its tool_calls, and
// each tool_result references the originating tool_call_id. Without this the
// model sees an orphan tool_result and returns empty (Bug②).
func TestBuildLLMMessages_ReconstructsAssistantToolCallsAndLinkedToolResult(t *testing.T) {
	toolCallsJSON, _ := json.Marshal([]ai.ToolCall{
		{ID: "c1", Name: "list_schedule", Args: json.RawMessage(`{"from":"2026-08-09","to":"2026-08-09"}`)},
	})
	history := []*model.AgentMessage{
		{Role: "user", Content: "我一会有啥安排吗？"},
		{Role: "assistant", Content: "我来帮你查看", ToolCalls: strPtr(string(toolCallsJSON))},
		{Role: "tool_result", ToolName: strPtr("list_schedule"), ToolResult: strPtr(`{"count":3}`), ParentID: strPtr("c1")},
	}

	msgs := buildLLMMessages("sys", history)

	// [0]=system, [1]=user, [2]=assistant (with tool_calls), [3]=tool (linked)
	if len(msgs) != 4 {
		t.Fatalf("len(msgs) = %d, want 4: %+v", len(msgs), msgs)
	}
	asst := msgs[2]
	if asst.Role != "assistant" || asst.Content != "我来帮你查看" {
		t.Errorf("assistant msg = %+v", asst)
	}
	if len(asst.ToolCalls) != 1 || asst.ToolCalls[0].ID != "c1" || asst.ToolCalls[0].Name != "list_schedule" {
		t.Errorf("assistant tool_calls = %+v, want one with id=c1 name=list_schedule", asst.ToolCalls)
	}
	tool := msgs[3]
	if tool.Role != "tool" || tool.ToolCallID != "c1" {
		t.Errorf("tool msg = %+v, want role=tool tool_call_id=c1", tool)
	}
}

// An assistant message without tool_calls reconstructs to a plain content msg
// (regression guard for the normal no-tool turn).
func TestBuildLLMMessages_AssistantWithoutToolCalls(t *testing.T) {
	history := []*model.AgentMessage{
		{Role: "user", Content: "hi"},
		{Role: "assistant", Content: "hello"},
	}
	msgs := buildLLMMessages("sys", history)
	if len(msgs) != 3 || msgs[2].Role != "assistant" || len(msgs[2].ToolCalls) != 0 {
		t.Fatalf("msgs = %+v", msgs)
	}
}
