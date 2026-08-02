package service

import (
	"math"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"time"
)

type AnalyticsService struct {
	analyticsRepo repository.AnalyticsRepository
	taskRepo      repository.TaskRepository
	sessionRepo   repository.SessionRepository
	settingRepo   repository.SettingRepository
}

func NewAnalyticsService(
	analyticsRepo repository.AnalyticsRepository,
	taskRepo repository.TaskRepository,
	sessionRepo repository.SessionRepository,
	settingRepo repository.SettingRepository,
) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo: analyticsRepo,
		taskRepo:      taskRepo,
		sessionRepo:   sessionRepo,
		settingRepo:   settingRepo,
	}
}

type DailySummary struct {
	CompletedPomodoros int `json:"completed_pomodoros"`
	TotalFocusTime     int `json:"total_focus_time"`
	CompletedTasks     int `json:"completed_tasks"`
	CreatedTasks       int `json:"created_tasks"`
}

type TrendDataPoint struct {
	Date      string `json:"date"`
	FocusTime int    `json:"focus_time"`
	Pomodoros int    `json:"pomodoros"`
}

type TrendData struct {
	Data []TrendDataPoint `json:"data"`
}

type QuadrantStats struct {
	Total     int `json:"total"`
	Completed int `json:"completed"`
}

type DistributionStats struct {
	QuadrantStats map[int]QuadrantStats `json:"quadrant_stats"`
	TaskStats     struct {
		Total          int     `json:"total"`
		Completed      int     `json:"completed"`
		CompletionRate float64 `json:"completion_rate"`
	} `json:"task_stats"`
}

// GetSummary 获取指定日期的概览统计
func (s *AnalyticsService) GetSummary(date time.Time) (*DailySummary, error) {
	stats, err := s.analyticsRepo.GetDailyStats(date)
	if err != nil {
		// 如果没有数据，返回零值
		return &DailySummary{}, nil
	}

	return &DailySummary{
		CompletedPomodoros: stats.CompletedPomodoros,
		TotalFocusTime:     stats.TotalFocusTime,
		CompletedTasks:     stats.CompletedTasks,
		CreatedTasks:       stats.CreatedTasks,
	}, nil
}

// GetTrend 获取趋势数据
func (s *AnalyticsService) GetTrend(days int) (*TrendData, error) {
	end := time.Now().Truncate(24 * time.Hour)
	start := end.AddDate(0, 0, -days+1)

	stats, err := s.analyticsRepo.GetDailyStatsRange(start, end.Add(24*time.Hour))
	if err != nil {
		return nil, err
	}

	// 创建日期映射
	statsMap := make(map[string]*model.DailyStats)
	for i := range stats {
		dateStr := stats[i].Date.Format("2006-01-02")
		statsMap[dateStr] = &stats[i]
	}

	// 生成完整的日期序列
	data := make([]TrendDataPoint, 0, days)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i)
		dateStr := date.Format("2006-01-02")

		point := TrendDataPoint{
			Date:      dateStr,
			FocusTime: 0,
			Pomodoros: 0,
		}

		if stat, exists := statsMap[dateStr]; exists {
			point.FocusTime = stat.TotalFocusTime
			point.Pomodoros = stat.CompletedPomodoros
		}

		data = append(data, point)
	}

	return &TrendData{Data: data}, nil
}

// GetDistribution 获取任务分布统计
func (s *AnalyticsService) GetDistribution(start, end time.Time) (*DistributionStats, error) {
	// 获取所有任务
	tasks, err := s.taskRepo.GetAll()
	if err != nil {
		return nil, err
	}

	result := &DistributionStats{
		QuadrantStats: make(map[int]QuadrantStats),
	}

	// 初始化象限统计
	for q := 1; q <= 4; q++ {
		result.QuadrantStats[q] = QuadrantStats{}
	}

	// 统计各象限任务
	for _, task := range tasks {
		// 只统计时间范围内的任务
		if task.CreatedAt.Before(start) || task.CreatedAt.After(end) {
			continue
		}

		stats := result.QuadrantStats[int(task.Quadrant)]
		stats.Total++
		if task.Status == model.StatusCompleted {
			stats.Completed++
		}
		result.QuadrantStats[int(task.Quadrant)] = stats

		result.TaskStats.Total++
		if task.Status == model.StatusCompleted {
			result.TaskStats.Completed++
		}
	}

	// 计算完成率
	if result.TaskStats.Total > 0 {
		result.TaskStats.CompletionRate = float64(result.TaskStats.Completed) / float64(result.TaskStats.Total)
	}

	return result, nil
}

// PomodoroByTaskItem represents one task in the pomodoro ranking.
type PomodoroByTaskItem struct {
	TaskID             string `json:"task_id"`
	TaskTitle          string `json:"task_title"`
	PlannedPomodoros   int    `json:"planned_pomodoros"`
	CompletedPomodoros int    `json:"completed_pomodoros"`
	TotalFocusMinutes  int    `json:"total_focus_minutes"`
	Status             string `json:"status"`
}

// PomodoroByTaskResult wraps the ranking list.
type PomodoroByTaskResult struct {
	Tasks []PomodoroByTaskItem `json:"tasks"`
}

// PomodoroTrendDay represents one day in the planned vs actual trend.
type PomodoroTrendDay struct {
	Date           string `json:"date"`
	Planned        int    `json:"planned"`
	Actual         int    `json:"actual"`
	CompletedTasks int    `json:"completed_tasks"`
	ExceededTasks  int    `json:"exceeded_tasks"`
}

// PomodoroTrendsResult wraps the trend day list.
type PomodoroTrendsResult struct {
	Days []PomodoroTrendDay `json:"days"`
}

// GetPomodoroByTask returns per-task pomodoro statistics ranked by completed count.
func (s *AnalyticsService) GetPomodoroByTask(period string) (*PomodoroByTaskResult, error) {
	start, end := s.periodToRange(period)

	sessions, err := s.sessionRepo.GetCompletedWorkByDateRange(start, end)
	if err != nil {
		return nil, err
	}

	// Build taskID -> {count, focusMinutes} map
	type taskAgg struct {
		count        int
		focusMinutes int
	}
	aggMap := make(map[string]*taskAgg)
	for _, sess := range sessions {
		if sess.TaskID == nil {
			continue
		}
		id := *sess.TaskID
		if aggMap[id] == nil {
			aggMap[id] = &taskAgg{}
		}
		aggMap[id].count++
		if sess.ActualDuration != nil {
			aggMap[id].focusMinutes += *sess.ActualDuration / 60
		}
	}

	workDuration, _ := s.getWorkDurationMinutes()
	if workDuration == 0 {
		workDuration = 25
	}

	tasks, err := s.taskRepo.GetAll()
	if err != nil {
		return nil, err
	}

	items := make([]PomodoroByTaskItem, 0)
	for _, t := range tasks {
		agg := aggMap[t.ID]
		planned := 0
		if t.EstimatedTime > 0 {
			planned = int(math.Ceil(float64(t.EstimatedTime) / float64(workDuration)))
		}
		completed := 0
		focusMin := 0
		if agg != nil {
			completed = agg.count
			focusMin = agg.focusMinutes
		}
		// Include tasks that have sessions or have an estimate
		if completed > 0 || planned > 0 {
			items = append(items, PomodoroByTaskItem{
				TaskID:             t.ID,
				TaskTitle:          t.Title,
				PlannedPomodoros:   planned,
				CompletedPomodoros: completed,
				TotalFocusMinutes:  focusMin,
				Status:             computePomodoroStatus(planned, completed),
			})
		}
	}

	// Sort descending by CompletedPomodoros
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			if items[j].CompletedPomodoros > items[i].CompletedPomodoros {
				items[i], items[j] = items[j], items[i]
			}
		}
	}

	return &PomodoroByTaskResult{Tasks: items}, nil
}

// GetPomodoroTrends returns daily planned vs actual pomodoro comparison.
func (s *AnalyticsService) GetPomodoroTrends(period string) (*PomodoroTrendsResult, error) {
	start, end := s.periodToRange(period)

	sessions, err := s.sessionRepo.GetCompletedWorkByDateRange(start, end)
	if err != nil {
		return nil, err
	}

	// Group sessions by day: date -> actual count
	actualByDay := make(map[string]int)
	for _, sess := range sessions {
		dayStr := sess.StartTime.Format("2006-01-02")
		actualByDay[dayStr]++
	}

	// Get all completed work sessions (no date filter) to compute per-task cumulative status
	allSessions, _ := s.sessionRepo.GetCompletedWorkByDateRange(time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), end)
	completedByTask := make(map[string]int)
	for _, sess := range allSessions {
		if sess.TaskID != nil {
			completedByTask[*sess.TaskID]++
		}
	}

	workDuration, _ := s.getWorkDurationMinutes()
	if workDuration == 0 {
		workDuration = 25
	}

	tasks, err := s.taskRepo.GetAll()
	if err != nil {
		return nil, err
	}

	// Build task info
	type taskInfo struct {
		planned  int
		status   string
		created  string // date string
	}
	taskMap := make(map[string]taskInfo)
	for _, t := range tasks {
		planned := 0
		if t.EstimatedTime > 0 {
			planned = int(math.Ceil(float64(t.EstimatedTime) / float64(workDuration)))
		}
		completed := completedByTask[t.ID]
		taskMap[t.ID] = taskInfo{
			planned: planned,
			status:  computePomodoroStatus(planned, completed),
			created: t.CreatedAt.Format("2006-01-02"),
		}
	}

	// Generate daily trend
	days := int(end.Sub(start).Hours()/24) + 1
	result := make([]PomodoroTrendDay, 0, days)
	for i := 0; i < days; i++ {
		date := start.AddDate(0, 0, i)
		dateStr := date.Format("2006-01-02")

		planned := 0
		completedTasks := 0
		exceededTasks := 0
		for _, info := range taskMap {
			if info.created <= dateStr && info.planned > 0 {
				planned += info.planned
			}
			if info.status == "completed" {
				completedTasks++
			}
			if info.status == "exceeded" {
				exceededTasks++
			}
		}

		result = append(result, PomodoroTrendDay{
			Date:           dateStr,
			Planned:        planned,
			Actual:         actualByDay[dateStr],
			CompletedTasks: completedTasks,
			ExceededTasks:  exceededTasks,
		})
	}

	return &PomodoroTrendsResult{Days: result}, nil
}

func (s *AnalyticsService) periodToRange(period string) (start, end time.Time) {
	now := time.Now()
	end = time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 0, now.Location()).Add(1 * time.Second)
	if period == "month" {
		start = end.AddDate(0, 0, -30)
	} else {
		start = end.AddDate(0, 0, -7)
	}
	return
}

func (s *AnalyticsService) getWorkDurationMinutes() (int, error) {
	settings, err := s.settingRepo.GetPomodoroSettings()
	if err != nil {
		return 0, err
	}
	return settings.WorkDuration / 60, nil
}