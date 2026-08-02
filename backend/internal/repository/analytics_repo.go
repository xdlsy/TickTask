package repository

import (
	"ticktask/internal/model"
	"time"

	"gorm.io/gorm"
)

type AnalyticsRepository interface {
	GetDailyStats(date time.Time) (*model.DailyStats, error)
	GetDailyStatsRange(start, end time.Time) ([]model.DailyStats, error)
	CreateDailyStats(stats *model.DailyStats) error
	UpdateDailyStats(stats *model.DailyStats) error
	IncrementCompletedPomodoros(date time.Time) error
	IncrementFocusTime(date time.Time, seconds int) error
	IncrementCompletedTasks(date time.Time) error
	IncrementCreatedTasks(date time.Time) error
}

type analyticsRepository struct {
	db *gorm.DB
}

func NewAnalyticsRepository(db *gorm.DB) AnalyticsRepository {
	return &analyticsRepository{db: db}
}

func (r *analyticsRepository) GetDailyStats(date time.Time) (*model.DailyStats, error) {
	var stats model.DailyStats
	err := r.db.Where("date = ?", date).First(&stats).Error
	if err != nil {
		return nil, err
	}
	return &stats, nil
}

func (r *analyticsRepository) GetDailyStatsRange(start, end time.Time) ([]model.DailyStats, error) {
	var stats []model.DailyStats
	err := r.db.Where("date >= ? AND date < ?", start, end).
		Order("date ASC").
		Find(&stats).Error
	return stats, err
}

func (r *analyticsRepository) CreateDailyStats(stats *model.DailyStats) error {
	return r.db.Create(stats).Error
}

func (r *analyticsRepository) UpdateDailyStats(stats *model.DailyStats) error {
	return r.db.Save(stats).Error
}

func (r *analyticsRepository) IncrementCompletedPomodoros(date time.Time) error {
	return r.db.Model(&model.DailyStats{}).
		Where("date = ?", date).
		UpdateColumn("completed_pomodoros", gorm.Expr("completed_pomodoros + 1")).Error
}

func (r *analyticsRepository) IncrementFocusTime(date time.Time, seconds int) error {
	return r.db.Model(&model.DailyStats{}).
		Where("date = ?", date).
		UpdateColumn("total_focus_time", gorm.Expr("total_focus_time + ?", seconds)).Error
}

func (r *analyticsRepository) IncrementCompletedTasks(date time.Time) error {
	return r.db.Model(&model.DailyStats{}).
		Where("date = ?", date).
		UpdateColumn("completed_tasks", gorm.Expr("completed_tasks + 1")).Error
}

func (r *analyticsRepository) IncrementCreatedTasks(date time.Time) error {
	return r.db.Model(&model.DailyStats{}).
		Where("date = ?", date).
		UpdateColumn("created_tasks", gorm.Expr("created_tasks + 1")).Error
}
