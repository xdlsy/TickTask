package handler

import (
	"net/http"
	"ticktask/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

type AIHandler struct {
	aiService   *service.AIService
	taskService *service.TaskService
}

func NewAIHandler(aiService *service.AIService, taskService *service.TaskService) *AIHandler {
	return &AIHandler{
		aiService:   aiService,
		taskService: taskService,
	}
}

// ClassifyTaskRequest 分类任务请求
type ClassifyTaskRequest struct {
	TaskID string `json:"task_id" binding:"required"`
}

// ClassifyTasksRequest 批量分类请求
type ClassifyTasksRequest struct {
	TaskIDs []string `json:"task_ids" binding:"required"`
}

// ScheduleRequest 日程生成请求
type ScheduleRequest struct {
	StartTime string `json:"start_time" binding:"required"` // HH:MM 格式
	EndTime   string `json:"end_time" binding:"required"`   // HH:MM 格式
}

// ClassifyTask 智能分类单个任务
func (h *AIHandler) ClassifyTask(c *gin.Context) {
	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "AI service not configured. Please set API key in settings.",
		})
		return
	}

	var req ClassifyTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	task, err := h.taskService.GetTask(req.TaskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "task not found"})
		return
	}

	ctx, cancel := service.GetAIServiceWithTimeout(30 * time.Second)
	defer cancel()

	result, err := h.aiService.ClassifyTask(ctx, task)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// ClassifyTasks 批量分类任务
func (h *AIHandler) ClassifyTasks(c *gin.Context) {
	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "AI service not configured. Please set API key in settings.",
		})
		return
	}

	var req ClassifyTasksRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := service.GetAIServiceWithTimeout(60 * time.Second)
	defer cancel()

	results, err := h.aiService.ClassifyTasks(ctx, req.TaskIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

// GenerateSchedule 生成今日日程
func (h *AIHandler) GenerateSchedule(c *gin.Context) {
	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "AI service not configured. Please set API key in settings.",
		})
		return
	}

	var req ScheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取番茄设置
	settings, err := h.taskService.GetPomodoroSettings()
	if err != nil {
		settings = nil
	}

	ctx, cancel := service.GetAIServiceWithTimeout(60 * time.Second)
	defer cancel()

	schedule, err := h.aiService.GenerateDailySchedule(ctx, req.StartTime, req.EndTime, settings)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// GetPrioritySuggestions 获取优先级建议
func (h *AIHandler) GetPrioritySuggestions(c *gin.Context) {
	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "AI service not configured. Please set API key in settings.",
		})
		return
	}

	ctx, cancel := service.GetAIServiceWithTimeout(30 * time.Second)
	defer cancel()

	result, err := h.aiService.GetPrioritySuggestions(ctx)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetAIStatus 获取 AI 服务状态
func (h *AIHandler) GetAIStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"configured": h.aiService.IsConfigured(),
	})
}