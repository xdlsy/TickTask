package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"ticktask/internal/model"
	"testing"
)

// Test: GET /api/settings - 获取所有设置
func TestSettingHandler_GetSettings(t *testing.T) {
	settingRepo := newMockSettingRepository()
	handler := NewSettingHandler(settingRepo)

	router := setupTestRouter()
	router.GET("/api/settings", handler.GetSettings)

	req, _ := http.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// Should have pomodoro and ai settings
	if _, ok := response["pomodoro"]; !ok {
		t.Error("expected 'pomodoro' key in response")
	}
	if _, ok := response["ai"]; !ok {
		t.Error("expected 'ai' key in response")
	}
}

// Test: GET /api/settings - 验证默认值
func TestSettingHandler_GetSettings_DefaultValues(t *testing.T) {
	settingRepo := newMockSettingRepository()
	handler := NewSettingHandler(settingRepo)

	router := setupTestRouter()
	router.GET("/api/settings", handler.GetSettings)

	req, _ := http.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	pomodoro := response["pomodoro"].(map[string]interface{})

	// Verify default pomodoro settings
	if pomodoro["work_duration"].(float64) != 1500 {
		t.Errorf("expected work_duration 1500, got %v", pomodoro["work_duration"])
	}
	if pomodoro["short_break_duration"].(float64) != 300 {
		t.Errorf("expected short_break_duration 300, got %v", pomodoro["short_break_duration"])
	}
	if pomodoro["long_break_duration"].(float64) != 900 {
		t.Errorf("expected long_break_duration 900, got %v", pomodoro["long_break_duration"])
	}
	if pomodoro["long_break_after"].(float64) != 4 {
		t.Errorf("expected long_break_after 4, got %v", pomodoro["long_break_after"])
	}
}

// Test: GET /api/settings - API Key 隐藏测试
func TestSettingHandler_GetSettings_APIKeyHidden(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "sk-1234567890abcdefghijklmnop",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
	})

	handler := NewSettingHandler(settingRepo)
	router := setupTestRouter()
	router.GET("/api/settings", handler.GetSettings)

	req, _ := http.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	ai := response["ai"].(map[string]interface{})
	apiKey := ai["api_key"].(string)

	// API key should be masked
	if apiKey == "sk-1234567890abcdefghijklmnop" {
		t.Error("API key should be masked")
	}
	if len(apiKey) < 8 {
		t.Error("masked API key should have some characters")
	}
}

// Test: GET /api/settings - 短 API Key 隐藏测试
func TestSettingHandler_GetSettings_ShortAPIKeyHidden(t *testing.T) {
	settingRepo := newMockSettingRepository()
	settingRepo.UpdateAISettings(&model.AISettings{
		Provider: "openai",
		APIKey:   "short",
		BaseURL:  "https://api.openai.com/v1",
		Model:    "gpt-4o-mini",
	})

	handler := NewSettingHandler(settingRepo)
	router := setupTestRouter()
	router.GET("/api/settings", handler.GetSettings)

	req, _ := http.NewRequest("GET", "/api/settings", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	ai := response["ai"].(map[string]interface{})
	apiKey := ai["api_key"].(string)

	// Short API key should be completely masked
	if apiKey != "****" {
		t.Errorf("expected short API key to be '****', got %s", apiKey)
	}
}

// Test: PUT /api/settings/pomodoro - 更新番茄设置
func TestSettingHandler_UpdatePomodoroSettings(t *testing.T) {
	settingRepo := newMockSettingRepository()
	handler := NewSettingHandler(settingRepo)

	router := setupTestRouter()
	router.PUT("/api/settings/pomodoro", handler.UpdatePomodoroSettings)

	body := map[string]interface{}{
		"work_duration":         1800, // 30 minutes
		"short_break_duration":  600,  // 10 minutes
		"long_break_duration":   1200, // 20 minutes
		"long_break_after":      3,
		"auto_start_break":      true,
		"auto_start_work":       false,
		"enable_sound":          true,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/settings/pomodoro", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "Settings updated" {
		t.Errorf("expected message 'Settings updated', got %v", response["message"])
	}

	// Verify settings were updated
	settings, _ := settingRepo.GetPomodoroSettings()
	if settings.WorkDuration != 1800 {
		t.Errorf("expected work_duration 1800, got %d", settings.WorkDuration)
	}
}

// Test: PUT /api/settings/pomodoro - 验证更新后的值
func TestSettingHandler_UpdatePomodoroSettings_VerifyValues(t *testing.T) {
	settingRepo := newMockSettingRepository()
	handler := NewSettingHandler(settingRepo)

	// Update settings
	router := setupTestRouter()
	router.PUT("/api/settings/pomodoro", handler.UpdatePomodoroSettings)

	body := map[string]interface{}{
		"work_duration":         1200,
		"short_break_duration":  240,
		"long_break_duration":   600,
		"long_break_after":      2,
		"auto_start_break":      true,
		"auto_start_work":       true,
		"enable_sound":          false,
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/settings/pomodoro", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Verify all values
	settings, _ := settingRepo.GetPomodoroSettings()
	if settings.WorkDuration != 1200 {
		t.Errorf("expected work_duration 1200, got %d", settings.WorkDuration)
	}
	if settings.ShortBreakDuration != 240 {
		t.Errorf("expected short_break_duration 240, got %d", settings.ShortBreakDuration)
	}
	if settings.LongBreakDuration != 600 {
		t.Errorf("expected long_break_duration 600, got %d", settings.LongBreakDuration)
	}
	if settings.LongBreakAfter != 2 {
		t.Errorf("expected long_break_after 2, got %d", settings.LongBreakAfter)
	}
	if !settings.AutoStartBreak {
		t.Error("expected auto_start_break to be true")
	}
	if !settings.AutoStartWork {
		t.Error("expected auto_start_work to be true")
	}
	if settings.EnableSound {
		t.Error("expected enable_sound to be false")
	}
}

// Test: PUT /api/settings/ai - 更新 AI 设置
func TestSettingHandler_UpdateAISettings(t *testing.T) {
	settingRepo := newMockSettingRepository()
	handler := NewSettingHandler(settingRepo)

	router := setupTestRouter()
	router.PUT("/api/settings/ai", handler.UpdateAISettings)

	body := map[string]interface{}{
		"provider": "openai",
		"api_key":  "sk-new-api-key-12345678",
		"base_url": "https://api.openai.com/v1",
		"model":    "gpt-4o",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/settings/ai", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["message"] != "AI settings updated" {
		t.Errorf("expected message 'AI settings updated', got %v", response["message"])
	}
}

// Test: PUT /api/settings/ai - 更新 Anthropic 设置
func TestSettingHandler_UpdateAISettings_Anthropic(t *testing.T) {
	settingRepo := newMockSettingRepository()
	handler := NewSettingHandler(settingRepo)

	router := setupTestRouter()
	router.PUT("/api/settings/ai", handler.UpdateAISettings)

	body := map[string]interface{}{
		"provider": "anthropic",
		"api_key":  "sk-ant-api-key-12345678",
		"base_url": "https://api.anthropic.com/v1",
		"model":    "claude-3-5-sonnet-latest",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/settings/ai", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify settings were updated
	settings, _ := settingRepo.GetAISettings()
	if settings.Provider != "anthropic" {
		t.Errorf("expected provider anthropic, got %s", settings.Provider)
	}
	if settings.Model != "claude-3-5-sonnet-latest" {
		t.Errorf("expected model claude-3-5-sonnet-latest, got %s", settings.Model)
	}
}

// Test: PUT /api/settings/ai - 更新自定义提供商设置
func TestSettingHandler_UpdateAISettings_Custom(t *testing.T) {
	settingRepo := newMockSettingRepository()
	handler := NewSettingHandler(settingRepo)

	router := setupTestRouter()
	router.PUT("/api/settings/ai", handler.UpdateAISettings)

	body := map[string]interface{}{
		"provider": "custom",
		"api_key":  "custom-api-key",
		"base_url": "https://custom-api.example.com/v1",
		"model":    "custom-model",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/settings/ai", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	settings, _ := settingRepo.GetAISettings()
	if settings.Provider != "custom" {
		t.Errorf("expected provider custom, got %s", settings.Provider)
	}
	if settings.BaseURL != "https://custom-api.example.com/v1" {
		t.Errorf("expected custom base_url, got %s", settings.BaseURL)
	}
}

// Test: PUT /api/settings/pomodoro - 无效请求体
func TestSettingHandler_UpdatePomodoroSettings_InvalidBody(t *testing.T) {
	settingRepo := newMockSettingRepository()
	handler := NewSettingHandler(settingRepo)

	router := setupTestRouter()
	router.PUT("/api/settings/pomodoro", handler.UpdatePomodoroSettings)

	req, _ := http.NewRequest("PUT", "/api/settings/pomodoro", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test: PUT /api/settings/ai - 无效请求体
func TestSettingHandler_UpdateAISettings_InvalidBody(t *testing.T) {
	settingRepo := newMockSettingRepository()
	handler := NewSettingHandler(settingRepo)

	router := setupTestRouter()
	router.PUT("/api/settings/ai", handler.UpdateAISettings)

	req, _ := http.NewRequest("PUT", "/api/settings/ai", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}