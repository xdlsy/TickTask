package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"ticktask/internal/ai"
	"ticktask/internal/model"
	"ticktask/internal/repository"
)

// ── Mock WorkLog repo ──

type mockWorkLogRepo struct {
	logs    map[string]*model.WorkLog
	reports map[string]*model.WorkReport
}

func newMockWorkLogRepo() *mockWorkLogRepo {
	return &mockWorkLogRepo{
		logs:    make(map[string]*model.WorkLog),
		reports: make(map[string]*model.WorkReport),
	}
}

func (m *mockWorkLogRepo) CreateWorkLog(log *model.WorkLog) error {
	if _, ok := m.logs[log.Date]; ok {
		return errors.New("duplicate")
	}
	cp := *log
	m.logs[log.Date] = &cp
	return nil
}

func (m *mockWorkLogRepo) GetWorkLogByDate(date string) (*model.WorkLog, error) {
	if l, ok := m.logs[date]; ok {
		return l, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockWorkLogRepo) GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error) {
	var out []*model.WorkLog
	for d, l := range m.logs {
		if d >= from && d <= to {
			out = append(out, l)
		}
	}
	return out, nil
}

func (m *mockWorkLogRepo) UpsertWorkLog(log *model.WorkLog) error {
	cp := *log
	m.logs[log.Date] = &cp
	return nil
}

func (m *mockWorkLogRepo) ReplaceItems(workLogID string, items []model.WorkItem) error {
	for _, l := range m.logs {
		if l.ID == workLogID {
			l.Items = items
			return nil
		}
	}
	return repository.ErrNotFound
}

func (m *mockWorkLogRepo) CreateWorkReport(r *model.WorkReport) error {
	key := string(r.Type) + ":" + r.PeriodKey
	if _, ok := m.reports[key]; ok {
		return errors.New("duplicate")
	}
	m.reports[key] = r
	return nil
}

func (m *mockWorkLogRepo) UpdateWorkReport(r *model.WorkReport) error {
	key := string(r.Type) + ":" + r.PeriodKey
	m.reports[key] = r
	return nil
}

func (m *mockWorkLogRepo) GetWorkReportByTypeAndPeriod(t model.WorkReportType, k string) (*model.WorkReport, error) {
	if r, ok := m.reports[string(t)+":"+k]; ok {
		return r, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockWorkLogRepo) ListWorkReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	prefix := string(t) + ":"
	var out []*model.WorkReport
	for k, r := range m.reports {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			out = append(out, r)
		}
	}
	return out, nil
}

func (m *mockWorkLogRepo) AppendItem(workLogID string, item model.WorkItem) error {
	return errors.New("AppendItem not supported in this mock")
}

func (m *mockWorkLogRepo) UpdateItem(workLogID string, itemID string, updates map[string]any) error {
	return errors.New("UpdateItem not supported in this mock")
}

func (m *mockWorkLogRepo) DeleteItem(workLogID string, itemID string) error {
	return errors.New("DeleteItem not supported in this mock")
}

func (m *mockWorkLogRepo) UpdateWorkLogSummary(date string, summary string) error {
	l, ok := m.logs[date]
	if !ok {
		return repository.ErrNotFound
	}
	l.Summary = summary
	return nil
}

// ── Mock LLM client ──
//
// WorkLogService 现在直接持有 ai.LLMClient（Task 15 移除 WorkLogAIClient 后）。
// 这个 mock 把所有 prompt 记录到 lastPrompt，并按 prompt 内容路由到不同的 canned
// 响应：
//   - 拆条 prompt（含 WorkLogStructureSystem）→ structureOut / structureErr
//   - 任意报告 prompt → 固定 ReportSummary JSON
//
// 测试通过断言 lastPrompt 的子串来验证：哪些 items / reports 被传给了 LLM。
// 这取代了原先的 weeklyInput / monthlyInput / halfYearInput / yearlyInput 字段。

type mockLLMForWorkLog struct {
	structureOut *StructuredWorkLog
	structureErr error
	lastPrompt   string
	allPrompts   []string
}

func (m *mockLLMForWorkLog) ChatCompletion(ctx context.Context, prompt string) (string, error) {
	m.lastPrompt = prompt
	m.allPrompts = append(m.allPrompts, prompt)
	if strings.HasPrefix(prompt, ai.WorkLogStructureSystem) {
		if m.structureErr != nil {
			return "", m.structureErr
		}
		if m.structureOut == nil {
			return "", nil
		}
		b, _ := json.Marshal(m.structureOut)
		return string(b), nil
	}
	// 报告类 prompt：返回固定 ReportSummary JSON
	resp := ReportSummary{CoreWork: "cw", MainProgress: "mp", OpenIssues: "oi", NextFocus: "nf"}
	b, _ := json.Marshal(resp)
	return string(b), nil
}

func (m *mockLLMForWorkLog) ChatWithTools(ctx context.Context, messages []ai.Message, tools []ai.ToolSpec) (ai.ToolResponse, error) {
	return ai.ToolResponse{}, ai.ErrFunctionCallNotSupported
}

// ── Mock Task / Session repos (interface-embedded to avoid full impl) ──

type mockTaskRepoForWorkLog struct {
	repository.TaskRepository // embedded nil; unimplemented methods nil-panic if called
	tasks                     []*model.Task
}

func (m *mockTaskRepoForWorkLog) GetCompletedTasksInRange(start, end time.Time) ([]*model.Task, error) {
	var out []*model.Task
	for _, t := range m.tasks {
		if t.CompletedAt != nil && !t.CompletedAt.Before(start) && t.CompletedAt.Before(end) {
			out = append(out, t)
		}
	}
	return out, nil
}

type mockSessionRepoForWorkLog struct {
	repository.SessionRepository
	sessions []model.PomodoroSession
}

func (m *mockSessionRepoForWorkLog) GetCompletedWorkByDateRange(start, end time.Time) ([]model.PomodoroSession, error) {
	var out []model.PomodoroSession
	for _, s := range m.sessions {
		if !s.StartTime.Before(start) && s.StartTime.Before(end) {
			out = append(out, s)
		}
	}
	return out, nil
}

// ── Service factory ──

func newServiceForTest() (*WorkLogService, *mockWorkLogRepo, *mockLLMForWorkLog) {
	repo := newMockWorkLogRepo()
	llm := &mockLLMForWorkLog{}
	svc := &WorkLogService{
		repo:        repo,
		llm:         llm,
		idGenerator: func() string { return "test-id" },
	}
	return svc, repo, llm
}

func newServiceWithTaskSessionForTest(taskRepo repository.TaskRepository, sessionRepo repository.SessionRepository) *WorkLogService {
	return &WorkLogService{
		repo:        newMockWorkLogRepo(),
		taskRepo:    taskRepo,
		sessionRepo: sessionRepo,
		llm:         &mockLLMForWorkLog{},
		idGenerator: func() string { return "test-id" },
	}
}

// ── Tests: SaveWorkLog ──

func TestSaveWorkLog_New(t *testing.T) {
	svc, _, _ := newServiceForTest()
	in := SaveWorkLogInput{
		Date:    "2026-08-02",
		Summary: "今日 ok",
		Items:   []SaveItemInput{{Seq: 1, Title: "T1", Content: "c1"}},
	}
	log, err := svc.SaveWorkLog(in)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if log.ID == "" || len(log.Items) != 1 {
		t.Errorf("unexpected log: %+v", log)
	}
}

func TestSaveWorkLog_DuplicateReturnsErr(t *testing.T) {
	svc, _, _ := newServiceForTest()
	in := SaveWorkLogInput{Date: "2026-08-02", Items: []SaveItemInput{{Seq: 1, Title: "T1"}}}
	if _, err := svc.SaveWorkLog(in); err != nil {
		t.Fatalf("first save: %v", err)
	}
	existing, err := svc.SaveWorkLog(in)
	if !errors.Is(err, ErrWorkLogAlreadyExists) {
		t.Errorf("err = %v, want ErrWorkLogAlreadyExists", err)
	}
	if existing == nil || existing.Date != "2026-08-02" {
		t.Errorf("should return existing log on conflict")
	}
}

func TestSaveWorkLog_InvalidDate(t *testing.T) {
	svc, _, _ := newServiceForTest()
	_, err := svc.SaveWorkLog(SaveWorkLogInput{Date: "2026-02-30"})
	if err == nil {
		t.Errorf("expected error for invalid date")
	}
}

// ── Tests: UpdateWorkLog ──

func TestUpdateWorkLog_FullReplace(t *testing.T) {
	svc, _, _ := newServiceForTest()
	svc.SaveWorkLog(SaveWorkLogInput{
		Date:  "2026-08-02",
		Items: []SaveItemInput{{Seq: 1, Title: "T1"}},
	})
	_, err := svc.UpdateWorkLog(SaveWorkLogInput{
		Date: "2026-08-02",
		Items: []SaveItemInput{
			{Seq: 1, Title: "T2"},
			{Seq: 2, Title: "T3"},
		},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := svc.GetWorkLog("2026-08-02")
	if len(got.Items) != 2 {
		t.Errorf("items len = %d, want 2 (full replace)", len(got.Items))
	}
}

// ── Tests: StructureBrainDump ──

func TestStructureBrainDump_NilOutput_Fails(t *testing.T) {
	svc, _, llm := newServiceForTest()
	llm.structureOut = nil
	_, err := svc.StructureBrainDump(BrainDumpInput{BrainDump: "x"})
	if !errors.Is(err, ErrAIStructureFailed) {
		t.Errorf("err = %v, want ErrAIStructureFailed", err)
	}
}

func TestStructureBrainDump_AIClientErr_Fails(t *testing.T) {
	svc, _, llm := newServiceForTest()
	llm.structureErr = errors.New("upstream timeout")
	_, err := svc.StructureBrainDump(BrainDumpInput{BrainDump: "x"})
	if !errors.Is(err, ErrAIStructureFailed) {
		t.Errorf("err = %v, want ErrAIStructureFailed", err)
	}
}

func TestStructureBrainDump_FillsPendingForMissingDims(t *testing.T) {
	svc, _, llm := newServiceForTest()
	llm.structureOut = &StructuredWorkLog{
		Items: []StructuredItem{{Content: "c1"}},
	}
	out, err := svc.StructureBrainDump(BrainDumpInput{BrainDump: "x"})
	if err != nil {
		t.Fatalf("structure: %v", err)
	}
	item := out.Items[0]
	if item.Content != "c1" {
		t.Errorf("Content = %q, want 'c1' (preserved)", item.Content)
	}
	if item.ProblemSolved != "（待补充）" {
		t.Errorf("ProblemSolved = %q, want '（待补充）'", item.ProblemSolved)
	}
	if item.Result != "（待补充）" {
		t.Errorf("Result = %q, want '（待补充）'", item.Result)
	}
	if item.Impact != "（待补充）" {
		t.Errorf("Impact = %q, want '（待补充）'", item.Impact)
	}
}

// ── Tests: GenerateReport ──

// countReportObjects counts how many JSON objects (e.g. WorkItem briefs or
// WorkReport maps) were embedded in the captured LLM prompt. Each object has a
// stable top-level key ("title" for items, "period_key" for reports); we count
// occurrences of that key.
func countReportObjects(prompt, key string) int {
	return strings.Count(prompt, `"`+key+`":`)
}

func TestGenerateReport_Weekly_GathersItems(t *testing.T) {
	svc, repo, llm := newServiceForTest()
	repo.logs["2026-08-03"] = &model.WorkLog{
		ID: "wl-1", Date: "2026-08-03",
		Items: []model.WorkItem{{ID: "wi-1", WorkLogID: "wl-1", Title: "T1"}},
	}
	report, err := svc.GenerateReport(GenerateReportInput{
		Type: model.ReportWeekly, PeriodKey: "2026-W32", Force: true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got := countReportObjects(llm.lastPrompt, "title"); got != 1 {
		t.Errorf("weekly should gather 1 item, prompt has %d objects; prompt=%s", got, llm.lastPrompt)
	}
	if !strings.Contains(llm.lastPrompt, "T1") {
		t.Errorf("weekly prompt should contain 'T1'; prompt=%s", llm.lastPrompt)
	}
	if report.PeriodKey != "2026-W32" {
		t.Errorf("period_key = %s", report.PeriodKey)
	}
}

func TestGenerateReport_AlreadyExists_NoForce(t *testing.T) {
	svc, repo, _ := newServiceForTest()
	repo.reports["weekly:2026-W32"] = &model.WorkReport{
		ID: "wr-1", Type: model.ReportWeekly, PeriodKey: "2026-W32",
		StartDate: "2026-08-03", EndDate: "2026-08-09",
	}
	_, err := svc.GenerateReport(GenerateReportInput{
		Type: model.ReportWeekly, PeriodKey: "2026-W32",
	})
	if !errors.Is(err, ErrReportAlreadyExists) {
		t.Errorf("err = %v, want ErrReportAlreadyExists", err)
	}
}

// INVARIANT: monthly 只读 weekly 报告 + 月内孤儿 items，绝不读所有原始 items
func TestGenerateReport_Monthly_ReadsWeekliesAndOrphans(t *testing.T) {
	svc, repo, llm := newServiceForTest()
	// 周报：8/3~8/9（覆盖月初前段）
	repo.reports["weekly:2026-W32"] = &model.WorkReport{
		ID: "wr-1", Type: model.ReportWeekly, PeriodKey: "2026-W32",
		StartDate: "2026-08-03", EndDate: "2026-08-09",
	}
	// 孤儿：8/12 不在周报覆盖内（8/10~8/16 才是 W33）
	// 实际 W33 = 8/10~8/16，所以 8/12 在 W33 内；改用 8/2（属 W31）作孤儿
	repo.logs["2026-08-02"] = &model.WorkLog{
		ID: "wl-1", Date: "2026-08-02",
		Items: []model.WorkItem{{ID: "wi-1", WorkLogID: "wl-1", Title: "Orphan"}},
	}
	// 周报覆盖内的 log（8/4 属 W32），它的 items 不应被作为孤儿传入
	repo.logs["2026-08-04"] = &model.WorkLog{
		ID: "wl-2", Date: "2026-08-04",
		Items: []model.WorkItem{{ID: "wi-2", WorkLogID: "wl-2", Title: "Covered"}},
	}
	_, err := svc.GenerateReport(GenerateReportInput{
		Type: model.ReportMonthly, PeriodKey: "2026-08", Force: true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// 月报 prompt 同时含 weekly JSON 数组和 orphan items JSON。Weekly JSON 用
	// "period_key" 标识；orphan items 用 "title" 标识。
	if got := countReportObjects(llm.lastPrompt, "period_key"); got != 1 {
		t.Errorf("monthly should embed 1 weekly report, got %d; prompt=%s", got, llm.lastPrompt)
	}
	// "Covered" 必须不出现，"Orphan" 必须出现
	if strings.Contains(llm.lastPrompt, "Covered") {
		t.Errorf("monthly must not include items covered by weeklies; prompt=%s", llm.lastPrompt)
	}
	if !strings.Contains(llm.lastPrompt, "Orphan") {
		t.Errorf("monthly prompt should include orphan item 'Orphan'; prompt=%s", llm.lastPrompt)
	}
}

// INVARIANT: halfyear 只读 monthly 报告，绝不读原始 items
func TestGenerateReport_HalfYear_ReadsOnlyMonthlies(t *testing.T) {
	svc, repo, llm := newServiceForTest()
	repo.reports["monthly:2026-07"] = &model.WorkReport{
		ID: "mr-1", Type: model.ReportMonthly, PeriodKey: "2026-07",
		StartDate: "2026-07-01", EndDate: "2026-07-31",
	}
	// 原始 items 存在但不应被读取
	repo.logs["2026-07-15"] = &model.WorkLog{
		ID: "wl-1", Date: "2026-07-15",
		Items: []model.WorkItem{{ID: "wi-1", WorkLogID: "wl-1", Title: "Should be ignored"}},
	}
	_, err := svc.GenerateReport(GenerateReportInput{
		Type: model.ReportHalfYear, PeriodKey: "2026-H2", Force: true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got := countReportObjects(llm.lastPrompt, "period_key"); got != 1 {
		t.Errorf("halfyear should embed 1 monthly, got %d; prompt=%s", got, llm.lastPrompt)
	}
	if strings.Contains(llm.lastPrompt, "Should be ignored") {
		t.Errorf("halfyear must not touch raw items; prompt=%s", llm.lastPrompt)
	}
}

// INVARIANT: yearly 只读 monthly 报告，绝不读原始 items 或 weekly 报告
func TestGenerateReport_Yearly_ReadsOnlyMonthlies(t *testing.T) {
	svc, repo, llm := newServiceForTest()
	repo.reports["monthly:2026-01"] = &model.WorkReport{
		ID: "mr-1", Type: model.ReportMonthly, PeriodKey: "2026-01",
		StartDate: "2026-01-01", EndDate: "2026-01-31",
	}
	repo.reports["weekly:2026-W01"] = &model.WorkReport{
		ID: "wr-1", Type: model.ReportWeekly, PeriodKey: "2026-W01",
		StartDate: "2025-12-29", EndDate: "2026-01-04",
	}
	_, err := svc.GenerateReport(GenerateReportInput{
		Type: model.ReportYearly, PeriodKey: "2026", Force: true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if got := countReportObjects(llm.lastPrompt, "period_key"); got != 1 {
		t.Errorf("yearly should embed only 1 monthly (must exclude weeklies), got %d; prompt=%s", got, llm.lastPrompt)
	}
	if strings.Contains(llm.lastPrompt, "2026-W01") {
		t.Errorf("yearly must not embed weekly reports; prompt=%s", llm.lastPrompt)
	}
}

func TestGenerateReport_Force_OverwritesExisting(t *testing.T) {
	svc, repo, _ := newServiceForTest()
	repo.reports["weekly:2026-W32"] = &model.WorkReport{
		ID: "wr-old", Type: model.ReportWeekly, PeriodKey: "2026-W32",
		StartDate: "2026-08-03", EndDate: "2026-08-09",
	}
	_, err := svc.GenerateReport(GenerateReportInput{
		Type: model.ReportWeekly, PeriodKey: "2026-W32", Force: true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	updated := repo.reports["weekly:2026-W32"]
	if updated.ID != "wr-old" {
		t.Errorf("should preserve existing ID, got %s", updated.ID)
	}
}

func TestGenerateReport_BadPeriodKey(t *testing.T) {
	svc, _, _ := newServiceForTest()
	_, err := svc.GenerateReport(GenerateReportInput{
		Type: model.ReportWeekly, PeriodKey: "garbage", Force: true,
	})
	if err == nil {
		t.Errorf("expected error for bad period key")
	}
}

// ── Tests: GetTodayContext ──

func TestGetTodayContext_AggregatesTasksAndSessions(t *testing.T) {
	day := "2026-08-02"
	t0, _ := time.Parse("2006-01-02", day)
	taskDoneAt := t0.Add(2 * time.Hour)
	taskID := "task-1"
	endTime := t0.Add(1*time.Hour + 25*time.Minute)
	svc := newServiceWithTaskSessionForTest(
		&mockTaskRepoForWorkLog{tasks: []*model.Task{
			{ID: taskID, Title: "写周报", Status: model.StatusCompleted, CompletedAt: &taskDoneAt},
		}},
		&mockSessionRepoForWorkLog{sessions: []model.PomodoroSession{
			{
				ID:        "sess-1",
				TaskID:    &taskID,
				Type:      model.SessionWork,
				Status:    model.SessionCompleted,
				StartTime: t0.Add(1 * time.Hour),
				EndTime:   &endTime,
			},
		}},
	)

	ctx, err := svc.GetTodayContext(day)
	if err != nil {
		t.Fatalf("GetTodayContext: %v", err)
	}
	if ctx.Date != day {
		t.Errorf("Date = %q, want %q", ctx.Date, day)
	}
	if len(ctx.CompletedTasks) != 1 || ctx.CompletedTasks[0].Title != "写周报" {
		t.Errorf("CompletedTasks = %+v", ctx.CompletedTasks)
	}
	if len(ctx.PomodoroSessions) != 1 {
		t.Fatalf("PomodoroSessions len = %d, want 1", len(ctx.PomodoroSessions))
	}
	if ctx.PomodoroSummary.Count != 1 {
		t.Errorf("summary count = %d, want 1", ctx.PomodoroSummary.Count)
	}
	if ctx.PomodoroSummary.TotalMinutes != 25 {
		t.Errorf("TotalMinutes = %d, want 25", ctx.PomodoroSummary.TotalMinutes)
	}
	if ctx.PomodoroSessions[0].TaskTitle != "写周报" {
		t.Errorf("TaskTitle = %q, want '写周报'", ctx.PomodoroSessions[0].TaskTitle)
	}
}

func TestGetTodayContext_InvalidDate(t *testing.T) {
	svc := newServiceWithTaskSessionForTest(
		&mockTaskRepoForWorkLog{},
		&mockSessionRepoForWorkLog{},
	)
	if _, err := svc.GetTodayContext("2026-02-30"); err == nil {
		t.Errorf("expected error for invalid date")
	}
}
