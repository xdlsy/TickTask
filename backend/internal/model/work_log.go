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

// WorkItem 日报里的一条核心工作，四维独立字段
type WorkItem struct {
	ID            string `gorm:"primaryKey;size:36" json:"id"`
	WorkLogID     string `gorm:"index;size:36;not null" json:"work_log_id"`
	Seq           int    `gorm:"not null" json:"seq"` // 顺序，从 1 开始
	Title         string `gorm:"size:200" json:"title"`
	Content       string `gorm:"size:1000" json:"content"`       // 做了什么
	ProblemSolved string `gorm:"size:1000" json:"problem_solved"` // 解决了什么问题
	Result        string `gorm:"size:1000" json:"result"`         // 已产生的结果，缺则"（待补充）"
	Impact        string `gorm:"size:1000" json:"impact"`         // 对后续的影响
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
