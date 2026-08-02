# 工作日志快捷录入 + 今日全景 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在「工作日志」页面顶部新增快捷录入表单 + 今日全景表，复用 WorkItem 表（扩展 5 字段），AI 流程与 manual 流程通过 `source` 字段隔离互不擦除。

**Architecture:** WorkItem 表加 5 字段（activity/start_time/end_time/quadrant/source），新接口走 `/api/work-logs/:date/items` 子资源。`UpsertWorkLog` 内部删除改加 `WHERE source='ai'` 过滤，保留 manual 条目。前端在 `WorkLog.vue` 顶部插 QuickEntryForm + TodayPanorama 两块组件，共享 `currentDate`。

**Tech Stack:** Go 1.21 + Gin + GORM + SQLite | Vue 3.5 + Pinia (Composition API) + Element Plus 2.8 + TypeScript 5.6 strict | Vitest 2.1

**Spec:** `docs/superpowers/specs/2026-08-02-work-log-quick-entry-design.md`

**Branch:** evolve/work-log-quick-entry（从 evolve/work-log-integration 切出）

---

## File Structure

**Backend (modify):**
- `backend/internal/model/work_log.go` — 扩展 WorkItem 5 字段
- `backend/internal/repository/errors.go` — 新增 2 个 sentinel 错误
- `backend/internal/repository/work_log_repo.go` — 接口加 3 方法 + UpsertWorkLog/ReplaceItems 改造
- `backend/internal/service/work_log_service.go` — DTO + 3 个 service 方法
- `backend/internal/api/handler/work_log.go` — 3 个新 handler
- `backend/internal/api/router.go` — 注册 3 条新路由
- `backend/internal/api/handler/mocks_test.go` — 新增 mockWorkLogRepository

**Backend (create):**
- `backend/internal/repository/work_log_repo_test.go` — repo 测试
- `backend/internal/service/work_log_service_quick_test.go` — quick entry service 测试（独立文件避免污染现有大文件）

**Frontend (modify):**
- `frontend/src/types/index.ts` — 扩展 WorkItem
- `frontend/src/api/client.ts` — 加 3 个 api 方法
- `frontend/src/stores/workLog.ts` — getter + 3 actions
- `frontend/src/views/WorkLog.vue` — 接线新组件

**Frontend (create):**
- `frontend/src/components/work-log/QuickEntryForm.vue`
- `frontend/src/components/work-log/TodayPanorama.vue`
- `frontend/src/components/work-log/QuickEntryForm.spec.ts`
- `frontend/src/components/work-log/TodayPanorama.spec.ts`
- `frontend/src/stores/workLog.spec.ts`（若不存在则创建）

---

## Task 1: 扩展 WorkItem model（5 字段）

**Files:**
- Modify: `backend/internal/model/work_log.go:21-30`

- [ ] **Step 1: 修改 WorkItem struct**

替换 `backend/internal/model/work_log.go` 第 21-30 行整个 WorkItem struct：

```go
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
```

- [ ] **Step 2: 验证编译通过**

Run: `cd backend && go build ./...`
Expected: 无报错退出

- [ ] **Step 3: 启动 backend 验证 AutoMigrate 加列**

Run: `cd backend && CGO_ENABLED=1 go run cmd/server/main.go`，等打印 `listening on :8080` 后 Ctrl+C 退出。

然后检查 SQLite schema（需要 sqlite3 CLI；若没有可跳过此步，依赖后续 repo 测试覆盖）：

```bash
sqlite3 backend/data/ticktask.db ".schema work_items"
```

Expected: 输出包含 `activity TEXT`、`start_time TEXT`、`end_time TEXT`、`quadrant INTEGER`、`source TEXT DEFAULT 'ai'` 这 5 个新列。

- [ ] **Step 4: Commit**

```bash
git add backend/internal/model/work_log.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): extend WorkItem with activity/time/quadrant/source fields"
```

---

## Task 2: 新增 repo sentinel 错误

**Files:**
- Modify: `backend/internal/repository/errors.go`

- [ ] **Step 1: 添加新错误**

把 `backend/internal/repository/errors.go` 整个文件替换为：

```go
package repository

import "errors"

// 资源不存在
var ErrNotFound = errors.New("resource not found")

// WorkItem 不存在
var ErrItemNotFound = errors.New("work item not found")

// WorkItem 不能通过 quick-entry 接口修改（如 source='ai' 的条目）
var ErrItemNotEditable = errors.New("work item is not editable via this endpoint")
```

- [ ] **Step 2: 验证编译**

Run: `cd backend && go build ./...`
Expected: 无报错

- [ ] **Step 3: Commit**

```bash
git add backend/internal/repository/errors.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add ErrItemNotFound + ErrItemNotEditable sentinels"
```

---

## Task 3: 扩展 WorkLogRepository 接口 + AppendItem/UpdateItem/DeleteItem 实现（TDD）

**Files:**
- Modify: `backend/internal/repository/work_log_repo.go:13-28`（接口）、文件末尾（实现）
- Create: `backend/internal/repository/work_log_repo_test.go`

- [ ] **Step 1: 写失败测试**

创建 `backend/internal/repository/work_log_repo_test.go`：

```go
package repository

import (
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"ticktask/internal/model"
)

func newTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkLog{}, &model.WorkItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func strPtr(s string) *string { return &s }
func intPtr(i int) *int       { return &i }

func TestAppendItem_ToExistingWorkLog(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	if err := repo.CreateWorkLog(log); err != nil {
		t.Fatalf("create log: %v", err)
	}
	item := model.WorkItem{
		Activity:  strPtr("晨会"),
		StartTime: strPtr("09:00"),
		EndTime:   strPtr("10:00"),
		Quadrant:  intPtr(1),
		Source:    "manual",
		Seq:       1,
	}
	if err := repo.AppendItem(log.ID, item); err != nil {
		t.Fatalf("append: %v", err)
	}
	got, err := repo.GetWorkLogByDate("2026-08-02")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(got.Items) != 1 || got.Items[0].Activity == nil || *got.Items[0].Activity != "晨会" {
		t.Fatalf("unexpected items: %+v", got.Items)
	}
}

func TestAppendItem_WorkLogNotExists(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	err := repo.AppendItem("nonexistent", model.WorkItem{Source: "manual"})
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateItem_ManualSucceeds(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	repo.CreateWorkLog(log)
	item := model.WorkItem{ID: uuid.New().String(), WorkLogID: log.ID, Source: "manual", Activity: strPtr("old"), Seq: 1}
	repo.AppendItem(log.ID, item)
	updates := map[string]any{"activity": "new"}
	if err := repo.UpdateItem(log.ID, item.ID, updates); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := repo.GetWorkLogByDate("2026-08-02")
	if *got.Items[0].Activity != "new" {
		t.Fatalf("activity not updated: %+v", got.Items[0])
	}
}

func TestUpdateItem_AIItemReturnsErrItemNotEditable(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	repo.CreateWorkLog(log)
	item := model.WorkItem{ID: uuid.New().String(), WorkLogID: log.ID, Source: "ai", Title: "x", Seq: 1}
	repo.AppendItem(log.ID, item)
	err := repo.UpdateItem(log.ID, item.ID, map[string]any{"title": "y"})
	if err != ErrItemNotEditable {
		t.Fatalf("expected ErrItemNotEditable, got %v", err)
	}
}

func TestUpdateItem_NotExistsReturnsErrItemNotFound(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	repo.CreateWorkLog(log)
	err := repo.UpdateItem(log.ID, "nonexistent", map[string]any{"activity": "x"})
	if err != ErrItemNotFound {
		t.Fatalf("expected ErrItemNotFound, got %v", err)
	}
}

func TestDeleteItem_ManualSucceeds(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	repo.CreateWorkLog(log)
	item := model.WorkItem{ID: uuid.New().String(), WorkLogID: log.ID, Source: "manual", Seq: 1}
	repo.AppendItem(log.ID, item)
	if err := repo.DeleteItem(log.ID, item.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	got, _ := repo.GetWorkLogByDate("2026-08-02")
	if len(got.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(got.Items))
	}
}

func TestDeleteItem_AIItemReturnsErrItemNotEditable(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	log := &model.WorkLog{ID: uuid.New().String(), Date: "2026-08-02"}
	repo.CreateWorkLog(log)
	item := model.WorkItem{ID: uuid.New().String(), WorkLogID: log.ID, Source: "ai", Title: "x", Seq: 1}
	repo.AppendItem(log.ID, item)
	err := repo.DeleteItem(log.ID, item.ID)
	if err != ErrItemNotEditable {
		t.Fatalf("expected ErrItemNotEditable, got %v", err)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd backend && go test -v ./internal/repository/ -run TestAppendItem_ToExistingWorkLog`
Expected: FAIL — `undefined: AppendItem`（接口里没这方法）

- [ ] **Step 3: 扩展接口**

修改 `backend/internal/repository/work_log_repo.go` 第 13-28 行，把 WorkLogRepository interface 改成（保留所有现有方法，仅插入 3 个新方法）：

```go
type WorkLogRepository interface {
	// WorkLog CRUD
	CreateWorkLog(log *model.WorkLog) error
	GetWorkLogByDate(date string) (*model.WorkLog, error)
	GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error)
	UpsertWorkLog(log *model.WorkLog) error

	// WorkItem
	ReplaceItems(workLogID string, items []model.WorkItem) error
	AppendItem(workLogID string, item model.WorkItem) error
	UpdateItem(workLogID string, itemID string, updates map[string]any) error
	DeleteItem(workLogID string, itemID string) error

	// WorkReport
	CreateWorkReport(report *model.WorkReport) error
	UpdateWorkReport(report *model.WorkReport) error
	GetWorkReportByTypeAndPeriod(t model.WorkReportType, periodKey string) (*model.WorkReport, error)
	ListWorkReports(t model.WorkReportType) ([]*model.WorkReport, error)
}
```

- [ ] **Step 4: 实现三个新方法**

在 `backend/internal/repository/work_log_repo.go` 文件末尾追加：

```go
func (r *workLogRepository) AppendItem(workLogID string, item model.WorkItem) error {
	// 校验 WorkLog 存在
	var log model.WorkLog
	err := r.db.Where("id = ?", workLogID).First(&log).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	item.ID = uuid.New().String()
	item.WorkLogID = workLogID
	return r.db.Create(&item).Error
}

func (r *workLogRepository) UpdateItem(workLogID string, itemID string, updates map[string]any) error {
	var existing model.WorkItem
	err := r.db.Where("work_log_id = ? AND id = ?", workLogID, itemID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrItemNotFound
	}
	if err != nil {
		return err
	}
	if existing.Source != "manual" {
		return ErrItemNotEditable
	}
	return r.db.Model(&model.WorkItem{}).
		Where("id = ? AND work_log_id = ?", itemID, workLogID).
		Updates(updates).Error
}

func (r *workLogRepository) DeleteItem(workLogID string, itemID string) error {
	var existing model.WorkItem
	err := r.db.Where("work_log_id = ? AND id = ?", workLogID, itemID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrItemNotFound
	}
	if err != nil {
		return err
	}
	if existing.Source != "manual" {
		return ErrItemNotEditable
	}
	return r.db.Where("id = ? AND work_log_id = ?", itemID, workLogID).
		Delete(&model.WorkItem{}).Error
}
```

- [ ] **Step 5: 运行所有 repo 测试**

Run: `cd backend && go test -v ./internal/repository/ -run 'TestAppendItem|TestUpdateItem|TestDeleteItem'`
Expected: 7 个测试全 PASS

- [ ] **Step 6: Commit**

```bash
git add backend/internal/repository/work_log_repo.go backend/internal/repository/work_log_repo_test.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add AppendItem/UpdateItem/DeleteItem to repo"
```

---

## Task 4: UpsertWorkLog + ReplaceItems 加 `source='ai'` 过滤（关键不变式 TDD）

**Files:**
- Modify: `backend/internal/repository/work_log_repo.go:81`、`97`
- Modify: `backend/internal/repository/work_log_repo_test.go`（追加回归测试）

- [ ] **Step 1: 写失败回归测试**

在 `backend/internal/repository/work_log_repo_test.go` 末尾追加：

```go
func TestUpsertWorkLog_PreservesManualItems(t *testing.T) {
	db := newTestDB(t)
	repo := NewWorkLogRepository(db)
	// 第一次：建一个含 manual item 的 WorkLog
	log1 := &model.WorkLog{
		ID:    uuid.New().String(),
		Date:  "2026-08-02",
		Items: []model.WorkItem{
			{Activity: strPtr("晨会"), StartTime: strPtr("09:00"), EndTime: strPtr("10:00"), Quadrant: intPtr(1), Source: "manual", Seq: 1},
		},
	}
	if err := repo.CreateWorkLog(log1); err != nil {
		t.Fatalf("create: %v", err)
	}
	// 第二次：UpsertWorkLog 来一波 ai items（模拟用户跑了 AI 流程）
	aiLog := &model.WorkLog{
		ID:    log1.ID,
		Date:  "2026-08-02",
		Items: []model.WorkItem{
			{Title: "ai-item-1", Source: "ai", Seq: 1},
			{Title: "ai-item-2", Source: "ai", Seq: 2},
		},
	}
	if err := repo.UpsertWorkLog(aiLog); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	got, err := repo.GetWorkLogByDate("2026-08-02")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	// 关键不变式：manual item 必须保留 + ai items 也存在
	manualCount, aiCount := 0, 0
	for _, it := range got.Items {
		if it.Source == "manual" {
			manualCount++
		} else {
			aiCount++
		}
	}
	if manualCount != 1 {
		t.Fatalf("manual items wiped! expected 1, got %d (items: %+v)", manualCount, got.Items)
	}
	if aiCount != 2 {
		t.Fatalf("ai items wrong: expected 2, got %d", aiCount)
	}
}
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd backend && go test -v ./internal/repository/ -run TestUpsertWorkLog_PreservesManualItems`
Expected: FAIL — `manual items wiped! expected 1, got 0`

- [ ] **Step 3: 修复 UpsertWorkLog 的删除语句**

修改 `backend/internal/repository/work_log_repo.go` 第 81 行：

旧：
```go
		if err := tx.Where("work_log_id = ?", log.ID).Delete(&model.WorkItem{}).Error; err != nil {
```

新：
```go
		// 关键不变式：只删除 ai items，保留 manual items（快捷录入）
		if err := tx.Where("work_log_id = ? AND source = ?", log.ID, "ai").Delete(&model.WorkItem{}).Error; err != nil {
```

- [ ] **Step 4: 同样修复 ReplaceItems**

修改 `backend/internal/repository/work_log_repo.go` 第 97 行：

旧：
```go
		if err := tx.Where("work_log_id = ?", workLogID).Delete(&model.WorkItem{}).Error; err != nil {
```

新：
```go
		// 保留 manual items（虽然 ReplaceItems 当前只被 AI 流程内部调用，仍守不变式）
		if err := tx.Where("work_log_id = ? AND source = ?", workLogID, "ai").Delete(&model.WorkItem{}).Error; err != nil {
```

- [ ] **Step 5: 运行所有 repo 测试**

Run: `cd backend && go test -v ./internal/repository/`
Expected: 所有测试 PASS（包括之前的 7 个 + 新增的 1 个回归）

- [ ] **Step 6: Commit**

```bash
git add backend/internal/repository/work_log_repo.go backend/internal/repository/work_log_repo_test.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "fix(work-log): preserve manual items in UpsertWorkLog/ReplaceItems (critical invariant)"
```

---

## Task 5: Service 层 — DTO + 3 个 quick-entry 方法（TDD）

**Files:**
- Modify: `backend/internal/service/work_log_service.go`（DTO 末尾追加 + service 方法末尾追加）
- Create: `backend/internal/service/work_log_service_quick_test.go`

- [ ] **Step 1: 写失败测试**

创建 `backend/internal/service/work_log_service_quick_test.go`：

```go
package service

import (
	"errors"
	"testing"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"ticktask/internal/model"
	"ticktask/internal/repository"
)

func newQuickTestDB(t *testing.T) (*gorm.DB, repository.WorkLogRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	db.AutoMigrate(&model.WorkLog{}, &model.WorkItem{})
	return db, repository.NewWorkLogRepository(db)
}

func newQuickService(t *testing.T) *WorkLogService {
	db, repo := newQuickTestDB(t)
	_ = db
	return NewWorkLogService(repo, nil, nil, nil)
}

func TestAddQuickEntry_AutoCreatesWorkLog(t *testing.T) {
	svc := newQuickService(t)
	in := CreateQuickEntryInput{
		Activity:  "晨会",
		StartTime: "09:00",
		EndTime:   "10:00",
		Quadrant:  1,
	}
	item, err := svc.AddQuickEntry("2026-08-02", in)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if item.Activity == nil || *item.Activity != "晨会" {
		t.Fatalf("bad item: %+v", item)
	}
	if item.Source != "manual" {
		t.Fatalf("expected source=manual, got %s", item.Source)
	}
	// 验证 WorkLog 自动建了
	log, err := svc.repo.GetWorkLogByDate("2026-08-02")
	if err != nil {
		t.Fatalf("reload log: %v", err)
	}
	if len(log.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(log.Items))
	}
}

func TestAddQuickEntry_AppendsToExistingWorkLog(t *testing.T) {
	svc := newQuickService(t)
	// 先建一条
	svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{Activity: "a", StartTime: "09:00", EndTime: "10:00", Quadrant: 1})
	// 再建一条
	_, err := svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{Activity: "b", StartTime: "10:00", EndTime: "11:00", Quadrant: 2})
	if err != nil {
		t.Fatalf("add 2nd: %v", err)
	}
	log, _ := svc.repo.GetWorkLogByDate("2026-08-02")
	if len(log.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(log.Items))
	}
	if log.Items[1].Seq != 2 {
		t.Fatalf("expected seq=2, got %d", log.Items[1].Seq)
	}
}

func TestAddQuickEntry_RejectsEndBeforeStart(t *testing.T) {
	svc := newQuickService(t)
	_, err := svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{
		Activity: "x", StartTime: "11:00", EndTime: "10:00", Quadrant: 1,
	})
	if err == nil {
		t.Fatal("expected error for end < start")
	}
}

func TestAddQuickEntry_RejectsBadDate(t *testing.T) {
	svc := newQuickService(t)
	_, err := svc.AddQuickEntry("not-a-date", CreateQuickEntryInput{
		Activity: "x", StartTime: "09:00", EndTime: "10:00", Quadrant: 1,
	})
	if err == nil {
		t.Fatal("expected error for bad date")
	}
}

func TestUpdateQuickEntry_HappyPath(t *testing.T) {
	svc := newQuickService(t)
	item, _ := svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{Activity: "old", StartTime: "09:00", EndTime: "10:00", Quadrant: 1})
	err := svc.UpdateQuickEntry("2026-08-02", item.ID, UpdateQuickEntryInput{
		Activity: strPtrService("new"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	log, _ := svc.repo.GetWorkLogByDate("2026-08-02")
	if *log.Items[0].Activity != "new" {
		t.Fatalf("not updated: %+v", log.Items[0])
	}
}

func TestUpdateQuickEntry_NotFound(t *testing.T) {
	svc := newQuickService(t)
	err := svc.UpdateQuickEntry("2026-08-02", "nonexistent", UpdateQuickEntryInput{Activity: strPtrService("x")})
	if !errors.Is(err, repository.ErrItemNotFound) {
		t.Fatalf("expected ErrItemNotFound, got %v", err)
	}
}

func TestDeleteQuickEntry_HappyPath(t *testing.T) {
	svc := newQuickService(t)
	item, _ := svc.AddQuickEntry("2026-08-02", CreateQuickEntryInput{Activity: "x", StartTime: "09:00", EndTime: "10:00", Quadrant: 1})
	if err := svc.DeleteQuickEntry("2026-08-02", item.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	log, _ := svc.repo.GetWorkLogByDate("2026-08-02")
	if len(log.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(log.Items))
	}
}

// 局部 helper：避免污染包级别
func strPtrService(s string) *string { return &s }

// 防止 uuid 未使用
var _ = uuid.New
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd backend && go test -v ./internal/service/ -run TestAddQuickEntry_AutoCreatesWorkLog`
Expected: FAIL — `undefined: CreateQuickEntryInput` / `undefined: AddQuickEntry`

- [ ] **Step 3: 添加 DTO**

在 `backend/internal/service/work_log_service.go` 第 97 行（`GenerateReportInput` struct 之后、`ReportSummary` 之前）插入：

```go
// CreateQuickEntryInput 快捷录入新增输入
type CreateQuickEntryInput struct {
	Activity  string `json:"activity"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
	Quadrant  int    `json:"quadrant"`
}

// UpdateQuickEntryInput 快捷录入编辑输入（指针 = 部分更新）
type UpdateQuickEntryInput struct {
	Activity  *string `json:"activity,omitempty"`
	StartTime *string `json:"start_time,omitempty"`
	EndTime   *string `json:"end_time,omitempty"`
	Quadrant  *int    `json:"quadrant,omitempty"`
}
```

- [ ] **Step 4: 添加 service 方法**

在 `backend/internal/service/work_log_service.go` 文件末尾（最后那个 `var _ = json.Marshal` 之前）追加：

```go
// AddQuickEntry 快捷录入：自动建 WorkLog（如不存在）+ 追加 manual item
func (s *WorkLogService) AddQuickEntry(date string, in CreateQuickEntryInput) (*model.WorkItem, error) {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, fmt.Errorf("invalid date: %w", err)
	}
	if in.StartTime >= in.EndTime {
		return nil, errors.New("end_time must be after start_time")
	}

	log, err := s.repo.GetWorkLogByDate(date)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}
	if log == nil {
		log = &model.WorkLog{
			ID:           s.idGenerator(),
			Date:         date,
			Summary:      "",
			RawBrainDump: "",
		}
		if err := s.repo.CreateWorkLog(log); err != nil {
			return nil, err
		}
	}

	maxSeq := 0
	for _, it := range log.Items {
		if it.Seq > maxSeq {
			maxSeq = it.Seq
		}
	}

	item := model.WorkItem{
		ID:        s.idGenerator(),
		WorkLogID: log.ID,
		Seq:       maxSeq + 1,
		Activity:  &in.Activity,
		StartTime: &in.StartTime,
		EndTime:   &in.EndTime,
		Quadrant:  &in.Quadrant,
		Source:    "manual",
	}
	if err := s.repo.AppendItem(log.ID, item); err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateQuickEntry 编辑快捷录入条目
func (s *WorkLogService) UpdateQuickEntry(date string, itemID string, in UpdateQuickEntryInput) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}
	if in.StartTime != nil && in.EndTime != nil && *in.StartTime >= *in.EndTime {
		return errors.New("end_time must be after start_time")
	}

	log, err := s.repo.GetWorkLogByDate(date)
	if err != nil {
		return err
	}

	updates := map[string]any{}
	if in.Activity != nil {
		updates["activity"] = *in.Activity
	}
	if in.StartTime != nil {
		updates["start_time"] = *in.StartTime
	}
	if in.EndTime != nil {
		updates["end_time"] = *in.EndTime
	}
	if in.Quadrant != nil {
		updates["quadrant"] = *in.Quadrant
	}
	if len(updates) == 0 {
		return nil
	}
	return s.repo.UpdateItem(log.ID, itemID, updates)
}

// DeleteQuickEntry 删除快捷录入条目
func (s *WorkLogService) DeleteQuickEntry(date string, itemID string) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}
	log, err := s.repo.GetWorkLogByDate(date)
	if err != nil {
		return err
	}
	return s.repo.DeleteItem(log.ID, itemID)
}
```

- [ ] **Step 5: 运行所有 quick service 测试**

Run: `cd backend && go test -v ./internal/service/ -run 'TestAddQuickEntry|TestUpdateQuickEntry|TestDeleteQuickEntry'`
Expected: 7 个测试全 PASS

- [ ] **Step 6: 运行全部 service 测试，确保未破坏现有**

Run: `cd backend && go test ./internal/service/...`
Expected: 全 PASS

- [ ] **Step 7: Commit**

```bash
git add backend/internal/service/work_log_service.go backend/internal/service/work_log_service_quick_test.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add AddQuickEntry/UpdateQuickEntry/DeleteQuickEntry service methods"
```

---

## Task 6: 新增 mockWorkLogRepository + 3 个 handler（TDD）

**Files:**
- Modify: `backend/internal/api/handler/mocks_test.go`（追加 mock）
- Modify: `backend/internal/api/handler/work_log.go`（追加 DTO + 3 handler）
- Modify: `backend/internal/api/handler/work_log_test.go`（若存在则追加，否则创建）

- [ ] **Step 1: 在 mocks_test.go 末尾追加 mockWorkLogRepository**

在 `backend/internal/api/handler/mocks_test.go` 文件末尾追加：

```go
// mockWorkLogRepository implements repository.WorkLogRepository for testing
type mockWorkLogRepository struct {
	logs   map[string]*model.WorkLog
	items  map[string]*model.WorkItem // itemID -> item
	reports map[string]*model.WorkReport
}

func newMockWorkLogRepository() *mockWorkLogRepository {
	return &mockWorkLogRepository{
		logs:    make(map[string]*model.WorkLog),
		items:   make(map[string]*model.WorkItem),
		reports: make(map[string]*model.WorkReport),
	}
}

func (m *mockWorkLogRepository) CreateWorkLog(log *model.WorkLog) error {
	m.logs[log.Date] = log
	for i := range log.Items {
		m.items[log.Items[i].ID] = &log.Items[i]
	}
	return nil
}

func (m *mockWorkLogRepository) GetWorkLogByDate(date string) (*model.WorkLog, error) {
	if log, ok := m.logs[date]; ok {
		return log, nil
	}
	return nil, repository.ErrNotFound
}

func (m *mockWorkLogRepository) GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error) {
	var result []*model.WorkLog
	for _, log := range m.logs {
		if log.Date >= from && log.Date <= to {
			result = append(result, log)
		}
	}
	return result, nil
}

func (m *mockWorkLogRepository) UpsertWorkLog(log *model.WorkLog) error {
	existing, ok := m.logs[log.Date]
	if !ok {
		m.logs[log.Date] = log
		for i := range log.Items {
			m.items[log.Items[i].ID] = &log.Items[i]
		}
		return nil
	}
	log.ID = existing.ID
	// 关键不变式：只删 ai items
	newItems := []model.WorkItem{}
	for _, it := range existing.Items {
		if it.Source == "manual" {
			newItems = append(newItems, it)
		} else {
			delete(m.items, it.ID)
		}
	}
	newItems = append(newItems, log.Items...)
	log.Items = newItems
	m.logs[log.Date] = log
	for i := range log.Items {
		m.items[log.Items[i].ID] = &log.Items[i]
	}
	return nil
}

func (m *mockWorkLogRepository) ReplaceItems(workLogID string, items []model.WorkItem) error {
	return nil // 测试不依赖
}

func (m *mockWorkLogRepository) AppendItem(workLogID string, item model.WorkItem) error {
	// 校验 WorkLog 存在
	var found *model.WorkLog
	for _, log := range m.logs {
		if log.ID == workLogID {
			found = log
			break
		}
	}
	if found == nil {
		return repository.ErrNotFound
	}
	item.WorkLogID = workLogID
	found.Items = append(found.Items, item)
	m.items[item.ID] = &item
	return nil
}

func (m *mockWorkLogRepository) UpdateItem(workLogID string, itemID string, updates map[string]any) error {
	item, ok := m.items[itemID]
	if !ok || item.WorkLogID != workLogID {
		return repository.ErrItemNotFound
	}
	if item.Source != "manual" {
		return repository.ErrItemNotEditable
	}
	if v, ok := updates["activity"]; ok {
		s := v.(string)
		item.Activity = &s
	}
	if v, ok := updates["start_time"]; ok {
		s := v.(string)
		item.StartTime = &s
	}
	if v, ok := updates["end_time"]; ok {
		s := v.(string)
		item.EndTime = &s
	}
	if v, ok := updates["quadrant"]; ok {
		i := v.(int)
		item.Quadrant = &i
	}
	return nil
}

func (m *mockWorkLogRepository) DeleteItem(workLogID string, itemID string) error {
	item, ok := m.items[itemID]
	if !ok || item.WorkLogID != workLogID {
		return repository.ErrItemNotFound
	}
	if item.Source != "manual" {
		return repository.ErrItemNotEditable
	}
	delete(m.items, itemID)
	for _, log := range m.logs {
		if log.ID == workLogID {
			for i, it := range log.Items {
				if it.ID == itemID {
					log.Items = append(log.Items[:i], log.Items[i+1:]...)
					break
				}
			}
			break
		}
	}
	return nil
}

func (m *mockWorkLogRepository) CreateWorkReport(report *model.WorkReport) error {
	m.reports[report.Type+":"+report.PeriodKey] = report
	return nil
}
func (m *mockWorkLogRepository) UpdateWorkReport(report *model.WorkReport) error {
	m.reports[report.Type+":"+report.PeriodKey] = report
	return nil
}
func (m *mockWorkLogRepository) GetWorkReportByTypeAndPeriod(t model.WorkReportType, periodKey string) (*model.WorkReport, error) {
	if r, ok := m.reports[string(t)+":"+periodKey]; ok {
		return r, nil
	}
	return nil, repository.ErrNotFound
}
func (m *mockWorkLogRepository) ListWorkReports(t model.WorkReportType) ([]*model.WorkReport, error) {
	var result []*model.WorkReport
	for _, r := range m.reports {
		if r.Type == t {
			result = append(result, r)
		}
	}
	return result, nil
}
```

- [ ] **Step 2: 验证 mock 编译通过**

Run: `cd backend && go build ./internal/api/handler/...`
Expected: 编译通过（mock 在 _test.go 中，build 默认不编译测试。改用 `go vet ./internal/api/handler/...` 或先写测试再验证）

- [ ] **Step 3: 写 handler 失败测试**

检查 `backend/internal/api/handler/work_log_test.go` 是否存在：

```bash
ls backend/internal/api/handler/work_log_test.go 2>/dev/null
```

如果不存在就创建 `backend/internal/api/handler/work_log_test.go`（如下完整内容）。如果已存在，把下面 3 个测试函数追加到文件末尾（不要重复 package 声明）。

完整文件内容：

```go
package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"ticktask/internal/model"
	"ticktask/internal/repository"
	"ticktask/internal/service"
)

func TestAddQuickEntry_HandlerHappyPath(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r.POST("/api/work-logs/:date/items", h.AddQuickEntry)

	body, _ := json.Marshal(gin.H{
		"activity": "晨会", "start_time": "09:00", "end_time": "10:00", "quadrant": 1,
	})
	req := httptest.NewRequest("POST", "/api/work-logs/2026-08-02/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	var resp model.WorkItem
	json.Unmarshal(w.Body.Bytes(), &resp)
	if resp.Source != "manual" || resp.Activity == nil || *resp.Activity != "晨会" {
		t.Fatalf("bad response: %+v", resp)
	}
}

func TestAddQuickEntry_HandlerRejectsBadTime(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r.POST("/api/work-logs/:date/items", h.AddQuickEntry)

	body, _ := json.Marshal(gin.H{
		"activity": "x", "start_time": "11:00", "end_time": "10:00", "quadrant": 1,
	})
	req := httptest.NewRequest("POST", "/api/work-logs/2026-08-02/items", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestDeleteQuickEntry_HandlerReturns403ForAIItem(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	repo := newMockWorkLogRepository()
	svc := service.NewWorkLogService(repo, nil, nil, nil)
	h := NewWorkLogHandler(svc)
	r.DELETE("/api/work-logs/:date/items/:itemId", h.DeleteQuickEntry)

	aiItem := model.WorkItem{ID: "ai-1", WorkLogID: "log-1", Source: "ai", Title: "x"}
	repo.logs["2026-08-02"] = &model.WorkLog{ID: "log-1", Date: "2026-08-02", Items: []model.WorkItem{aiItem}}
	repo.items["ai-1"] = &aiItem

	req := httptest.NewRequest("DELETE", "/api/work-logs/2026-08-02/items/ai-1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

var _ = repository.ErrItemNotEditable
```

- [ ] **Step 4: 运行测试验证失败**

Run: `cd backend && go test -v ./internal/api/handler/ -run TestAddQuickEntry_HandlerHappyPath`
Expected: FAIL — `undefined: AddQuickEntry` on handler

- [ ] **Step 5: 加 handler DTO 和 3 个方法**

在 `backend/internal/api/handler/work_log.go` 文件末尾追加：

```go
// ── 快捷录入端点 ──

type createQuickEntryInput struct {
	Activity  string `json:"activity" binding:"required"`
	StartTime string `json:"start_time" binding:"required"`
	EndTime   string `json:"end_time" binding:"required"`
	Quadrant  int    `json:"quadrant" binding:"required,min=1,max=4"`
}

type updateQuickEntryInput struct {
	Activity  *string `json:"activity,omitempty"`
	StartTime *string `json:"start_time,omitempty"`
	EndTime   *string `json:"end_time,omitempty"`
	Quadrant  *int    `json:"quadrant,omitempty" binding:"omitempty,min=1,max=4"`
}

// AddQuickEntry POST /api/work-logs/:date/items
func (h *WorkLogHandler) AddQuickEntry(c *gin.Context) {
	date := c.Param("date")
	var req createQuickEntryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	item, err := h.svc.AddQuickEntry(date, service.CreateQuickEntryInput{
		Activity:  req.Activity,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Quadrant:  req.Quadrant,
	})
	if err != nil {
		status := mapQuickEntryErrorStatus(err)
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusCreated, item)
}

// UpdateQuickEntry PATCH /api/work-logs/:date/items/:itemId
func (h *WorkLogHandler) UpdateQuickEntry(c *gin.Context) {
	date := c.Param("date")
	itemID := c.Param("itemId")
	var req updateQuickEntryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.svc.UpdateQuickEntry(date, itemID, service.UpdateQuickEntryInput{
		Activity:  req.Activity,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
		Quadrant:  req.Quadrant,
	})
	if err != nil {
		c.JSON(mapQuickEntryErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// DeleteQuickEntry DELETE /api/work-logs/:date/items/:itemId
func (h *WorkLogHandler) DeleteQuickEntry(c *gin.Context) {
	date := c.Param("date")
	itemID := c.Param("itemId")
	if err := h.svc.DeleteQuickEntry(date, itemID); err != nil {
		c.JSON(mapQuickEntryErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

func mapQuickEntryErrorStatus(err error) int {
	if errors.Is(err, repository.ErrItemNotFound) {
		return http.StatusNotFound
	}
	if errors.Is(err, repository.ErrItemNotEditable) {
		return http.StatusForbidden
	}
	if strings.HasPrefix(err.Error(), "invalid date:") {
		return http.StatusBadRequest
	}
	if err.Error() == "end_time must be after start_time" {
		return http.StatusBadRequest
	}
	return http.StatusInternalServerError
}
```

- [ ] **Step 6: 运行测试验证通过**

Run: `cd backend && go test -v ./internal/api/handler/ -run 'TestAddQuickEntry|TestDeleteQuickEntry'`
Expected: 3 个测试全 PASS

- [ ] **Step 7: Commit**

```bash
git add backend/internal/api/handler/mocks_test.go backend/internal/api/handler/work_log.go backend/internal/api/handler/work_log_test.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add quick-entry HTTP handlers + mockWorkLogRepository"
```

---

## Task 7: 注册 3 条新路由

**Files:**
- Modify: `backend/internal/api/router.go:115-125`

- [ ] **Step 1: 在 workLogs group 里加 3 条路由**

修改 `backend/internal/api/router.go` 第 116-125 行的 workLogs group，把：

```go
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
```

改成：

```go
		workLogs := api.Group("/work-logs")
		{
			wlHandler := handler.NewWorkLogHandler(workLogService)
			workLogs.GET("/today/context", wlHandler.GetTodayContext)
			workLogs.POST("/structure", wlHandler.Structure)
			workLogs.GET("", wlHandler.ListWorkLogs)
			workLogs.POST("", wlHandler.CreateWorkLog)
			workLogs.GET("/:date", wlHandler.GetWorkLog)
			workLogs.PUT("/:date", wlHandler.UpdateWorkLog)
			// 快捷录入（今日全景）
			workLogs.POST("/:date/items", wlHandler.AddQuickEntry)
			workLogs.PATCH("/:date/items/:itemId", wlHandler.UpdateQuickEntry)
			workLogs.DELETE("/:date/items/:itemId", wlHandler.DeleteQuickEntry)
		}
```

- [ ] **Step 2: 验证编译 + 启动**

Run: `cd backend && go build ./...`
Expected: 编译通过

启动后端做冒烟（Ctrl+C 退出即可）：
```bash
cd backend && CGO_ENABLED=1 go run cmd/server/main.go
```

期望：日志正常启动到 `listening on :8080`。

- [ ] **Step 3: 用 curl 验证全链路**

后端跑起来后，新开一个 shell：

```bash
curl -s -X POST http://localhost:8080/api/work-logs/2026-08-02/items \
  -H 'Content-Type: application/json' \
  -d '{"activity":"测试","start_time":"09:00","end_time":"10:00","quadrant":1}'
```

Expected: HTTP 201 + 返回 body 包含 `"source":"manual"` 和 `"activity":"测试"`

```bash
curl -s http://localhost:8080/api/work-logs/2026-08-02 | head -100
```

Expected: 返回的 WorkLog JSON 里 items 数组包含刚加的条目。

- [ ] **Step 4: Commit**

```bash
git add backend/internal/api/router.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): register quick-entry routes"
```

---

## Task 8: 前端 — 扩展 WorkItem 类型 + 3 个 API 方法

**Files:**
- Modify: `frontend/src/types/index.ts:315-324`
- Modify: `frontend/src/api/client.ts:84-99`

- [ ] **Step 1: 扩展 WorkItem 接口**

修改 `frontend/src/types/index.ts` 第 315-324 行：

```ts
export interface WorkItem {
  id: string
  work_log_id: string
  seq: number
  title: string
  content: string
  problem_solved: string
  result: string
  impact: string
  // 快捷录入字段（manual 必填，ai 为 null）
  activity?: string | null
  start_time?: string | null
  end_time?: string | null
  quadrant?: Quadrant | null
  source?: 'manual' | 'ai'
}

// 快捷录入新建输入
export interface CreateQuickEntryInput {
  activity: string
  start_time: string
  end_time: string
  quadrant: Quadrant
}

// 快捷录入编辑输入（部分更新）
export interface UpdateQuickEntryInput {
  activity?: string
  start_time?: string
  end_time?: string
  quadrant?: Quadrant
}
```

- [ ] **Step 2: 在 api client 末尾追加 3 个方法**

修改 `frontend/src/api/client.ts` 第 84-99 行的「工作日志」段，在 `getWorkReport` 之后追加（并在文件顶部 import 加上新类型）：

文件顶部 import 行（第 1 行）末尾追加 `CreateQuickEntryInput, UpdateQuickEntryInput, WorkItem`：

旧：
```ts
import type { Task, TaskResponse, PomodoroSession, ClassificationResult, PrioritySuggestion, AIStatus, PomodoroSettings, AISettings, TaskTimeStats, DailySummary, TrendData, DistributionStats, ScheduleEvent, CreateScheduleDTO, UpdateScheduleDTO, MoveScheduleDTO, RescheduleResult, DailyInsights, ReviseResponse, PomodoroByTaskResult, PomodoroTrendsResult, WorkLog, WorkReport, WorkReportType, TodayContext, StructuredWorkLog, SaveWorkLogInput } from '@/types'
```

新：
```ts
import type { Task, TaskResponse, PomodoroSession, ClassificationResult, PrioritySuggestion, AIStatus, PomodoroSettings, AISettings, TaskTimeStats, DailySummary, TrendData, DistributionStats, ScheduleEvent, CreateScheduleDTO, UpdateScheduleDTO, MoveScheduleDTO, RescheduleResult, DailyInsights, ReviseResponse, PomodoroByTaskResult, PomodoroTrendsResult, WorkLog, WorkReport, WorkReportType, TodayContext, StructuredWorkLog, SaveWorkLogInput, CreateQuickEntryInput, UpdateQuickEntryInput, WorkItem } from '@/types'
```

api 对象里，把 `getWorkReport` 那行后面追加（注意保留原有 `}` 闭合）：

```ts
  // 快捷录入（今日全景）
  appendWorkItem: (date: string, data: CreateQuickEntryInput) =>
    client.post<WorkItem>(`/work-logs/${date}/items`, data),
  updateWorkItem: (date: string, itemId: string, data: UpdateQuickEntryInput) =>
    client.patch<{ ok: boolean }>(`/work-logs/${date}/items/${itemId}`, data),
  deleteWorkItem: (date: string, itemId: string) =>
    client.delete<{ ok: boolean }>(`/work-logs/${date}/items/${itemId}`),
```

最终完整「工作日志」段（第 84-105 行左右）应该是：

```ts
  // 工作日志
  getTodayContext: (date?: string) => client.get<TodayContext>('/work-logs/today/context', { params: { date } }),
  structureBrainDump: (brain_dump: string, context: TodayContext) =>
    client.post<StructuredWorkLog>('/work-logs/structure', { brain_dump, context }, { timeout: 120000 }),
  listWorkLogs: (from: string, to: string) =>
    client.get<{ logs: WorkLog[] }>('/work-logs', { params: { from, to } }),
  getWorkLog: (date: string) => client.get<WorkLog>(`/work-logs/${date}`),
  createWorkLog: (data: SaveWorkLogInput) => client.post<WorkLog>('/work-logs', data),
  updateWorkLog: (date: string, data: SaveWorkLogInput) => client.put<WorkLog>(`/work-logs/${date}`, data),
  generateWorkReport: (type: WorkReportType, period_key?: string, force = false) =>
    client.post<WorkReport>('/work-reports/generate', { type, period_key, force }, { timeout: 180000 }),
  listWorkReports: (type: WorkReportType) =>
    client.get<{ reports: WorkReport[] }>('/work-reports', { params: { type } }),
  getWorkReport: (type: WorkReportType, periodKey: string) =>
    client.get<WorkReport>(`/work-reports/${type}/${periodKey}`),
  // 快捷录入（今日全景）
  appendWorkItem: (date: string, data: CreateQuickEntryInput) =>
    client.post<WorkItem>(`/work-logs/${date}/items`, data),
  updateWorkItem: (date: string, itemId: string, data: UpdateQuickEntryInput) =>
    client.patch<{ ok: boolean }>(`/work-logs/${date}/items/${itemId}`, data),
  deleteWorkItem: (date: string, itemId: string) =>
    client.delete<{ ok: boolean }>(`/work-logs/${date}/items/${itemId}`)
}
```

注意：原文件末尾对象最后一个方法 `getWorkReport` 没有逗号，加了 3 个新方法后 `getWorkReport` 行需要加逗号。

- [ ] **Step 3: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无错误

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/index.ts frontend/src/api/client.ts
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): extend WorkItem type + add quick-entry api methods"
```

---

## Task 9: 前端 store — getter + 3 个 actions（TDD）

**Files:**
- Modify: `frontend/src/stores/workLog.ts`
- Modify or Create: `frontend/src/stores/workLog.spec.ts`

- [ ] **Step 1: 检查 store spec 是否存在**

```bash
ls frontend/src/stores/workLog.spec.ts 2>/dev/null
```

如果不存在，创建 `frontend/src/stores/workLog.spec.ts`（含 quick-entry 测试）；如果存在，把测试追加到末尾。

创建版本：

```ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { createPinia, setActivePinia } from 'pinia'
import { useWorkLogStore } from './workLog'
import type { WorkLog, WorkItem } from '@/types'

vi.mock('@/api/client', () => ({
  api: {
    listWorkLogs: vi.fn().mockResolvedValue({ data: { logs: [] } }),
    getWorkLog: vi.fn(),
    appendWorkItem: vi.fn(),
    updateWorkItem: vi.fn(),
    deleteWorkItem: vi.fn(),
  },
}))

import { api } from '@/api/client'

function makeItem(over: Partial<WorkItem>): WorkItem {
  return {
    id: 'x', work_log_id: 'log-1', seq: 0, title: '', content: '',
    problem_solved: '', result: '', impact: '', source: 'manual',
    ...over,
  } as WorkItem
}

describe('workLog store - quick entry', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('todayManualItems filters source=manual and sorts by start_time', () => {
    const store = useWorkLogStore()
    store.currentLog = {
      id: 'log-1', date: '2026-08-02', summary: '', raw_brain_dump: '',
      created_at: '', updated_at: '',
      items: [
        makeItem({ id: '1', source: 'ai', title: 'ai-item' }),
        makeItem({ id: '2', source: 'manual', start_time: '10:00', activity: 'b' }),
        makeItem({ id: '3', source: 'manual', start_time: '09:00', activity: 'a' }),
      ],
    } as WorkLog
    const got = store.todayManualItems
    expect(got).toHaveLength(2)
    expect(got[0].id).toBe('3')  // 09:00 排前
    expect(got[1].id).toBe('2')
  })

  it('addQuickEntry calls api and refetches currentLog', async () => {
    vi.mocked(api.appendWorkItem).mockResolvedValueOnce({ data: {} as WorkItem })
    vi.mocked(api.getWorkLog).mockResolvedValueOnce({ data: { id: 'l1', date: '2026-08-02', items: [] } as WorkLog })
    const store = useWorkLogStore()
    await store.addQuickEntry('2026-08-02', {
      activity: '晨会', start_time: '09:00', end_time: '10:00', quadrant: 1,
    })
    expect(api.appendWorkItem).toHaveBeenCalledWith('2026-08-02', {
      activity: '晨会', start_time: '09:00', end_time: '10:00', quadrant: 1,
    })
    expect(api.getWorkLog).toHaveBeenCalledWith('2026-08-02')
  })

  it('updateQuickEntry calls api and refetches', async () => {
    vi.mocked(api.updateWorkItem).mockResolvedValueOnce({ data: { ok: true } })
    vi.mocked(api.getWorkLog).mockResolvedValueOnce({ data: { id: 'l1', date: '2026-08-02', items: [] } as WorkLog })
    const store = useWorkLogStore()
    await store.updateQuickEntry('2026-08-02', 'item-1', { activity: 'new' })
    expect(api.updateWorkItem).toHaveBeenCalledWith('2026-08-02', 'item-1', { activity: 'new' })
  })

  it('deleteQuickEntry calls api and refetches', async () => {
    vi.mocked(api.deleteWorkItem).mockResolvedValueOnce({ data: { ok: true } })
    vi.mocked(api.getWorkLog).mockResolvedValueOnce({ data: { id: 'l1', date: '2026-08-02', items: [] } as WorkLog })
    const store = useWorkLogStore()
    await store.deleteQuickEntry('2026-08-02', 'item-1')
    expect(api.deleteWorkItem).toHaveBeenCalledWith('2026-08-02', 'item-1')
  })

  it('addQuickEntry does not call fetchTodayContext on success', async () => {
    vi.mocked(api.appendWorkItem).mockResolvedValueOnce({ data: {} as WorkItem })
    vi.mocked(api.getWorkLog).mockResolvedValueOnce({ data: { id: 'l1', date: '2026-08-02', items: [] } as WorkLog })
    const getTodaySpy = vi.mocked(api.getTodayContext)
    const store = useWorkLogStore()
    await store.addQuickEntry('2026-08-02', { activity: 'x', start_time: '09:00', end_time: '10:00', quadrant: 1 })
    expect(getTodaySpy).not.toHaveBeenCalled()
  })
})
```

注意 mock 必须包含 `getTodayContext`（最后一个测试断言它未被调用）：第一个 `vi.mock('@/api/client', ...)` 块里已经包含 `getTodayContext: vi.fn()` ✓。

- [ ] **Step 2: 运行测试验证失败**

Run: `cd frontend && npx vitest run src/stores/workLog.spec.ts`
Expected: FAIL — `store.todayManualItems is not a function` 或类似

- [ ] **Step 3: 修改 store**

修改 `frontend/src/stores/workLog.ts`：

3a. 顶部 import 加 `computed`：

旧：
```ts
import { ref } from 'vue'
```

新：
```ts
import { ref, computed } from 'vue'
```

3b. 在 `loading` ref 之后、`fetchInitialRange` 之前插入 getter：

```ts
  const todayManualItems = computed(() => {
    const items = currentLog.value?.items ?? []
    return items
      .filter(i => i.source === 'manual')
      .sort((a, b) => (a.start_time ?? '').localeCompare(b.start_time ?? ''))
  })
```

3c. 在 `saveWorkLog` 之后追加 3 个 actions（**显式传 date 参数**，避免依赖 currentLog 状态）：

```ts
  async function addQuickEntry(date: string, payload: CreateQuickEntryInput): Promise<boolean> {
    try {
      await api.appendWorkItem(date, payload)
      await fetchLog(date) // 只刷新 WorkLog，不调 fetchTodayContext
      return true
    } catch (e: any) {
      ElMessage.error('添加失败：' + (e?.response?.data?.error || e?.message || ''))
      return false
    }
  }

  async function updateQuickEntry(date: string, itemId: string, payload: UpdateQuickEntryInput): Promise<boolean> {
    try {
      await api.updateWorkItem(date, itemId, payload)
      await fetchLog(date)
      return true
    } catch (e: any) {
      ElMessage.error('更新失败：' + (e?.response?.data?.error || e?.message || ''))
      return false
    }
  }

  async function deleteQuickEntry(date: string, itemId: string): Promise<boolean> {
    try {
      await api.deleteWorkItem(date, itemId)
      await fetchLog(date)
      return true
    } catch (e: any) {
      ElMessage.error('删除失败：' + (e?.response?.data?.error || e?.message || ''))
      return false
    }
  }
```

3d. import 加新类型：

旧：
```ts
import type {
  WorkLog, WorkReport, WorkReportType, TodayContext,
  StructuredWorkLog, SaveWorkLogInput,
} from '@/types'
```

新：
```ts
import type {
  WorkLog, WorkReport, WorkReportType, TodayContext,
  StructuredWorkLog, SaveWorkLogInput,
  CreateQuickEntryInput, UpdateQuickEntryInput,
} from '@/types'
```

3e. 在 store 返回对象里加上新成员。修改末尾 return 块：

旧：
```ts
  return {
    logs, currentLog, todayContext, reports, currentReport, selected, loading,
    fetchInitialRange, fetchLog, fetchTodayContext, structureBrainDump,
    saveWorkLog, generateReport, fetchReports, fetchReport, selectNode,
  }
```

新：
```ts
  return {
    logs, currentLog, todayContext, reports, currentReport, selected, loading,
    todayManualItems,
    fetchInitialRange, fetchLog, fetchTodayContext, structureBrainDump,
    saveWorkLog, generateReport, fetchReports, fetchReport, selectNode,
    addQuickEntry, updateQuickEntry, deleteQuickEntry,
  }
```

- [ ] **Step 4: 运行测试**

Run: `cd frontend && npx vitest run src/stores/workLog.spec.ts`
Expected: 5 个测试全 PASS

- [ ] **Step 5: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无错误

- [ ] **Step 6: Commit**

```bash
git add frontend/src/stores/workLog.ts frontend/src/stores/workLog.spec.ts
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add todayManualItems getter + quick-entry actions"
```

---

## Task 10: QuickEntryForm 组件（TDD）

**Files:**
- Create: `frontend/src/components/work-log/QuickEntryForm.vue`
- Create: `frontend/src/components/work-log/QuickEntryForm.spec.ts`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/components/work-log/QuickEntryForm.spec.ts`：

```ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import QuickEntryForm from './QuickEntryForm.vue'

vi.mock('@/api/client', () => ({
  api: {
    appendWorkItem: vi.fn().mockResolvedValue({ data: {} }),
    updateWorkItem: vi.fn().mockResolvedValue({ data: { ok: true } }),
    getWorkLog: vi.fn().mockResolvedValue({ data: { id: 'l1', date: '2026-08-02', items: [] } }),
    getTodayContext: vi.fn(),
  },
}))
vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn() },
}))

describe('QuickEntryForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders with default props', () => {
    const wrapper = mount(QuickEntryForm, {
      props: { date: '2026-08-02', mode: 'add' },
    })
    expect(wrapper.find('[data-test="submit-btn"]').exists()).toBe(true)
  })

  it('emits cancel when mode=edit and cancel clicked', async () => {
    const wrapper = mount(QuickEntryForm, {
      props: {
        date: '2026-08-02',
        mode: 'edit',
        itemId: 'item-1',
        initial: {
          activity: 'x', start_time: '09:00', end_time: '10:00', quadrant: 1,
        },
      },
    })
    await wrapper.find('[data-test="cancel-btn"]').trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
  })
})
```

- [ ] **Step 2: 运行测试验证失败**

Run: `cd frontend && npx vitest run src/components/work-log/QuickEntryForm.spec.ts`
Expected: FAIL — `Cannot find module './QuickEntryForm.vue'`

- [ ] **Step 3: 创建组件**

创建 `frontend/src/components/work-log/QuickEntryForm.vue`：

```vue
<template>
  <div class="quick-entry-form">
    <el-form inline :model="form" @submit.prevent="onSubmit">
      <el-form-item label="日期">
        <el-date-picker
          data-test="date-input"
          v-model="form.date"
          type="date"
          value-format="YYYY-MM-DD"
          :disabled="mode === 'edit'"
          style="width: 140px"
        />
      </el-form-item>

      <el-form-item label="活动">
        <el-input
          data-test="activity-input"
          v-model="form.activity"
          placeholder="做了什么"
          style="width: 200px"
        />
      </el-form-item>

      <el-form-item label="时段">
        <el-time-select
          data-test="start-input"
          v-model="form.start_time"
          :max-time="form.end_time"
          placeholder="开始"
          start="00:00"
          step="00:30"
          end="23:30"
          style="width: 110px"
        />
        <span style="margin: 0 6px">-</span>
        <el-time-select
          data-test="end-input"
          v-model="form.end_time"
          :min-time="form.start_time"
          placeholder="结束"
          start="00:30"
          step="00:30"
          end="24:00"
          style="width: 110px"
        />
      </el-form-item>

      <el-form-item label="象限">
        <el-radio-group v-model="form.quadrant" data-test="quadrant-input">
          <el-radio-button
            v-for="q in [1, 2, 3, 4] as Quadrant[]"
            :key="q"
            :value="q"
            :label="q"
          >
            Q{{ q }}
          </el-radio-button>
        </el-radio-group>
      </el-form-item>

      <el-form-item>
        <el-button
          data-test="submit-btn"
          type="primary"
          :loading="saving"
          @click="onSubmit"
        >
          {{ mode === 'edit' ? '保存' : '添加' }}
        </el-button>
        <el-button
          v-if="mode === 'edit'"
          data-test="cancel-btn"
          @click="$emit('cancel')"
        >
          取消
        </el-button>
      </el-form-item>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useWorkLogStore } from '@/stores/workLog'
import { QUADRANT_INFO } from '@/types'
import type { Quadrant } from '@/types'

interface InitialData {
  activity?: string
  start_time?: string
  end_time?: string
  quadrant?: Quadrant
}

const props = withDefaults(defineProps<{
  date: string
  mode?: 'add' | 'edit'
  initial?: InitialData
}>(), {
  mode: 'add',
  initial: () => ({}) as InitialData,
})

const emit = defineEmits<{
  added: []
  saved: []
  cancel: []
}>()

const store = useWorkLogStore()

const form = reactive({
  date: props.date,
  activity: props.initial.activity ?? '',
  start_time: props.initial.start_time ?? '09:00',
  end_time: props.initial.end_time ?? '10:00',
  quadrant: props.initial.quadrant ?? 2 as Quadrant,
})

watch(() => props.date, (d) => { if (props.mode === 'add') form.date = d })

const saving = ref(false)

async function onSubmit() {
  if (!form.activity.trim()) {
    ElMessage.error('活动名称必填')
    return
  }
  if (!form.start_time || !form.end_time) {
    ElMessage.error('时间段必填')
    return
  }
  if (form.start_time >= form.end_time) {
    ElMessage.error('结束时间必须晚于开始时间')
    return
  }
  saving.value = true
  try {
    if (props.mode === 'edit') {
      const ok = await store.updateQuickEntry(form.date, (props as any).itemId ?? '', {
        activity: form.activity,
        start_time: form.start_time,
        end_time: form.end_time,
        quadrant: form.quadrant,
      })
      if (ok) {
        ElMessage.success('已更新')
        emit('saved')
      }
    } else {
      const ok = await store.addQuickEntry(form.date, {
        activity: form.activity,
        start_time: form.start_time,
        end_time: form.end_time,
        quadrant: form.quadrant,
      })
      if (ok) {
        ElMessage.success('已添加')
        form.activity = '' // 清空活动，保留时段/象限方便连记
        emit('added')
      }
    }
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.quick-entry-form {
  background: var(--bg-card, #FFFEFC);
  border: 1px solid var(--border-color, #e5e5e5);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
}
</style>
```

注意：上面 `(props as any).itemId` 写法是因为 `edit` 模式需要 itemId，但 props 没声明。修正——加上 `itemId?: string` 到 props：

```ts
const props = withDefaults(defineProps<{
  date: string
  mode?: 'add' | 'edit'
  itemId?: string
  initial?: InitialData
}>(), {
  mode: 'add',
  itemId: '',
  initial: () => ({}) as InitialData,
})
```

并把 edit 分支改为 `props.itemId`：
```ts
const ok = await store.updateQuickEntry(form.date, props.itemId, { ... })
```

- [ ] **Step 4: 运行测试**

Run: `cd frontend && npx vitest run src/components/work-log/QuickEntryForm.spec.ts`
Expected: 2 个测试 PASS

- [ ] **Step 5: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无错误（如有 unused import 警告，删掉 `QUADRANT_INFO` import）

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/work-log/QuickEntryForm.vue frontend/src/components/work-log/QuickEntryForm.spec.ts
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add QuickEntryForm component"
```

---

## Task 11: TodayPanorama 组件（TDD）

**Files:**
- Create: `frontend/src/components/work-log/TodayPanorama.vue`
- Create: `frontend/src/components/work-log/TodayPanorama.spec.ts`

- [ ] **Step 1: 写失败测试**

创建 `frontend/src/components/work-log/TodayPanorama.spec.ts`：

```ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import TodayPanorama from './TodayPanorama.vue'

vi.mock('@/api/client', () => ({
  api: {
    deleteWorkItem: vi.fn().mockResolvedValue({ data: { ok: true } }),
    getWorkLog: vi.fn().mockResolvedValue({ data: { id: 'l1', date: '2026-08-02', items: [] } }),
    getTodayContext: vi.fn(),
  },
}))
vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn() },
}))

import { useWorkLogStore } from '@/stores/workLog'
import type { WorkLog, WorkItem } from '@/types'

function makeItem(over: Partial<WorkItem>): WorkItem {
  return {
    id: 'x', work_log_id: 'l1', seq: 1, title: '', content: '',
    problem_solved: '', result: '', impact: '', source: 'manual',
    activity: 'test', start_time: '09:00', end_time: '10:00', quadrant: 1,
    ...over,
  } as WorkItem
}

describe('TodayPanorama', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  it('shows empty state when no manual items', () => {
    const store = useWorkLogStore()
    store.currentLog = { id: 'l1', date: '2026-08-02', items: [] } as WorkLog
    const wrapper = mount(TodayPanorama, {
      props: { date: '2026-08-02' },
    })
    expect(wrapper.text()).toContain('还没有记录')
  })

  it('renders manual items sorted by time', () => {
    const store = useWorkLogStore()
    store.currentLog = {
      id: 'l1', date: '2026-08-02', items: [
        makeItem({ id: '1', start_time: '10:00', end_time: '11:00', activity: 'b' }),
        makeItem({ id: '2', start_time: '09:00', end_time: '10:00', activity: 'a' }),
      ],
    } as WorkLog
    const wrapper = mount(TodayPanorama, {
      props: { date: '2026-08-02' },
    })
    const rows = wrapper.findAll('[data-test="row"]')
    expect(rows).toHaveLength(2)
    // 09:00 应该排第一
    expect(rows[0].text()).toContain('a')
    expect(rows[0].text()).toContain('09:00')
  })

  it('emits edit with item id when edit clicked', async () => {
    const store = useWorkLogStore()
    store.currentLog = {
      id: 'l1', date: '2026-08-02', items: [
        makeItem({ id: 'x1', activity: '晨会' }),
      ],
    } as WorkLog
    const wrapper = mount(TodayPanorama, {
      props: { date: '2026-08-02' },
    })
    await wrapper.find('[data-test="edit-btn"]').trigger('click')
    expect(wrapper.emitted('edit')?.[0]).toEqual(['x1'])
  })
})
```

- [ ] **Step 2: 运行验证失败**

Run: `cd frontend && npx vitest run src/components/work-log/TodayPanorama.spec.ts`
Expected: FAIL — module not found

- [ ] **Step 3: 创建组件**

创建 `frontend/src/components/work-log/TodayPanorama.vue`：

```vue
<template>
  <div class="today-panorama">
    <h3 class="panorama-title">今日全景 · {{ date }}</h3>
    <el-table v-if="items.length" :data="items" stripe size="small">
      <el-table-column label="时段" width="160">
        <template #default="{ row }">
          {{ row.start_time }} - {{ row.end_time }}
        </template>
      </el-table-column>
      <el-table-column label="活动" prop="activity" />
      <el-table-column label="象限" width="140">
        <template #default="{ row }">
          <el-tag :color="quadrantColor(row.quadrant)" effect="dark" size="small">
            Q{{ row.quadrant }} {{ quadrantName(row.quadrant) }}
          </el-tag>
        </template>
      </el-table-column>
      <el-table-column label="操作" width="140">
        <template #default="{ row }">
          <el-button data-test="edit-btn" size="small" link @click="$emit('edit', row.id)">编辑</el-button>
          <el-popconfirm title="确定删除？" @confirm="onDelete(row.id)">
            <template #reference>
              <el-button data-test="delete-btn" size="small" link type="danger">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </el-table-column>
    </el-table>
    <el-empty v-else description="还没有记录，先在上面录入一条吧" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useWorkLogStore } from '@/stores/workLog'
import { QUADRANT_INFO } from '@/types'
import type { Quadrant } from '@/types'

const props = defineProps<{ date: string }>()
defineEmits<{ edit: [itemId: string] }>()

const store = useWorkLogStore()

const items = computed(() => {
  // todayManualItems 已经过滤+排序，直接用
  return store.todayManualItems
})

function quadrantColor(q: number | null | undefined): string {
  if (!q) return '#6b7280'
  return QUADRANT_INFO[q as Quadrant].color
}

function quadrantName(q: number | null | undefined): string {
  if (!q) return ''
  return QUADRANT_INFO[q as Quadrant].name
}

async function onDelete(itemId: string) {
  await store.deleteQuickEntry(props.date, itemId)
}
</script>

<style scoped>
.today-panorama {
  background: var(--bg-card, #FFFEFC);
  border: 1px solid var(--border-color, #e5e5e5);
  border-radius: 8px;
  padding: 16px;
  margin-bottom: 16px;
}
.panorama-title {
  font-family: var(--font-display);
  font-size: 16px;
  font-weight: 600;
  margin: 0 0 12px 0;
  color: var(--text-primary);
}
</style>
```

注意测试要给 `data-test="row"`——el-table 默认不会给 row 加这个 attr。需要用 `row-class-name`：

在 el-table 上加 `:row-class-name="() => 'pano-row'"`，测试用 `.pano-row` 选择行。或者更简单，改测试断言用 `wrapper.findAll('.el-table__row')`：

修改测试第 2 个用例最后两行：

```ts
const rows = wrapper.findAll('.el-table__row')
```

(`el-table__row` 是 Element Plus 默认 row class，不需额外配置)

- [ ] **Step 4: 运行测试**

Run: `cd frontend && npx vitest run src/components/work-log/TodayPanorama.spec.ts`
Expected: 3 个测试 PASS

- [ ] **Step 5: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无错误

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/work-log/TodayPanorama.vue frontend/src/components/work-log/TodayPanorama.spec.ts
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add TodayPanorama component"
```

---

## Task 12: 接线到 WorkLog.vue

**Files:**
- Modify: `frontend/src/views/WorkLog.vue`

- [ ] **Step 1: 在 script setup 顶部加 import**

修改 `frontend/src/views/WorkLog.vue` 第 56-61 行的 import 块，在 `ReportDetail` 之后加：

```ts
import QuickEntryForm from '@/components/work-log/QuickEntryForm.vue'
import TodayPanorama from '@/components/work-log/TodayPanorama.vue'
```

- [ ] **Step 2: 加 editing 状态**

在 `const currentDate = ref(...)` 之后加：

```ts
const editingItemId = ref<string | null>(null)
```

- [ ] **Step 3: 加 handler 函数**

在 `onSave` 之后加：

```ts
function onPanoramaEdit(itemId: string) {
  editingItemId.value = itemId
}

function onEditCancel() {
  editingItemId.value = null
}

function onEditSaved() {
  editingItemId.value = null
}

function onQuickAdded() {
  // store 已经 fetchLog 过，currentLog 已更新；panorama 通过 computed 自动刷新
}
```

- [ ] **Step 4: 在 template 里插入新组件**

修改 `frontend/src/views/WorkLog.vue` 第 21 行 `<template v-if="!store.selected || store.selected.kind === 'log'">` 之后、`<TodayContextCard>` 之前，插入：

```vue
          <QuickEntryForm
            v-if="!editingItemId"
            :date="currentDate"
            mode="add"
            @added="onQuickAdded"
          />
          <QuickEntryForm
            v-else
            :key="editingItemId"
            :date="currentDate"
            mode="edit"
            :item-id="editingItemId"
            :initial="editingInitial"
            @saved="onEditSaved"
            @cancel="onEditCancel"
          />
          <TodayPanorama :date="currentDate" @edit="onPanoramaEdit" />
```

- [ ] **Step 5: 加 editingInitial computed**

在 `editingItemId` 之后加：

```ts
const editingInitial = computed(() => {
  if (!editingItemId.value || !store.currentLog) return {}
  const it = store.currentLog.items.find(i => i.id === editingItemId.value)
  if (!it) return {}
  return {
    activity: it.activity ?? '',
    start_time: it.start_time ?? '09:00',
    end_time: it.end_time ?? '10:00',
    quadrant: it.quadrant ?? 2,
  }
})
```

- [ ] **Step 6: 类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无错误

- [ ] **Step 7: 启动全栈手测**

Run（一个 shell）:
```bash
cd backend && CGO_ENABLED=1 go run cmd/server/main.go
```

Run（另一个 shell）:
```bash
cd frontend && npm run dev
```

打开浏览器到 `http://localhost:5173/work-log`，验证：
1. 页面顶部出现「快捷录入表单」+「今日全景」两块
2. 默认日期是今天、象限默认 Q2
3. 填一条（活动=测试、9:00-10:00、Q1）→ 点添加 → 看到 ElMessage「已添加」+ 全景表出现该条
4. 点编辑 → 表单切到编辑态，预填字段 → 改活动名称 → 保存 → 全景表更新
5. 点删除 → 确认 → 全景表移除
6. 在脑暴区跑 AI 拆条（可选）→ 保存日报 → **回到全景表确认 manual 条目还在**（关键不变式）

- [ ] **Step 8: Commit**

```bash
git add frontend/src/views/WorkLog.vue
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): wire QuickEntryForm + TodayPanorama into WorkLog view"
```

---

## Task 13: 全量回归 + 收尾

**Files:** 无修改

- [ ] **Step 1: 后端全量测试**

Run: `cd backend && go test ./...`
Expected: 全 PASS

- [ ] **Step 2: 前端全量测试**

Run: `cd frontend && npx vitest run`
Expected: 全 PASS（包括新加的 3 个 spec 文件 + 现有所有）

- [ ] **Step 3: 前端类型检查**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: 无错误

- [ ] **Step 4: 前端构建**

Run: `cd frontend && npm run build`
Expected: 构建成功生成 dist/

- [ ] **Step 5: 更新 AGENTS.md（如有必要）**

如果新增的组件 / 接口需要在 AGENTS.md 的 Code map 里登记，做相应更新。检查：

```bash
git diff main..HEAD --stat
```

如果有非琐碎的目录结构变化，更新 `AGENTS.md` 的 Repository Structure / Module Dependency Graph 段落。

- [ ] **Step 6: 收尾 commit（如有文档改动）**

```bash
git add AGENTS.md
git -c user.name="lsy" -c user.email="lsy@local" commit -m "docs(work-log): update AGENTS.md with quick-entry module additions"
```

如果无需更新，跳过此步。

---

## 验收清单（手测脚本）

按下列顺序确认所有功能：

1. ✅ 工作日志页面顶部出现「快捷录入表单」+「今日全景」两块，且都在 TodayContextCard 之前
2. ✅ 表单字段：日期（默认今天）/ 活动（空）/ 时段（默认 09:00-10:00）/ 象限（默认 Q2）
3. ✅ 提交空活动 → 报错，不发请求
4. ✅ 提交 end <= start → 报错
5. ✅ 提交合法 → 201 + 全景表新增一行 + ElMessage「已添加」+ 活动字段清空（其他保留）
6. ✅ 全景表按时段升序排列
7. ✅ 点编辑 → 切换到编辑表单（取消按钮可见）→ 改字段 → 保存 → 全景表更新
8. ✅ 点删除 → 弹确认 → 确认后删除 → 全景表移除
9. ✅ **关键不变式**：先加快捷录入条目 → 跑脑暴 AI → 保存日报 → 全景表的快捷条目仍存在
10. ✅ 切到其他日期 → 表单/全景跟随切换
11. ✅ 后端、前端测试全绿，类型检查无错误

---

## 风险与回滚

- **最大风险**：Task 4 的 UpsertWorkLog 改造。回归测试 `TestUpsertWorkLog_PreservesManualItems` 是关键守门。如果手测发现 manual 条目被擦，立即回滚 commit 4。
- **次要风险**：el-time-select 的 step/start 配置在不同 Element Plus 版本表现不一。如果时段选择异常，参考 `TaskForm.vue` 的现有用法。
- **回滚单元**：每个 Task 是独立 commit，可 `git revert <hash>` 精确回滚。
