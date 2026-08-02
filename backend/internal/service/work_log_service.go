package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"ticktask/internal/model"
	"ticktask/internal/repository"
)

// ── DTO ──

// TodayContext 今日预填上下文（完成任务、番茄钟会话）
type TodayContext struct {
	Date             string         `json:"date"`
	CompletedTasks   []TaskBrief    `json:"completed_tasks"`
	PomodoroSessions []SessionBrief `json:"pomodoro_sessions"`
	PomodoroSummary  SessionSummary `json:"pomodoro_summary"`
}

// TaskBrief 任务摘要
type TaskBrief struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// SessionBrief 番茄钟会话摘要
type SessionBrief struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	TaskTitle string `json:"task_title"`
	StartedAt string `json:"started_at"`
	Minutes   int    `json:"minutes"`
}

// SessionSummary 番茄钟汇总
type SessionSummary struct {
	Count        int `json:"count"`
	TotalMinutes int `json:"total_minutes"`
}

// BrainDumpInput AI 拆条输入
type BrainDumpInput struct {
	BrainDump string       `json:"brain_dump"`
	Context   TodayContext `json:"context"`
}

// StructuredItem AI 拆条输出的单条工作（四维）
type StructuredItem struct {
	Title         string `json:"title"`
	Content       string `json:"content"`
	ProblemSolved string `json:"problem_solved"`
	Result        string `json:"result"`
	Impact        string `json:"impact"`
}

// StructuredWorkLog AI 拆条输出（预览，未落库）
type StructuredWorkLog struct {
	Items   []StructuredItem `json:"items"`
	Summary string           `json:"summary"`
}

// SaveWorkLogInput 保存日报输入
type SaveWorkLogInput struct {
	Date         string          `json:"date"`
	Summary      string          `json:"summary"`
	RawBrainDump string          `json:"raw_brain_dump"`
	Items        []SaveItemInput `json:"items"`
}

// SaveItemInput 保存时的单条 item
type SaveItemInput struct {
	Seq           int    `json:"seq"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	ProblemSolved string `json:"problem_solved"`
	Result        string `json:"result"`
	Impact        string `json:"impact"`
}

// GenerateReportInput 生成报告输入
type GenerateReportInput struct {
	Type      model.WorkReportType `json:"type"`
	PeriodKey string                `json:"period_key"`
	Force     bool                  `json:"force"`
}

// CreateQuickEntryInput 快捷录入新增输入
type CreateQuickEntryInput struct {
	Activity  string `json:"activity"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Quadrant  int    `json:"quadrant"`
}

// UpdateQuickEntryInput 快捷录入编辑输入（指针 = 部分更新）
type UpdateQuickEntryInput struct {
	Activity  *string `json:"activity,omitempty"`
	StartTime *string `json:"start_time,omitempty"`
	EndTime   *string `json:"end_time,omitempty"`
	Quadrant  *int    `json:"quadrant,omitempty"`
}

// ReportSummary 报告汇总结构（4 字段，所有报告 type 共用）
type ReportSummary struct {
	CoreWork     string `json:"core_work"`
	MainProgress string `json:"main_progress"`
	OpenIssues   string `json:"open_issues"`
	NextFocus    string `json:"next_focus"`
}

// ── 错误 ──

var (
	ErrWorkLogAlreadyExists = errors.New("work log already exists for this date")
	ErrReportAlreadyExists  = errors.New("report already exists, set force=true to overwrite")
	ErrAIStructureFailed    = errors.New("AI structuring failed")
)

// ── AI client interface（M1 stub，M2 接真实实现）──

// WorkLogAIClient AI 拆条/汇总客户端接口
type WorkLogAIClient interface {
	StructureBrainDump(input BrainDumpInput) (*StructuredWorkLog, error)
	GenerateWeeklyReport(items []model.WorkItem, start, end string) (*ReportSummary, error)
	GenerateMonthlyReport(weeklies []*model.WorkReport, orphanItems []model.WorkItem, start, end string) (*ReportSummary, error)
	GenerateHalfYearReport(monthlies []*model.WorkReport, start, end string) (*ReportSummary, error)
	GenerateYearlyReport(monthlies []*model.WorkReport, start, end string) (*ReportSummary, error)
}

// ── Service ──

// WorkLogService 工作日志业务编排
type WorkLogService struct {
	repo        repository.WorkLogRepository
	taskRepo    repository.TaskRepository
	sessionRepo repository.SessionRepository
	aiClient    WorkLogAIClient
	idGenerator func() string
}

// NewWorkLogService 构造
func NewWorkLogService(
	repo repository.WorkLogRepository,
	taskRepo repository.TaskRepository,
	sessionRepo repository.SessionRepository,
	aiClient WorkLogAIClient,
) *WorkLogService {
	return &WorkLogService{
		repo:        repo,
		taskRepo:    taskRepo,
		sessionRepo: sessionRepo,
		aiClient:    aiClient,
		idGenerator: func() string { return uuid.New().String() },
	}
}

// GetTodayContext 拉今日预填上下文
func (s *WorkLogService) GetTodayContext(date string) (*TodayContext, error) {
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}
	start := t
	end := t.Add(24 * time.Hour)

	tasks, err := s.taskRepo.GetCompletedTasksInRange(start, end)
	if err != nil {
		return nil, err
	}
	completed := make([]TaskBrief, 0, len(tasks))
	taskTitleByID := make(map[string]string, len(tasks))
	for _, t := range tasks {
		completed = append(completed, TaskBrief{ID: t.ID, Title: t.Title})
		taskTitleByID[t.ID] = t.Title
	}

	sessions, err := s.sessionRepo.GetCompletedWorkByDateRange(start, end)
	if err != nil {
		return nil, err
	}
	sessionBriefs := make([]SessionBrief, 0, len(sessions))
	totalMin := 0
	for _, sess := range sessions {
		minutes := 0
		// 优先用 ActualDuration（秒），否则用 EndTime - StartTime；EndTime 可能为 nil
		if sess.ActualDuration != nil {
			minutes = *sess.ActualDuration / 60
		} else if sess.EndTime != nil {
			minutes = int(sess.EndTime.Sub(sess.StartTime).Minutes())
		}
		totalMin += minutes

		taskID := ""
		taskTitle := ""
		if sess.TaskID != nil {
			taskID = *sess.TaskID
			taskTitle = taskTitleByID[taskID]
		}
		sessionBriefs = append(sessionBriefs, SessionBrief{
			ID:        sess.ID,
			TaskID:    taskID,
			TaskTitle: taskTitle,
			StartedAt: sess.StartTime.Format(time.RFC3339),
			Minutes:   minutes,
		})
	}

	return &TodayContext{
		Date:             date,
		CompletedTasks:   completed,
		PomodoroSessions: sessionBriefs,
		PomodoroSummary:  SessionSummary{Count: len(sessions), TotalMinutes: totalMin},
	}, nil
}

// StructureBrainDump AI 拆条（不落库）
func (s *WorkLogService) StructureBrainDump(input BrainDumpInput) (*StructuredWorkLog, error) {
	if s.aiClient == nil {
		return nil, ErrAIStructureFailed
	}
	out, err := s.aiClient.StructureBrainDump(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIStructureFailed, err)
	}
	if out == nil {
		return nil, fmt.Errorf("%w: nil output", ErrAIStructureFailed)
	}
	for i := range out.Items {
		if out.Items[i].Title == "" {
			return nil, fmt.Errorf("%w: item[%d] missing title", ErrAIStructureFailed, i)
		}
		if out.Items[i].Content == "" {
			out.Items[i].Content = "（待补充）"
		}
		if out.Items[i].ProblemSolved == "" {
			out.Items[i].ProblemSolved = "（待补充）"
		}
		if out.Items[i].Result == "" {
			out.Items[i].Result = "（待补充）"
		}
		if out.Items[i].Impact == "" {
			out.Items[i].Impact = "（待补充）"
		}
	}
	return out, nil
}

// SaveWorkLog 保存日报（POST 语义：同日已存在 → ErrWorkLogAlreadyExists）
func (s *WorkLogService) SaveWorkLog(input SaveWorkLogInput) (*model.WorkLog, error) {
	if _, err := time.Parse("2006-01-02", input.Date); err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}
	existing, err := s.repo.GetWorkLogByDate(input.Date)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if existing != nil {
		return existing, ErrWorkLogAlreadyExists
	}
	log := s.buildWorkLogFromInput(input)
	if err := s.repo.CreateWorkLog(log); err != nil {
		return nil, err
	}
	return log, nil
}

// UpdateWorkLog 更新日报（PUT 语义：items 全量替换 via UpsertWorkLog）
func (s *WorkLogService) UpdateWorkLog(input SaveWorkLogInput) (*model.WorkLog, error) {
	if _, err := time.Parse("2006-01-02", input.Date); err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}
	log := s.buildWorkLogFromInput(input)
	if err := s.repo.UpsertWorkLog(log); err != nil {
		return nil, err
	}
	updated, err := s.repo.GetWorkLogByDate(input.Date)
	if err != nil {
		return nil, fmt.Errorf("re-read after upsert: %w", err)
	}
	return updated, nil
}

// GetWorkLog 按日期读
func (s *WorkLogService) GetWorkLog(date string) (*model.WorkLog, error) {
	return s.repo.GetWorkLogByDate(date)
}

// ListWorkLogs 范围查
func (s *WorkLogService) ListWorkLogs(from, to string) ([]*model.WorkLog, error) {
	return s.repo.GetWorkLogsInRange(from, to)
}

// GenerateReport 生成周期报告
func (s *WorkLogService) GenerateReport(input GenerateReportInput) (*model.WorkReport, error) {
	if s.aiClient == nil {
		return nil, ErrAIStructureFailed
	}
	moment := time.Now()
	if input.PeriodKey != "" {
		m, err := parsePeriodKeyMoment(input.Type, input.PeriodKey)
		if err != nil {
			return nil, err
		}
		moment = m
	}
	start, end := RangeForType(input.Type, moment)
	startStr, endStr := DateRangeToYMD(start, end)
	periodKey := KeyForType(input.Type, moment)

	existing, err := s.repo.GetWorkReportByTypeAndPeriod(input.Type, periodKey)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if existing != nil && !input.Force {
		return existing, ErrReportAlreadyExists
	}

	var summary *ReportSummary
	switch input.Type {
	case model.ReportWeekly:
		summary, err = s.generateWeekly(startStr, endStr)
	case model.ReportMonthly:
		summary, err = s.generateMonthly(start, end, startStr, endStr)
	case model.ReportHalfYear:
		summary, err = s.generateHalfYear(startStr, endStr)
	case model.ReportYearly:
		summary, err = s.generateYearly(startStr, endStr)
	default:
		return nil, fmt.Errorf("unknown report type: %s", input.Type)
	}
	if err != nil {
		return nil, err
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return nil, fmt.Errorf("marshal summary: %w", err)
	}

	logs, err := s.repo.GetWorkLogsInRange(startStr, endStr)
	if err != nil {
		return nil, err
	}
	existDates := make([]string, 0, len(logs))
	for _, l := range logs {
		existDates = append(existDates, l.Date)
	}
	missing := MissingDays(start, end, existDates)

	report := &model.WorkReport{
		ID:          s.idGenerator(),
		Type:        input.Type,
		PeriodKey:   periodKey,
		StartDate:   startStr,
		EndDate:     endStr,
		SummaryJSON: string(summaryJSON),
		MissingDays: missing,
	}

	if existing != nil {
		report.ID = existing.ID
		report.CreatedAt = existing.CreatedAt
		if err := s.repo.UpdateWorkReport(report); err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.CreateWorkReport(report); err != nil {
			return nil, err
		}
	}
	return report, nil
}

// parsePeriodKeyMoment 从 period key 反推一个代表日期（取周期开始日）
func parsePeriodKeyMoment(t model.WorkReportType, key string) (time.Time, error) {
	switch t {
	case model.ReportWeekly:
		var year, week int
		if _, err := fmt.Sscanf(key, "%d-W%d", &year, &week); err != nil {
			return time.Time{}, fmt.Errorf("bad weekly key: %s", key)
		}
		cursor := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
		for cursor.Year() <= year+1 {
			cy, cw := cursor.ISOWeek()
			if cy == year && cw == week {
				return cursor, nil
			}
			cursor = cursor.AddDate(0, 0, 1)
		}
		return time.Time{}, fmt.Errorf("weekly key not found: %s", key)
	case model.ReportMonthly:
		t2, err := time.Parse("2006-01", key)
		if err != nil {
			return time.Time{}, fmt.Errorf("bad monthly key: %s", key)
		}
		return t2, nil
	case model.ReportHalfYear:
		var year int
		var h string
		if _, err := fmt.Sscanf(key, "%d-%s", &year, &h); err != nil {
			return time.Time{}, fmt.Errorf("bad halfyear key: %s", key)
		}
		month := 1
		if h == "H2" {
			month = 7
		}
		return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local), nil
	case model.ReportYearly:
		var year int
		if _, err := fmt.Sscanf(key, "%d", &year); err != nil {
			return time.Time{}, fmt.Errorf("bad yearly key: %s", key)
		}
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.Local), nil
	}
	return time.Time{}, fmt.Errorf("unknown type: %s", t)
}

// INVARIANT: weekly 只读 work_items，不读 reports
func (s *WorkLogService) generateWeekly(startStr, endStr string) (*ReportSummary, error) {
	logs, err := s.repo.GetWorkLogsInRange(startStr, endStr)
	if err != nil {
		return nil, err
	}
	var items []model.WorkItem
	for _, l := range logs {
		items = append(items, l.Items...)
	}
	return s.aiClient.GenerateWeeklyReport(items, startStr, endStr)
}

// INVARIANT: monthly 只读 weekly 报告 + 月内不属于完整周的零散 items，不读所有原始 items
func (s *WorkLogService) generateMonthly(start, end time.Time, startStr, endStr string) (*ReportSummary, error) {
	weeklies, err := s.repo.ListWorkReports(model.ReportWeekly)
	if err != nil {
		return nil, err
	}
	var in []*model.WorkReport
	for _, w := range weeklies {
		if w.EndDate >= startStr && w.StartDate <= endStr {
			in = append(in, w)
		}
	}
	covered := make(map[string]bool)
	for _, w := range in {
		ws, _ := time.Parse("2006-01-02", w.StartDate)
		we, _ := time.Parse("2006-01-02", w.EndDate)
		for d := ws; !d.After(we); d = d.AddDate(0, 0, 1) {
			covered[d.Format("2006-01-02")] = true
		}
	}
	logs, err := s.repo.GetWorkLogsInRange(startStr, endStr)
	if err != nil {
		return nil, err
	}
	var orphans []model.WorkItem
	for _, l := range logs {
		if !covered[l.Date] {
			orphans = append(orphans, l.Items...)
		}
	}
	return s.aiClient.GenerateMonthlyReport(in, orphans, startStr, endStr)
}

// INVARIANT: halfyear 只读 monthly 报告，绝不读原始 items
func (s *WorkLogService) generateHalfYear(startStr, endStr string) (*ReportSummary, error) {
	monthlies, err := s.repo.ListWorkReports(model.ReportMonthly)
	if err != nil {
		return nil, err
	}
	var in []*model.WorkReport
	for _, m := range monthlies {
		if m.EndDate >= startStr && m.StartDate <= endStr {
			in = append(in, m)
		}
	}
	return s.aiClient.GenerateHalfYearReport(in, startStr, endStr)
}

// INVARIANT: yearly 只读 monthly 报告，绝不读原始 items 或 weekly 报告
func (s *WorkLogService) generateYearly(startStr, endStr string) (*ReportSummary, error) {
	monthlies, err := s.repo.ListWorkReports(model.ReportMonthly)
	if err != nil {
		return nil, err
	}
	var in []*model.WorkReport
	for _, m := range monthlies {
		if m.StartDate >= startStr && m.EndDate <= endStr {
			in = append(in, m)
		}
	}
	return s.aiClient.GenerateYearlyReport(in, startStr, endStr)
}

// GetReport 读报告
func (s *WorkLogService) GetReport(t model.WorkReportType, periodKey string) (*model.WorkReport, error) {
	return s.repo.GetWorkReportByTypeAndPeriod(t, periodKey)
}

// ListReports 列表
func (s *WorkLogService) ListReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	return s.repo.ListWorkReports(t)
}

// ── 内部 ──

func (s *WorkLogService) buildWorkLogFromInput(input SaveWorkLogInput) *model.WorkLog {
	logID := s.idGenerator()
	items := make([]model.WorkItem, 0, len(input.Items))
	for _, it := range input.Items {
		items = append(items, model.WorkItem{
			ID:            s.idGenerator(),
			WorkLogID:     logID,
			Seq:           it.Seq,
			Title:         it.Title,
			Content:       it.Content,
			ProblemSolved: it.ProblemSolved,
			Result:        it.Result,
			Impact:        it.Impact,
			Source:        "ai",
		})
	}
	return &model.WorkLog{
		ID:           logID,
		Date:         input.Date,
		Summary:      input.Summary,
		RawBrainDump: input.RawBrainDump,
		Items:        items,
	}
}

// AddQuickEntry 快捷录入：自动建 WorkLog（如不存在）+ 追加 manual item
func (s *WorkLogService) AddQuickEntry(date string, in CreateQuickEntryInput) (*model.WorkItem, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}
	if _, err := time.Parse("15:04", in.StartTime); err != nil {
		return nil, fmt.Errorf("invalid time format: %w", err)
	}
	if _, err := time.Parse("15:04", in.EndTime); err != nil {
		return nil, fmt.Errorf("invalid time format: %w", err)
	}
	if in.StartTime >= in.EndTime {
		return nil, errors.New("end_time must be after start_time")
	}

	log, err := s.repo.GetWorkLogByDate(date)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if log == nil {
		log = &model.WorkLog{
			ID:           s.idGenerator(),
			Date:         date,
			Summary:      "",
			RawBrainDump: "",
		}
		if err := s.repo.CreateWorkLog(log); err != nil {
			return nil, err
		}
	}

	maxSeq := 0
	for _, it := range log.Items {
		if it.Seq > maxSeq {
			maxSeq = it.Seq
		}
	}

	item := model.WorkItem{
		ID:        s.idGenerator(),
		WorkLogID: log.ID,
		Seq:       maxSeq + 1,
		Activity:  &in.Activity,
		StartTime: &in.StartTime,
		EndTime:   &in.EndTime,
		Quadrant:  &in.Quadrant,
		Source:    "manual",
	}
	if err := s.repo.AppendItem(log.ID, item); err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateQuickEntry 编辑快捷录入条目
func (s *WorkLogService) UpdateQuickEntry(date string, itemID string, in UpdateQuickEntryInput) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}
	if in.StartTime != nil {
		if _, err := time.Parse("15:04", *in.StartTime); err != nil {
			return fmt.Errorf("invalid time format: %w", err)
		}
	}
	if in.EndTime != nil {
		if _, err := time.Parse("15:04", *in.EndTime); err != nil {
			return fmt.Errorf("invalid time format: %w", err)
		}
	}
	// 注意：仅当本次同时提供 start_time 和 end_time 时才校验顺序。
	// 若用户只改一项，无法在服务层判断最终顺序（另一项依赖库里现值）。
	// handler 层可在读取现值后做更严格的校验。
	if in.StartTime != nil && in.EndTime != nil && *in.StartTime >= *in.EndTime {
		return errors.New("end_time must be after start_time")
	}

	log, err := s.repo.GetWorkLogByDate(date)
	if err != nil {
		// WorkLog 不存在时，item 必然也不存在：归一化为 ErrItemNotFound，便于调用方分支处理
		if errors.Is(err, repository.ErrNotFound) {
			return repository.ErrItemNotFound
		}
		return err
	}

	updates := map[string]any{}
	if in.Activity != nil {
		updates["activity"] = *in.Activity
	}
	if in.StartTime != nil {
		updates["start_time"] = *in.StartTime
	}
	if in.EndTime != nil {
		updates["end_time"] = *in.EndTime
	}
	if in.Quadrant != nil {
		updates["quadrant"] = *in.Quadrant
	}
	if len(updates) == 0 {
		return nil
	}
	return s.repo.UpdateItem(log.ID, itemID, updates)
}

// DeleteQuickEntry 删除快捷录入条目
func (s *WorkLogService) DeleteQuickEntry(date string, itemID string) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}
	log, err := s.repo.GetWorkLogByDate(date)
	if err != nil {
		// WorkLog 不存在时，item 必然也不存在：归一化为 ErrItemNotFound，便于调用方分支处理
		if errors.Is(err, repository.ErrNotFound) {
			return repository.ErrItemNotFound
		}
		return err
	}
	return s.repo.DeleteItem(log.ID, itemID)
}

// 确保 ReportSummary / TodayContext 能 JSON 序列化（避免 unused 警告）
var _ = json.Marshal
