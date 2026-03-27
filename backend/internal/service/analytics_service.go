package service

import (
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"time"
)

type AnalyticsService struct {
	analyticsRepo repository.AnalyticsRepository
	taskRepo      repository.TaskRepository
	sessionRepo   repository.SessionRepository
}

func NewAnalyticsService(
	analyticsRepo repository.AnalyticsRepository,
	taskRepo repository.TaskRepository,
	sessionRepo repository.SessionRepository,
) *AnalyticsService {
	return &AnalyticsService{
		analyticsRepo: analyticsRepo,
		taskRepo:      taskRepo,
		sessionRepo:   sessionRepo,
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