package handler

import (
	"net/http"
	"strconv"
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

// ClassifyByTextRequest 根据文本分类请求
type ClassifyByTextRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

// RescheduleRequest 重排程请求
type RescheduleRequest struct {
	TaskID           string `json:"task_id" binding:"required"`
	CompletedMinutes int    `json:"completed_minutes"`
	PlannedMinutes   int    `json:"planned_minutes" binding:"required"`
	InterruptReason  string `json:"interrupt_reason"`
	WorkEndTime      string `json:"work_end_time" binding:"required"`
}

// DailyInsightsRequest 每日洞察请求
type DailyInsightsRequest struct {
	Date                 string `json:"date"`
	CompletedPomodoros   int    `json:"completed_pomodoros"`
	TotalFocusMinutes    int    `json:"total_focus_minutes"`
	CompletedTasks       int    `json:"completed_tasks"`
	TotalInterruptions   int    `json:"total_interruptions"`
	TaskDistribution     string `json:"task_distribution"`
}

// ClassifyTaskByText 根据文本智能分类任务
func (h *AIHandler) ClassifyTaskByText(c *gin.Context) {
	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "AI service not configured. Please set API key in settings.",
		})
		return
	}

	var req ClassifyByTextRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx, cancel := service.GetAIServiceWithTimeout(30 * time.Second)
	defer cancel()

	result, err := h.aiService.ClassifyTaskByText(ctx, req.Title, req.Description)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// RescheduleAfterInterrupt 被打断后 AI 重新排程
func (h *AIHandler) RescheduleAfterInterrupt(c *gin.Context) {
	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "AI service not configured. Please set API key in settings.",
		})
		return
	}

	var req RescheduleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	currentTime := time.Now().Format("15:04")

	ctx, cancel := service.GetAIServiceWithTimeout(60 * time.Second)
	defer cancel()

	result, err := h.aiService.RescheduleAfterInterrupt(
		ctx, req.TaskID, req.CompletedMinutes, req.PlannedMinutes,
		req.InterruptReason, currentTime, req.WorkEndTime,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetDailyInsights 获取每日 AI 洞察
func (h *AIHandler) GetDailyInsights(c *gin.Context) {
	if !h.aiService.IsConfigured() {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "AI service not configured. Please set API key in settings.",
		})
		return
	}

	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}

	// 从查询参数读取统计数据
	completedPomodoros, _ := strconv.Atoi(c.DefaultQuery("completed_pomodoros", "0"))
	totalFocusMinutes, _ := strconv.Atoi(c.DefaultQuery("total_focus_minutes", "0"))
	completedTasks, _ := strconv.Atoi(c.DefaultQuery("completed_tasks", "0"))
	totalInterruptions, _ := strconv.Atoi(c.DefaultQuery("total_interruptions", "0"))
	taskDistribution := c.DefaultQuery("task_distribution", "")

	ctx, cancel := service.GetAIServiceWithTimeout(30 * time.Second)
	defer cancel()

	result, err := h.aiService.GetDailyInsights(
		ctx, date, completedPomodoros, totalFocusMinutes,
		completedTasks, totalInterruptions, taskDistribution,
	)
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