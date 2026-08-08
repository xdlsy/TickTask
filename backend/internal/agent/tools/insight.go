package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"ticktask/internal/agent"
	"ticktask/internal/service"
)

// AnalyticsService is the subset of service.AnalyticsService methods that
// insight tools need. The production service.AnalyticsService struct satisfies
// this implicitly; tests substitute a mock that implements only GetSummary.
type AnalyticsService interface {
	GetSummary(date time.Time) (*service.DailySummary, error)
}

// =====================================================================
// get_daily_insights
// =====================================================================

// GetDailyInsightsTool returns the raw daily summary for a given date. The
// agent (which is itself the LLM) synthesizes insights from this structured
// data — we deliberately do NOT call AIService.GetDailyInsights internally,
// since that would re-prompt the LLM with the same context the agent loop
// already has. PermRead: auto-executed by the agent loop.
type GetDailyInsightsTool struct {
	Svc AnalyticsService
}

func (t *GetDailyInsightsTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "get_daily_insights",
		Function: agent.FunctionSpec{
			Name:        "get_daily_insights",
			Description: "Return the raw daily summary (focus time, pomodoros, tasks) for a date. The agent synthesizes insights from this data.",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"date": map[string]any{"type": "string", "description": "YYYY-MM-DD (defaults to today)"},
				},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *GetDailyInsightsTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Date *string `json:"date"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}

	loc := time.Now().Location()
	dateStr := time.Now().Format("2006-01-02")
	if in.Date != nil && *in.Date != "" {
		if _, err := time.Parse("2006-01-02", *in.Date); err != nil {
			return nil, fmt.Errorf("invalid date %q (expected YYYY-MM-DD): %w", *in.Date, err)
		}
		dateStr = *in.Date
	}

	date, err := time.ParseInLocation("2006-01-02", dateStr, loc)
	if err != nil {
		return nil, fmt.Errorf("parse date %q: %w", dateStr, err)
	}

	summary, err := t.Svc.GetSummary(date)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"date":    dateStr,
		"summary": summary,
	}, nil
}

func (t *GetDailyInsightsTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}
