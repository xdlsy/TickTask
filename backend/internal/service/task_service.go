package service

import (
	"fmt"
	"math"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"time"

	"github.com/google/uuid"
)

type TaskService struct {
	taskRepo    repository.TaskRepository
	analytics   repository.AnalyticsRepository
	settingRepo repository.SettingRepository
	sessionRepo repository.SessionRepository
}

func NewTaskService(
	taskRepo repository.TaskRepository,
	analytics repository.AnalyticsRepository,
	settingRepo repository.SettingRepository,
	sessionRepo repository.SessionRepository,
) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		analytics:   analytics,
		settingRepo: settingRepo,
		sessionRepo: sessionRepo,
	}
}

type CreateTaskRequest struct {
	Title              string             `json:"title" binding:"required"`
	Description        string             `json:"description"`
	Quadrant           model.Quadrant     `json:"quadrant"`
	IsImportant        bool               `json:"is_important"`
	IsUrgent           bool               `json:"is_urgent"`
	EstimatedTime      int                `json:"estimated_time"`
	Deadline           *time.Time         `json:"deadline"`
	StartDate          *time.Time         `json:"start_date"`
	DueDate            *time.Time         `json:"due_date"`
	IsRecurring        bool               `json:"is_recurring"`
	RecurrencePattern  string             `json:"recurrence_pattern"`
	PreferredStartTime string             `json:"preferred_start_time"`
	PreferredEndTime   string             `json:"preferred_end_time"`
	Tags               []string           `json:"tags"`
}

type UpdateTaskRequest struct {
	Title              *string            `json:"title"`
	Description        *string            `json:"description"`
	Quadrant           *model.Quadrant    `json:"quadrant"`
	IsImportant        *bool              `json:"is_important"`
	IsUrgent           *bool              `json:"is_urgent"`
	Status             *model.TaskStatus  `json:"status"`
	EstimatedTime      *int               `json:"estimated_time"`
	Deadline           *time.Time         `json:"deadline"`
	StartDate          *time.Time         `json:"start_date"`
	DueDate            *time.Time         `json:"due_date"`
	IsRecurring        *bool              `json:"is_recurring"`
	RecurrencePattern  *string            `json:"recurrence_pattern"`
	PreferredStartTime *string            `json:"preferred_start_time"`
	PreferredEndTime   *string            `json:"preferred_end_time"`
	Tags               []string           `json:"tags"`
	Order              *int               `json:"order"`
}

func (s *TaskService) CreateTask(req CreateTaskRequest) (*model.Task, error) {
	task := &model.Task{
		ID:                 uuid.New().String(),
		Title:              req.Title,
		Description:        req.Description,
		Quadrant:           req.Quadrant,
		IsImportant:        req.IsImportant,
		IsUrgent:           req.IsUrgent,
		Status:             model.StatusTodo,
		EstimatedTime:      req.EstimatedTime,
		Deadline:           req.Deadline,
		StartDate:          req.StartDate,
		DueDate:            req.DueDate,
		IsRecurring:        req.IsRecurring,
		RecurrencePattern:  req.RecurrencePattern,
		PreferredStartTime: req.PreferredStartTime,
		PreferredEndTime:   req.PreferredEndTime,
		Tags:               encodeTags(req.Tags),
		Order:              0,
		CreatedAt:          time.Now(),
		UpdatedAt:          time.Now(),
	}

	if err := s.taskRepo.Create(task); err != nil {
		return nil, err
	}

	// 更新统计
	s.analytics.IncrementCreatedTasks(time.Now())

	return task, nil
}

func (s *TaskService) UpdateTask(id string, req UpdateTaskRequest) error {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return err
	}

	if req.Title != nil {
		task.Title = *req.Title
	}
	if req.Description != nil {
		task.Description = *req.Description
	}
	if req.Quadrant != nil {
		task.Quadrant = *req.Quadrant
	}
	if req.IsImportant != nil {
		task.IsImportant = *req.IsImportant
	}
	if req.IsUrgent != nil {
		task.IsUrgent = *req.IsUrgent
	}
	if req.Status != nil {
		wasCompleted := task.Status == model.StatusCompleted
		task.Status = *req.Status

		// 状态变更为完成时
		if *req.Status == model.StatusCompleted && !wasCompleted {
			now := time.Now()
			task.CompletedAt = &now
			s.analytics.IncrementCompletedTasks(time.Now())
		}
	}
	if req.EstimatedTime != nil {
		task.EstimatedTime = *req.EstimatedTime
	}
	if req.Deadline != nil {
		task.Deadline = req.Deadline
	}
	if req.StartDate != nil {
		task.StartDate = req.StartDate
	}
	if req.DueDate != nil {
		task.DueDate = req.DueDate
	}
	if req.IsRecurring != nil {
		task.IsRecurring = *req.IsRecurring
	}
	if req.RecurrencePattern != nil {
		task.RecurrencePattern = *req.RecurrencePattern
	}
	if req.PreferredStartTime != nil {
		task.PreferredStartTime = *req.PreferredStartTime
	}
	if req.PreferredEndTime != nil {
		task.PreferredEndTime = *req.PreferredEndTime
	}
	if req.Tags != nil {
		task.Tags = encodeTags(req.Tags)
	}
	if req.Order != nil {
		task.Order = *req.Order
	}

	task.UpdatedAt = time.Now()

	return s.taskRepo.Update(task)
}

func (s *TaskService) DeleteTask(id string) error {
	return s.taskRepo.Delete(id)
}

func (s *TaskService) GetTask(id string) (*model.Task, error) {
	return s.taskRepo.GetByID(id)
}

func (s *TaskService) GetAllTasks() ([]model.Task, error) {
	return s.taskRepo.GetAll()
}

func (s *TaskService) GetTasksByQuadrant() (map[model.Quadrant][]model.Task, error) {
	return s.taskRepo.GetAllByQuadrant()
}

func (s *TaskService) MoveTask(id string, targetQuadrant model.Quadrant) error {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return err
	}

	task.Quadrant = targetQuadrant
	task.UpdatedAt = time.Now()

	// 根据 IsImportant 和 IsUrgent 自动更新
	if targetQuadrant == 1 {
		task.IsImportant = true
		task.IsUrgent = true
	} else if targetQuadrant == 2 {
		task.IsImportant = true
		task.IsUrgent = false
	} else if targetQuadrant == 3 {
		task.IsImportant = false
		task.IsUrgent = true
	} else {
		task.IsImportant = false
		task.IsUrgent = false
	}

	return s.taskRepo.Update(task)
}

// 自动计算象限
func (s *TaskService) CalculateQuadrant(important, urgent bool) model.Quadrant {
	if important && urgent {
		return model.Quadrant1
	}
	if important && !urgent {
		return model.Quadrant2
	}
	if !important && urgent {
		return model.Quadrant3
	}
	return model.Quadrant4
}

// GetPomodoroSettings 获取番茄设置
func (s *TaskService) GetPomodoroSettings() (*model.PomodoroSettings, error) {
	return s.settingRepo.GetPomodoroSettings()
}

func encodeTags(tags []string) string {
	// 简化处理，实际应使用 json.Marshal
	return ""
}

func decodeTags(s string) []string {
	// 简化处理，实际应使用 json.Unmarshal
	return []string{}
}

// TaskResponse is the enriched API response DTO with computed pomodoro fields.
type TaskResponse struct {
	ID                 string     `json:"id"`
	Title              string     `json:"title"`
	Description        string     `json:"description"`
	Quadrant           int        `json:"quadrant"`
	IsImportant        bool       `json:"is_important"`
	IsUrgent           bool       `json:"is_urgent"`
	Status             string     `json:"status"`
	EstimatedTime      int        `json:"estimated_time"`
	Deadline           *time.Time `json:"deadline"`
	StartDate          *time.Time `json:"start_date"`
	DueDate            *time.Time `json:"due_date"`
	IsRecurring        bool       `json:"is_recurring"`
	RecurrencePattern  string     `json:"recurrence_pattern"`
	RecurrenceDay      int        `json:"recurrence_day"`
	PreferredStartTime string     `json:"preferred_start_time"`
	PreferredEndTime   string     `json:"preferred_end_time"`
	Tags               string     `json:"tags"`
	Order              int        `json:"order"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	CompletedAt        *time.Time `json:"completed_at"`
	// Computed pomodoro fields
	PlannedPomodoros   int    `json:"planned_pomodoros"`
	CompletedPomodoros int    `json:"completed_pomodoros"`
	PomodoroStatus     string `json:"pomodoro_status"`
}

// GetTaskResponse returns a single task enriched with pomodoro info.
func (s *TaskService) GetTaskResponse(id string) (*TaskResponse, error) {
	task, err := s.taskRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	workDuration, err := s.getWorkDurationMinutes()
	if err != nil {
		workDuration = 25 // fallback default
	}
	return s.enrichTask(task, workDuration), nil
}

// GetAllTaskResponses returns all tasks enriched with pomodoro info.
func (s *TaskService) GetAllTaskResponses() ([]TaskResponse, error) {
	tasks, err := s.taskRepo.GetAll()
	if err != nil {
		return nil, err
	}
	workDuration, err := s.getWorkDurationMinutes()
	if err != nil {
		workDuration = 25
	}
	result := make([]TaskResponse, len(tasks))
	for i, t := range tasks {
		result[i] = *s.enrichTask(&t, workDuration)
	}
	return result, nil
}

// GetTasksByQuadrantResponse returns tasks grouped by quadrant, enriched with pomodoro info.
func (s *TaskService) GetTasksByQuadrantResponse() (map[string][]TaskResponse, error) {
	quadrantMap, err := s.taskRepo.GetAllByQuadrant()
	if err != nil {
		return nil, err
	}
	workDuration, err := s.getWorkDurationMinutes()
	if err != nil {
		workDuration = 25
	}
	result := make(map[string][]TaskResponse)
	for q, tasks := range quadrantMap {
		key := fmt.Sprintf("%d", int(q))
		enriched := make([]TaskResponse, len(tasks))
		for i, t := range tasks {
			enriched[i] = *s.enrichTask(&t, workDuration)
		}
		result[key] = enriched
	}
	return result, nil
}

func (s *TaskService) getWorkDurationMinutes() (int, error) {
	settings, err := s.settingRepo.GetPomodoroSettings()
	if err != nil {
		return 0, err
	}
	return settings.WorkDuration / 60, nil // seconds to minutes
}

func (s *TaskService) enrichTask(task *model.Task, workDurationMinutes int) *TaskResponse {
	resp := TaskResponse{
		ID:                 task.ID,
		Title:              task.Title,
		Description:        task.Description,
		Quadrant:           int(task.Quadrant),
		IsImportant:        task.IsImportant,
		IsUrgent:           task.IsUrgent,
		Status:             string(task.Status),
		EstimatedTime:      task.EstimatedTime,
		Deadline:           task.Deadline,
		StartDate:          task.StartDate,
		DueDate:            task.DueDate,
		IsRecurring:        task.IsRecurring,
		RecurrencePattern:  task.RecurrencePattern,
		RecurrenceDay:      task.RecurrenceDay,
		PreferredStartTime: task.PreferredStartTime,
		PreferredEndTime:   task.PreferredEndTime,
		Tags:               task.Tags,
		Order:              task.Order,
		CreatedAt:          task.CreatedAt,
		UpdatedAt:          task.UpdatedAt,
		CompletedAt:        task.CompletedAt,
	}

	// Compute planned pomodoros
	if task.EstimatedTime > 0 && workDurationMinutes > 0 {
		resp.PlannedPomodoros = int(math.Ceil(float64(task.EstimatedTime) / float64(workDurationMinutes)))
	}

	// Count completed work sessions
	completed, err := s.sessionRepo.CountByTaskID(task.ID, model.SessionWork, model.SessionCompleted)
	if err == nil {
		resp.CompletedPomodoros = completed
	}

	// Determine pomodoro status
	resp.PomodoroStatus = computePomodoroStatus(resp.PlannedPomodoros, resp.CompletedPomodoros)

	return &resp
}

func computePomodoroStatus(planned, completed int) string {
	if planned == 0 {
		return "not_started"
	}
	if completed == 0 {
		return "not_started"
	}
	if completed < planned {
		return "in_progress"
	}
	if completed == planned {
		return "completed"
	}
	return "exceeded"
}
