package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"ticktask/internal/model"
	"ticktask/internal/service"
	"testing"
)

func setupAITestServices() (*service.AIService, *service.TaskService, *mockSettingRepository) {
	taskRepo := newMockTaskRepository()
	analyticsRepo := newMockAnalyticsRepository()
	settingRepo := newMockSettingRepository()

	taskService := service.NewTaskService(taskRepo, analyticsRepo, settingRepo, newMockSessionRepository())

	// Get AI settings from the repository
	aiSettings, _ := settingRepo.GetAISettings()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)

	return aiService, taskService, settingRepo
}

// Test: GET /api/ai/status - AI 未配置
func TestAIHandler_GetAIStatus_NotConfigured(t *testing.T) {
	aiService, taskService, _ := setupAITestServices()

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.GET("/api/ai/status", handler.GetAIStatus)

	req, _ := http.NewRequest("GET", "/api/ai/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["configured"] != false {
		t.Errorf("expected configured false, got %v", response["configured"])
	}
}

// Test: GET /api/ai/status - AI 已配置
func TestAIHandler_GetAIStatus_Configured(t *testing.T) {
	aiService, taskService, settingRepo := setupAITestServices()

	// Configure AI
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key-12345678",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
	})

	// Re-create AI service with new settings
	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService = service.NewAIService(aiSettings, taskRepo, nil, nil)

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.GET("/api/ai/status", handler.GetAIStatus)

	req, _ := http.NewRequest("GET", "/api/ai/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["configured"] != true {
		t.Errorf("expected configured true, got %v", response["configured"])
	}
}

// Test: POST /api/ai/classify - AI 未配置
func TestAIHandler_ClassifyTask_NotConfigured(t *testing.T) {
	aiService, taskService, _ := setupAITestServices()

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/classify", handler.ClassifyTask)

	body := map[string]interface{}{
		"task_id": "test-task-id",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/classify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should return 503 Service Unavailable when AI is not configured
	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["error"] == nil {
		t.Error("expected error message when AI not configured")
	}
}

// Test: POST /api/ai/classify - 缺少 task_id
func TestAIHandler_ClassifyTask_MissingTaskID(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/classify", handler.ClassifyTask)

	body := map[string]interface{}{}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/classify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: POST /api/ai/classify - 任务不存在
func TestAIHandler_ClassifyTask_TaskNotFound(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/classify", handler.ClassifyTask)

	body := map[string]interface{}{
		"task_id": "nonexistent-task-id",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/classify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// Test: POST /api/ai/classify/batch - AI 未配置
func TestAIHandler_ClassifyTasks_NotConfigured(t *testing.T) {
	aiService, taskService, _ := setupAITestServices()

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/classify/batch", handler.ClassifyTasks)

	body := map[string]interface{}{
		"task_ids": []string{"task-1", "task-2"},
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/classify/batch", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// Test: POST /api/ai/classify/batch - 缺少 task_ids
func TestAIHandler_ClassifyTasks_MissingTaskIDs(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/classify/batch", handler.ClassifyTasks)

	body := map[string]interface{}{}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/classify/batch", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: POST /api/ai/schedule - AI 未配置
func TestAIHandler_GenerateSchedule_NotConfigured(t *testing.T) {
	aiService, taskService, _ := setupAITestServices()

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/schedule", handler.GenerateSchedule)

	body := map[string]interface{}{
		"start_time": "09:00",
		"end_time":   "18:00",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/schedule", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// Test: POST /api/ai/schedule - 缺少时间参数
func TestAIHandler_GenerateSchedule_MissingTime(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/schedule", handler.GenerateSchedule)

	body := map[string]interface{}{
		"start_time": "09:00",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/schedule", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: GET /api/ai/priority - AI 未配置
func TestAIHandler_GetPrioritySuggestions_NotConfigured(t *testing.T) {
	aiService, taskService, _ := setupAITestServices()

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.GET("/api/ai/priority", handler.GetPrioritySuggestions)

	req, _ := http.NewRequest("GET", "/api/ai/priority", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}
// Test: POST /api/ai/classify/text - AI 未配置
func TestAIHandler_ClassifyTaskByText_NotConfigured(t *testing.T) {
	aiService, taskService, _ := setupAITestServices()

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/classify/text", handler.ClassifyTaskByText)

	body := map[string]interface{}{
		"title": "重构用户模块",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/classify/text", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// Test: POST /api/ai/classify/text - 缺少 title
func TestAIHandler_ClassifyTaskByText_MissingTitle(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/classify/text", handler.ClassifyTaskByText)

	body := map[string]interface{}{
		"description": "some description without title",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/classify/text", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: POST /api/ai/classify/text - 无效 JSON
func TestAIHandler_ClassifyTaskByText_InvalidJSON(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/classify/text", handler.ClassifyTaskByText)

	req, _ := http.NewRequest("POST", "/api/ai/classify/text", bytes.NewBuffer([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: POST /api/ai/reschedule - AI 未配置
func TestAIHandler_RescheduleAfterInterrupt_NotConfigured(t *testing.T) {
	aiService, taskService, _ := setupAITestServices()

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/reschedule", handler.RescheduleAfterInterrupt)

	body := map[string]interface{}{
		"task_id":          "task-1",
		"completed_minutes": 10,
		"planned_minutes":  25,
		"interrupt_reason": "meeting",
		"work_end_time":    "18:00",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/reschedule", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// Test: POST /api/ai/reschedule - 缺少必填字段 task_id
func TestAIHandler_RescheduleAfterInterrupt_MissingTaskID(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/reschedule", handler.RescheduleAfterInterrupt)

	body := map[string]interface{}{
		"planned_minutes": 25,
		"work_end_time":   "18:00",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/reschedule", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: POST /api/ai/reschedule - 缺少必填字段 planned_minutes
func TestAIHandler_RescheduleAfterInterrupt_MissingPlannedMinutes(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/reschedule", handler.RescheduleAfterInterrupt)

	body := map[string]interface{}{
		"task_id":       "task-1",
		"work_end_time": "18:00",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/reschedule", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: POST /api/ai/reschedule - 缺少必填字段 work_end_time
func TestAIHandler_RescheduleAfterInterrupt_MissingWorkEndTime(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/reschedule", handler.RescheduleAfterInterrupt)

	body := map[string]interface{}{
		"task_id":          "task-1",
		"completed_minutes": 10,
		"planned_minutes":  25,
		"interrupt_reason": "call",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/ai/reschedule", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: POST /api/ai/reschedule - 无效 JSON
func TestAIHandler_RescheduleAfterInterrupt_InvalidJSON(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/reschedule", handler.RescheduleAfterInterrupt)

	req, _ := http.NewRequest("POST", "/api/ai/reschedule", bytes.NewBuffer([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: GET /api/ai/insights - AI 未配置
func TestAIHandler_GetDailyInsights_NotConfigured(t *testing.T) {
	aiService, taskService, _ := setupAITestServices()

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.GET("/api/ai/insights", handler.GetDailyInsights)

	req, _ := http.NewRequest("GET", "/api/ai/insights", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

// Test: GET /api/ai/insights - 使用默认参数不 panic
func TestAIHandler_GetDailyInsights_DefaultParams(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.GET("/api/ai/insights", handler.GetDailyInsights)

	req, _ := http.NewRequest("GET", "/api/ai/insights", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Should not panic; may return 200 or 500 depending on AI reachability
	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d or %d, got %d", http.StatusOK, http.StatusInternalServerError, w.Code)
	}
}

// Test: GET /api/ai/insights - 带完整查询参数不 panic
func TestAIHandler_GetDailyInsights_WithParams(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.GET("/api/ai/insights", handler.GetDailyInsights)

	req, _ := http.NewRequest("GET", "/api/ai/insights?date=2026-05-25&completed_pomodoros=8&total_focus_minutes=200&completed_tasks=5&total_interruptions=2", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK && w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d or %d, got %d", http.StatusOK, http.StatusInternalServerError, w.Code)
	}
}

// Test: GET /api/ai/status - 验证响应结构
func TestAIHandler_GetAIStatus_ResponseStructure(t *testing.T) {
	aiService, taskService, _ := setupAITestServices()

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.GET("/api/ai/status", handler.GetAIStatus)

	req, _ := http.NewRequest("GET", "/api/ai/status", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if _, ok := response["configured"]; !ok {
		t.Error("expected 'configured' key in AI status response")
	}
}

// Test: POST /api/ai/schedule - 无效 JSON
func TestAIHandler_GenerateSchedule_InvalidJSON(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-test-api-key",
		Model:    "gpt-4o-mini",
	})

	aiSettings, _ := settingRepo.GetAISettings()
	taskRepo := newMockTaskRepository()
	aiService := service.NewAIService(aiSettings, taskRepo, nil, nil)
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo, newMockSessionRepository())

	handler := NewAIHandler(aiService, taskService)
	router := setupTestRouter()
	router.POST("/api/ai/schedule", handler.GenerateSchedule)

	req, _ := http.NewRequest("POST", "/api/ai/schedule", bytes.NewBuffer([]byte("{")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}
