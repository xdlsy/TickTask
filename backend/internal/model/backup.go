package model

import "time"

// BackupSchemaVersion 当前备份文件结构版本。导入时若不一致 → 告警(本期不做自动迁移)。
const BackupSchemaVersion = 1

// SettingsBundle 组装态设置(对齐 GetSettings 与前端)。
type SettingsBundle struct {
	Pomodoro *PomodoroSettings `json:"pomodoro"`
	AI       *AISettings       `json:"ai"`
}

// BackupData 信封 data 字段:全量用户源数据(DailyStats 不在内)。
type BackupData struct {
	Tasks       []Task            `json:"tasks"`
	Sessions    []PomodoroSession `json:"sessions"`
	Schedules   []Schedule        `json:"schedules"`
	Settings    SettingsBundle    `json:"settings"`
	WorkLogs    []WorkLog         `json:"work_logs"` // items 嵌套
	WorkReports []WorkReport      `json:"work_reports"`
}

// BackupEnvelope 导出文件顶层结构。
type BackupEnvelope struct {
	App           string     `json:"app"`
	SchemaVersion int        `json:"schema_version"`
	ExportedAt    time.Time  `json:"exported_at"`
	Data          BackupData `json:"data"`
}

// ───── Preview(只读 diff 结果)─────

type FieldDiff struct {
	Field    string `json:"field"`
	Current  any    `json:"current"`
	Imported any    `json:"imported"`
}

type RecordConflict struct {
	ID     string      `json:"id"`
	Fields []FieldDiff `json:"fields"`
}

type SettingsFieldDiff struct {
	Section  string `json:"section"` // pomodoro | ai
	Field    string `json:"field"`
	Current  any    `json:"current"`
	Imported any    `json:"imported"`
}

// ModulePreview 单模块预览。集合类用 New/Identical/Conflict/Orphan + Conflicts;
// settings 模块仅用 SettingsConflicts。
type ModulePreview struct {
	New               int                 `json:"new"`
	Identical         int                 `json:"identical"`
	Conflict          int                 `json:"conflict"`
	Orphan            int                 `json:"orphan"`
	Conflicts         []RecordConflict    `json:"conflicts"`
	SettingsConflicts []SettingsFieldDiff `json:"settings_conflicts"`
}

type ImportPreview struct {
	SchemaVersion int                       `json:"schema_version"`
	SchemaWarning string                    `json:"schema_warning"`
	Modules       map[string]*ModulePreview `json:"modules"`
}

// ───── Apply(应用请求 + 执行计划 + 结果)─────

const (
	PolicyAddNewOnly   = "add_new_only"
	PolicyMergeFile    = "merge_file"
	PolicyMergeCurrent = "merge_current"
	PolicyReplace      = "replace"
)

const (
	ChoiceFile    = "file"
	ChoiceCurrent = "current"
)

type ModuleApply struct {
	Policy    string            `json:"policy"`
	Overrides map[string]string `json:"overrides"` // id -> "file"|"current"(仅冲突记录生效)
}

// ApplyImportRequest 自包含:含完整文件 payload + 每模块策略/override。
// settings 的最终值放在 Data.Settings(前端已合并冲突字段)。
type ApplyImportRequest struct {
	Data    BackupData             `json:"data"`
	Modules map[string]ModuleApply `json:"modules"`
}

// ApplyPlan 由 service 计算、repo 在单事务内执行。
type ApplyPlan struct {
	Tasks             []Task
	Sessions          []PomodoroSession
	Schedules         []Schedule
	WorkLogs          []WorkLog
	WorkReports       []WorkReport
	DeleteTasks       []string
	DeleteSessions    []string
	DeleteSchedules   []string
	DeleteWorkLogs    []string
	DeleteWorkReports []string
	Settings          *SettingsBundle
}

type ModuleApplyResult struct {
	Inserted int `json:"inserted"`
	Updated  int `json:"updated"`
	Deleted  int `json:"deleted"`
}

type ApplyResult struct {
	Applied map[string]ModuleApplyResult `json:"applied"`
}
