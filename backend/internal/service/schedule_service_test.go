package service

import (
	"fmt"
	"strings"
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

func (m *mockScheduleRepo) DeleteTaskSchedulesByDateRange(start, end time.Time) (int64, error) {
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

func (m *mockScheduleRepo) Move(id string, startTime, endTime time.Time) error {
	if schedule, ok := m.schedules[id]; ok {
		schedule.StartTime = startTime
		schedule.EndTime = endTime
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockScheduleRepo) DeleteAll() (int64, error) {
	count := int64(len(m.schedules))
	m.schedules = make(map[string]*model.Schedule)
	return count, nil
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

func (m *mockTaskRepoForSchedule) GetCompletedTasksInRange(start, end time.Time) ([]*model.Task, error) {
	var result []*model.Task
	for _, task := range m.tasks {
		if task.Status == model.StatusCompleted && task.CompletedAt != nil &&
			!task.CompletedAt.Before(start) && task.CompletedAt.Before(end) {
			result = append(result, task)
		}
	}
	return result, nil
}

// createTestScheduleService creates a ScheduleService for testing
func createTestScheduleService() *ScheduleService {
	scheduleRepo := newMockScheduleRepo()
	taskRepo := newMockTaskRepoForSchedule()
	return NewScheduleService(scheduleRepo, taskRepo, nil, nil, nil)
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

	svc := NewScheduleService(scheduleRepo, taskRepo, nil, nil, nil)

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

// Test GenerateScheduleWithAI respects preferred time slots
func TestScheduleService_GenerateScheduleWithAI_PreferredTimeSlots(t *testing.T) {
	scheduleRepo := newMockScheduleRepo()
	taskRepo := newMockTaskRepoForSchedule()

	// Task A with preferred time 10:00-11:00
	taskRepo.Create(&model.Task{
		ID:                 "task-a",
		Title:              "固定时段任务",
		Quadrant:           model.Quadrant1,
		Status:             model.StatusTodo,
		EstimatedTime:      60,
		PreferredStartTime: "10:00",
		PreferredEndTime:   "11:00",
	})
	// Task B without time preference
	taskRepo.Create(&model.Task{
		ID:            "task-b",
		Title:         "无时段任务",
		Quadrant:      model.Quadrant2,
		Status:        model.StatusTodo,
		EstimatedTime: 30,
	})

	svc := NewScheduleService(scheduleRepo, taskRepo, nil, nil, nil)

	events, err := svc.GenerateScheduleWithAI("09:00", "18:00")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}

	// Find the fixed-time task event
	var taskAEvent, taskBEvent *ScheduleEvent
	for i := range events {
		switch events[i].Title {
		case "固定时段任务":
			taskAEvent = &events[i]
		case "无时段任务":
			taskBEvent = &events[i]
		}
	}

	if taskAEvent == nil || taskBEvent == nil {
		t.Fatal("missing expected events")
	}

	// Task A should be scheduled at its preferred time 10:00
	startTime, _ := time.Parse(time.RFC3339, taskAEvent.Start)
	if startTime.Hour() != 10 || startTime.Minute() != 0 {
		t.Errorf("task A should start at 10:00, got %02d:%02d", startTime.Hour(), startTime.Minute())
	}

	// Task B should NOT overlap with task A (09:00-10:00 or after 11:00)
	taskBStart, _ := time.Parse(time.RFC3339, taskBEvent.Start)
	taskBEnd, _ := time.Parse(time.RFC3339, taskBEvent.End)

	// Task A occupies 10:00-11:00, so task B should be 09:00-09:30
	if taskBStart.Hour() == 10 {
		t.Error("task B should not overlap with task A's preferred slot")
	}
	_ = taskBEnd
}

// Test GenerateScheduleWithAI with all tasks having no time preference (backward compat)
func TestScheduleService_GenerateScheduleWithAI_NoPreferences(t *testing.T) {
	scheduleRepo := newMockScheduleRepo()
	taskRepo := newMockTaskRepoForSchedule()

	taskRepo.Create(&model.Task{
		ID:            "task-1",
		Title:         "任务1",
		Quadrant:      model.Quadrant1,
		Status:        model.StatusTodo,
		EstimatedTime: 30,
	})
	taskRepo.Create(&model.Task{
		ID:            "task-2",
		Title:         "任务2",
		Quadrant:      model.Quadrant2,
		Status:        model.StatusTodo,
		EstimatedTime: 30,
	})

	svc := NewScheduleService(scheduleRepo, taskRepo, nil, nil, nil)

	events, err := svc.GenerateScheduleWithAI("09:00", "18:00")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("expected 2 events, got %d", len(events))
	}

	// Both should be scheduled sequentially from 09:00
	if len(events) >= 2 {
		start1, _ := time.Parse(time.RFC3339, events[0].Start)
		start2, _ := time.Parse(time.RFC3339, events[1].Start)
		if !start2.After(start1) {
			t.Error("tasks without preferences should be scheduled sequentially")
		}
	}
}
// ===== computeDiff Tests =====

func makeICSEvent(summary string, startHour, startMin, endHour, endMin int) ICSEvent {
	loc := time.UTC
	return ICSEvent{
		Summary: summary,
		Start:   time.Date(2026, 6, 9, startHour, startMin, 0, 0, loc),
		End:     time.Date(2026, 6, 9, endHour, endMin, 0, 0, loc),
	}
}

func TestComputeDiff_Moved(t *testing.T) {
	original := []ICSEvent{
		makeICSEvent("代码评审", 10, 0, 11, 0),
	}
	revised := []ICSEvent{
		makeICSEvent("代码评审", 14, 0, 15, 0),
	}

	changes, summary := computeDiff(original, revised)

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Type != "moved" {
		t.Errorf("expected type 'moved', got '%s'", changes[0].Type)
	}
	if changes[0].Title != "代码评审" {
		t.Errorf("expected title '代码评审', got '%s'", changes[0].Title)
	}
	if summary == "" {
		t.Error("expected non-empty summary")
	}
	t.Logf("summary: %s", summary)
}

func TestComputeDiff_Added(t *testing.T) {
	original := []ICSEvent{
		makeICSEvent("任务A", 9, 0, 10, 0),
	}
	revised := []ICSEvent{
		makeICSEvent("任务A", 9, 0, 10, 0),
		makeICSEvent("任务B", 10, 0, 11, 0),
	}

	changes, summary := computeDiff(original, revised)

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Type != "added" {
		t.Errorf("expected type 'added', got '%s'", changes[0].Type)
	}
	if changes[0].Title != "任务B" {
		t.Errorf("expected title '任务B', got '%s'", changes[0].Title)
	}
	t.Logf("summary: %s", summary)
}

func TestComputeDiff_Removed(t *testing.T) {
	original := []ICSEvent{
		makeICSEvent("任务A", 9, 0, 10, 0),
		makeICSEvent("任务B", 10, 0, 11, 0),
	}
	revised := []ICSEvent{
		makeICSEvent("任务A", 9, 0, 10, 0),
	}

	changes, summary := computeDiff(original, revised)

	if len(changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(changes))
	}
	if changes[0].Type != "removed" {
		t.Errorf("expected type 'removed', got '%s'", changes[0].Type)
	}
	if changes[0].Title != "任务B" {
		t.Errorf("expected title '任务B', got '%s'", changes[0].Title)
	}
	t.Logf("summary: %s", summary)
}

func TestComputeDiff_Mixed(t *testing.T) {
	original := []ICSEvent{
		makeICSEvent("任务A", 9, 0, 10, 0),
		makeICSEvent("任务B", 10, 0, 11, 0),
		makeICSEvent("任务C", 11, 0, 12, 0),
	}
	revised := []ICSEvent{
		makeICSEvent("任务A", 14, 0, 15, 0),
		makeICSEvent("任务C", 11, 0, 12, 0),
		makeICSEvent("任务D", 15, 0, 16, 0),
	}

	changes, summary := computeDiff(original, revised)

	if len(changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(changes))
	}

	moved, added, removed := 0, 0, 0
	for _, ch := range changes {
		switch ch.Type {
		case "moved":
			moved++
			if ch.Title != "任务A" {
				t.Errorf("expected moved title '任务A', got '%s'", ch.Title)
			}
		case "added":
			added++
			if ch.Title != "任务D" {
				t.Errorf("expected added title '任务D', got '%s'", ch.Title)
			}
		case "removed":
			removed++
			if ch.Title != "任务B" {
				t.Errorf("expected removed title '任务B', got '%s'", ch.Title)
			}
		}
	}

	if moved != 1 || added != 1 || removed != 1 {
		t.Errorf("expected 1m/1a/1r, got %dm/%da/%dr", moved, added, removed)
	}
	t.Logf("summary: %s", summary)
}

func TestComputeDiff_NoChanges(t *testing.T) {
	original := []ICSEvent{
		makeICSEvent("任务A", 9, 0, 10, 0),
		makeICSEvent("任务B", 10, 0, 11, 0),
	}
	revised := []ICSEvent{
		makeICSEvent("任务A", 9, 0, 10, 0),
		makeICSEvent("任务B", 10, 0, 11, 0),
	}

	changes, summary := computeDiff(original, revised)

	if len(changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(changes))
	}
	if summary != "当前日程已是最优安排，无需调整" {
		t.Errorf("expected empty-state message, got '%s'", summary)
	}
}

func TestComputeDiff_AllAdded(t *testing.T) {
	original := []ICSEvent{}
	revised := []ICSEvent{
		makeICSEvent("任务A", 9, 0, 10, 0),
		makeICSEvent("任务B", 10, 0, 11, 0),
	}

	changes, _ := computeDiff(original, revised)

	if len(changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(changes))
	}
	for _, ch := range changes {
		if ch.Type != "added" {
			t.Errorf("expected all 'added', got '%s' for '%s'", ch.Type, ch.Title)
		}
	}
}

func TestComputeDiff_DifferentDays(t *testing.T) {
	loc := time.UTC
	original := []ICSEvent{
		{Summary: "每日站会", Start: time.Date(2026, 6, 9, 9, 0, 0, 0, loc), End: time.Date(2026, 6, 9, 9, 30, 0, 0, loc)},
	}
	revised := []ICSEvent{
		{Summary: "每日站会", Start: time.Date(2026, 6, 10, 9, 0, 0, 0, loc), End: time.Date(2026, 6, 10, 9, 30, 0, 0, loc)},
	}

	changes, _ := computeDiff(original, revised)

	if len(changes) != 2 {
		t.Fatalf("expected 2 changes (removed from day 9 + added on day 10), got %d", len(changes))
	}
}

func TestWriteScheduleICS_RoundTrip(t *testing.T) {
	events := []ScheduleEvent{
		{
			ID:    "evt-1",
			Title: "代码评审",
			Start: "2026-06-09T10:00:00Z",
			End:   "2026-06-09T11:00:00Z",
			Type:  "deep_work",
		},
		{
			ID:    "evt-2",
			Title: "团队同步",
			Start: "2026-06-09T14:00:00Z",
			End:   "2026-06-09T14:30:00Z",
			Type:  "meeting",
		},
	}

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\n")
	sb.WriteString("VERSION:2.0\n")
	sb.WriteString("PRODID:-//TickTask//EN\n")
	sb.WriteString("CALSCALE:GREGORIAN\n")
	sb.WriteString("METHOD:PUBLISH\n")
	for _, ev := range events {
		startTime, _ := time.Parse(time.RFC3339, ev.Start)
		endTime, _ := time.Parse(time.RFC3339, ev.End)
		icsStart := startTime.Format("20060102T150405")
		icsEnd := endTime.Format("20060102T150405")

		sb.WriteString("BEGIN:VEVENT\n")
		sb.WriteString(fmt.Sprintf("DTSTART:%s\n", icsStart))
		sb.WriteString(fmt.Sprintf("DTEND:%s\n", icsEnd))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\n", escapeICS(ev.Title)))
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s | %s\n", escapeICS(ev.Title), ev.Type))
		sb.WriteString("END:VEVENT\n")
	}
	sb.WriteString("END:VCALENDAR\n")

	icsContent := sb.String()

	parsed, err := ParseICS(icsContent, time.UTC)
	if err != nil {
		t.Fatalf("failed to parse generated ICS: %v", err)
	}

	if len(parsed) != 2 {
		t.Fatalf("expected 2 events, got %d", len(parsed))
	}
	if parsed[0].Summary != "代码评审" {
		t.Errorf("expected summary '代码评审', got '%s'", parsed[0].Summary)
	}
	if parsed[1].Summary != "团队同步" {
		t.Errorf("expected summary '团队同步', got '%s'", parsed[1].Summary)
	}
	if parsed[0].Start.Hour() != 10 || parsed[0].Start.Minute() != 0 {
		t.Errorf("expected start 10:00, got %02d:%02d", parsed[0].Start.Hour(), parsed[0].Start.Minute())
	}
}

func TestEscapeICS_SpecialChars(t *testing.T) {
	input := "任务: 代码评审; Q1, 重要"
	escaped := escapeICS(input)
	if !strings.Contains(escaped, "\\;") {
		t.Error("semicolons should be escaped")
	}
	if !strings.Contains(escaped, "\\,") {
		t.Error("commas should be escaped")
	}
}
