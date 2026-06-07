package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"ticktask/internal/model"
	"ticktask/internal/service"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func setupAnalyticsTest(t *testing.T) (*service.AnalyticsService, *gin.Engine) {
	gin.SetMode(gin.TestMode)

	sessionRepo := newMockSessionRepository()
	taskRepo := newMockTaskRepository()
	analyticsRepo := newMockAnalyticsRepository()
	settingRepo := newMockSettingRepository()

	analyticsService := service.NewAnalyticsService(analyticsRepo, taskRepo, sessionRepo, settingRepo)
	handler := NewAnalyticsHandler(analyticsService)

	router := gin.New()
	router.GET("/api/analytics/pomodoro-by-task", handler.GetPomodoroByTask)
	router.GET("/api/analytics/pomodoro-trends", handler.GetPomodoroTrends)

	return analyticsService, router
}

func TestAnalyticsHandler_GetPomodoroByTask(t *testing.T) {
	_, router := setupAnalyticsTest(t)

	req, _ := http.NewRequest("GET", "/api/analytics/pomodoro-by-task?period=week", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result service.PomodoroByTaskResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if result.Tasks == nil {
		t.Error("expected tasks array, got nil")
	}
}

func TestAnalyticsHandler_GetPomodoroByTask_InvalidPeriod(t *testing.T) {
	_, router := setupAnalyticsTest(t)

	req, _ := http.NewRequest("GET", "/api/analytics/pomodoro-by-task?period=year", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAnalyticsHandler_GetPomodoroByTask_DefaultPeriod(t *testing.T) {
	_, router := setupAnalyticsTest(t)

	req, _ := http.NewRequest("GET", "/api/analytics/pomodoro-by-task", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAnalyticsHandler_GetPomodoroTrends(t *testing.T) {
	_, router := setupAnalyticsTest(t)

	req, _ := http.NewRequest("GET", "/api/analytics/pomodoro-trends?period=month", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result service.PomodoroTrendsResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(result.Days) == 0 {
		t.Error("expected non-empty days array")
	}
}

func TestAnalyticsHandler_GetPomodoroTrends_InvalidPeriod(t *testing.T) {
	_, router := setupAnalyticsTest(t)

	req, _ := http.NewRequest("GET", "/api/analytics/pomodoro-trends?period=quarter", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestAnalyticsHandler_GetPomodoroByTask_WithData(t *testing.T) {
	gin.SetMode(gin.TestMode)

	sessionRepo := newMockSessionRepository()
	taskRepo := newMockTaskRepository()
	analyticsRepo := newMockAnalyticsRepository()
	settingRepo := newMockSettingRepository()

	// Seed a task
	task := &model.Task{
		ID:            "task-1",
		Title:         "Test Task",
		Quadrant:      model.Quadrant1,
		Status:        model.StatusInProgress,
		EstimatedTime: 50, // 50 min → ceil(50/25) = 2 planned
		CreatedAt:     time.Now().Add(-24 * time.Hour),
		UpdatedAt:     time.Now(),
	}
	taskRepo.tasks[task.ID] = task

	// Seed completed work sessions
	taskID := "task-1"
	sessionRepo.sessions["sess-1"] = &model.PomodoroSession{
		ID:              "sess-1",
		TaskID:          &taskID,
		Type:            model.SessionWork,
		Status:          model.SessionCompleted,
		StartTime:       time.Now().Add(-2 * time.Hour),
		PlannedDuration: 1500,
		ActualDuration:  intPtr(1500),
		CreatedAt:       time.Now().Add(-2 * time.Hour),
	}

	analyticsService := service.NewAnalyticsService(analyticsRepo, taskRepo, sessionRepo, settingRepo)
	handler := NewAnalyticsHandler(analyticsService)

	router := gin.New()
	router.GET("/api/analytics/pomodoro-by-task", handler.GetPomodoroByTask)

	req, _ := http.NewRequest("GET", "/api/analytics/pomodoro-by-task?period=week", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, w.Code, w.Body.String())
	}

	var result service.PomodoroByTaskResult
	if err := json.Unmarshal(w.Body.Bytes(), &result); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if len(result.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(result.Tasks))
	}

	item := result.Tasks[0]
	if item.TaskID != "task-1" {
		t.Errorf("expected task_id task-1, got %s", item.TaskID)
	}
	if item.PlannedPomodoros != 2 {
		t.Errorf("expected planned_pomodoros 2, got %d", item.PlannedPomodoros)
	}
	if item.CompletedPomodoros != 1 {
		t.Errorf("expected completed_pomodoros 1, got %d", item.CompletedPomodoros)
	}
	if item.Status != "in_progress" {
		t.Errorf("expected status in_progress, got %s", item.Status)
	}
	if item.TotalFocusMinutes != 25 {
		t.Errorf("expected total_focus_minutes 25, got %d", item.TotalFocusMinutes)
	}
}

func intPtr(i int) *int {
	return &i
}
