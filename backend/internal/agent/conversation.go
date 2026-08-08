package agent

import (
	"encoding/json"

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
			am := ai.Message{Role: "assistant", Content: m.Content}
			// Reconstruct tool_calls the assistant requested this turn so the
			// model can pair the following tool_result messages with their
			// originating call ids (OpenAI/Anthropic tool protocol).
			if m.ToolCalls != nil && *m.ToolCalls != "" {
				var tcs []ai.ToolCall
				if err := json.Unmarshal([]byte(*m.ToolCalls), &tcs); err == nil {
					am.ToolCalls = tcs
				}
			}
			msgs = append(msgs, am)
		case "tool_result", "tool_call":
			// tool_result = PermRead auto-executed results; tool_call = PermWrite/
			// Dangerous results persisted after user confirmation. Both reconstruct
			// as a tool message whose tool_call_id is the originating call's id
			// (ParentID), pairing each result with the assistant's preceding
			// tool_calls (Bug 4: tool_call was skipped, orphaning write-tool calls
			// and breaking every turn after a create/update/delete/start).
			if m.ToolName != nil {
				msgs = append(msgs, ai.Message{
					Role: "tool", Name: *m.ToolName, ToolCallID: deref(m.ParentID), Content: derep(m.ToolResult),
				})
			}
		}
	}
	return msgs
}

func deref(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derep(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
