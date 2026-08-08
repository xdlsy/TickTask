package service

import (
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"testing"
	"time"
)

// Mock TaskRepository
type MockTaskRepository struct {
	tasks    map[string]*model.Task
	createFn func(task *model.Task) error
	updateFn func(task *model.Task) error
	deleteFn func(id string) error
}

func NewMockTaskRepository() *MockTaskRepository {
	return &MockTaskRepository{
		tasks: make(map[string]*model.Task),
	}
}

func (m *MockTaskRepository) Create(task *model.Task) error {
	if m.createFn != nil {
		return m.createFn(task)
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *MockTaskRepository) Update(task *model.Task) error {
	if m.updateFn != nil {
		return m.updateFn(task)
	}
	m.tasks[task.ID] = task
	return nil
}

func (m *MockTaskRepository) Delete(id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(id)
	}
	delete(m.tasks, id)
	return nil
}

func (m *MockTaskRepository) GetByID(id string) (*model.Task, error) {
	if task, ok := m.tasks[id]; ok {
		return task, nil
	}
	return nil, repository.ErrNotFound
}

func (m *MockTaskRepository) GetAll() ([]model.Task, error) {
	var result []model.Task
	for _, task := range m.tasks {
		result = append(result, *task)
	}
	return result, nil
}

func (m *MockTaskRepository) GetByStatus(status model.TaskStatus) ([]model.Task, error) {
	var result []model.Task
	for _, task := range m.tasks {
		if task.Status == status {
			result = append(result, *task)
		}
	}
	return result, nil
}

func (m *MockTaskRepository) GetByQuadrant(quadrant model.Quadrant) ([]model.Task, error) {
	var result []model.Task
	for _, task := range m.tasks {
		if task.Quadrant == quadrant {
			result = append(result, *task)
		}
	}
	return result, nil
}

func (m *MockTaskRepository) GetAllByQuadrant() (map[model.Quadrant][]model.Task, error) {
	result := make(map[model.Quadrant][]model.Task)
	for i := 1; i <= 4; i++ {
		result[model.Quadrant(i)] = []model.Task{}
	}
	for _, task := range m.tasks {
		result[task.Quadrant] = append(result[task.Quadrant], *task)
	}
	return result, nil
}

func (m *MockTaskRepository) GetCompletedTasksInRange(start, end time.Time) ([]*model.Task, error) {
	var result []*model.Task
	for _, task := range m.tasks {
		if task.Status == model.StatusCompleted && task.CompletedAt != nil &&
			!task.CompletedAt.Before(start) && task.CompletedAt.Before(end) {
			result = append(result, task)
		}
	}
	return result, nil
}

// Mock AnalyticsRepository
type MockAnalyticsRepository struct {
	createdTasks    int
	completedTasks  int
	focusTime       int
	pomodoros       int
}

func NewMockAnalyticsRepository() *MockAnalyticsRepository {
	return &MockAnalyticsRepository{}
}

func (m *MockAnalyticsRepository) GetDailyStats(date time.Time) (*model.DailyStats, error) {
	return &model.DailyStats{}, nil
}

func (m *MockAnalyticsRepository) GetDailyStatsRange(start, end time.Time) ([]model.DailyStats, error) {
	return []model.DailyStats{}, nil
}

func (m *MockAnalyticsRepository) CreateDailyStats(stats *model.DailyStats) error {
	return nil
}

func (m *MockAnalyticsRepository) UpdateDailyStats(stats *model.DailyStats) error {
	return nil
}

func (m *MockAnalyticsRepository) IncrementCompletedPomodoros(date time.Time) error {
	m.pomodoros++
	return nil
}

func (m *MockAnalyticsRepository) IncrementFocusTime(date time.Time, seconds int) error {
	m.focusTime += seconds
	return nil
}

func (m *MockAnalyticsRepository) IncrementCompletedTasks(date time.Time) error {
	m.completedTasks++
	return nil
}

func (m *MockAnalyticsRepository) IncrementCreatedTasks(date time.Time) error {
	m.createdTasks++
	return nil
}

// Mock SettingRepository
type MockSettingRepository struct {
	pomodoroSettings *model.PomodoroSettings
	aiSettings       *model.AISettings
}

func NewMockSettingRepository() *MockSettingRepository {
	return &MockSettingRepository{
		pomodoroSettings: model.DefaultPomodoroSettings(),
		aiSettings:       model.DefaultAISettings(),
	}
}

func (m *MockSettingRepository) Get(key string) (*model.Setting, error) {
	return nil, repository.ErrNotFound
}

func (m *MockSettingRepository) Set(key, value string) error {
	return nil
}

func (m *MockSettingRepository) GetPomodoroSettings() (*model.PomodoroSettings, error) {
	return m.pomodoroSettings, nil
}

func (m *MockSettingRepository) UpdatePomodoroSettings(settings *model.PomodoroSettings) error {
	m.pomodoroSettings = settings
	return nil
}

func (m *MockSettingRepository) GetAISettings() (*model.AISettings, error) {
	return m.aiSettings, nil
}

func (m *MockSettingRepository) UpdateAISettings(settings *model.AISettings) error {
	m.aiSettings = settings
	return nil
}

// MigrateLegacyAPIKey satisfies the Task 2 SettingRepository interface
// extension. TaskService never invokes it; stubbed for interface conformance.
func (m *MockSettingRepository) MigrateLegacyAPIKey() error {
	return nil
}

// Tests

func TestTaskService_CreateTask(t *testing.T) {
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()

	service := NewTaskService(taskRepo, analyticsRepo, settingRepo, NewMockSessionRepository())

	req := CreateTaskRequest{
		Title:       "Test Task",
		Description: "Test Description",
		Quadrant:    model.Quadrant1,
		IsImportant: true,
		IsUrgent:    true,
	}

	task, err := service.CreateTask(req)
	if err != nil {
		t.Fatalf("CreateTask failed: %v", err)
	}

	if task.Title != req.Title {
		t.Errorf("expected title %s, got %s", req.Title, task.Title)
	}

	if task.Status != model.StatusTodo {
		t.Errorf("expected status %s, got %s", model.StatusTodo, task.Status)
	}

	if analyticsRepo.createdTasks != 1 {
		t.Errorf("expected createdTasks to be 1, got %d", analyticsRepo.createdTasks)
	}
}

func TestTaskService_UpdateTask(t *testing.T) {
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()

	service := NewTaskService(taskRepo, analyticsRepo, settingRepo, NewMockSessionRepository())

	// Create a task first
	task, _ := service.CreateTask(CreateTaskRequest{
		Title:    "Original Title",
		Quadrant: model.Quadrant2,
	})

	// Update the task
	newTitle := "Updated Title"
	err := service.UpdateTask(task.ID, UpdateTaskRequest{
		Title: &newTitle,
	})
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Verify the update
	updatedTask, _ := taskRepo.GetByID(task.ID)
	if updatedTask.Title != newTitle {
		t.Errorf("expected title %s, got %s", newTitle, updatedTask.Title)
	}
}

func TestTaskService_DeleteTask(t *testing.T) {
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()

	service := NewTaskService(taskRepo, analyticsRepo, settingRepo, NewMockSessionRepository())

	task, _ := service.CreateTask(CreateTaskRequest{
		Title:    "Task to Delete",
		Quadrant: model.Quadrant1,
	})

	err := service.DeleteTask(task.ID)
	if err != nil {
		t.Fatalf("DeleteTask failed: %v", err)
	}

	// Verify deletion
	_, err = taskRepo.GetByID(task.ID)
	if err == nil {
		t.Error("expected task to be deleted")
	}
}

func TestTaskService_MoveTask(t *testing.T) {
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()

	service := NewTaskService(taskRepo, analyticsRepo, settingRepo, NewMockSessionRepository())

	task, _ := service.CreateTask(CreateTaskRequest{
		Title:    "Move Test",
		Quadrant: model.Quadrant1,
	})

	// Move to quadrant 2
	err := service.MoveTask(task.ID, model.Quadrant2)
	if err != nil {
		t.Fatalf("MoveTask failed: %v", err)
	}

	movedTask, _ := taskRepo.GetByID(task.ID)

	// Verify quadrant
	if movedTask.Quadrant != model.Quadrant2 {
		t.Errorf("expected quadrant %d, got %d", model.Quadrant2, movedTask.Quadrant)
	}

	// Verify IsImportant and IsUrgent are updated correctly
	if !movedTask.IsImportant {
		t.Error("expected IsImportant to be true for quadrant 2")
	}
	if movedTask.IsUrgent {
		t.Error("expected IsUrgent to be false for quadrant 2")
	}
}

func TestTaskService_CalculateQuadrant(t *testing.T) {
	service := &TaskService{}

	tests := []struct {
		important bool
		urgent    bool
		expected  model.Quadrant
	}{
		{true, true, model.Quadrant1},
		{true, false, model.Quadrant2},
		{false, true, model.Quadrant3},
		{false, false, model.Quadrant4},
	}

	for _, tt := range tests {
		result := service.CalculateQuadrant(tt.important, tt.urgent)
		if result != tt.expected {
			t.Errorf("CalculateQuadrant(%v, %v) = %d, expected %d",
				tt.important, tt.urgent, result, tt.expected)
		}
	}
}

func TestTaskService_GetTasksByQuadrant(t *testing.T) {
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()

	service := NewTaskService(taskRepo, analyticsRepo, settingRepo, NewMockSessionRepository())

	// Create tasks in different quadrants
	service.CreateTask(CreateTaskRequest{Title: "Q1 Task", Quadrant: model.Quadrant1})
	service.CreateTask(CreateTaskRequest{Title: "Q2 Task", Quadrant: model.Quadrant2})
	service.CreateTask(CreateTaskRequest{Title: "Q1 Task 2", Quadrant: model.Quadrant1})

	result, err := service.GetTasksByQuadrant()
	if err != nil {
		t.Fatalf("GetTasksByQuadrant failed: %v", err)
	}

	if len(result[model.Quadrant1]) != 2 {
		t.Errorf("expected 2 tasks in quadrant 1, got %d", len(result[model.Quadrant1]))
	}

	if len(result[model.Quadrant2]) != 1 {
		t.Errorf("expected 1 task in quadrant 2, got %d", len(result[model.Quadrant2]))
	}

	if len(result[model.Quadrant3]) != 0 {
		t.Errorf("expected 0 tasks in quadrant 3, got %d", len(result[model.Quadrant3]))
	}
}

func TestTaskService_MarkCompleted(t *testing.T) {
	taskRepo := NewMockTaskRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()

	service := NewTaskService(taskRepo, analyticsRepo, settingRepo, NewMockSessionRepository())

	task, _ := service.CreateTask(CreateTaskRequest{
		Title:    "Complete Me",
		Quadrant: model.Quadrant1,
	})

	// Mark as completed
	completed := model.StatusCompleted
	err := service.UpdateTask(task.ID, UpdateTaskRequest{
		Status: &completed,
	})
	if err != nil {
		t.Fatalf("UpdateTask failed: %v", err)
	}

	// Verify completed status
	updatedTask, _ := taskRepo.GetByID(task.ID)
	if updatedTask.Status != model.StatusCompleted {
		t.Errorf("expected status %s, got %s", model.StatusCompleted, updatedTask.Status)
	}

	if updatedTask.CompletedAt == nil {
		t.Error("expected CompletedAt to be set")
	}

	if analyticsRepo.completedTasks != 1 {
		t.Errorf("expected completedTasks to be 1, got %d", analyticsRepo.completedTasks)
	}
}

// --- Pomodoro Enrichment Tests ---

func TestComputePomodoroStatus(t *testing.T) {
	tests := []struct {
		name      string
		planned   int
		completed int
		expected  string
	}{
		{"no estimate", 0, 0, "not_started"},
		{"planned but not started", 4, 0, "not_started"},
		{"in progress", 4, 2, "in_progress"},
		{"exactly completed", 4, 4, "completed"},
		{"exceeded", 4, 6, "exceeded"},
		{"no estimate with sessions", 0, 3, "not_started"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := computePomodoroStatus(tt.planned, tt.completed)
			if result != tt.expected {
				t.Errorf("computePomodoroStatus(%d, %d) = %q, want %q", tt.planned, tt.completed, result, tt.expected)
			}
		})
	}
}

func TestTaskService_GetTaskResponse(t *testing.T) {
	taskRepo := NewMockTaskRepository()
	sessionRepo := NewMockSessionRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()

	svc := NewTaskService(taskRepo, analyticsRepo, settingRepo, sessionRepo)

	// Create task with estimate
	task, _ := svc.CreateTask(CreateTaskRequest{
		Title:         "Test Task",
		Quadrant:      model.Quadrant1,
		EstimatedTime: 100, // 100 min / 25 min = 4 planned
	})

	// Seed completed sessions
	taskID := task.ID
	sessionRepo.sessions["s1"] = &model.PomodoroSession{
		ID:        "s1",
		TaskID:    &taskID,
		Type:      model.SessionWork,
		Status:    model.SessionCompleted,
		StartTime: time.Now(),
	}
	sessionRepo.sessions["s2"] = &model.PomodoroSession{
		ID:        "s2",
		TaskID:    &taskID,
		Type:      model.SessionWork,
		Status:    model.SessionCompleted,
		StartTime: time.Now(),
	}

	resp, err := svc.GetTaskResponse(task.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PlannedPomodoros != 4 {
		t.Errorf("expected PlannedPomodoros 4, got %d", resp.PlannedPomodoros)
	}
	if resp.CompletedPomodoros != 2 {
		t.Errorf("expected CompletedPomodoros 2, got %d", resp.CompletedPomodoros)
	}
	if resp.PomodoroStatus != "in_progress" {
		t.Errorf("expected PomodoroStatus in_progress, got %s", resp.PomodoroStatus)
	}
}

func TestTaskService_GetTaskResponse_NoEstimate(t *testing.T) {
	taskRepo := NewMockTaskRepository()
	sessionRepo := NewMockSessionRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()

	svc := NewTaskService(taskRepo, analyticsRepo, settingRepo, sessionRepo)

	task, _ := svc.CreateTask(CreateTaskRequest{
		Title:    "No Estimate Task",
		Quadrant: model.Quadrant2,
	})

	resp, err := svc.GetTaskResponse(task.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if resp.PlannedPomodoros != 0 {
		t.Errorf("expected PlannedPomodoros 0, got %d", resp.PlannedPomodoros)
	}
	if resp.PomodoroStatus != "not_started" {
		t.Errorf("expected PomodoroStatus not_started, got %s", resp.PomodoroStatus)
	}
}

func TestTaskService_GetAllTaskResponses(t *testing.T) {
	taskRepo := NewMockTaskRepository()
	sessionRepo := NewMockSessionRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()

	svc := NewTaskService(taskRepo, analyticsRepo, settingRepo, sessionRepo)

	svc.CreateTask(CreateTaskRequest{Title: "Task 1", Quadrant: model.Quadrant1, EstimatedTime: 50})
	svc.CreateTask(CreateTaskRequest{Title: "Task 2", Quadrant: model.Quadrant2})

	responses, err := svc.GetAllTaskResponses()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(responses) != 2 {
		t.Fatalf("expected 2 responses, got %d", len(responses))
	}

	found := false
	for _, r := range responses {
		if r.Title == "Task 1" {
			found = true
			if r.PlannedPomodoros != 2 {
				t.Errorf("expected PlannedPomodoros 2, got %d", r.PlannedPomodoros)
			}
		}
	}
	if !found {
		t.Error("Task 1 not found in responses")
	}
}

func TestTaskService_GetTasksByQuadrantResponse(t *testing.T) {
	taskRepo := NewMockTaskRepository()
	sessionRepo := NewMockSessionRepository()
	analyticsRepo := NewMockAnalyticsRepository()
	settingRepo := NewMockSettingRepository()

	svc := NewTaskService(taskRepo, analyticsRepo, settingRepo, sessionRepo)

	svc.CreateTask(CreateTaskRequest{Title: "Q1 Task", Quadrant: model.Quadrant1, EstimatedTime: 25})
	svc.CreateTask(CreateTaskRequest{Title: "Q2 Task", Quadrant: model.Quadrant2, EstimatedTime: 60})

	responses, err := svc.GetTasksByQuadrantResponse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(responses) != 4 {
		t.Errorf("expected 4 quadrant keys, got %d", len(responses))
	}

	q1Tasks, ok := responses["1"]
	if !ok {
		t.Fatal("expected quadrant 1 key")
	}
	if len(q1Tasks) != 1 {
		t.Errorf("expected 1 task in Q1, got %d", len(q1Tasks))
	}
	if q1Tasks[0].PlannedPomodoros != 1 {
		t.Errorf("expected PlannedPomodoros 1, got %d", q1Tasks[0].PlannedPomodoros)
	}

	q2Tasks := responses["2"]
	if q2Tasks[0].PlannedPomodoros != 3 {
		t.Errorf("expected PlannedPomodoros 3, got %d", q2Tasks[0].PlannedPomodoros)
	}
}