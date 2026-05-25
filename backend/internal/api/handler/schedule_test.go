package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/internal/service"
	"testing"
	"time"

	"github.com/google/uuid"
)

// mockScheduleRepositoryForService implements repository.ScheduleRepository for service testing
type mockScheduleRepositoryForService struct {
	schedules map[string]*model.Schedule
}

func newMockScheduleRepositoryForService() *mockScheduleRepositoryForService {
	return &mockScheduleRepositoryForService{
		schedules: make(map[string]*model.Schedule),
	}
}

func (m *mockScheduleRepositoryForService) Create(schedule *model.Schedule) error {
	m.schedules[schedule.ID] = schedule
	return nil
}

func (m *mockScheduleRepositoryForService) GetByID(id string) (*model.Schedule, error) {
	if schedule, ok := m.schedules[id]; ok {
		return schedule, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockScheduleRepositoryForService) GetByTimeRange(start, end time.Time) ([]model.Schedule, error) {
	var result []model.Schedule
	for _, schedule := range m.schedules {
		if (schedule.StartTime.After(start) || schedule.StartTime.Equal(start)) &&
			(schedule.StartTime.Before(end) || schedule.StartTime.Equal(end)) {
			result = append(result, *schedule)
		}
	}
	return result, nil
}

func (m *mockScheduleRepositoryForService) GetByTaskID(taskID string) ([]model.Schedule, error) {
	var result []model.Schedule
	for _, schedule := range m.schedules {
		if schedule.TaskID != nil && *schedule.TaskID == taskID {
			result = append(result, *schedule)
		}
	}
	return result, nil
}

func (m *mockScheduleRepositoryForService) GetByDate(date time.Time) ([]model.Schedule, error) {
	var result []model.Schedule
	for _, schedule := range m.schedules {
		scheduleDate := schedule.StartTime.In(date.Location())
		if scheduleDate.Year() == date.Year() &&
			scheduleDate.Month() == date.Month() &&
			scheduleDate.Day() == date.Day() {
			result = append(result, *schedule)
		}
	}
	return result, nil
}

func (m *mockScheduleRepositoryForService) Update(schedule *model.Schedule) error {
	m.schedules[schedule.ID] = schedule
	return nil
}

func (m *mockScheduleRepositoryForService) Delete(id string) error {
	delete(m.schedules, id)
	return nil
}

func (m *mockScheduleRepositoryForService) UpdateStatus(id string, status model.ScheduleStatus) error {
	if schedule, ok := m.schedules[id]; ok {
		schedule.Status = status
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockScheduleRepositoryForService) DeleteTaskSchedulesByDateRange(start, end time.Time) (int64, error) {
	var count int64
	for id, s := range m.schedules {
		if s.TaskID != nil && *s.TaskID != "" &&
			!s.StartTime.Before(start) && s.StartTime.Before(end) {
			delete(m.schedules, id)
			count++
		}
	}
	return count, nil
}

func (m *mockScheduleRepositoryForService) Move(id string, startTime, endTime time.Time) error {
	if schedule, ok := m.schedules[id]; ok {
		schedule.StartTime = startTime
		schedule.EndTime = endTime
		return nil
	}
	return repository.ErrNotFound
}

// createScheduleService creates a ScheduleService with mock repositories
func createScheduleService() *service.ScheduleService {
	scheduleRepo := newMockScheduleRepositoryForService()
	taskRepo := newMockTaskRepository()
	aiService := &service.AIService{} // Empty AI service for basic tests
	return service.NewScheduleService(scheduleRepo, taskRepo, aiService)
}

// Test GetSchedules - 获取日程列表
func TestScheduleHandler_GetSchedules(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.GET("/api/schedules", handler.GetSchedules)

	req, _ := http.NewRequest("GET", "/api/schedules", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if _, ok := response["events"]; !ok {
		t.Error("expected 'events' key in response")
	}
}

// Test GetSchedules with date range - 带日期范围获取日程
func TestScheduleHandler_GetSchedules_WithDateRange(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.GET("/api/schedules", handler.GetSchedules)

	req, _ := http.NewRequest("GET", "/api/schedules?start=2026-03-01&end=2026-03-31", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test CreateSchedule - 创建日程
func TestScheduleHandler_CreateSchedule(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.POST("/api/schedules", handler.CreateSchedule)

	body := map[string]interface{}{
		"title":      "测试日程",
		"start_time": "2026-03-18T09:00:00Z",
		"end_time":   "2026-03-18T10:00:00Z",
		"type":       "task",
		"color":      "#3b82f6",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/schedules", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["title"] != "测试日程" {
		t.Errorf("expected title '测试日程', got %v", response["title"])
	}
}

// Test CreateSchedule with missing title - 缺少标题创建日程
func TestScheduleHandler_CreateSchedule_MissingTitle(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.POST("/api/schedules", handler.CreateSchedule)

	body := map[string]interface{}{
		"start_time": "2026-03-18T09:00:00Z",
		"end_time":   "2026-03-18T10:00:00Z",
		"type":       "task",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/schedules", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Without title and task_id, should fail
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test CreateSchedule with missing time - 缺少时间创建日程
func TestScheduleHandler_CreateSchedule_MissingTime(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.POST("/api/schedules", handler.CreateSchedule)

	body := map[string]interface{}{
		"title": "测试日程",
		"type":  "task",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/schedules", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test GetSchedule - 获取单个日程
func TestScheduleHandler_GetSchedule(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.GET("/api/schedules/:id", handler.GetSchedule)

	// Create a schedule first
	schedule, _ := scheduleService.CreateSchedule(&service.CreateScheduleDTO{
		Title:     "获取测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	req, _ := http.NewRequest("GET", "/api/schedules/"+schedule.ID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test GetSchedule not found - 获取不存在的日程
func TestScheduleHandler_GetSchedule_NotFound(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.GET("/api/schedules/:id", handler.GetSchedule)

	req, _ := http.NewRequest("GET", "/api/schedules/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

// Test UpdateSchedule - 更新日程
func TestScheduleHandler_UpdateSchedule(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.PUT("/api/schedules/:id", handler.UpdateSchedule)

	// Create a schedule first
	schedule, _ := scheduleService.CreateSchedule(&service.CreateScheduleDTO{
		Title:     "更新测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	body := map[string]interface{}{
		"title": "更新后的标题",
		"color": "#ef4444",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/schedules/"+schedule.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test DeleteSchedule - 删除日程
func TestScheduleHandler_DeleteSchedule(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.DELETE("/api/schedules/:id", handler.DeleteSchedule)

	// Create a schedule first
	schedule, _ := scheduleService.CreateSchedule(&service.CreateScheduleDTO{
		Title:     "删除测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	req, _ := http.NewRequest("DELETE", "/api/schedules/"+schedule.ID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test MoveSchedule - 移动日程
func TestScheduleHandler_MoveSchedule(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.PUT("/api/schedules/:id/move", handler.MoveSchedule)

	// Create a schedule first
	schedule, _ := scheduleService.CreateSchedule(&service.CreateScheduleDTO{
		Title:     "移动测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	body := map[string]interface{}{
		"start_time": "2026-03-18T14:00:00Z",
		"end_time":   "2026-03-18T15:00:00Z",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/schedules/"+schedule.ID+"/move", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test MoveSchedule with invalid time format - 无效时间格式移动日程
func TestScheduleHandler_MoveSchedule_InvalidTimeFormat(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.PUT("/api/schedules/:id/move", handler.MoveSchedule)

	// Create a schedule first
	schedule, _ := scheduleService.CreateSchedule(&service.CreateScheduleDTO{
		Title:     "移动测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	body := map[string]interface{}{
		"start_time": "invalid-time",
		"end_time":   "2026-03-18T15:00:00Z",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/schedules/"+schedule.ID+"/move", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test GenerateWithAI - AI生成日程
func TestScheduleHandler_GenerateWithAI(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.POST("/api/schedules/generate", handler.GenerateWithAI)

	body := map[string]interface{}{
		"start_time": "09:00",
		"end_time":   "18:00",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/schedules/generate", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if _, ok := response["events"]; !ok {
		t.Error("expected 'events' key in response")
	}
}

// Test GenerateWithAI with default time - 使用默认时间生成日程
func TestScheduleHandler_GenerateWithAI_DefaultTime(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.POST("/api/schedules/generate", handler.GenerateWithAI)

	// Empty body - should use default times
	body := map[string]interface{}{}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/schedules/generate", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test different schedule types - 不同类型日程
func TestScheduleHandler_CreateSchedule_DifferentTypes(t *testing.T) {
	types := []struct {
		scheduleType string
		expectedOk   bool
	}{
		{"task", true},
		{"pomodoro", true},
		{"break", true},
		{"custom", true},
	}

	for _, tc := range types {
		t.Run(tc.scheduleType, func(t *testing.T) {
			scheduleService := createScheduleService()
			handler := NewScheduleHandler(scheduleService)
			router := setupTestRouter()
			router.POST("/api/schedules", handler.CreateSchedule)

			body := map[string]interface{}{
				"title":      tc.scheduleType + "日程",
				"start_time": "2026-03-18T09:00:00Z",
				"end_time":   "2026-03-18T10:00:00Z",
				"type":       tc.scheduleType,
			}
			jsonBody, _ := json.Marshal(body)

			req, _ := http.NewRequest("POST", "/api/schedules", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if tc.expectedOk && w.Code != http.StatusCreated {
				t.Errorf("type %s: expected status %d, got %d", tc.scheduleType, http.StatusCreated, w.Code)
			}
		})
	}
}

// Test CreateSchedule with task_id - 带任务ID创建日程
func TestScheduleHandler_CreateSchedule_WithTaskID(t *testing.T) {
	// Use shared repositories so the task can be found
	taskRepo := newMockTaskRepository()
	scheduleRepo := newMockScheduleRepositoryForService()
	scheduleService := service.NewScheduleService(scheduleRepo, taskRepo, nil)

	// Create a task in the same repository
	task := &model.Task{
		ID:        uuid.New().String(),
		Title:     "关联任务",
		Quadrant:  model.Quadrant1,
		Status:    model.StatusTodo,
		CreatedAt: time.Now(),
	}
	taskRepo.Create(task)

	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.POST("/api/schedules", handler.CreateSchedule)

	body := map[string]interface{}{
		"task_id":    task.ID,
		"start_time": "2026-03-18T09:00:00Z",
		"end_time":   "2026-03-18T10:00:00Z",
		"type":       "task",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/schedules", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["title"] != "关联任务" {
		t.Errorf("expected title to be task title '关联任务', got %v", response["title"])
	}
}

// ========== 补充测试用例 ==========

// Test GetSchedules with invalid date format - 无效日期格式获取日程
func TestScheduleHandler_GetSchedules_InvalidDateFormat(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.GET("/api/schedules", handler.GetSchedules)

	req, _ := http.NewRequest("GET", "/api/schedules?start=invalid-date", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

// Test UpdateSchedule not found - 更新不存在的日程
func TestScheduleHandler_UpdateSchedule_NotFound(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.PUT("/api/schedules/:id", handler.UpdateSchedule)

	body := map[string]interface{}{
		"title": "更新标题",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/schedules/"+uuid.New().String(), bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// Test UpdateSchedule with status change - 更新日程状态
func TestScheduleHandler_UpdateSchedule_WithStatusChange(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.PUT("/api/schedules/:id", handler.UpdateSchedule)

	// Create a schedule first
	schedule, _ := scheduleService.CreateSchedule(&service.CreateScheduleDTO{
		Title:     "状态测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	body := map[string]interface{}{
		"status": "completed",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/schedules/"+schedule.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify status changed
	updated, _ := scheduleService.GetSchedule(schedule.ID)
	if updated.Status != model.ScheduleStatusCompleted {
		t.Errorf("expected status completed, got %s", updated.Status)
	}
}

// Test UpdateSchedule with time change - 更新日程时间
func TestScheduleHandler_UpdateSchedule_WithTimeChange(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.PUT("/api/schedules/:id", handler.UpdateSchedule)

	// Create a schedule first
	schedule, _ := scheduleService.CreateSchedule(&service.CreateScheduleDTO{
		Title:     "时间更新测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	body := map[string]interface{}{
		"start_time": "2026-03-18T14:00:00Z",
		"end_time":   "2026-03-18T15:30:00Z",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/schedules/"+schedule.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test DeleteSchedule not found - 删除不存在的日程（幂等操作，应返回成功）
func TestScheduleHandler_DeleteSchedule_NotFound(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.DELETE("/api/schedules/:id", handler.DeleteSchedule)

	req, _ := http.NewRequest("DELETE", "/api/schedules/"+uuid.New().String(), nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	// Delete is idempotent - returns success even if not found
	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

// Test MoveSchedule not found - 移动不存在的日程
func TestScheduleHandler_MoveSchedule_NotFound(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.PUT("/api/schedules/:id/move", handler.MoveSchedule)

	body := map[string]interface{}{
		"start_time": "2026-03-18T14:00:00Z",
		"end_time":   "2026-03-18T15:00:00Z",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/schedules/"+uuid.New().String()+"/move", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// Test CreateSchedule with description - 创建带描述的日程
func TestScheduleHandler_CreateSchedule_WithDescription(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.POST("/api/schedules", handler.CreateSchedule)

	body := map[string]interface{}{
		"title":       "带描述的日程",
		"description": "这是日程的详细描述",
		"start_time":  "2026-03-18T09:00:00Z",
		"end_time":    "2026-03-18T10:00:00Z",
		"type":        "task",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/schedules", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

// Test CreateSchedule with invalid time format - 无效时间格式创建日程
func TestScheduleHandler_CreateSchedule_InvalidTimeFormat(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.POST("/api/schedules", handler.CreateSchedule)

	body := map[string]interface{}{
		"title":      "无效时间",
		"start_time": "invalid-time",
		"end_time":   "2026-03-18T10:00:00Z",
		"type":       "task",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/schedules", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

// Test CreateSchedule with custom color - 创建带自定义颜色的日程
func TestScheduleHandler_CreateSchedule_WithCustomColor(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.POST("/api/schedules", handler.CreateSchedule)

	body := map[string]interface{}{
		"title":      "自定义颜色日程",
		"start_time": "2026-03-18T09:00:00Z",
		"end_time":   "2026-03-18T10:00:00Z",
		"type":       "task",
		"color":      "#a855f7",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/schedules", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["color"] != "#a855f7" {
		t.Errorf("expected color '#a855f7', got %v", response["color"])
	}
}

// Test UpdateSchedule with all fields - 更新所有字段
func TestScheduleHandler_UpdateSchedule_AllFields(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.PUT("/api/schedules/:id", handler.UpdateSchedule)

	// Create a schedule first
	schedule, _ := scheduleService.CreateSchedule(&service.CreateScheduleDTO{
		Title:     "原始标题",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	body := map[string]interface{}{
		"title":       "更新后标题",
		"description": "更新后描述",
		"start_time":  "2026-03-18T14:00:00Z",
		"end_time":    "2026-03-18T16:00:00Z",
		"status":      "in_progress",
		"color":       "#22c55e",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("PUT", "/api/schedules/"+schedule.ID, bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	// Verify all fields updated
	updated, _ := scheduleService.GetSchedule(schedule.ID)
	if updated.Title != "更新后标题" {
		t.Errorf("expected title '更新后标题', got %s", updated.Title)
	}
	if updated.Status != model.ScheduleStatusInProgress {
		t.Errorf("expected status 'in_progress', got %s", updated.Status)
	}
}

// Test different schedule statuses - 不同日程状态
func TestScheduleHandler_ScheduleStatusTransitions(t *testing.T) {
	scheduleService := createScheduleService()

	// Create a schedule
	schedule, _ := scheduleService.CreateSchedule(&service.CreateScheduleDTO{
		Title:     "状态转换测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	// Initial status should be planned
	if schedule.Status != model.ScheduleStatusPlanned {
		t.Errorf("expected initial status 'planned', got %s", schedule.Status)
	}

	// Update to in_progress
	err := scheduleService.UpdateScheduleStatus(schedule.ID, model.ScheduleStatusInProgress)
	if err != nil {
		t.Errorf("failed to update status to in_progress: %v", err)
	}

	// Update to completed
	err = scheduleService.UpdateScheduleStatus(schedule.ID, model.ScheduleStatusCompleted)
	if err != nil {
		t.Errorf("failed to update status to completed: %v", err)
	}

	// Verify completed schedules are not editable
	event, _ := scheduleService.GetSchedule(schedule.ID)
	if event.Status != model.ScheduleStatusCompleted {
		t.Errorf("expected status 'completed', got %s", event.Status)
	}
}

// Test CreateSchedule with pomodoro type - 创建番茄钟类型日程
func TestScheduleHandler_CreateSchedule_PomodoroType(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.POST("/api/schedules", handler.CreateSchedule)

	body := map[string]interface{}{
		"title":      "番茄钟日程",
		"start_time": "2026-03-18T09:00:00Z",
		"end_time":   "2026-03-18T09:25:00Z",
		"type":       "pomodoro",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/schedules", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// Pomodoro type should have orange color by default
	if response["color"] != "#f59e0b" {
		t.Errorf("expected pomodoro color '#f59e0b', got %v", response["color"])
	}
}

// Test CreateSchedule with break type - 创建休息类型日程
func TestScheduleHandler_CreateSchedule_BreakType(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.POST("/api/schedules", handler.CreateSchedule)

	body := map[string]interface{}{
		"title":      "休息时间",
		"start_time": "2026-03-18T09:25:00Z",
		"end_time":   "2026-03-18T09:30:00Z",
		"type":       "break",
	}
	jsonBody, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/schedules", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// Break type should have green color by default
	if response["color"] != "#22c55e" {
		t.Errorf("expected break color '#22c55e', got %v", response["color"])
	}
}

// Test GetSchedules empty result - 空结果获取日程
func TestScheduleHandler_GetSchedules_EmptyResult(t *testing.T) {
	scheduleService := createScheduleService()
	handler := NewScheduleHandler(scheduleService)
	router := setupTestRouter()
	router.GET("/api/schedules", handler.GetSchedules)

	// Query a far future date range with no schedules
	req, _ := http.NewRequest("GET", "/api/schedules?start=2030-01-01&end=2030-01-31", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	events := response["events"].([]interface{})
	if len(events) != 0 {
		t.Errorf("expected empty events array, got %d events", len(events))
	}
}