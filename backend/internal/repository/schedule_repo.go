package repository

import (
	"ticktask/internal/model"
	"time"

	"gorm.io/gorm"
)

type ScheduleRepository interface {
	Create(schedule *model.Schedule) error
	GetByID(id string) (*model.Schedule, error)
	GetByTimeRange(start, end time.Time) ([]model.Schedule, error)
	GetByTaskID(taskID string) ([]model.Schedule, error)
	GetByDate(date time.Time) ([]model.Schedule, error)
	Update(schedule *model.Schedule) error
	Delete(id string) error
	UpdateStatus(id string, status model.ScheduleStatus) error
	Move(id string, startTime, endTime time.Time) error
	DeleteTaskSchedulesByDateRange(start, end time.Time) (int64, error)
}

type scheduleRepository struct {
	db *gorm.DB
}

func NewScheduleRepository(db *gorm.DB) ScheduleRepository {
	return &scheduleRepository{db: db}
}

func (r *scheduleRepository) Create(schedule *model.Schedule) error {
	return r.db.Create(schedule).Error
}

func (r *scheduleRepository) GetByID(id string) (*model.Schedule, error) {
	var schedule model.Schedule
	err := r.db.Where("id = ?", id).First(&schedule).Error
	if err != nil {
		return nil, err
	}
	return &schedule, nil
}

func (r *scheduleRepository) GetByTimeRange(start, end time.Time) ([]model.Schedule, error) {
	var schedules []model.Schedule
	err := r.db.Where("start_time >= ? AND end_time <= ?", start, end).
		Or("start_time < ? AND end_time > ?", start, start).
		Or("start_time < ? AND end_time > ?", end, end).
		Order("start_time ASC").
		Find(&schedules).Error
	return schedules, err
}

func (r *scheduleRepository) GetByTaskID(taskID string) ([]model.Schedule, error) {
	var schedules []model.Schedule
	err := r.db.Where("task_id = ?", taskID).
		Order("start_time ASC").
		Find(&schedules).Error
	return schedules, err
}

func (r *scheduleRepository) GetByDate(date time.Time) ([]model.Schedule, error) {
	startOfDay := date.Truncate(24 * time.Hour)
	endOfDay := startOfDay.Add(24 * time.Hour)
	return r.GetByTimeRange(startOfDay, endOfDay)
}

func (r *scheduleRepository) Update(schedule *model.Schedule) error {
	return r.db.Save(schedule).Error
}

func (r *scheduleRepository) Delete(id string) error {
	return r.db.Delete(&model.Schedule{}, "id = ?", id).Error
}

func (r *scheduleRepository) UpdateStatus(id string, status model.ScheduleStatus) error {
	return r.db.Model(&model.Schedule{}).Where("id = ?", id).Update("status", status).Error
}

func (r *scheduleRepository) Move(id string, startTime, endTime time.Time) error {
	return r.db.Model(&model.Schedule{}).Where("id = ?", id).Updates(map[string]interface{}{
		"start_time": startTime,
		"end_time":   endTime,
	}).Error
}

func (r *scheduleRepository) DeleteTaskSchedulesByDateRange(start, end time.Time) (int64, error) {
	result := r.db.Where("task_id IS NOT NULL AND task_id != '' AND start_time >= ? AND start_time < ?", start, end).
		Delete(&model.Schedule{})
	return result.RowsAffected, result.Error
}