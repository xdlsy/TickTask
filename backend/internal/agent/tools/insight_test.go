package tools

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ticktask/internal/repository"
	"ticktask/internal/service"
)

// --- mocks ---

// mockAnalyticsSvc is an in-memory implementation of the tools-package
// AnalyticsService interface.
type mockAnalyticsSvc struct {
	// GetSummary fields
	summary      *service.DailySummary
	summaryErr   error
	summaryDate  time.Time
	summaryCalls int
	// GetTrend fields
	trendOut   *service.TrendData
	trendErr   error
	trendDays  int
	// GetDistribution fields
	distOut   *service.DistributionStats
	distErr   error
	distStart time.Time
	distEnd   time.Time
	// GetPomodoroByTask fields
	byTaskOut  *service.PomodoroByTaskResult
	byTaskErr  error
	byTaskArg  string
	// GetPomodoroTrends fields
	pTrendsOut *service.PomodoroTrendsResult
	pTrendsErr error
	pTrendsArg string
}

func (m *mockAnalyticsSvc) GetSummary(date time.Time) (*service.DailySummary, error) {
	m.summaryCalls++
	m.summaryDate = date
	if m.summaryErr != nil {
		return nil, m.summaryErr
	}
	if m.summary == nil {
		// Default non-nil summary
		return &service.DailySummary{}, nil
	}
	return m.summary, nil
}

func (m *mockAnalyticsSvc) GetTrend(days int) (*service.TrendData, error) {
	m.trendDays = days
	if m.trendErr != nil {
		return nil, m.trendErr
	}
	return m.trendOut, nil
}

func (m *mockAnalyticsSvc) GetDistribution(start, end time.Time) (*service.DistributionStats, error) {
	m.distStart = start
	m.distEnd = end
	if m.distErr != nil {
		return nil, m.distErr
	}
	return m.distOut, nil
}

func (m *mockAnalyticsSvc) GetPomodoroByTask(period string) (*service.PomodoroByTaskResult, error) {
	m.byTaskArg = period
	if m.byTaskErr != nil {
		return nil, m.byTaskErr
	}
	return m.byTaskOut, nil
}

func (m *mockAnalyticsSvc) GetPomodoroTrends(period string) (*service.PomodoroTrendsResult, error) {
	m.pTrendsArg = period
	if m.pTrendsErr != nil {
		return nil, m.pTrendsErr
	}
	return m.pTrendsOut, nil
}

// --- GetDailyInsightsTool ---

func TestGetDailyInsights(t *testing.T) {
	svc := &mockAnalyticsSvc{
		summary: &service.DailySummary{
			CompletedPomodoros: 4,
			TotalFocusTime:     6000,
			CompletedTasks:     2,
			CreatedTasks:       3,
		},
	}
	tool := &GetDailyInsightsTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"2026-08-08"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.summaryCalls != 1 {
		t.Fatalf("GetSummary calls = %d, want 1", svc.summaryCalls)
	}
	// Compare by calendar date only — the tool parses with local time zone,
	// so we assert the YYYY-MM-DD string, not the absolute UTC instant.
	got := svc.summaryDate.Format("2006-01-02")
	if got != "2026-08-08" {
		t.Fatalf("summaryDate = %v, want date 2026-08-08", svc.summaryDate)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if !strings.Contains(body, `"completed_pomodoros":4`) {
		t.Fatalf("expected raw focus data: %s", body)
	}
	if !strings.Contains(body, `"total_focus_time":6000`) {
		t.Fatalf("expected total_focus_time: %s", body)
	}
}

func TestGetDailyInsights_DefaultsToToday(t *testing.T) {
	svc := &mockAnalyticsSvc{summary: &service.DailySummary{}}
	tool := &GetDailyInsightsTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	today := time.Now().Format("2006-01-02")
	if svc.summaryDate.Format("2006-01-02") != today {
		t.Fatalf("default date = %q, want today %q", svc.summaryDate.Format("2006-01-02"), today)
	}
}

func TestGetDailyInsights_MissingDateFallsBack(t *testing.T) {
	svc := &mockAnalyticsSvc{summary: &service.DailySummary{CompletedPomodoros: 1}}
	tool := &GetDailyInsightsTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), `"completed_pomodoros":1`) {
		t.Fatalf("expected raw data even without date: %s", m)
	}
}

func TestGetDailyInsights_InvalidDate(t *testing.T) {
	svc := &mockAnalyticsSvc{}
	tool := &GetDailyInsightsTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"not-a-date"}`))
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
	if svc.summaryCalls != 0 {
		t.Fatalf("GetSummary should not be called on parse failure")
	}
}

func TestGetDailyInsights_SchemaValidationFails(t *testing.T) {
	svc := &mockAnalyticsSvc{}
	tool := &GetDailyInsightsTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"date":42}`))
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema error, got %v", err)
	}
}

func TestGetDailyInsights_ServiceError(t *testing.T) {
	svc := &mockAnalyticsSvc{summaryErr: repository.ErrNotFound}
	tool := &GetDailyInsightsTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"2026-08-08"}`))
	if err == nil {
		t.Fatal("expected error from GetSummary")
	}
}

func TestGetDailyInsights_Preview_EqualsExecute(t *testing.T) {
	svc := &mockAnalyticsSvc{summary: &service.DailySummary{CompletedPomodoros: 7}}
	tool := &GetDailyInsightsTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{"date":"2026-08-08"}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(preview)
	if !strings.Contains(string(m), `"completed_pomodoros":7`) {
		t.Fatalf("preview should mirror execute for read tool: %s", m)
	}
}

func TestGetAnalytics_DispatchesByMetric(t *testing.T) {
	cases := []struct {
		args   string
		metric string
	}{
		{`{"metric":"trend","days":7}`, "trend"},
		{`{"metric":"distribution","from":"2026-08-01","to":"2026-08-09"}`, "distribution"},
		{`{"metric":"pomodoro_by_task","period":"week"}`, "pomodoro_by_task"},
		{`{"metric":"pomodoro_trends","period":"week"}`, "pomodoro_trends"},
	}
	for _, c := range cases {
		svc := &mockAnalyticsSvc{
			trendOut: &service.TrendData{}, distOut: &service.DistributionStats{},
			byTaskOut: &service.PomodoroByTaskResult{}, pTrendsOut: &service.PomodoroTrendsResult{},
		}
		tool := &GetAnalyticsTool{Svc: svc}
		res, err := tool.Execute(context.Background(), json.RawMessage(c.args))
		if err != nil {
			t.Errorf("metric %s: %v", c.metric, err)
		}
		m, _ := json.Marshal(res)
		if !strings.Contains(string(m), `"metric":"`+c.metric+`"`) {
			t.Errorf("metric %s: result should echo metric, got %s", c.metric, m)
		}
	}
	// bad metric rejected before service
	svc := &mockAnalyticsSvc{}
	tool := &GetAnalyticsTool{Svc: svc}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"metric":"bogus"}`)); err == nil {
		t.Error("expected error for bad metric")
	}
}

func TestGetAnalytics_TrendDefaultsDays(t *testing.T) {
	svc := &mockAnalyticsSvc{trendOut: &service.TrendData{}}
	tool := &GetAnalyticsTool{Svc: svc}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"metric":"trend"}`)); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.trendDays != 7 {
		t.Errorf("trend without days should default to 7, got %d", svc.trendDays)
	}
}

func TestGetAnalytics_DistributionForwardsRange(t *testing.T) {
	svc := &mockAnalyticsSvc{distOut: &service.DistributionStats{}}
	tool := &GetAnalyticsTool{Svc: svc}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"metric":"distribution","from":"2026-08-01","to":"2026-08-09"}`)); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.distStart.Format("2006-01-02") != "2026-08-01" || svc.distEnd.Format("2006-01-02") != "2026-08-09" {
		t.Errorf("range not forwarded: start=%v end=%v", svc.distStart, svc.distEnd)
	}
}

func TestGetAnalytics_PomodoroDefaultsPeriod(t *testing.T) {
	svc := &mockAnalyticsSvc{byTaskOut: &service.PomodoroByTaskResult{}}
	tool := &GetAnalyticsTool{Svc: svc}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"metric":"pomodoro_by_task"}`)); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.byTaskArg != "week" {
		t.Errorf("pomodoro_by_task without period should default to week, got %q", svc.byTaskArg)
	}
}

func TestGetAnalytics_DistributionDefaultsToCurrentWeek(t *testing.T) {
	svc := &mockAnalyticsSvc{distOut: &service.DistributionStats{}}
	tool := &GetAnalyticsTool{Svc: svc}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"metric":"distribution"}`)); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.distEnd.Sub(svc.distStart) != 7*24*time.Hour {
		t.Errorf("default range should span 7 days, got %v", svc.distEnd.Sub(svc.distStart))
	}
	if svc.distStart.Weekday() != time.Monday {
		t.Errorf("default start should be Monday, got %v", svc.distStart.Weekday())
	}
}

// --- RegisterAll extended ---

func TestRegisterAll_RegistersAllTools(t *testing.T) {
	reg := newTestRegistry()
	wantNames := []string{
		"list_tasks", "create_task", "update_task", "delete_task", "classify_task",
		"start_pomodoro", "stop_pomodoro", "get_timer_status",
		"generate_schedule", "list_schedule",
		"get_daily_insights",
		"structure_worklog", "save_worklog",
	}
	for _, name := range wantNames {
		if _, err := reg.Lookup(name); err != nil {
			t.Errorf("tool %q not registered: %v", name, err)
		}
	}
}
