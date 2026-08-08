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
