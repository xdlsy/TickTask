package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

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
		idGenerator: func() string { return fmt.Sprintf("id-%d", time.Now().UnixNano()) },
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

// GenerateReport 生成报告（M4 实现，M1 stub）
func (s *WorkLogService) GenerateReport(input GenerateReportInput) (*model.WorkReport, error) {
	return nil, errors.New("GenerateReport not implemented yet (M4)")
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

// 确保 ReportSummary / TodayContext 能 JSON 序列化（避免 unused 警告）
var _ = json.Marshal
