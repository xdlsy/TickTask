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

	taskService := service.NewTaskService(taskRepo, analyticsRepo, settingRepo)

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
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo)

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
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo)

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
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo)

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
	taskService := service.NewTaskService(taskRepo, newMockAnalyticsRepository(), settingRepo)

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