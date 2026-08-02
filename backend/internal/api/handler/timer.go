package handler

import (
	"net/http"
	"strconv"
	"ticktask/internal/model"
	"ticktask/internal/service"

	"github.com/gin-gonic/gin"
)

type TimerHandler struct {
	timerService *service.TimerService
}

func NewTimerHandler(timerService *service.TimerService) *TimerHandler {
	return &TimerHandler{timerService: timerService}
}

type CreateSessionInput struct {
	TaskID   *string `json:"task_id"`
	Type     string  `json:"type" binding:"required"` // work, short_break, long_break
	Duration int     `json:"duration"`                // 秒
}

type ControlSessionInput struct {
	Action          string `json:"action" binding:"required"` // pause, resume, complete, abandon
	InterruptReason string `json:"interrupt_reason"`          // meeting/call/urgent/other
}

// GetActiveSession 获取当前活跃会话
func (h *TimerHandler) GetActiveSession(c *gin.Context) {
	session, err := h.timerService.GetActiveSession()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"session": nil})
		return
	}
	c.JSON(http.StatusOK, session)
}

// CreateSession 创建并启动新会话
func (h *TimerHandler) CreateSession(c *gin.Context) {
	var input CreateSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	sessionType := model.SessionType(input.Type)
	req := service.CreateSessionRequest{
		TaskID:   input.TaskID,
		Type:     sessionType,
		Duration: input.Duration,
	}

	session, err := h.timerService.StartSession(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, session)
}

// ControlSession 控制会话
func (h *TimerHandler) ControlSession(c *gin.Context) {
	id := c.Param("id")
	var input ControlSessionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.timerService.ControlSession(id, input.Action, input.InterruptReason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "action completed"})
}

// GetRecentSessions 获取最近会话
func (h *TimerHandler) GetRecentSessions(c *gin.Context) {
	limit := 20
	if l := c.Query("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	sessions, err := h.timerService.GetRecentSessions(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, sessions)
}

// GetTodayTaskStats 获取今日各任务的投入统计
func (h *TimerHandler) GetTodayTaskStats(c *gin.Context) {
	stats, err := h.timerService.GetTodayTaskStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, stats)
}
