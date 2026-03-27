package handler

import (
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"time"
)

// mockTaskRepository implements repository.TaskRepository for testing
type mockTaskRepository struct {
	tasks map[string]*model.Task
}

func newMockTaskRepository() *mockTaskRepository {
	return &mockTaskRepository{
		tasks: make(map[string]*model.Task),
	}
}

func (m *mockTaskRepository) Create(task *model.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepository) Update(task *model.Task) error {
	m.tasks[task.ID] = task
	return nil
}

func (m *mockTaskRepository) Delete(id string) error {
	delete(m.tasks, id)
	return nil
}

func (m *mockTaskRepository) GetByID(id string) (*model.Task, error) {
	if task, ok := m.tasks[id]; ok {
		return task, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockTaskRepository) GetAll() ([]model.Task, error) {
	var result []model.Task
	for _, task := range m.tasks {
		result = append(result, *task)
	}
	return result, nil
}

func (m *mockTaskRepository) GetByStatus(status model.TaskStatus) ([]model.Task, error) {
	var result []model.Task
	for _, task := range m.tasks {
		if task.Status == status {
			result = append(result, *task)
		}
	}
	return result, nil
}

func (m *mockTaskRepository) GetByQuadrant(quadrant model.Quadrant) ([]model.Task, error) {
	var result []model.Task
	for _, task := range m.tasks {
		if task.Quadrant == quadrant {
			result = append(result, *task)
		}
	}
	return result, nil
}

func (m *mockTaskRepository) GetAllByQuadrant() (map[model.Quadrant][]model.Task, error) {
	result := make(map[model.Quadrant][]model.Task)
	for i := 1; i <= 4; i++ {
		result[model.Quadrant(i)] = []model.Task{}
	}
	for _, task := range m.tasks {
		result[task.Quadrant] = append(result[task.Quadrant], *task)
	}
	return result, nil
}

// mockSessionRepository implements repository.SessionRepository for testing
type mockSessionRepository struct {
	sessions map[string]*model.PomodoroSession
	active   *model.PomodoroSession
}

func newMockSessionRepository() *mockSessionRepository {
	return &mockSessionRepository{
		sessions: make(map[string]*model.PomodoroSession),
	}
}

func (m *mockSessionRepository) Create(session *model.PomodoroSession) error {
	m.sessions[session.ID] = session
	if session.Status == model.SessionRunning {
		m.active = session
	}
	return nil
}

func (m *mockSessionRepository) Update(session *model.PomodoroSession) error {
	m.sessions[session.ID] = session
	return nil
}

func (m *mockSessionRepository) GetByID(id string) (*model.PomodoroSession, error) {
	if session, ok := m.sessions[id]; ok {
		return session, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockSessionRepository) GetActive() (*model.PomodoroSession, error) {
	return m.active, nil
}

func (m *mockSessionRepository) GetRecent(limit int) ([]model.PomodoroSession, error) {
	var result []model.PomodoroSession
	for _, session := range m.sessions {
		result = append(result, *session)
	}
	return result, nil
}

func (m *mockSessionRepository) GetByDate(date time.Time) ([]model.PomodoroSession, error) {
	var result []model.PomodoroSession
	for _, session := range m.sessions {
		sessionDate := session.StartTime.In(date.Location())
		if sessionDate.Year() == date.Year() && sessionDate.Month() == date.Month() && sessionDate.Day() == date.Day() {
			result = append(result, *session)
		}
	}
	return result, nil
}

// mockAnalyticsRepository implements repository.AnalyticsRepository for testing
type mockAnalyticsRepository struct {
	createdTasks   int
	completedTasks int
	focusTime      int
	pomodoros      int
}

func newMockAnalyticsRepository() *mockAnalyticsRepository {
	return &mockAnalyticsRepository{}
}

func (m *mockAnalyticsRepository) GetDailyStats(date time.Time) (*model.DailyStats, error) {
	return &model.DailyStats{}, nil
}

func (m *mockAnalyticsRepository) GetDailyStatsRange(start, end time.Time) ([]model.DailyStats, error) {
	return []model.DailyStats{}, nil
}

func (m *mockAnalyticsRepository) CreateDailyStats(stats *model.DailyStats) error {
	return nil
}

func (m *mockAnalyticsRepository) UpdateDailyStats(stats *model.DailyStats) error {
	return nil
}

func (m *mockAnalyticsRepository) IncrementCompletedPomodoros(date time.Time) error {
	m.pomodoros++
	return nil
}

func (m *mockAnalyticsRepository) IncrementFocusTime(date time.Time, seconds int) error {
	m.focusTime += seconds
	return nil
}

func (m *mockAnalyticsRepository) IncrementCompletedTasks(date time.Time) error {
	m.completedTasks++
	return nil
}

func (m *mockAnalyticsRepository) IncrementCreatedTasks(date time.Time) error {
	m.createdTasks++
	return nil
}

// mockSettingRepository implements repository.SettingRepository for testing
type mockSettingRepository struct {
	pomodoroSettings *model.PomodoroSettings
	aiSettings       *model.AISettings
}

func newMockSettingRepository() *mockSettingRepository {
	return &mockSettingRepository{
		pomodoroSettings: model.DefaultPomodoroSettings(),
		aiSettings:       model.DefaultAISettings(),
	}
}

func (m *mockSettingRepository) Get(key string) (*model.Setting, error) {
	return nil, repository.ErrNotFound
}

func (m *mockSettingRepository) Set(key, value string) error {
	return nil
}

func (m *mockSettingRepository) GetPomodoroSettings() (*model.PomodoroSettings, error) {
	return m.pomodoroSettings, nil
}

func (m *mockSettingRepository) UpdatePomodoroSettings(settings *model.PomodoroSettings) error {
	m.pomodoroSettings = settings
	return nil
}

func (m *mockSettingRepository) GetAISettings() (*model.AISettings, error) {
	return m.aiSettings, nil
}

func (m *mockSettingRepository) UpdateAISettings(settings *model.AISettings) error {
	m.aiSettings = settings
	return nil
}

// mockScheduleRepository implements repository.ScheduleRepository for testing
type mockScheduleRepository struct {
	schedules map[string]*model.Schedule
}

func newMockScheduleRepository() *mockScheduleRepository {
	return &mockScheduleRepository{
		schedules: make(map[string]*model.Schedule),
	}
}

func (m *mockScheduleRepository) Create(schedule *model.Schedule) error {
	m.schedules[schedule.ID] = schedule
	return nil
}

func (m *mockScheduleRepository) GetByID(id string) (*model.Schedule, error) {
	if schedule, ok := m.schedules[id]; ok {
		return schedule, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockScheduleRepository) GetByTimeRange(start, end time.Time) ([]model.Schedule, error) {
	var result []model.Schedule
	for _, schedule := range m.schedules {
		if (schedule.StartTime.After(start) || schedule.StartTime.Equal(start)) &&
			(schedule.StartTime.Before(end) || schedule.StartTime.Equal(end)) {
			result = append(result, *schedule)
		}
	}
	return result, nil
}

func (m *mockScheduleRepository) GetByTaskID(taskID string) ([]model.Schedule, error) {
	var result []model.Schedule
	for _, schedule := range m.schedules {
		if schedule.TaskID != nil && *schedule.TaskID == taskID {
			result = append(result, *schedule)
		}
	}
	return result, nil
}

func (m *mockScheduleRepository) GetByDate(date time.Time) ([]model.Schedule, error) {
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

func (m *mockScheduleRepository) Update(schedule *model.Schedule) error {
	m.schedules[schedule.ID] = schedule
	return nil
}

func (m *mockScheduleRepository) Delete(id string) error {
	delete(m.schedules, id)
	return nil
}

func (m *mockScheduleRepository) UpdateStatus(id string, status model.ScheduleStatus) error {
	if schedule, ok := m.schedules[id]; ok {
		schedule.Status = status
		return nil
	}
	return repository.ErrNotFound
}

func (m *mockScheduleRepository) Move(id string, startTime, endTime time.Time) error {
	if schedule, ok := m.schedules[id]; ok {
		schedule.StartTime = startTime
		schedule.EndTime = endTime
		return nil
	}
	return repository.ErrNotFound
}