package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ticktask/internal/agent"
	"ticktask/internal/service"
)

// ScheduleService is the subset of service.ScheduleService methods that
// schedule tools need. The production service.ScheduleService struct satisfies
// this implicitly; tests substitute a mock that implements only these two
// methods. Defining the interface here keeps the tools package decoupled from
// the concrete ScheduleService (which owns an aiService dependency — Task 15
// will refactor that out, but for Task 11 we treat ScheduleService as a black
// box and just call its existing methods).
type ScheduleService interface {
	GetSchedules(start, end time.Time) ([]service.ScheduleEvent, error)
	GenerateSchedule(startTime, endTime string) ([]service.ScheduleEvent, string, error)
	DeleteSchedule(id string) error
	UpdateSchedule(id string, dto *service.UpdateScheduleDTO) error
	CreateScheduleEvent(dto *service.CreateScheduleDTO) (*service.ScheduleEvent, error)
}

// =====================================================================
// generate_schedule
// =====================================================================

// GenerateScheduleTool triggers AI-driven schedule generation for a single day.
// The underlying ScheduleService.GenerateSchedule already delegates to the
// auto-schedule skill (Claude CLI / OpenAI), so the tool itself needs no
// LLMClient — we reuse the existing service path. PermWrite: it persists
// schedules to the DB.
type GenerateScheduleTool struct {
	Svc ScheduleService
}

func (t *GenerateScheduleTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "generate_schedule",
		Function: agent.FunctionSpec{
			Name:        "generate_schedule",
			Description: "Generate AI-driven schedules for a single day. Defaults to today when date is omitted.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"date": map[string]any{"type": "string", "description": "YYYY-MM-DD (defaults to today)"},
				},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *GenerateScheduleTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Date *string `json:"date"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	dateStr := time.Now().Format("2006-01-02")
	if in.Date != nil && *in.Date != "" {
		// Validate the user-supplied date format.
		if _, err := time.Parse("2006-01-02", *in.Date); err != nil {
			return nil, fmt.Errorf("invalid date %q (expected YYYY-MM-DD): %w", *in.Date, err)
		}
		dateStr = *in.Date
	}

	// Build the work-window strings expected by ScheduleService.GenerateSchedule
	// (full-day window so the scheduling skill can spread tasks across the day).
	startTime := dateStr + " 09:00"
	endTime := dateStr + " 23:00"

	events, summary, err := t.Svc.GenerateSchedule(startTime, endTime)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"date":    dateStr,
		"events":  events,
		"summary": summary,
		"count":   len(events),
	}, nil
}

func (t *GenerateScheduleTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		Date *string `json:"date"`
	}
	_ = json.Unmarshal(args, &in)
	plan := map[string]any{
		"action": "generate_schedule",
	}
	if in.Date != nil {
		plan["date"] = *in.Date
	} else {
		plan["date"] = time.Now().Format("2006-01-02")
	}
	return plan, nil
}

// =====================================================================
// list_schedule
// =====================================================================

// ListScheduleTool returns schedules overlapping the [from, to] date range.
// PermRead: auto-executed by the agent loop.
type ListScheduleTool struct {
	Svc ScheduleService
}

func (t *ListScheduleTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "list_schedule",
		Function: agent.FunctionSpec{
			Name:        "list_schedule",
			Description: "List schedules in the given YYYY-MM-DD date range (inclusive).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"from": map[string]any{"type": "string", "description": "YYYY-MM-DD inclusive"},
					"to":   map[string]any{"type": "string", "description": "YYYY-MM-DD inclusive"},
				},
				"required": []any{"from", "to"},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *ListScheduleTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		From string `json:"from"`
		To   string `json:"to"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.From == "" {
		return nil, fmt.Errorf("schema: missing required field from")
	}
	if in.To == "" {
		return nil, fmt.Errorf("schema: missing required field to")
	}

	// Parse dates as local midnight; expand [to] to next-day midnight so the
	// upper bound is inclusive of the entire "to" day.
	loc := time.Now().Location()
	startDate, err := time.ParseInLocation("2006-01-02", in.From, loc)
	if err != nil {
		return nil, fmt.Errorf("invalid from %q (expected YYYY-MM-DD): %w", in.From, err)
	}
	endDate, err := time.ParseInLocation("2006-01-02", in.To, loc)
	if err != nil {
		return nil, fmt.Errorf("invalid to %q (expected YYYY-MM-DD): %w", in.To, err)
	}
	endExclusive := endDate.Add(24 * time.Hour)

	events, err := t.Svc.GetSchedules(startDate, endExclusive)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"from":   in.From,
		"to":     in.To,
		"events": events,
		"count":  len(events),
	}, nil
}

func (t *ListScheduleTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}

// =====================================================================
// delete_schedule
// =====================================================================

// DeleteScheduleTool removes a single schedule by id. PermDangerous: deletion is
// irreversible, so the agent loop requires explicit user confirmation. The id
// comes from a prior list_schedule call. This closes the tool-coverage gap
// where the agent previously had no way to delete schedules and would fake it
// by misusing delete_task.
type DeleteScheduleTool struct {
	Svc ScheduleService
}

func (t *DeleteScheduleTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "delete_schedule",
		Function: agent.FunctionSpec{
			Name:        "delete_schedule",
			Description: "Delete one schedule by its id (irreversible). Get the id from list_schedule first.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"schedule_id": map[string]any{"type": "string", "description": "id of the schedule to delete"},
				},
				"required": []any{"schedule_id"},
			},
		},
		Permission: agent.PermDangerous,
	}
}

func (t *DeleteScheduleTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		ScheduleID string `json:"schedule_id"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.ScheduleID == "" {
		return nil, fmt.Errorf("schema: missing required field schedule_id")
	}
	if err := t.Svc.DeleteSchedule(in.ScheduleID); err != nil {
		return nil, err
	}
	return map[string]any{"deleted": true, "schedule_id": in.ScheduleID}, nil
}

func (t *DeleteScheduleTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		ScheduleID string `json:"schedule_id"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "delete_schedule", "schedule_id": in.ScheduleID}, nil
}

// =====================================================================
// update_schedule
// =====================================================================

// UpdateScheduleTool partially updates a schedule event (from list_schedule):
// title/description/start/end/status/color. Only provided fields change. To mark
// a schedule finished set status="completed". PermWrite: requires confirmation.
// NOTE: use schedule ids from list_schedule; do NOT pass task ids here (use
// update_task for tasks) — crossing id domains was the original failure mode.
type UpdateScheduleTool struct {
	Svc ScheduleService
}

var scheduleStatuses = map[string]bool{
	"planned": true, "in_progress": true, "completed": true, "cancelled": true,
}

func (t *UpdateScheduleTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "update_schedule",
		Function: agent.FunctionSpec{
			Name:        "update_schedule",
			Description: "Partially update a schedule event (from list_schedule). Only provided fields change. Set status='completed' to mark a schedule done. Use update_task for tasks, not this.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"schedule_id":  map[string]any{"type": "string", "description": "id from list_schedule"},
					"title":        map[string]any{"type": "string"},
					"description":  map[string]any{"type": "string"},
					"start":        map[string]any{"type": "string", "description": "RFC3339"},
					"end":          map[string]any{"type": "string", "description": "RFC3339"},
					"status":       map[string]any{"type": "string", "description": "planned|in_progress|completed|cancelled"},
					"color":        map[string]any{"type": "string"},
				},
				"required": []any{"schedule_id"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *UpdateScheduleTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		ScheduleID  string  `json:"schedule_id"`
		Title       *string `json:"title"`
		Description *string `json:"description"`
		Start       *string `json:"start"`
		End         *string `json:"end"`
		Status      *string `json:"status"`
		Color       *string `json:"color"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.ScheduleID == "" {
		return nil, fmt.Errorf("schema: missing required field schedule_id")
	}
	if in.Status != nil && *in.Status != "" && !scheduleStatuses[*in.Status] {
		return nil, fmt.Errorf("schema: status must be planned|in_progress|completed|cancelled, got %q", *in.Status)
	}
	dto := &service.UpdateScheduleDTO{}
	if in.Title != nil {
		dto.Title = *in.Title
	}
	if in.Description != nil {
		dto.Description = *in.Description
	}
	if in.Start != nil {
		dto.StartTime = *in.Start
	}
	if in.End != nil {
		dto.EndTime = *in.End
	}
	if in.Status != nil {
		dto.Status = *in.Status
	}
	if in.Color != nil {
		dto.Color = *in.Color
	}
	if err := t.Svc.UpdateSchedule(in.ScheduleID, dto); err != nil {
		return nil, err
	}
	return map[string]any{"schedule_id": in.ScheduleID, "updated": true}, nil
}

func (t *UpdateScheduleTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		ScheduleID string `json:"schedule_id"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "update_schedule", "schedule_id": in.ScheduleID, "args": json.RawMessage(args)}, nil
}

// =====================================================================
// create_schedule
// =====================================================================

// CreateScheduleTool creates a single ad-hoc schedule event (e.g. "加个会，明天
// 下午3点"). PermWrite: requires confirmation. Distinct from generate_schedule,
// which AI-generates a whole day.
type CreateScheduleTool struct {
	Svc ScheduleService
}

var scheduleTypes = map[string]bool{
	"task": true, "pomodoro": true, "break": true, "custom": true,
}

func (t *CreateScheduleTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "create_schedule",
		Function: agent.FunctionSpec{
			Name:        "create_schedule",
			Description: "Create a single ad-hoc schedule event with explicit start/end times (RFC3339). Use generate_schedule for AI bulk scheduling.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"title":       map[string]any{"type": "string"},
					"description": map[string]any{"type": "string"},
					"start":       map[string]any{"type": "string", "description": "RFC3339"},
					"end":         map[string]any{"type": "string", "description": "RFC3339"},
					"type":        map[string]any{"type": "string", "description": "task|pomodoro|break|custom (default task)"},
					"color":       map[string]any{"type": "string"},
				},
				"required": []any{"title", "start", "end"},
			},
		},
		Permission: agent.PermWrite,
	}
}

func (t *CreateScheduleTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Title       string  `json:"title"`
		Description *string `json:"description"`
		Start       string  `json:"start"`
		End         string  `json:"end"`
		Type        *string `json:"type"`
		Color       *string `json:"color"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	if in.Title == "" {
		return nil, fmt.Errorf("schema: missing required field title")
	}
	if in.Start == "" {
		return nil, fmt.Errorf("schema: missing required field start")
	}
	if in.End == "" {
		return nil, fmt.Errorf("schema: missing required field end")
	}
	dto := &service.CreateScheduleDTO{
		Title:     in.Title,
		StartTime: in.Start,
		EndTime:   in.End,
	}
	if in.Description != nil {
		dto.Description = *in.Description
	}
	if in.Type != nil && *in.Type != "" {
		if !scheduleTypes[*in.Type] {
			return nil, fmt.Errorf("schema: type must be task|pomodoro|break|custom, got %q", *in.Type)
		}
		dto.Type = *in.Type
	} else {
		dto.Type = "task"
	}
	if in.Color != nil {
		dto.Color = *in.Color
	}
	return t.Svc.CreateScheduleEvent(dto)
}

func (t *CreateScheduleTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	var in struct {
		Title string `json:"title"`
		Start string `json:"start"`
		End   string `json:"end"`
	}
	_ = json.Unmarshal(args, &in)
	return map[string]any{"action": "create_schedule", "title": in.Title, "start": in.Start, "end": in.End}, nil
}
