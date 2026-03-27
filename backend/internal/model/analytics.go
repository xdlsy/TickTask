package model

import "time"

type DailyStats struct {
	Date                time.Time `gorm:"primaryKey" json:"date"`
	CompletedPomodoros  int       `gorm:"default:0" json:"completed_pomodoros"`
	TotalFocusTime      int       `gorm:"default:0" json:"total_focus_time"`   // 秒
	CompletedTasks      int       `gorm:"default:0" json:"completed_tasks"`
	CreatedTasks        int       `gorm:"default:0" json:"created_tasks"`
}

// TableName 指定表名
func (DailyStats) TableName() string {
	return "daily_stats"
}
