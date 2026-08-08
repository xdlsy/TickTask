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
	summary      *service.DailySummary
	summaryErr   error
	summaryDate  time.Time
	summaryCalls int
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
