package service

import (
	"fmt"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"time"

	"github.com/google/uuid"
)

type ScheduleService struct {
	scheduleRepo repository.ScheduleRepository
	taskRepo     repository.TaskRepository
	aiService    *AIService
}

func NewScheduleService(
	scheduleRepo repository.ScheduleRepository,
	taskRepo repository.TaskRepository,
	aiService *AIService,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		taskRepo:     taskRepo,
		aiService:    aiService,
	}
}

// ScheduleEvent 日程事件（用于日历显示）
type ScheduleEvent struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Color     string `json:"color"`
	TaskID    string `json:"task_id,omitempty"`
	AllDay    bool   `json:"allDay"`
	Editable  bool   `json:"editable"`
}

// CreateScheduleDTO 创建日程请求
type CreateScheduleDTO struct {
	TaskID      string `json:"task_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Type        string `json:"type"`
	Color       string `json:"color"`
}

// UpdateScheduleDTO 更新日程请求
type UpdateScheduleDTO struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Status      string `json:"status"`
	Color       string `json:"color"`
}

// MoveScheduleDTO 移动日程请求
type MoveScheduleDTO struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// GetSchedules 获取日程列表
func (s *ScheduleService) GetSchedules(start, end time.Time) ([]ScheduleEvent, error) {
	schedules, err := s.scheduleRepo.GetByTimeRange(start, end)
	if err != nil {
		return nil, err
	}

	events := make([]ScheduleEvent, len(schedules))
	for i, schedule := range schedules {
		events[i] = s.toEvent(&schedule)
	}
	return events, nil
}

// GetSchedule 获取单个日程
func (s *ScheduleService) GetSchedule(id string) (*model.Schedule, error) {
	return s.scheduleRepo.GetByID(id)
}

// CreateSchedule 创建日程（返回 model）
func (s *ScheduleService) CreateSchedule(dto *CreateScheduleDTO) (*model.Schedule, error) {
	startTime, err := time.Parse(time.RFC3339, dto.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format")
	}

	endTime, err := time.Parse(time.RFC3339, dto.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time format")
	}

	schedule := &model.Schedule{
		ID:          uuid.New().String(),
		Title:       dto.Title,
		Description: dto.Description,
		StartTime:   startTime,
		EndTime:     endTime,
		Type:        model.ScheduleType(dto.Type),
		Status:      model.ScheduleStatusPlanned,
		Color:       dto.Color,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if dto.TaskID != "" {
		schedule.TaskID = &dto.TaskID
	}

	// 如果没有设置标题，尝试从任务获取
	if schedule.Title == "" && schedule.TaskID != nil {
		task, err := s.taskRepo.GetByID(*schedule.TaskID)
		if err == nil {
			schedule.Title = task.Title
		}
	}

	// 设置默认颜色
	if schedule.Color == "" {
		schedule.Color = s.getDefaultColor(schedule.Type)
	}

	if err := s.scheduleRepo.Create(schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// CreateScheduleEvent 创建日程（返回前端事件格式）
func (s *ScheduleService) CreateScheduleEvent(dto *CreateScheduleDTO) (*ScheduleEvent, error) {
	schedule, err := s.CreateSchedule(dto)
	if err != nil {
		return nil, err
	}
	event := s.toEvent(schedule)
	return &event, nil
}

// UpdateSchedule 更新日程
func (s *ScheduleService) UpdateSchedule(id string, dto *UpdateScheduleDTO) error {
	schedule, err := s.scheduleRepo.GetByID(id)
	if err != nil {
		return err
	}

	if dto.Title != "" {
		schedule.Title = dto.Title
	}
	if dto.Description != "" {
		schedule.Description = dto.Description
	}
	if dto.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, dto.StartTime)
		if err != nil {
			return fmt.Errorf("invalid start_time format")
		}
		schedule.StartTime = startTime
	}
	if dto.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, dto.EndTime)
		if err != nil {
			return fmt.Errorf("invalid end_time format")
		}
		schedule.EndTime = endTime
	}
	if dto.Status != "" {
		schedule.Status = model.ScheduleStatus(dto.Status)
	}
	if dto.Color != "" {
		schedule.Color = dto.Color
	}

	schedule.UpdatedAt = time.Now()
	return s.scheduleRepo.Update(schedule)
}

// DeleteSchedule 删除日程
func (s *ScheduleService) DeleteSchedule(id string) error {
	return s.scheduleRepo.Delete(id)
}

// MoveSchedule 移动日程
func (s *ScheduleService) MoveSchedule(id string, startTime, endTime time.Time) error {
	return s.scheduleRepo.Move(id, startTime, endTime)
}

// UpdateScheduleStatus 更新日程状态
func (s *ScheduleService) UpdateScheduleStatus(id string, status model.ScheduleStatus) error {
	return s.scheduleRepo.UpdateStatus(id, status)
}

// GenerateScheduleWithAI AI生成日程
func (s *ScheduleService) GenerateScheduleWithAI(startTime, endTime string) ([]ScheduleEvent, error) {
	// 获取待办任务
	tasks, err := s.taskRepo.GetByStatus(model.StatusTodo)
	if err != nil {
		return nil, err
	}

	// 简单按优先级安排日程
	return s.simpleSchedule(tasks, startTime, endTime)
}

func (s *ScheduleService) simpleSchedule(tasks []model.Task, startTime, endTime string) ([]ScheduleEvent, error) {
	// 简单的日程安排逻辑
	var events []ScheduleEvent

	start, err := time.Parse("15:04", startTime)
	if err != nil {
		return nil, err
	}

	for _, task := range tasks {
		duration := 30 * time.Minute // 默认 30 分钟
		if task.EstimatedTime > 0 {
			duration = time.Duration(task.EstimatedTime) * time.Minute
		}

		now := time.Now()
		scheduleStart := time.Date(now.Year(), now.Month(), now.Day(), start.Hour(), start.Minute(), 0, 0, now.Location())
		scheduleEnd := scheduleStart.Add(duration)

		dto := &CreateScheduleDTO{
			TaskID:    task.ID,
			Title:     task.Title,
			StartTime: scheduleStart.Format(time.RFC3339),
			EndTime:   scheduleEnd.Format(time.RFC3339),
			Type:      "task",
			Color:     s.getQuadrantColor(task.Quadrant),
		}

		schedule, err := s.CreateSchedule(dto)
		if err != nil {
			continue
		}

		events = append(events, s.toEvent(schedule))
		start = scheduleEnd
	}

	return events, nil
}

func (s *ScheduleService) toEvent(schedule *model.Schedule) ScheduleEvent {
	event := ScheduleEvent{
		ID:       schedule.ID,
		Title:    schedule.Title,
		Start:    schedule.StartTime.Format(time.RFC3339),
		End:      schedule.EndTime.Format(time.RFC3339),
		Type:     string(schedule.Type),
		Status:   string(schedule.Status),
		Color:    schedule.Color,
		AllDay:   false,
		Editable: schedule.Status != model.ScheduleStatusCompleted,
	}

	if schedule.TaskID != nil {
		event.TaskID = *schedule.TaskID
	}

	return event
}

func (s *ScheduleService) getDefaultColor(scheduleType model.ScheduleType) string {
	switch scheduleType {
	case model.ScheduleTypeTask:
		return "#3b82f6"
	case model.ScheduleTypePomodoro:
		return "#f59e0b"
	case model.ScheduleTypeBreak:
		return "#22c55e"
	default:
		return "#6b7280"
	}
}

func (s *ScheduleService) getQuadrantColor(quadrant model.Quadrant) string {
	switch quadrant {
	case model.Quadrant1:
		return "#ef4444"
	case model.Quadrant2:
		return "#f59e0b"
	case model.Quadrant3:
		return "#3b82f6"
	case model.Quadrant4:
		return "#6b7280"
	default:
		return "#3b82f6"
	}
}