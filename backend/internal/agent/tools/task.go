package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ticktask/internal/agent"
	"ticktask/internal/model"
	"ticktask/internal/service"
)

// TaskService is the subset of service.TaskService methods that task tools
// need. The production service.TaskService struct satisfies this implicitly;
// tests substitute a mock that implements only these five methods. Defining
// the interface here keeps the tools package decoupled from service.TaskService's
// concrete struct (which has many more methods used by other callers).
type TaskService interface {
	CreateTask(req service.CreateTaskRequest) (*model.Task, error)
	UpdateTask(id string, req service.UpdateTaskRequest) error
	DeleteTask(id string) error
	GetTask(id string) (*model.Task, error)
	GetAllTasks() ([]model.Task, error)
}

// priorityMapping converts the agent-facing "priority" enum (used by the LLM)
// into the IsImportant/IsUrgent bools the underlying Task model uses. The four
// values are the canonical Eisenhower-matrix strings.
var priorityMapping = map[string]struct {
	important bool
	urgent    bool
}{
	"important_urgent":         {true, true},
	"important_not_urgent":     {true, false},
	"not_important_urgent":     {false, true},
	"not_important_not_urgent": {false, false},
}

// quadrantOf computes the model.Quadrant from the importance/urgency flags.
// Mirrors service.CalculateQuadrant (which is bound to a *TaskService struct)
// so the tools package can stay decoupled from the concrete service type.
func quadrantOf(important, urgent bool) model.Quadrant {
	switch {
	case important && urgent:
		return model.Quadrant1
	case important && !urgent:
		return model.Quadrant2
	case !important && urgent:
		return model.Quadrant3
	default:
		return model.Quadrant4
	}
}

// =====================================================================
// list_tasks
// =====================================================================

// ListTasksTool returns tasks optionally filtered by status, quadrant, or due
// date. It is PermRead so the agent loop auto-executes it.
type ListTasksTool struct {
	Svc TaskService
}

func (t *ListTasksTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "list_tasks",
		Function: agent.FunctionSpec{
			Name:        "list_tasks",
			Description: "List tasks with optional filters (status, quadrant, due date).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"status":   map[string]any{"type": "string", "description": "todo|in_progress|completed|cancelled"},
					"due":      map[string]any{"type": "string", "description": "YYYY-MM-DD"},
					"quadrant": map[string]any{"type": "integer", "description": "1-4 (Eisenhower matrix)"},
				},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *ListTasksTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Status   *string `json:"status"`
		Due      *string `json:"due"`
		Quadrant *int    `json:"quadrant"`
	}
	// Best-effort decode; ValidateArgs already verified the JSON is sound.
	_ = json.Unmarshal(args, &in)

	all, err := t.Svc.GetAllTasks()
	if err != nil {
		return nil, err
	}

	out := make([]model.Task, 0, len(all))
	for _, task := range all {
		if in.Status != nil && string(task.Status) != *in.Status {
			continue
		}
		if in.Quadrant != nil && int(task.Quadrant) != *in.Quadrant {
			continue
		}
		if in.Due != nil && *in.Due != "" {
			if !matchesDueDate(task.DueDate, *in.Due) {
				continue
			}
		}
		out = append(out, task)
	}
	return map[string]any{"tasks": out, "count": len(out)}, nil
}

func (t *ListTasksTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}

// matchesDueDate reports whether the optional due time falls on the given
// calendar date (YYYY-MM-DD). A nil DueDate never matches.
func matchesDueDate(due *time.Time, yyyyMMdd string) bool {
	if due == nil {
		return false
	}
	return due.Format("2006-01-02") == yyyyMMdd
}

// =====================================================================
// create_task
// =====================================================================

// CreateTaskTool creates a new task. PermWrite: requires confirmation.
type CreateTaskTool struct {
	Svc TaskService
}

func (t *CreateTaskTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "create_task",
		Function: agent.FunctionSpec{
			Name:        "create_task",
			Description: "Create a new task. Returns the created task with its ID.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":       map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"priority":    map[string]any{"type": "string", "description": "important_urgent|important_not_urgent|not_important_urgent|not_important_not_urgent"},
					"due":         map[string]any{"type": "string", "description": "YYYY-MM-DD"},
				},
				"required": []any{"title"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *CreateTaskTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
		Priority    *string `json:"priority"`
		Due         *string `json:"due"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.Title == "" {
		return nil, fmt.Errorf("schema: missing required field title")
	}
	req := service.CreateTaskRequest{
		Title:       in.Title,
		Description: derefStr(in.Description),
	}
	if in.Priority != nil {
		if m, ok := priorityMapping[*in.Priority]; ok {
			req.IsImportant = m.important
			req.IsUrgent = m.urgent
			req.Quadrant = quadrantOf(m.important, m.urgent)
		}
		// Unknown priority strings are silently ignored — the agent can still
		// classify later via classify_task.
	}
	if in.Due != nil && *in.Due != "" {
		t, err := time.Parse("2006-01-02", *in.Due)
		if err != nil {
			return nil, fmt.Errorf("invalid due date %q (expected YYYY-MM-DD): %w", *in.Due, err)
		}
		req.DueDate = &t
	}
	return t.Svc.CreateTask(req)
}

func (t *CreateTaskTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	// Don't run schema validation in preview — preview is allowed to surface
	// partial info even when args are incomplete. We still parse the title for
	// the human-readable summary.
	var in struct {
		Title    string  `json:"title"`
		Priority *string `json:"priority"`
		Due      *string `json:"due"`
	}
	_ = json.Unmarshal(args, &in)
	plan := map[string]any{
		"action": "create",
		"title":  in.Title,
	}
	if in.Priority != nil {
		plan["priority"] = *in.Priority
	}
	if in.Due != nil {
		plan["due"] = *in.Due
	}
	return plan, nil
}

// =====================================================================
// update_task
// =====================================================================

// UpdateTaskTool applies a partial update to a task. PermWrite: requires
// confirmation. Only fields the agent explicitly sets are touched.
type UpdateTaskTool struct {
	Svc TaskService
}

func (t *UpdateTaskTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "update_task",
		Function: agent.FunctionSpec{
			Name:        "update_task",
			Description: "Partially update a task. Only provided fields are changed.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task_id":     map[string]any{"type": "string"},
					"title":       map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"priority":    map[string]any{"type": "string"},
					"status":      map[string]any{"type": "string", "description": "todo|in_progress|completed|cancelled"},
					"due":         map[string]any{"type": "string", "description": "YYYY-MM-DD"},
				},
				"required": []any{"task_id"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *UpdateTaskTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		TaskID      string  `json:"task_id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Priority    *string `json:"priority"`
		Status      *string `json:"status"`
		Due         *string `json:"due"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.TaskID == "" {
		return nil, fmt.Errorf("schema: missing required field task_id")
	}
	req := service.UpdateTaskRequest{
		Title:       in.Title,
		Description: in.Description,
	}
	if in.Status != nil {
		st := model.TaskStatus(*in.Status)
		req.Status = &st
	}
	if in.Priority != nil {
		if m, ok := priorityMapping[*in.Priority]; ok {
			req.IsImportant = &m.important
			req.IsUrgent = &m.urgent
			q := quadrantOf(m.important, m.urgent)
			req.Quadrant = &q
		}
	}
	if in.Due != nil && *in.Due != "" {
		tt, err := time.Parse("2006-01-02", *in.Due)
		if err != nil {
			return nil, fmt.Errorf("invalid due date %q (expected YYYY-MM-DD): %w", *in.Due, err)
		}
		req.DueDate = &tt
	}
	if err := t.Svc.UpdateTask(in.TaskID, req); err != nil {
		return nil, err
	}
	return map[string]any{"task_id": in.TaskID, "updated": true}, nil
}

func (t *UpdateTaskTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		TaskID string `json:"task_id"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{
		"action":  "update",
		"task_id": in.TaskID,
		"args":    json.RawMessage(args),
	}, nil
}

// =====================================================================
// delete_task
// =====================================================================

// DeleteTaskTool permanently removes a task. PermDangerous: requires the
// second-step confirmation flow.
type DeleteTaskTool struct {
	Svc TaskService
}

func (t *DeleteTaskTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "delete_task",
		Function: agent.FunctionSpec{
			Name:        "delete_task",
			Description: "Permanently delete a task. Cannot be undone.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"task_id": map[string]any{"type": "string"},
				},
				"required": []any{"task_id"},
			},
		},
		Permission: agent.PermDangerous,
	}
}

func (t *DeleteTaskTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		TaskID string `json:"task_id"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.TaskID == "" {
		return nil, fmt.Errorf("schema: missing required field task_id")
	}
	if err := t.Svc.DeleteTask(in.TaskID); err != nil {
		return nil, err
	}
	return map[string]any{"task_id": in.TaskID, "deleted": true}, nil
}

func (t *DeleteTaskTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		TaskID string `json:"task_id"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{
		"action":  "delete",
		"task_id": in.TaskID,
		"warning": "this action is permanent",
	}, nil
}

// derefStr safely dereferences an optional string arg, returning "" for nil.
func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}
