package model

import "time"

type SessionType string

const (
	SessionWork      SessionType = "work"
	SessionShortBreak SessionType = "short_break"
	SessionLongBreak SessionType = "long_break"
)

type SessionStatus string

const (
	SessionPending    SessionStatus = "pending"
	SessionRunning    SessionStatus = "running"
	SessionPaused     SessionStatus = "paused"
	SessionCompleted  SessionStatus = "completed"
	SessionAbandoned SessionStatus = "abandoned"
)

type PomodoroSession struct {
	ID              string        `gorm:"primaryKey;size:36" json:"id"`
	TaskID          *string       `gorm:"size:36" json:"task_id"`
	Type            SessionType   `gorm:"not null" json:"type"`
	Status          SessionStatus `gorm:"not null;default:pending" json:"status"`
	StartTime       time.Time     `json:"start_time"`
	EndTime         *time.Time    `json:"end_time"`
	PlannedDuration int           `gorm:"not null" json:"planned_duration"` // 秒
	ActualDuration  *int          `json:"actual_duration"`                  // 秒
	Interruptions   int           `gorm:"default:0" json:"interruptions"`
	InterruptReason *string       `gorm:"size:50" json:"interrupt_reason"` // meeting/call/urgent/other
	CreatedAt       time.Time     `json:"created_at"`
}

// TableName 指定表名
func (PomodoroSession) TableName() string {
	return "sessions"
}

// Task 关联任务
func (s *PomodoroSession) Task() *Task {
	return &Task{ID: *s.TaskID}
}
