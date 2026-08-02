package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"ticktask/internal/ai"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"time"
)

// AIService AI 服务
type AIService struct {
	client       ai.LLMClient
	taskRepo     repository.TaskRepository
	scheduleRepo repository.ScheduleRepository
	sessionRepo  repository.SessionRepository
}

// NewAIService 创建 AI 服务
func NewAIService(cfg *model.AISettings, taskRepo repository.TaskRepository, scheduleRepo repository.ScheduleRepository, sessionRepo repository.SessionRepository) *AIService {
	var client ai.LLMClient
	if cfg != nil {
		switch cfg.Provider {
		case "claude":
			client = ai.NewCLIClient()
		case "anthropic":
			if cfg.APIKey == "" { break }
			client = ai.NewAnthropicClient(cfg.APIKey, cfg.Model)
		default:
			if cfg.APIKey == "" { break }
			client = ai.NewOpenAIClient(cfg.APIKey, cfg.BaseURL, cfg.Model)
		}
	}
	return &AIService{
		client:       client,
		taskRepo:     taskRepo,
		scheduleRepo: scheduleRepo,
		sessionRepo:  sessionRepo,
	}
}

// ClassificationResult 任务分类结果
type ClassificationResult struct {
	TaskID    string `json:"task_id"`
	Important bool   `json:"important"`
	Urgent    bool   `json:"urgent"`
	Quadrant  int    `json:"quadrant"`
	Reason    string `json:"reason"`
}

// ClassifyTask 智能分类单个任务
func (s *AIService) ClassifyTask(ctx context.Context, task *model.Task) (*ClassificationResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("AI service not configured")
	}

	deadline := "无"
	if task.Deadline != nil {
		deadline = task.Deadline.Format("2006-01-02 15:04")
	}

	prompt := fmt.Sprintf(ai.ClassifyPrompt, task.Title, task.Description, deadline)

	response, err := s.client.ChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	// 解析 JSON 响应
	result, err := parseClassifyResponse(response)
	if err != nil {
		return nil, err
	}

	result.TaskID = task.ID
	result.Quadrant = calculateQuadrant(result.Important, result.Urgent)

	return result, nil
}

// ClassifyTasks 批量分类任务
func (s *AIService) ClassifyTasks(ctx context.Context, taskIDs []string) ([]*ClassificationResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("AI service not configured")
	}

	results := make([]*ClassificationResult, 0, len(taskIDs))

	for _, id := range taskIDs {
		task, err := s.taskRepo.GetByID(id)
		if err != nil {
			continue
		}

		result, err := s.ClassifyTask(ctx, task)
		if err != nil {
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// ScheduleItem 日程项
type ScheduleItem struct {
	TaskID        string `json:"task_id"`
	Title         string `json:"title"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	PomodoroCount int    `json:"pomodoro_count"`
}

// DailySchedule 每日日程
type DailySchedule struct {
	Schedule []ScheduleItem `json:"schedule"`
}

// GenerateDailySchedule 生成今日日程
func (s *AIService) GenerateDailySchedule(ctx context.Context, startTime, endTime string, pomodoroSettings *model.PomodoroSettings) (*DailySchedule, error) {
	if s.client == nil {
		return nil, fmt.Errorf("AI service not configured")
	}

	// 获取待办任务
	tasks, err := s.taskRepo.GetByStatus(model.StatusTodo)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(tasks) == 0 {
		return &DailySchedule{Schedule: []ScheduleItem{}}, nil
	}

	// 构建任务列表字符串
	var taskList strings.Builder
	for _, t := range tasks {
		deadline := "无截止时间"
		if t.Deadline != nil {
			deadline = t.Deadline.Format("2006-01-02")
		}
		taskList.WriteString(fmt.Sprintf("- ID: %s, 标题: %s, 预估时间: %d分钟, 截止: %s\n",
			t.ID, t.Title, t.EstimatedTime, deadline))
	}

	prompt := fmt.Sprintf(ai.SchedulePrompt,
		startTime, endTime,
		pomodoroSettings.WorkDuration/60,
		pomodoroSettings.ShortBreakDuration/60,
		pomodoroSettings.LongBreakAfter,
		taskList.String(),
	)

	response, err := s.client.ChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	// 解析响应
	schedule, err := parseScheduleResponse(response)
	if err != nil {
		return nil, err
	}

	return schedule, nil
}

// PrioritySuggestion 优先级建议
type PrioritySuggestion struct {
	PriorityOrder []string `json:"priority_order"`
}

// GetPrioritySuggestions 获取优先级建议
func (s *AIService) GetPrioritySuggestions(ctx context.Context) (*PrioritySuggestion, error) {
	if s.client == nil {
		return nil, fmt.Errorf("AI service not configured")
	}

	// 获取待办任务
	tasks, err := s.taskRepo.GetByStatus(model.StatusTodo)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	if len(tasks) == 0 {
		return &PrioritySuggestion{PriorityOrder: []string{}}, nil
	}

	// 构建任务列表
	var taskList strings.Builder
	for _, t := range tasks {
		deadline := "无"
		if t.Deadline != nil {
			deadline = t.Deadline.Format("2006-01-02")
		}
		taskList.WriteString(fmt.Sprintf("- ID: %s, 标题: %s, 象限: %d, 截止: %s\n",
			t.ID, t.Title, t.Quadrant, deadline))
	}

	prompt := fmt.Sprintf(ai.PriorityPrompt, taskList.String())

	response, err := s.client.ChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	// 解析响应
	result, err := parsePriorityResponse(response)
	if err != nil {
		return nil, err
	}

	return result, nil
}

// ClassifyTaskByText 根据文本分类任务（无需任务 ID）
func (s *AIService) ClassifyTaskByText(ctx context.Context, title, description string) (*ClassificationResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("AI service not configured")
	}

	prompt := fmt.Sprintf(ai.ClassifyByTextPrompt, title, description)

	response, err := s.client.ChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	result, err := parseClassifyResponse(response)
	if err != nil {
		return nil, err
	}

	result.Quadrant = calculateQuadrant(result.Important, result.Urgent)

	return result, nil
}

// RescheduleResult 重排程结果
type RescheduleResult struct {
	AdjustedSchedule []AdjustedItem `json:"adjusted_schedule"`
	Summary          string         `json:"summary"`
}

// AdjustedItem 调整后的日程项
type AdjustedItem struct {
	TaskID     string `json:"task_id"`
	Title      string `json:"title"`
	StartTime  string `json:"start_time"`
	EndTime    string `json:"end_time"`
	Adjustment string `json:"adjustment"`
	Reason     string `json:"reason"`
}

// RescheduleAfterInterrupt 被打断后重新排程
func (s *AIService) RescheduleAfterInterrupt(ctx context.Context, interruptedTaskID string, completedMinutes, plannedMinutes int, interruptReason, currentTime, workEndTime string) (*RescheduleResult, error) {
	if s.client == nil {
		return nil, fmt.Errorf("AI service not configured")
	}

	// 获取被打断的任务信息
	interruptedTitle := "未知任务"
	if task, err := s.taskRepo.GetByID(interruptedTaskID); err == nil {
		interruptedTitle = task.Title
	}

	// 获取今日剩余待办任务
	tasks, err := s.taskRepo.GetByStatus(model.StatusTodo)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks: %w", err)
	}

	var taskList strings.Builder
	for _, t := range tasks {
		if t.ID == interruptedTaskID {
			continue
		}
		deadline := "无"
		if t.Deadline != nil {
			deadline = t.Deadline.Format("2006-01-02")
		}
		taskList.WriteString(fmt.Sprintf("- ID: %s, 标题: %s, 预估: %d分钟, 截止: %s\n",
			t.ID, t.Title, t.EstimatedTime, deadline))
	}

	prompt := fmt.Sprintf(ai.ReschedulePrompt,
		interruptedTitle, completedMinutes, plannedMinutes,
		interruptReason, currentTime, currentTime, workEndTime,
		taskList.String(),
	)

	response, err := s.client.ChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	jsonStr := extractJSON(response)
	var result RescheduleResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse reschedule response: %w", err)
	}

	return &result, nil
}

// DailyInsights 每日洞察
type DailyInsights struct {
	ProductivityScore int      `json:"productivity_score"`
	PeakHours         string   `json:"peak_hours"`
	Achievements      []string `json:"achievements"`
	Suggestions       []string `json:"suggestions"`
	Motivation        string   `json:"motivation"`
}

// GetDailyInsights 获取每日 AI 洞察
func (s *AIService) GetDailyInsights(ctx context.Context, date string, completedPomodoros, totalFocusMinutes, completedTasks, totalInterruptions int, taskDistribution string) (*DailyInsights, error) {
	if s.client == nil {
		return nil, fmt.Errorf("AI service not configured")
	}

	prompt := fmt.Sprintf(ai.DailyInsightsPrompt,
		date, completedPomodoros, totalFocusMinutes,
		completedTasks, totalInterruptions, taskDistribution,
	)

	response, err := s.client.ChatCompletion(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("AI request failed: %w", err)
	}

	jsonStr := extractJSON(response)
	var result DailyInsights
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse insights response: %w", err)
	}

	return &result, nil
}

// IsConfigured 检查 AI 是否已配置
func (s *AIService) IsConfigured() bool {
	return s.client != nil
}

// 辅助函数

func parseClassifyResponse(response string) (*ClassificationResult, error) {
	// 尝试提取 JSON
	jsonStr := extractJSON(response)

	var result struct {
		Important bool   `json:"important"`
		Urgent    bool   `json:"urgent"`
		Reason    string `json:"reason"`
	}

	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse AI response: %w", err)
	}

	return &ClassificationResult{
		Important: result.Important,
		Urgent:    result.Urgent,
		Reason:    result.Reason,
	}, nil
}

func parseScheduleResponse(response string) (*DailySchedule, error) {
	jsonStr := extractJSON(response)

	var result DailySchedule
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse schedule response: %w", err)
	}

	return &result, nil
}

func parsePriorityResponse(response string) (*PrioritySuggestion, error) {
	jsonStr := extractJSON(response)

	var result PrioritySuggestion
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse priority response: %w", err)
	}

	return &result, nil
}

// extractJSON 从响应中提取 JSON 内容
func extractJSON(response string) string {
	// 查找第一个 { 和最后一个 }
	start := strings.Index(response, "{")
	end := strings.LastIndex(response, "}")

	if start == -1 || end == -1 || start > end {
		return response
	}

	return response[start : end+1]
}

// calculateQuadrant 根据重要性和紧急度计算象限
func calculateQuadrant(important, urgent bool) int {
	if important && urgent {
		return 1 // 重要且紧急
	} else if important && !urgent {
		return 2 // 重要不紧急
	} else if !important && urgent {
		return 3 // 紧急不重要
	} else {
		return 4 // 不重要不紧急
	}
}

// GetAIServiceWithTimeout 获取带超时的上下文
func GetAIServiceWithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}