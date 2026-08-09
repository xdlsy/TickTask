package tools

import "ticktask/internal/agent"

// Deps bundles the dependencies RegisterAll injects into every tool. Fields
// that are *pointer-like* (interfaces) stay nil when callers don't have that
// dependency — RegisterAll still constructs the tool, but the tool will return
// an "X not configured" error at execute time if Svc is nil. (Task 10's
// ClassifyTaskTool demonstrates the same pattern with the LLM field.) WorkLog
// is the concrete *service.WorkLogService for the work-log tools added in
// Task 12; both StructureWorklogTool and SaveWorklogTool share it because the
// underlying service.WorkLogService implements both
// WorkLogStructureSvc.StructureBrainDump and WorkLogSaveSvc.SaveWorkLog.
type Deps struct {
	Tasks     TaskService
	Timer     TimerService
	Schedule  ScheduleService
	Analytics AnalyticsService
	LLM       LLMClient
	WorkLog   interface {
		WorkLogStructureSvc
		WorkLogSaveSvc
		WorkLogReadSvc
		WorkLogReportSvc
	}
}

// RegisterAll wires every tool the agent package exposes into the given
// registry. After Task 3 it registers 24 tools: 5 task + 5 timer + 7 schedule + 1 insight + 6 work-log.
func RegisterAll(reg agent.ToolRegistry, deps Deps) {
	// Task tools (Task 10)
	reg.MustRegister(&ListTasksTool{Svc: deps.Tasks})
	reg.MustRegister(&CreateTaskTool{Svc: deps.Tasks})
	reg.MustRegister(&UpdateTaskTool{Svc: deps.Tasks})
	reg.MustRegister(&DeleteTaskTool{Svc: deps.Tasks})
	reg.MustRegister(&ClassifyTaskTool{Svc: deps.Tasks, LLM: deps.LLM})

	// Timer tools (Task 11)
	reg.MustRegister(&StartPomodoroTool{Svc: deps.Timer})
	reg.MustRegister(&StopPomodoroTool{Svc: deps.Timer})
	reg.MustRegister(&GetTimerStatusTool{Svc: deps.Timer})
	reg.MustRegister(&ControlPomodoroTool{Svc: deps.Timer})
	reg.MustRegister(&GetPomodoroStatsTool{Svc: deps.Timer})

	// Schedule tools (Task 11)
	reg.MustRegister(&GenerateScheduleTool{Svc: deps.Schedule})
	reg.MustRegister(&ListScheduleTool{Svc: deps.Schedule})
	reg.MustRegister(&DeleteScheduleTool{Svc: deps.Schedule})
	reg.MustRegister(&UpdateScheduleTool{Svc: deps.Schedule})
	reg.MustRegister(&CreateScheduleTool{Svc: deps.Schedule})
	reg.MustRegister(&ReviseScheduleTool{Svc: deps.Schedule})
	reg.MustRegister(&ApplyScheduleRevisionTool{Svc: deps.Schedule})

	// Insight tools (Task 11)
	reg.MustRegister(&GetDailyInsightsTool{Svc: deps.Analytics})

	// Work-log tools (Task 12)
	reg.MustRegister(&StructureWorklogTool{Svc: deps.WorkLog})
	reg.MustRegister(&SaveWorklogTool{Svc: deps.WorkLog})
	reg.MustRegister(&GetWorklogTool{Svc: deps.WorkLog})
	reg.MustRegister(&ListWorklogsTool{Svc: deps.WorkLog})
	reg.MustRegister(&GenerateWorkReportTool{Svc: deps.WorkLog})
	reg.MustRegister(&GetWorkReportTool{Svc: deps.WorkLog})
}
