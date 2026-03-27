package model

import "time"

// ScheduleType 日程类型
type ScheduleType string

const (
	ScheduleTypeTask     ScheduleType = "task"     // 任务
	ScheduleTypePomodoro ScheduleType = "pomodoro" // 番茄钟
	ScheduleTypeBreak    ScheduleType = "break"    // 休息
	ScheduleTypeCustom   ScheduleType = "custom"   // 自定义
)

// ScheduleStatus 日程状态
type ScheduleStatus string

const (
	ScheduleStatusPlanned     ScheduleStatus = "planned"     // 计划中
	ScheduleStatusInProgress  ScheduleStatus = "in_progress" // 进行中
	ScheduleStatusCompleted   ScheduleStatus = "completed"   // 已完成
	ScheduleStatusCancelled   ScheduleStatus = "cancelled"   // 已取消
)

// Schedule 任务日程安排
type Schedule struct {
	ID          string        `gorm:"primaryKey;size:36" json:"id"`
	TaskID      *string       `gorm:"size:36" json:"task_id"`
	Title       string        `gorm:"not null;size:200" json:"title"`
	Description string        `gorm:"size:500" json:"description"`
	StartTime   time.Time     `gorm:"not null" json:"start_time"`
	EndTime     time.Time     `gorm:"not null" json:"end_time"`
	Type        ScheduleType  `gorm:"not null;size:20;default:'task'" json:"type"`
	Status      ScheduleStatus `gorm:"not null;size:20;default:'planned'" json:"status"`
	Color       string        `gorm:"size:20" json:"color"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

// TableName 指定表名
func (Schedule) TableName() string {
	return "schedules"
}