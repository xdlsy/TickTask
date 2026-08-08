package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"ticktask/internal/ai"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/internal/websocket"
	"ticktask/pkg/logger"
	"time"

	"github.com/google/uuid"
)

type ScheduleService struct {
	scheduleRepo repository.ScheduleRepository
	taskRepo     repository.TaskRepository
	llm          ai.LLMClient
	settingRepo  repository.SettingRepository
	wsHub        *websocket.Hub
}

func NewScheduleService(
	scheduleRepo repository.ScheduleRepository,
	taskRepo repository.TaskRepository,
	llm ai.LLMClient,
	settingRepo repository.SettingRepository,
	wsHub *websocket.Hub,
) *ScheduleService {
	return &ScheduleService{
		scheduleRepo: scheduleRepo,
		taskRepo:     taskRepo,
		llm:          llm,
		settingRepo:  settingRepo,
		wsHub:        wsHub,
	}
}

// ScheduleEvent 日程事件（用于日历显示）
type ScheduleEvent struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	Start     string `json:"start"`
	End       string `json:"end"`
	Type      string `json:"type"`
	Status    string `json:"status"`
	Color     string `json:"color"`
	TaskID    string `json:"task_id,omitempty"`
	AllDay    bool   `json:"allDay"`
	Editable  bool   `json:"editable"`
}

// CreateScheduleDTO 创建日程请求
type CreateScheduleDTO struct {
	TaskID      string `json:"task_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Type        string `json:"type"`
	Color       string `json:"color"`
}

// UpdateScheduleDTO 更新日程请求
type UpdateScheduleDTO struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	StartTime   string `json:"start_time"`
	EndTime     string `json:"end_time"`
	Status      string `json:"status"`
	Color       string `json:"color"`
}

// MoveScheduleDTO 移动日程请求
type MoveScheduleDTO struct {
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

// ReviseRequest 修订日程请求
type ReviseRequest struct {
	Prompt string `json:"prompt"`
}

// RevisionChange 单个修订变更项
type RevisionChange struct {
	Type          string `json:"type"` // "moved" | "added" | "removed"
	Title         string `json:"title"`
	OriginalStart string `json:"original_start,omitempty"`
	OriginalEnd   string `json:"original_end,omitempty"`
	NewStart      string `json:"new_start,omitempty"`
	NewEnd        string `json:"new_end,omitempty"`
}

// ReviseResponse 修订日程响应
type ReviseResponse struct {
	Applied bool             `json:"applied"`
	Summary string           `json:"summary"`
	Changes []RevisionChange `json:"changes"`
	Events  []ScheduleEvent  `json:"events"`
}


// currentWeekRange returns Monday 00:00 and Sunday 00:00 of the current week.
func currentWeekRange() (monday, sunday time.Time) {
	now := time.Now()
	weekday := now.Weekday()
	offset := int(weekday) - 1
	if weekday == time.Sunday {
		offset = 6
	}
	monday = time.Date(now.Year(), now.Month(), now.Day()-offset, 0, 0, 0, 0, time.Now().Location())
	sunday = monday.AddDate(0, 0, 6)
	return
}

// GetSchedules 获取日程列表
func (s *ScheduleService) GetSchedules(start, end time.Time) ([]ScheduleEvent, error) {
	schedules, err := s.scheduleRepo.GetByTimeRange(start, end)
	if err != nil {
		return nil, err
	}

	events := make([]ScheduleEvent, len(schedules))
	for i, schedule := range schedules {
		events[i] = s.toEvent(&schedule)
	}
	return events, nil
}

// GetSchedule 获取单个日程
func (s *ScheduleService) GetSchedule(id string) (*model.Schedule, error) {
	return s.scheduleRepo.GetByID(id)
}

// CreateSchedule 创建日程（返回 model）
func (s *ScheduleService) CreateSchedule(dto *CreateScheduleDTO) (*model.Schedule, error) {
	startTime, err := time.Parse(time.RFC3339, dto.StartTime)
	if err != nil {
		return nil, fmt.Errorf("invalid start_time format")
	}

	endTime, err := time.Parse(time.RFC3339, dto.EndTime)
	if err != nil {
		return nil, fmt.Errorf("invalid end_time format")
	}

	schedule := &model.Schedule{
		ID:          uuid.New().String(),
		Title:       dto.Title,
		Description: dto.Description,
		StartTime:   startTime,
		EndTime:     endTime,
		Type:        model.ScheduleType(dto.Type),
		Status:      model.ScheduleStatusPlanned,
		Color:       dto.Color,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if dto.TaskID != "" {
		schedule.TaskID = &dto.TaskID
	}

	// 如果没有设置标题，尝试从任务获取
	if schedule.Title == "" && schedule.TaskID != nil {
		task, err := s.taskRepo.GetByID(*schedule.TaskID)
		if err == nil {
			schedule.Title = task.Title
		}
	}

	// 设置默认颜色
	if schedule.Color == "" {
		schedule.Color = s.getDefaultColor(schedule.Type)
	}

	if err := s.scheduleRepo.Create(schedule); err != nil {
		return nil, err
	}

	return schedule, nil
}

// CreateScheduleEvent 创建日程（返回前端事件格式）
func (s *ScheduleService) CreateScheduleEvent(dto *CreateScheduleDTO) (*ScheduleEvent, error) {
	schedule, err := s.CreateSchedule(dto)
	if err != nil {
		return nil, err
	}
	event := s.toEvent(schedule)
	return &event, nil
}

// UpdateSchedule 更新日程
func (s *ScheduleService) UpdateSchedule(id string, dto *UpdateScheduleDTO) error {
	schedule, err := s.scheduleRepo.GetByID(id)
	if err != nil {
		return err
	}

	if dto.Title != "" {
		schedule.Title = dto.Title
	}
	if dto.Description != "" {
		schedule.Description = dto.Description
	}
	if dto.StartTime != "" {
		startTime, err := time.Parse(time.RFC3339, dto.StartTime)
		if err != nil {
			return fmt.Errorf("invalid start_time format")
		}
		schedule.StartTime = startTime
	}
	if dto.EndTime != "" {
		endTime, err := time.Parse(time.RFC3339, dto.EndTime)
		if err != nil {
			return fmt.Errorf("invalid end_time format")
		}
		schedule.EndTime = endTime
	}
	if dto.Status != "" {
		schedule.Status = model.ScheduleStatus(dto.Status)
	}
	if dto.Color != "" {
		schedule.Color = dto.Color
	}

	schedule.UpdatedAt = time.Now()
	return s.scheduleRepo.Update(schedule)
}

// DeleteSchedule 删除日程
func (s *ScheduleService) DeleteSchedule(id string) error {
	return s.scheduleRepo.Delete(id)
}

// DeleteAllSchedules 清空所有日程
func (s *ScheduleService) DeleteAllSchedules() (int64, error) {
	return s.scheduleRepo.DeleteAll()
}

// MoveSchedule 移动日程
func (s *ScheduleService) MoveSchedule(id string, startTime, endTime time.Time) error {
	return s.scheduleRepo.Move(id, startTime, endTime)
}

// UpdateScheduleStatus 更新日程状态
func (s *ScheduleService) UpdateScheduleStatus(id string, status model.ScheduleStatus) error {
	return s.scheduleRepo.UpdateStatus(id, status)
}

// GenerateScheduleWithAI AI生成日程
func (s *ScheduleService) GenerateScheduleWithAI(startTime, endTime string) ([]ScheduleEvent, error) {
	start, err := time.Parse("15:04", startTime)
	if err != nil {
		return nil, err
	}

	// 清理当天已有的任务日程（避免重复）
	now := time.Now()
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Now().Location())
	dayEnd := dayStart.Add(24 * time.Hour)
	deleted, _ := s.scheduleRepo.DeleteTaskSchedulesByDateRange(dayStart, dayEnd)
	logger.Logger.Info("cleaned old task schedules before AI generation", "deleted", deleted)

	// 获取待办任务
	tasks, err := s.taskRepo.GetByStatus(model.StatusTodo)
	if err != nil {
		return nil, err
	}

	// 简单按优先级安排日程
	return s.simpleScheduleWithStart(tasks, start)
}

type timeSlot struct {
	start time.Time
	end   time.Time
}

func (s *ScheduleService) simpleScheduleWithStart(tasks []model.Task, start time.Time) ([]ScheduleEvent, error) {
	var events []ScheduleEvent
	now := time.Now()
	loc := time.Now().Location()

	var occupied []timeSlot

	// 第一轮：安排有固定时段偏好的任务
	for _, task := range tasks {
		if task.PreferredStartTime == "" || task.PreferredEndTime == "" {
			continue
		}

		slotStart, err := time.Parse("15:04", task.PreferredStartTime)
		if err != nil {
			continue
		}
		slotEnd, err := time.Parse("15:04", task.PreferredEndTime)
		if err != nil {
			continue
		}

		scheduleStart := time.Date(now.Year(), now.Month(), now.Day(), slotStart.Hour(), slotStart.Minute(), 0, 0, loc)
		scheduleEnd := time.Date(now.Year(), now.Month(), now.Day(), slotEnd.Hour(), slotEnd.Minute(), 0, 0, loc)

		if !scheduleEnd.After(scheduleStart) {
			continue
		}

		event, err := s.createScheduleEvent(task, scheduleStart, scheduleEnd)
		if err != nil {
			continue
		}
		events = append(events, event)
		occupied = append(occupied, timeSlot{scheduleStart, scheduleEnd})
	}

	// 第二轮：用光标法，在空闲时段中安排无时段偏好的任务
	cursor := time.Date(now.Year(), now.Month(), now.Day(), start.Hour(), start.Minute(), 0, 0, loc)

	for _, task := range tasks {
		if task.PreferredStartTime != "" && task.PreferredEndTime != "" {
			continue
		}

		duration := 30 * time.Minute
		if task.EstimatedTime > 0 {
			duration = time.Duration(task.EstimatedTime) * time.Minute
		}

		scheduleStart, scheduleEnd := findNextAvailableSlot(cursor, duration, occupied)
		event, err := s.createScheduleEvent(task, scheduleStart, scheduleEnd)
		if err != nil {
			continue
		}
		events = append(events, event)
		occupied = append(occupied, timeSlot{scheduleStart, scheduleEnd})
		cursor = scheduleEnd
	}

	return events, nil
}

// findNextAvailableSlot finds the next unoccupied time window starting from cursor.
func findNextAvailableSlot(cursor time.Time, duration time.Duration, occupied []timeSlot) (time.Time, time.Time) {
	candidateStart := cursor
	candidateEnd := candidateStart.Add(duration)

	for {
		collision := false
		for _, slot := range occupied {
			if candidateStart.Before(slot.end) && candidateEnd.After(slot.start) {
				candidateStart = slot.end
				candidateEnd = candidateStart.Add(duration)
				collision = true
				break
			}
		}
		if !collision {
			return candidateStart, candidateEnd
		}
	}
}

func (s *ScheduleService) createScheduleEvent(task model.Task, scheduleStart, scheduleEnd time.Time) (ScheduleEvent, error) {
	dto := &CreateScheduleDTO{
		TaskID:    task.ID,
		Title:     task.Title,
		StartTime: scheduleStart.Format(time.RFC3339),
		EndTime:   scheduleEnd.Format(time.RFC3339),
		Type:      "task",
		Color:     s.getQuadrantColor(task.Quadrant),
	}

	schedule, err := s.CreateSchedule(dto)
	if err != nil {
		return ScheduleEvent{}, err
	}

	return s.toEvent(schedule), nil
}

func (s *ScheduleService) toEvent(schedule *model.Schedule) ScheduleEvent {
	event := ScheduleEvent{
		ID:       schedule.ID,
		Title:    schedule.Title,
		Start:    schedule.StartTime.Format(time.RFC3339),
		End:      schedule.EndTime.Format(time.RFC3339),
		Type:     string(schedule.Type),
		Status:   string(schedule.Status),
		Color:    schedule.Color,
		AllDay:   false,
		Editable: schedule.Status != model.ScheduleStatusCompleted,
	}

	if schedule.TaskID != nil {
		event.TaskID = *schedule.TaskID
	}

	return event
}

func (s *ScheduleService) getDefaultColor(scheduleType model.ScheduleType) string {
	switch scheduleType {
	case model.ScheduleTypeTask:
		return "#3b82f6"
	case model.ScheduleTypePomodoro:
		return "#f59e0b"
	case model.ScheduleTypeBreak:
		return "#22c55e"
	default:
		return "#6b7280"
	}
}

func (s *ScheduleService) getQuadrantColor(quadrant model.Quadrant) string {
	switch quadrant {
	case model.Quadrant1:
		return "#ef4444"
	case model.Quadrant2:
		return "#f59e0b"
	case model.Quadrant3:
		return "#3b82f6"
	case model.Quadrant4:
		return "#6b7280"
	default:
		return "#3b82f6"
	}
}
// OutputCallback receives streaming output chunks.
type OutputCallback func(chunk string, isStderr bool)

// GenerateSchedule 写配置文件并调用 Claude 执行 auto-schedule skill。
// Claude 输出通过 WebSocket 实时流式推送到前端终端。
func (s *ScheduleService) GenerateSchedule(startTime, endTime string) ([]ScheduleEvent, string, error) {
	monday, sunday := currentWeekRange()

	broadcast := func(chunk string, isStderr bool) {
		if s.wsHub != nil {
			s.wsHub.BroadcastTerminalOutput(chunk, isStderr)
		}
	}
	broadcastStatus := func(status, message, detail string) {
		if s.wsHub != nil {
			s.wsHub.BroadcastTerminalStatus(status, message, detail)
		}
	}

	broadcastStatus("started", "启动日程排程引擎",
		fmt.Sprintf("日期范围: %s — %s | 工作时间: %s — %s",
			monday.Format("01/02"), sunday.Format("01/02"), startTime, endTime))

	// 清理本周已有的任务日程
	deleted, _ := s.scheduleRepo.DeleteTaskSchedulesByDateRange(monday, sunday.Add(24*time.Hour))
	logger.Logger.Info("cleaned old task schedules", "from", monday.Format("2006-01-02"), "to", sunday.Format("2006-01-02"), "deleted", deleted)

	// 获取并过滤任务
	allTasks, err := s.taskRepo.GetAll()
	if err != nil {
		broadcastStatus("error", "获取任务失败", err.Error())
		return nil, "", fmt.Errorf("获取任务失败: %w", err)
	}

	candidates := filterTasksForWeek(allTasks, monday, sunday)
	sortTasksForScheduling(candidates)

	if len(candidates) == 0 {
		broadcastStatus("completed", "日程生成完成", "当前没有需要排程的任务")
		return []ScheduleEvent{}, "当前没有需要排程的任务", nil
	}

	pomodoroSettings, _ := s.getPomodoroSettings()

	// 写配置文件（全周共用，LLM 通过 skill 读取）
	if err := WriteHabitMD(pomodoroSettings, startTime, endTime); err != nil {
		broadcastStatus("error", "写入配置失败", err.Error())
		return nil, "", fmt.Errorf("写入 habit.md: %w", err)
	}
	if err := WriteTodoJSON(candidates, monday); err != nil {
		broadcastStatus("error", "写入配置失败", err.Error())
		return nil, "", fmt.Errorf("写入 todo.json: %w", err)
	}
	broadcast(fmt.Sprintf("✓ 配置文件已刷新: %d 个任务\n", len(candidates)), false)

	allEvents := make([]ScheduleEvent, 0)
	seenEventKeys := map[string]bool{}

	// ======== Phase 1: AI 生成 ICS ========
	if s.llm == nil {
		broadcastStatus("error", "AI 未配置", "请在设置中配置 AI 服务")
		return nil, "", fmt.Errorf("AI not configured")
	}

	weekPrompt := buildWeekSchedulePrompt(monday, sunday)
	broadcastStatus("started", "AI 排程引擎",
		fmt.Sprintf("日期: %s — %s | %d 任务", monday.Format("01/02"), sunday.Format("01/02"), len(candidates)))
	broadcast(fmt.Sprintf("$ claude -p \"%s\"\n\n", weekPrompt), false)
	if aiErr := runClaudeStreamJSON(weekPrompt, broadcast); aiErr != nil {
		broadcastStatus("error", "AI 排程失败", aiErr.Error())
		return nil, "", fmt.Errorf("AI scheduling failed: %w", aiErr)
	}

	// ======== Phase 2: 读取 AI 生成的 ICS ========
	icsContent, err := ReadScheduleICS()
	if err != nil {
		broadcastStatus("error", "读取日程文件失败", err.Error())
		return nil, "", fmt.Errorf("read schedule.ics: %w", err)
	}
	broadcast(fmt.Sprintf("✓ ICS 已读取，共 %d 字节\n", len(icsContent)), false)

	// ======== Phase 3: 解析入库 ========
	parsedEvents, err := ParseICS(icsContent, time.Now().Location())
	if err != nil {
		broadcastStatus("error", "解析日程文件失败", err.Error())
		return nil, "", fmt.Errorf("parse ICS: %w", err)
	}

	for _, ev := range parsedEvents {
		eventDate := ev.Start
		if eventDate.IsZero() {
			continue
		}
		d := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, eventDate.Location())
		matched := matchTaskByTitle(candidates, ev.Summary)
		dto := icsEventToDTO(ev, d)
		if dto == nil {
			continue
		}
		if matched != nil {
			dto.TaskID = matched.ID
			dto.Title = matched.Title
			dto.Type = string(model.ScheduleTypeTask)
			dto.Color = "#3b82f6"
		}
		if dto.TaskID != "" {
			eventKey := fmt.Sprintf("%s|%s|%s", dto.TaskID, d.Format("2006-01-02"), dto.StartTime)
			if seenEventKeys[eventKey] {
				continue
			}
			seenEventKeys[eventKey] = true
		}
		schedule, err := s.CreateSchedule(dto)
		if err != nil {
			logger.Logger.Warn("failed to create schedule", "title", dto.Title, "error", err)
			continue
		}
		allEvents = append(allEvents, s.toEvent(schedule))
	}

	// ======== Phase 4: 偏好 + 重复日校验（仅报告）========
	prefMismatches := validateDayEvents(allEvents, candidates)
	recMismatches := validateRecurrenceDays(allEvents, candidates)

	if len(prefMismatches) > 0 {
		logger.Logger.Info("preference mismatches", "count", len(prefMismatches))
		broadcast(fmt.Sprintf("⚠ %d 个偏好时段不匹配\n", len(prefMismatches)), true)
	} else {
		broadcast("✅ 偏好时段校验全部通过\n", false)
	}
	if len(recMismatches) > 0 {
		logger.Logger.Info("recurrence mismatches", "count", len(recMismatches))
		broadcast(fmt.Sprintf("⚠ %d 个重复日不匹配\n", len(recMismatches)), true)
		for _, rm := range recMismatches {
			broadcast(fmt.Sprintf("  - %s\n", rm.String()), true)
		}
	} else {
		broadcast("✅ 重复日校验全部通过\n", false)
	}

	scheduledCount := 0
	for _, ev := range allEvents {
		if ev.TaskID != "" {
			scheduledCount++
		}
	}

	taskCount := 0
	seenTasks := map[string]bool{}
	for _, ev := range allEvents {
		if ev.TaskID != "" && !seenTasks[ev.TaskID] {
			seenTasks[ev.TaskID] = true
			taskCount++
		}
	}
	summary := fmt.Sprintf("成功生成 %d 个日程安排，覆盖 %s 至 %s（共 %d 个任务）",
		len(allEvents), monday.Format("01/02"), sunday.Format("01/02"), taskCount)

	broadcastStatus("completed", "日程生成完成", summary)

	return allEvents, summary, nil
}

// buildSkillPrompt builds a prompt to invoke a scheduling skill via Claude CLI.
func buildSkillPrompt(skillName, action string, monday, sunday time.Time, extra string) string {
	root := findProjectRoot()
	base := fmt.Sprintf("项目路径: %s。执行 docs/skills/%s skill，为 %s 至 %s%s。",
		root, skillName, monday.Format("2006-01-02"), sunday.Format("2006-01-02"), action)
	if extra != "" {
		base += extra
	}
	return base
}

// ReviseSchedule runs the revise-schedule skill via Claude CLI, compares the original
// and revised ICS, and returns a preview of changes without writing to the database.
func (s *ScheduleService) ReviseSchedule(prompt string) (*ReviseResponse, error) {
	// 1. Compute week range (same algorithm as GenerateSchedule)
	monday, sunday := currentWeekRange()

	// 2. Set up WebSocket broadcast closures
	broadcast := func(chunk string, isStderr bool) {
		if s.wsHub != nil {
			s.wsHub.BroadcastTerminalOutput(chunk, isStderr)
		}
	}
	broadcastStatus := func(status, message, detail string) {
		if s.wsHub != nil {
			s.wsHub.BroadcastTerminalStatus(status, message, detail)
		}
	}

	broadcastStatus("started", "启动日程修订引擎",
		fmt.Sprintf("日期范围: %s — %s | 修订指令: %s",
			monday.Format("01/02"), sunday.Format("01/02"), prompt))

	// 3. Write current schedules as baseline ICS
	currentEvents, err := s.GetSchedules(monday, sunday.Add(24*time.Hour))
	if err != nil {
		broadcastStatus("error", "获取当前日程失败", err.Error())
		return nil, fmt.Errorf("获取当前日程: %w", err)
	}

	if err := WriteScheduleICS(currentEvents); err != nil {
		broadcastStatus("error", "写入日程基线失败", err.Error())
		return nil, fmt.Errorf("写入日程基线: %w", err)
	}

	// Save original ICS content for diff comparison
	originalICS, err := ReadScheduleICS()
	if err != nil {
		broadcastStatus("error", "读取日程基线失败", err.Error())
		return nil, fmt.Errorf("读取日程基线: %w", err)
	}

	// 4. Get and filter tasks, write config files
	allTasks, err := s.taskRepo.GetAll()
	if err != nil {
		broadcastStatus("error", "获取任务失败", err.Error())
		return nil, fmt.Errorf("获取任务: %w", err)
	}

	candidates := filterTasksForWeek(allTasks, monday, sunday)
	sortTasksForScheduling(candidates)

	pomodoroSettings, _ := s.getPomodoroSettings()

	// Use default work hours for config generation
	if err := WriteHabitMD(pomodoroSettings, "09:00", "18:00"); err != nil {
		broadcastStatus("error", "写入配置失败", err.Error())
		return nil, fmt.Errorf("写入 habit.md: %w", err)
	}
	if err := WriteTodoJSON(candidates, monday); err != nil {
		broadcastStatus("error", "写入配置失败", err.Error())
		return nil, fmt.Errorf("写入 todo.json: %w", err)
	}
	broadcast(fmt.Sprintf("✓ 配置文件已刷新: %d 个任务，%d 个当前日程事件\n", len(candidates), len(currentEvents)), false)

	// 5. Check AI is configured
	if s.llm == nil {
		broadcastStatus("error", "AI 未配置", "请在设置中配置 AI 服务")
		return nil, fmt.Errorf("AI not configured")
	}

	// 6. Build prompt and run Claude CLI
	revisePrompt := buildSkillPrompt("revise-schedule", "修订整周日程", monday, sunday, "。修订指令："+prompt)
	broadcastStatus("started", "AI 修订引擎",
		fmt.Sprintf("日期: %s — %s | %d 任务 | %d 当前事件",
			monday.Format("01/02"), sunday.Format("01/02"), len(candidates), len(currentEvents)))
	broadcast(fmt.Sprintf("$ claude -p \"%s\"\n\n", revisePrompt), false)

	if aiErr := runClaudeStreamJSON(revisePrompt, broadcast); aiErr != nil {
		broadcastStatus("error", "AI 修订失败", aiErr.Error())
		return nil, fmt.Errorf("AI revision failed: %w", aiErr)
	}

	// 7. Read revised ICS (Claude overwrites schedule.ics)
	revisedICS, err := ReadScheduleICS()
	if err != nil {
		broadcastStatus("error", "读取修订后日程失败", err.Error())
		return nil, fmt.Errorf("读取修订后日程: %w", err)
	}

	// 8. Parse both ICS files
	originalEvents, err := ParseICS(originalICS, time.Now().Location())
	if err != nil {
		broadcastStatus("error", "解析原始日程失败", err.Error())
		return nil, fmt.Errorf("解析原始日程: %w", err)
	}

	revisedEvents, err := ParseICS(revisedICS, time.Now().Location())
	if err != nil {
		broadcastStatus("error", "解析修订后日程失败", err.Error())
		return nil, fmt.Errorf("解析修订后日程: %w", err)
	}

	// 9. Compute diff
	changes, summary := computeDiff(originalEvents, revisedEvents)

	broadcast(fmt.Sprintf("✓ 修订分析完成: %s\n", summary), false)
	broadcastStatus("completed", "日程修订分析完成", summary)

	return &ReviseResponse{
		Applied: false,
		Summary: summary,
		Changes: changes,
		Events:  []ScheduleEvent{},
	}, nil
}

// ApplyRevision applies the revised schedule from schedule.ics to the database.
// It deletes old task schedules for the current week and persists the new ones.
func (s *ScheduleService) ApplyRevision() ([]ScheduleEvent, error) {
	monday, sunday := currentWeekRange()

	// Read and parse revised ICS
	revisedICS, err := ReadScheduleICS()
	if err != nil {
		return nil, fmt.Errorf("读取修订后日程: %w", err)
	}

	parsedEvents, err := ParseICS(revisedICS, time.Now().Location())
	if err != nil {
		return nil, fmt.Errorf("解析修订后日程: %w", err)
	}

	// Get tasks for title matching
	allTasks, err := s.taskRepo.GetAll()
	if err != nil {
		return nil, fmt.Errorf("获取任务: %w", err)
	}
	candidates := filterTasksForWeek(allTasks, monday, sunday)

	// Delete old task schedules for the week
	deleted, _ := s.scheduleRepo.DeleteTaskSchedulesByDateRange(monday, sunday.Add(24*time.Hour))
	logger.Logger.Info("apply revision: cleaned old task schedules",
		"from", monday.Format("2006-01-02"),
		"to", sunday.Format("2006-01-02"),
		"deleted", deleted)

	// Persist revised events (reuse same pattern as GenerateSchedule Phase 3)
	allEvents := make([]ScheduleEvent, 0)
	seenEventKeys := map[string]bool{}

	for _, ev := range parsedEvents {
		eventDate := ev.Start
		if eventDate.IsZero() {
			continue
		}
		d := time.Date(eventDate.Year(), eventDate.Month(), eventDate.Day(), 0, 0, 0, 0, eventDate.Location())
		matched := matchTaskByTitle(candidates, ev.Summary)
		dto := icsEventToDTO(ev, d)
		if dto == nil {
			continue
		}
		if matched != nil {
			dto.TaskID = matched.ID
			dto.Title = matched.Title
			dto.Type = string(model.ScheduleTypeTask)
			dto.Color = "#3b82f6"
		}
		if dto.TaskID != "" {
			eventKey := fmt.Sprintf("%s|%s|%s", dto.TaskID, d.Format("2006-01-02"), dto.StartTime)
			if seenEventKeys[eventKey] {
				continue
			}
			seenEventKeys[eventKey] = true
		}
		schedule, err := s.CreateSchedule(dto)
		if err != nil {
			logger.Logger.Warn("apply revision: failed to create schedule", "title", dto.Title, "error", err)
			continue
		}
		allEvents = append(allEvents, s.toEvent(schedule))
	}


	return allEvents, nil
}

// eventKey uniquely identifies an ICS event by title and date.
type eventKey struct {
	title string
	date  string // YYYY-MM-DD
}

// indexEventsByTitleDate builds a lookup map keyed by (title, date).
func indexEventsByTitleDate(events []ICSEvent) map[eventKey]ICSEvent {
	m := map[eventKey]ICSEvent{}
	for _, ev := range events {
		if ev.Summary == "" {
			continue
		}
		m[eventKey{title: ev.Summary, date: ev.Start.Format("2006-01-02")}] = ev
	}
	return m
}

// computeDiff compares original and revised ICS events and returns a list of changes.
func computeDiff(originalEvents, revisedEvents []ICSEvent) ([]RevisionChange, string) {
	origByKey := indexEventsByTitleDate(originalEvents)
	revByKey := indexEventsByTitleDate(revisedEvents)

	var changes []RevisionChange
	moved, added, removed := 0, 0, 0

	// Find moved and added events
	for key, revEv := range revByKey {
		origEv, exists := origByKey[key]
		if !exists {
			added++
			changes = append(changes, RevisionChange{
				Type:     "added",
				Title:    revEv.Summary,
				NewStart: revEv.Start.Format(time.RFC3339),
				NewEnd:   revEv.End.Format(time.RFC3339),
			})
		} else {
			origStart := origEv.Start.Format(time.RFC3339)
			origEnd := origEv.End.Format(time.RFC3339)
			newStart := revEv.Start.Format(time.RFC3339)
			newEnd := revEv.End.Format(time.RFC3339)
			if origStart != newStart || origEnd != newEnd {
				moved++
				changes = append(changes, RevisionChange{
					Type:          "moved",
					Title:         revEv.Summary,
					OriginalStart: origStart,
					OriginalEnd:   origEnd,
					NewStart:      newStart,
					NewEnd:        newEnd,
				})
			}
		}
	}

	// Find removed events
	for key, origEv := range origByKey {
		if _, exists := revByKey[key]; !exists {
			removed++
			changes = append(changes, RevisionChange{
				Type:          "removed",
				Title:         origEv.Summary,
				OriginalStart: origEv.Start.Format(time.RFC3339),
				OriginalEnd:   origEv.End.Format(time.RFC3339),
			})
		}
	}

	total := len(changes)
	var summary string
	if total == 0 {
		summary = "当前日程已是最优安排，无需调整"
	} else {
		summary = fmt.Sprintf("共调整 %d 个任务：%d 个移动，%d 个新增，%d 个移除",
			total, moved, added, removed)
	}

	return changes, summary
}

// icsEventToDTO converts a parsed ICS event to a CreateScheduleDTO.
func icsEventToDTO(ev ICSEvent, fallbackDate time.Time) *CreateScheduleDTO {
	if ev.Summary == "" {
		return nil
	}
	start := ev.Start
	end := ev.End
	if start.IsZero() {
		start = fallbackDate
	}
	if end.IsZero() || !end.After(start) {
		end = start.Add(30 * time.Minute)
	}

	// 过滤掉休息、午餐、缓冲等非任务日程
	if isBreakTitle(ev.Summary) {
		return nil
	}

	return &CreateScheduleDTO{
		Title:       ev.Summary,
		Description: ev.Description,
		StartTime:   start.Format(time.RFC3339),
		EndTime:     end.Format(time.RFC3339),
		Type:        string(model.ScheduleTypeTask),
		Color:       "#3b82f6",
	}
}

func isBreakTitle(title string) bool {
	for _, kw := range []string{"休息", "午餐", "缓冲", "弹性"} {
		if strings.Contains(title, kw) {
			return true
		}
	}
	return false
}

func matchTaskByTitle(tasks []model.Task, summary string) *model.Task {
	cleanSummary := summary
	if idx := strings.LastIndex(summary, " ("); idx > 0 && strings.HasSuffix(summary, ")") {
		if strings.Contains(summary[idx+2:len(summary)-1], "/") {
			cleanSummary = summary[:idx]
		}
	}
	for i := range tasks {
		if tasks[i].Title == cleanSummary || tasks[i].Title == summary {
			return &tasks[i]
		}
	}
	return nil
}

func (s *ScheduleService) getPomodoroSettings() (*model.PomodoroSettings, error) {
	if s.settingRepo != nil {
		return s.settingRepo.GetPomodoroSettings()
	}
	return model.DefaultPomodoroSettings(), nil
}

// --- Project root ---

func findProjectRoot() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			if _, err := os.Stat(filepath.Join(dir, "docs")); err == nil {
				return dir
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return dir
		}
		dir = parent
	}
}

// --- Claude stream-json ---

type claudeStreamMessage struct {
	Type    string `json:"type"`
	Message struct {
		Content []struct {
			Type  string `json:"type"`
			Text  string `json:"text"`
			Name  string `json:"name"`
			Input struct {
				FilePath string `json:"file_path"`
				Command  string `json:"command"`
			} `json:"input"`
		} `json:"content"`
	} `json:"message"`
}

func runClaudeStreamJSON(prompt string, onOutput OutputCallback) error {
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // 整周生成需要更长时间
	defer cancel()

	cmd := exec.CommandContext(ctx, "claude", "-p", prompt,
		"--output-format", "stream-json", "--verbose",
		"--permission-mode", "acceptEdits",
		"--dangerously-skip-permissions",
		"--setting-sources", "user,project,local")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("stdout pipe: %w", err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("start claude: %w", err)
	}

	go func() {
		buf := make([]byte, 256)
		for {
			n, err := stderr.Read(buf)
			if n > 0 && onOutput != nil {
				onOutput(string(buf[:n]), true)
			}
			if err != nil {
				break
			}
		}
	}()

	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var msg claudeStreamMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			if onOutput != nil {
				onOutput(string(line)+"\n", false)
			}
			continue
		}
		if msg.Type == "assistant" {
			for _, content := range msg.Message.Content {
				switch content.Type {
				case "text":
					if content.Text != "" && onOutput != nil {
						onOutput(content.Text, false)
					}
				case "tool_use":
					if onOutput != nil {
						switch content.Name {
						case "Bash":
							if content.Input.Command != "" {
								onOutput(fmt.Sprintf("\n$ %s\n", content.Input.Command), false)
							}
						default:
							onOutput(fmt.Sprintf("\n🔧 %s", content.Name), false)
							if content.Input.FilePath != "" {
								onOutput(fmt.Sprintf(" %s", content.Input.FilePath), false)
							}
							onOutput("\n", false)
						}
					}
				}
			}
		}
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("claude error: %w", err)
	}
	return nil
}

// validateDayEvents checks generated events against task preferred time windows.
// Returns mismatches that need correction.
func validateDayEvents(dayEvents []ScheduleEvent, tasks []model.Task) []ValidationMismatch {
	var mismatches []ValidationMismatch
	taskMap := make(map[string]model.Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	for _, ev := range dayEvents {
		if ev.TaskID == "" {
			continue
		}
		task, ok := taskMap[ev.TaskID]
		if !ok || task.PreferredStartTime == "" || task.PreferredEndTime == "" {
			continue
		}

		// Parse event start time (RFC3339 string)
		startTime, err := time.Parse(time.RFC3339, ev.Start)
		if err != nil {
			continue
		}
		endTime, _ := time.Parse(time.RFC3339, ev.End)

		prefStart, err1 := time.Parse("15:04", task.PreferredStartTime)
		prefEnd, err2 := time.Parse("15:04", task.PreferredEndTime)
		if err1 != nil || err2 != nil {
			continue
		}

		// Convert preferred times to the event's date
		prefStartDT := time.Date(startTime.Year(), startTime.Month(), startTime.Day(),
			prefStart.Hour(), prefStart.Minute(), 0, 0, startTime.Location())
		prefEndDT := time.Date(startTime.Year(), startTime.Month(), startTime.Day(),
			prefEnd.Hour(), prefEnd.Minute(), 0, 0, startTime.Location())

		// Check if event starts within the preferred window
		if startTime.Before(prefStartDT) || startTime.After(prefEndDT) {
			mismatches = append(mismatches, ValidationMismatch{
				TaskID:         task.ID,
				TaskTitle:      task.Title,
				PreferredStart: task.PreferredStartTime,
				PreferredEnd:   task.PreferredEndTime,
				ActualStart:    startTime,
				ActualEnd:      endTime,
			})
		}
	}
	return mismatches
}

// validateRecurrenceDays checks generated events against task recurrence constraints.
func validateRecurrenceDays(dayEvents []ScheduleEvent, tasks []model.Task) []RecurrenceMismatch {
	var mismatches []RecurrenceMismatch
	taskMap := make(map[string]model.Task)
	for _, t := range tasks {
		taskMap[t.ID] = t
	}

	// Group events by task ID
	eventsByTask := make(map[string][]ScheduleEvent)
	for _, ev := range dayEvents {
		if ev.TaskID != "" {
			eventsByTask[ev.TaskID] = append(eventsByTask[ev.TaskID], ev)
		}
	}

	// Collect all dates that have events
	allDates := make(map[string]bool)
	for _, ev := range dayEvents {
		startTime, err := time.Parse(time.RFC3339, ev.Start)
		if err != nil {
			continue
		}
		allDates[startTime.Format("2006-01-02")] = true
	}

	dayNames := []string{"", "周一", "周二", "周三", "周四", "周五", "周六", "周日"}

	for _, task := range tasks {
		taskEvents := eventsByTask[task.ID]

		if !task.IsRecurring || task.RecurrencePattern == "" {
			// Non-recurring tasks should appear at most once
			if len(taskEvents) > 1 {
				var dates []string
				for _, ev := range taskEvents {
					st, _ := time.Parse(time.RFC3339, ev.Start)
					if !st.IsZero() {
						dates = append(dates, st.Format("2006-01-02"))
					}
				}
				mismatches = append(mismatches, RecurrenceMismatch{
					TaskID:    task.ID,
					TaskTitle: task.Title,
					Reason:    fmt.Sprintf("非重复任务出现了 %d 次，预期最多 1 次", len(taskEvents)),
					Expected:  "最多 1 次",
					Actual:    fmt.Sprintf("%d 次 (%s)", len(taskEvents), stringsJoin(dates, ", ")),
				})
			}
			continue
		}

		switch task.RecurrencePattern {
		case "weekly":
			expectedWeekday := task.RecurrenceDay // 1=Mon..7=Sun
			if expectedWeekday < 1 || expectedWeekday > 7 {
				continue
			}

			if len(taskEvents) == 0 {
				// Find the expected date in the range
				for dateStr := range allDates {
					d, _ := time.Parse("2006-01-02", dateStr)
					if d.IsZero() {
						continue
					}
					wd := int(d.Weekday())
					if wd == 0 {
						wd = 7
					}
					if wd == expectedWeekday {
						mismatches = append(mismatches, RecurrenceMismatch{
							TaskID:    task.ID,
							TaskTitle: task.Title,
							Date:      dateStr,
							Reason:    fmt.Sprintf("每周重复任务应在%s出现，但完全缺失", dayNames[expectedWeekday]),
							Expected:  fmt.Sprintf("应在 %s 出现 1 次", dayNames[expectedWeekday]),
							Actual:    "缺失",
						})
						break
					}
				}
			} else if len(taskEvents) > 1 {
				var dates []string
				for _, ev := range taskEvents {
					st, _ := time.Parse(time.RFC3339, ev.Start)
					if !st.IsZero() {
						dates = append(dates, st.Format("2006-01-02"))
					}
				}
				mismatches = append(mismatches, RecurrenceMismatch{
					TaskID:    task.ID,
					TaskTitle: task.Title,
					Reason:    fmt.Sprintf("每周重复任务应只出现 1 次，实际出现 %d 次", len(taskEvents)),
					Expected:  fmt.Sprintf("应在 %s 出现 1 次", dayNames[expectedWeekday]),
					Actual:    fmt.Sprintf("%d 次", len(taskEvents)),
				})
			} else {
				// Exactly 1 event - check if on correct day
				st, _ := time.Parse(time.RFC3339, taskEvents[0].Start)
				if !st.IsZero() {
					actualWeekday := int(st.Weekday())
					if actualWeekday == 0 {
						actualWeekday = 7
					}
					if actualWeekday != expectedWeekday {
						mismatches = append(mismatches, RecurrenceMismatch{
							TaskID:    task.ID,
							TaskTitle: task.Title,
							Date:      st.Format("2006-01-02"),
							Reason:    fmt.Sprintf("每周重复任务应在%s，实际排在%s", dayNames[expectedWeekday], dayNames[actualWeekday]),
							Expected:  fmt.Sprintf("recurrence_day=%d (%s)", expectedWeekday, dayNames[expectedWeekday]),
							Actual:    fmt.Sprintf("recurrence_day=%d (%s)", actualWeekday, dayNames[actualWeekday]),
						})
					}
				}
			}

		case "daily":
			// Daily tasks should appear on every day in the date range
			if len(allDates) == 0 {
				continue
			}
			eventDates := make(map[string]bool)
			for _, ev := range taskEvents {
				st, _ := time.Parse(time.RFC3339, ev.Start)
				if !st.IsZero() {
					eventDates[st.Format("2006-01-02")] = true
				}
			}
			for dateStr := range allDates {
				if !eventDates[dateStr] {
					mismatches = append(mismatches, RecurrenceMismatch{
						TaskID:    task.ID,
						TaskTitle: task.Title,
						Date:      dateStr,
						Reason:    fmt.Sprintf("每日重复任务缺少 %s 的实例", dateStr),
						Expected:  fmt.Sprintf("应在 %s 出现", dateStr),
						Actual:    "缺失",
					})
				}
			}

		case "monthly":
			expectedDOM := task.RecurrenceDay // 1-31
			if expectedDOM < 1 || expectedDOM > 31 {
				continue
			}
			for _, ev := range taskEvents {
				st, _ := time.Parse(time.RFC3339, ev.Start)
				if st.IsZero() {
					continue
				}
				actualDOM := st.Day()
				if actualDOM != expectedDOM {
					mismatches = append(mismatches, RecurrenceMismatch{
						TaskID:    task.ID,
						TaskTitle: task.Title,
						Date:      st.Format("2006-01-02"),
						Reason:    fmt.Sprintf("每月重复任务应在第 %d 天，实际排在第 %d 天", expectedDOM, actualDOM),
						Expected:  fmt.Sprintf("每月第 %d 天", expectedDOM),
						Actual:    fmt.Sprintf("每月第 %d 天", actualDOM),
					})
				}
			}
		}
	}
	return mismatches
}

func stringsJoin(ss []string, sep string) string {
	if len(ss) == 0 {
		return ""
	}
	result := ss[0]
	for i := 1; i < len(ss) && i < 20; i++ {
		result += sep + ss[i]
	}
	if len(ss) > 20 {
		result += fmt.Sprintf(" ... 共%d个", len(ss))
	}
	return result
}

// deleteEventsByDate removes all schedules for a given date.
func (s *ScheduleService) deleteEventsByDate(date time.Time) error {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)
	_, err := s.scheduleRepo.DeleteTaskSchedulesByDateRange(startOfDay, endOfDay)
	return err
}


// buildWeekSchedulePrompt builds a prompt to generate schedules for a full week.
func buildWeekSchedulePrompt(monday, sunday time.Time) string {
	return buildSkillPrompt("auto-schedule", "生成整周日程", monday, sunday, "")
}

// --- Task filtering ---

// matchesRecurrenceInRange checks if a recurring task has any occurrence within the date range.
func matchesRecurrenceInRange(task model.Task, start, end time.Time) bool {
	current := start
	for !current.After(end) {
		if matchesRecurrence(task, current) {
			return true
		}
		current = current.AddDate(0, 0, 1)
	}
	return false
}

// matchesRecurrence checks if a recurring task applies on a specific date.
func matchesRecurrence(task model.Task, date time.Time) bool {
	if !task.IsRecurring || task.RecurrencePattern == "" {
		return false
	}
	switch task.RecurrencePattern {
	case "daily":
		return true
	case "weekly":
		weekday := int(date.Weekday())
		if weekday == 0 {
			weekday = 7 // Sunday -> 7
		}
		return task.RecurrenceDay == 0 || task.RecurrenceDay == weekday
	case "monthly":
		dom := date.Day()
		return task.RecurrenceDay == 0 || task.RecurrenceDay == dom
	default:
		return true // unknown pattern, assume matches
	}
}

// filterTasksForWeek filters tasks to only those relevant for the given week.
func filterTasksForWeek(allTasks []model.Task, monday, sunday time.Time) []model.Task {
	twoWeeksAgo := monday.AddDate(0, 0, -14)
	twoWeeksAhead := sunday.AddDate(0, 0, 14)

	var candidates []model.Task
	for _, t := range allTasks {
		if t.IsRecurring && matchesRecurrenceInRange(t, monday, sunday) {
			candidates = append(candidates, t)
			continue
		}
		if t.Status == model.StatusCompleted || t.Status == model.StatusCancelled {
			continue
		}
		if t.StartDate != nil {
			if t.StartDate.Before(twoWeeksAgo) || t.StartDate.After(twoWeeksAhead) {
				continue
			}
		}
		if t.DueDate != nil {
			if t.DueDate.Before(twoWeeksAgo) {
				continue
			}
		}
		candidates = append(candidates, t)
	}
	return candidates
}

// sortTasksForScheduling sorts tasks by scheduling priority:
// Q1 (important+urgent) first, then Q2, Q3, Q4.
// Within the same quadrant: sooner deadline first, then longer estimated time first.
func sortTasksForScheduling(tasks []model.Task) {
	sort.SliceStable(tasks, func(i, j int) bool {
		a, b := tasks[i], tasks[j]
		// Sort by quadrant first
		if a.Quadrant != b.Quadrant {
			return a.Quadrant < b.Quadrant
		}
		// Within same quadrant: deadline priority (sooner first, nil deadlines last)
		if a.Deadline != nil && b.Deadline != nil {
			if !a.Deadline.Equal(*b.Deadline) {
				return a.Deadline.Before(*b.Deadline)
			}
		}
		if a.Deadline != nil && b.Deadline == nil {
			return true
		}
		if a.Deadline == nil && b.Deadline != nil {
			return false
		}
		// Longer estimated time first (big rocks first)
		return a.EstimatedTime > b.EstimatedTime
	})
}

// filterTasksForDate filters tasks to only those relevant for a specific date.
// For recurring tasks, checks if the recurrence pattern matches this date.
func filterTasksForDate(tasks []model.Task, date time.Time) []model.Task {
	var result []model.Task
	for _, t := range tasks {
		if t.IsRecurring {
			if matchesRecurrence(t, date) {
				result = append(result, t)
			}
		} else {
			result = append(result, t)
		}
	}
	return result
}

func filterUnscheduled(tasks []model.Task, scheduled map[string]bool) []model.Task {
	var remaining []model.Task
	for _, t := range tasks {
		if !scheduled[t.ID] {
			remaining = append(remaining, t)
		}
	}
	if len(remaining) > 30 {
		remaining = remaining[:30]
	}
	return remaining
}

// --- ICS Generation (Go native, replaces Python script) ---

// generateWeekICS generates an ICS file for a full week of tasks.
