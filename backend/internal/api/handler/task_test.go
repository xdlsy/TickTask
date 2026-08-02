package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"ticktask/internal/model"
	"ticktask/internal/service"
	"testing"

	"github.com/gin-gonic/gin"
)

// Setup test router
func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	return gin.New()
}

func TestTaskHandler_GetTasks(t *testing.T) {
	// Create mock services
	taskService := service.NewTaskService(
			newMockTaskRepository(),
			newMockAnalyticsRepository(),
			newMockSettingRepository(),
			newMockSessionRepository(),
		)

	handler := NewTaskHandler(taskService)
	router := setupTestRouter()
	router.GET("/api/tasks", handler.GetTasks)

	// Create request
	req, _ := http.NewRequest("GET", "/api/tasks", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestTaskHandler_CreateTask(t *testing.T) {
	taskService := service.NewTaskService(
			newMockTaskRepository(),
			newMockAnalyticsRepository(),
			newMockSettingRepository(),
			newMockSessionRepository(),
		)

	handler := NewTaskHandler(taskService)
	router := setupTestRouter()
	router.POST("/api/tasks", handler.CreateTask)

	// Create request body
	body := map[string]interface{}{
		"title":       "Test Task",
		"description": "Test Description",
		"quadrant":    1,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestTaskHandler_CreateTask_MissingTitle(t *testing.T) {
	taskService := service.NewTaskService(
			newMockTaskRepository(),
			newMockAnalyticsRepository(),
			newMockSettingRepository(),
			newMockSessionRepository(),
		)

	handler := NewTaskHandler(taskService)
	router := setupTestRouter()
	router.POST("/api/tasks", handler.CreateTask)

	// Missing title
	body := map[string]interface{}{
		"description": "Test Description",
		"quadrant":    1,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should fail validation
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestTaskHandler_CreateTask_InvalidQuadrant(t *testing.T) {
	taskService := service.NewTaskService(
			newMockTaskRepository(),
			newMockAnalyticsRepository(),
			newMockSettingRepository(),
			newMockSessionRepository(),
		)

	handler := NewTaskHandler(taskService)
	router := setupTestRouter()
	router.POST("/api/tasks", handler.CreateTask)

	// Invalid quadrant (5)
	body := map[string]interface{}{
		"title":    "Test Task",
		"quadrant": 5,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestTaskHandler_GetTasksByQuadrant(t *testing.T) {
	taskService := service.NewTaskService(
			newMockTaskRepository(),
			newMockAnalyticsRepository(),
			newMockSettingRepository(),
			newMockSessionRepository(),
		)

	handler := NewTaskHandler(taskService)
	router := setupTestRouter()
	router.GET("/api/tasks/quadrant", handler.GetTasksByQuadrant)

	req, _ := http.NewRequest("GET", "/api/tasks/quadrant", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify response is a map with 4 quadrants
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	for i := 1; i <= 4; i++ {
		key := string(rune('0' + i))
		if _, ok := response[key]; !ok {
			t.Errorf("expected quadrant %s in response", key)
		}
	}
}

func TestTaskHandler_MoveTask(t *testing.T) {
	taskService := service.NewTaskService(
			newMockTaskRepository(),
			newMockAnalyticsRepository(),
			newMockSettingRepository(),
			newMockSessionRepository(),
		)

	handler := NewTaskHandler(taskService)
	router := setupTestRouter()
	router.PATCH("/api/tasks/:id/move", handler.MoveTask)

	// First create a task
	task, _ := taskService.CreateTask(service.CreateTaskRequest{
		Title:    "Move Test",
		Quadrant: model.Quadrant1,
	})

	// Move request
	body := map[string]interface{}{
		"quadrant": 2,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PATCH", "/api/tasks/"+task.ID+"/move", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestTaskHandler_DeleteTask(t *testing.T) {
	taskService := service.NewTaskService(
			newMockTaskRepository(),
			newMockAnalyticsRepository(),
			newMockSettingRepository(),
			newMockSessionRepository(),
		)

	handler := NewTaskHandler(taskService)
	router := setupTestRouter()
	router.DELETE("/api/tasks/:id", handler.DeleteTask)

	// Create a task first
	task, _ := taskService.CreateTask(service.CreateTaskRequest{
		Title:    "Delete Test",
		Quadrant: model.Quadrant1,
	})

	req, _ := http.NewRequest("DELETE", "/api/tasks/"+task.ID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestTaskHandler_GetTask(t *testing.T) {
	taskService := service.NewTaskService(
			newMockTaskRepository(),
			newMockAnalyticsRepository(),
			newMockSettingRepository(),
			newMockSessionRepository(),
		)

	handler := NewTaskHandler(taskService)
	router := setupTestRouter()
	router.GET("/api/tasks/:id", handler.GetTask)

	// Create a task
	task, _ := taskService.CreateTask(service.CreateTaskRequest{
		Title:    "Get Test",
		Quadrant: model.Quadrant1,
	})

	req, _ := http.NewRequest("GET", "/api/tasks/"+task.ID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestTaskHandler_GetTask_NotFound(t *testing.T) {
	taskService := service.NewTaskService(
			newMockTaskRepository(),
			newMockAnalyticsRepository(),
			newMockSettingRepository(),
			newMockSessionRepository(),
		)

	handler := NewTaskHandler(taskService)
	router := setupTestRouter()
	router.GET("/api/tasks/:id", handler.GetTask)

	req, _ := http.NewRequest("GET", "/api/tasks/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestTaskHandler_CreateTask_WithNewFields(t *testing.T) {
	taskService := service.NewTaskService(
			newMockTaskRepository(),
			newMockAnalyticsRepository(),
			newMockSettingRepository(),
			newMockSessionRepository(),
		)

	handler := NewTaskHandler(taskService)
	router := setupTestRouter()
	router.POST("/api/tasks", handler.CreateTask)

	body := map[string]interface{}{
		"title":                "Recurring Task",
		"description":          "A task with all new fields",
		"quadrant":             2,
		"estimated_time":       45,
		"is_recurring":         true,
		"recurrence_pattern":   "daily",
		"preferred_start_time": "09:00",
		"preferred_end_time":   "10:30",
		"start_date":           "2026-06-01T00:00:00Z",
		"due_date":             "2026-06-30T00:00:00Z",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/tasks", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, w.Code, w.Body.String())
	}

	var task model.Task
	json.Unmarshal(w.Body.Bytes(), &task)

	if task.Title != "Recurring Task" {
		t.Errorf("expected title 'Recurring Task', got %s", task.Title)
	}
	if !task.IsRecurring {
		t.Error("expected is_recurring to be true")
	}
	if task.RecurrencePattern != "daily" {
		t.Errorf("expected recurrence_pattern 'daily', got %s", task.RecurrencePattern)
	}
	if task.PreferredStartTime != "09:00" {
		t.Errorf("expected preferred_start_time '09:00', got %s", task.PreferredStartTime)
	}
	if task.PreferredEndTime != "10:30" {
		t.Errorf("expected preferred_end_time '10:30', got %s", task.PreferredEndTime)
	}
	if task.StartDate == nil {
		t.Error("expected start_date to be set")
	}
	if task.DueDate == nil {
		t.Error("expected due_date to be set")
	}
}