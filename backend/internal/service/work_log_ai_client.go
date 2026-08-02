package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"ticktask/internal/ai"
	"ticktask/internal/model"
)

// aiCallTimeout 单次 AI 调用最长等待。
// WorkLogAIClient 接口当前不传 ctx，内部用 context.WithTimeout 控制 AI 调用上限。
const aiCallTimeout = 90 * time.Second

// workLogAIClient 真实实现：调 AIService.CallLLM + JSON 解析
type workLogAIClient struct {
	aiService *AIService
}

// NewWorkLogAIClient 构造。aiService 必须非 nil；底层 client 可在运行时为 nil（CallLLM 会返回错误）。
func NewWorkLogAIClient(aiService *AIService) WorkLogAIClient {
	return &workLogAIClient{aiService: aiService}
}

func (c *workLogAIClient) callLLM(system, user string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), aiCallTimeout)
	defer cancel()
	return c.aiService.CallLLM(ctx, system, user)
}

// StructureBrainDump 拆条
func (c *workLogAIClient) StructureBrainDump(input BrainDumpInput) (*StructuredWorkLog, error) {
	userPrompt := fmt.Sprintf(ai.WorkLogStructureUser, input.BrainDump, formatContextForPrompt(input.Context))
	raw, err := c.callLLM(ai.WorkLogStructureSystem, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call: %w", err)
	}
	cleaned := stripCodeFence(raw)
	var out StructuredWorkLog
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return nil, fmt.Errorf("parse LLM JSON: %w; raw=%s", err, truncated(raw, 500))
	}
	return &out, nil
}

// GenerateWeeklyReport 周报
func (c *workLogAIClient) GenerateWeeklyReport(items []model.WorkItem, start, end string) (*ReportSummary, error) {
	userPrompt := fmt.Sprintf("本周（%s ~ %s）的工作条目 JSON：\n%s", start, end, itemsToJSON(items))
	raw, err := c.callLLM(ai.WorkLogWeeklyReportSystem, userPrompt)
	if err != nil {
		return nil, err
	}
	return parseReportSummary(raw)
}

// GenerateMonthlyReport 月报
func (c *workLogAIClient) GenerateMonthlyReport(weeklies []*model.WorkReport, orphanItems []model.WorkItem, start, end string) (*ReportSummary, error) {
	userPrompt := fmt.Sprintf("本月（%s ~ %s）的周报 JSON 数组：\n%s\n\n未被周报覆盖的零散 items：\n%s",
		start, end, reportsToJSON(weeklies), itemsToJSON(orphanItems))
	raw, err := c.callLLM(ai.WorkLogMonthlyReportSystem, userPrompt)
	if err != nil {
		return nil, err
	}
	return parseReportSummary(raw)
}

// GenerateHalfYearReport 半年报
func (c *workLogAIClient) GenerateHalfYearReport(monthlies []*model.WorkReport, start, end string) (*ReportSummary, error) {
	userPrompt := fmt.Sprintf("该半年（%s ~ %s）的月报 JSON 数组：\n%s",
		start, end, reportsToJSON(monthlies))
	raw, err := c.callLLM(ai.WorkLogHalfYearReportSystem, userPrompt)
	if err != nil {
		return nil, err
	}
	return parseReportSummary(raw)
}

// GenerateYearlyReport 年报
func (c *workLogAIClient) GenerateYearlyReport(monthlies []*model.WorkReport, start, end string) (*ReportSummary, error) {
	userPrompt := fmt.Sprintf("本年（%s ~ %s）的月报 JSON 数组：\n%s",
		start, end, reportsToJSON(monthlies))
	raw, err := c.callLLM(ai.WorkLogYearlyReportSystem, userPrompt)
	if err != nil {
		return nil, err
	}
	return parseReportSummary(raw)
}

// ── helpers ──

func formatContextForPrompt(ctx TodayContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("已完成任务 %d 条：\n", len(ctx.CompletedTasks)))
	for _, t := range ctx.CompletedTasks {
		sb.WriteString(fmt.Sprintf("- %s\n", t.Title))
	}
	sb.WriteString(fmt.Sprintf("\n番茄钟会话 %d 个，共 %d 分钟。\n",
		ctx.PomodoroSummary.Count, ctx.PomodoroSummary.TotalMinutes))
	return sb.String()
}

func itemsToJSON(items []model.WorkItem) string {
	type brief struct {
		Title         string `json:"title"`
		Content       string `json:"content"`
		ProblemSolved string `json:"problem_solved"`
		Result        string `json:"result"`
		Impact        string `json:"impact"`
	}
	out := make([]brief, len(items))
	for i, it := range items {
		out[i] = brief{
			Title:         it.Title,
			Content:       it.Content,
			ProblemSolved: it.ProblemSolved,
			Result:        it.Result,
			Impact:        it.Impact,
		}
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func reportsToJSON(reports []*model.WorkReport) string {
	out := make([]map[string]string, len(reports))
	for i, r := range reports {
		out[i] = map[string]string{
			"period_key":   r.PeriodKey,
			"start_date":   r.StartDate,
			"end_date":     r.EndDate,
			"summary_json": r.SummaryJSON,
		}
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func parseReportSummary(raw string) (*ReportSummary, error) {
	cleaned := stripCodeFence(raw)
	var s ReportSummary
	if err := json.Unmarshal([]byte(cleaned), &s); err != nil {
		return nil, fmt.Errorf("parse report JSON: %w; raw=%s", err, truncated(raw, 500))
	}
	return &s, nil
}

// stripCodeFence 移除可能的 ```json ... ``` 包裹（LLM 偶尔会包一层 markdown）
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	return strings.TrimSpace(s)
}

func truncated(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
