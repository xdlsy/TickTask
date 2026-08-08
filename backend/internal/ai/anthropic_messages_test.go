package ai

import (
	"encoding/json"
	"testing"
)

// convertMessagesToAnthropic must turn the unified (OpenAI-style) message list
// into Anthropic's Messages API shape: assistant tool_calls -> tool_use blocks,
// role=tool results -> user tool_result blocks. Without this the Anthropic
// protocol (used by anthropic + minimax) cannot consume tool results, so the
// model returns empty on every turn after a tool call (Bug 3).
func TestConvertMessagesToAnthropic_ToolUseAndToolResult(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "我一会有啥安排吗？"},
		{Role: "assistant", Content: "我来查一下", ToolCalls: []ToolCall{
			{ID: "call_1", Name: "list_schedule", Args: json.RawMessage(`{"from":"2026-08-09","to":"2026-08-09"}`)},
		}},
		{Role: "tool", ToolCallID: "call_1", Name: "list_schedule", Content: `{"count":3}`},
	}

	out := convertMessagesToAnthropic(msgs)

	if len(out) != 3 {
		t.Fatalf("len = %d, want 3: %+v", len(out), out)
	}
	if out[0]["role"] != "user" {
		t.Errorf("out[0] role = %v, want user", out[0]["role"])
	}
	// assistant -> content is a block list with one text + one tool_use
	if out[1]["role"] != "assistant" {
		t.Fatalf("out[1] role = %v", out[1]["role"])
	}
	blocks := out[1]["content"].([]map[string]any)
	if len(blocks) != 2 || blocks[0]["type"] != "text" || blocks[1]["type"] != "tool_use" {
		t.Fatalf("assistant blocks = %+v", blocks)
	}
	if blocks[1]["id"] != "call_1" || blocks[1]["name"] != "list_schedule" {
		t.Errorf("tool_use block = %+v", blocks[1])
	}
	if _, ok := blocks[1]["input"].(map[string]any); !ok {
		t.Errorf("tool_use input not an object: %T", blocks[1]["input"])
	}
	// tool result -> user message with one tool_result block referencing the id
	if out[2]["role"] != "user" {
		t.Fatalf("out[2] role = %v, want user (tool_result)", out[2]["role"])
	}
	rblocks := out[2]["content"].([]map[string]any)
	if len(rblocks) != 1 || rblocks[0]["type"] != "tool_result" || rblocks[0]["tool_use_id"] != "call_1" {
		t.Fatalf("tool_result blocks = %+v", rblocks)
	}
}

// Multiple tool results from one assistant turn must merge into a single user
// message (Anthropic requires alternating roles).
func TestConvertMessagesToAnthropic_MergesConsecutiveToolResults(t *testing.T) {
	msgs := []Message{
		{Role: "user", Content: "go"},
		{Role: "assistant", ToolCalls: []ToolCall{
			{ID: "a", Name: "t1", Args: json.RawMessage(`{}`)},
			{ID: "b", Name: "t2", Args: json.RawMessage(`{}`)},
		}},
		{Role: "tool", ToolCallID: "a", Content: "r1"},
		{Role: "tool", ToolCallID: "b", Content: "r2"},
	}
	out := convertMessagesToAnthropic(msgs)
	// user, assistant, user(merged) = 3
	if len(out) != 3 {
		t.Fatalf("len = %d, want 3 (tool results merged): %+v", len(out), out)
	}
	merged := out[2]["content"].([]map[string]any)
	if len(merged) != 2 {
		t.Fatalf("merged tool_result blocks = %d, want 2", len(merged))
	}
}

// System messages are stripped (Anthropic takes system separately).
func TestConvertMessagesToAnthropic_StripsSystem(t *testing.T) {
	out := convertMessagesToAnthropic([]Message{
		{Role: "system", Content: "sys"},
		{Role: "user", Content: "hi"},
	})
	if len(out) != 1 || out[0]["role"] != "user" {
		t.Fatalf("system not stripped: %+v", out)
	}
}
