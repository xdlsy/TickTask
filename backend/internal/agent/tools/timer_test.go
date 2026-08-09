package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/internal/service"
)

// --- mocks ---

// mockTimerSvc is an in-memory implementation of the tools-package TimerService
// interface. It records the calls made so tests can assert on them.
type mockTimerSvc struct {
	// state
	active    *model.PomodoroSession
	startErr  error
	pauseErr  error
	activeErr error

	// call recording
	startReq    *service.CreateSessionRequest
	startCalls  int
	pausedID    string
	pauseCalls  int
	activeCalls int

	// new fields for control_pomodoro and get_pomodoro_stats
	resumeErr   error
	resumeCalls int
	completeErr error
	completeIDs []string
	abandonErr  error
	abandonIDs  []string
	abandonWhy  []string
	todayStats  []service.TaskTimeStats
	todayErr    error
	recent      []model.PomodoroSession
	recentErr   error
	recentLimit int
}

func (m *mockTimerSvc) StartSession(req service.CreateSessionRequest) (*model.PomodoroSession, error) {
	m.startCalls++
	m.startReq = &req
	if m.startErr != nil {
		return nil, m.startErr
	}
	sess := &model.PomodoroSession{
		ID:              "sess-1",
		TaskID:          req.TaskID,
		Type:            req.Type,
		Status:          model.SessionRunning,
		StartTime:       time.Now(),
		PlannedDuration: req.Duration,
	}
	if sess.PlannedDuration == 0 {
		sess.PlannedDuration = 1500 // 25 min default
	}
	m.active = sess
	return sess, nil
}

func (m *mockTimerSvc) PauseSession(sessionID string) error {
	m.pauseCalls++
	m.pausedID = sessionID
	if m.pauseErr != nil {
		return m.pauseErr
	}
	if m.active != nil && m.active.ID == sessionID {
		m.active = nil
	}
	return nil
}

func (m *mockTimerSvc) GetActiveSession() (*model.PomodoroSession, error) {
	m.activeCalls++
	if m.activeErr != nil {
		return nil, m.activeErr
	}
	return m.active, nil
}

func (m *mockTimerSvc) ResumeSession(sessionID string) error {
	m.resumeCalls++
	return m.resumeErr
}

func (m *mockTimerSvc) CompleteSession(sessionID string) error {
	m.completeIDs = append(m.completeIDs, sessionID)
	return m.completeErr
}

func (m *mockTimerSvc) AbandonSession(sessionID string, interruptReason string) error {
	m.abandonIDs = append(m.abandonIDs, sessionID)
	m.abandonWhy = append(m.abandonWhy, interruptReason)
	return m.abandonErr
}

func (m *mockTimerSvc) GetTodayTaskStats() ([]service.TaskTimeStats, error) {
	if m.todayErr != nil {
		return nil, m.todayErr
	}
	return m.todayStats, nil
}

func (m *mockTimerSvc) GetRecentSessions(limit int) ([]model.PomodoroSession, error) {
	m.recentLimit = limit
	if m.recentErr != nil {
		return nil, m.recentErr
	}
	return m.recent, nil
}

// --- StartPomodoroTool ---

func TestStartPomodoro_Bare(t *testing.T) {
	svc := &mockTimerSvc{}
	tool := &StartPomodoroTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.startReq == nil {
		t.Fatal("StartSession not called")
	}
	if svc.startReq.Type != model.SessionWork {
		t.Fatalf("type = %q, want work", svc.startReq.Type)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if !strings.Contains(body, `"sess-1"`) {
		t.Fatalf("expected session id in result: %s", body)
	}
}

func TestStartPomodoro_WithTaskAndDuration(t *testing.T) {
	svc := &mockTimerSvc{}
	tool := &StartPomodoroTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":"t-1","duration_min":50}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.startReq.TaskID == nil || *svc.startReq.TaskID != "t-1" {
		t.Fatalf("task id = %v", svc.startReq.TaskID)
	}
	// duration_min is converted to seconds for CreateSessionRequest.Duration
	if svc.startReq.Duration != 50*60 {
		t.Fatalf("duration = %d, want %d", svc.startReq.Duration, 50*60)
	}
}

func TestStartPomodoro_SchemaValidationFails(t *testing.T) {
	svc := &mockTimerSvc{}
	tool := &StartPomodoroTool{Svc: svc}
	// task_id wrong type
	_, err := tool.Execute(context.Background(), json.RawMessage(`{"task_id":123}`))
	if err == nil {
		t.Fatal("expected schema validation error")
	}
	if !strings.Contains(err.Error(), "schema") {
		t.Fatalf("expected schema error, got %v", err)
	}
	if svc.startCalls != 0 {
		t.Fatalf("StartSession should not be called on validation failure, got %d", svc.startCalls)
	}
}

func TestStartPomodoro_Preview(t *testing.T) {
	svc := &mockTimerSvc{}
	tool := &StartPomodoroTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{"task_id":"t-1","duration_min":30}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(preview)
	body := string(m)
	if !strings.Contains(body, `"start_pomodoro"`) {
		t.Fatalf("preview should expose action=start_pomodoro: %s", body)
	}
	if svc.startCalls != 0 {
		t.Fatalf("Preview must not call StartSession, got %d calls", svc.startCalls)
	}
}

func TestStartPomodoro_ServiceError(t *testing.T) {
	svc := &mockTimerSvc{startErr: errors.New("db locked")}
	tool := &StartPomodoroTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil || !strings.Contains(err.Error(), "db locked") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

// --- StopPomodoroTool ---

func TestStopPomodoro_Active(t *testing.T) {
	svc := &mockTimerSvc{active: &model.PomodoroSession{ID: "s-active", Status: model.SessionRunning}}
	tool := &StopPomodoroTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	if svc.pausedID != "s-active" {
		t.Fatalf("paused id = %q, want s-active", svc.pausedID)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), `"stopped":true`) {
		t.Fatalf("expected stopped=true: %s", m)
	}
}

func TestStopPomodoro_NoActive(t *testing.T) {
	svc := &mockTimerSvc{active: nil}
	tool := &StopPomodoroTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), `"already_stopped":true`) {
		t.Fatalf("expected already_stopped=true: %s", m)
	}
	if svc.pauseCalls != 0 {
		t.Fatalf("PauseSession should not be called when no active session, got %d", svc.pauseCalls)
	}
}

func TestStopPomodoro_GetActiveError(t *testing.T) {
	svc := &mockTimerSvc{activeErr: repository.ErrNotFound}
	tool := &StopPomodoroTool{Svc: svc}
	_, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error from GetActiveSession")
	}
}

func TestStopPomodoro_Preview(t *testing.T) {
	svc := &mockTimerSvc{active: &model.PomodoroSession{ID: "s-active"}}
	tool := &StopPomodoroTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(preview)
	body := string(m)
	if !strings.Contains(body, `"stop_pomodoro"`) {
		t.Fatalf("preview should expose action=stop_pomodoro: %s", body)
	}
	if svc.pauseCalls != 0 {
		t.Fatalf("Preview must not call PauseSession, got %d", svc.pauseCalls)
	}
}

// --- GetTimerStatusTool ---

func TestGetTimerStatus_Active(t *testing.T) {
	taskID := "t-1"
	svc := &mockTimerSvc{active: &model.PomodoroSession{
		ID:              "s-active",
		TaskID:          &taskID,
		Type:            model.SessionWork,
		Status:          model.SessionRunning,
		PlannedDuration: 1500,
	}}
	tool := &GetTimerStatusTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if !strings.Contains(body, `"active":true`) {
		t.Fatalf("expected active=true: %s", body)
	}
	if !strings.Contains(body, `"s-active"`) {
		t.Fatalf("expected session id in result: %s", body)
	}
}

func TestGetTimerStatus_Inactive(t *testing.T) {
	svc := &mockTimerSvc{active: nil}
	tool := &GetTimerStatusTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	body := string(m)
	if !strings.Contains(body, `"active":false`) {
		t.Fatalf("expected active=false: %s", body)
	}
}

func TestGetTimerStatus_Preview_EqualsExecute(t *testing.T) {
	svc := &mockTimerSvc{active: &model.PomodoroSession{ID: "s-active"}}
	tool := &GetTimerStatusTool{Svc: svc}
	preview, err := tool.Preview(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(preview)
	if !strings.Contains(string(m), `"s-active"`) {
		t.Fatalf("preview should mirror execute for read tool: %s", m)
	}
}

// --- ControlPomodoroTool ---

func TestControlPomodoro_RoutesActionToSession(t *testing.T) {
	cases := []struct {
		action string
		check  func(*mockTimerSvc) string // returns "" if ok, else a failure reason
	}{
		{"resume", func(s *mockTimerSvc) string {
			if s.resumeCalls != 1 {
				return fmt.Sprintf("resume: ResumeSession calls = %d, want 1", s.resumeCalls)
			}
			return ""
		}},
		{"complete", func(s *mockTimerSvc) string {
			if len(s.completeIDs) != 1 {
				return fmt.Sprintf("complete: CompleteSession calls = %d, want 1", len(s.completeIDs))
			}
			return ""
		}},
		{"abandon", func(s *mockTimerSvc) string {
			if len(s.abandonIDs) != 1 {
				return fmt.Sprintf("abandon: AbandonSession calls = %d, want 1", len(s.abandonIDs))
			}
			return ""
		}},
	}
	for _, c := range cases {
		svc := &mockTimerSvc{active: &model.PomodoroSession{ID: "s1"}}
		tool := &ControlPomodoroTool{Svc: svc}
		if _, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"`+c.action+`"}`)); err != nil {
			t.Errorf("action %s: %v", c.action, err)
			continue
		}
		if reason := c.check(svc); reason != "" {
			t.Errorf("action %s: %s", c.action, reason)
		}
	}
	// cross-check: each action must call ONLY its own method
	rsvc := &mockTimerSvc{active: &model.PomodoroSession{ID: "s1"}}
	rtool := &ControlPomodoroTool{Svc: rsvc}
	_, _ = rtool.Execute(context.Background(), json.RawMessage(`{"action":"resume"}`))
	if len(rsvc.completeIDs) != 0 || len(rsvc.abandonIDs) != 0 {
		t.Errorf("resume must not call complete/abandon: complete=%d abandon=%d", len(rsvc.completeIDs), len(rsvc.abandonIDs))
	}

	// unknown action rejected before touching the service
	badSvc := &mockTimerSvc{}
	badTool := &ControlPomodoroTool{Svc: badSvc}
	_, err := badTool.Execute(context.Background(), json.RawMessage(`{"action":"bogus"}`))
	if err == nil || !strings.Contains(err.Error(), "action") {
		t.Fatalf("expected action enum error, got %v", err)
	}
	if badSvc.resumeCalls != 0 || len(badSvc.completeIDs) != 0 || len(badSvc.abandonIDs) != 0 {
		t.Errorf("service must not be called on bad action")
	}
}

func TestControlPomodoro_AbandonForwardsReason(t *testing.T) {
	svc := &mockTimerSvc{active: &model.PomodoroSession{ID: "s1"}}
	tool := &ControlPomodoroTool{Svc: svc}
	if _, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"abandon","reason":"被打断了"}`)); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(svc.abandonWhy) != 1 || svc.abandonWhy[0] != "被打断了" {
		t.Errorf("reason not forwarded, got %+v", svc.abandonWhy)
	}
	// and: abandon with no reason defaults to ""
	svc2 := &mockTimerSvc{active: &model.PomodoroSession{ID: "s2"}}
	tool2 := &ControlPomodoroTool{Svc: svc2}
	if _, err := tool2.Execute(context.Background(), json.RawMessage(`{"action":"abandon"}`)); err != nil {
		t.Fatalf("execute: %v", err)
	}
	if len(svc2.abandonWhy) != 1 || svc2.abandonWhy[0] != "" {
		t.Errorf("missing reason should default to \"\", got %+v", svc2.abandonWhy)
	}
}

func TestControlPomodoro_RequiresActiveSession(t *testing.T) {
	svc := &mockTimerSvc{} // no active session set → GetActiveSession returns nil
	tool := &ControlPomodoroTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{"action":"resume"}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := res.(map[string]any)
	if m["active"] != false {
		t.Errorf("expected active=false when no session, got %+v", res)
	}
}

func TestControlPomodoro_PreviewNoSideEffect(t *testing.T) {
	svc := &mockTimerSvc{}
	tool := &ControlPomodoroTool{Svc: svc}
	pv, err := tool.Preview(context.Background(), json.RawMessage(`{"action":"resume"}`))
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	m, _ := json.Marshal(pv)
	if !strings.Contains(string(m), "control_pomodoro") || svc.resumeCalls != 0 {
		t.Fatalf("preview must echo plan and not call service: %s", m)
	}
}

// --- GetPomodoroStatsTool ---

func TestGetPomodoroStats_Aggregates(t *testing.T) {
	svc := &mockTimerSvc{
		todayStats: []service.TaskTimeStats{{TaskID: "t1", TaskTitle: "写文档", SessionCount: 2, TotalTime: 1800}},
		recent:     []model.PomodoroSession{{ID: "s1"}},
	}
	tool := &GetPomodoroStatsTool{Svc: svc}
	res, err := tool.Execute(context.Background(), json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("execute: %v", err)
	}
	m, _ := json.Marshal(res)
	if !strings.Contains(string(m), "写文档") || !strings.Contains(string(m), `"s1"`) {
		t.Errorf("result should include today stats + recent sessions: %s", m)
	}
	if svc.recentLimit != 10 {
		t.Errorf("default recent_limit should be 10, got %d", svc.recentLimit)
	}
}
