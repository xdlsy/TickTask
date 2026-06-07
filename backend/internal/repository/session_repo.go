package repository

import (
	"ticktask/internal/model"
	"time"

	"gorm.io/gorm"
)

type SessionRepository interface {
	Create(session *model.PomodoroSession) error
	GetByID(id string) (*model.PomodoroSession, error)
	GetActive() (*model.PomodoroSession, error)
	Update(session *model.PomodoroSession) error
	GetByDate(date time.Time) ([]model.PomodoroSession, error)
	GetRecent(limit int) ([]model.PomodoroSession, error)
	CountByTaskID(taskID string, sessionType model.SessionType, status model.SessionStatus) (int, error)
	GetCompletedWorkByDateRange(start, end time.Time) ([]model.PomodoroSession, error)
}

type sessionRepository struct {
	db *gorm.DB
}

func NewSessionRepository(db *gorm.DB) SessionRepository {
	return &sessionRepository{db: db}
}

func (r *sessionRepository) Create(session *model.PomodoroSession) error {
	return r.db.Create(session).Error
}

func (r *sessionRepository) GetByID(id string) (*model.PomodoroSession, error) {
	var session model.PomodoroSession
	err := r.db.Where("id = ?", id).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepository) GetActive() (*model.PomodoroSession, error) {
	var session model.PomodoroSession
	err := r.db.Where("status IN ?", []model.SessionStatus{
		model.SessionRunning,
		model.SessionPaused,
	}).Order("created_at DESC").First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

func (r *sessionRepository) Update(session *model.PomodoroSession) error {
	return r.db.Save(session).Error
}

func (r *sessionRepository) GetByDate(date time.Time) ([]model.PomodoroSession, error) {
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var sessions []model.PomodoroSession
	err := r.db.Where("start_time >= ? AND start_time < ?", startOfDay, endOfDay).
		Order("start_time DESC").
		Find(&sessions).Error
	return sessions, err
}

func (r *sessionRepository) GetRecent(limit int) ([]model.PomodoroSession, error) {
	var sessions []model.PomodoroSession
	err := r.db.Order("start_time DESC").Limit(limit).Find(&sessions).Error
	return sessions, err
}

func (r *sessionRepository) CountByTaskID(taskID string, sessionType model.SessionType, status model.SessionStatus) (int, error) {
	var count int64
	err := r.db.Model(&model.PomodoroSession{}).
		Where("task_id = ? AND type = ? AND status = ?", taskID, sessionType, status).
		Count(&count).Error
	return int(count), err
}

func (r *sessionRepository) GetCompletedWorkByDateRange(start, end time.Time) ([]model.PomodoroSession, error) {
	var sessions []model.PomodoroSession
	err := r.db.Where("type = ? AND status = ? AND start_time >= ? AND start_time < ?",
		model.SessionWork, model.SessionCompleted, start, end).
		Order("start_time ASC").
		Find(&sessions).Error
	return sessions, err
}
