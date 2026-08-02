package handler

import (
	"net/http"
	"strconv"
	"ticktask/internal/service"
	"time"

	"github.com/gin-gonic/gin"
)

type AnalyticsHandler struct {
	analyticsService *service.AnalyticsService
}

func NewAnalyticsHandler(analyticsService *service.AnalyticsService) *AnalyticsHandler {
	return &AnalyticsHandler{analyticsService: analyticsService}
}

// GetSummary 获取日期概览
func (h *AnalyticsHandler) GetSummary(c *gin.Context) {
	dateStr := c.Query("date")
	var date time.Time

	if dateStr != "" {
		var err error
		date, err = time.Parse("2006-01-02", dateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid date format"})
			return
		}
	} else {
		date = time.Now().Truncate(24 * time.Hour)
	}

	summary, err := h.analyticsService.GetSummary(date)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

// GetTrend 获取趋势数据
func (h *AnalyticsHandler) GetTrend(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days < 1 || days > 90 {
		days = 7
	}

	trend, err := h.analyticsService.GetTrend(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, trend)
}

// GetDistribution 获取任务分布统计
func (h *AnalyticsHandler) GetDistribution(c *gin.Context) {
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
		// 默认本周
		now := time.Now()
		start = now.AddDate(0, 0, -int(now.Weekday())+1).Truncate(24 * time.Hour)
	}

	if endStr != "" {
		end, err = time.Parse("2006-01-02", endStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid end date format"})
			return
		}
	} else {
		end = time.Now().Truncate(24 * time.Hour).Add(24 * time.Hour)
	}

	distribution, err := h.analyticsService.GetDistribution(start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, distribution)
}

// GetPomodoroByTask 获取按任务聚合的番茄钟统计（排行榜）
func (h *AnalyticsHandler) GetPomodoroByTask(c *gin.Context) {
	period := c.DefaultQuery("period", "week")
	if period != "week" && period != "month" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid period, use 'week' or 'month'"})
		return
	}

	result, err := h.analyticsService.GetPomodoroByTask(period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetPomodoroTrends 获取番茄钟计划 vs 实际趋势对比
func (h *AnalyticsHandler) GetPomodoroTrends(c *gin.Context) {
	period := c.DefaultQuery("period", "week")
	if period != "week" && period != "month" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid period, use 'week' or 'month'"})
		return
	}

	result, err := h.analyticsService.GetPomodoroTrends(period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}