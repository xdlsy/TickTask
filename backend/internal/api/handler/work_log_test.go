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
		Items:   []service.StructuredItem{{Title: "T1", Content: "c1"}},
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
