package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"ticktask/internal/agent"
	"ticktask/internal/model"
	"ticktask/internal/service"
)

// TimerService is the subset of service.TimerService methods that timer tools
// need. The production service.TimerService struct satisfies this implicitly;
// tests substitute a mock that implements only these three methods. Defining
// the interface here keeps the tools package decoupled from the concrete
// TimerService struct (which has many more methods plus a goroutine state).
type TimerService interface {
	StartSession(req service.CreateSessionRequest) (*model.PomodoroSession, error)
	PauseSession(sessionID string) error
	GetActiveSession() (*model.PomodoroSession, error)
}

// =====================================================================
// start_pomodoro
// =====================================================================

// StartPomodoroTool starts a new pomodoro work session, optionally tied to a
// task. PermWrite: requires confirmation before mutating timer state.
type StartPomodoroTool struct {
	Svc TimerService
}

func (t *StartPomodoroTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "start_pomodoro",
		Function: agent.FunctionSpec{
			Name:        "start_pomodoro",
			Description: "Start a pomodoro work session. Optionally tie it to a task and override the default duration.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task_id":      map[string]any{"type": "string", "description": "Optional task ID to associate the session with"},
					"duration_min": map[string]any{"type": "integer", "description": "Optional duration in minutes (default: setting work duration, e.g. 25)"},
				},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *StartPomodoroTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		TaskID      *string `json:"task_id"`
		DurationMin *int    `json:"duration_min"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	req := service.CreateSessionRequest{
		Type: model.SessionWork,
	}
	if in.TaskID != nil && *in.TaskID != "" {
		tid := *in.TaskID
		req.TaskID = &tid
	}
	if in.DurationMin != nil && *in.DurationMin > 0 {
		// Convert minutes → seconds (CreateSessionRequest.Duration is in seconds).
		req.Duration = *in.DurationMin * 60
	}
	sess, err := t.Svc.StartSession(req)
	if err != nil {
		return nil, err
	}
	return sess, nil
}

func (t *StartPomodoroTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	// Don't run schema validation in preview — preview is allowed to surface
	// partial info even when args are incomplete.
	var in struct {
		TaskID      *string `json:"task_id"`
		DurationMin *int    `json:"duration_min"`
	}
	_ = json.Unmarshal(args, &in)
	plan := map[string]any{
		"action": "start_pomodoro",
	}
	if in.TaskID != nil {
		plan["task_id"] = *in.TaskID
	}
	if in.DurationMin != nil {
		plan["duration_min"] = *in.DurationMin
	}
	return plan, nil
}

// =====================================================================
// stop_pomodoro
// =====================================================================

// StopPomodoroTool pauses the currently active pomodoro session (state is kept
// so it can be resumed). If no session is active, it is a no-op. PermWrite:
// mutating timer state needs confirmation.
type StopPomodoroTool struct {
	Svc TimerService
}

func (t *StopPomodoroTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "stop_pomodoro",
		Function: agent.FunctionSpec{
			Name:        "stop_pomodoro",
			Description: "Pause the currently running pomodoro session. No-op if no session is active.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *StopPomodoroTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	active, err := t.Svc.GetActiveSession()
	if err != nil {
		return nil, err
	}
	if active == nil {
		return map[string]any{"already_stopped": true}, nil
	}
	if err := t.Svc.PauseSession(active.ID); err != nil {
		return nil, err
	}
	return map[string]any{
		"stopped":     true,
		"session_id":  active.ID,
		"task_id":     active.TaskID,
		"planned_sec": active.PlannedDuration,
	}, nil
}

func (t *StopPomodoroTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return map[string]any{"action": "stop_pomodoro"}, nil
}

// =====================================================================
// get_timer_status
// =====================================================================

// GetTimerStatusTool returns the current pomodoro session (or active=false if
// none is running). PermRead: auto-executed by the agent loop.
type GetTimerStatusTool struct {
	Svc TimerService
}

func (t *GetTimerStatusTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "get_timer_status",
		Function: agent.FunctionSpec{
			Name:        "get_timer_status",
			Description: "Return the currently running pomodoro session, or active=false if none.",
			Parameters: map[string]any{
				"type":       "object",
				"properties": map[string]any{},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *GetTimerStatusTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	active, err := t.Svc.GetActiveSession()
	if err != nil {
		return nil, err
	}
	if active == nil {
		return map[string]any{"active": false}, nil
	}
	return map[string]any{
		"active":        true,
		"session_id":    active.ID,
		"task_id":       active.TaskID,
		"type":          string(active.Type),
		"status":        string(active.Status),
		"planned_sec":   active.PlannedDuration,
		"start_time":    active.StartTime,
		"interruptions": active.Interruptions,
	}, nil
}

func (t *GetTimerStatusTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}
