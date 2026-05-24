package service

import (
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/internal/websocket"
	"testing"
	"time"
)

// Mock SessionRepository
type MockSessionRepository struct {
	sessions  map[string]*model.PomodoroSession
	active    *model.PomodoroSession
	createFn  func(session *model.PomodoroSession) error
	updateFn  func(session *model.PomodoroSession) error
}

func NewMockSessionRepository() *MockSessionRepository {
	return &MockSessionRepository{
		sessions: make(map[string]*model.PomodoroSession),
	}
}

func (m *MockSessionRepository) Create(session *model.PomodoroSession) error {
	if m.createFn != nil {
		return m.createFn(session)
	}
	m.sessions[session.ID] = session
	if session.Status == model.SessionRunning {
		m.active = session
	}
	return nil
}

func (m *MockSessionRepository) Update(session *model.PomodoroSession) error {
	if m.updateFn != nil {
		return m.updateFn(session)
	}
	m.sessions[session.ID] = session
	return nil
}

func (m *MockSessionRepository) GetByID(id string) (*model.PomodoroSession, error) {
	if session, ok := m.sessions[id]; ok {
		return session, nil
	}
	return nil, repository.ErrNotFound
}

func (m *MockSessionRepository) GetActive() (*model.PomodoroSession, error) {
	if m.active != nil {
		return m.active, nil
	}
	return nil, nil
}

func (m *MockSessionRepository) GetRecent(limit int) ([]model.PomodoroSession, error) {
	var result []model.PomodoroSession
	for _, session := range m.sessions {
		result = append(result, *session)
	}
	return result, nil
}

func (m *MockSessionRepository) GetByDate(date time.Time) ([]model.PomodoroSession, error) {
	var result []model.PomodoroSession
	for _, session := range m.sessions {
		sessionDate := session.StartTime.In(date.Location())
		if sessionDate.Year() == date.Year() && sessionDate.Month() == date.Month() && sessionDate.Day() == date.Day() {
			result = append(result, *session)
		}
	}
	return result, nil
}

// Tests

func TestTimerService_StartSession(t *testing.T) {
	sessionRepo := NewMockSessionRepository()
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()
	wsHub := websocket.NewHub()

	service := NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)

	req := CreateSessionRequest{
		Type:     model.SessionWork,
		Duration: 1500,
	}

	session, err := service.StartSession(req)
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	if session.Type != model.SessionWork {
		t.Errorf("expected type %s, got %s", model.SessionWork, session.Type)
	}

	if session.Status != model.SessionRunning {
		t.Errorf("expected status %s, got %s", model.SessionRunning, session.Status)
	}

	if session.PlannedDuration != 1500 {
		t.Errorf("expected duration 1500, got %d", session.PlannedDuration)
	}
}

func TestTimerService_StartSession_WithTask(t *testing.T) {
	sessionRepo := NewMockSessionRepository()
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()
	wsHub := websocket.NewHub()

	service := NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)

	taskID := "task-123"
	req := CreateSessionRequest{
		TaskID:   &taskID,
		Type:     model.SessionWork,
		Duration: 1500,
	}

	session, err := service.StartSession(req)
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	if session.TaskID == nil || *session.TaskID != taskID {
		t.Errorf("expected task_id %s, got %v", taskID, session.TaskID)
	}
}

func TestTimerService_GetActiveSession(t *testing.T) {
	sessionRepo := NewMockSessionRepository()
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()
	wsHub := websocket.NewHub()

	service := NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)

	// No active session initially
	active, err := service.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active != nil {
		t.Error("expected no active session initially")
	}

	// Start a session
	_, _ = service.StartSession(CreateSessionRequest{
		Type:     model.SessionWork,
		Duration: 60,
	})

	// Now there should be an active session
	active, err = service.GetActiveSession()
	if err != nil {
		t.Fatalf("GetActiveSession failed: %v", err)
	}
	if active == nil {
		t.Error("expected active session")
	}
}

func TestTimerService_ControlSession_Complete(t *testing.T) {
	sessionRepo := NewMockSessionRepository()
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()
	wsHub := websocket.NewHub()

	service := NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)

	// Start a short session for testing
	session, _ := service.StartSession(CreateSessionRequest{
		Type:     model.SessionWork,
		Duration: 1, // 1 second for quick test
	})

	// Wait a moment then complete
	time.Sleep(100 * time.Millisecond)

	err := service.ControlSession(session.ID, "complete", "")
	if err != nil {
		t.Fatalf("ControlSession complete failed: %v", err)
	}

	// Verify session is completed
	updatedSession, _ := sessionRepo.GetByID(session.ID)
	if updatedSession.Status != model.SessionCompleted {
		t.Errorf("expected status %s, got %s", model.SessionCompleted, updatedSession.Status)
	}

	if updatedSession.EndTime == nil {
		t.Error("expected EndTime to be set")
	}

	if updatedSession.ActualDuration == nil {
		t.Error("expected ActualDuration to be set")
	}
}

func TestTimerService_ControlSession_Abandon(t *testing.T) {
	sessionRepo := NewMockSessionRepository()
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()
	wsHub := websocket.NewHub()

	service := NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)

	session, _ := service.StartSession(CreateSessionRequest{
		Type:     model.SessionWork,
		Duration: 1500,
	})

	err := service.ControlSession(session.ID, "abandon", "meeting")
	if err != nil {
		t.Fatalf("ControlSession abandon failed: %v", err)
	}

	updatedSession, _ := sessionRepo.GetByID(session.ID)
	if updatedSession.Status != model.SessionAbandoned {
		t.Errorf("expected status %s, got %s", model.SessionAbandoned, updatedSession.Status)
	}
}

func TestTimerService_GetRecentSessions(t *testing.T) {
	sessionRepo := NewMockSessionRepository()
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()
	wsHub := websocket.NewHub()

	service := NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)

	// Create multiple sessions
	service.StartSession(CreateSessionRequest{Type: model.SessionWork, Duration: 1500})
	service.ControlSession("test", "complete", "") // This will fail but that's ok for this test

	recent, err := service.GetRecentSessions(10)
	if err != nil {
		t.Fatalf("GetRecentSessions failed: %v", err)
	}

	// At least one session should exist
	if len(recent) < 1 {
		t.Error("expected at least one recent session")
	}
}

func TestTimerService_StartBreak(t *testing.T) {
	sessionRepo := NewMockSessionRepository()
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()
	wsHub := websocket.NewHub()

	service := NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)

	// Start a short break
	session, err := service.StartSession(CreateSessionRequest{
		Type:     model.SessionShortBreak,
		Duration: 300,
	})
	if err != nil {
		t.Fatalf("StartSession failed: %v", err)
	}

	if session.Type != model.SessionShortBreak {
		t.Errorf("expected type %s, got %s", model.SessionShortBreak, session.Type)
	}
}

func TestTimerService_SessionTypes(t *testing.T) {
	sessionRepo := NewMockSessionRepository()
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()
	wsHub := websocket.NewHub()

	service := NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)

	tests := []struct {
		sessionType model.SessionType
		duration    int
	}{
		{model.SessionWork, 1500},
		{model.SessionShortBreak, 300},
		{model.SessionLongBreak, 900},
	}

	for _, tt := range tests {
		session, err := service.StartSession(CreateSessionRequest{
			Type:     tt.sessionType,
			Duration: tt.duration,
		})
		if err != nil {
			t.Fatalf("StartSession failed for %s: %v", tt.sessionType, err)
		}

		if session.Type != tt.sessionType {
			t.Errorf("expected type %s, got %s", tt.sessionType, session.Type)
		}

		// Stop the timer for next test
		service.stopTimer()
		sessionRepo.active = nil
	}
}