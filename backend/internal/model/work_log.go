// backend/internal/model/work_log.go
package model

import "time"

// WorkLog 一天一条工作日报
type WorkLog struct {
	ID           string     `gorm:"primaryKey;size:36" json:"id"`
	Date         string     `gorm:"uniqueIndex;size:10;not null" json:"date"` // YYYY-MM-DD
	Summary      string     `gorm:"size:500" json:"summary"`                  // 今日小结
	RawBrainDump string     `gorm:"type:text" json:"raw_brain_dump"`          // 用户原始脑暴
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	Items []WorkItem `gorm:"foreignKey:WorkLogID;references:ID" json:"items"`
}

func (WorkLog) TableName() string { return "work_logs" }

// WorkItem 日报里的一条工作。两类来源：
//   - source='manual'：快捷录入（activity/start_time/end_time/quadrant 必填，叙事字段为空）
//   - source='ai'：AI 拆条（4 维叙事字段必填，新字段为 nil）
type WorkItem struct {
	ID            string  `gorm:"primaryKey;size:36" json:"id"`
	WorkLogID     string  `gorm:"index;size:36;not null" json:"work_log_id"`
	Seq           int     `gorm:"not null" json:"seq"`
	Title         string  `gorm:"size:200" json:"title"`
	Content       string  `gorm:"size:1000" json:"content"`
	ProblemSolved string  `gorm:"size:1000" json:"problem_solved"`
	Result        string  `gorm:"size:1000" json:"result"`
	Impact        string  `gorm:"size:1000" json:"impact"`

	// 快捷录入字段（manual 必填，ai 为 nil）
	Activity  *string `gorm:"column:activity;size:200" json:"activity,omitempty"`
	StartTime *string `gorm:"column:start_time;size:5" json:"start_time,omitempty"` // "HH:MM"
	EndTime   *string `gorm:"column:end_time;size:5" json:"end_time,omitempty"`     // "HH:MM"
	Quadrant  *int    `gorm:"column:quadrant" json:"quadrant,omitempty"`            // 1-4
	Source    string  `gorm:"column:source;size:10;default:'ai'" json:"source"`
}

func (WorkItem) TableName() string { return "work_items" }

// WorkReportType 周期报告类型
type WorkReportType string

const (
	ReportWeekly   WorkReportType = "weekly"
	ReportMonthly  WorkReportType = "monthly"
	ReportHalfYear WorkReportType = "halfyear"
	ReportYearly   WorkReportType = "yearly"
)

// WorkReport 周期报告，按 (type, period_key) 唯一
type WorkReport struct {
	ID          string         `gorm:"primaryKey;size:36" json:"id"`
	Type        WorkReportType `gorm:"uniqueIndex:idx_report_period,priority:1;size:20;not null" json:"type"`
	PeriodKey   string         `gorm:"uniqueIndex:idx_report_period,priority:2;size:20;not null" json:"period_key"` // 2026-W31 / 2026-07 / 2026-H1 / 2026
	StartDate   string         `gorm:"size:10;not null" json:"start_date"`
	EndDate     string         `gorm:"size:10;not null" json:"end_date"`
	SummaryJSON string         `gorm:"type:text" json:"summary_json"` // 结构化字段，按 type 不同 schema
	MissingDays string         `gorm:"size:200" json:"missing_days"`   // 点名缺失天，逗号分隔
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

func (WorkReport) TableName() string { return "work_reports" }
