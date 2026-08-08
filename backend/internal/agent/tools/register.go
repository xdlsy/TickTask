package tools

import "ticktask/internal/agent"

// Deps bundles the dependencies RegisterAll injects into every tool. Task 12
// will add a WorkLog field. Fields that are *pointer-like* (interfaces) stay
// nil when callers don't have that dependency — RegisterAll still constructs
// the tool, but the tool will return an "X not configured" error at execute
// time if Svc is nil. (Task 10's ClassifyTaskTool demonstrates the same
// pattern with the LLM field.)
type Deps struct {
	Tasks     TaskService
	Timer     TimerService
	Schedule  ScheduleService
	Analytics AnalyticsService
	LLM       LLMClient
}

// RegisterAll wires every tool the agent package exposes into the given
// registry. Task 12 will extend this with work-log tools. For Task 11 we
// register 11 tools: 5 task tools + 3 timer tools + 2 schedule tools +
// 1 insight tool.
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

	// Schedule tools (Task 11)
	reg.MustRegister(&GenerateScheduleTool{Svc: deps.Schedule})
	reg.MustRegister(&ListScheduleTool{Svc: deps.Schedule})

	// Insight tools (Task 11)
	reg.MustRegister(&GetDailyInsightsTool{Svc: deps.Analytics})
}
