package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/internal/service"
)

// WorkLogHandler 工作日志 HTTP 处理器
type WorkLogHandler struct {
	svc *service.WorkLogService
}

// NewWorkLogHandler 构造
func NewWorkLogHandler(svc *service.WorkLogService) *WorkLogHandler {
	return &WorkLogHandler{svc: svc}
}

// ── 日报端点 ──

// GetTodayContext GET /api/work-logs/today/context?date=YYYY-MM-DD
func (h *WorkLogHandler) GetTodayContext(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	ctx, err := h.svc.GetTodayContext(date)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.HasPrefix(err.Error(), "invalid date:") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ctx)
}

type structureRequest struct {
	BrainDump string               `json:"brain_dump"`
	Context   service.TodayContext `json:"context"`
}

// Structure POST /api/work-logs/structure — AI 拆条（不落库）
func (h *WorkLogHandler) Structure(c *gin.Context) {
	var req structureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.BrainDump) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "brain_dump required"})
		return
	}
	out, err := h.svc.StructureBrainDump(service.BrainDumpInput{
		BrainDump: req.BrainDump,
		Context:   req.Context,
	})
	if err != nil {
		if errors.Is(err, service.ErrAIStructureFailed) {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// CreateWorkLog POST /api/work-logs
func (h *WorkLogHandler) CreateWorkLog(c *gin.Context) {
	var req service.SaveWorkLogInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log, err := h.svc.SaveWorkLog(req)
	if err != nil {
		if errors.Is(err, service.ErrWorkLogAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error":             err.Error(),
				"existing_work_log": log,
			})
			return
		}
		status := http.StatusInternalServerError
		if strings.HasPrefix(err.Error(), "invalid date:") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, log)
}

// ListWorkLogs GET /api/work-logs?from=YYYY-MM-DD&to=YYYY-MM-DD
func (h *WorkLogHandler) ListWorkLogs(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to required"})
		return
	}
	logs, err := h.svc.ListWorkLogs(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

// GetWorkLog GET /api/work-logs/:date
func (h *WorkLogHandler) GetWorkLog(c *gin.Context) {
	date := c.Param("date")
	log, err := h.svc.GetWorkLog(date)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, log)
}

// UpdateWorkLog PUT /api/work-logs/:date
func (h *WorkLogHandler) UpdateWorkLog(c *gin.Context) {
	date := c.Param("date")
	var req service.SaveWorkLogInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Date = date
	log, err := h.svc.UpdateWorkLog(req)
	if err != nil {
		status := http.StatusInternalServerError
		if strings.HasPrefix(err.Error(), "invalid date:") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, log)
}

// ── 报告端点 ──

// GenerateReport POST /api/work-reports/generate
func (h *WorkLogHandler) GenerateReport(c *gin.Context) {
	var req service.GenerateReportInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	report, err := h.svc.GenerateReport(req)
	if err != nil {
		if errors.Is(err, service.ErrReportAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		// M1 stub returns plain errors.New(...) — surface as 502 (M4 will return proper reports)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, report)
}

// GetReport GET /api/work-reports/:type/:periodKey
func (h *WorkLogHandler) GetReport(c *gin.Context) {
	t := model.WorkReportType(c.Param("type"))
	periodKey := c.Param("periodKey")
	if t == "" || periodKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type and periodKey required"})
		return
	}
	report, err := h.svc.GetReport(t, periodKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

// ListReports GET /api/work-reports?type=weekly|monthly|halfyear|yearly
func (h *WorkLogHandler) ListReports(c *gin.Context) {
	t := model.WorkReportType(c.Query("type"))
	if t == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type required"})
		return
	}
	reports, err := h.svc.ListReports(t)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

// ── 快捷录入端点 ──

type createQuickEntryInput struct {
	Activity      string `json:"activity" binding:"required"`
	StartTime     string `json:"start_time" binding:"required"`
	EndTime       string `json:"end_time" binding:"required"`
	Quadrant      int    `json:"quadrant" binding:"required,min=1,max=4"`
	Content       string `json:"content"`
	ProblemSolved string `json:"problem_solved"`
	Result        string `json:"result"`
	Impact        string `json:"impact"`
}

type updateQuickEntryInput struct {
	Activity      *string `json:"activity,omitempty"`
	StartTime     *string `json:"start_time,omitempty"`
	EndTime       *string `json:"end_time,omitempty"`
	Quadrant      *int    `json:"quadrant,omitempty" binding:"omitempty,min=1,max=4"`
	Content       *string `json:"content,omitempty"`
	ProblemSolved *string `json:"problem_solved,omitempty"`
	Result        *string `json:"result,omitempty"`
	Impact        *string `json:"impact,omitempty"`
}

// AddQuickEntry POST /api/work-logs/:date/items
func (h *WorkLogHandler) AddQuickEntry(c *gin.Context) {
	date := c.Param("date")
	var req createQuickEntryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.AddQuickEntry(date, service.CreateQuickEntryInput{
		Activity:      req.Activity,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		Quadrant:      req.Quadrant,
		Content:       req.Content,
		ProblemSolved: req.ProblemSolved,
		Result:        req.Result,
		Impact:        req.Impact,
	})
	if err != nil {
		status := mapQuickEntryErrorStatus(err)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

// UpdateQuickEntry PATCH /api/work-logs/:date/items/:itemId
func (h *WorkLogHandler) UpdateQuickEntry(c *gin.Context) {
	date := c.Param("date")
	itemID := c.Param("itemId")
	var req updateQuickEntryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.svc.UpdateQuickEntry(date, itemID, service.UpdateQuickEntryInput{
		Activity:      req.Activity,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		Quadrant:      req.Quadrant,
		Content:       req.Content,
		ProblemSolved: req.ProblemSolved,
		Result:        req.Result,
		Impact:        req.Impact,
	})
	if err != nil {
		c.JSON(mapQuickEntryErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DeleteQuickEntry DELETE /api/work-logs/:date/items/:itemId
func (h *WorkLogHandler) DeleteQuickEntry(c *gin.Context) {
	date := c.Param("date")
	itemID := c.Param("itemId")
	if err := h.svc.DeleteQuickEntry(date, itemID); err != nil {
		c.JSON(mapQuickEntryErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

type updateSummaryInput struct {
	Summary string `json:"summary"`
}

// UpdateSummary PATCH /api/work-logs/:date/summary
func (h *WorkLogHandler) UpdateSummary(c *gin.Context) {
	date := c.Param("date")
	var req updateSummaryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.svc.UpdateSummary(date, req.Summary)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		status := http.StatusInternalServerError
		if strings.HasPrefix(err.Error(), "invalid date:") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func mapQuickEntryErrorStatus(err error) int {
	if errors.Is(err, repository.ErrItemNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, repository.ErrItemNotEditable) {
		return http.StatusForbidden
	}
	if strings.HasPrefix(err.Error(), "invalid date:") {
		return http.StatusBadRequest
	}
	if strings.HasPrefix(err.Error(), "invalid time format:") {
		return http.StatusBadRequest
	}
	if err.Error() == "end_time must be after start_time" {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
