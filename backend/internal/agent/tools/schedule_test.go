package tools

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"ticktask/internal/service"
)

// --- mocks ---

// mockScheduleSvc is an in-memory implementation of the tools-package
// ScheduleService interface.
type mockScheduleSvc struct {
	listEvents []service.ScheduleEvent
	listErr    error
	listStart  time.Time
	listEnd    time.Time
	listCalls  int

	genEvents  []service.ScheduleEvent
	genSummary string
	genErr     error
	genStart   string
	genEnd     string
	genCalls   int

	delErr    error
	delCalls  int
	delIDs    []string
}

func (m *mockScheduleSvc) DeleteSchedule(id string) error {
	m.delCalls++
	m.delIDs = append(m.delIDs, id)
	return m.delErr
}

// --- DeleteScheduleTool ---

func TestDeleteSchedule_DelegatesAndRequiresID(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &DeleteScheduleTool{Svc: svc}

	res, err := tool.Execute(context.Background(), json.RawMessage(`{"schedule_id":"s-1"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.delCalls != 1 || len(svc.delIDs) != 1 || svc.delIDs[0] != "s-1" {
		t.Errorf("DeleteSchedule call = %+v", svc.delIDs)
	}
	m, _ := res.(map[string]any)
	if m["deleted"] != true || m["schedule_id"] != "s-1" {
		t.Errorf("result = %+v", res)
	}

	// missing id must fail BEFORE touching the service
	svc.delCalls = 0
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{}`)); err == nil {
		t.Error("expected error for missing schedule_id")
	}
	if svc.delCalls != 0 {
		t.Errorf("service called %d times for invalid args", svc.delCalls)
	}
}

func (m *mockScheduleSvc) GetSchedules(start, end time.Time) ([]service.ScheduleEvent, error) {
	m.listCalls++
	m.listStart = start
	m.listEnd = end
	if m.listErr != nil {
		return nil, m.listErr
	}
	return m.listEvents, nil
}

func (m *mockScheduleSvc) GenerateSchedule(startTime, endTime string) ([]service.ScheduleEvent, string, error) {
	m.genCalls++
	m.genStart = startTime
	m.genEnd = endTime
	if m.genErr != nil {
		return nil, "", m.genErr
	}
	return m.genEvents, m.genSummary, nil
}

// --- GenerateScheduleTool ---

func TestGenerateSchedule_DefaultDate(t *testing.T) {
	svc := &mockScheduleSvc{
		genEvents:  []service.ScheduleEvent{{ID: "e1", Title: "demo"}},
		genSummary: "scheduled 1 event",
	}
	tool := &GenerateScheduleTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.genCalls != 1 {
		t.Fatalf("GenerateSchedule calls = %d, want 1", svc.genCalls)
	}
	// Should default to today's date for both start and end
	if svc.genStart == "" || svc.genEnd == "" {
		t.Fatalf("expected non-empty start/end strings, got start=%q end=%q", svc.genStart, svc.genEnd)
	}
	if !strings.HasPrefix(svc.genStart, time.Now().Format("2006-01-02")) {
		t.Fatalf("start should default to today: %q", svc.genStart)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if !strings.Contains(body, `"demo"`) || !strings.Contains(body, "scheduled 1 event") {
		t.Fatalf("expected events + summary in result: %s", body)
	}
}

func TestGenerateSchedule_ExplicitDate(t *testing.T) {
	svc := &mockScheduleSvc{
		genEvents: []service.ScheduleEvent{{ID: "e1"}},
	}
	tool := &GenerateScheduleTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"2026-12-25"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if !strings.HasPrefix(svc.genStart, "2026-12-25") {
		t.Fatalf("start = %q, want date prefix 2026-12-25", svc.genStart)
	}
	if !strings.HasPrefix(svc.genEnd, "2026-12-25") {
		t.Fatalf("end = %q, want date prefix 2026-12-25", svc.genEnd)
	}
}

func TestGenerateSchedule_InvalidDate(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &GenerateScheduleTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"date":"not-a-date"}`))
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
	if svc.genCalls != 0 {
		t.Fatalf("GenerateSchedule should not be called on parse failure")
	}
}

func TestGenerateSchedule_SchemaValidationFails(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &GenerateScheduleTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"date":12345}`))
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema error, got %v", err)
	}
}

func TestGenerateSchedule_ServiceError(t *testing.T) {
	svc := &mockScheduleSvc{genErr: errors.New("AI not configured")}
	tool := &GenerateScheduleTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "AI not configured") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestGenerateSchedule_Preview(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &GenerateScheduleTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{"date":"2026-08-08"}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(preview)
	body := string(m)
	if !strings.Contains(body, `"generate_schedule"`) || !strings.Contains(body, `"2026-08-08"`) {
		t.Fatalf("preview should expose action + date: %s", body)
	}
	if svc.genCalls != 0 {
		t.Fatalf("Preview must not call GenerateSchedule, got %d", svc.genCalls)
	}
}

// --- ListScheduleTool ---

func TestListSchedule(t *testing.T) {
	svc := &mockScheduleSvc{
		listEvents: []service.ScheduleEvent{
			{ID: "e1", Title: "standup"},
			{ID: "e2", Title: "deep work"},
		},
	}
	tool := &ListScheduleTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"from":"2026-08-01","to":"2026-08-31"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if !strings.Contains(body, `"standup"`) || !strings.Contains(body, `"deep work"`) {
		t.Fatalf("expected both events: %s", body)
	}
	// Compare by calendar date only — the tool parses with local time zone.
	gotStart := svc.listStart.Format("2006-01-02")
	if gotStart != "2026-08-01" {
		t.Fatalf("start = %v, want date 2026-08-01", svc.listStart)
	}
	gotEnd := svc.listEnd.Format("2006-01-02")
	if gotEnd != "2026-09-01" {
		t.Fatalf("end (exclusive next day) = %v, want date 2026-09-01", svc.listEnd)
	}
}

func TestListSchedule_MissingRequired(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &ListScheduleTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"from":"2026-08-01"}`))
	if err == nil {
		t.Fatal("expected required-field error")
	}
	if !strings.Contains(err.Error(), "to") {
		t.Fatalf("error should mention missing 'to' field: %v", err)
	}
	if svc.listCalls != 0 {
		t.Fatalf("GetSchedules should not be called on validation failure")
	}
}

func TestListSchedule_InvalidDate(t *testing.T) {
	svc := &mockScheduleSvc{}
	tool := &ListScheduleTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"from":"2026-08-01","to":"garbage"}`))
	if err == nil {
		t.Fatal("expected error for invalid date")
	}
}

func TestListSchedule_ServiceError(t *testing.T) {
	svc := &mockScheduleSvc{listErr: errors.New("db locked")}
	tool := &ListScheduleTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"from":"2026-08-01","to":"2026-08-31"}`))
	if err == nil || !strings.Contains(err.Error(), "db locked") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestListSchedule_Preview_EqualsExecute(t *testing.T) {
	svc := &mockScheduleSvc{listEvents: []service.ScheduleEvent{{ID: "e1", Title: "x"}}}
	tool := &ListScheduleTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{"from":"2026-08-01","to":"2026-08-02"}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(preview)
	if !strings.Contains(string(m), `"x"`) {
		t.Fatalf("preview should mirror execute for read tool: %s", m)
	}
}
