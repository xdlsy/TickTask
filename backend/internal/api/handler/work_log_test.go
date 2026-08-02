package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/internal/service"
)

func init() { gin.SetMode(gin.TestMode) }

// ── In-memory repo for handler tests (implements service.WorkLogRepository) ──

type handlerTestRepo struct {
	logs    map[string]*model.WorkLog
	reports map[string]*model.WorkReport
}

func newHandlerTestRepo() *handlerTestRepo {
	return &handlerTestRepo{
		logs:    make(map[string]*model.WorkLog),
		reports: make(map[string]*model.WorkReport),
	}
}

func (r *handlerTestRepo) CreateWorkLog(log *model.WorkLog) error {
	if _, ok := r.logs[log.Date]; ok {
		return errors.New("duplicate")
	}
	cp := *log
	r.logs[log.Date] = &cp
	return nil
}
func (r *handlerTestRepo) GetWorkLogByDate(date string) (*model.WorkLog, error) {
	if l, ok := r.logs[date]; ok {
		return l, nil
	}
	return nil, repository.ErrNotFound
}
func (r *handlerTestRepo) GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error) {
	var out []*model.WorkLog
	for d, l := range r.logs {
		if d >= from && d <= to {
			out = append(out, l)
		}
	}
	return out, nil
}
func (r *handlerTestRepo) UpsertWorkLog(log *model.WorkLog) error {
	cp := *log
	r.logs[log.Date] = &cp
	return nil
}
func (r *handlerTestRepo) ReplaceItems(workLogID string, items []model.WorkItem) error {
	for _, l := range r.logs {
		if l.ID == workLogID {
			l.Items = items
			return nil
		}
	}
	return repository.ErrNotFound
}
func (r *handlerTestRepo) CreateWorkReport(rep *model.WorkReport) error {
	key := string(rep.Type) + ":" + rep.PeriodKey
	if _, ok := r.reports[key]; ok {
		return errors.New("duplicate")
	}
	r.reports[key] = rep
	return nil
}
func (r *handlerTestRepo) UpdateWorkReport(rep *model.WorkReport) error {
	key := string(rep.Type) + ":" + rep.PeriodKey
	r.reports[key] = rep
	return nil
}
func (r *handlerTestRepo) GetWorkReportByTypeAndPeriod(t model.WorkReportType, k string) (*model.WorkReport, error) {
	if rep, ok := r.reports[string(t)+":"+k]; ok {
		return rep, nil
	}
	return nil, repository.ErrNotFound
}
func (r *handlerTestRepo) ListWorkReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	prefix := string(t) + ":"
	var out []*model.WorkReport
	for k, rep := range r.reports {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, rep)
		}
	}
	return out, nil
}

func (r *handlerTestRepo) AppendItem(workLogID string, item model.WorkItem) error {
	return errors.New("AppendItem not supported in this mock")
}

func (r *handlerTestRepo) UpdateItem(workLogID string, itemID string, updates map[string]any) error {
	return errors.New("UpdateItem not supported in this mock")
}

func (r *handlerTestRepo) DeleteItem(workLogID string, itemID string) error {
	return errors.New("DeleteItem not supported in this mock")
}

func (r *handlerTestRepo) UpdateWorkLogSummary(date string, summary string) error {
	return errors.New("UpdateWorkLogSummary not supported in this mock")
}

// ── Mock AI client for handler tests ──

type handlerMockAIClient struct {
	structuredOut *service.StructuredWorkLog
	structuredErr error
}

func (m *handlerMockAIClient) StructureBrainDump(input service.BrainDumpInput) (*service.StructuredWorkLog, error) {
	if m.structuredErr != nil {
		return nil, m.structuredErr
	}
	return m.structuredOut, nil
}
func (m *handlerMockAIClient) GenerateWeeklyReport(items []model.WorkItem, start, end string) (*service.ReportSummary, error) {
	return nil, errors.New("not impl")
}
func (m *handlerMockAIClient) GenerateMonthlyReport(w []*model.WorkReport, o []model.WorkItem, start, end string) (*service.ReportSummary, error) {
	return nil, errors.New("not impl")
}
func (m *handlerMockAIClient) GenerateHalfYearReport(mo []*model.WorkReport, start, end string) (*service.ReportSummary, error) {
	return nil, errors.New("not impl")
}
func (m *handlerMockAIClient) GenerateYearlyReport(mo []*model.WorkReport, start, end string) (*service.ReportSummary, error) {
	return nil, errors.New("not impl")
}

// ── Helpers ──

func newTestWorkLogRouter(svc *service.WorkLogService) *gin.Engine {
	r := gin.New()
	h := NewWorkLogHandler(svc)
	r.GET("/api/work-logs/today/context", h.GetTodayContext)
	r.POST("/api/work-logs/structure", h.Structure)
	r.GET("/api/work-logs", h.ListWorkLogs)
	r.POST("/api/work-logs", h.CreateWorkLog)
	r.GET("/api/work-logs/:date", h.GetWorkLog)
	r.PUT("/api/work-logs/:date", h.UpdateWorkLog)
	r.POST("/api/work-reports/generate", h.GenerateReport)
	r.GET("/api/work-reports", h.ListReports)
	r.GET("/api/work-reports/:type/:periodKey", h.GetReport)
	return r
}

func newHandlerService(repo *handlerTestRepo, ai service.WorkLogAIClient) *service.WorkLogService {
	return service.NewWorkLogService(repo, nil, nil, ai)
}

func doJSON(t *testing.T, router *gin.Engine, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("marshal body: %v", err)
		}
		buf = *bytes.NewBuffer(b)
	}
	req, _ := http.NewRequest(method, path, &buf)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

// ── CreateWorkLog tests ──

func TestHandler_CreateWorkLog_Happy(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, nil))

	body := service.SaveWorkLogInput{
		Date:    "2026-08-02",
		Summary: "s",
		Items:   []service.SaveItemInput{{Seq: 1, Title: "T1", Content: "c1"}},
	}
	w := doJSON(t, router, "POST", "/api/work-logs", body)
	if w.Code != http.StatusCreated {
		t.Fatalf("code = %d, want 201; body = %s", w.Code, w.Body.String())
	}
	var got model.WorkLog
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Date != "2026-08-02" || len(got.Items) != 1 {
		t.Errorf("unexpected log: %+v", got)
	}
}

func TestHandler_CreateWorkLog_Conflict(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, nil))
	body := service.SaveWorkLogInput{Date: "2026-08-02", Items: []service.SaveItemInput{{Seq: 1, Title: "T1"}}}

	if w := doJSON(t, router, "POST", "/api/work-logs", body); w.Code != http.StatusCreated {
		t.Fatalf("first create code = %d, want 201; body = %s", w.Code, w.Body.String())
	}
	w := doJSON(t, router, "POST", "/api/work-logs", body)
	if w.Code != http.StatusConflict {
		t.Errorf("second create code = %d, want 409; body = %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte("existing_work_log")) {
		t.Errorf("conflict body should include existing_work_log; got %s", w.Body.String())
	}
}

func TestHandler_CreateWorkLog_InvalidDate(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, nil))
	body := service.SaveWorkLogInput{Date: "2026-02-30"}
	w := doJSON(t, router, "POST", "/api/work-logs", body)
	if w.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

// ── GetWorkLog tests ──

func TestHandler_GetWorkLog_Happy(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, nil))
	doJSON(t, router, "POST", "/api/work-logs", service.SaveWorkLogInput{Date: "2026-08-02"})

	w := doJSON(t, router, "GET", "/api/work-logs/2026-08-02", nil)
	if w.Code != http.StatusOK {
		t.Errorf("code = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetWorkLog_Missing(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, nil))

	w := doJSON(t, router, "GET", "/api/work-logs/2026-08-02", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404; body = %s", w.Code, w.Body.String())
	}
}

// ── ListWorkLogs tests ──

func TestHandler_ListWorkLogs_MissingParams(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, nil))

	w := doJSON(t, router, "GET", "/api/work-logs", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", w.Code)
	}
}

func TestHandler_ListWorkLogs_Happy(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, nil))
	doJSON(t, router, "POST", "/api/work-logs", service.SaveWorkLogInput{Date: "2026-08-02"})

	w := doJSON(t, router, "GET", "/api/work-logs?from=2026-08-01&to=2026-08-31", nil)
	if w.Code != http.StatusOK {
		t.Errorf("code = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"logs":`)) {
		t.Errorf("body should contain 'logs' key; got %s", w.Body.String())
	}
}

// ── UpdateWorkLog tests ──

func TestHandler_UpdateWorkLog_Happy(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, nil))
	doJSON(t, router, "POST", "/api/work-logs", service.SaveWorkLogInput{Date: "2026-08-02"})

	w := doJSON(t, router, "PUT", "/api/work-logs/2026-08-02", service.SaveWorkLogInput{
		Items: []service.SaveItemInput{{Seq: 1, Title: "T1"}, {Seq: 2, Title: "T2"}},
	})
	if w.Code != http.StatusOK {
		t.Errorf("code = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

// ── Structure tests ──

func TestHandler_Structure_EmptyBody(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, &handlerMockAIClient{}))

	w := doJSON(t, router, "POST", "/api/work-logs/structure", map[string]interface{}{
		"brain_dump": "  ",
		"context":    map[string]interface{}{},
	})
	if w.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestHandler_Structure_Happy(t *testing.T) {
	repo := newHandlerTestRepo()
	ai := &handlerMockAIClient{structuredOut: &service.StructuredWorkLog{
		Items:   []service.StructuredItem{{Content: "c1"}},
		Summary: "今日小结",
	}}
	router := newTestWorkLogRouter(newHandlerService(repo, ai))

	w := doJSON(t, router, "POST", "/api/work-logs/structure", map[string]interface{}{
		"brain_dump": "今天做了 T1",
		"context":    map[string]interface{}{},
	})
	if w.Code != http.StatusOK {
		t.Errorf("code = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"summary":"今日小结"`)) {
		t.Errorf("body should contain summary; got %s", w.Body.String())
	}
}

func TestHandler_Structure_AIFail_502(t *testing.T) {
	repo := newHandlerTestRepo()
	ai := &handlerMockAIClient{structuredErr: errors.New("upstream")}
	router := newTestWorkLogRouter(newHandlerService(repo, ai))

	w := doJSON(t, router, "POST", "/api/work-logs/structure", map[string]interface{}{
		"brain_dump": "x",
		"context":    map[string]interface{}{},
	})
	if w.Code != http.StatusBadGateway {
		t.Errorf("code = %d, want 502; body = %s", w.Code, w.Body.String())
	}
}

// ── GenerateReport tests ──

func TestHandler_GenerateReport_Stub_502(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, nil))

	body := service.GenerateReportInput{Type: model.ReportWeekly, PeriodKey: "2026-W31"}
	w := doJSON(t, router, "POST", "/api/work-reports/generate", body)
	if w.Code != http.StatusBadGateway {
		t.Errorf("code = %d, want 502 (M1 stub); body = %s", w.Code, w.Body.String())
	}
}

// ── ListReports tests ──

func TestHandler_ListReports_MissingType(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, nil))

	w := doJSON(t, router, "GET", "/api/work-reports", nil)
	if w.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400", w.Code)
	}
}

func TestHandler_ListReports_Happy(t *testing.T) {
	repo := newHandlerTestRepo()
	repo.reports["weekly:2026-W31"] = &model.WorkReport{
		ID:        "r1",
		Type:      model.ReportWeekly,
		PeriodKey: "2026-W31",
	}
	router := newTestWorkLogRouter(newHandlerService(repo, nil))

	w := doJSON(t, router, "GET", "/api/work-reports?type=weekly", nil)
	if w.Code != http.StatusOK {
		t.Errorf("code = %d, want 200; body = %s", w.Code, w.Body.String())
	}
	if !bytes.Contains(w.Body.Bytes(), []byte(`"reports":`)) {
		t.Errorf("body should contain 'reports' key; got %s", w.Body.String())
	}
}

// ── GetReport tests ──

func TestHandler_GetReport_Happy(t *testing.T) {
	repo := newHandlerTestRepo()
	repo.reports["weekly:2026-W31"] = &model.WorkReport{
		ID:        "r1",
		Type:      model.ReportWeekly,
		PeriodKey: "2026-W31",
	}
	router := newTestWorkLogRouter(newHandlerService(repo, nil))

	w := doJSON(t, router, "GET", "/api/work-reports/weekly/2026-W31", nil)
	if w.Code != http.StatusOK {
		t.Errorf("code = %d, want 200; body = %s", w.Code, w.Body.String())
	}
}

func TestHandler_GetReport_Missing(t *testing.T) {
	repo := newHandlerTestRepo()
	router := newTestWorkLogRouter(newHandlerService(repo, nil))

	w := doJSON(t, router, "GET", "/api/work-reports/weekly/2026-W31", nil)
	if w.Code != http.StatusNotFound {
		t.Errorf("code = %d, want 404", w.Code)
	}
}

// ── Quick entry handler tests ──

func TestAddQuickEntry_HandlerHappyPath(t *testing.T) {
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r := gin.New()
	r.POST("/api/work-logs/:date/items", h.AddQuickEntry)

	body, _ := json.Marshal(map[string]any{
		"activity": "晨会", "start_time": "09:00", "end_time": "10:00", "quadrant": 1,
	})
	req := httptest.NewRequest("POST", "/api/work-logs/2026-08-02/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp model.WorkItem
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Source != "manual" {
		t.Fatalf("bad source: %s", resp.Source)
	}
	if resp.Activity == nil || *resp.Activity != "晨会" {
		t.Fatalf("bad activity: %+v", resp.Activity)
	}
	if resp.StartTime == nil || *resp.StartTime != "09:00" {
		t.Fatalf("bad start_time: %+v", resp.StartTime)
	}
	if resp.EndTime == nil || *resp.EndTime != "10:00" {
		t.Fatalf("bad end_time: %+v", resp.EndTime)
	}
	if resp.Quadrant == nil || *resp.Quadrant != 1 {
		t.Fatalf("bad quadrant: %+v", resp.Quadrant)
	}
}

func TestAddQuickEntry_HandlerRejectsBadTime(t *testing.T) {
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r := gin.New()
	r.POST("/api/work-logs/:date/items", h.AddQuickEntry)

	body, _ := json.Marshal(map[string]any{
		"activity": "x", "start_time": "11:00", "end_time": "10:00", "quadrant": 1,
	})
	req := httptest.NewRequest("POST", "/api/work-logs/2026-08-02/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteQuickEntry_HandlerReturns403ForAIItem(t *testing.T) {
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r := gin.New()
	r.DELETE("/api/work-logs/:date/items/:itemId", h.DeleteQuickEntry)

	aiItem := model.WorkItem{ID: "ai-1", WorkLogID: "log-1", Source: "ai", Title: "x"}
	repo.logs["2026-08-02"] = &model.WorkLog{ID: "log-1", Date: "2026-08-02", Items: []model.WorkItem{aiItem}}
	repo.items["ai-1"] = &aiItem

	req := httptest.NewRequest("DELETE", "/api/work-logs/2026-08-02/items/ai-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// ── UpdateQuickEntry handler tests ──

func TestUpdateQuickEntry_HandlerHappyPath(t *testing.T) {
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r := gin.New()
	r.POST("/api/work-logs/:date/items", h.AddQuickEntry)
	r.PATCH("/api/work-logs/:date/items/:itemId", h.UpdateQuickEntry)

	// 先用 AddQuickEntry 创建一条
	createBody, _ := json.Marshal(map[string]any{
		"activity": "old", "start_time": "09:00", "end_time": "10:00", "quadrant": 1,
	})
	req1 := httptest.NewRequest("POST", "/api/work-logs/2026-08-02/items", bytes.NewReader(createBody))
	req1.Header.Set("Content-Type", "application/json")
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusCreated {
		t.Fatalf("setup create failed: %d %s", w1.Code, w1.Body.String())
	}
	var created model.WorkItem
	json.Unmarshal(w1.Body.Bytes(), &created)
	if created.ID == "" {
		t.Fatalf("no item ID returned: %+v", created)
	}

	// PATCH 改 activity
	updateBody, _ := json.Marshal(map[string]any{"activity": "new"})
	req2 := httptest.NewRequest("PATCH", "/api/work-logs/2026-08-02/items/"+created.ID, bytes.NewReader(updateBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	if w2.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w2.Code, w2.Body.String())
	}
}

func TestUpdateQuickEntry_HandlerReturns404ForMissingItem(t *testing.T) {
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r := gin.New()
	r.PATCH("/api/work-logs/:date/items/:itemId", h.UpdateQuickEntry)

	body, _ := json.Marshal(map[string]any{"activity": "x"})
	req := httptest.NewRequest("PATCH", "/api/work-logs/2026-08-02/items/nonexistent", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateQuickEntry_HandlerReturns403ForAIItem(t *testing.T) {
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r := gin.New()
	r.PATCH("/api/work-logs/:date/items/:itemId", h.UpdateQuickEntry)

	aiItem := model.WorkItem{ID: "ai-1", WorkLogID: "log-1", Source: "ai", Title: "x"}
	repo.logs["2026-08-02"] = &model.WorkLog{ID: "log-1", Date: "2026-08-02", Items: []model.WorkItem{aiItem}}
	repo.items["ai-1"] = &aiItem

	body, _ := json.Marshal(map[string]any{"activity": "y"})
	req := httptest.NewRequest("PATCH", "/api/work-logs/2026-08-02/items/ai-1", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

// ── mapQuickEntryErrorStatus unit tests ──

func TestMapQuickEntryErrorStatus_Default500(t *testing.T) {
	// 未识别的错误应回落到 500，避免把任意内部错误暴露为 4xx
	err := errors.New("some unexpected internal error")
	if got := mapQuickEntryErrorStatus(err); got != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", got)
	}
}

// ── Optional fields + UpdateSummary tests ──

func TestAddQuickEntry_WithOptionalFields(t *testing.T) {
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r := gin.New()
	r.POST("/api/work-logs/:date/items", h.AddQuickEntry)

	body, _ := json.Marshal(map[string]any{
		"activity":       "x",
		"start_time":     "09:00",
		"end_time":       "10:00",
		"quadrant":       1,
		"content":        "可选内容",
		"problem_solved": "解决了 Y",
		"result":         "产出了 Z",
		"impact":         "影响了 W",
	})
	req := httptest.NewRequest("POST", "/api/work-logs/2026-08-03/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp model.WorkItem
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Content != "可选内容" {
		t.Errorf("expected content '可选内容', got %q", resp.Content)
	}
	if resp.ProblemSolved != "解决了 Y" {
		t.Errorf("expected problem_solved, got %q", resp.ProblemSolved)
	}
	if resp.Result != "产出了 Z" {
		t.Errorf("expected result, got %q", resp.Result)
	}
	if resp.Impact != "影响了 W" {
		t.Errorf("expected impact, got %q", resp.Impact)
	}
	if resp.Title != "x" {
		t.Errorf("expected title synced from activity, got %q", resp.Title)
	}
}

func TestUpdateSummary_HandlerSuccess(t *testing.T) {
	repo := newMockWorkLogRepository()
	repo.logs["2026-08-03"] = &model.WorkLog{ID: "log-1", Date: "2026-08-03", Summary: "old"}
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r := gin.New()
	r.PATCH("/api/work-logs/:date/summary", h.UpdateSummary)

	body, _ := json.Marshal(map[string]any{"summary": "今日小结"})
	req := httptest.NewRequest("PATCH", "/api/work-logs/2026-08-03/summary", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if repo.logs["2026-08-03"].Summary != "今日小结" {
		t.Errorf("summary not updated: %q", repo.logs["2026-08-03"].Summary)
	}
}

func TestUpdateSummary_HandlerReturns404ForMissingLog(t *testing.T) {
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r := gin.New()
	r.PATCH("/api/work-logs/:date/summary", h.UpdateSummary)

	body, _ := json.Marshal(map[string]any{"summary": "x"})
	req := httptest.NewRequest("PATCH", "/api/work-logs/2099-12-31/summary", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestUpdateSummary_HandlerReturns400ForInvalidDate(t *testing.T) {
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r := gin.New()
	r.PATCH("/api/work-logs/:date/summary", h.UpdateSummary)

	body, _ := json.Marshal(map[string]any{"summary": "x"})
	req := httptest.NewRequest("PATCH", "/api/work-logs/not-a-date/summary", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
