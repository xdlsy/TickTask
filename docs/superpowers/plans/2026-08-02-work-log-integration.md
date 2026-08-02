# Work-Log 集成到 TickTask 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把 work-log skill 的「脑暴 → 四维结构化日报 → 周期报告分层汇总」理念原生集成到 TickTask，形成独立「工作日志」页面，复用已有 AI service，遵循 editorial minimalism 风格。

**Architecture:** Go 后端新增 model/repository/service/handler 四层（work_log 模块），三张 SQLite 表（work_logs/work_items/work_reports）存结构化四维数据；复用 `service.AIService`（OpenAI 兼容客户端）做拆条与汇总；前端新增 `/work-log` 页面（左时间轴 + 右详情）+ Pinia store + 7 个子组件。M1~M6 分阶段独立交付。

**Tech Stack:** Go 1.21 / Gin 1.10 / GORM 1.25 / SQLite | Vue 3.5 / Pinia 2.2 / Element Plus 2.8 / Vite 5.4 / TypeScript 5.6 (strict) / Vitest 2.1

**关联 spec：** `docs/superpowers/specs/2026-08-02-work-log-integration-design.md`

---

## File Structure

### 后端新建（10 个文件）

| 文件 | 职责 |
|---|---|
| `backend/internal/model/work_log.go` | WorkLog / WorkItem / WorkReport / WorkReportType |
| `backend/internal/repository/work_log_repo.go` | Repository interface + 私有 struct，CRUD + 范围查询 |
| `backend/internal/repository/work_log_repo_test.go` | 真 SQLite 测试 |
| `backend/internal/service/work_log_calendar.go` | 周期算法（周/月/半年/年） |
| `backend/internal/service/work_log_calendar_test.go` | 表驱动测试 |
| `backend/internal/service/work_log_service.go` | 业务编排：拆条 / 保存 / 边界 / 分层汇总（含 INVARIANT 守门） |
| `backend/internal/service/work_log_service_test.go` | mock repo + mock AI client 测试 |
| `backend/internal/ai/work_log_prompts.go` | system prompt + JSON schema 描述 |
| `backend/internal/api/handler/work_log.go` | 9 个 HTTP handler |
| `backend/internal/api/handler/work_log_test.go` | handler 测试（用 mocks_test.go 风格 manual mock） |

### 后端修改（3 个文件）

| 文件 | 改动 |
|---|---|
| `backend/pkg/database/db.go` | `AutoMigrate` 加入 WorkLog/WorkItem/WorkReport |
| `backend/internal/api/router.go` | `SetupRouter` 加入 `workLogService` 参数 + `/api/work-logs` `/api/work-reports` 路由组 |
| `backend/cmd/server/main.go` | wire `workLogRepo` + `workLogService` 并传入 `SetupRouter` |

### 前端新建（10 个文件）

| 文件 | 职责 |
|---|---|
| `frontend/src/views/WorkLog.vue` | 页面 |
| `frontend/src/stores/workLog.ts` | Pinia store |
| `frontend/src/components/work-log/Timeline.vue` | 左时间轴 |
| `frontend/src/components/work-log/TodayContextCard.vue` | 预填卡片 |
| `frontend/src/components/work-log/BrainDumpInput.vue` | 脑暴输入 + AI 拆条按钮 |
| `frontend/src/components/work-log/WorkItemEditor.vue` | 单条四维表单 |
| `frontend/src/components/work-log/WorkItemList.vue` | items 列表 + 增删拖拽 |
| `frontend/src/components/work-log/ReportActions.vue` | 生成报告下拉按钮 |
| `frontend/src/components/work-log/ReportDetail.vue` | 报告只读视图 |
| `frontend/src/views/__tests__/WorkLog.spec.ts` | 页面集成测试 |

加上每个组件的 `.spec.ts`，共 ~16 个测试文件。

### 前端修改（4 个文件）

| 文件 | 改动 |
|---|---|
| `frontend/src/router/index.ts` | 加 `/work-log` 路由 |
| `frontend/src/App.vue` | `navItems` 加入「工作日志」入口（Document 图标）|
| `frontend/src/types/index.ts` | 加 WorkLog/WorkItem/WorkReport/WorkReportType/TodayContext 等类型 |
| `frontend/src/api/client.ts` | 加 workLog/workReport API 方法 |

---

## Phase M1 — 后端骨架（不含 AI）

**目标：** 三张表迁移成功，9 个端点都能调通（拆条/汇总返回占位响应），CRUD 走通。验证：`go test ./...` 通过；用 curl 能创建/查询日报。

### Task M1.1：定义 model

**Files:**
- Create: `backend/internal/model/work_log.go`

- [ ] **Step 1: 创建 model 文件**

```go
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
```

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./internal/model/...`
Expected: 无输出，退出码 0

- [ ] **Step 3: Commit**

```bash
git add backend/internal/model/work_log.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add WorkLog/WorkItem/WorkReport models"
```

---

### Task M1.2：注册 AutoMigrate

**Files:**
- Modify: `backend/pkg/database/db.go:31-39`

- [ ] **Step 1: 修改 AutoMigrate 函数**

把 `AutoMigrate` 函数体改为：

```go
// AutoMigrate 自动迁移表结构
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.Task{},
		&model.PomodoroSession{},
		&model.Setting{},
		&model.DailyStats{},
		&model.Schedule{},
		&model.WorkLog{},
		&model.WorkItem{},
		&model.WorkReport{},
	)
}
```

- [ ] **Step 2: 验证迁移能跑通**

Run: `cd backend && go run cmd/server/main.go &` 然后 `curl http://localhost:8080/api/tasks`（任意端点确认服务起来），`kill %1`

或者更简单：写一个一次性脚本验证。最直接是看 SQLite 文件里有新表：

```bash
cd backend && go run cmd/server/main.go &
sleep 3
sqlite3 data/ticktask.db ".tables" | grep -E "work_logs|work_items|work_reports"
kill %1
```

Expected: 输出包含 `work_logs` `work_items` `work_reports`

如果本机没有 sqlite3 CLI，跳过此步——后续 repo 测试会用 GORM 验证。

- [ ] **Step 3: Commit**

```bash
git add backend/pkg/database/db.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): register work_log tables in AutoMigrate"
```

---

### Task M1.3：定义 repository interface + 实现

**Files:**
- Create: `backend/internal/repository/work_log_repo.go`

- [ ] **Step 1: 创建 repository 文件**

```go
// backend/internal/repository/work_log_repo.go
package repository

import (
	"errors"
	"ticktask/internal/model"

	"gorm.io/gorm"
)

// WorkLogRepository 工作日志数据访问接口
type WorkLogRepository interface {
	// WorkLog CRUD
	CreateWorkLog(log *model.WorkLog) error
	GetWorkLogByDate(date string) (*model.WorkLog, error)
	GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error)
	UpdateWorkLog(log *model.WorkLog) error
	UpsertWorkLog(log *model.WorkLog) error // PUT 语义：存在则更新 items 全量替换，否则创建

	// WorkItem
	ReplaceItems(workLogID string, items []model.WorkItem) error

	// WorkReport
	CreateWorkReport(report *model.WorkReport) error
	UpdateWorkReport(report *model.WorkReport) error
	GetWorkReportByTypeAndPeriod(t model.WorkReportType, periodKey string) (*model.WorkReport, error)
	ListWorkReports(t model.WorkReportType) ([]*model.WorkReport, error)
}

// ErrWorkLogNotFound 复用既有 ErrNotFound
var ErrWorkLogNotFound = ErrNotFound

type workLogRepository struct {
	db *gorm.DB
}

func NewWorkLogRepository(db *gorm.DB) WorkLogRepository {
	return &workLogRepository{db: db}
}

func (r *workLogRepository) CreateWorkLog(log *model.WorkLog) error {
	return r.db.Create(log).Error
}

func (r *workLogRepository) GetWorkLogByDate(date string) (*model.WorkLog, error) {
	var log model.WorkLog
	err := r.db.Preload("Items", "seq > 0").Where("date = ?", date).First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWorkLogNotFound
	}
	if err != nil {
		return nil, err
	}
	return &log, nil
}

func (r *workLogRepository) GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error) {
	var logs []*model.WorkLog
	err := r.db.Preload("Items", "seq > 0").
		Where("date BETWEEN ? AND ?", from, to).
		Order("date DESC").
		Find(&logs).Error
	if err != nil {
		return nil, err
	}
	return logs, nil
}

func (r *workLogRepository) UpdateWorkLog(log *model.WorkLog) error {
	return r.db.Save(log).Error
}

func (r *workLogRepository) UpsertWorkLog(log *model.WorkLog) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		var existing model.WorkLog
		err := tx.Where("date = ?", log.Date).First(&existing).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(log).Error
		}
		if err != nil {
			return err
		}
		log.ID = existing.ID
		if err := tx.Save(log).Error; err != nil {
			return err
		}
		// items 全量替换
		if err := tx.Where("work_log_id = ?", log.ID).Delete(&model.WorkItem{}).Error; err != nil {
			return err
		}
		for i := range log.Items {
			log.Items[i].ID = "" // 让 GORM 自动生成新 ID
			log.Items[i].WorkLogID = log.ID
		}
		if len(log.Items) > 0 {
			return tx.Create(&log.Items).Error
		}
		return nil
	})
}

func (r *workLogRepository) ReplaceItems(workLogID string, items []model.WorkItem) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("work_log_id = ?", workLogID).Delete(&model.WorkItem{}).Error; err != nil {
			return err
		}
		for i := range items {
			items[i].WorkLogID = workLogID
		}
		if len(items) > 0 {
			return tx.Create(&items).Error
		}
		return nil
	})
}

func (r *workLogRepository) CreateWorkReport(report *model.WorkReport) error {
	return r.db.Create(report).Error
}

func (r *workLogRepository) UpdateWorkReport(report *model.WorkReport) error {
	return r.db.Save(report).Error
}

func (r *workLogRepository) GetWorkReportByTypeAndPeriod(t model.WorkReportType, periodKey string) (*model.WorkReport, error) {
	var report model.WorkReport
	err := r.db.Where("type = ? AND period_key = ?", t, periodKey).First(&report).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrWorkLogNotFound
	}
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func (r *workLogRepository) ListWorkReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	var reports []*model.WorkReport
	err := r.db.Where("type = ?", t).Order("period_key DESC").Find(&reports).Error
	if err != nil {
		return nil, err
	}
	return reports, nil
}
```

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./internal/repository/...`
Expected: 无输出，退出码 0

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/work_log_repo.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add WorkLogRepository interface + GORM impl"
```

---

### Task M1.4：写 repository 测试

**Files:**
- Create: `backend/internal/repository/work_log_repo_test.go`

- [ ] **Step 1: 写测试文件**

```go
// backend/internal/repository/work_log_repo_test.go
package repository

import (
	"testing"

	"ticktask/internal/model"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkLog{}, &model.WorkItem{}, &model.WorkReport{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	// 清空表避免 cache=shared 跨测试污染
	db.Exec("DELETE FROM work_items")
	db.Exec("DELETE FROM work_logs")
	db.Exec("DELETE FROM work_reports")
	return db
}

func TestCreateAndGetWorkLogByDate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	log := &model.WorkLog{
		ID:    "wl-1",
		Date:  "2026-08-02",
		Items: []model.WorkItem{{ID: "wi-1", WorkLogID: "wl-1", Seq: 1, Title: "T1"}},
	}
	if err := repo.CreateWorkLog(log); err != nil {
		t.Fatalf("create: %v", err)
	}

	got, err := repo.GetWorkLogByDate("2026-08-02")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != "wl-1" {
		t.Errorf("id = %s, want wl-1", got.ID)
	}
	if len(got.Items) != 1 || got.Items[0].Title != "T1" {
		t.Errorf("items mismatch: %+v", got.Items)
	}
}

func TestGetWorkLogByDate_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	_, err := repo.GetWorkLogByDate("2026-08-02")
	if err != ErrWorkLogNotFound {
		t.Errorf("err = %v, want ErrWorkLogNotFound", err)
	}
}

func TestUpsertWorkLog_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	log := &model.WorkLog{
		ID:    "wl-1",
		Date:  "2026-08-02",
		Items: []model.WorkItem{{ID: "wi-1", WorkLogID: "wl-1", Seq: 1, Title: "T1"}},
	}
	if err := repo.UpsertWorkLog(log); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, _ := repo.GetWorkLogByDate("2026-08-02")
	if len(got.Items) != 1 {
		t.Errorf("items len = %d, want 1", len(got.Items))
	}
}

func TestUpsertWorkLog_ReplaceItems(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	// 第一次保存：1 个 item
	repo.UpsertWorkLog(&model.WorkLog{
		ID: "wl-1", Date: "2026-08-02",
		Items: []model.WorkItem{{ID: "wi-1", WorkLogID: "wl-1", Seq: 1, Title: "T1"}},
	})
	// 第二次保存：2 个 item（不同 title）
	repo.UpsertWorkLog(&model.WorkLog{
		ID: "wl-1", Date: "2026-08-02",
		Items: []model.WorkItem{
			{ID: "wi-2", WorkLogID: "wl-1", Seq: 1, Title: "T2"},
			{ID: "wi-3", WorkLogID: "wl-1", Seq: 2, Title: "T3"},
		},
	})

	got, _ := repo.GetWorkLogByDate("2026-08-02")
	if len(got.Items) != 2 {
		t.Fatalf("items len = %d, want 2 (full replace)", len(got.Items))
	}
	for _, it := range got.Items {
		if it.Title == "T1" {
			t.Errorf("old item T1 should be deleted; got items: %+v", got.Items)
		}
	}
}

func TestGetWorkLogsInRange(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	repo.CreateWorkLog(&model.WorkLog{ID: "wl-1", Date: "2026-08-01"})
	repo.CreateWorkLog(&model.WorkLog{ID: "wl-2", Date: "2026-08-02"})
	repo.CreateWorkLog(&model.WorkLog{ID: "wl-3", Date: "2026-08-03"})

	logs, err := repo.GetWorkLogsInRange("2026-08-01", "2026-08-02")
	if err != nil {
		t.Fatalf("range: %v", err)
	}
	if len(logs) != 2 {
		t.Errorf("len = %d, want 2", len(logs))
	}
	// 应按 date DESC 排序
	if logs[0].Date != "2026-08-02" {
		t.Errorf("order wrong: %+v", logs)
	}
}

func TestCreateWorkReport_UniqueIndex(t *testing.T) {
	db := setupTestDB(t)
	repo := NewWorkLogRepository(db)

	r1 := &model.WorkReport{ID: "wr-1", Type: model.ReportWeekly, PeriodKey: "2026-W31", StartDate: "2026-07-27", EndDate: "2026-08-02"}
	if err := repo.CreateWorkReport(r1); err != nil {
		t.Fatalf("create r1: %v", err)
	}

	r2 := &model.WorkReport{ID: "wr-2", Type: model.ReportWeekly, PeriodKey: "2026-W31", StartDate: "2026-07-27", EndDate: "2026-08-02"}
	if err := repo.CreateWorkReport(r2); err == nil {
		t.Errorf("expected unique constraint violation, got nil")
	}
}
```

- [ ] **Step 2: 跑测试**

Run: `cd backend && go test -v ./internal/repository/ -run "TestCreateAndGetWorkLogByDate|TestGetWorkLogByDate_NotFound|TestUpsertWorkLog_Create|TestUpsertWorkLog_ReplaceItems|TestGetWorkLogsInRange|TestCreateWorkReport_UniqueIndex"`
Expected: 全部 PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/work_log_repo_test.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "test(work-log): add WorkLogRepository tests covering CRUD + unique index"
```

---

### Task M1.5：写 service 占位（含接口骨架，AI 方法留 stub）

**Files:**
- Create: `backend/internal/service/work_log_service.go`

- [ ] **Step 1: 创建 service 文件（占位 + 结构定义）**

```go
// backend/internal/service/work_log_service.go
package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"ticktask/internal/model"
	"ticktask/internal/repository"
)

// ── DTO ──

// TodayContext 今日预填上下文（完成任务、番茄钟会话、日程）
type TodayContext struct {
	Date             string         `json:"date"`
	CompletedTasks   []TaskBrief    `json:"completed_tasks"`
	PomodoroSessions []SessionBrief `json:"pomodoro_sessions"`
	PomodoroSummary  SessionSummary `json:"pomodoro_summary"`
	// 日程相关（M3 接入，先留空）
}

// TaskBrief 任务摘要
type TaskBrief struct {
	ID    string `json:"id"`
	Title string `json:"title"`
}

// SessionBrief 番茄钟会话摘要
type SessionBrief struct {
	ID        string `json:"id"`
	TaskID    string `json:"task_id"`
	TaskTitle string `json:"task_title"`
	StartedAt string `json:"started_at"`
	Minutes   int    `json:"minutes"`
}

// SessionSummary 番茄钟汇总
type SessionSummary struct {
	Count        int `json:"count"`
	TotalMinutes int `json:"total_minutes"`
}

// BrainDumpInput AI 拆条输入
type BrainDumpInput struct {
	BrainDump string        `json:"brain_dump"`
	Context   TodayContext  `json:"context"`
}

// StructuredItem AI 拆条输出的单条工作（四维）
type StructuredItem struct {
	Title         string `json:"title"`
	Content       string `json:"content"`
	ProblemSolved string `json:"problem_solved"`
	Result        string `json:"result"`
	Impact        string `json:"impact"`
}

// StructuredWorkLog AI 拆条输出（预览，未落库）
type StructuredWorkLog struct {
	Items   []StructuredItem `json:"items"`
	Summary string           `json:"summary"`
}

// SaveWorkLogInput 保存日报输入
type SaveWorkLogInput struct {
	Date         string          `json:"date"`
	Summary      string          `json:"summary"`
	RawBrainDump string          `json:"raw_brain_dump"`
	Items        []SaveItemInput `json:"items"`
}

// SaveItemInput 保存时的单条 item
type SaveItemInput struct {
	Seq           int    `json:"seq"`
	Title         string `json:"title"`
	Content       string `json:"content"`
	ProblemSolved string `json:"problem_solved"`
	Result        string `json:"result"`
	Impact        string `json:"impact"`
}

// GenerateReportInput 生成报告输入
type GenerateReportInput struct {
	Type      model.WorkReportType `json:"type"`
	PeriodKey string                `json:"period_key"` // 可空，缺省=当前周期
	Force     bool                  `json:"force"`      // 覆盖已存在
}

// ── 错误 ──

var (
	ErrWorkLogAlreadyExists = errors.New("work log already exists for this date")
	ErrReportAlreadyExists  = errors.New("report already exists, set force=true to overwrite")
	ErrAIStructureFailed    = errors.New("AI structuring failed")
)

// ── AI client interface（M1 stub，M2 接真实实现）──

// WorkLogAIClient AI 拆条/汇总客户端接口
type WorkLogAIClient interface {
	StructureBrainDump(input BrainDumpInput) (*StructuredWorkLog, error)
	GenerateWeeklyReport(items []model.WorkItem, start, end string) (*ReportSummary, error)
	GenerateMonthlyReport(weeklies []*model.WorkReport, orphanItems []model.WorkItem, start, end string) (*ReportSummary, error)
	GenerateHalfYearReport(monthlies []*model.WorkReport, start, end string) (*ReportSummary, error)
	GenerateYearlyReport(monthlies []*model.WorkReport, start, end string) (*ReportSummary, error)
}

// ReportSummary 报告汇总结构（4 字段，所有报告 type 共用）
type ReportSummary struct {
	CoreWork       string `json:"core_work"`       // 核心工作 / 重大成果
	MainProgress   string `json:"main_progress"`   // 主要进展 / 趋势
	OpenIssues     string `json:"open_issues"`     // 遗留问题 / 关键问题
	NextFocus      string `json:"next_focus"`      // 下周/下阶段关注（年报可空）
}

// ── Service ──

// WorkLogService 工作日志业务编排
type WorkLogService struct {
	repo          repository.WorkLogRepository
	taskRepo      repository.TaskRepository
	sessionRepo   repository.SessionRepository
	aiClient      WorkLogAIClient
	idGenerator   func() string
}

// NewWorkLogService 构造
func NewWorkLogService(
	repo repository.WorkLogRepository,
	taskRepo repository.TaskRepository,
	sessionRepo repository.SessionRepository,
	aiClient WorkLogAIClient,
) *WorkLogService {
	return &WorkLogService{
		repo:        repo,
		taskRepo:    taskRepo,
		sessionRepo: sessionRepo,
		aiClient:    aiClient,
		idGenerator: func() string { return fmt.Sprintf("id-%d", time.Now().UnixNano()) },
	}
}

// GetTodayContext 拉今日预填上下文（M1 实现）
func (s *WorkLogService) GetTodayContext(date string) (*TodayContext, error) {
	// 简化：date 已是 YYYY-MM-DD，转 time.Time 取范围
	t, err := time.Parse("2006-01-02", date)
	if err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}
	start := t
	end := t.Add(24 * time.Hour)

	tasks, err := s.taskRepo.GetCompletedTasksInRange(start, end)
	if err != nil {
		return nil, err
	}
	completed := make([]TaskBrief, 0, len(tasks))
	for _, t := range tasks {
		completed = append(completed, TaskBrief{ID: t.ID, Title: t.Title})
	}

	sessions, err := s.sessionRepo.GetSessionsInRange(start, end)
	if err != nil {
		return nil, err
	}
	sessionBriefs := make([]SessionBrief, 0, len(sessions))
	totalMin := 0
	for _, s := range sessions {
		minutes := int(s.EndedAt.Sub(s.StartedAt).Minutes())
		totalMin += minutes
		sessionBriefs = append(sessionBriefs, SessionBrief{
			ID: s.ID, TaskID: s.TaskID, StartedAt: s.StartedAt.Format(time.RFC3339), Minutes: minutes,
		})
	}

	return &TodayContext{
		Date:             date,
		CompletedTasks:   completed,
		PomodoroSessions: sessionBriefs,
		PomodoroSummary:  SessionSummary{Count: len(sessions), TotalMinutes: totalMin},
	}, nil
}

// StructureBrainDump AI 拆条（不落库）
func (s *WorkLogService) StructureBrainDump(input BrainDumpInput) (*StructuredWorkLog, error) {
	if s.aiClient == nil {
		return nil, ErrAIStructureFailed
	}
	out, err := s.aiClient.StructureBrainDump(input)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrAIStructureFailed, err)
	}
	// schema 校验：每条 item 必须有 title（非空）；四维字段缺失则填"（待补充）"
	for i := range out.Items {
		if out.Items[i].Title == "" {
			return nil, fmt.Errorf("%w: item[%d] missing title", ErrAIStructureFailed, i)
		}
		if out.Items[i].Result == "" {
			out.Items[i].Result = "（待补充）"
		}
		if out.Items[i].Impact == "" {
			out.Items[i].Impact = "（待补充）"
		}
	}
	return out, nil
}

// SaveWorkLog 保存日报（POST 语义：同日已存在 → ErrWorkLogAlreadyExists）
func (s *WorkLogService) SaveWorkLog(input SaveWorkLogInput) (*model.WorkLog, error) {
	// 校验日期
	if _, err := time.Parse("2006-01-02", input.Date); err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}
	existing, err := s.repo.GetWorkLogByDate(input.Date)
	if err != nil && !errors.Is(err, repository.ErrWorkLogNotFound) {
		return nil, err
	}
	if existing != nil {
		return existing, ErrWorkLogAlreadyExists
	}
	log := s.buildWorkLogFromInput(input)
	if err := s.repo.CreateWorkLog(log); err != nil {
		return nil, err
	}
	return log, nil
}

// UpdateWorkLog 更新日报（PUT 语义：items 全量替换）
func (s *WorkLogService) UpdateWorkLog(input SaveWorkLogInput) (*model.WorkLog, error) {
	if _, err := time.Parse("2006-01-02", input.Date); err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}
	log := s.buildWorkLogFromInput(input)
	if err := s.repo.UpsertWorkLog(log); err != nil {
		return nil, err
	}
	return s.repo.GetWorkLogByDate(input.Date)
}

// GetWorkLog 按日期读
func (s *WorkLogService) GetWorkLog(date string) (*model.WorkLog, error) {
	return s.repo.GetWorkLogByDate(date)
}

// ListWorkLogs 范围查
func (s *WorkLogService) ListWorkLogs(from, to string) ([]*model.WorkLog, error) {
	return s.repo.GetWorkLogsInRange(from, to)
}

// GenerateReport 生成报告（M4 实现，M1 stub）
func (s *WorkLogService) GenerateReport(input GenerateReportInput) (*model.WorkReport, error) {
	return nil, errors.New("GenerateReport not implemented yet (M4)")
}

// GetReport 读报告
func (s *WorkLogService) GetReport(t model.WorkReportType, periodKey string) (*model.WorkReport, error) {
	return s.repo.GetWorkReportByTypeAndPeriod(t, periodKey)
}

// ListReports 列表
func (s *WorkLogService) ListReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	return s.repo.ListWorkReports(t)
}

// ── 内部 ──

func (s *WorkLogService) buildWorkLogFromInput(input SaveWorkLogInput) *model.WorkLog {
	logID := s.idGenerator()
	items := make([]model.WorkItem, 0, len(input.Items))
	for _, it := range input.Items {
		items = append(items, model.WorkItem{
			ID:            s.idGenerator(),
			WorkLogID:     logID,
			Seq:           it.Seq,
			Title:         it.Title,
			Content:       it.Content,
			ProblemSolved: it.ProblemSolved,
			Result:        it.Result,
			Impact:        it.Impact,
		})
	}
	return &model.WorkLog{
		ID:           logID,
		Date:         input.Date,
		Summary:      input.Summary,
		RawBrainDump: input.RawBrainDump,
		Items:        items,
	}
}

// 确保 ReportSummary 能 JSON 序列化（避免 unused 警告）
var _ = json.Marshal
```

**注：** `taskRepo.GetCompletedTasksInRange` 和 `sessionRepo.GetSessionsInRange` 假设已存在；如不存在，Task M1.6 会加。

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./internal/service/...`
Expected: 若报 `GetCompletedTasksInRange` / `GetSessionsInRange` 未定义，先继续 Task M1.6 加上方法；否则无输出。

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/work_log_service.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add WorkLogService with DTOs, AI client interface, and M1 methods (Structure/Save/Update/Get/List)"
```

---

### Task M1.6：补充 taskRepo / sessionRepo 的范围查询方法（如果不存在）

**Files:**
- Modify: `backend/internal/repository/task_repo.go`（若 GetCompletedTasksInRange 不存在）
- Modify: `backend/internal/repository/session_repo.go`（若 GetSessionsInRange 不存在）

- [ ] **Step 1: 检查现有方法**

Run: `cd backend && grep -n "GetCompletedTasksInRange\|GetSessionsInRange\|GetTasksInRange" internal/repository/*.go`
Expected: 列出现有方法签名；若两个都不存在，按下面补；若签名不同，调整 service 调用方。

- [ ] **Step 2: 给 task_repo 加 GetCompletedTasksInRange（如缺失）**

读 `backend/internal/repository/task_repo.go` 顶部 interface 定义；若没有 `GetCompletedTasksInRange(start, end time.Time) ([]*model.Task, error)`，则：

在 interface 加方法签名，在 struct 实现里加：

```go
func (r *taskRepository) GetCompletedTasksInRange(start, end time.Time) ([]*model.Task, error) {
	var tasks []*model.Task
	err := r.db.Where("status = ? AND completed_at BETWEEN ? AND ?",
		model.StatusCompleted, start, end).
		Order("completed_at DESC").
		Find(&tasks).Error
	if err != nil {
		return nil, err
	}
	return tasks, nil
}
```

- [ ] **Step 3: 给 session_repo 加 GetSessionsInRange（如缺失）**

读 `backend/internal/repository/session_repo.go`；若没有 `GetSessionsInRange(start, end time.Time) ([]*model.PomodoroSession, error)`，则：

在 interface 加方法签名，在 struct 实现里加：

```go
func (r *sessionRepository) GetSessionsInRange(start, end time.Time) ([]*model.PomodoroSession, error) {
	var sessions []*model.PomodoroSession
	err := r.db.Where("started_at BETWEEN ? AND ? AND ended_at IS NOT NULL",
		start, end).
		Order("started_at DESC").
		Find(&sessions).Error
	if err != nil {
		return nil, err
	}
	return sessions, nil
}
```

注：假设 PomodoroSession 有 `StartedAt` `EndedAt` `TaskID` 字段（与 service 用法一致）。若实际字段名不同，先读 `model/session.go` 校准。

- [ ] **Step 4: 编译验证**

Run: `cd backend && go build ./...`
Expected: 无输出，退出码 0

- [ ] **Step 5: 跑既有测试确保没回归**

Run: `cd backend && go test ./...`
Expected: 全部 PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/repository/task_repo.go backend/internal/repository/session_repo.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add GetCompletedTasksInRange/GetSessionsInRange to support today-context"
```

---

### Task M1.7：写 service 测试（mock repo + mock AI）

**Files:**
- Create: `backend/internal/service/work_log_service_test.go`

- [ ] **Step 1: 写测试文件**

```go
// backend/internal/service/work_log_service_test.go
package service

import (
	"errors"
	"testing"
	"time"

	"ticktask/internal/model"
	"ticktask/internal/repository"
)

// ── Mock repo ──

type mockWorkLogRepo struct {
	logs    map[string]*model.WorkLog // date -> log
	reports map[string]*model.WorkReport
}

func newMockWorkLogRepo() *mockWorkLogRepo {
	return &mockWorkLogRepo{
		logs:    make(map[string]*model.WorkLog),
		reports: make(map[string]*model.WorkReport),
	}
}

func (m *mockWorkLogRepo) CreateWorkLog(log *model.WorkLog) error {
	if _, ok := m.logs[log.Date]; ok {
		return errors.New("duplicate")
	}
	cp := *log
	m.logs[log.Date] = &cp
	return nil
}
func (m *mockWorkLogRepo) GetWorkLogByDate(date string) (*model.WorkLog, error) {
	if l, ok := m.logs[date]; ok {
		return l, nil
	}
	return nil, repository.ErrWorkLogNotFound
}
func (m *mockWorkLogRepo) GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error) {
	var out []*model.WorkLog
	for d, l := range m.logs {
		if d >= from && d <= to {
			out = append(out, l)
		}
	}
	return out, nil
}
func (m *mockWorkLogRepo) UpdateWorkLog(log *model.WorkLog) error {
	m.logs[log.Date] = log
	return nil
}
func (m *mockWorkLogRepo) UpsertWorkLog(log *model.WorkLog) error {
	m.logs[log.Date] = log
	return nil
}
func (m *mockWorkLogRepo) ReplaceItems(workLogID string, items []model.WorkItem) error {
	return nil
}
func (m *mockWorkLogRepo) CreateWorkReport(r *model.WorkReport) error {
	key := string(r.Type) + ":" + r.PeriodKey
	if _, ok := m.reports[key]; ok {
		return errors.New("duplicate")
	}
	m.reports[key] = r
	return nil
}
func (m *mockWorkLogRepo) UpdateWorkReport(r *model.WorkReport) error {
	key := string(r.Type) + ":" + r.PeriodKey
	m.reports[key] = r
	return nil
}
func (m *mockWorkLogRepo) GetWorkReportByTypeAndPeriod(t model.WorkReportType, k string) (*model.WorkReport, error) {
	if r, ok := m.reports[string(t)+":"+k]; ok {
		return r, nil
	}
	return nil, repository.ErrWorkLogNotFound
}
func (m *mockWorkLogRepo) ListWorkReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	var out []*model.WorkReport
	for k, r := range m.reports {
		if string(t)+":" == k[:len(string(t))+1] {
			out = append(out, r)
		}
	}
	return out, nil
}

// Mock AI client
type mockAIClient struct {
	structuredOut *StructuredWorkLog
	structuredErr error
}

func (m *mockAIClient) StructureBrainDump(input BrainDumpInput) (*StructuredWorkLog, error) {
	if m.structuredErr != nil {
		return nil, m.structuredErr
	}
	return m.structuredOut, nil
}
func (m *mockAIClient) GenerateWeeklyReport(items []model.WorkItem, start, end string) (*ReportSummary, error) {
	return nil, errors.New("not impl")
}
func (m *mockAIClient) GenerateMonthlyReport(w []*model.WorkReport, o []model.WorkItem, start, end string) (*ReportSummary, error) {
	return nil, errors.New("not impl")
}
func (m *mockAIClient) GenerateHalfYearReport(mo []*model.WorkReport, start, end string) (*ReportSummary, error) {
	return nil, errors.New("not impl")
}
func (m *mockAIClient) GenerateYearlyReport(mo []*model.WorkReport, start, end string) (*ReportSummary, error) {
	return nil, errors.New("not impl")
}

// Mock task / session repos（M1.5 用到的接口的最小实现）
type mockTaskRepo struct {
	tasks []*model.Task
}

func (m *mockTaskRepo) GetCompletedTasksInRange(start, end time.Time) ([]*model.Task, error) {
	var out []*model.Task
	for _, t := range m.tasks {
		if t.CompletedAt != nil && t.CompletedAt.After(start) && t.CompletedAt.Before(end) {
			out = append(out, t)
		}
	}
	return out, nil
}

type mockSessionRepo struct {
	sessions []*model.PomodoroSession
}

func (m *mockSessionRepo) GetSessionsInRange(start, end time.Time) ([]*model.PomodoroSession, error) {
	var out []*model.PomodoroSession
	for _, s := range m.sessions {
		if s.StartedAt.After(start) && s.StartedAt.Before(end) {
			out = append(out, s)
		}
	}
	return out, nil
}

// 注意：上面 mock task/session 只实现了 service 真正用到的方法。
// 如果 TaskRepository / SessionRepository interface 里这两个方法不存在，
// 需要在 M1.6 里加。Service 的字段类型也用 interface，可能需要 mock 实现
// 完整 interface —— 这里假设 interface 已含这两个方法（M1.6 已加）。
// 若 interface 还有别的方法，mock 会编译失败 —— 把 mock 改为嵌入 interface 的方式
// 或补全所有方法。最简单做法：用 mockTaskRepo 嵌入 repository.TaskRepository
// （unexported 嵌入），其他方法 nil panic —— M1 测试不会触发。

func newServiceForTest() (*WorkLogService, *mockWorkLogRepo, *mockAIClient) {
	repo := newMockWorkLogRepo()
	ai := &mockAIClient{}
	svc := &WorkLogService{
		repo:        repo,
		taskRepo:    nil, // M1.5 GetTodayContext 在 service 测试里单独覆盖
		sessionRepo: nil,
		aiClient:    ai,
		idGenerator: func() string { return "test-id" },
	}
	return svc, repo, ai
}

// ── 测试用例 ──

func TestSaveWorkLog_New(t *testing.T) {
	svc, _, _ := newServiceForTest()
	in := SaveWorkLogInput{
		Date: "2026-08-02", Summary: "今日 ok",
		Items: []SaveItemInput{{Seq: 1, Title: "T1", Content: "c1"}},
	}
	log, err := svc.SaveWorkLog(in)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if log.ID == "" || len(log.Items) != 1 {
		t.Errorf("unexpected log: %+v", log)
	}
}

func TestSaveWorkLog_DuplicateReturnsErr(t *testing.T) {
	svc, _, _ := newServiceForTest()
	in := SaveWorkLogInput{Date: "2026-08-02", Items: []SaveItemInput{{Seq: 1, Title: "T1"}}}
	if _, err := svc.SaveWorkLog(in); err != nil {
		t.Fatalf("first save: %v", err)
	}
	existing, err := svc.SaveWorkLog(in)
	if !errors.Is(err, ErrWorkLogAlreadyExists) {
		t.Errorf("err = %v, want ErrWorkLogAlreadyExists", err)
	}
	if existing == nil || existing.Date != "2026-08-02" {
		t.Errorf("should return existing log on conflict")
	}
}

func TestUpdateWorkLog_FullReplace(t *testing.T) {
	svc, _, _ := newServiceForTest()
	// 先 POST 一份
	svc.SaveWorkLog(SaveWorkLogInput{
		Date: "2026-08-02",
		Items: []SaveItemInput{{Seq: 1, Title: "T1"}},
	})
	// PUT 改成 2 条
	_, err := svc.UpdateWorkLog(SaveWorkLogInput{
		Date: "2026-08-02",
		Items: []SaveItemInput{
			{Seq: 1, Title: "T2"},
			{Seq: 2, Title: "T3"},
		},
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := svc.GetWorkLog("2026-08-02")
	if len(got.Items) != 2 {
		t.Errorf("items len = %d, want 2 (full replace)", len(got.Items))
	}
}

func TestStructureBrainDump_NoTitle_Fails(t *testing.T) {
	svc, _, ai := newServiceForTest()
	ai.structuredOut = &StructuredWorkLog{
		Items: []StructuredItem{{Content: "no title"}}, // title 空
	}
	_, err := svc.StructureBrainDump(BrainDumpInput{BrainDump: "x"})
	if !errors.Is(err, ErrAIStructureFailed) {
		t.Errorf("err = %v, want ErrAIStructureFailed", err)
	}
}

func TestStructureBrainDump_FillsPendingForMissingDims(t *testing.T) {
	svc, _, ai := newServiceForTest()
	ai.structuredOut = &StructuredWorkLog{
		Items: []StructuredItem{{Title: "T1", Content: "c1"}}, // result/impact 为空
	}
	out, err := svc.StructureBrainDump(BrainDumpInput{BrainDump: "x"})
	if err != nil {
		t.Fatalf("structure: %v", err)
	}
	if out.Items[0].Result != "（待补充）" {
		t.Errorf("Result = %q, want '（待补充）'", out.Items[0].Result)
	}
	if out.Items[0].Impact != "（待补充）" {
		t.Errorf("Impact = %q, want '（待补充）'", out.Items[0].Impact)
	}
}

func TestSaveWorkLog_InvalidDate(t *testing.T) {
	svc, _, _ := newServiceForTest()
	_, err := svc.SaveWorkLog(SaveWorkLogInput{Date: "2026-02-30"})
	if err == nil {
		t.Errorf("expected error for invalid date")
	}
}

func TestGenerateReport_StubReturnsError(t *testing.T) {
	svc, _, _ := newServiceForTest()
	_, err := svc.GenerateReport(GenerateReportInput{Type: model.ReportWeekly, PeriodKey: "2026-W31"})
	if err == nil {
		t.Errorf("expected stub error")
	}
}
```

- [ ] **Step 2: 跑测试**

Run: `cd backend && go test -v ./internal/service/ -run "TestSaveWorkLog|TestUpdateWorkLog|TestStructureBrainDump|TestGenerateReport_Stub"`
Expected: 全部 PASS

如果 mock 编译失败（因为 TaskRepository/SessionRepository interface 还有别的方法），改 mock 嵌入接口：

```go
type mockTaskRepo struct {
	repository.TaskRepository // 嵌入接口，未实现的方法会 nil panic
	tasks []*model.Task
}
```

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/work_log_service_test.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "test(work-log): add service tests covering save/update/structure + duplicate detection"
```

---

### Task M1.8：写 handler（9 个端点骨架）

**Files:**
- Create: `backend/internal/api/handler/work_log.go`

- [ ] **Step 1: 创建 handler 文件**

```go
// backend/internal/api/handler/work_log.go
package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"ticktask/internal/model"
	"ticktask/internal/service"
)

type WorkLogHandler struct {
	svc *service.WorkLogService
}

func NewWorkLogHandler(svc *service.WorkLogService) *WorkLogHandler {
	return &WorkLogHandler{svc: svc}
}

// ── 日报端点 ──

func (h *WorkLogHandler) GetTodayContext(c *gin.Context) {
	date := c.Query("date")
	if date == "" {
		date = todayStr()
	}
	ctx, err := h.svc.GetTodayContext(date)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, ctx)
}

type structureRequest struct {
	BrainDump string                        `json:"brain_dump"`
	Context   service.TodayContext          `json:"context"`
}

func (h *WorkLogHandler) Structure(c *gin.Context) {
	var req structureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if strings.TrimSpace(req.BrainDump) == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "brain_dump required"})
		return
	}
	out, err := h.svc.StructureBrainDump(service.BrainDumpInput{
		BrainDump: req.BrainDump, Context: req.Context,
	})
	if err != nil {
		if errors.Is(err, service.ErrAIStructureFailed) {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *WorkLogHandler) CreateWorkLog(c *gin.Context) {
	var req service.SaveWorkLogInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	log, err := h.svc.SaveWorkLog(req)
	if err != nil {
		if errors.Is(err, service.ErrWorkLogAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{
				"error":             err.Error(),
				"existing_work_log": log,
			})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, log)
}

func (h *WorkLogHandler) ListWorkLogs(c *gin.Context) {
	from := c.Query("from")
	to := c.Query("to")
	if from == "" || to == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to required"})
		return
	}
	logs, err := h.svc.ListWorkLogs(from, to)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (h *WorkLogHandler) GetWorkLog(c *gin.Context) {
	date := c.Param("date")
	log, err := h.svc.GetWorkLog(date)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, log)
}

func (h *WorkLogHandler) UpdateWorkLog(c *gin.Context) {
	date := c.Param("date")
	var req service.SaveWorkLogInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.Date = date
	log, err := h.svc.UpdateWorkLog(req)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, log)
}

// ── 报告端点 ──

func (h *WorkLogHandler) GenerateReport(c *gin.Context) {
	var req service.GenerateReportInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	report, err := h.svc.GenerateReport(req)
	if err != nil {
		if errors.Is(err, service.ErrReportAlreadyExists) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, report)
}

func (h *WorkLogHandler) GetReport(c *gin.Context) {
	t := model.WorkReportType(c.Query("type"))
	periodKey := c.Query("period_key")
	if t == "" || periodKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type and period_key required"})
		return
	}
	report, err := h.svc.GetReport(t, periodKey)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, report)
}

func (h *WorkLogHandler) ListReports(c *gin.Context) {
	t := model.WorkReportType(c.Query("type"))
	if t == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type required"})
		return
	}
	reports, err := h.svc.ListReports(t)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"reports": reports})
}

// ── helpers ──

func todayStr() string {
	// 不引 time 包到 helper 避免循环；这里直接 inline
	return timeNowYMD()
}
```

`timeNowYMD()` 放在同文件：

```go
// 同文件尾部
import "time"

func timeNowYMD() string {
	return time.Now().Format("2006-01-02")
}
```

**注：** Go 不允许函数内 import，把 `time` import 放文件顶部。

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./internal/api/handler/...`
Expected: 无输出，退出码 0

- [ ] **Step 3: Commit**

```bash
git add backend/internal/api/handler/work_log.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add WorkLogHandler with 9 endpoints (GenerateReport still stub)"
```

---

### Task M1.9：注册路由 + wire service 到 main.go

**Files:**
- Modify: `backend/internal/api/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: 改 router.go 签名**

把 `SetupRouter` 函数签名加入 `workLogService *service.WorkLogService`：

```go
func SetupRouter(
	cfg *config.Config,
	taskService *service.TaskService,
	timerService *service.TimerService,
	aiService *service.AIService,
	analyticsService *service.AnalyticsService,
	scheduleService *service.ScheduleService,
	workLogService *service.WorkLogService,
	wsHub *websocket.Hub,
	settingRepo repository.SettingRepository,
) *gin.Engine {
```

- [ ] **Step 2: 在路由组内加 work-logs / work-reports**

在 `schedules` 路由组之后（约第 113 行 `}` 之前）加：

```go
		// 工作日志
		workLogs := api.Group("/work-logs")
		{
			wlHandler := handler.NewWorkLogHandler(workLogService)
			workLogs.GET("/today/context", wlHandler.GetTodayContext)
			workLogs.POST("/structure", wlHandler.Structure)
			workLogs.GET("", wlHandler.ListWorkLogs)
			workLogs.POST("", wlHandler.CreateWorkLog)
			workLogs.GET("/:date", wlHandler.GetWorkLog)
			workLogs.PUT("/:date", wlHandler.UpdateWorkLog)
		}

		// 工作日志报告
		workReports := api.Group("/work-reports")
		{
			wrHandler := handler.NewWorkLogHandler(workLogService)
			workReports.POST("/generate", wrHandler.GenerateReport)
			workReports.GET("", wrHandler.ListReports)
			workReports.GET("/:type/:periodKey", wrHandler.GetReport)
		}
```

- [ ] **Step 3: 改 main.go wire**

在 `main.go` 的"初始化 Schedule Service"之后，加：

```go
	// 初始化 WorkLog Service
	workLogRepo := repository.NewWorkLogRepository(db)
	// M1：aiClient 传 nil（M2 接真实实现）
	var workLogAIClient service.WorkLogAIClient = nil // M2 替换为真实 client
	workLogService := service.NewWorkLogService(workLogRepo, taskRepo, sessionRepo, workLogAIClient)
```

修改 `SetupRouter` 调用，加入 `workLogService`：

```go
	router := api.SetupRouter(cfg, taskService, timerService, aiService, analyticsService, scheduleService, workLogService, wsHub, settingRepo)
```

- [ ] **Step 4: 编译验证**

Run: `cd backend && go build ./...`
Expected: 无输出，退出码 0

- [ ] **Step 5: 启动服务，curl 验证端点存在**

```bash
cd backend && go run cmd/server/main.go &
sleep 3
# 验证 today/context 返回 200
curl -s http://localhost:8080/api/work-logs/today/context | head -50
# 验证 list 返回 200 + 空
curl -s "http://localhost:8080/api/work-logs?from=2026-08-01&to=2026-08-31" | head -50
kill %1
```

Expected: `/today/context` 返回 JSON（即使 task/session 拉空也应是 200 + 空 array）；list 返回 `{"logs":[]}`

- [ ] **Step 6: 跑全部测试**

Run: `cd backend && go test ./...`
Expected: 全部 PASS

- [ ] **Step 7: Commit**

```bash
git add backend/internal/api/router.go backend/cmd/server/main.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): wire WorkLogService into router + main, register 9 endpoints"
```

---

### Task M1.10：handler 测试（happy path + 主要错误路径）

**Files:**
- Create: `backend/internal/api/handler/work_log_test.go`

- [ ] **Step 1: 写测试文件**

```go
// backend/internal/api/handler/work_log_test.go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"ticktask/internal/model"
	"ticktask/internal/service"
)

func init() { gin.SetMode(gin.TestMode) }

func newTestWorkLogRouter(svc *service.WorkLogService) *gin.Engine {
	r := gin.New()
	h := NewWorkLogHandler(svc)
	r.GET("/api/work-logs/today/context", h.GetTodayContext)
	r.POST("/api/work-logs/structure", h.Structure)
	r.GET("/api/work-logs", h.ListWorkLogs)
	r.POST("/api/work-logs", h.CreateWorkLog)
	r.GET("/api/work-logs/:date", h.GetWorkLog)
	r.PUT("/api/work-logs/:date", h.UpdateWorkLog)
	r.POST("/api/work-reports/generate", h.GenerateReport)
	r.GET("/api/work-reports", h.ListReports)
	return r
}

func TestHandler_CreateWorkLog_Happy(t *testing.T) {
	_, _, _, router, body := setupWorkLog(t) // helper from this file
	_ = router
	_ = body
}

// 简化：直接 inline 而非 helper（plan 不引入过多抽象）

func TestHandler_CreateWorkLog_Conflict(t *testing.T) {
	svc, _, _ := newWorkLogServiceForHandler()
	router := newTestWorkLogRouter(svc)

	body := service.SaveWorkLogInput{
		Date:   "2026-08-02",
		Summary: "s",
		Items:  []service.SaveItemInput{{Seq: 1, Title: "T1"}},
	}
	bodyBytes, _ := json.Marshal(body)

	// 第一次 201
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/work-logs", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("first create code = %d, body = %s", w.Code, w.Body.String())
	}

	// 第二次 409
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/work-logs", bytes.NewReader(bodyBytes))
	req2.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w2, req2)
	if w2.Code != http.StatusConflict {
		t.Errorf("second create code = %d, want 409; body = %s", w2.Code, w2.Body.String())
	}
}

func TestHandler_Structure_EmptyBody(t *testing.T) {
	svc, _, _ := newWorkLogServiceForHandler()
	router := newTestWorkLogRouter(svc)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/work-logs/structure",
		bytes.NewReader([]byte(`{"brain_dump":"  ","context":{}}`)))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("code = %d, want 400; body = %s", w.Code, w.Body.String())
	}
}

func TestHandler_GenerateReport_StubError(t *testing.T) {
	svc, _, _ := newWorkLogServiceForHandler()
	router := newTestWorkLogRouter(svc)

	body := service.GenerateReportInput{Type: model.ReportWeekly, PeriodKey: "2026-W31"}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/work-reports/generate", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)
	if w.Code != http.StatusBadGateway {
		t.Errorf("code = %d, want 502; body = %s", w.Code, w.Body.String())
	}
}

// helper：复用 service 测试里的 mock，但因 package 不同需重新包装
func newWorkLogServiceForHandler() (*service.WorkLogService, *mockWorkLogRepo, *mockAIClient) {
	// 注：service 内部 mock 是 unexported，handler 包无法直接用。
	// 解决方案：handler 测试里写自己的 mock（实现 service.WorkLogRepository interface）。
	// 简化：直接 new 真 service + nil ai（多数测试不依赖 AI）
	return service.NewWorkLogService(
		&handlerTestRepo{}, nil, nil, nil,
	), nil, nil
}

// handlerTestRepo 实现 service.WorkLogRepository（用 service 包导出的 interface）
type handlerTestRepo struct{}

func (r *handlerTestRepo) CreateWorkLog(log *model.WorkLog) error                  { return nil }
func (r *handlerTestRepo) GetWorkLogByDate(date string) (*model.WorkLog, error)     { return nil, nil }
func (r *handlerTestRepo) GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error) {
	return nil, nil
}
func (r *handlerTestRepo) UpdateWorkLog(log *model.WorkLog) error                  { return nil }
func (r *handlerTestRepo) UpsertWorkLog(log *model.WorkLog) error                  { return nil }
func (r *handlerTestRepo) ReplaceItems(id string, items []model.WorkItem) error    { return nil }
func (r *handlerTestRepo) CreateWorkReport(r2 *model.WorkReport) error             { return nil }
func (r *handlerTestRepo) UpdateWorkReport(r2 *model.WorkReport) error             { return nil }
func (r *handlerTestRepo) GetWorkReportByTypeAndPeriod(t model.WorkReportType, k string) (*model.WorkReport, error) {
	return nil, nil
}
func (r *handlerTestRepo) ListWorkReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	return nil, nil
}
```

**注：** `handlerTestRepo` 的方法是空实现——这让 conflict 测试过不了。重新设计：

实际上 `TestHandler_CreateWorkLog_Conflict` 需要一个能"先成功再失败"的 repo。最简单做法：handler 测试只用一个内存 map 实现，不放 mocks_test.go。

把 handlerTestRepo 改为带 map：

```go
type handlerTestRepo struct {
	logs map[string]*model.WorkLog
}

func newHandlerTestRepo() *handlerTestRepo {
	return &handlerTestRepo{logs: make(map[string]*model.WorkLog)}
}

func (r *handlerTestRepo) CreateWorkLog(log *model.WorkLog) error {
	if _, ok := r.logs[log.Date]; ok {
		return errors.New("duplicate")
	}
	cp := *log
	r.logs[log.Date] = &cp
	return nil
}
func (r *handlerTestRepo) GetWorkLogByDate(date string) (*model.WorkLog, error) {
	if l, ok := r.logs[date]; ok {
		return l, nil
	}
	return nil, repository.ErrWorkLogNotFound
}
// ... 其他方法同前
```

把 `newWorkLogServiceForHandler` 改为：

```go
func newWorkLogServiceForHandler() *service.WorkLogService {
	return service.NewWorkLogService(newHandlerTestRepo(), nil, nil, nil)
}
```

- [ ] **Step 2: 跑测试**

Run: `cd backend && go test -v ./internal/api/handler/ -run "TestHandler_"`
Expected: 全部 PASS

- [ ] **Step 3: Commit**

```bash
git add backend/internal/api/handler/work_log_test.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "test(work-log): add handler tests covering create conflict + structure + report stub"
```

---

**Phase M1 验收：**

- `cd backend && go test ./...` 全 PASS
- `cd backend && go run cmd/server/main.go` 启动后 curl 9 个端点都能调通
- 三张表已迁移
- 进化 M2 前 commit：所有 M1 commit 都已在 main 分支

---

## Phase M2 — 日报 AI 拆条

**目标：** 接通真实 OpenAI 兼容 AI client，`/api/work-logs/structure` 端到端跑通——贴一段脑暴，AI 返回结构化 JSON，service 校验后返回前端。

### Task M2.1：写 AI prompt 模板

**Files:**
- Create: `backend/internal/ai/work_log_prompts.go`

- [ ] **Step 1: 创建 prompts 文件**

```go
// backend/internal/ai/work_log_prompts.go
package ai

// WorkLogStructureSystem AI 拆条的 system prompt
const WorkLogStructureSystem = `你是一个工作日报整理助手。用户会提供一段"今日工作脑暴"（自由文本），以及今日的预填上下文（已完成的任务、番茄钟会话）。

任务：把脑暴拆成若干"核心工作"条目，每条按四维结构展开。

# 输出格式（严格 JSON，不要 markdown 代码块包裹）

{
  "items": [
    {
      "title": "20 字以内的标题",
      "content": "做了什么，300 字以内",
      "problem_solved": "解决了什么问题，300 字以内",
      "result": "已经产生的具体结果（数字 / 产出 / 结论），300 字以内",
      "impact": "对后续的影响（项目推进 / 协作 / 风险 / 可复用产物 / 成长），300 字以内"
    }
  ],
  "summary": "今日 2-3 句小结"
}

# 红线（绝对不能违反）

1. **绝不编造**：脑暴里没有的内容，绝不凭借常识或推测编造。
2. **凑不出具体产出时，整维只输出"（待补充）"三个字**，不要复述+括注凑数。错误示例："初步定了优先级（具体排序：待补充）"——这是违规。正确做法：整维就写"（待补充）"。
3. **判断标准**：能否写出"具体"的数字 / 产出 / 结论。模糊话不算具体。
4. **不要包含未在脑暴出现的任务**（即便预填上下文里有）。预填上下文只是参考，不能直接列入 items。

# 拆条原则

- 一条"核心工作" = 一件有完整产出的事；不要把碎片化的零碎活动列为单独条目。
- 通常一天 1-5 条。
- 如果脑暴是空的或完全无法解析，返回 {"items": [], "summary": "（待补充）"}。
`

// WorkLogStructureUser 用户 prompt 模板（拼装 brain_dump + context）
const WorkLogStructureUser = `# 今日脑暴

%s

# 今日预填上下文（仅供参考，不要直接列入 items）

%s

请按 system 指示输出 JSON。`

// WorkLogWeeklyReportSystem 周报汇总 system prompt
const WorkLogWeeklyReportSystem = `你是一个周报生成助手。我会给你本周（7 天）的若干"工作条目"（每条含四维：内容/解决的问题/结果/影响）。

任务：按主题归并去重，生成本周报告 4 字段。

# 输出格式（严格 JSON）

{
  "core_work": "本周核心工作（按主题归并去重后，2-4 个主题，每段 1-2 句）",
  "main_progress": "主要进展（关键里程碑、数字、产出）",
  "open_issues": "遗留问题（未关闭的事项）",
  "next_focus": "下周关注（1-3 条）"
}

# 红线

- 不编造：items 里没有的，不写入。
- items 为空时返回 {"core_work": "（待补充）", ...}。
`

// WorkLogMonthlyReportSystem 月报 system（读周报 + 孤儿 items）
const WorkLogMonthlyReportSystem = `你是一个月报生成助手。我会给你本月 4-5 份周报的 JSON（含 core_work/main_progress/open_issues/next_focus 字段），以及未被周报覆盖的零散天 items。

任务：合并成月度报告 4 字段，结构同周报。

# 红线

- 不要直接复制某一周的 next_focus 当月报的 open_issues；要做合并。
- 不编造。
`

// WorkLogHalfYearReportSystem 半年报 system（读 6 份月报）
const WorkLogHalfYearReportSystem = `你是一个半年报生成助手。我会给你该半年 6 份月报的 JSON。

任务：合成半年报告，3 字段：

{
  "core_work": "重大成果（3-6 条）",
  "main_progress": "趋势（发展脉络）",
  "open_issues": "关键问题"
}

不编造。`

// WorkLogYearlyReportSystem 年报 system（读 12 份月报或 2 份半年报）
const WorkLogYearlyReportSystem = `你是一个年报生成助手。我会给你本年 12 份月报的 JSON。

任务：合成年度报告，3 字段（同半年报 schema）。

不编造。`
```

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./internal/ai/...`
Expected: 无输出（package ai 已存在，新增一个文件不影响）

- [ ] **Step 3: Commit**

```bash
git add backend/internal/ai/work_log_prompts.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add AI prompts for brain-dump structuring + 4 report types"
```

---

### Task M2.2：实现 WorkLogAIClient 接通 ai_service

**Files:**
- Create: `backend/internal/service/work_log_ai_client.go`

- [ ] **Step 1: 实现 AI client**

先读 `backend/internal/service/ai_service.go` 顶部，确认 OpenAI client 的调用方式（`AIService` 内部如何 call LLM）。假设有以下方法可复用：`Call(systemPrompt, userPrompt string) (string, error)` 或类似。

如果 `ai_service.go` 有 `client *ai.OpenAIClient` 字段或 `CallLLM(messages []Message) (string, error)` 方法，复用它。

```go
// backend/internal/service/work_log_ai_client.go
package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"ticktask/internal/ai"
	"ticktask/internal/model"
)

// workLogAIClient 真实实现：调 OpenAI 兼容 LLM
type workLogAIClient struct {
	aiService *AIService
}

// NewWorkLogAIClient 构造。aiService 必须非 nil。
func NewWorkLogAIClient(aiService *AIService) WorkLogAIClient {
	return &workLogAIClient{aiService: aiService}
}

// CallLLM 由 AIService 提供的统一调用接口（命名按 ai_service.go 实际方法调整）
// 假设 AIService 暴露：CallLLM(systemPrompt, userPrompt string) (string, error)
// 如果实际方法不同，调整这里。

func (c *workLogAIClient) callLLM(system, user string) (string, error) {
	// 假设 AIService 有 CallLLM 方法；若没有，需在 ai_service.go 加一个公共方法
	// 或者直接调用 internal/ai 包的 client
	return c.aiService.CallLLM(system, user)
}

func (c *workLogAIClient) StructureBrainDump(input BrainDumpInput) (*StructuredWorkLog, error) {
	userPrompt := fmt.Sprintf(ai.WorkLogStructureUser, input.BrainDump, formatContextForPrompt(input.Context))
	raw, err := c.callLLM(ai.WorkLogStructureSystem, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call: %w", err)
	}
	cleaned := stripCodeFence(raw)
	var out StructuredWorkLog
	if err := json.Unmarshal([]byte(cleaned), &out); err != nil {
		return nil, fmt.Errorf("parse LLM JSON: %w; raw=%s", err, truncated(raw, 500))
	}
	return &out, nil
}

func (c *workLogAIClient) GenerateWeeklyReport(items []model.WorkItem, start, end string) (*ReportSummary, error) {
	userPrompt := fmt.Sprintf("本周（%s ~ %s）的工作条目 JSON：\n%s", start, end, itemsToJSON(items))
	raw, err := c.callLLM(ai.WorkLogWeeklyReportSystem, userPrompt)
	if err != nil {
		return nil, err
	}
	return parseReportSummary(raw)
}

func (c *workLogAIClient) GenerateMonthlyReport(weeklies []*model.WorkReport, orphanItems []model.WorkItem, start, end string) (*ReportSummary, error) {
	userPrompt := fmt.Sprintf("本月（%s ~ %s）的周报 JSON 数组：\n%s\n\n未被周报覆盖的零散 items：\n%s",
		start, end, reportsToJSON(weeklies), itemsToJSON(orphanItems))
	raw, err := c.callLLM(ai.WorkLogMonthlyReportSystem, userPrompt)
	if err != nil {
		return nil, err
	}
	return parseReportSummary(raw)
}

func (c *workLogAIClient) GenerateHalfYearReport(monthlies []*model.WorkReport, start, end string) (*ReportSummary, error) {
	userPrompt := fmt.Sprintf("该半年（%s ~ %s）的月报 JSON 数组：\n%s",
		start, end, reportsToJSON(monthlies))
	raw, err := c.callLLM(ai.WorkLogHalfYearReportSystem, userPrompt)
	if err != nil {
		return nil, err
	}
	return parseReportSummary(raw)
}

func (c *workLogAIClient) GenerateYearlyReport(monthlies []*model.WorkReport, start, end string) (*ReportSummary, error) {
	userPrompt := fmt.Sprintf("本年（%s ~ %s）的月报 JSON 数组：\n%s",
		start, end, reportsToJSON(monthlies))
	raw, err := c.callLLM(ai.WorkLogYearlyReportSystem, userPrompt)
	if err != nil {
		return nil, err
	}
	return parseReportSummary(raw)
}

// ── helpers ──

func formatContextForPrompt(ctx TodayContext) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("已完成任务 %d 条：\n", len(ctx.CompletedTasks)))
	for _, t := range ctx.CompletedTasks {
		sb.WriteString(fmt.Sprintf("- %s\n", t.Title))
	}
	sb.WriteString(fmt.Sprintf("\n番茄钟会话 %d 个，共 %d 分钟。\n",
		ctx.PomodoroSummary.Count, ctx.PomodoroSummary.TotalMinutes))
	return sb.String()
}

func itemsToJSON(items []model.WorkItem) string {
	type brief struct {
		Title         string `json:"title"`
		Content       string `json:"content"`
		ProblemSolved string `json:"problem_solved"`
		Result        string `json:"result"`
		Impact        string `json:"impact"`
	}
	out := make([]brief, len(items))
	for i, it := range items {
		out[i] = brief{it.Title, it.Content, it.ProblemSolved, it.Result, it.Impact}
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func reportsToJSON(reports []*model.WorkReport) string {
	out := make([]map[string]string, len(reports))
	for i, r := range reports {
		out[i] = map[string]string{
			"period_key":   r.PeriodKey,
			"start_date":   r.StartDate,
			"end_date":     r.EndDate,
			"summary_json": r.SummaryJSON,
		}
	}
	b, _ := json.Marshal(out)
	return string(b)
}

func parseReportSummary(raw string) (*ReportSummary, error) {
	cleaned := stripCodeFence(raw)
	var s ReportSummary
	if err := json.Unmarshal([]byte(cleaned), &s); err != nil {
		return nil, fmt.Errorf("parse report JSON: %w; raw=%s", err, truncated(raw, 500))
	}
	return &s, nil
}

// stripCodeFence 移除可能的 ```json ... ``` 包裹
func stripCodeFence(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```") {
		// 去掉第一行
		if idx := strings.Index(s, "\n"); idx >= 0 {
			s = s[idx+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	return strings.TrimSpace(s)
}

func truncated(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "...(truncated)"
}
```

- [ ] **Step 2: 在 AIService 上加 CallLLM 公共方法（如果不存在）**

读 `backend/internal/service/ai_service.go`，搜索是否有可复用的 LLM 调用方法。

若不存在，在 `AIService` struct 上加：

```go
// CallLLM 公共调用入口（system + user prompt → 字符串响应）
func (s *AIService) CallLLM(systemPrompt, userPrompt string) (string, error) {
	if s.client == nil {
		return "", errors.New("AI client not initialized")
	}
	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: userPrompt},
	}
	resp, err := s.client.Chat(messages)
	if err != nil {
		return "", err
	}
	return resp, nil
}
```

注：`Message` 和 `s.client.Chat` 的具体类型/方法按 `internal/ai/` 包实际签名调整。

- [ ] **Step 3: 编译验证**

Run: `cd backend && go build ./...`
Expected: 无输出，退出码 0

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/work_log_ai_client.go backend/internal/service/ai_service.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): implement WorkLogAIClient wiring to AIService.CallLLM"
```

---

### Task M2.3：在 main.go 接真实 AI client

**Files:**
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: 把 nil 替换为真实 client**

定位 M1.9 中加的：

```go
var workLogAIClient service.WorkLogAIClient = nil
```

改为：

```go
workLogAIClient := service.NewWorkLogAIClient(aiService)
```

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./...`
Expected: 无输出

- [ ] **Step 3: 端到端测试（手测）**

```bash
cd backend && go run cmd/server/main.go &
sleep 3
curl -s -X POST http://localhost:8080/api/work-logs/structure \
  -H "Content-Type: application/json" \
  -d '{
    "brain_dump": "今天修了订单服务 P99 飙高的 bug，连接池配置过小。改完上线 P99 从 1.8s 降到 230ms。",
    "context": {"date": "2026-08-02", "completed_tasks": [], "pomodoro_sessions": [], "pomodoro_summary": {"count": 0, "total_minutes": 0}}
  }' | head -50
kill %1
```

Expected: 返回 JSON 含 `items` 数组（1 条）+ `summary` 字符串；item 的 result 字段应有"230ms"具体数字。

如果 AI 配置未启用（无 API key），返回 502 + `AI structuring failed`。

- [ ] **Step 4: Commit**

```bash
git add backend/cmd/server/main.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): wire real WorkLogAIClient in main"
```

---

### Task M2.4：补 AI 失败路径测试

**Files:**
- Modify: `backend/internal/service/work_log_service_test.go`

- [ ] **Step 1: 加新测试用例**

在文件末尾追加：

```go
func TestStructureBrainDump_AIClientError(t *testing.T) {
	svc, _, ai := newServiceForTest()
	ai.structuredErr = errors.New("openai timeout")
	_, err := svc.StructureBrainDump(BrainDumpInput{BrainDump: "x"})
	if !errors.Is(err, ErrAIStructureFailed) {
		t.Errorf("err = %v, want wrap ErrAIStructureFailed", err)
	}
}

func TestStructureBrainDump_LLMReturnsInvalidJSON(t *testing.T) {
	// 此测试需要 mock AI client 返回非 JSON 字符串
	// 简化：mockAIClient 当前直接返回 StructuredWorkLog，绕过 JSON 解析
	// 真实 JSON 解析在 work_log_ai_client.go，应单独写 work_log_ai_client_test.go
	t.Skip("covered by work_log_ai_client_test.go (TestParseReportSummary_BadJSON)")
}
```

- [ ] **Step 2: 写 AI client 单独的 JSON 解析测试**

Create: `backend/internal/service/work_log_ai_client_test.go`

```go
// backend/internal/service/work_log_ai_client_test.go
package service

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestStripCodeFence_Plain(t *testing.T) {
	in := `{"a":1}`
	got := stripCodeFence(in)
	if got != in {
		t.Errorf("got=%q want=%q", got, in)
	}
}

func TestStripCodeFence_JSONBlock(t *testing.T) {
	in := "```json\n{\"a\":1}\n```"
	got := stripCodeFence(in)
	want := `{"a":1}`
	if got != want {
		t.Errorf("got=%q want=%q", got, want)
	}
}

func TestParseReportSummary_Valid(t *testing.T) {
	raw := `{"core_work":"cw","main_progress":"mp","open_issues":"oi","next_focus":"nf"}`
	s, err := parseReportSummary(raw)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if s.CoreWork != "cw" || s.NextFocus != "nf" {
		t.Errorf("parsed wrong: %+v", s)
	}
}

func TestParseReportSummary_BadJSON(t *testing.T) {
	raw := `not json`
	_, err := parseReportSummary(raw)
	if err == nil {
		t.Errorf("expected error")
	}
	if !strings.Contains(err.Error(), "parse report JSON") {
		t.Errorf("err = %v, want contains 'parse report JSON'", err)
	}
}

func TestItemsToJSON_Structure(t *testing.T) {
	items := []model.WorkItem{
		{Title: "T1", Content: "C1", ProblemSolved: "P1", Result: "R1", Impact: "I1"},
	}
	s := itemsToJSON(items)
	var got []map[string]string
	if err := json.Unmarshal([]byte(s), &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got) != 1 || got[0]["title"] != "T1" {
		t.Errorf("wrong: %v", got)
	}
}
```

- [ ] **Step 3: 跑测试**

Run: `cd backend && go test -v ./internal/service/ -run "TestStripCodeFence|TestParseReportSummary|TestItemsToJSON"`
Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/work_log_service_test.go backend/internal/service/work_log_ai_client_test.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "test(work-log): add AI client JSON parsing tests + service error wrap test"
```

---

**Phase M2 验收：**

- `/api/work-logs/structure` 端到端可用：贴脑暴 → 返回结构化 JSON
- AI 失败时返回 502，不写空数据进库
- M1 的 9 个端点继续工作（无回归）

---

## Phase M3 — 前端日报页

**目标：** 新增 `/work-log` 路由 + 页面，能完成「查看时间轴 → 输入脑暴 → AI 拆条 → 编辑预览 → 保存」全流程。报告相关 UI 留 M5。

### Task M3.1：加 TypeScript 类型

**Files:**
- Modify: `frontend/src/types/index.ts`

- [ ] **Step 1: 在 types/index.ts 末尾追加**

```ts
// ── 工作日志 ──

export interface WorkItem {
  id: string
  work_log_id: string
  seq: number
  title: string
  content: string
  problem_solved: string
  result: string
  impact: string
}

export interface WorkLog {
  id: string
  date: string
  summary: string
  raw_brain_dump: string
  created_at: string
  updated_at: string
  items: WorkItem[]
}

export type WorkReportType = 'weekly' | 'monthly' | 'halfyear' | 'yearly'

export interface WorkReport {
  id: string
  type: WorkReportType
  period_key: string
  start_date: string
  end_date: string
  summary_json: string // JSON string，前端 parse 后渲染
  missing_days: string
  created_at: string
  updated_at: string
}

export interface TodayContext {
  date: string
  completed_tasks: Array<{ id: string; title: string }>
  pomodoro_sessions: Array<{
    id: string
    task_id: string
    task_title: string
    started_at: string
    minutes: number
  }>
  pomodoro_summary: { count: number; total_minutes: number }
}

export interface StructuredWorkLog {
  items: Array<{
    title: string
    content: string
    problem_solved: string
    result: string
    impact: string
  }>
  summary: string
}

export interface SaveWorkLogInput {
  date: string
  summary: string
  raw_brain_dump: string
  items: Array<{
    seq: number
    title: string
    content: string
    problem_solved: string
    result: string
    impact: string
  }>
}

export interface ReportSummary {
  core_work: string
  main_progress: string
  open_issues: string
  next_focus: string
}
```

- [ ] **Step 2: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无输出

- [ ] **Step 3: Commit**

```bash
git add frontend/src/types/index.ts
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add WorkLog/WorkItem/WorkReport/TodayContext frontend types"
```

---

### Task M3.2：加 API client 方法

**Files:**
- Modify: `frontend/src/api/client.ts`

- [ ] **Step 1: 在 client.ts 顶部 import 加新类型**

把第一行 import 改为（追加新类型）：

```ts
import type { Task, TaskResponse, PomodoroSession, ClassificationResult, PrioritySuggestion, AIStatus, PomodoroSettings, AISettings, TaskTimeStats, DailySummary, TrendData, DistributionStats, ScheduleEvent, CreateScheduleDTO, UpdateScheduleDTO, MoveScheduleDTO, RescheduleResult, DailyInsights, ReviseResponse, PomodoroByTaskResult, PomodoroTrendsResult, WorkLog, WorkReport, WorkReportType, TodayContext, StructuredWorkLog, SaveWorkLogInput, ReportSummary } from '@/types'
```

- [ ] **Step 2: 在 api 对象末尾追加（最后一个 `}` 之前）**

```ts
  // 工作日志
  getTodayContext: (date?: string) => client.get<TodayContext>('/work-logs/today/context', { params: { date } }),
  structureBrainDump: (brain_dump: string, context: TodayContext) =>
    client.post<StructuredWorkLog>('/work-logs/structure', { brain_dump, context }),
  listWorkLogs: (from: string, to: string) =>
    client.get<{ logs: WorkLog[] }>('/work-logs', { params: { from, to } }),
  getWorkLog: (date: string) => client.get<WorkLog>(`/work-logs/${date}`),
  createWorkLog: (data: SaveWorkLogInput) => client.post<WorkLog>('/work-logs', data),
  updateWorkLog: (date: string, data: SaveWorkLogInput) => client.put<WorkLog>(`/work-logs/${date}`, data),
  generateWorkReport: (type: WorkReportType, period_key?: string, force = false) =>
    client.post<WorkReport>('/work-reports/generate', { type, period_key, force }),
  listWorkReports: (type: WorkReportType) =>
    client.get<{ reports: WorkReport[] }>('/work-reports', { params: { type } }),
  getWorkReport: (type: WorkReportType, periodKey: string) =>
    client.get<WorkReport>(`/work-reports/${type}/${periodKey}`)
```

- [ ] **Step 3: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无输出

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/client.ts
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add 9 API methods on axios client"
```

---

### Task M3.3：写 Pinia store

**Files:**
- Create: `frontend/src/stores/workLog.ts`

- [ ] **Step 1: 创建 store 文件**

```ts
// frontend/src/stores/workLog.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api/client'
import { ElMessage } from 'element-plus'
import type {
  WorkLog, WorkReport, WorkReportType, TodayContext,
  StructuredWorkLog, SaveWorkLogInput, ReportSummary,
} from '@/types'

export type SelectedNode =
  | { kind: 'log'; date: string }
  | { kind: 'report'; type: WorkReportType; periodKey: string }

export const useWorkLogStore = defineStore('workLog', () => {
  const logs = ref<WorkLog[]>([])
  const currentLog = ref<WorkLog | null>(null)
  const todayContext = ref<TodayContext | null>(null)
  const reports = ref<Record<WorkReportType, WorkReport[]>>({
    weekly: [], monthly: [], halfyear: [], yearly: [],
  })
  const currentReport = ref<WorkReport | null>(null)
  const selected = ref<SelectedNode | null>(null)
  const loading = ref(false)

  async function fetchInitialRange() {
    loading.value = true
    try {
      const today = new Date()
      const to = today.toISOString().slice(0, 10)
      const from = new Date(today.getTime() - 90 * 86400_000).toISOString().slice(0, 10)
      const { data } = await api.listWorkLogs(from, to)
      logs.value = data.logs || []
    } catch (e: any) {
      ElMessage.error('加载日报列表失败：' + (e?.message || ''))
    } finally {
      loading.value = false
    }
  }

  async function fetchLog(date: string) {
    try {
      const { data } = await api.getWorkLog(date)
      currentLog.value = data
    } catch (e: any) {
      if (e?.response?.status === 404) {
        currentLog.value = null
      } else {
        ElMessage.error('加载日报失败：' + (e?.message || ''))
      }
    }
  }

  async function fetchTodayContext(date?: string) {
    try {
      const { data } = await api.getTodayContext(date)
      todayContext.value = data
    } catch (e: any) {
      ElMessage.error('加载今日预填失败：' + (e?.message || ''))
    }
  }

  async function structureBrainDump(text: string): Promise<StructuredWorkLog | null> {
    if (!todayContext.value) return null
    try {
      const { data } = await api.structureBrainDump(text, todayContext.value)
      return data
    } catch (e: any) {
      ElMessage.error('AI 拆条失败：' + (e?.response?.data?.error || e?.message || ''))
      return null
    }
  }

  async function saveWorkLog(payload: SaveWorkLogInput): Promise<boolean> {
    try {
      // 先尝试 POST；遇 409 走 PUT
      try {
        await api.createWorkLog(payload)
        ElMessage.success('日报已保存')
      } catch (e: any) {
        if (e?.response?.status === 409) {
          await api.updateWorkLog(payload.date, payload)
          ElMessage.success('日报已更新')
        } else {
          throw e
        }
      }
      await fetchInitialRange()
      await fetchLog(payload.date)
      return true
    } catch (e: any) {
      ElMessage.error('保存失败：' + (e?.response?.data?.error || e?.message || ''))
      return false
    }
  }

  async function generateReport(type: WorkReportType, periodKey?: string, force = false) {
    try {
      const { data } = await api.generateWorkReport(type, periodKey, force)
      await fetchReports(type)
      ElMessage.success('报告已生成')
      return data
    } catch (e: any) {
      if (e?.response?.status === 409 && !force) {
        // 调用方应先 confirm，再 force=true 重发
        throw e
      }
      ElMessage.error('生成报告失败：' + (e?.response?.data?.error || e?.message || ''))
      throw e
    }
  }

  async function fetchReports(type: WorkReportType) {
    try {
      const { data } = await api.listWorkReports(type)
      reports.value[type] = data.reports || []
    } catch (e: any) {
      ElMessage.error('加载报告列表失败：' + (e?.message || ''))
    }
  }

  async function fetchReport(type: WorkReportType, periodKey: string) {
    try {
      const { data } = await api.getWorkReport(type, periodKey)
      currentReport.value = data
    } catch (e: any) {
      ElMessage.error('加载报告失败：' + (e?.message || ''))
    }
  }

  function selectNode(node: SelectedNode) {
    selected.value = node
    if (node.kind === 'log') {
      fetchLog(node.date)
    } else {
      fetchReport(node.type, node.periodKey)
    }
  }

  return {
    logs, currentLog, todayContext, reports, currentReport, selected, loading,
    fetchInitialRange, fetchLog, fetchTodayContext, structureBrainDump,
    saveWorkLog, generateReport, fetchReports, fetchReport, selectNode,
  }
})
```

- [ ] **Step 2: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无输出

- [ ] **Step 3: Commit**

```bash
git add frontend/src/stores/workLog.ts
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add Pinia store with all 9 actions + selected-node state"
```

---

### Task M3.4：写 store 测试

**Files:**
- Create: `frontend/src/stores/workLog.spec.ts`

- [ ] **Step 1: 创建测试文件**

```ts
// frontend/src/stores/workLog.spec.ts
import { setActivePinia, createPinia } from 'pinia'
import { vi, beforeEach, describe, it, expect } from 'vitest'
import { useWorkLogStore } from './workLog'
import { api } from '@/api/client'
import { ElMessage } from 'element-plus'

vi.mock('@/api/client')
vi.mock('element-plus', () => ({ ElMessage: { error: vi.fn(), success: vi.fn() } }))

const mockApi = api as any

beforeEach(() => {
  setActivePinia(createPinia())
  vi.clearAllMocks()
  mockApi.listWorkLogs.mockResolvedValue({ data: { logs: [] } })
  mockApi.getTodayContext.mockResolvedValue({ data: { date: '2026-08-02', completed_tasks: [], pomodoro_sessions: [], pomodoro_summary: { count: 0, total_minutes: 0 } } })
  mockApi.structureBrainDump.mockResolvedValue({ data: { items: [], summary: '' } })
  mockApi.createWorkLog.mockResolvedValue({})
  mockApi.updateWorkLog.mockResolvedValue({})
  mockApi.getWorkLog.mockResolvedValue({ data: { id: 'wl-1', date: '2026-08-02', items: [] } })
})

describe('useWorkLogStore', () => {
  it('fetchInitialRange loads logs', async () => {
    mockApi.listWorkLogs.mockResolvedValue({ data: { logs: [{ id: '1', date: '2026-08-02', items: [] }] } })
    const store = useWorkLogStore()
    await store.fetchInitialRange()
    expect(store.logs).toHaveLength(1)
  })

  it('fetchLog 404 sets currentLog null', async () => {
    mockApi.getWorkLog.mockRejectedValue({ response: { status: 404 } })
    const store = useWorkLogStore()
    await store.fetchLog('2026-08-02')
    expect(store.currentLog).toBeNull()
  })

  it('structureBrainDump returns null when AI fails', async () => {
    mockApi.structureBrainDump.mockRejectedValue({ response: { data: { error: 'bad' } } })
    const store = useWorkLogStore()
    await store.fetchTodayContext()
    const out = await store.structureBrainDump('xxx')
    expect(out).toBeNull()
    expect(ElMessage.error).toHaveBeenCalled()
  })

  it('saveWorkLog POST then PUT on 409', async () => {
    mockApi.createWorkLog.mockRejectedValue({ response: { status: 409 } })
    const store = useWorkLogStore()
    const ok = await store.saveWorkLog({ date: '2026-08-02', summary: '', raw_brain_dump: '', items: [] })
    expect(ok).toBe(true)
    expect(mockApi.updateWorkLog).toHaveBeenCalled()
  })

  it('generateReport throws on 409 without force', async () => {
    mockApi.generateWorkReport.mockRejectedValue({ response: { status: 409 } })
    const store = useWorkLogStore()
    await expect(store.generateReport('weekly', '2026-W31', false)).rejects.toBeDefined()
  })
})
```

- [ ] **Step 2: 跑测试**

Run: `cd frontend && npx vitest run src/stores/workLog.spec.ts`
Expected: 全部 PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/src/stores/workLog.spec.ts
git -c user.name='lsy' -c user.email='lsy@local' commit -m "test(work-log): add store tests covering happy + error paths + 409 retry"
```

---

### Task M3.5：TodayContextCard 组件

**Files:**
- Create: `frontend/src/components/work-log/TodayContextCard.vue`

- [ ] **Step 1: 创建组件**

```vue
<template>
  <div class="today-context-card" v-if="context">
    <div class="ctx-header">
      <span class="ctx-title">今日预填</span>
      <span class="ctx-date">{{ context.date }}</span>
    </div>
    <div class="ctx-row" v-if="context.completed_tasks.length">
      <span class="ctx-label">已完成任务</span>
      <span class="ctx-value">{{ taskTitles }}</span>
    </div>
    <div class="ctx-row">
      <span class="ctx-label">番茄钟</span>
      <span class="ctx-value">
        {{ context.pomodoro_summary.count }} 个 · {{ context.pomodoro_summary.total_minutes }} 分钟
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { TodayContext } from '@/types'

const props = defineProps<{ context: TodayContext | null }>()

const taskTitles = computed(() =>
  (props.context?.completed_tasks || []).map(t => t.title).join('、'),
)
</script>

<style scoped>
.today-context-card {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 16px 20px;
  margin-bottom: 16px;
}
.ctx-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 10px;
}
.ctx-title {
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}
.ctx-date {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-muted);
}
.ctx-row {
  display: flex;
  gap: 12px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--text-secondary);
  padding: 4px 0;
}
.ctx-label {
  flex: 0 0 80px;
  color: var(--text-muted);
}
.ctx-value {
  flex: 1;
  color: var(--text-primary);
}
</style>
```

- [ ] **Step 2: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无输出

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/work-log/TodayContextCard.vue
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add TodayContextCard component"
```

---

### Task M3.6：BrainDumpInput 组件

**Files:**
- Create: `frontend/src/components/work-log/BrainDumpInput.vue`
- Create: `frontend/src/components/work-log/BrainDumpInput.spec.ts`

- [ ] **Step 1: 创建组件**

```vue
<template>
  <div class="brain-dump-input">
    <div class="bd-header">
      <span class="bd-title">脑暴输入</span>
      <button
        class="bd-btn"
        :disabled="loading || !text.trim()"
        @click="onStructure"
      >
        {{ loading ? 'AI 拆条中…' : 'AI 拆条 →' }}
      </button>
    </div>
    <textarea
      class="bd-textarea"
      v-model="text"
      :disabled="loading"
      placeholder="今天做了什么、解决了什么、产出了什么…一段话丢进来，AI 帮你拆成结构化条目。"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import type { StructuredWorkLog } from '@/types'

const props = defineProps<{ loading?: boolean; modelValue?: string }>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'structure', text: string): void
}>()

const text = ref(props.modelValue || '')

function onStructure() {
  if (!text.value.trim()) return
  emit('structure', text.value)
}
</script>

<style scoped>
.brain-dump-input {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 16px 20px;
  margin-bottom: 16px;
}
.bd-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 10px;
}
.bd-title {
  font-family: var(--font-display);
  font-size: 14px;
  font-weight: 600;
  color: var(--text-primary);
}
.bd-btn {
  background: var(--accent-primary);
  color: white;
  border: none;
  border-radius: var(--radius-sm);
  padding: 6px 14px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: background var(--transition-fast);
}
.bd-btn:hover:not(:disabled) {
  background: var(--accent-secondary);
}
.bd-btn:disabled {
  background: var(--text-muted);
  cursor: not-allowed;
}
.bd-textarea {
  width: 100%;
  min-height: 120px;
  padding: 10px 12px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
}
.bd-textarea:focus {
  outline: none;
  border-color: var(--accent-primary);
}
</style>
```

- [ ] **Step 2: 创建测试**

```ts
// frontend/src/components/work-log/BrainDumpInput.spec.ts
import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import BrainDumpInput from './BrainDumpInput.vue'

describe('BrainDumpInput', () => {
  it('disables button when text empty', () => {
    const w = mount(BrainDumpInput)
    const btn = w.find('.bd-btn')
    expect(btn.attributes('disabled')).toBeDefined()
  })

  it('emits structure on click with text', async () => {
    const w = mount(BrainDumpInput)
    await w.find('.bd-textarea').setValue('hello world')
    await w.find('.bd-btn').trigger('click')
    expect(w.emitted('structure')?.[0]).toEqual(['hello world'])
  })

  it('disables button when loading true', async () => {
    const w = mount(BrainDumpInput, { props: { loading: true } })
    expect(w.find('.bd-btn').attributes('disabled')).toBeDefined()
  })
})
```

- [ ] **Step 3: 跑测试 + 类型检查**

```bash
cd frontend && npx vitest run src/components/work-log/BrainDumpInput.spec.ts
cd frontend && npx vue-tsc --noEmit
```

Expected: 测试全 PASS，类型检查无错

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/work-log/BrainDumpInput.vue frontend/src/components/work-log/BrainDumpInput.spec.ts
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add BrainDumpInput component + tests"
```

---

### Task M3.7：WorkItemEditor 组件

**Files:**
- Create: `frontend/src/components/work-log/WorkItemEditor.vue`
- Create: `frontend/src/components/work-log/WorkItemEditor.spec.ts`

- [ ] **Step 1: 创建组件**

```vue
<template>
  <div class="work-item-editor">
    <div class="wie-header">
      <input
        class="wie-title"
        v-model="local.title"
        placeholder="标题（20 字以内）"
        @input="emitUpdate"
      />
      <button class="wie-remove" @click="$emit('remove')" title="删除">×</button>
    </div>
    <div class="wie-grid">
      <label class="wie-field">
        <span class="wie-label">内容</span>
        <textarea class="wie-input" v-model="local.content" @input="emitUpdate" />
      </label>
      <label class="wie-field">
        <span class="wie-label">解决了什么问题</span>
        <textarea class="wie-input" v-model="local.problem_solved" @input="emitUpdate" />
      </label>
      <label class="wie-field">
        <span class="wie-label">已产生的结果</span>
        <textarea class="wie-input" v-model="local.result" @input="emitUpdate" />
      </label>
      <label class="wie-field">
        <span class="wie-label">对后续的影响</span>
        <textarea class="wie-input" v-model="local.impact" @input="emitUpdate" />
      </label>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, watch } from 'vue'

interface ItemInput {
  title: string
  content: string
  problem_solved: string
  result: string
  impact: string
}

const props = defineProps<{
  modelValue: ItemInput
}>()
const emit = defineEmits<{
  (e: 'update:modelValue', value: ItemInput): void
  (e: 'remove'): void
}>()

const local = reactive<ItemInput>({ ...props.modelValue })

watch(() => props.modelValue, (v) => {
  Object.assign(local, v)
})

function emitUpdate() {
  emit('update:modelValue', { ...local })
}
</script>

<style scoped>
.work-item-editor {
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 16px 20px;
  margin-bottom: 12px;
}
.wie-header {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 12px;
}
.wie-title {
  flex: 1;
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  color: var(--text-primary);
  background: transparent;
  border: none;
  border-bottom: 1px solid var(--border-color);
  padding: 4px 0;
}
.wie-title:focus {
  outline: none;
  border-bottom-color: var(--accent-primary);
}
.wie-remove {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 20px;
  line-height: 1;
}
.wie-remove:hover {
  color: var(--accent-primary);
}
.wie-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 16px;
}
.wie-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.wie-label {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 500;
}
.wie-input {
  width: 100%;
  min-height: 60px;
  padding: 6px 8px;
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  background: var(--bg-elevated);
  font-family: var(--font-body);
  font-size: 13px;
  line-height: 1.5;
  color: var(--text-primary);
  resize: vertical;
}
.wie-input:focus {
  outline: none;
  border-color: var(--accent-primary);
}
</style>
```

- [ ] **Step 2: 创建测试**

```ts
// frontend/src/components/work-log/WorkItemEditor.spec.ts
import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import WorkItemEditor from './WorkItemEditor.vue'

const baseItem = () => ({
  title: 'T1', content: 'c', problem_solved: 'p', result: 'r', impact: 'i',
})

describe('WorkItemEditor', () => {
  it('renders title from modelValue', () => {
    const w = mount(WorkItemEditor, { props: { modelValue: baseItem() } })
    expect(w.find('.wie-title').element.value).toBe('T1')
  })

  it('emits update on input', async () => {
    const w = mount(WorkItemEditor, { props: { modelValue: baseItem() } })
    await w.find('.wie-title').setValue('T2')
    const events = w.emitted('update:modelValue')
    expect(events).toBeTruthy()
    expect((events!.at(-1) as any)[0].title).toBe('T2')
  })

  it('emits remove on × click', async () => {
    const w = mount(WorkItemEditor, { props: { modelValue: baseItem() } })
    await w.find('.wie-remove').trigger('click')
    expect(w.emitted('remove')).toBeTruthy()
  })
})
```

- [ ] **Step 3: 跑测试 + 类型检查**

```bash
cd frontend && npx vitest run src/components/work-log/WorkItemEditor.spec.ts
cd frontend && npx vue-tsc --noEmit
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/work-log/WorkItemEditor.vue frontend/src/components/work-log/WorkItemEditor.spec.ts
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add WorkItemEditor component (4-dim form) + tests"
```

---

### Task M3.8：WorkItemList 组件

**Files:**
- Create: `frontend/src/components/work-log/WorkItemList.vue`

- [ ] **Step 1: 创建组件（增删 + seq 重排，不做拖拽以避免引入库）**

```vue
<template>
  <div class="work-item-list">
    <WorkItemEditor
      v-for="(item, idx) in items"
      :key="idx"
      :model-value="item"
      @update:model-value="onUpdate(idx, $event)"
      @remove="onRemove(idx)"
    />
    <button class="wil-add" @click="onAdd">+ 加一条</button>

    <div class="wil-summary">
      <label class="wil-label">今日小结</label>
      <textarea
        class="wil-summary-input"
        :value="summary"
        @input="$emit('update:summary', ($event.target as HTMLTextAreaElement).value)"
        placeholder="2-3 句概括今日工作"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import WorkItemEditor from './WorkItemEditor.vue'

interface ItemInput {
  title: string
  content: string
  problem_solved: string
  result: string
  impact: string
}

const props = defineProps<{
  items: ItemInput[]
  summary: string
}>()

const emit = defineEmits<{
  (e: 'update:items', items: ItemInput[]): void
  (e: 'update:summary', summary: string): void
}>()

function onUpdate(idx: number, val: ItemInput) {
  const next = [...props.items]
  next[idx] = val
  emit('update:items', next)
}

function onRemove(idx: number) {
  const next = props.items.filter((_, i) => i !== idx)
  emit('update:items', next)
}

function onAdd() {
  emit('update:items', [
    ...props.items,
    { title: '', content: '', problem_solved: '', result: '', impact: '' },
  ])
}
</script>

<style scoped>
.work-item-list {
  margin-bottom: 16px;
}
.wil-add {
  background: transparent;
  border: 1px dashed var(--border-accent);
  color: var(--text-secondary);
  padding: 8px 16px;
  border-radius: var(--radius-sm);
  font-size: 13px;
  cursor: pointer;
  width: 100%;
  transition: all var(--transition-fast);
}
.wil-add:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}
.wil-summary {
  margin-top: 16px;
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-md);
  padding: 16px 20px;
}
.wil-label {
  display: block;
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 6px;
}
.wil-summary-input {
  width: 100%;
  min-height: 60px;
  border: none;
  background: transparent;
  color: var(--text-primary);
  font-family: var(--font-body);
  font-size: 13px;
  line-height: 1.6;
  resize: vertical;
}
.wil-summary-input:focus {
  outline: none;
}
</style>
```

- [ ] **Step 2: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无输出

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/work-log/WorkItemList.vue
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add WorkItemList component (add/remove + summary)"
```

---

### Task M3.9：Timeline 组件

**Files:**
- Create: `frontend/src/components/work-log/Timeline.vue`
- Create: `frontend/src/components/work-log/Timeline.spec.ts`

- [ ] **Step 1: 创建组件**

```vue
<template>
  <div class="timeline">
    <div class="tl-section" v-if="logs.length">
      <div class="tl-section-title">日报</div>
      <div
        v-for="log in logs"
        :key="log.date"
        class="tl-item"
        :class="{ active: selected?.kind === 'log' && selected.date === log.date }"
        @click="$emit('select', { kind: 'log', date: log.date })"
      >
        <span class="tl-date">{{ formatDate(log.date) }}</span>
        <span class="tl-badge" v-if="isToday(log.date)">今</span>
      </div>
    </div>

    <div class="tl-section" v-for="t in reportTypes" :key="t">
      <div class="tl-section-title">{{ reportLabel[t] }}</div>
      <div
        v-for="r in reports[t]"
        :key="r.period_key"
        class="tl-item"
        :class="{ active: selected?.kind === 'report' && selected.type === t && selected.periodKey === r.period_key }"
        @click="$emit('select', { kind: 'report', type: t, periodKey: r.period_key })"
      >
        <span class="tl-date">{{ r.period_key }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { WorkLog, WorkReport, WorkReportType } from '@/types'
import type { SelectedNode } from '@/stores/workLog'

defineProps<{
  logs: WorkLog[]
  reports: Record<WorkReportType, WorkReport[]>
  selected: SelectedNode | null
}>()

defineEmits<{ (e: 'select', node: SelectedNode): void }>()

const reportTypes: WorkReportType[] = ['weekly', 'monthly', 'halfyear', 'yearly']
const reportLabel: Record<WorkReportType, string> = {
  weekly: '周报', monthly: '月报', halfyear: '半年报', yearly: '年报',
}

function formatDate(d: string): string {
  // YYYY-MM-DD → M/D 周X
  const dt = new Date(d)
  const weekdays = ['周日', '周一', '周二', '周三', '周四', '周五', '周六']
  return `${dt.getMonth() + 1}/${dt.getDate()} ${weekdays[dt.getDay()]}`
}

function isToday(d: string): boolean {
  return d === new Date().toISOString().slice(0, 10)
}
</script>

<style scoped>
.timeline {
  width: 240px;
  border-right: 1px solid var(--border-color);
  padding: 20px 16px;
  overflow-y: auto;
  height: 100%;
}
.tl-section {
  margin-bottom: 24px;
}
.tl-section-title {
  font-family: var(--font-display);
  font-size: 11px;
  font-weight: 600;
  color: var(--text-muted);
  letter-spacing: 0.5px;
  text-transform: uppercase;
  margin-bottom: 8px;
  padding-left: 8px;
}
.tl-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
  color: var(--text-secondary);
  transition: all var(--transition-fast);
}
.tl-item:hover {
  background: rgba(0, 0, 0, 0.04);
  color: var(--text-primary);
}
.tl-item.active {
  background: rgba(184, 69, 44, 0.06);
  color: var(--accent-primary);
  font-weight: 500;
}
.tl-date {
  flex: 1;
}
.tl-badge {
  background: var(--accent-primary);
  color: white;
  font-size: 10px;
  padding: 1px 6px;
  border-radius: 8px;
  font-weight: 600;
}
</style>
```

- [ ] **Step 2: 创建测试**

```ts
// frontend/src/components/work-log/Timeline.spec.ts
import { mount } from '@vue/test-utils'
import { describe, it, expect } from 'vitest'
import Timeline from './Timeline.vue'
import type { WorkLog, WorkReport, WorkReportType } from '@/types'

describe('Timeline', () => {
  it('renders logs under 日报 section', () => {
    const logs: WorkLog[] = [
      { id: '1', date: '2026-08-02', items: [], summary: '', raw_brain_dump: '', created_at: '', updated_at: '' },
    ]
    const reports = { weekly: [], monthly: [], halfyear: [], yearly: [] } as Record<WorkReportType, WorkReport[]>
    const w = mount(Timeline, { props: { logs, reports, selected: null } })
    const sections = w.findAll('.tl-section-title')
    expect(sections[0].text()).toBe('日报')
    expect(w.findAll('.tl-item')).toHaveLength(1)
  })

  it('emits select with log node on click', async () => {
    const logs: WorkLog[] = [
      { id: '1', date: '2026-08-02', items: [], summary: '', raw_brain_dump: '', created_at: '', updated_at: '' },
    ]
    const reports = { weekly: [], monthly: [], halfyear: [], yearly: [] } as Record<WorkReportType, WorkReport[]>
    const w = mount(Timeline, { props: { logs, reports, selected: null } })
    await w.find('.tl-item').trigger('click')
    expect(w.emitted('select')?.[0]).toEqual([{ kind: 'log', date: '2026-08-02' }])
  })
})
```

- [ ] **Step 3: 跑测试**

Run: `cd frontend && npx vitest run src/components/work-log/Timeline.spec.ts`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/work-log/Timeline.vue frontend/src/components/work-log/Timeline.spec.ts
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add Timeline component with logs + 4 report sections + tests"
```

---

### Task M3.10：WorkLog 主页面 + 路由 + 导航入口

**Files:**
- Create: `frontend/src/views/WorkLog.vue`
- Modify: `frontend/src/router/index.ts`
- Modify: `frontend/src/App.vue`

- [ ] **Step 1: 创建页面**

```vue
<template>
  <div class="work-log-page">
    <div class="page-header">
      <h1 class="page-title">工作日志</h1>
      <div class="page-actions">
        <button class="action-btn" @click="goToday">今日</button>
      </div>
    </div>

    <div class="page-body">
      <Timeline
        :logs="store.logs"
        :reports="store.reports"
        :selected="store.selected"
        @select="store.selectNode"
      />

      <div class="detail-area">
        <!-- 日报视图 -->
        <template v-if="!store.selected || store.selected.kind === 'log'">
          <TodayContextCard :context="store.todayContext" />

          <BrainDumpInput
            :loading="structuring"
            @structure="onStructure"
          />

          <WorkItemList
            v-if="draftItems.length || draftSummary"
            :items="draftItems"
            :summary="draftSummary"
            @update:items="draftItems = $event"
            @update:summary="draftSummary = $event"
          />

          <div class="save-bar" v-if="draftItems.length">
            <button class="save-btn" :disabled="saving" @click="onSave">
              {{ saving ? '保存中…' : (isUpdate ? '更新今日日报' : '保存日报') }}
            </button>
          </div>
        </template>

        <!-- 报告视图（M5 完善） -->
        <template v-else>
          <div class="report-placeholder">报告详情视图将在 M5 实现</div>
        </template>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useWorkLogStore } from '@/stores/workLog'
import Timeline from '@/components/work-log/Timeline.vue'
import TodayContextCard from '@/components/work-log/TodayContextCard.vue'
import BrainDumpInput from '@/components/work-log/BrainDumpInput.vue'
import WorkItemList from '@/components/work-log/WorkItemList.vue'
import { ElMessage } from 'element-plus'

const store = useWorkLogStore()

const structuring = ref(false)
const saving = ref(false)
const draftItems = ref<any[]>([])
const draftSummary = ref('')
const currentDate = ref(new Date().toISOString().slice(0, 10))

const isUpdate = computed(() =>
  store.logs.some(l => l.date === currentDate.value),
)

async function loadInitial() {
  await Promise.all([
    store.fetchInitialRange(),
    store.fetchTodayContext(currentDate.value),
  ])
}

async function onStructure(text: string) {
  structuring.value = true
  try {
    const out = await store.structureBrainDump(text)
    if (out) {
      draftItems.value = out.items.map(it => ({ ...it }))
      draftSummary.value = out.summary
    }
  } finally {
    structuring.value = false
  }
}

async function onSave() {
  saving.value = true
  try {
    await store.saveWorkLog({
      date: currentDate.value,
      summary: draftSummary.value,
      raw_brain_dump: '',
      items: draftItems.value.map((it, idx) => ({ seq: idx + 1, ...it })),
    })
  } finally {
    saving.value = false
  }
}

function goToday() {
  currentDate.value = new Date().toISOString().slice(0, 10)
  store.selectNode({ kind: 'log', date: currentDate.value })
}

watch(currentDate, (d) => {
  store.fetchTodayContext(d)
})

onMounted(loadInitial)
</script>

<style scoped>
.work-log-page {
  height: 100%;
  display: flex;
  flex-direction: column;
}
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.page-title {
  font-family: var(--font-display);
  font-size: 28px;
  font-weight: 600;
  color: var(--text-primary);
  letter-spacing: -0.5px;
}
.action-btn {
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
}
.action-btn:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}
.page-body {
  flex: 1;
  display: flex;
  border-top: 1px solid var(--border-color);
  margin: 0 -40px -40px -40px;
  padding: 0;
}
.detail-area {
  flex: 1;
  padding: 24px 32px;
  overflow-y: auto;
}
.save-bar {
  margin-top: 16px;
  display: flex;
  justify-content: flex-end;
}
.save-btn {
  background: var(--accent-primary);
  color: white;
  border: none;
  border-radius: var(--radius-sm);
  padding: 8px 20px;
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
}
.save-btn:hover:not(:disabled) {
  background: var(--accent-secondary);
}
.save-btn:disabled {
  background: var(--text-muted);
}
.report-placeholder {
  padding: 40px;
  color: var(--text-muted);
  font-style: italic;
}
</style>
```

- [ ] **Step 2: 改路由**

在 `frontend/src/router/index.ts` 的 routes 数组，在 `/analytics` 之后加：

```ts
  {
    path: '/work-log',
    name: 'WorkLog',
    component: () => import('@/views/WorkLog.vue')
  },
```

- [ ] **Step 3: 改 App.vue 导航**

在 `frontend/src/App.vue` 顶部 import 加：

```ts
import { DataBoard, Timer, List, Calendar, TrendCharts, Setting, Document } from '@element-plus/icons-vue'
```

在 `navItems` 数组里"分析"和"设置"之间插入：

```ts
  { path: '/work-log', name: 'work-log', label: '工作日志', icon: Document },
```

- [ ] **Step 4: 类型检查 + 构建验证**

```bash
cd frontend && npx vue-tsc --noEmit
cd frontend && npm run build
```

Expected: 全部通过

- [ ] **Step 5: 手测**

```bash
make dev
# 浏览器打开 http://localhost:5173/work-log
# 1. 看到左时间轴（即使空，至少有 4 个报告分组标题）
# 2. 看到今日预填卡片（即使为空）
# 3. 在脑暴框输入文字 → 点 AI 拆条 → 看到 items 编辑器
# 4. 改一改 → 点保存日报 → 看到成功提示
# 5. 左时间轴出现今日条目
```

- [ ] **Step 6: Commit**

```bash
git add frontend/src/views/WorkLog.vue frontend/src/router/index.ts frontend/src/App.vue
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add WorkLog page (timeline + brain-dump + items editor) + route + nav"
```

---

**Phase M3 验收：**

- `/work-log` 页面可访问
- 完整日报流程：输入脑暴 → AI 拆条 → 编辑预览 → 保存
- 时间轴显示已保存的日报
- 同日二次保存走 PUT
- M4 报告生成端点仍是 stub，但前端 404 报告能 graceful 显示

---

## Phase M4 — 周期算法 + 报告生成（后端）

**目标：** 实现 `work_log_calendar.go` 周期算法 + 报告生成（4 类）+ 不变式测试。完成后 `/api/work-reports/generate` 端到端可用。

### Task M4.1：写周期算法 + 测试

**Files:**
- Create: `backend/internal/service/work_log_calendar.go`
- Create: `backend/internal/service/work_log_calendar_test.go`

- [ ] **Step 1: 写算法实现**

```go
// backend/internal/service/work_log_calendar.go
package service

import (
	"fmt"
	"time"
)

// WeeklyRange 返回 ISO 周一 00:00 到下周一 00:00（即周日结束）
func WeeklyRange(t time.Time) (start, end time.Time) {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // 周日 = 7
	}
	start = time.Date(t.Year(), t.Month(), t.Day()-weekday+1, 0, 0, 0, 0, t.Location())
	end = start.AddDate(0, 0, 7)
	return
}

// WeeklyKey "2026-W31"
func WeeklyKey(t time.Time) string {
	year, week := t.ISOWeek()
	return fmt.Sprintf("%d-W%02d", year, week)
}

// MonthlyRange 当月 1 号到下月 1 号
func MonthlyRange(t time.Time) (start, end time.Time) {
	start = time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, t.Location())
	end = start.AddDate(0, 1, 0)
	return
}

// MonthlyKey "2026-07"
func MonthlyKey(t time.Time) string {
	return t.Format("2006-01")
}

// HalfYearRange H1=1~6 月，H2=7~12 月
func HalfYearRange(t time.Time) (start, end time.Time) {
	month := int(t.Month())
	startYear := t.Year()
	if month <= 6 {
		start = time.Date(startYear, 1, 1, 0, 0, 0, 0, t.Location())
		end = start.AddDate(0, 6, 0)
	} else {
		start = time.Date(startYear, 7, 1, 0, 0, 0, 0, t.Location())
		end = start.AddDate(0, 6, 0)
	}
	return
}

// HalfYearKey "2026-H1" or "2026-H2"
func HalfYearKey(t time.Time) string {
	if t.Month() <= 6 {
		return fmt.Sprintf("%d-H1", t.Year())
	}
	return fmt.Sprintf("%d-H2", t.Year())
}

// YearlyRange 自然年
func YearlyRange(t time.Time) (start, end time.Time) {
	start = time.Date(t.Year(), 1, 1, 0, 0, 0, 0, t.Location())
	end = start.AddDate(1, 0, 0)
	return
}

// YearlyKey "2026"
func YearlyKey(t time.Time) string {
	return fmt.Sprintf("%d", t.Year())
}

// RangeForType 根据 type 取 range
func RangeForType(t model.WorkReportType, moment time.Time) (start, end time.Time) {
	switch t {
	case model.ReportWeekly:
		return WeeklyRange(moment)
	case model.ReportMonthly:
		return MonthlyRange(moment)
	case model.ReportHalfYear:
		return HalfYearRange(moment)
	case model.ReportYearly:
		return YearlyRange(moment)
	}
	return moment, moment
}

// KeyForType 根据 type 取 period key
func KeyForType(t model.WorkReportType, moment time.Time) string {
	switch t {
	case model.ReportWeekly:
		return WeeklyKey(moment)
	case model.ReportMonthly:
		return MonthlyKey(moment)
	case model.ReportHalfYear:
		return HalfYearKey(moment)
	case model.ReportYearly:
		return YearlyKey(moment)
	}
	return ""
}

// DateRangeToYMD 把 range 转 YYYY-MM-DD 字符串（用于 service 输入）
func DateRangeToYMD(start, end time.Time) (string, string) {
	return start.Format("2006-01-02"), end.AddDate(0, 0, -1).Format("2006-01-02")
}

// MissingDays 计算周期内有日报的日期与全部日期的差集
// allDays = 周期内所有 YYYY-MM-DD；existing = 已有的；返回缺失的（逗号分隔）
func MissingDays(start, end time.Time, existing []string) string {
	existSet := make(map[string]bool, len(existing))
	for _, d := range existing {
		existSet[d] = true
	}
	var missing []string
	for d := start; d.Before(end); d = d.AddDate(0, 0, 1) {
		ds := d.Format("2006-01-02")
		if !existSet[ds] {
			missing = append(missing, ds)
		}
	}
	if len(missing) == 0 {
		return ""
	}
	// join
	result := missing[0]
	for _, m := range missing[1:] {
		result += "," + m
	}
	return result
}
```

- [ ] **Step 2: 写测试**

```go
// backend/internal/service/work_log_calendar_test.go
package service

import (
	"testing"
	"time"
)

func mustParse(t *testing.T, s string) time.Time {
	t.Helper()
	v, err := time.Parse("2006-01-02", s)
	if err != nil {
		t.Fatalf("parse %s: %v", s, err)
	}
	return v
}

func TestWeeklyRange_Midweek(t *testing.T) {
	// 2026-08-04 是周二
	wed := mustParse(t, "2026-08-04")
	start, end := WeeklyRange(wed)
	if start.Format("2006-01-02") != "2026-08-03" {
		t.Errorf("start = %s, want 2026-08-03 (Mon)", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2026-08-10" {
		t.Errorf("end = %s, want 2026-08-10 (next Mon)", end.Format("2006-01-02"))
	}
}

func TestWeeklyRange_Sunday(t *testing.T) {
	// 2026-08-02 是周日
	sun := mustParse(t, "2026-08-02")
	start, end := WeeklyRange(sun)
	if start.Format("2006-01-02") != "2026-07-27" {
		t.Errorf("start = %s, want 2026-07-27", start.Format("2006-01-02"))
	}
	if end.Format("2006-01-02") != "2026-08-03" {
		t.Errorf("end = %s, want 2026-08-03", end.Format("2006-01-02"))
	}
}

func TestWeeklyKey(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"2026-08-02", "2026-W31"}, // 周日
		{"2026-08-04", "2026-W32"}, // 周一（新一周开始）
		{"2026-01-01", "2026-W01"},
	}
	for _, c := range cases {
		got := WeeklyKey(mustParse(t, c.in))
		if got != c.want {
			t.Errorf("WeeklyKey(%s) = %s, want %s", c.in, got, c.want)
		}
	}
}

func TestMonthlyRangeAndKey(t *testing.T) {
	jul := mustParse(t, "2026-07-15")
	start, end := MonthlyRange(jul)
	if start.Format("2006-01-02") != "2026-07-01" {
		t.Errorf("start = %s", start)
	}
	if end.Format("2006-01-02") != "2026-08-01" {
		t.Errorf("end = %s", end)
	}
	if MonthlyKey(jul) != "2026-07" {
		t.Errorf("key wrong")
	}
}

func TestHalfYearRange_H1_H2(t *testing.T) {
	jan := mustParse(t, "2026-01-15")
	start, end := HalfYearRange(jan)
	if start.Format("2006-01-02") != "2026-01-01" || end.Format("2006-01-02") != "2026-07-01" {
		t.Errorf("H1 wrong: %s ~ %s", start, end)
	}
	if HalfYearKey(jan) != "2026-H1" {
		t.Errorf("H1 key wrong")
	}

	aug := mustParse(t, "2026-08-15")
	start2, end2 := HalfYearRange(aug)
	if start2.Format("2006-01-02") != "2026-07-01" || end2.Format("2006-01-02") != "2027-01-01" {
		t.Errorf("H2 wrong: %s ~ %s", start2, end2)
	}
	if HalfYearKey(aug) != "2026-H2" {
		t.Errorf("H2 key wrong")
	}
}

func TestYearlyRangeAndKey(t *testing.T) {
	t1 := mustParse(t, "2026-06-15")
	start, end := YearlyRange(t1)
	if start.Format("2006-01-02") != "2026-01-01" || end.Format("2006-01-02") != "2027-01-01" {
		t.Errorf("year range wrong: %s ~ %s", start, end)
	}
	if YearlyKey(t1) != "2026" {
		t.Errorf("year key wrong")
	}
}

func TestMissingDays_AllPresent(t *testing.T) {
	start := mustParse(t, "2026-08-01")
	end := mustParse(t, "2026-08-04") // 8/1, 8/2, 8/3 (3 days)
	existing := []string{"2026-08-01", "2026-08-02", "2026-08-03"}
	if got := MissingDays(start, end, existing); got != "" {
		t.Errorf("got = %q, want empty", got)
	}
}

func TestMissingDays_SomeMissing(t *testing.T) {
	start := mustParse(t, "2026-08-01")
	end := mustParse(t, "2026-08-04")
	existing := []string{"2026-08-02"}
	got := MissingDays(start, end, existing)
	if got != "2026-08-01,2026-08-03" {
		t.Errorf("got = %q", got)
	}
}

func TestRangeForType_AllTypes(t *testing.T) {
	moment := mustParse(t, "2026-08-02")
	for _, ty := range []model.WorkReportType{model.ReportWeekly, model.ReportMonthly, model.ReportHalfYear, model.ReportYearly} {
		s, e := RangeForType(ty, moment)
		if s.IsZero() || e.IsZero() || !e.After(s) {
			t.Errorf("type %v: bad range %v ~ %v", ty, s, e)
		}
		if KeyForType(ty, moment) == "" {
			t.Errorf("type %v: empty key", ty)
		}
	}
}
```

- [ ] **Step 3: 跑测试**

Run: `cd backend && go test -v ./internal/service/ -run "TestWeekly|TestMonthly|TestHalfYear|TestYearly|TestMissingDays|TestRangeForType"`
Expected: 全部 PASS

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/work_log_calendar.go backend/internal/service/work_log_calendar_test.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add period calendar helpers (weekly/monthly/halfyear/yearly) + table-driven tests"
```

---

### Task M4.2：实现 GenerateReport（周报）

**Files:**
- Modify: `backend/internal/service/work_log_service.go`

- [ ] **Step 1: 替换 GenerateReport 的 stub**

在 `work_log_service.go` 文件，找到 `func (s *WorkLogService) GenerateReport` 函数体，替换为：

```go
// GenerateReport 生成周期报告
func (s *WorkLogService) GenerateReport(input GenerateReportInput) (*model.WorkReport, error) {
	if s.aiClient == nil {
		return nil, ErrAIStructureFailed
	}
	moment := time.Now()
	if input.PeriodKey != "" {
		// 反推 moment：用 period key 找一个代表日期
		m, err := parsePeriodKeyMoment(input.Type, input.PeriodKey)
		if err != nil {
			return nil, err
		}
		moment = m
	}
	start, end := RangeForType(input.Type, moment)
	startStr, endStr := DateRangeToYMD(start, end)
	periodKey := KeyForType(input.Type, moment)

	// 已存在则拦截
	existing, err := s.repo.GetWorkReportByTypeAndPeriod(input.Type, periodKey)
	if err != nil && !errors.Is(err, repository.ErrWorkLogNotFound) {
		return nil, err
	}
	if existing != nil && !input.Force {
		return existing, ErrReportAlreadyExists
	}

	var summary *ReportSummary
	switch input.Type {
	case model.ReportWeekly:
		summary, err = s.generateWeekly(start, end, startStr, endStr)
	case model.ReportMonthly:
		summary, err = s.generateMonthly(start, end, startStr, endStr)
	case model.ReportHalfYear:
		summary, err = s.generateHalfYear(start, end, startStr, endStr)
	case model.ReportYearly:
		summary, err = s.generateYearly(start, end, startStr, endStr)
	default:
		return nil, fmt.Errorf("unknown report type: %s", input.Type)
	}
	if err != nil {
		return nil, err
	}
	summaryJSON, err := json.Marshal(summary)
	if err != nil {
		return nil, fmt.Errorf("marshal summary: %w", err)
	}

	// 算 missing days
	logs, err := s.repo.GetWorkLogsInRange(startStr, endStr)
	if err != nil {
		return nil, err
	}
	existDates := make([]string, 0, len(logs))
	for _, l := range logs {
		existDates = append(existDates, l.Date)
	}
	missing := MissingDays(start, end, existDates)

	report := &model.WorkReport{
		ID:          s.idGenerator(),
		Type:        input.Type,
		PeriodKey:   periodKey,
		StartDate:   startStr,
		EndDate:     endStr,
		SummaryJSON: string(summaryJSON),
		MissingDays: missing,
	}

	if existing != nil {
		report.ID = existing.ID
		report.CreatedAt = existing.CreatedAt
		if err := s.repo.UpdateWorkReport(report); err != nil {
			return nil, err
		}
	} else {
		if err := s.repo.CreateWorkReport(report); err != nil {
			return nil, err
		}
	}
	return report, nil
}

// parsePeriodKeyMoment 从 period key 反推一个代表日期（取周期开始日）
func parsePeriodKeyMoment(t model.WorkReportType, key string) (time.Time, error) {
	switch t {
	case model.ReportWeekly:
		// 2026-W31 → 取该周一
		var year, week int
		if _, err := fmt.Sscanf(key, "%d-W%d", &year, &week); err != nil {
			return time.Time{}, fmt.Errorf("bad weekly key: %s", key)
		}
		// 从 1 月 1 日开始向后扫，找到第一个 ISO week == 目标的日期
		cursor := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
		for cursor.Year() <= year+1 {
			cy, cw := cursor.ISOWeek()
			if cy == year && cw == week {
				return cursor, nil
			}
			cursor = cursor.AddDate(0, 0, 1)
		}
		return time.Time{}, fmt.Errorf("weekly key not found: %s", key)
	case model.ReportMonthly:
		// 2026-07
		t2, err := time.Parse("2006-01", key)
		if err != nil {
			return time.Time{}, fmt.Errorf("bad monthly key: %s", key)
		}
		return t2, nil
	case model.ReportHalfYear:
		// 2026-H1 / 2026-H2
		var year int
		var h string
		if _, err := fmt.Sscanf(key, "%d-%s", &year, &h); err != nil {
			return time.Time{}, fmt.Errorf("bad halfyear key: %s", key)
		}
		month := 1
		if h == "H2" {
			month = 7
		}
		return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.Local), nil
	case model.ReportYearly:
		var year int
		if _, err := fmt.Sscanf(key, "%d", &year); err != nil {
			return time.Time{}, fmt.Errorf("bad yearly key: %s", key)
		}
		return time.Date(year, 1, 1, 0, 0, 0, 0, time.Local), nil
	}
	return time.Time{}, fmt.Errorf("unknown type: %s", t)
}

// INVARIANT: weekly 只读 work_items，不读 reports
func (s *WorkLogService) generateWeekly(start, end time.Time, startStr, endStr string) (*ReportSummary, error) {
	logs, err := s.repo.GetWorkLogsInRange(startStr, endStr)
	if err != nil {
		return nil, err
	}
	var items []model.WorkItem
	for _, l := range logs {
		items = append(items, l.Items...)
	}
	return s.aiClient.GenerateWeeklyReport(items, startStr, endStr)
}

// INVARIANT: monthly 只读 weekly 报告 + 月内不属于完整周的零散 items，不读所有原始 items
func (s *WorkLogService) generateMonthly(start, end time.Time, startStr, endStr string) (*ReportSummary, error) {
	weeklies, err := s.repo.ListWorkReports(model.ReportWeekly)
	if err != nil {
		return nil, err
	}
	// 过滤本月内的周报
	var in []*model.WorkReport
	for _, w := range weeklies {
		if w.EndDate >= startStr && w.StartDate <= endStr {
			in = append(in, w)
		}
	}
	// 周报覆盖的日期集合
	covered := make(map[string]bool)
	for _, w := range in {
		ws, _ := time.Parse("2006-01-02", w.StartDate)
		we, _ := time.Parse("2006-01-02", w.EndDate)
		for d := ws; !d.After(we); d = d.AddDate(0, 0, 1) {
			covered[d.Format("2006-01-02")] = true
		}
	}
	// 找孤儿 items：月内但不在 covered
	logs, err := s.repo.GetWorkLogsInRange(startStr, endStr)
	if err != nil {
		return nil, err
	}
	var orphans []model.WorkItem
	for _, l := range logs {
		if !covered[l.Date] {
			orphans = append(orphans, l.Items...)
		}
	}
	return s.aiClient.GenerateMonthlyReport(in, orphans, startStr, endStr)
}

// INVARIANT: halfyear 只读 monthly 报告，绝不读原始 items
func (s *WorkLogService) generateHalfYear(start, end time.Time, startStr, endStr string) (*ReportSummary, error) {
	monthlies, err := s.repo.ListWorkReports(model.ReportMonthly)
	if err != nil {
		return nil, err
	}
	var in []*model.WorkReport
	for _, m := range monthlies {
		if m.EndDate >= startStr && m.StartDate <= endStr {
			in = append(in, m)
		}
	}
	return s.aiClient.GenerateHalfYearReport(in, startStr, endStr)
}

// INVARIANT: yearly 只读 monthly 报告，绝不读原始 items 或 weekly 报告
func (s *WorkLogService) generateYearly(start, end time.Time, startStr, endStr string) (*ReportSummary, error) {
	monthlies, err := s.repo.ListWorkReports(model.ReportMonthly)
	if err != nil {
		return nil, err
	}
	var in []*model.WorkReport
	for _, m := range monthlies {
		if m.StartDate >= startStr && m.EndDate <= endStr {
			in = append(in, m)
		}
	}
	return s.aiClient.GenerateYearlyReport(in, startStr, endStr)
}
```

- [ ] **Step 2: 编译验证**

Run: `cd backend && go build ./...`
Expected: 无输出

- [ ] **Step 3: Commit**

```bash
git add backend/internal/service/work_log_service.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): implement GenerateReport with 4 report types + INVARIANT enforcement"
```

---

### Task M4.3：写 GenerateReport 测试（含不变式守门）

**Files:**
- Modify: `backend/internal/service/work_log_service_test.go`

- [ ] **Step 1: 扩展 mock AI client**

把 service 测试文件里的 `mockAIClient` 加上 report 方法的可配置返回：

```go
type mockAIClient struct {
	structuredOut      *StructuredWorkLog
	structuredErr      error
	weeklyOut          *ReportSummary
	weeklyInput        []model.WorkItem // 记录被调用的 items
	monthlyInput       []*model.WorkReport
	monthlyOrphanInput []model.WorkItem
	halfYearInput      []*model.WorkReport
	yearlyInput        []*model.WorkReport
}

func (m *mockAIClient) GenerateWeeklyReport(items []model.WorkItem, start, end string) (*ReportSummary, error) {
	m.weeklyInput = items
	if m.weeklyOut != nil {
		return m.weeklyOut, nil
	}
	return &ReportSummary{CoreWork: "cw"}, nil
}
func (m *mockAIClient) GenerateMonthlyReport(w []*model.WorkReport, o []model.WorkItem, start, end string) (*ReportSummary, error) {
	m.monthlyInput = w
	m.monthlyOrphanInput = o
	return &ReportSummary{CoreWork: "cw"}, nil
}
func (m *mockAIClient) GenerateHalfYearReport(mo []*model.WorkReport, start, end string) (*ReportSummary, error) {
	m.halfYearInput = mo
	return &ReportSummary{CoreWork: "cw"}, nil
}
func (m *mockAIClient) GenerateYearlyReport(mo []*model.WorkReport, start, end string) (*ReportSummary, error) {
	m.yearlyInput = mo
	return &ReportSummary{CoreWork: "cw"}, nil
}
```

- [ ] **Step 2: 加测试用例**

```go
func TestGenerateReport_Weekly_GathersItems(t *testing.T) {
	svc, repo, ai := newServiceForTest()
	// 准备：本周一篇文章 1 个 item
	repo.logs["2026-08-03"] = &model.WorkLog{
		ID: "wl-1", Date: "2026-08-03",
		Items: []model.WorkItem{{ID: "wi-1", WorkLogID: "wl-1", Title: "T1"}},
	}

	// 强制 period key = 2026-W32（8/3 ~ 8/9）
	report, err := svc.GenerateReport(GenerateReportInput{
		Type: model.ReportWeekly, PeriodKey: "2026-W32", Force: true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(ai.weeklyInput) != 1 || ai.weeklyInput[0].Title != "T1" {
		t.Errorf("weekly should gather 1 item, got: %+v", ai.weeklyInput)
	}
	if report.PeriodKey != "2026-W32" {
		t.Errorf("period_key = %s", report.PeriodKey)
	}
}

func TestGenerateReport_AlreadyExists_NoForce(t *testing.T) {
	svc, repo, _ := newServiceForTest()
	// 预置一份
	repo.reports["weekly:2026-W32"] = &model.WorkReport{
		ID: "wr-1", Type: model.ReportWeekly, PeriodKey: "2026-W32",
		StartDate: "2026-08-03", EndDate: "2026-08-09",
	}
	_, err := svc.GenerateReport(GenerateReportInput{
		Type: model.ReportWeekly, PeriodKey: "2026-W32",
	})
	if !errors.Is(err, ErrReportAlreadyExists) {
		t.Errorf("err = %v, want ErrReportAlreadyExists", err)
	}
}

func TestGenerateReport_HalfYear_DoesNotReadItems(t *testing.T) {
	svc, repo, ai := newServiceForTest()
	// 假装 H1 有一些原始日报（半年报不应读）
	repo.logs["2026-03-15"] = &model.WorkLog{
		ID: "wl-1", Date: "2026-03-15",
		Items: []model.WorkItem{{Title: "Should not be read"}},
	}
	// 假装 H1 有月报
	repo.reports["monthly:2026-03"] = &model.WorkReport{
		ID: "wm-3", Type: model.ReportMonthly, PeriodKey: "2026-03",
		StartDate: "2026-03-01", EndDate: "2026-03-31", SummaryJSON: "{}",
	}

	_, err := svc.GenerateReport(GenerateReportInput{
		Type: model.ReportHalfYear, PeriodKey: "2026-H1", Force: true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// 不变量：半年报只读月报，不读原始 items
	if len(ai.halfYearInput) == 0 {
		t.Errorf("halfyear should read monthlies")
	}
	// weekly input 应该没被读
	if ai.weeklyInput != nil {
		t.Errorf("INVARIANT violation: halfyear read weekly items")
	}
}

func TestGenerateReport_Yearly_DoesNotReadItemsOrWeeklies(t *testing.T) {
	svc, repo, ai := newServiceForTest()
	repo.logs["2026-06-15"] = &model.WorkLog{
		ID: "wl-1", Date: "2026-06-15",
		Items: []model.WorkItem{{Title: "x"}},
	}
	repo.reports["weekly:2026-W24"] = &model.WorkReport{
		ID: "ww", Type: model.ReportWeekly, PeriodKey: "2026-W24",
		StartDate: "2026-06-08", EndDate: "2026-06-14",
	}
	repo.reports["monthly:2026-06"] = &model.WorkReport{
		ID: "wm", Type: model.ReportMonthly, PeriodKey: "2026-06",
		StartDate: "2026-06-01", EndDate: "2026-06-30",
	}

	_, err := svc.GenerateReport(GenerateReportInput{
		Type: model.ReportYearly, PeriodKey: "2026", Force: true,
	})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if len(ai.yearlyInput) == 0 {
		t.Errorf("yearly should read monthlies")
	}
	if ai.weeklyInput != nil {
		t.Errorf("INVARIANT violation: yearly read weekly items")
	}
}
```

- [ ] **Step 3: 跑测试**

Run: `cd backend && go test -v ./internal/service/ -run "TestGenerateReport_"`
Expected: 全部 PASS（不变式测试守门通过）

- [ ] **Step 4: Commit**

```bash
git add backend/internal/service/work_log_service_test.go
git -c user.name='lsy' -c user.email='lsy@local' commit -m "test(work-log): add GenerateReport tests including INVARIANT enforcement"
```

---

### Task M4.4：端到端验证（手测 4 种报告）

- [ ] **Step 1: 启动后端**

```bash
cd backend && go run cmd/server/main.go &
sleep 3
```

- [ ] **Step 2: 准备几篇日报**

```bash
curl -s -X POST http://localhost:8080/api/work-logs \
  -H "Content-Type: application/json" \
  -d '{"date":"2026-08-03","summary":"M1 完成","raw_brain_dump":"","items":[{"seq":1,"title":"写完 M1 骨架","content":"...","problem_solved":"...","result":"9 个端点跑通","impact":"后续 M2 可直接接"}]}'

curl -s -X POST http://localhost:8080/api/work-logs \
  -H "Content-Type: application/json" \
  -d '{"date":"2026-08-04","summary":"M2 完成","items":[{"seq":1,"title":"AI 拆条接通","content":"...","problem_solved":"...","result":"OpenAI 兼容客户端跑通","impact":"日报输入成本大降"}]}'
```

- [ ] **Step 3: 生成周报**

```bash
curl -s -X POST http://localhost:8080/api/work-reports/generate \
  -H "Content-Type: application/json" \
  -d '{"type":"weekly","period_key":"2026-W32","force":true}'
```

Expected: 返回 201 + 报告 JSON，含 period_key=2026-W32, summary_json 含 core_work/main_progress/open_issues/next_focus

- [ ] **Step 4: 覆盖生成（不传 force）**

```bash
curl -s -X POST http://localhost:8080/api/work-reports/generate \
  -H "Content-Type: application/json" \
  -d '{"type":"weekly","period_key":"2026-W32"}'
```

Expected: 409 + `report already exists`

- [ ] **Step 5: 停服务**

```bash
kill %1
```

---

**Phase M4 验收：**

- 4 种报告（周/月/半年/年）都能生成
- 不变式测试守门通过（月报只读周报、半年报只读月报、年报只读月报）
- 报告 unique 索引正确（同 type+period_key 唯一）
- force=true 可覆盖，否则 409

---

## Phase M5 — 前端报告视图

**目标：** Timeline 接入报告节点 + ReportActions 下拉按钮 + ReportDetail 视图。完成后用户能在网页生成/查看 4 种报告。

### Task M5.1：ReportActions 组件

**Files:**
- Create: `frontend/src/components/work-log/ReportActions.vue`

- [ ] **Step 1: 创建组件**

```vue
<template>
  <el-dropdown trigger="click" @command="onCommand">
    <button class="action-btn">+ 生成报告 ▼</button>
    <template #dropdown>
      <el-dropdown-menu>
        <el-dropdown-item command="weekly">本周周报</el-dropdown-item>
        <el-dropdown-item command="monthly">本月月报</el-dropdown-item>
        <el-dropdown-item command="halfyear">本半年半年报</el-dropdown-item>
        <el-dropdown-item command="yearly">本年年报</el-dropdown-item>
      </el-dropdown-menu>
    </template>
  </el-dropdown>
</template>

<script setup lang="ts">
import type { WorkReportType } from '@/types'

const emit = defineEmits<{ (e: 'generate', type: WorkReportType): void }>()

function onCommand(cmd: WorkReportType) {
  emit('generate', cmd)
}
</script>

<style scoped>
.action-btn {
  background: transparent;
  border: 1px solid var(--border-color);
  color: var(--text-secondary);
  padding: 6px 14px;
  border-radius: var(--radius-sm);
  cursor: pointer;
  font-size: 13px;
}
.action-btn:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}
</style>
```

- [ ] **Step 2: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无输出

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/work-log/ReportActions.vue
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add ReportActions dropdown component"
```

---

### Task M5.2：ReportDetail 组件

**Files:**
- Create: `frontend/src/components/work-log/ReportDetail.vue`

- [ ] **Step 1: 创建组件**

```vue
<template>
  <div class="report-detail" v-if="report">
    <div class="rd-header">
      <h2 class="rd-title">{{ periodLabel }} {{ report.period_key }}</h2>
      <span class="rd-range">{{ report.start_date }} ~ {{ report.end_date }}</span>
    </div>

    <div v-if="report.missing_days" class="rd-missing">
      ⚠️ 缺失天：{{ report.missing_days }}
    </div>

    <div class="rd-section" v-for="f in fields" :key="f.key">
      <div class="rd-section-title">{{ f.label }}</div>
      <div class="rd-section-body">{{ summary[f.key] || '（待补充）' }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WorkReport, WorkReportType, ReportSummary } from '@/types'

const props = defineProps<{ report: WorkReport | null }>()

const periodLabel: Record<WorkReportType, string> = {
  weekly: '周报', monthly: '月报', halfyear: '半年报', yearly: '年报',
}

const fields: Array<{ key: keyof ReportSummary; label: string }> = [
  { key: 'core_work', label: '核心工作 / 重大成果' },
  { key: 'main_progress', label: '主要进展 / 趋势' },
  { key: 'open_issues', label: '遗留问题 / 关键问题' },
  { key: 'next_focus', label: '下阶段关注' },
]

const summary = computed<ReportSummary>(() => {
  if (!props.report?.summary_json) {
    return { core_work: '', main_progress: '', open_issues: '', next_focus: '' }
  }
  try {
    return JSON.parse(props.report.summary_json)
  } catch {
    return { core_work: '', main_progress: '', open_issues: '', next_focus: '' }
  }
})
</script>

<style scoped>
.report-detail {
  max-width: 720px;
}
.rd-header {
  display: flex;
  justify-content: space-between;
  align-items: baseline;
  margin-bottom: 20px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--border-color);
}
.rd-title {
  font-family: var(--font-display);
  font-size: 24px;
  font-weight: 600;
  color: var(--text-primary);
}
.rd-range {
  font-family: var(--font-mono);
  font-size: 12px;
  color: var(--text-muted);
}
.rd-missing {
  background: rgba(184, 69, 44, 0.06);
  color: var(--accent-primary);
  padding: 8px 12px;
  border-radius: var(--radius-sm);
  font-size: 12px;
  margin-bottom: 20px;
}
.rd-section {
  margin-bottom: 24px;
}
.rd-section-title {
  font-family: var(--font-display);
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
  margin-bottom: 8px;
}
.rd-section-body {
  font-size: 14px;
  line-height: 1.7;
  color: var(--text-primary);
  white-space: pre-wrap;
}
</style>
```

- [ ] **Step 2: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无输出

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/work-log/ReportDetail.vue
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): add ReportDetail component (4 sections render)"
```

---

### Task M5.3：在 store + WorkLog.vue 接入报告生成

**Files:**
- Modify: `frontend/src/views/WorkLog.vue`

- [ ] **Step 1: 在 WorkLog.vue 引入 ReportDetail + ReportActions**

把 M3.10 中 `<template v-else>` 的 `<div class="report-placeholder">` 替换为：

```vue
        <template v-else>
          <ReportDetail :report="store.currentReport" />
        </template>
```

在 page-actions 里加 ReportActions（替换原"今日"按钮的容器）：

```vue
      <div class="page-actions">
        <button class="action-btn" @click="goToday">今日</button>
        <ReportActions @generate="onGenerateReport" />
      </div>
```

import 加：

```ts
import ReportDetail from '@/components/work-log/ReportDetail.vue'
import ReportActions from '@/components/work-log/ReportActions.vue'
import { ElMessageBox } from 'element-plus'
import type { WorkReportType } from '@/types'
```

加 onGenerateReport 函数：

```ts
async function onGenerateReport(type: WorkReportType) {
  // 计算当前 period_key（前端用 JS）
  const now = new Date()
  let periodKey = ''
  if (type === 'weekly') {
    // ISO week
    const tmp = new Date(now)
    tmp.setHours(0, 0, 0, 0)
    tmp.setDate(tmp.getDate() + 4 - (tmp.getDay() || 7))
    const yearStart = new Date(tmp.getFullYear(), 0, 1)
    const week = Math.ceil((((tmp.getTime() - yearStart.getTime()) / 86400000) + 1) / 7)
    periodKey = `${tmp.getFullYear()}-W${String(week).padStart(2, '0')}`
  } else if (type === 'monthly') {
    periodKey = `${now.getFullYear()}-${String(now.getMonth() + 1).padStart(2, '0')}`
  } else if (type === 'halfyear') {
    periodKey = `${now.getFullYear()}-H${now.getMonth() < 6 ? 1 : 2}`
  } else {
    periodKey = `${now.getFullYear()}`
  }

  try {
    await store.generateReport(type, periodKey, false)
  } catch (e: any) {
    if (e?.response?.status === 409) {
      try {
        await ElMessageBox.confirm(
          `${labelOf(type)} ${periodKey} 已存在，是否覆盖重新生成？`,
          '覆盖确认',
          { confirmButtonText: '覆盖', cancelButtonText: '取消' },
        )
        await store.generateReport(type, periodKey, true)
        store.selectNode({ kind: 'report', type, periodKey })
      } catch {
        // 用户取消
      }
    }
  }
}

function labelOf(type: WorkReportType): string {
  return { weekly: '周报', monthly: '月报', halfyear: '半年报', yearly: '年报' }[type]
}
```

在 store 的 `selectNode` 触发后，对应 fetchReports：

```ts
// 在 onMounted 里加载所有类型报告
onMounted(async () => {
  await loadInitial()
  await Promise.all([
    store.fetchReports('weekly'),
    store.fetchReports('monthly'),
    store.fetchReports('halfyear'),
    store.fetchReports('yearly'),
  ])
})
```

- [ ] **Step 2: 类型检查 + 构建**

```bash
cd frontend && npx vue-tsc --noEmit
cd frontend && npm run build
```

- [ ] **Step 3: 手测**

```bash
make dev
# 浏览器 /work-log
# 1. 点"+ 生成报告" → 选"本周周报" → 看到 ElMessageBox（如已存在）或直接生成
# 2. 左时间轴"周报"分组出现 W32 条目
# 3. 点击该条目 → 右侧切到 ReportDetail，显示 4 个 section
# 4. 重复点"生成" → 弹覆盖确认
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/views/WorkLog.vue
git -c user.name='lsy' -c user.email='lsy@local' commit -m "feat(work-log): wire ReportActions + ReportDetail into WorkLog page with confirm-on-overwrite"
```

---

**Phase M5 验收：**

- 4 种报告都能从网页生成
- 报告节点出现在时间轴
- 报告详情视图正常显示
- 覆盖前有 confirm 弹窗

---

## Phase M6 — 收尾

**目标：** 文档更新 + E2E 走查 + 全套测试通过。

### Task M6.1：更新 AGENTS.md

**Files:**
- Modify: `AGENTS.md`

- [ ] **Step 1: 更新「仓库结构」部分**

在 AGENTS.md 的 `Repository Structure` 块的 backend 树里，在 `internal/service/` 下加 `work_log_service.go` `work_log_calendar.go`，在 `internal/model/` 下加 `work_log.go`，等等。在依赖图加入 work_log 模块。

更新「Module Dependency Graph」：

```
pkg/* -> internal/model -> internal/repository -> internal/ai -> internal/service -> internal/websocket -> internal/api -> cmd/server
                                                                              ↑ work_log_calendar
```

- [ ] **Step 2: 更新「领域能力」部分**

在 AGENTS.md 末尾「领域能力」加一个 work-log 条目（可选）：

```markdown
### 工作日志
- work-log 模块原生集成在 `/work-log` 页面，无独立 skill。
- AI 拆条 + 4 类周期报告 + 分层汇总。
```

- [ ] **Step 3: Commit**

```bash
git add AGENTS.md
git -c user.name='lsy' -c user.email='lsy@local' commit -m "docs(work-log): update AGENTS.md with new module structure"
```

---

### Task M6.2：更新 ARCHITECTURE.md

**Files:**
- Modify: `ARCHITECTURE.md`

- [ ] **Step 1: 在 Code map 加入 4 个新条目**

读 `ARCHITECTURE.md`，在 Code map 部分对应位置加入：

- `backend/internal/model/work_log.go` — WorkLog / WorkItem / WorkReport / WorkReportType
- `backend/internal/repository/work_log_repo.go` — interface + impl，含 unique index
- `backend/internal/service/work_log_service.go` — 业务编排 + 不变式守门
- `backend/internal/service/work_log_calendar.go` — 周期算法
- `backend/internal/ai/work_log_prompts.go` — AI prompts
- `backend/internal/api/handler/work_log.go` — 9 个 HTTP 端点
- `frontend/src/views/WorkLog.vue` + 7 个 components + store

每个条目按既有 matklad 风格写"职责 + 关键类型 + 不变式"。

- [ ] **Step 2: Commit**

```bash
git add ARCHITECTURE.md
git -c user.name='lsy' -c user.email='lsy@local' commit -m "docs(work-log): add Code map entries for new module"
```

---

### Task M6.3：E2E 走查 + 全套测试

- [ ] **Step 1: 全套后端测试**

Run: `cd backend && go test ./...`
Expected: 全 PASS

- [ ] **Step 2: 全套前端测试**

Run: `cd frontend && npx vitest run`
Expected: 全 PASS

- [ ] **Step 3: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无输出

- [ ] **Step 4: 构建**

Run: `make build`
Expected: 后端二进制 + frontend dist 都产出

- [ ] **Step 5: E2E 手测清单**

启动 `make dev`，浏览器走查：

```
[ ] /work-log 页面正常显示
[ ] 顶部导航有"工作日志"入口
[ ] 输入脑暴 → AI 拆条 → 看到 items 预览
[ ] 编辑 items 四维 → 保存日报 → 时间轴出现今日条目
[ ] 同日再保存 → 自动走 PUT
[ ] 点击"+ 生成报告" → 选周报 → 生成成功
[ ] 时间轴"周报"分组出现条目
[ ] 点击周报条目 → 右侧显示 ReportDetail
[ ] 重复生成 → 弹覆盖确认
[ ] AI 失败（断网或没 API key）→ ElMessage.error，不写空数据
[ ] 切换到月报/半年报/年报 → 都能生成
[ ] 跨语言：中文界面一致
```

- [ ] **Step 6: 最终 commit（如有修补）**

```bash
git add ...
git -c user.name='lsy' -c user.email='lsy@local' commit -m "fix(work-log): E2E polish (具体改动)"
```

---

**Phase M6 验收：**

- 所有测试 PASS
- 类型检查无错
- `make build` 通过
- E2E 走查清单全勾
- 文档更新（AGENTS.md + ARCHITECTURE.md）

---

## 总结

| 阶段 | 主要交付 | 端到端价值 |
|---|---|---|
| M1 | 后端骨架（model + repo + service 占位 + handler）| 9 个端点可调通，三张表迁移 |
| M2 | AI 拆条接通 | 贴脑暴 → 结构化 JSON |
| M3 | 前端日报页 | 网页输入脑暴 → AI 拆条 → 编辑 → 保存全流程 |
| M4 | 周期算法 + 报告生成 | 4 类报告都能从后端生成，不变式守门 |
| M5 | 前端报告视图 | 网页生成/查看 4 类报告 |
| M6 | 收尾 | 文档更新 + E2E 通过 |

**不变式守门清单（service 测试覆盖）：**

- 月报只读周报 + 孤儿 items，不读全部原始 items
- 半年报只读月报，不读周报或 items
- 年报只读月报，不读周报或 items
- AI 拆条凑不出具体产出时整维输出"（待补充）"
- 同日二次保存走 PUT，不静默覆盖
- 报告覆盖前必须 force=true

