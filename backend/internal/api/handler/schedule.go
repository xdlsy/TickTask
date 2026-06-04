package handler

import (
	"net/http"
	"ticktask/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

type ScheduleHandler struct {
	scheduleService *service.ScheduleService
}

func NewScheduleHandler(scheduleService *service.ScheduleService) *ScheduleHandler {
	return &ScheduleHandler{scheduleService: scheduleService}
}

// GetSchedules 获取日程列表
func (h *ScheduleHandler) GetSchedules(c *gin.Context) {
	startStr := c.Query("start")
	endStr := c.Query("end")

	var start, end time.Time
	var err error

	if startStr != "" {
		start, err = time.Parse("2006-01-02", startStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start date format"})
			return
		}
	} else {
		start = time.Now().Truncate(24 * time.Hour).AddDate(0, 0, -7)
	}

	if endStr != "" {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
			return
		}
		end = end.Add(24 * time.Hour) // 包含结束日期当天
	} else {
		end = start.AddDate(0, 0, 14)
	}

	events, err := h.scheduleService.GetSchedules(start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events})
}

// GetSchedule 获取单个日程
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	id := c.Param("id")

	schedule, err := h.scheduleService.GetSchedule(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "schedule not found"})
		return
	}

	c.JSON(http.StatusOK, schedule)
}

// CreateSchedule 创建日程
func (h *ScheduleHandler) CreateSchedule(c *gin.Context) {
	var dto service.CreateScheduleDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 验证必填字段
	if dto.Title == "" && dto.TaskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title or task_id is required"})
		return
	}

	if dto.StartTime == "" || dto.EndTime == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "start_time and end_time are required"})
		return
	}

	// 默认类型
	if dto.Type == "" {
		dto.Type = "task"
	}

	event, err := h.scheduleService.CreateScheduleEvent(&dto)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, event)
}

// UpdateSchedule 更新日程
func (h *ScheduleHandler) UpdateSchedule(c *gin.Context) {
	id := c.Param("id")

	var dto service.UpdateScheduleDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.scheduleService.UpdateSchedule(id, &dto); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "schedule updated"})
}

// DeleteSchedule 删除日程
func (h *ScheduleHandler) DeleteSchedule(c *gin.Context) {
	id := c.Param("id")

	if err := h.scheduleService.DeleteSchedule(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "schedule deleted"})
}

// MoveSchedule 移动日程
func (h *ScheduleHandler) MoveSchedule(c *gin.Context) {
	id := c.Param("id")

	var dto service.MoveScheduleDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	startTime, err := time.Parse(time.RFC3339, dto.StartTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid start_time format"})
		return
	}

	endTime, err := time.Parse(time.RFC3339, dto.EndTime)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end_time format"})
		return
	}

	if err := h.scheduleService.MoveSchedule(id, startTime, endTime); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "schedule moved"})
}

// GenerateWithAI AI生成日程
func (h *ScheduleHandler) GenerateWithAI(c *gin.Context) {
	var req struct {
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.StartTime == "" || req.EndTime == "" {
		// 使用默认值
		req.StartTime = "09:00"
		req.EndTime = "18:00"
	}

	events, reasoning, err := h.scheduleService.GenerateSchedule(req.StartTime, req.EndTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"events": events, "reasoning": reasoning})
}

// ReviseWithAI AI 修订日程 — returns a preview of changes without applying.
func (h *ScheduleHandler) ReviseWithAI(c *gin.Context) {
	var req struct {
		Prompt string `json:"prompt"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "prompt is required"})
		return
	}

	response, err := h.scheduleService.ReviseSchedule(req.Prompt)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"applied": response.Applied,
		"summary": response.Summary,
		"changes": response.Changes,
		"events":  response.Events,
	})
}

// ApplyRevision 确认应用修订 — persists the revised schedule to the database.
func (h *ScheduleHandler) ApplyRevision(c *gin.Context) {
	events, err := h.scheduleService.ApplyRevision()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"applied": true,
		"events":  events,
	})
}

// DeleteAll 清空所有日程
func (h *ScheduleHandler) DeleteAll(c *gin.Context) {
	deleted, err := h.scheduleService.DeleteAllSchedules()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"deleted": deleted})
}