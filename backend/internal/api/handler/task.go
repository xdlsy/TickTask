package handler

import (
	"net/http"
	"ticktask/internal/model"
	"ticktask/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

type TaskHandler struct {
	taskService *service.TaskService
}

func NewTaskHandler(taskService *service.TaskService) *TaskHandler {
	return &TaskHandler{taskService: taskService}
}

type CreateTaskInput struct {
	Title              string    `json:"title" binding:"required"`
	Description        string    `json:"description"`
	Quadrant           int       `json:"quadrant" binding:"required,min=1,max=4"`
	IsImportant        bool      `json:"is_important"`
	IsUrgent           bool      `json:"is_urgent"`
	EstimatedTime      int       `json:"estimated_time"`
	Deadline           time.Time `json:"deadline"`
	StartDate          time.Time `json:"start_date"`
	DueDate            time.Time `json:"due_date"`
	IsRecurring        bool      `json:"is_recurring"`
	RecurrencePattern  string    `json:"recurrence_pattern"`
	PreferredStartTime string    `json:"preferred_start_time"`
	PreferredEndTime   string    `json:"preferred_end_time"`
	Tags               []string  `json:"tags"`
}

type UpdateTaskInput struct {
	Title              *string    `json:"title"`
	Description        *string    `json:"description"`
	Quadrant           *int       `json:"quadrant"`
	IsImportant        *bool      `json:"is_important"`
	IsUrgent           *bool      `json:"is_urgent"`
	Status             *string    `json:"status"`
	EstimatedTime      *int       `json:"estimated_time"`
	Deadline           *time.Time `json:"deadline"`
	StartDate          *time.Time `json:"start_date"`
	DueDate            *time.Time `json:"due_date"`
	IsRecurring        *bool      `json:"is_recurring"`
	RecurrencePattern  *string    `json:"recurrence_pattern"`
	PreferredStartTime *string    `json:"preferred_start_time"`
	PreferredEndTime   *string    `json:"preferred_end_time"`
	Tags               []string   `json:"tags"`
	Order              *int       `json:"order"`
}

// GetTasks 获取任务列表
func (h *TaskHandler) GetTasks(c *gin.Context) {
	tasks, err := h.taskService.GetAllTaskResponses()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

// GetTasksByQuadrant 获取按象限分组的任务
func (h *TaskHandler) GetTasksByQuadrant(c *gin.Context) {
	tasks, err := h.taskService.GetTasksByQuadrantResponse()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, tasks)
}

// GetTask 获取单个任务
func (h *TaskHandler) GetTask(c *gin.Context) {
	id := c.Param("id")
	task, err := h.taskService.GetTaskResponse(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}
	c.JSON(http.StatusOK, task)
}

// CreateTask 创建任务
func (h *TaskHandler) CreateTask(c *gin.Context) {
	var input CreateTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var deadline *time.Time
	if !input.Deadline.IsZero() {
		deadline = &input.Deadline
	}
	var startDate *time.Time
	if !input.StartDate.IsZero() {
		startDate = &input.StartDate
	}
	var dueDate *time.Time
	if !input.DueDate.IsZero() {
		dueDate = &input.DueDate
	}

	req := service.CreateTaskRequest{
		Title:              input.Title,
		Description:        input.Description,
		Quadrant:           model.Quadrant(input.Quadrant),
		IsImportant:        input.IsImportant,
		IsUrgent:           input.IsUrgent,
		EstimatedTime:      input.EstimatedTime,
		Deadline:           deadline,
		StartDate:          startDate,
		DueDate:            dueDate,
		IsRecurring:        input.IsRecurring,
		RecurrencePattern:  input.RecurrencePattern,
		PreferredStartTime: input.PreferredStartTime,
		PreferredEndTime:   input.PreferredEndTime,
		Tags:               input.Tags,
	}

	task, err := h.taskService.CreateTask(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, task)
}

// UpdateTask 更新任务
func (h *TaskHandler) UpdateTask(c *gin.Context) {
	id := c.Param("id")
	var input UpdateTaskInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req := service.UpdateTaskRequest{}

	if input.Title != nil {
		req.Title = input.Title
	}
	if input.Description != nil {
		req.Description = input.Description
	}
	if input.Quadrant != nil {
		q := model.Quadrant(*input.Quadrant)
		req.Quadrant = &q
	}
	if input.IsImportant != nil {
		req.IsImportant = input.IsImportant
	}
	if input.IsUrgent != nil {
		req.IsUrgent = input.IsUrgent
	}
	if input.Status != nil {
		s := model.TaskStatus(*input.Status)
		req.Status = &s
	}
	if input.EstimatedTime != nil {
		req.EstimatedTime = input.EstimatedTime
	}
	if input.Deadline != nil {
		req.Deadline = input.Deadline
	}
	if input.StartDate != nil {
		req.StartDate = input.StartDate
	}
	if input.DueDate != nil {
		req.DueDate = input.DueDate
	}
	if input.IsRecurring != nil {
		req.IsRecurring = input.IsRecurring
	}
	if input.RecurrencePattern != nil {
		req.RecurrencePattern = input.RecurrencePattern
	}
	if input.PreferredStartTime != nil {
		req.PreferredStartTime = input.PreferredStartTime
	}
	if input.PreferredEndTime != nil {
		req.PreferredEndTime = input.PreferredEndTime
	}
	if input.Tags != nil {
		req.Tags = input.Tags
	}
	if input.Order != nil {
		req.Order = input.Order
	}

	if err := h.taskService.UpdateTask(id, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

// DeleteTask 删除任务
func (h *TaskHandler) DeleteTask(c *gin.Context) {
	id := c.Param("id")
	if err := h.taskService.DeleteTask(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

// MoveTask 移动任务到其他象限
func (h *TaskHandler) MoveTask(c *gin.Context) {
	id := c.Param("id")
	var input struct {
		Quadrant int `json:"quadrant" binding:"required,min=1,max=4"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.taskService.MoveTask(id, model.Quadrant(input.Quadrant)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "moved"})
}
