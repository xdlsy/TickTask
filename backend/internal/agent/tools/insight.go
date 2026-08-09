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
	GetTrend(days int) (*service.TrendData, error)
	GetDistribution(start, end time.Time) (*service.DistributionStats, error)
	GetPomodoroByTask(period string) (*service.PomodoroByTaskResult, error)
	GetPomodoroTrends(period string) (*service.PomodoroTrendsResult, error)
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

// =====================================================================
// get_analytics
// =====================================================================

// GetAnalyticsTool dispatches to one of four analytics queries by `metric`.
// PermRead. (get_daily_insights stays the dedicated daily-summary tool.)
type GetAnalyticsTool struct{ Svc AnalyticsService }

func (t *GetAnalyticsTool) Schema() agent.ToolSchema {
	return agent.ToolSchema{
		Name: "get_analytics",
		Function: agent.FunctionSpec{
			Name:        "get_analytics",
			Description: "Query analytics: metric=trend (days N), distribution (from/to YYYY-MM-DD), pomodoro_by_task (period week|month), or pomodoro_trends (period week|month).",
			Parameters: map[string]any{
				"type": "object",
				"properties": map[string]any{
					"metric": map[string]any{"type": "string", "description": "trend|distribution|pomodoro_by_task|pomodoro_trends"},
					"days":   map[string]any{"type": "integer", "description": "trend: number of days"},
					"from":   map[string]any{"type": "string", "description": "distribution: YYYY-MM-DD"},
					"to":     map[string]any{"type": "string", "description": "distribution: YYYY-MM-DD"},
					"period": map[string]any{"type": "string", "description": "pomodoro_by_task/pomodoro_trends: week|month"},
				},
				"required": []any{"metric"},
			},
		},
		Permission: agent.PermRead,
	}
}

func (t *GetAnalyticsTool) Execute(ctx context.Context, args json.RawMessage) (any, error) {
	if err := agent.ValidateArgs(t.Schema().Function.Parameters, args); err != nil {
		return nil, err
	}
	var in struct {
		Metric string  `json:"metric"`
		Days   *int    `json:"days"`
		From   *string `json:"from"`
		To     *string `json:"to"`
		Period *string `json:"period"`
	}
	if err := json.Unmarshal(args, &in); err != nil {
		return nil, fmt.Errorf("parse args: %w", err)
	}
	loc := time.Now().Location()
	switch in.Metric {
	case "trend":
		days := 7
		if in.Days != nil && *in.Days > 0 {
			days = *in.Days
		}
		out, err := t.Svc.GetTrend(days)
		if err != nil {
			return nil, err
		}
		return map[string]any{"metric": "trend", "data": out}, nil
	case "distribution":
		from, to, err := parseAnalyticsRange(in.From, in.To, loc)
		if err != nil {
			return nil, err
		}
		out, err := t.Svc.GetDistribution(from, to)
		if err != nil {
			return nil, err
		}
		return map[string]any{"metric": "distribution", "data": out}, nil
	case "pomodoro_by_task":
		out, err := t.Svc.GetPomodoroByTask(periodOrDefault(in.Period))
		if err != nil {
			return nil, err
		}
		return map[string]any{"metric": "pomodoro_by_task", "data": out}, nil
	case "pomodoro_trends":
		out, err := t.Svc.GetPomodoroTrends(periodOrDefault(in.Period))
		if err != nil {
			return nil, err
		}
		return map[string]any{"metric": "pomodoro_trends", "data": out}, nil
	default:
		return nil, fmt.Errorf("schema: metric must be trend|distribution|pomodoro_by_task|pomodoro_trends, got %q", in.Metric)
	}
}

func (t *GetAnalyticsTool) Preview(ctx context.Context, args json.RawMessage) (any, error) {
	return t.Execute(ctx, args)
}

// parseAnalyticsRange resolves distribution from/to. When both are provided they
// are used as-is; otherwise it defaults to the current week — Monday 00:00
// through the following Monday 00:00 (a 7-day half-open range).
func parseAnalyticsRange(from, to *string, loc *time.Location) (time.Time, time.Time, error) {
	parse := func(s string) (time.Time, error) {
		return time.ParseInLocation("2006-01-02", s, loc)
	}
	if from != nil && *from != "" && to != nil && *to != "" {
		f, err := parse(*from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid from %q: %w", *from, err)
		}
		tt, err := parse(*to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid to %q: %w", *to, err)
		}
		return f, tt, nil
	}
	now := time.Now()
	day := (int(now.Weekday()) + 6) % 7
	mon := time.Date(now.Year(), now.Month(), now.Day()-day, 0, 0, 0, 0, loc)
	return mon, mon.AddDate(0, 0, 7), nil
}

func periodOrDefault(p *string) string {
	if p != nil && *p != "" {
		return *p
	}
	return "week"
}
