package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"ticktask/internal/model"
	"ticktask/internal/service"
	"ticktask/internal/websocket"
	"testing"
)

func setupTimerTestServices() (*service.TimerService, *mockSessionRepository) {
	sessionRepo := newMockSessionRepository()
	taskRepo := newMockTaskRepository()
	analyticsRepo := newMockAnalyticsRepository()
	settingRepo := newMockSettingRepository()
	wsHub := websocket.NewHub()

	timerService := service.NewTimerService(sessionRepo, taskRepo, analyticsRepo, settingRepo, wsHub)
	return timerService, sessionRepo
}

// Test: GET /api/sessions/active - 无活跃会话
func TestTimerHandler_GetActiveSession_NoActive(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.GET("/api/sessions/active", handler.GetActiveSession)

	req, _ := http.NewRequest("GET", "/api/sessions/active", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// When no active session, handler returns null (nil session)
	// or {"session": null} depending on error condition
	body := w.Body.String()
	if body == "null" {
		return // This is expected when no active session
	}

	// Also accept {"session": null} format
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err == nil {
		if _, ok := response["session"]; ok {
			return // Valid response format
		}
	}

	t.Errorf("unexpected response: %s", body)
}

// Test: GET /api/sessions/active - 有活跃会话
func TestTimerHandler_GetActiveSession_WithActive(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	// Create a session first
	timerService.StartSession(service.CreateSessionRequest{
		Type:     model.SessionWork,
		Duration: 1500,
	})

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.GET("/api/sessions/active", handler.GetActiveSession)

	req, _ := http.NewRequest("GET", "/api/sessions/active", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// Should have session data when active
	if response["status"] != string(model.SessionRunning) {
		t.Errorf("expected status running, got %v", response["status"])
	}
}

// Test: POST /api/sessions - 创建工作会话
func TestTimerHandler_CreateSession_Work(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.POST("/api/sessions", handler.CreateSession)

	body := map[string]interface{}{
		"type":     "work",
		"duration": 1500,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/sessions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["type"] != "work" {
		t.Errorf("expected type work, got %v", response["type"])
	}

	if response["status"] != "running" {
		t.Errorf("expected status running, got %v", response["status"])
	}
}

// Test: POST /api/sessions - 创建短休息会话
func TestTimerHandler_CreateSession_ShortBreak(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.POST("/api/sessions", handler.CreateSession)

	body := map[string]interface{}{
		"type":     "short_break",
		"duration": 300,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/sessions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["type"] != "short_break" {
		t.Errorf("expected type short_break, got %v", response["type"])
	}
}

// Test: POST /api/sessions - 创建长休息会话
func TestTimerHandler_CreateSession_LongBreak(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.POST("/api/sessions", handler.CreateSession)

	body := map[string]interface{}{
		"type":     "long_break",
		"duration": 900,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/sessions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["type"] != "long_break" {
		t.Errorf("expected type long_break, got %v", response["type"])
	}
}

// Test: POST /api/sessions - 关联任务
func TestTimerHandler_CreateSession_WithTask(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	// Create a task first
	taskService := service.NewTaskService(
			newMockTaskRepository(),
			newMockAnalyticsRepository(),
			newMockSettingRepository(),
			newMockSessionRepository(),
		)
	task, _ := taskService.CreateTask(service.CreateTaskRequest{
		Title:    "Test Task",
		Quadrant: model.Quadrant1,
	})

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.POST("/api/sessions", handler.CreateSession)

	body := map[string]interface{}{
		"task_id":  task.ID,
		"type":     "work",
		"duration": 1500,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/sessions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["task_id"] != task.ID {
		t.Errorf("expected task_id %s, got %v", task.ID, response["task_id"])
	}
}

// Test: POST /api/sessions - 缺少 type 参数
func TestTimerHandler_CreateSession_MissingType(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.POST("/api/sessions", handler.CreateSession)

	body := map[string]interface{}{
		"duration": 1500,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/sessions", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: PATCH /api/sessions/:id/control - 暂停会话
func TestTimerHandler_ControlSession_Pause(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	// Create a session
	session, _ := timerService.StartSession(service.CreateSessionRequest{
		Type:     model.SessionWork,
		Duration: 1500,
	})

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.PATCH("/api/sessions/:id/control", handler.ControlSession)

	body := map[string]interface{}{
		"action": "pause",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PATCH", "/api/sessions/"+session.ID+"/control", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test: PATCH /api/sessions/:id/control - 完成会话
func TestTimerHandler_ControlSession_Complete(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	// Create a session
	session, _ := timerService.StartSession(service.CreateSessionRequest{
		Type:     model.SessionWork,
		Duration: 1500,
	})

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.PATCH("/api/sessions/:id/control", handler.ControlSession)

	body := map[string]interface{}{
		"action": "complete",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PATCH", "/api/sessions/"+session.ID+"/control", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "action completed" {
		t.Errorf("expected message 'action completed', got %v", response["message"])
	}
}

// Test: PATCH /api/sessions/:id/control - 放弃会话
func TestTimerHandler_ControlSession_Abandon(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	// Create a session
	session, _ := timerService.StartSession(service.CreateSessionRequest{
		Type:     model.SessionWork,
		Duration: 1500,
	})

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.PATCH("/api/sessions/:id/control", handler.ControlSession)

	body := map[string]interface{}{
		"action": "abandon",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PATCH", "/api/sessions/"+session.ID+"/control", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test: PATCH /api/sessions/:id/control - 缺少 action 参数
func TestTimerHandler_ControlSession_MissingAction(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	// Create a session
	session, _ := timerService.StartSession(service.CreateSessionRequest{
		Type:     model.SessionWork,
		Duration: 1500,
	})

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.PATCH("/api/sessions/:id/control", handler.ControlSession)

	body := map[string]interface{}{}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PATCH", "/api/sessions/"+session.ID+"/control", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: GET /api/sessions/recent - 获取最近会话
func TestTimerHandler_GetRecentSessions(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	// Create some sessions
	timerService.StartSession(service.CreateSessionRequest{Type: model.SessionWork, Duration: 1500})
	timerService.StartSession(service.CreateSessionRequest{Type: model.SessionShortBreak, Duration: 300})

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.GET("/api/sessions/recent", handler.GetRecentSessions)

	req, _ := http.NewRequest("GET", "/api/sessions/recent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response []map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if len(response) < 1 {
		t.Error("expected at least one session")
	}
}

// Test: GET /api/sessions/recent?limit=5 - 带限制参数
func TestTimerHandler_GetRecentSessions_WithLimit(t *testing.T) {
	timerService, _ := setupTimerTestServices()

	handler := NewTimerHandler(timerService)
	router := setupTestRouter()
	router.GET("/api/sessions/recent", handler.GetRecentSessions)

	req, _ := http.NewRequest("GET", "/api/sessions/recent?limit=5", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}