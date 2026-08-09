package tools

import (
	"ticktask/internal/agent"
	"ticktask/internal/model"
	"ticktask/internal/service"
)

// mockWorkLogSvc is a combined mock satisfying both WorkLogStructureSvc and
// WorkLogSaveSvc, used only by the cross-tool RegisterAll test. The per-tool
// tests in worklog_test.go keep separate single-method mocks for clarity.
type mockWorkLogSvc struct {
	mockWorkLogStructureSvc
	mockWorkLogSaveSvc
	mockWorkLogReadSvc
	mockWorkLogReportSvc
}

// Compile-time guards: ensure the combined mock satisfies both interfaces.
var (
	_ WorkLogStructureSvc = (*mockWorkLogSvc)(nil)
	_ WorkLogSaveSvc      = (*mockWorkLogSvc)(nil)
	_ WorkLogReadSvc      = (*mockWorkLogSvc)(nil)
	_ WorkLogReportSvc    = (*mockWorkLogSvc)(nil)
)

// newTestRegistry wires RegisterAll with all mocks pre-populated, returning
// the registry for assertion. Used by the cross-tool TestRegisterAll_* tests.
func newTestRegistry() agent.ToolRegistry {
	workLog := &mockWorkLogSvc{
		mockWorkLogStructureSvc: mockWorkLogStructureSvc{
			structureOut: &service.StructuredWorkLog{},
		},
		mockWorkLogSaveSvc: mockWorkLogSaveSvc{
			saveOut: &model.WorkLog{},
		},
	}
	reg := agent.NewToolRegistry()
	RegisterAll(reg, Deps{
		Tasks:     &mockTaskSvc{},
		Timer:     &mockTimerSvc{},
		Schedule:  &mockScheduleSvc{},
		Analytics: &mockAnalyticsSvc{},
		LLM:       &mockLLM{},
		WorkLog:   workLog,
	})
	return reg
}
