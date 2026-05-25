package service

import (
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"time"

	"github.com/google/uuid"
)

type TaskService struct {
	taskRepo    repository.TaskRepository
	analytics   repository.AnalyticsRepository
	settingRepo repository.SettingRepository
}

func NewTaskService(
	taskRepo repository.TaskRepository,
	analytics repository.AnalyticsRepository,
	settingRepo repository.SettingRepository,
) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		analytics:   analytics,
		settingRepo: settingRepo,
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
