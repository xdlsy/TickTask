package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"ticktask/internal/ai"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"testing"
	"time"
)

func TestAIService_IsConfigured(t *testing.T) {
	// Without client
	service := &AIService{client: nil}
	if service.IsConfigured() {
		t.Error("expected IsConfigured to be false when client is nil")
	}

	// With client
	service = &AIService{client: ai.NewOpenAIClient("test-key", "", "")}
	if !service.IsConfigured() {
		t.Error("expected IsConfigured to be true when client is set")
	}
}

func TestAIService_CalculateQuadrant(t *testing.T) {
	tests := []struct {
		important bool
		urgent    bool
		expected  int
	}{
		{true, true, 1},
		{true, false, 2},
		{false, true, 3},
		{false, false, 4},
	}

	for _, tt := range tests {
		result := calculateQuadrant(tt.important, tt.urgent)
		if result != tt.expected {
			t.Errorf("calculateQuadrant(%v, %v) = %d, expected %d",
				tt.important, tt.urgent, result, tt.expected)
		}
	}
}

func TestAIService_ParseClassifyResponse(t *testing.T) {
	jsonResponse := `{"important": true, "urgent": false, "reason": "Test reason"}`

	result, err := parseClassifyResponse(jsonResponse)
	if err != nil {
		t.Fatalf("parseClassifyResponse failed: %v", err)
	}

	if result.Important != true {
		t.Errorf("expected Important to be true, got %v", result.Important)
	}

	if result.Urgent != false {
		t.Errorf("expected Urgent to be false, got %v", result.Urgent)
	}

	if result.Reason != "Test reason" {
		t.Errorf("expected Reason 'Test reason', got %s", result.Reason)
	}
}

func TestAIService_ParseClassifyResponse_WithExtraContent(t *testing.T) {
	// Test with markdown code blocks
	jsonResponse := "```json\n{\"important\": true, \"urgent\": true, \"reason\": \"Test\"}\n```"

	result, err := parseClassifyResponse(jsonResponse)
	if err != nil {
		t.Fatalf("parseClassifyResponse failed: %v", err)
	}

	if result.Important != true {
		t.Errorf("expected Important to be true")
	}
}

func TestAIService_ParseScheduleResponse(t *testing.T) {
	jsonResponse := `{
		"schedule": [
			{
				"task_id": "task-1",
				"title": "Task 1",
				"start_time": "09:00",
				"end_time": "10:30",
				"pomodoro_count": 3
			}
		]
	}`

	result, err := parseScheduleResponse(jsonResponse)
	if err != nil {
		t.Fatalf("parseScheduleResponse failed: %v", err)
	}

	if len(result.Schedule) != 1 {
		t.Fatalf("expected 1 schedule item, got %d", len(result.Schedule))
	}

	if result.Schedule[0].TaskID != "task-1" {
		t.Errorf("expected task_id 'task-1', got %s", result.Schedule[0].TaskID)
	}

	if result.Schedule[0].PomodoroCount != 3 {
		t.Errorf("expected pomodoro_count 3, got %d", result.Schedule[0].PomodoroCount)
	}
}

func TestAIService_ParsePriorityResponse(t *testing.T) {
	jsonResponse := `{"priority_order": ["task-1", "task-2", "task-3"]}`

	result, err := parsePriorityResponse(jsonResponse)
	if err != nil {
		t.Fatalf("parsePriorityResponse failed: %v", err)
	}

	if len(result.PriorityOrder) != 3 {
		t.Fatalf("expected 3 items, got %d", len(result.PriorityOrder))
	}

	if result.PriorityOrder[0] != "task-1" {
		t.Errorf("expected first item 'task-1', got %s", result.PriorityOrder[0])
	}
}

func TestAIService_ExtractJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    `{"key": "value"}`,
			expected: `{"key": "value"}`,
		},
		{
			input:    `Here is the result: {"key": "value"} and more text`,
			expected: `{"key": "value"}`,
		},
		{
			input:    `Some text before {"a": 1, "b": 2} some text after`,
			expected: `{"a": 1, "b": 2}`,
		},
	}

	for _, tt := range tests {
		result := extractJSON(tt.input)
		if result != tt.expected {
			t.Errorf("extractJSON(%s) = %s, expected %s", tt.input, result, tt.expected)
		}
	}
}

// Integration test with mock HTTP server
func TestAIService_ChatCompletion(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Error("missing or wrong Authorization header")
		}

		// Return mock response
		resp := ai.ChatResponse{
			Choices: []ai.Choice{
				{
					Message: ai.Message{
						Role:    "assistant",
						Content: `{"important": true, "urgent": false, "reason": "Test"}`,
					},
				},
			},
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Create client with mock server URL
	client := ai.NewOpenAIClient("test-key", server.URL, "gpt-4o-mini")

	// Make request
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	response, err := client.ChatCompletion(ctx, "Test prompt")
	if err != nil {
		t.Fatalf("ChatCompletion failed: %v", err)
	}

	if response == "" {
		t.Error("expected non-empty response")
	}
}

func TestAIService_ChatCompletion_Error(t *testing.T) {
	// Create mock server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		resp := ai.ChatResponse{
			Error: &ai.APIError{
				Message: "Invalid API key",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := ai.NewOpenAIClient("invalid-key", server.URL, "gpt-4o-mini")

	ctx := context.Background()
	_, err := client.ChatCompletion(ctx, "Test prompt")
	if err == nil {
		t.Error("expected error for invalid API key")
	}
}

func TestAIService_ChatCompletion_Timeout(t *testing.T) {
	// Create mock server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := ai.NewOpenAIClient("test-key", server.URL, "gpt-4o-mini")

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := client.ChatCompletion(ctx, "Test prompt")
	if err == nil {
		t.Error("expected timeout error")
	}
}

// mockLLMClient implements ai.LLMClient for service testing
type mockLLMClient struct {
	response string
	err      error
}

func (m *mockLLMClient) ChatCompletion(ctx context.Context, prompt string) (string, error) {
	return m.response, m.err
}

// mockTaskRepository is a minimal in-memory repo for service tests
type mockTaskRepo struct {
	tasks map[string]*model.Task
}

func newMockTaskRepo() *mockTaskRepo {
	return &mockTaskRepo{tasks: make(map[string]*model.Task)}
}

func (m *mockTaskRepo) Create(task *model.Task) error        { m.tasks[task.ID] = task; return nil }
func (m *mockTaskRepo) Update(task *model.Task) error        { m.tasks[task.ID] = task; return nil }
func (m *mockTaskRepo) Delete(id string) error               { delete(m.tasks, id); return nil }
func (m *mockTaskRepo) GetByID(id string) (*model.Task, error) {
	if t, ok := m.tasks[id]; ok { return t, nil }
	return nil, repository.ErrNotFound
}
func (m *mockTaskRepo) GetAll() ([]model.Task, error) {
	result := make([]model.Task, 0, len(m.tasks))
	for _, t := range m.tasks { result = append(result, *t) }
	return result, nil
}
func (m *mockTaskRepo) GetByStatus(status model.TaskStatus) ([]model.Task, error) {
	var result []model.Task
	for _, t := range m.tasks {
		if t.Status == status { result = append(result, *t) }
	}
	return result, nil
}
func (m *mockTaskRepo) GetByQuadrant(quadrant model.Quadrant) ([]model.Task, error) { return nil, nil }
func (m *mockTaskRepo) GetAllByQuadrant() (map[model.Quadrant][]model.Task, error) { return nil, nil }

func TestAIService_RescheduleAfterInterrupt_NotConfigured(t *testing.T) {
	taskRepo := newMockTaskRepo()
	service := &AIService{client: nil, taskRepo: taskRepo}

	ctx := context.Background()
	_, err := service.RescheduleAfterInterrupt(ctx, "task-1", 10, 25, "meeting", "14:00", "18:00")
	if err == nil {
		t.Error("expected error when AI not configured")
	}
	if err.Error() != "AI service not configured" {
		t.Errorf("expected 'AI service not configured', got '%s'", err.Error())
	}
}

func TestAIService_RescheduleAfterInterrupt_Success(t *testing.T) {
	taskRepo := newMockTaskRepo()

	// Add the interrupted task
	taskRepo.Create(&model.Task{
		ID: "task-1", Title: "代码审查", Status: model.StatusTodo,
		EstimatedTime: 25, Quadrant: model.Quadrant2,
	})

	// Add another pending task
	taskRepo.Create(&model.Task{
		ID: "task-2", Title: "写周报", Status: model.StatusTodo,
		EstimatedTime: 30, Quadrant: model.Quadrant3,
	})

	mockResponse := `{
		"adjusted_schedule": [
			{
				"task_id": "task-1",
				"title": "代码审查（调整后）",
				"start_time": "14:00",
				"end_time": "14:15",
				"adjustment": "shortened",
				"reason": "被打断后剩余15分钟"
			},
			{
				"task_id": "task-2",
				"title": "写周报",
				"start_time": "14:15",
				"end_time": "14:45",
				"adjustment": "postponed",
				"reason": "前序任务被打断导致推迟"
			}
		],
		"summary": "调整了2个任务的排程"
	}`

	mockClient := &mockLLMClient{response: mockResponse, err: nil}
	service := &AIService{client: mockClient, taskRepo: taskRepo}

	ctx := context.Background()
	result, err := service.RescheduleAfterInterrupt(ctx, "task-1", 10, 25, "同事临时会议", "14:00", "18:00")
	if err != nil {
		t.Fatalf("RescheduleAfterInterrupt failed: %v", err)
	}

	if result.Summary != "调整了2个任务的排程" {
		t.Errorf("expected summary '调整了2个任务的排程', got '%s'", result.Summary)
	}

	if len(result.AdjustedSchedule) != 2 {
		t.Fatalf("expected 2 adjusted items, got %d", len(result.AdjustedSchedule))
	}

	if result.AdjustedSchedule[0].Adjustment != "shortened" {
		t.Errorf("expected first adjustment 'shortened', got '%s'", result.AdjustedSchedule[0].Adjustment)
	}

	if result.AdjustedSchedule[1].Adjustment != "postponed" {
		t.Errorf("expected second adjustment 'postponed', got '%s'", result.AdjustedSchedule[1].Adjustment)
	}

	if result.AdjustedSchedule[0].TaskID != "task-1" {
		t.Errorf("expected first task_id 'task-1', got '%s'", result.AdjustedSchedule[0].TaskID)
	}
}

func TestAIService_RescheduleAfterInterrupt_LLMError(t *testing.T) {
	taskRepo := newMockTaskRepo()
	taskRepo.Create(&model.Task{
		ID: "task-1", Title: "测试任务", Status: model.StatusTodo,
		EstimatedTime: 25,
	})

	mockClient := &mockLLMClient{response: "", err: errors.New("LLM service timeout")}
	service := &AIService{client: mockClient, taskRepo: taskRepo}

	ctx := context.Background()
	_, err := service.RescheduleAfterInterrupt(ctx, "task-1", 10, 25, "打断", "14:00", "18:00")
	if err == nil {
		t.Error("expected error when LLM fails")
	}
}

func TestAIService_RescheduleAfterInterrupt_MalformedJSON(t *testing.T) {
	taskRepo := newMockTaskRepo()
	taskRepo.Create(&model.Task{
		ID: "task-1", Title: "测试", Status: model.StatusTodo,
		EstimatedTime: 25,
	})

	// Return non-JSON text
	mockClient := &mockLLMClient{response: "Sorry, I cannot help with that.", err: nil}
	service := &AIService{client: mockClient, taskRepo: taskRepo}

	ctx := context.Background()
	_, err := service.RescheduleAfterInterrupt(ctx, "task-1", 10, 25, "meeting", "14:00", "18:00")
	if err == nil {
		t.Error("expected parse error for malformed response")
	}
}

func TestAIService_RescheduleAfterInterrupt_NoRemainingTasks(t *testing.T) {
	taskRepo := newMockTaskRepo()

	mockResponse := `{
		"adjusted_schedule": [
			{
				"task_id": "task-1",
				"title": "唯一的任务（调整后）",
				"start_time": "14:00",
				"end_time": "14:15",
				"adjustment": "shortened",
				"reason": "被打断后调整剩余时间"
			}
		],
		"summary": "无其他任务，仅调整当前任务时长"
	}`

	mockClient := &mockLLMClient{response: mockResponse, err: nil}
	service := &AIService{client: mockClient, taskRepo: taskRepo}

	ctx := context.Background()
	result, err := service.RescheduleAfterInterrupt(ctx, "task-1", 5, 20, "紧急会议", "15:00", "18:00")
	if err != nil {
		t.Fatalf("RescheduleAfterInterrupt failed: %v", err)
	}

	if len(result.AdjustedSchedule) != 1 {
		t.Errorf("expected 1 adjusted item, got %d", len(result.AdjustedSchedule))
	}
}

func TestAIService_RescheduleAfterInterrupt_JSONWithMarkdown(t *testing.T) {
	taskRepo := newMockTaskRepo()

	// Response wrapped in markdown code fences (AI sometimes does this)
	markdownResponse := "```json\n{\n  \"adjusted_schedule\": [],\n  \"summary\": \"无需调整，当前没有其他待办任务\"\n}\n```"

	mockClient := &mockLLMClient{response: markdownResponse, err: nil}
	service := &AIService{client: mockClient, taskRepo: taskRepo}

	ctx := context.Background()
	result, err := service.RescheduleAfterInterrupt(ctx, "task-x", 0, 30, "电话", "10:00", "18:00")
	if err != nil {
		t.Fatalf("RescheduleAfterInterrupt with markdown JSON failed: %v", err)
	}

	if result.Summary != "无需调整，当前没有其他待办任务" {
		t.Errorf("expected summary, got '%s'", result.Summary)
	}
}

func TestAIService_GenerateDailySchedule_NotConfigured(t *testing.T) {
	taskRepo := newMockTaskRepo()
	service := &AIService{client: nil, taskRepo: taskRepo}

	ctx := context.Background()
	_, err := service.GenerateDailySchedule(ctx, "09:00", "18:00", nil)
	if err == nil {
		t.Error("expected error when AI not configured")
	}
}

func TestAIService_GenerateDailySchedule_EmptyTasks(t *testing.T) {
	taskRepo := newMockTaskRepo()
	mockClient := &mockLLMClient{response: "", err: nil}
	service := &AIService{client: mockClient, taskRepo: taskRepo}

	ctx := context.Background()
	result, err := service.GenerateDailySchedule(ctx, "09:00", "18:00", &model.PomodoroSettings{
		WorkDuration: 1500, ShortBreakDuration: 300, LongBreakAfter: 4,
	})
	if err != nil {
		t.Fatalf("GenerateDailySchedule failed: %v", err)
	}

	if len(result.Schedule) != 0 {
		t.Errorf("expected empty schedule when no tasks exist, got %d items", len(result.Schedule))
	}
}

func TestAIService_GetPrioritySuggestions_NotConfigured(t *testing.T) {
	taskRepo := newMockTaskRepo()
	service := &AIService{client: nil, taskRepo: taskRepo}

	ctx := context.Background()
	_, err := service.GetPrioritySuggestions(ctx)
	if err == nil {
		t.Error("expected error when AI not configured")
	}
}

func TestAIService_GetDailyInsights_NotConfigured(t *testing.T) {
	taskRepo := newMockTaskRepo()
	service := &AIService{client: nil, taskRepo: taskRepo}

	ctx := context.Background()
	_, err := service.GetDailyInsights(ctx, "2026-05-26", 8, 200, 3, 2, "")
	if err == nil {
		t.Error("expected error when AI not configured")
	}
}