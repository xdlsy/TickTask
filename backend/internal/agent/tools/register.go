package tools

import "ticktask/internal/agent"

// Deps bundles the dependencies RegisterAll injects into the task tools.
// Tasks 11 and 12 will add more fields (timer/schedule/insight deps, work-log
// deps) and extend RegisterAll — for now only the task-tool deps are wired.
type Deps struct {
	Tasks TaskService
	LLM   LLMClient
}

// RegisterAll wires every tool the agent package exposes into the given
// registry. Tasks 11/12 will extend this with timer/schedule/insight/work-log
// tools (or call their own register helpers from here). For Task 10 we register
// only the 5 task tools.
func RegisterAll(reg agent.ToolRegistry, deps Deps) {
	reg.MustRegister(&ListTasksTool{Svc: deps.Tasks})
	reg.MustRegister(&CreateTaskTool{Svc: deps.Tasks})
	reg.MustRegister(&UpdateTaskTool{Svc: deps.Tasks})
	reg.MustRegister(&DeleteTaskTool{Svc: deps.Tasks})
	reg.MustRegister(&ClassifyTaskTool{Svc: deps.Tasks, LLM: deps.LLM})
}
