package tools

import (
	"ticktask/internal/agent"
)

// newTestRegistry wires RegisterAll with all mocks pre-populated, returning
// the registry for assertion. Used by the cross-tool TestRegisterAll_* tests.
func newTestRegistry() agent.ToolRegistry {
	reg := agent.NewToolRegistry()
	RegisterAll(reg, Deps{
		Tasks:     &mockTaskSvc{},
		Timer:     &mockTimerSvc{},
		Schedule:  &mockScheduleSvc{},
		Analytics: &mockAnalyticsSvc{},
		LLM:       &mockLLM{},
	})
	return reg
}
