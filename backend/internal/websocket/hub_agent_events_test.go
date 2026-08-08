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
