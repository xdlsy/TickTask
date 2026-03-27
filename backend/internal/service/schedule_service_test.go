package service

import (
	"testing"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"time"
)

// mockScheduleRepo implements repository.ScheduleRepository for service testing
type mockScheduleRepo struct {
	schedules map[string]*model.Schedule
}

func newMockScheduleRepo() *mockScheduleRepo {
	return &mockScheduleRepo{
		schedules: make(map[string]*model.Schedule),
	}
}

func (m *mockScheduleRepo) Create(schedule *model.Schedule) error {
	m.schedules[schedule.ID] = schedule
	return nil
}

func (m *mockScheduleRepo) GetByID(id string) (*model.Schedule, error) {
	if schedule, ok := m.schedules[id]; ok {
		return schedule, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockScheduleRepo) GetByTimeRange(start, end time.Time) ([]model.Schedule, error) {
	var result []model.Schedule
	for _, schedule := range m.schedules {
		if (schedule.StartTime.After(start) || schedule.StartTime.Equal(start)) &&
			(schedule.StartTime.Before(end) || schedule.StartTime.Equal(end)) {
			result = append(result, *schedule)
		}
	}
	return result, nil
}

func (m *mockScheduleRepo) GetByTaskID(taskID string) ([]model.Schedule, error) {
	var result []model.Schedule
	for _, schedule := range m.schedules {
		if schedule.TaskID != nil && *schedule.TaskID == taskID {
			result = append(result, *schedule)
		}
	}
	return result, nil
}

func (m *mockScheduleRepo) GetByDate(date time.Time) ([]model.Schedule, error) {
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

func (m *mockScheduleRepo) Update(schedule *model.Schedule) error {
	m.schedules[schedule.ID] = schedule
	return nil
}

func (m *mockScheduleRepo) Delete(id string) error {
	delete(m.schedules, id)
	return nil
}

func (m *mockScheduleRepo) UpdateStatus(id string, status model.ScheduleStatus) error {
	if schedule, ok := m.schedules[id]; ok {
		schedule.Status = status
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockScheduleRepo) Move(id string, startTime, endTime time.Time) error {
	if schedule, ok := m.schedules[id]; ok {
		schedule.StartTime = startTime
		schedule.EndTime = endTime
		return nil
	}
	return repository.ErrNotFound
}

// mockTaskRepo implements repository.TaskRepository for service testing
type mockTaskRepoForSchedule struct {
	tasks map[string]*model.Task
}

func newMockTaskRepoForSchedule() *mockTaskRepoForSchedule {
	return &mockTaskRepoForSchedule{
		tasks: make(map[string]*model.Task),
	}
}

func (m *mockTaskRepoForSchedule) Create(task *model.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepoForSchedule) Update(task *model.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepoForSchedule) Delete(id string) error {
	delete(m.tasks, id)
	return nil
}

func (m *mockTaskRepoForSchedule) GetByID(id string) (*model.Task, error) {
	if task, ok := m.tasks[id]; ok {
		return task, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockTaskRepoForSchedule) GetAll() ([]model.Task, error) {
	var result []model.Task
	for _, task := range m.tasks {
		result = append(result, *task)
	}
	return result, nil
}

func (m *mockTaskRepoForSchedule) GetByStatus(status model.TaskStatus) ([]model.Task, error) {
	var result []model.Task
	for _, task := range m.tasks {
		if task.Status == status {
			result = append(result, *task)
		}
	}
	return result, nil
}

func (m *mockTaskRepoForSchedule) GetByQuadrant(quadrant model.Quadrant) ([]model.Task, error) {
	var result []model.Task
	for _, task := range m.tasks {
		if task.Quadrant == quadrant {
			result = append(result, *task)
		}
	}
	return result, nil
}

func (m *mockTaskRepoForSchedule) GetAllByQuadrant() (map[model.Quadrant][]model.Task, error) {
	result := make(map[model.Quadrant][]model.Task)
	for i := 1; i <= 4; i++ {
		result[model.Quadrant(i)] = []model.Task{}
	}
	for _, task := range m.tasks {
		result[task.Quadrant] = append(result[task.Quadrant], *task)
	}
	return result, nil
}

// createTestScheduleService creates a ScheduleService for testing
func createTestScheduleService() *ScheduleService {
	scheduleRepo := newMockScheduleRepo()
	taskRepo := newMockTaskRepoForSchedule()
	return NewScheduleService(scheduleRepo, taskRepo, nil)
}

// Test GetSchedules - 获取日程列表
func TestScheduleService_GetSchedules(t *testing.T) {
	svc := createTestScheduleService()

	// Create some schedules
	svc.CreateSchedule(&CreateScheduleDTO{
		Title:     "日程1",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})
	svc.CreateSchedule(&CreateScheduleDTO{
		Title:     "日程2",
		StartTime: "2026-03-18T11:00:00Z",
		EndTime:   "2026-03-18T12:00:00Z",
		Type:      "pomodoro",
	})

	start := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 3, 31, 23, 59, 59, 0, time.UTC)

	events, err := svc.GetSchedules(start, end)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

// Test CreateSchedule - 创建日程
func TestScheduleService_CreateSchedule(t *testing.T) {
	svc := createTestScheduleService()

	dto := &CreateScheduleDTO{
		Title:     "测试日程",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
		Color:     "#3b82f6",
	}

	schedule, err := svc.CreateSchedule(dto)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if schedule.Title != "测试日程" {
		t.Errorf("expected title '测试日程', got %s", schedule.Title)
	}

	if schedule.Type != model.ScheduleTypeTask {
		t.Errorf("expected type task, got %s", schedule.Type)
	}

	if schedule.Status != model.ScheduleStatusPlanned {
		t.Errorf("expected status planned, got %s", schedule.Status)
	}
}

// Test CreateSchedule with invalid time - 无效时间创建日程
func TestScheduleService_CreateSchedule_InvalidTime(t *testing.T) {
	svc := createTestScheduleService()

	dto := &CreateScheduleDTO{
		Title:     "测试日程",
		StartTime: "invalid-time",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	}

	_, err := svc.CreateSchedule(dto)
	if err == nil {
		t.Error("expected error for invalid start_time")
	}
}

// Test GetSchedule - 获取单个日程
func TestScheduleService_GetSchedule(t *testing.T) {
	svc := createTestScheduleService()

	// Create a schedule first
	created, _ := svc.CreateSchedule(&CreateScheduleDTO{
		Title:     "获取测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	// Get the schedule
	schedule, err := svc.GetSchedule(created.ID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if schedule.ID != created.ID {
		t.Errorf("expected ID %s, got %s", created.ID, schedule.ID)
	}
}

// Test GetSchedule not found - 获取不存在的日程
func TestScheduleService_GetSchedule_NotFound(t *testing.T) {
	svc := createTestScheduleService()

	_, err := svc.GetSchedule("non-existent-id")
	if err == nil {
		t.Error("expected error for non-existent schedule")
	}
}

// Test UpdateSchedule - 更新日程
func TestScheduleService_UpdateSchedule(t *testing.T) {
	svc := createTestScheduleService()

	// Create a schedule first
	created, _ := svc.CreateSchedule(&CreateScheduleDTO{
		Title:     "更新测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	// Update the schedule
	dto := &UpdateScheduleDTO{
		Title: "更新后的标题",
		Color: "#ef4444",
	}

	err := svc.UpdateSchedule(created.ID, dto)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify update
	schedule, _ := svc.GetSchedule(created.ID)
	if schedule.Title != "更新后的标题" {
		t.Errorf("expected title '更新后的标题', got %s", schedule.Title)
	}
	if schedule.Color != "#ef4444" {
		t.Errorf("expected color '#ef4444', got %s", schedule.Color)
	}
}

// Test UpdateSchedule with time - 更新日程时间
func TestScheduleService_UpdateSchedule_WithTime(t *testing.T) {
	svc := createTestScheduleService()

	// Create a schedule first
	created, _ := svc.CreateSchedule(&CreateScheduleDTO{
		Title:     "更新时间测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	// Update the schedule time
	dto := &UpdateScheduleDTO{
		StartTime: "2026-03-18T14:00:00Z",
		EndTime:   "2026-03-18T15:00:00Z",
	}

	err := svc.UpdateSchedule(created.ID, dto)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// Test DeleteSchedule - 删除日程
func TestScheduleService_DeleteSchedule(t *testing.T) {
	svc := createTestScheduleService()

	// Create a schedule first
	created, _ := svc.CreateSchedule(&CreateScheduleDTO{
		Title:     "删除测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	// Delete the schedule
	err := svc.DeleteSchedule(created.ID)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify deletion
	_, err = svc.GetSchedule(created.ID)
	if err == nil {
		t.Error("expected error for deleted schedule")
	}
}

// Test MoveSchedule - 移动日程
func TestScheduleService_MoveSchedule(t *testing.T) {
	svc := createTestScheduleService()

	// Create a schedule first
	created, _ := svc.CreateSchedule(&CreateScheduleDTO{
		Title:     "移动测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	// Move the schedule
	newStart := time.Date(2026, 3, 18, 14, 0, 0, 0, time.UTC)
	newEnd := time.Date(2026, 3, 18, 15, 0, 0, 0, time.UTC)

	err := svc.MoveSchedule(created.ID, newStart, newEnd)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify move
	schedule, _ := svc.GetSchedule(created.ID)
	if schedule.StartTime.Hour() != 14 {
		t.Errorf("expected start hour 14, got %d", schedule.StartTime.Hour())
	}
}

// Test UpdateScheduleStatus - 更新日程状态
func TestScheduleService_UpdateScheduleStatus(t *testing.T) {
	svc := createTestScheduleService()

	// Create a schedule first
	created, _ := svc.CreateSchedule(&CreateScheduleDTO{
		Title:     "状态测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	})

	// Update status
	err := svc.UpdateScheduleStatus(created.ID, model.ScheduleStatusCompleted)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify status
	schedule, _ := svc.GetSchedule(created.ID)
	if schedule.Status != model.ScheduleStatusCompleted {
		t.Errorf("expected status completed, got %s", schedule.Status)
	}
}

// Test different schedule types - 不同类型日程
func TestScheduleService_CreateSchedule_DifferentTypes(t *testing.T) {
	types := []struct {
		inputType   string
		expectedType model.ScheduleType
	}{
		{"task", model.ScheduleTypeTask},
		{"pomodoro", model.ScheduleTypePomodoro},
		{"break", model.ScheduleTypeBreak},
		{"custom", model.ScheduleTypeCustom},
	}

	for _, tc := range types {
		t.Run(tc.inputType, func(t *testing.T) {
			svc := createTestScheduleService()

			dto := &CreateScheduleDTO{
				Title:     tc.inputType + "日程",
				StartTime: "2026-03-18T09:00:00Z",
				EndTime:   "2026-03-18T10:00:00Z",
				Type:      tc.inputType,
			}

			schedule, err := svc.CreateSchedule(dto)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if schedule.Type != tc.expectedType {
				t.Errorf("expected type %s, got %s", tc.expectedType, schedule.Type)
			}
		})
	}
}

// Test default color - 默认颜色
func TestScheduleService_CreateSchedule_DefaultColor(t *testing.T) {
	svc := createTestScheduleService()

	// Without color specified
	dto := &CreateScheduleDTO{
		Title:     "默认颜色测试",
		StartTime: "2026-03-18T09:00:00Z",
		EndTime:   "2026-03-18T10:00:00Z",
		Type:      "task",
	}

	schedule, err := svc.CreateSchedule(dto)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should have default color for task type
	if schedule.Color != "#3b82f6" {
		t.Errorf("expected default color '#3b82f6', got %s", schedule.Color)
	}
}

// Test toEvent conversion - 转换为事件
func TestScheduleService_ToEvent(t *testing.T) {
	svc := createTestScheduleService()

	schedule := &model.Schedule{
		ID:        "test-id",
		Title:     "转换测试",
		StartTime: time.Date(2026, 3, 18, 9, 0, 0, 0, time.UTC),
		EndTime:   time.Date(2026, 3, 18, 10, 0, 0, 0, time.UTC),
		Type:      model.ScheduleTypeTask,
		Status:    model.ScheduleStatusPlanned,
		Color:     "#3b82f6",
	}

	event := svc.toEvent(schedule)

	if event.ID != schedule.ID {
		t.Errorf("expected ID %s, got %s", schedule.ID, event.ID)
	}

	if event.Title != schedule.Title {
		t.Errorf("expected title %s, got %s", schedule.Title, event.Title)
	}

	if event.AllDay != false {
		t.Error("expected allDay to be false")
	}

	if event.Editable != true {
		t.Error("expected editable to be true for planned status")
	}
}

// Test GenerateScheduleWithAI - AI生成日程
func TestScheduleService_GenerateScheduleWithAI(t *testing.T) {
	scheduleRepo := newMockScheduleRepo()
	taskRepo := newMockTaskRepoForSchedule()

	// Add some tasks
	taskRepo.Create(&model.Task{
		ID:        "task-1",
		Title:     "任务1",
		Quadrant:  model.Quadrant1,
		Status:    model.StatusTodo,
	})
	taskRepo.Create(&model.Task{
		ID:        "task-2",
		Title:     "任务2",
		Quadrant:  model.Quadrant2,
		Status:    model.StatusTodo,
	})

	svc := NewScheduleService(scheduleRepo, taskRepo, nil)

	events, err := svc.GenerateScheduleWithAI("09:00", "18:00")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Should generate events for tasks
	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}
}

// Test GenerateScheduleWithAI with invalid time format - 无效时间格式
func TestScheduleService_GenerateScheduleWithAI_InvalidTime(t *testing.T) {
	svc := createTestScheduleService()

	_, err := svc.GenerateScheduleWithAI("invalid", "18:00")
	if err == nil {
		t.Error("expected error for invalid time format")
	}
}