package model

import "time"

type Quadrant int

const (
	Quadrant1 Quadrant = 1 // 重要且紧急
	Quadrant2 Quadrant = 2 // 重要不紧急
	Quadrant3 Quadrant = 3 // 紧急不重要
	Quadrant4 Quadrant = 4 // 不重要不紧急
)

type TaskStatus string

const (
	StatusTodo        TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusCompleted   TaskStatus = "completed"
	StatusCancelled   TaskStatus = "cancelled"
)

type Task struct {
	ID            string     `gorm:"primaryKey;size:36" json:"id"`
	Title         string     `gorm:"not null;size:200" json:"title"`
	Description   string     `gorm:"size:1000" json:"description"`
	Quadrant      Quadrant   `gorm:"not null" json:"quadrant"`
	IsImportant   bool       `gorm:"not null;default:false" json:"is_important"`
	IsUrgent      bool       `gorm:"not null;default:false" json:"is_urgent"`
	Status        TaskStatus `gorm:"not null;default:todo" json:"status"`
	EstimatedTime int        `gorm:"default:0" json:"estimated_time"` // 分钟
	Deadline      *time.Time `json:"deadline"`
	Tags          string     `gorm:"size:500" json:"tags"` // JSON 数组字符串
	Order         int        `gorm:"default:0" json:"order"` // 同象限内排序
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	CompletedAt   *time.Time `json:"completed_at"`
}

// TableName 指定表名
func (Task) TableName() string {
	return "tasks"
}
