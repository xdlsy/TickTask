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

func (m *mockTaskRepository) GetCompletedTasksInRange(start, end time.Time) ([]*model.Task, error) {
	var result []*model.Task
	for _, task := range m.tasks {
		if task.Status == model.StatusCompleted && task.CompletedAt != nil &&
			!task.CompletedAt.Before(start) && task.CompletedAt.Before(end) {
			result = append(result, task)
		}
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

func (m *mockSessionRepository) CountByTaskID(taskID string, sessionType model.SessionType, status model.SessionStatus) (int, error) {
	count := 0
	for _, s := range m.sessions {
		if s.TaskID != nil && *s.TaskID == taskID && s.Type == sessionType && s.Status == status {
			count++
		}
	}
	return count, nil
}

func (m *mockSessionRepository) GetCompletedWorkByDateRange(start, end time.Time) ([]model.PomodoroSession, error) {
	var result []model.PomodoroSession
	for _, s := range m.sessions {
		if s.Type == model.SessionWork && s.Status == model.SessionCompleted && !s.StartTime.Before(start) && s.StartTime.Before(end) {
			result = append(result, *s)
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
	migrateCalls     int
	migrateErr       error
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
	// Mirror the real repo's "preserve on empty api_key" contract so handler
	// tests can verify the mask-roundtrip fix end-to-end without a DB.
	if settings.APIKey == "" && m.aiSettings != nil {
		preserved := *settings
		preserved.APIKey = m.aiSettings.APIKey
		m.aiSettings = &preserved
		return nil
	}
	m.aiSettings = settings
	return nil
}

func (m *mockSettingRepository) MigrateLegacyAPIKey() error {
	m.migrateCalls++
	return m.migrateErr
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

func (m *mockScheduleRepository) DeleteTaskSchedulesByDateRange(start, end time.Time) (int64, error) {
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

func (m *mockScheduleRepository) Move(id string, startTime, endTime time.Time) error {
	if schedule, ok := m.schedules[id]; ok {
		schedule.StartTime = startTime
		schedule.EndTime = endTime
		return nil
	}
	return repository.ErrNotFound
}

// mockWorkLogRepository implements repository.WorkLogRepository for testing
type mockWorkLogRepository struct {
	logs    map[string]*model.WorkLog
	items   map[string]*model.WorkItem // itemID -> item
	reports map[string]*model.WorkReport
}

func newMockWorkLogRepository() *mockWorkLogRepository {
	return &mockWorkLogRepository{
		logs:    make(map[string]*model.WorkLog),
		items:   make(map[string]*model.WorkItem),
		reports: make(map[string]*model.WorkReport),
	}
}

func (m *mockWorkLogRepository) CreateWorkLog(log *model.WorkLog) error {
	m.logs[log.Date] = log
	for i := range log.Items {
		m.items[log.Items[i].ID] = &log.Items[i]
	}
	return nil
}

func (m *mockWorkLogRepository) GetWorkLogByDate(date string) (*model.WorkLog, error) {
	if log, ok := m.logs[date]; ok {
		return log, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockWorkLogRepository) GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error) {
	var result []*model.WorkLog
	for _, log := range m.logs {
		if log.Date >= from && log.Date <= to {
			result = append(result, log)
		}
	}
	return result, nil
}

func (m *mockWorkLogRepository) UpsertWorkLog(log *model.WorkLog) error {
	existing, ok := m.logs[log.Date]
	if !ok {
		m.logs[log.Date] = log
		for i := range log.Items {
			m.items[log.Items[i].ID] = &log.Items[i]
		}
		return nil
	}
	log.ID = existing.ID
	// 关键不变式：只删 ai items
	newItems := []model.WorkItem{}
	for _, it := range existing.Items {
		if it.Source == "manual" {
			newItems = append(newItems, it)
		} else {
			delete(m.items, it.ID)
		}
	}
	newItems = append(newItems, log.Items...)
	log.Items = newItems
	m.logs[log.Date] = log
	for i := range log.Items {
		m.items[log.Items[i].ID] = &log.Items[i]
	}
	return nil
}

func (m *mockWorkLogRepository) ReplaceItems(workLogID string, items []model.WorkItem) error {
	return nil // 测试不依赖
}

func (m *mockWorkLogRepository) AppendItem(workLogID string, item model.WorkItem) error {
	// 校验 WorkLog 存在
	var found *model.WorkLog
	for _, log := range m.logs {
		if log.ID == workLogID {
			found = log
			break
		}
	}
	if found == nil {
		return repository.ErrNotFound
	}
	item.WorkLogID = workLogID
	found.Items = append(found.Items, item)
	m.items[item.ID] = &item
	return nil
}

func (m *mockWorkLogRepository) UpdateItem(workLogID string, itemID string, updates map[string]any) error {
	item, ok := m.items[itemID]
	if !ok || item.WorkLogID != workLogID {
		return repository.ErrItemNotFound
	}
	if item.Source != "manual" {
		return repository.ErrItemNotEditable
	}
	if v, ok := updates["activity"]; ok {
		s := v.(string)
		item.Activity = &s
	}
	if v, ok := updates["start_time"]; ok {
		s := v.(string)
		item.StartTime = &s
	}
	if v, ok := updates["end_time"]; ok {
		s := v.(string)
		item.EndTime = &s
	}
	if v, ok := updates["quadrant"]; ok {
		i := v.(int)
		item.Quadrant = &i
	}
	return nil
}

func (m *mockWorkLogRepository) DeleteItem(workLogID string, itemID string) error {
	item, ok := m.items[itemID]
	if !ok || item.WorkLogID != workLogID {
		return repository.ErrItemNotFound
	}
	if item.Source != "manual" {
		return repository.ErrItemNotEditable
	}
	delete(m.items, itemID)
	for _, log := range m.logs {
		if log.ID == workLogID {
			for i, it := range log.Items {
				if it.ID == itemID {
					log.Items = append(log.Items[:i], log.Items[i+1:]...)
					break
				}
			}
			break
		}
	}
	return nil
}

func (m *mockWorkLogRepository) CreateWorkReport(report *model.WorkReport) error {
	m.reports[string(report.Type)+":"+report.PeriodKey] = report
	return nil
}
func (m *mockWorkLogRepository) UpdateWorkReport(report *model.WorkReport) error {
	m.reports[string(report.Type)+":"+report.PeriodKey] = report
	return nil
}
func (m *mockWorkLogRepository) GetWorkReportByTypeAndPeriod(t model.WorkReportType, periodKey string) (*model.WorkReport, error) {
	if r, ok := m.reports[string(t)+":"+periodKey]; ok {
		return r, nil
	}
	return nil, repository.ErrNotFound
}
func (m *mockWorkLogRepository) ListWorkReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	var result []*model.WorkReport
	for _, r := range m.reports {
		if r.Type == t {
			result = append(result, r)
		}
	}
	return result, nil
}

func (m *mockWorkLogRepository) UpdateWorkLogSummary(date string, summary string) error {
	for _, log := range m.logs {
		if log.Date == date {
			log.Summary = summary
			return nil
		}
	}
	return repository.ErrNotFound
}