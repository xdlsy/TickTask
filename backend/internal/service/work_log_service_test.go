package service

import (
	"errors"
	"testing"
	"time"

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

// ── Mock AI client ──

type mockAIClient struct {
	structuredOut *StructuredWorkLog
	structuredErr error
}

func (m *mockAIClient) StructureBrainDump(input BrainDumpInput) (*StructuredWorkLog, error) {
	if m.structuredErr != nil {
		return nil, m.structuredErr
	}
	return m.structuredOut, nil
}
func (m *mockAIClient) GenerateWeeklyReport(items []model.WorkItem, start, end string) (*ReportSummary, error) {
	return nil, errors.New("not impl")
}
func (m *mockAIClient) GenerateMonthlyReport(w []*model.WorkReport, o []model.WorkItem, start, end string) (*ReportSummary, error) {
	return nil, errors.New("not impl")
}
func (m *mockAIClient) GenerateHalfYearReport(mo []*model.WorkReport, start, end string) (*ReportSummary, error) {
	return nil, errors.New("not impl")
}
func (m *mockAIClient) GenerateYearlyReport(mo []*model.WorkReport, start, end string) (*ReportSummary, error) {
	return nil, errors.New("not impl")
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

func newServiceForTest() (*WorkLogService, *mockWorkLogRepo, *mockAIClient) {
	repo := newMockWorkLogRepo()
	ai := &mockAIClient{}
	svc := &WorkLogService{
		repo:        repo,
		aiClient:    ai,
		idGenerator: func() string { return "test-id" },
	}
	return svc, repo, ai
}

func newServiceWithTaskSessionForTest(taskRepo repository.TaskRepository, sessionRepo repository.SessionRepository) *WorkLogService {
	return &WorkLogService{
		repo:        newMockWorkLogRepo(),
		taskRepo:    taskRepo,
		sessionRepo: sessionRepo,
		aiClient:    &mockAIClient{},
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

func TestStructureBrainDump_NoTitle_Fails(t *testing.T) {
	svc, _, ai := newServiceForTest()
	ai.structuredOut = &StructuredWorkLog{
		Items: []StructuredItem{{Content: "no title"}},
	}
	_, err := svc.StructureBrainDump(BrainDumpInput{BrainDump: "x"})
	if !errors.Is(err, ErrAIStructureFailed) {
		t.Errorf("err = %v, want ErrAIStructureFailed", err)
	}
}

func TestStructureBrainDump_NilOutput_Fails(t *testing.T) {
	svc, _, ai := newServiceForTest()
	ai.structuredOut = nil
	_, err := svc.StructureBrainDump(BrainDumpInput{BrainDump: "x"})
	if !errors.Is(err, ErrAIStructureFailed) {
		t.Errorf("err = %v, want ErrAIStructureFailed", err)
	}
}

func TestStructureBrainDump_AIClientErr_Fails(t *testing.T) {
	svc, _, ai := newServiceForTest()
	ai.structuredErr = errors.New("upstream timeout")
	_, err := svc.StructureBrainDump(BrainDumpInput{BrainDump: "x"})
	if !errors.Is(err, ErrAIStructureFailed) {
		t.Errorf("err = %v, want ErrAIStructureFailed", err)
	}
}

func TestStructureBrainDump_FillsPendingForMissingDims(t *testing.T) {
	svc, _, ai := newServiceForTest()
	ai.structuredOut = &StructuredWorkLog{
		Items: []StructuredItem{{Title: "T1", Content: "c1"}},
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

// ── Tests: GenerateReport stub ──

func TestGenerateReport_StubReturnsError(t *testing.T) {
	svc, _, _ := newServiceForTest()
	_, err := svc.GenerateReport(GenerateReportInput{Type: model.ReportWeekly, PeriodKey: "2026-W31"})
	if err == nil {
		t.Errorf("expected stub error")
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
