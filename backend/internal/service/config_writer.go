package service

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"ticktask/internal/model"
	"time"
)

// configPath returns the project root's config/ directory.
func configPath() string {
	return filepath.Join(findProjectRoot(), "config")
}

// TodoTask is a task entry in todo.json format for the scheduling script.
type TodoTask struct {
	Title              string `json:"title"`
	Description        string `json:"description"`
	EstimatedMinutes   int    `json:"estimated_minutes"`
	Priority           string `json:"priority"`
	Type               string `json:"type"`
	PreferredStartTime string `json:"preferred_start_time"`
	PreferredEndTime   string `json:"preferred_end_time"`
	IsRecurring        bool   `json:"is_recurring"`
	RecurrencePattern  string `json:"recurrence_pattern"`
	RecurrenceDay      int    `json:"recurrence_day"`
	FixedTime          any    `json:"fixed_time"`
	Deadline           any    `json:"deadline"`
}

// TodoFile is the structure of todo.json.
type TodoFile struct {
	Date  string     `json:"date"`
	Tasks []TodoTask `json:"tasks"`
}

// WriteTodoJSON writes eligible tasks to config/todo.json.
func WriteTodoJSON(tasks []model.Task, date time.Time) error {
	if err := os.MkdirAll(configPath(), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	capacity := len(tasks)
	if capacity < 0 {
		capacity = 0
	}
	todo := TodoFile{
		Date:  date.Format("2006-01-02"),
		Tasks: make([]TodoTask, 0, capacity),
	}

	for _, t := range tasks {
		todo.Tasks = append(todo.Tasks, TodoTask{
			Title:              t.Title,
			Description:        t.Description,
			EstimatedMinutes:   t.EstimatedTime,
			Priority:           mapPriority(t.Quadrant),
			Type:               mapTaskType(t),
			PreferredStartTime: t.PreferredStartTime,
			PreferredEndTime:   t.PreferredEndTime,
			IsRecurring:        t.IsRecurring,
			RecurrencePattern:  t.RecurrencePattern,
			RecurrenceDay:      t.RecurrenceDay,
			FixedTime:          nil,
			Deadline:           formatDeadline(t.DueDate),
		})
	}

	data, err := json.MarshalIndent(todo, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal todo.json: %w", err)
	}

	path := filepath.Join(configPath(), "todo.json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write todo.json: %w", err)
	}
	return nil
}

// WriteHabitMD writes pomodoro/scheduling settings to config/habit.md.
func WriteHabitMD(settings *model.PomodoroSettings, workStart, workEnd string) error {
	if err := os.MkdirAll(configPath(), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	if workStart == "" {
		workStart = "09:00"
	}
	if workEnd == "" {
		workEnd = "18:00"
	}

	prefs := parseTaskTimePreferences(settings.TaskTimePreferences)

	var sb strings.Builder
	sb.WriteString("# 用户习惯配置\n\n")
	sb.WriteString("> 请根据你的实际情况修改以下配置项。格式为 `key: value`，不要修改 key 名称。\n\n")
	sb.WriteString("## 工作时间\n")
	sb.WriteString(fmt.Sprintf("work_start: %s\n", workStart))
	sb.WriteString(fmt.Sprintf("work_end: %s\n", workEnd))
	sb.WriteString("\n## 专注节奏（番茄钟）\n")
	sb.WriteString(fmt.Sprintf("focus_work_minutes: %d\n", settings.WorkDuration/60))
	sb.WriteString(fmt.Sprintf("focus_break_minutes: %d\n", settings.ShortBreakDuration/60))
	sb.WriteString("\n## 精力曲线\n")
	sb.WriteString("energy_morning: high\n")
	sb.WriteString("energy_early_afternoon: medium\n")
	sb.WriteString("energy_late_afternoon: low\n")
	sb.WriteString("\n## 任务类型偏好\n")
	sb.WriteString(fmt.Sprintf("pref_deep_work: %s\n", prefStr(prefs, "deep_work", "morning")))
	sb.WriteString(fmt.Sprintf("pref_shallow_work: %s\n", prefStr(prefs, "shallow_work", "late_afternoon")))
	sb.WriteString(fmt.Sprintf("pref_meeting: %s\n", prefStr(prefs, "meeting", "early_afternoon")))
	sb.WriteString(fmt.Sprintf("pref_errand: %s\n", prefStr(prefs, "errand", "late_afternoon")))
	sb.WriteString("\n## 固定休息\n")
	sb.WriteString("lunch_start: 12:00\n")
	sb.WriteString("lunch_end: 13:00\n")
	sb.WriteString("\n## 最长连续专注\n")
	longest := settings.WorkDuration * settings.LongBreakAfter / 60
	if longest < 50 {
		longest = 120
	}
	sb.WriteString(fmt.Sprintf("longest_focus_minutes: %d\n", longest))

	path := filepath.Join(configPath(), "habit.md")
	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write habit.md: %w", err)
	}
	return nil
}

// ReadScheduleICS reads the generated config/schedule.ics file.
func ReadScheduleICS() (string, error) {
	path := filepath.Join(configPath(), "schedule.ics")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read schedule.ics: %w", err)
	}
	return string(data), nil
}

// WriteScheduleICS serializes schedule events to ICS format and writes to config/schedule.ics.
// This serves as the baseline for revision operations.
func WriteScheduleICS(events []ScheduleEvent) error {
	if err := os.MkdirAll(configPath(), 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("BEGIN:VCALENDAR\r\n")
	sb.WriteString("VERSION:2.0\r\n")
	sb.WriteString("PRODID:-//TickTask//EN\r\n")
	sb.WriteString("CALSCALE:GREGORIAN\r\n")
	sb.WriteString("METHOD:PUBLISH\r\n")

	for _, ev := range events {
		// Parse RFC3339 times to ICS local-time format (YYYYMMDDTHHMMSS)
		startTime, err := time.Parse(time.RFC3339, ev.Start)
		if err != nil {
			continue
		}
		endTime, err := time.Parse(time.RFC3339, ev.End)
		if err != nil {
			continue
		}

		icsStart := startTime.Format("20060102T150405")
		icsEnd := endTime.Format("20060102T150405")

		desc := ev.Title
		if ev.Type != "" {
			desc = fmt.Sprintf("%s | %s", ev.Title, ev.Type)
		}

		sb.WriteString("BEGIN:VEVENT\r\n")
		sb.WriteString(fmt.Sprintf("DTSTART:%s\r\n", icsStart))
		sb.WriteString(fmt.Sprintf("DTEND:%s\r\n", icsEnd))
		sb.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICS(ev.Title)))
		sb.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICS(desc)))
		sb.WriteString("END:VEVENT\r\n")
	}

	sb.WriteString("END:VCALENDAR\r\n")

	path := filepath.Join(configPath(), "schedule.ics")
	if err := os.WriteFile(path, []byte(sb.String()), 0644); err != nil {
		return fmt.Errorf("write schedule.ics: %w", err)
	}
	return nil
}

// escapeICS escapes special characters for ICS text fields.
func escapeICS(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, ";", "\\;")
	s = strings.ReplaceAll(s, ",", "\\,")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

func mapPriority(q model.Quadrant) string {
	switch q {
	case model.Quadrant1:
		return "high"
	case model.Quadrant2:
		return "high"
	case model.Quadrant3:
		return "medium"
	default:
		return "low"
	}
}

func mapTaskType(t model.Task) string {
	tags := strings.ToLower(t.Tags)
	if strings.Contains(tags, "meeting") || strings.Contains(tags, "会议") {
		return "meeting"
	}
	if strings.Contains(tags, "errand") || strings.Contains(tags, "杂务") || strings.Contains(tags, "快递") {
		return "errand"
	}
	switch t.Quadrant {
	case model.Quadrant1, model.Quadrant2:
		if t.EstimatedTime >= 60 {
			return "deep_work"
		}
		return "shallow_work"
	case model.Quadrant3:
		return "shallow_work"
	default:
		return "errand"
	}
}

func formatDeadline(dueDate *time.Time) any {
	if dueDate == nil {
		return nil
	}
	return dueDate.Format("2006-01-02T15:04:05")
}

func parseTaskTimePreferences(jsonStr string) map[string]string {
	result := map[string]string{}
	if jsonStr == "" {
		return result
	}
	var raw map[string]string
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return result
	}
	for category, period := range raw {
		category = strings.ToLower(category)
		period = strings.ToLower(period)
		switch {
		case strings.Contains(category, "deep") || strings.Contains(category, "dev") || strings.Contains(category, "开发"):
			result["deep_work"] = mapPeriod(period)
		case strings.Contains(category, "shallow") || strings.Contains(category, "admin") || strings.Contains(category, "管理"):
			result["shallow_work"] = mapPeriod(period)
		case strings.Contains(category, "meeting") || strings.Contains(category, "会议"):
			result["meeting"] = mapPeriod(period)
		default:
			result["shallow_work"] = mapPeriod(period)
		}
	}
	return result
}

func mapPeriod(p string) string {
	switch p {
	case "morning", "上午", "早上":
		return "morning"
	case "early_afternoon", "下午早期":
		return "early_afternoon"
	case "late_afternoon", "下午晚期", "afternoon", "下午":
		return "late_afternoon"
	default:
		return "morning"
	}
}

func prefStr(prefs map[string]string, key, fallback string) string {
	if v, ok := prefs[key]; ok {
		return v
	}
	return fallback
}
