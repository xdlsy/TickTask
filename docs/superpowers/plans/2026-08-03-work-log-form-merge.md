# 工作日志合并表单 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 把工作日志页面顶部的 `QuickEntryForm` 和底部的 `WorkItemList`/`WorkItemEditor` 合并为一个统一的 `WorkItemForm`（4 必填 + 4 可选）+ 一个表格式 `BatchTableEditor`（承接 AI 脑暴批量入库）；后端扩展 DTO 接受可选字段、加 `PATCH /work-logs/:date/summary` 端点、AI prompt 去掉 title。

**Architecture:** 主表单立即落库（沿用 `POST /work-logs/:date/items`）；AI 批量入库循环调用同一端点；summary 走独立 PATCH 避免覆盖 items；service 层在 add/update 时把 `activity` 同步写入 `title` 字段；启动时一次性 SQLite UPDATE 把旧 manual items 的 `title` 从 `activity` 回填。

**Tech Stack:** Go 1.21 + Gin + GORM + SQLite | Vue 3.5 + Pinia (Composition API) + Element Plus 2.8 + TypeScript 5.6 strict | Vitest 2.1 + happy-dom

**Spec:** `docs/superpowers/specs/2026-08-03-work-log-form-merge-design.md`

**Branch:** `evolve/work-log-quick-entry`（沿用）

---

## File Structure

**Backend (modify):**
- `backend/internal/repository/work_log_repo.go` — 接口加 `UpdateWorkLogSummary` 方法
- `backend/internal/repository/work_log_repo_test.go` — 加 `UpdateWorkLogSummary` 测试
- `backend/internal/service/work_log_service.go` — DTO 加 4 可选字段；`AddQuickEntry`/`UpdateQuickEntry` 同步 title + 写可选字段；新增 `UpdateSummary` 方法；`StructuredItem` 去掉 Title
- `backend/internal/service/work_log_service_quick_test.go` — 扩展 quick entry 测试
- `backend/internal/service/work_log_service_test.go` — 加 `UpdateSummary` 测试
- `backend/internal/service/work_log_ai_client_test.go` — 调整 mock 响应去掉 title
- `backend/internal/ai/work_log_prompts.go` — `WorkLogStructureSystem` 输出去掉 title
- `backend/internal/api/handler/work_log.go` — DTO 加 4 可选字段；新增 `UpdateSummary` handler
- `backend/internal/api/handler/work_log_test.go` — 加可选字段绑定 + UpdateSummary 测试
- `backend/internal/api/router.go` — 注册 `PATCH /work-logs/:date/summary`
- `backend/pkg/database/db.go` — `Init` 调用 `MigrateWorkItemsTitleBackfill`

**Backend (create):**
- `backend/pkg/database/migrate_work_items_title.go` — 一次性回填函数
- `backend/pkg/database/migrate_work_items_title_test.go` — 幂等性测试

**Frontend (modify):**
- `frontend/src/types/index.ts` — `CreateQuickEntryInput`/`UpdateQuickEntryInput` 加 4 可选字段；`StructuredItem` 去掉 title；新增 `UpdateSummaryInput`
- `frontend/src/api/client.ts` — 加 `updateWorkLogSummary` 方法
- `frontend/src/stores/workLog.ts` — 加 `addWorkItemsBatch` + `updateSummary` actions
- `frontend/src/stores/workLog.spec.ts` — 加批量入库 + summary 测试
- `frontend/src/views/WorkLog.vue` — 替换为新组件接线

**Frontend (create):**
- `frontend/src/components/work-log/WorkItemForm.vue` — 主表单（add/edit 双模式）
- `frontend/src/components/work-log/WorkItemForm.spec.ts`
- `frontend/src/components/work-log/BatchTableEditor.vue` — AI 草稿批量编辑
- `frontend/src/components/work-log/BatchTableEditor.spec.ts`

**Frontend (delete):**
- `frontend/src/components/work-log/QuickEntryForm.vue`
- `frontend/src/components/work-log/QuickEntryForm.spec.ts`
- `frontend/src/components/work-log/WorkItemList.vue`
- `frontend/src/components/work-log/WorkItemEditor.vue`

---

## Phase A — Backend (TDD)

### Task 1: Migration helper for title backfill

**Files:**
- Create: `backend/pkg/database/migrate_work_items_title.go`
- Test: `backend/pkg/database/migrate_work_items_title_test.go`

- [ ] **Step 1: Write failing test**

Create `backend/pkg/database/migrate_work_items_title_test.go`:

```go
package database

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"ticktask/internal/model"
)

func newMigrateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkLog{}, &model.WorkItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestMigrateWorkItemsTitleBackfill_FillsEmptyTitleFromActivity(t *testing.T) {
	db := newMigrateTestDB(t)
	activity := "晨会"
	db.Create(&model.WorkLog{ID: "wl-1", Date: "2026-08-01"})
	db.Create(&model.WorkItem{
		ID: "wi-1", WorkLogID: "wl-1", Seq: 1,
		Title: "", Activity: &activity, Source: "manual",
	})

	if err := MigrateWorkItemsTitleBackfill(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var got model.WorkItem
	db.First(&got, "id = ?", "wi-1")
	if got.Title != activity {
		t.Fatalf("expected title=%q, got %q", activity, got.Title)
	}
}

func TestMigrateWorkItemsTitleBackfill_SkipsAIItems(t *testing.T) {
	db := newMigrateTestDB(t)
	db.Create(&model.WorkLog{ID: "wl-1", Date: "2026-08-01"})
	db.Create(&model.WorkItem{
		ID: "wi-1", WorkLogID: "wl-1", Seq: 1,
		Title: "AI 生成的标题", Source: "ai", // Activity nil
	})

	if err := MigrateWorkItemsTitleBackfill(db); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var got model.WorkItem
	db.First(&got, "id = ?", "wi-1")
	if got.Title != "AI 生成的标题" {
		t.Fatalf("AI item title should be untouched, got %q", got.Title)
	}
}

func TestMigrateWorkItemsTitleBackfill_Idempotent(t *testing.T) {
	db := newMigrateTestDB(t)
	activity := "晨会"
	db.Create(&model.WorkLog{ID: "wl-1", Date: "2026-08-01"})
	db.Create(&model.WorkItem{
		ID: "wi-1", WorkLogID: "wl-1", Seq: 1,
		Title: "", Activity: &activity, Source: "manual",
	})

	// 第一次
	if err := MigrateWorkItemsTitleBackfill(db); err != nil {
		t.Fatalf("first migrate: %v", err)
	}
	var got model.WorkItem
	db.First(&got, "id = ?", "wi-1")
	firstTitle := got.Title

	// 第二次
	if err := MigrateWorkItemsTitleBackfill(db); err != nil {
		t.Fatalf("second migrate: %v", err)
	}
	db.First(&got, "id = ?", "wi-1")
	if got.Title != firstTitle {
		t.Fatalf("second run changed title: %q → %q", firstTitle, got.Title)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./pkg/database/... -run TestMigrateWorkItemsTitleBackfill -v
```

Expected: build failure `undefined: MigrateWorkItemsTitleBackfill`.

- [ ] **Step 3: Implement migration**

Create `backend/pkg/database/migrate_work_items_title.go`:

```go
package database

import "gorm.io/gorm"

// MigrateWorkItemsTitleBackfill 把 source='manual' 且 title 为空但 activity 非空的 items 的 title 回填为 activity。
// 幂等：WHERE 条件保证重复调用不会覆盖已填好的 title。
func MigrateWorkItemsTitleBackfill(db *gorm.DB) error {
	return db.Exec(`
		UPDATE work_items
		SET title = activity
		WHERE source = 'manual'
		  AND (title = '' OR title IS NULL)
		  AND activity IS NOT NULL
	`).Error
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./pkg/database/... -run TestMigrateWorkItemsTitleBackfill -v
```

Expected: 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/pkg/database/migrate_work_items_title.go backend/pkg/database/migrate_work_items_title_test.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add MigrateWorkItemsTitleBackfill helper"
```

---

### Task 2: Wire migration into database.Init

**Files:**
- Modify: `backend/pkg/database/db.go`

- [ ] **Step 1: Modify Init to call migration after AutoMigrate**

In `backend/pkg/database/db.go`, modify the `Init` function. Replace lines 14-28 (the whole `Init` function):

```go
// Init 初始化数据库连接
func Init(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	// 自动迁移表结构
	if err := AutoMigrate(db); err != nil {
		return nil, err
	}

	// 一次性数据迁移：旧 manual items 的 title 从 activity 回填
	if err := MigrateWorkItemsTitleBackfill(db); err != nil {
		return nil, err
	}

	return db, nil
}
```

- [ ] **Step 2: Verify build**

```bash
cd backend && go build ./...
```

Expected: no output (success).

- [ ] **Step 3: Commit**

```bash
git add backend/pkg/database/db.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): wire MigrateWorkItemsTitleBackfill into Init"
```

---

### Task 3: Repository UpdateWorkLogSummary method

**Files:**
- Modify: `backend/internal/repository/work_log_repo.go` (interface + impl)
- Test: `backend/internal/repository/work_log_repo_test.go`

- [ ] **Step 1: Write failing test**

Append to `backend/internal/repository/work_log_repo_test.go` (create the file if it doesn't exist):

```go
package repository

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"ticktask/internal/model"
)

func newRepoTestDB(t *testing.T) (*gorm.DB, WorkLogRepository) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&model.WorkLog{}, &model.WorkItem{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db, NewWorkLogRepository(db)
}

func TestUpdateWorkLogSummary_UpdatesExistingLog(t *testing.T) {
	_, repo := newRepoTestDB(t)
	repo.CreateWorkLog(&model.WorkLog{ID: "wl-1", Date: "2026-08-01", Summary: "old"})

	err := repo.UpdateWorkLogSummary("2026-08-01", "new summary")
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	log, _ := repo.GetWorkLogByDate("2026-08-01")
	if log.Summary != "new summary" {
		t.Fatalf("expected 'new summary', got %q", log.Summary)
	}
}

func TestUpdateWorkLogSummary_ReturnsNotFoundForMissingLog(t *testing.T) {
	_, repo := newRepoTestDB(t)
	err := repo.UpdateWorkLogSummary("2026-99-99", "x")
	if err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/repository/... -run TestUpdateWorkLogSummary -v
```

Expected: build failure `undefined: ...UpdateWorkLogSummary`.

- [ ] **Step 3: Add to interface and implement**

In `backend/internal/repository/work_log_repo.go`, modify the `WorkLogRepository` interface (insert after `UpsertWorkLog` line ~18):

```go
// WorkLog CRUD
CreateWorkLog(log *model.WorkLog) error
GetWorkLogByDate(date string) (*model.WorkLog, error)
GetWorkLogsInRange(from, to string) ([]*model.WorkLog, error)
UpsertWorkLog(log *model.WorkLog) error
UpdateWorkLogSummary(date string, summary string) error // 新增

// WorkItem
...
```

Append at end of file (after `DeleteItem`):

```go
func (r *workLogRepository) UpdateWorkLogSummary(date string, summary string) error {
	result := r.db.Model(&model.WorkLog{}).Where("date = ?", date).Update("summary", summary)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/repository/... -run TestUpdateWorkLogSummary -v
```

Expected: 2 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/repository/work_log_repo.go backend/internal/repository/work_log_repo_test.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add UpdateWorkLogSummary to repository"
```

---

### Task 4: Service UpdateSummary method

**Files:**
- Modify: `backend/internal/service/work_log_service.go`
- Test: `backend/internal/service/work_log_service_test.go`

- [ ] **Step 1: Write failing test**

Append to `backend/internal/service/work_log_service_test.go`:

```go
func TestUpdateSummary_UpdatesSummary(t *testing.T) {
	svc := newQuickService(t)
	// 先建一个 WorkLog
	svc.AddQuickEntry("2026-08-01", CreateQuickEntryInput{
		Activity: "x", StartTime: "09:00", EndTime: "10:00", Quadrant: 1,
	})

	err := svc.UpdateSummary("2026-08-01", "今日小结文本")
	if err != nil {
		t.Fatalf("update summary: %v", err)
	}

	log, _ := svc.repo.GetWorkLogByDate("2026-08-01")
	if log.Summary != "今日小结文本" {
		t.Fatalf("expected summary updated, got %q", log.Summary)
	}
}

func TestUpdateSummary_MissingWorkLogReturnsNotFound(t *testing.T) {
	svc := newQuickService(t)
	err := svc.UpdateSummary("2026-99-99", "x")
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestUpdateSummary_InvalidDateReturnsBadRequest(t *testing.T) {
	svc := newQuickService(t)
	err := svc.UpdateSummary("not-a-date", "x")
	if err == nil || !strings.HasPrefix(err.Error(), "invalid date:") {
		t.Fatalf("expected invalid date error, got %v", err)
	}
}
```

If `errors` and `strings` aren't imported in this test file, add to its import block.

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/service/... -run TestUpdateSummary -v
```

Expected: build failure `undefined: ...UpdateSummary`.

- [ ] **Step 3: Add service method**

In `backend/internal/service/work_log_service.go`, append at the end of the file:

```go
// UpdateSummary 仅更新 WorkLog.summary 字段，不动 items。
// 若 WorkLog 不存在返回 ErrNotFound（不自动创建，避免空日报）。
func (s *WorkLogService) UpdateSummary(date string, summary string) error {
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return fmt.Errorf("invalid date: %w", err)
	}
	return s.repo.UpdateWorkLogSummary(date, summary)
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/service/... -run TestUpdateSummary -v
```

Expected: 3 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/work_log_service.go backend/internal/service/work_log_service_test.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add UpdateSummary service method"
```

---

### Task 5: Extend CreateQuickEntryInput + AddQuickEntry title sync

**Files:**
- Modify: `backend/internal/service/work_log_service.go`
- Test: `backend/internal/service/work_log_service_quick_test.go`

- [ ] **Step 1: Write failing test**

Append to `backend/internal/service/work_log_service_quick_test.go`:

```go
func TestAddQuickEntry_WithOptionalFields_SyncsTitle(t *testing.T) {
	svc := newQuickService(t)
	in := CreateQuickEntryInput{
		Activity:      "登录开发",
		StartTime:     "09:00",
		EndTime:       "10:00",
		Quadrant:      2,
		Content:       "完成 OAuth 联调",
		ProblemSolved: "回调失败",
		Result:        "成功率提升",
		Impact:        "释放测试资源",
	}
	item, err := svc.AddQuickEntry("2026-08-03", in)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if item.Title != "登录开发" {
		t.Fatalf("expected title='登录开发', got %q", item.Title)
	}
	if item.Content != "完成 OAuth 联调" {
		t.Fatalf("expected content set, got %q", item.Content)
	}
	if item.ProblemSolved != "回调失败" {
		t.Fatalf("expected problem_solved set, got %q", item.ProblemSolved)
	}
	if item.Result != "成功率提升" {
		t.Fatalf("expected result set, got %q", item.Result)
	}
	if item.Impact != "释放测试资源" {
		t.Fatalf("expected impact set, got %q", item.Impact)
	}
}

func TestAddQuickEntry_NoOptionalFields_PersistsWithEmptyStrings(t *testing.T) {
	svc := newQuickService(t)
	in := CreateQuickEntryInput{
		Activity:  "x",
		StartTime: "09:00",
		EndTime:   "10:00",
		Quadrant:  1,
	}
	item, err := svc.AddQuickEntry("2026-08-03", in)
	if err != nil {
		t.Fatalf("add: %v", err)
	}
	if item.Title != "x" {
		t.Fatalf("expected title synced from activity, got %q", item.Title)
	}
	if item.Content != "" || item.ProblemSolved != "" || item.Result != "" || item.Impact != "" {
		t.Fatalf("optional fields should be empty, got %+v", item)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/service/... -run TestAddQuickEntry_With -v
```

Expected: compile failure (CreateQuickEntryInput has no `Content` etc. fields).

- [ ] **Step 3: Extend DTO and AddQuickEntry**

In `backend/internal/service/work_log_service.go`, replace the `CreateQuickEntryInput` struct (lines ~93-98):

```go
// CreateQuickEntryInput 快捷录入新增输入
type CreateQuickEntryInput struct {
	Activity      string `json:"activity"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	Quadrant      int    `json:"quadrant"`
	Content       string `json:"content"`        // 可选
	ProblemSolved string `json:"problem_solved"` // 可选
	Result        string `json:"result"`         // 可选
	Impact        string `json:"impact"`         // 可选
}
```

In the same file, in `AddQuickEntry` method (around line 573), replace the `item := model.WorkItem{...}` block:

```go
	item := model.WorkItem{
		ID:            s.idGenerator(),
		WorkLogID:     log.ID,
		Seq:           maxSeq + 1,
		Title:         in.Activity, // 同步：title = activity
		Content:       in.Content,
		ProblemSolved: in.ProblemSolved,
		Result:        in.Result,
		Impact:        in.Impact,
		Activity:      &in.Activity,
		StartTime:     &in.StartTime,
		EndTime:       &in.EndTime,
		Quadrant:      &in.Quadrant,
		Source:        "manual",
	}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd backend && go test ./internal/service/... -run TestAddQuickEntry_With -v
cd backend && go test ./internal/service/... -run TestAddQuickEntry_No -v
```

Expected: 2 tests PASS.

Also run existing quick entry tests to verify no regression:

```bash
cd backend && go test ./internal/service/... -run TestAddQuickEntry -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/work_log_service.go backend/internal/service/work_log_service_quick_test.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): extend CreateQuickEntryInput with optional fields, sync title=activity"
```

---

### Task 6: Extend UpdateQuickEntryInput + activity syncs title

**Files:**
- Modify: `backend/internal/service/work_log_service.go`
- Test: `backend/internal/service/work_log_service_quick_test.go`

- [ ] **Step 1: Write failing test**

Append to `backend/internal/service/work_log_service_quick_test.go`:

```go
func TestUpdateQuickEntry_ActivityChangeSyncsTitle(t *testing.T) {
	svc := newQuickService(t)
	item, _ := svc.AddQuickEntry("2026-08-03", CreateQuickEntryInput{
		Activity: "old", StartTime: "09:00", EndTime: "10:00", Quadrant: 1,
	})

	newActivity := "new activity"
	err := svc.UpdateQuickEntry("2026-08-03", item.ID, UpdateQuickEntryInput{
		Activity: &newActivity,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	log, _ := svc.repo.GetWorkLogByDate("2026-08-03")
	for _, it := range log.Items {
		if it.ID == item.ID {
			if it.Title != "new activity" {
				t.Fatalf("expected title synced, got %q", it.Title)
			}
			if it.Activity == nil || *it.Activity != "new activity" {
				t.Fatalf("expected activity updated, got %v", it.Activity)
			}
		}
	}
}

func TestUpdateQuickEntry_OptionalFieldsUpdate(t *testing.T) {
	svc := newQuickService(t)
	item, _ := svc.AddQuickEntry("2026-08-03", CreateQuickEntryInput{
		Activity: "x", StartTime: "09:00", EndTime: "10:00", Quadrant: 1,
	})

	newContent := "新内容"
	newResult := "新结果"
	err := svc.UpdateQuickEntry("2026-08-03", item.ID, UpdateQuickEntryInput{
		Content: &newContent,
		Result:  &newResult,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	log, _ := svc.repo.GetWorkLogByDate("2026-08-03")
	for _, it := range log.Items {
		if it.ID == item.ID {
			if it.Content != "新内容" {
				t.Fatalf("expected content updated, got %q", it.Content)
			}
			if it.Result != "新结果" {
				t.Fatalf("expected result updated, got %q", it.Result)
			}
		}
	}
}

func TestUpdateQuickEntry_ClearOptionalFieldWithEmptyString(t *testing.T) {
	svc := newQuickService(t)
	item, _ := svc.AddQuickEntry("2026-08-03", CreateQuickEntryInput{
		Activity: "x", StartTime: "09:00", EndTime: "10:00", Quadrant: 1,
		Content: "原内容",
	})

	emptyStr := ""
	err := svc.UpdateQuickEntry("2026-08-03", item.ID, UpdateQuickEntryInput{
		Content: &emptyStr,
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}

	log, _ := svc.repo.GetWorkLogByDate("2026-08-03")
	for _, it := range log.Items {
		if it.ID == item.ID {
			if it.Content != "" {
				t.Fatalf("expected content cleared, got %q", it.Content)
			}
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/service/... -run TestUpdateQuickEntry_ActivityChangeSyncsTitle -v
```

Expected: compile failure (UpdateQuickEntryInput has no `Content` field).

- [ ] **Step 3: Extend DTO and UpdateQuickEntry**

In `backend/internal/service/work_log_service.go`, replace `UpdateQuickEntryInput` struct (lines ~101-106):

```go
// UpdateQuickEntryInput 快捷录入编辑输入（指针 = 部分更新）
type UpdateQuickEntryInput struct {
	Activity      *string `json:"activity,omitempty"`
	StartTime     *string `json:"start_time,omitempty"`
	EndTime       *string `json:"end_time,omitempty"`
	Quadrant      *int    `json:"quadrant,omitempty"`
	Content       *string `json:"content,omitempty"`
	ProblemSolved *string `json:"problem_solved,omitempty"`
	Result        *string `json:"result,omitempty"`
	Impact        *string `json:"impact,omitempty"`
}
```

In the same file, in `UpdateQuickEntry` method (around line 620), replace the `updates := map[string]any{}` block through the end of updates building (before `if len(updates) == 0`):

```go
	updates := map[string]any{}
	if in.Activity != nil {
		updates["activity"] = *in.Activity
		updates["title"] = *in.Activity // 同步：title 跟随 activity
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
	if in.Content != nil {
		updates["content"] = *in.Content
	}
	if in.ProblemSolved != nil {
		updates["problem_solved"] = *in.ProblemSolved
	}
	if in.Result != nil {
		updates["result"] = *in.Result
	}
	if in.Impact != nil {
		updates["impact"] = *in.Impact
	}
```

- [ ] **Step 4: Run all UpdateQuickEntry tests**

```bash
cd backend && go test ./internal/service/... -run TestUpdateQuickEntry -v
```

Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/service/work_log_service.go backend/internal/service/work_log_service_quick_test.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): extend UpdateQuickEntryInput, sync title on activity change"
```

---

### Task 7: AI prompt drop title + adjust StructuredItem

**Files:**
- Modify: `backend/internal/ai/work_log_prompts.go`
- Modify: `backend/internal/service/work_log_service.go`
- Test: `backend/internal/service/work_log_ai_client_test.go`

- [ ] **Step 1: Write failing test**

Open `backend/internal/service/work_log_ai_client_test.go` and find the mock AI response used for StructureBrainDump tests. Replace any `"title": "..."` field in mock responses with no title field.

If the test asserts `items[0].Title != ""`, change it to assert `items[0].Content != ""` (the mock should still have content). Add this new explicit test:

```go
func TestStructureBrainDump_ResponseHasNoTitleField(t *testing.T) {
	// mock client 返回不含 title 字段
	mockClient := &mockAIClient{
		structureResp: &StructuredWorkLog{
			Items: []StructuredItem{
				{Content: "做了 X", ProblemSolved: "解决了 Y", Result: "产出了 Z", Impact: "推进了 W"},
			},
			Summary: "今日小结",
		},
	}
	svc := NewWorkLogService(nil, nil, nil, mockClient)
	out, err := svc.StructureBrainDump(BrainDumpInput{BrainDump: "text"})
	if err != nil {
		t.Fatalf("structure: %v", err)
	}
	if len(out.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(out.Items))
	}
	if out.Items[0].Content != "做了 X" {
		t.Fatalf("expected content, got %q", out.Items[0].Content)
	}
	// StructuredItem 不再有 Title 字段——这本身是编译期保证
}
```

> 注：若 mock 类型/字段名与现有不同，按现有 mockAIClient 的命名风格调整。重点：mock 响应不含 title 字段。

- [ ] **Step 2: Run test to verify it fails**

```bash
cd backend && go test ./internal/service/... -run TestStructureBrainDump_ResponseHasNoTitleField -v
```

Expected: compile failure (`StructuredItem` 还存在 Title 字段，且 mock 响应里可能仍传 title)。

- [ ] **Step 3: Drop title from StructuredItem and update prompt**

In `backend/internal/service/work_log_service.go`, replace the `StructuredItem` struct (lines ~53-59):

```go
// StructuredItem AI 拆条输出的单条工作（4 维，无 title——activity 由用户在批量表中补齐）
type StructuredItem struct {
	Content       string `json:"content"`
	ProblemSolved string `json:"problem_solved"`
	Result        string `json:"result"`
	Impact        string `json:"impact"`
}
```

In `backend/internal/ai/work_log_prompts.go`, replace the `WorkLogStructureSystem` const (entire value, lines 4-35). 关键修改是 items 元素去掉 title 字段，并在拆条原则里说明 activity 由用户填写：

```go
const WorkLogStructureSystem = `你是一个工作日报整理助手。用户会提供一段"今日工作脑暴"（自由文本），以及今日的预填上下文（已完成的任务、番茄钟会话）。

任务：把脑暴拆成若干"核心工作"条目，每条按四维结构展开。每条目应能被 5~15 字的活动名概括（用户会在批量入库时填写 activity 字段，AI 不生成 activity）。

# 输出格式（严格 JSON，不要 markdown 代码块包裹）

{
  "items": [
    {
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
```

Also fix any other place in `work_log_service.go` or `work_log_ai_client.go` that references `StructuredItem.Title`. Search and remove.

```bash
cd backend && grep -rn "\.Title" internal/service/work_log_ai_client*.go internal/service/work_log_service.go
```

If any reference to `.Title` on a StructuredItem remains, remove or rename. (Most likely just the struct itself.)

- [ ] **Step 4: Run all AI client tests**

```bash
cd backend && go test ./internal/service/... -run TestStructureBrainDump -v
```

Expected: all PASS.

```bash
cd backend && go test ./internal/service/... -v
```

Expected: ALL tests PASS (no regressions).

- [ ] **Step 5: Commit**

```bash
git add backend/internal/ai/work_log_prompts.go backend/internal/service/work_log_service.go backend/internal/service/work_log_ai_client_test.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): drop title from AI structured output (4-dim only)"
```

---

### Task 8: Handler extend DTOs + UpdateSummary handler

**Files:**
- Modify: `backend/internal/api/handler/work_log.go`
- Test: `backend/internal/api/handler/work_log_test.go`

- [ ] **Step 1: Write failing tests**

Append to `backend/internal/api/handler/work_log_test.go`. The test for `AddQuickEntry` with optional fields:

```go
func TestAddQuickEntry_WithOptionalFields(t *testing.T) {
	svc := newMockService(t) // 复用文件里已有的 mock service 构造
	// 假设现有 mock 已经能记录最后一次 AddQuickEntry 的入参；若没有，按现有 mockService 风格扩展
	body := `{
		"activity": "x",
		"start_time": "09:00",
		"end_time": "10:00",
		"quadrant": 1,
		"content": "可选内容",
		"problem_solved": "解决了 Y",
		"result": "产出了 Z",
		"impact": "影响了 W"
	}`

	req := httptest.NewRequest("POST", "/api/work-logs/2026-08-03/items", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// 调用 handler（按现有测试如何调用 handler.AddQuickEntry 的方式来）
	// 期望：HTTP 201 + 调用 svc.AddQuickEntry 时 in.Content == "可选内容" 等
	// ...（具体调用方式参考现有 TestAddQuickEntry 测试）
}
```

> 注：现有 handler 测试文件的 mock 风格需要先读。如果用 `mockWorkLogService` 接口记录入参，扩展它让 AddQuickEntry 把入参存到测试可见的字段。具体细节按现有测试结构调整——目标：发可选字段，断言 svc 收到。

Test for `UpdateSummary`:

```go
func TestUpdateSummary_Success(t *testing.T) {
	// 类似现有 PATCH 测试结构
	// POST/PATCH /api/work-logs/2026-08-03/summary body={"summary":"今日小结"}
	// 期望 HTTP 200 + svc.UpdateSummary 被调用，参数 ("2026-08-03", "今日小结")
}

func TestUpdateSummary_MissingLogReturns404(t *testing.T) {
	// svc.UpdateSummary 返回 repository.ErrNotFound
	// 期望 HTTP 404
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd backend && go test ./internal/api/handler/... -run TestAddQuickEntry_WithOptionalFields -v
cd backend && go test ./internal/api/handler/... -run TestUpdateSummary -v
```

Expected: compile failure (handler DTO 缺字段；UpdateSummary handler 不存在)。

- [ ] **Step 3: Extend handler DTOs**

In `backend/internal/api/handler/work_log.go`, replace the `createQuickEntryInput` struct (lines ~206-211):

```go
type createQuickEntryInput struct {
	Activity      string `json:"activity" binding:"required"`
	StartTime     string `json:"start_time" binding:"required"`
	EndTime       string `json:"end_time" binding:"required"`
	Quadrant      int    `json:"quadrant" binding:"required,min=1,max=4"`
	Content       string `json:"content"`
	ProblemSolved string `json:"problem_solved"`
	Result        string `json:"result"`
	Impact        string `json:"impact"`
}
```

Replace the `updateQuickEntryInput` struct (lines ~213-218):

```go
type updateQuickEntryInput struct {
	Activity      *string `json:"activity,omitempty"`
	StartTime     *string `json:"start_time,omitempty"`
	EndTime       *string `json:"end_time,omitempty"`
	Quadrant      *int    `json:"quadrant,omitempty" binding:"omitempty,min=1,max=4"`
	Content       *string `json:"content,omitempty"`
	ProblemSolved *string `json:"problem_solved,omitempty"`
	Result        *string `json:"result,omitempty"`
	Impact        *string `json:"impact,omitempty"`
}
```

In `AddQuickEntry` handler (around line 228), extend the service call:

```go
	item, err := h.svc.AddQuickEntry(date, service.CreateQuickEntryInput{
		Activity:      req.Activity,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		Quadrant:      req.Quadrant,
		Content:       req.Content,
		ProblemSolved: req.ProblemSolved,
		Result:        req.Result,
		Impact:        req.Impact,
	})
```

In `UpdateQuickEntry` handler (around line 251), extend the service call:

```go
	err := h.svc.UpdateQuickEntry(date, itemID, service.UpdateQuickEntryInput{
		Activity:      req.Activity,
		StartTime:     req.StartTime,
		EndTime:       req.EndTime,
		Quadrant:      req.Quadrant,
		Content:       req.Content,
		ProblemSolved: req.ProblemSolved,
		Result:        req.Result,
		Impact:        req.Impact,
	})
```

Add new `UpdateSummary` handler (append before `mapQuickEntryErrorStatus`):

```go
type updateSummaryInput struct {
	Summary string `json:"summary"`
}

// UpdateSummary PATCH /api/work-logs/:date/summary
func (h *WorkLogHandler) UpdateSummary(c *gin.Context) {
	date := c.Param("date")
	var req updateSummaryInput
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	err := h.svc.UpdateSummary(date, req.Summary)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		status := http.StatusInternalServerError
		if strings.HasPrefix(err.Error(), "invalid date:") {
			status = http.StatusBadRequest
		}
		c.JSON(status, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
```

- [ ] **Step 4: Run all handler tests**

```bash
cd backend && go test ./internal/api/handler/... -v
```

Expected: ALL PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/api/handler/work_log.go backend/internal/api/handler/work_log_test.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): extend handler DTOs + add UpdateSummary endpoint"
```

---

### Task 9: Router register PATCH /:date/summary

**Files:**
- Modify: `backend/internal/api/router.go:115-128`

- [ ] **Step 1: Add route registration**

In `backend/internal/api/router.go`, in the `workLogs` group (around line 117-128), insert a new line for the summary endpoint. Place it BEFORE the existing `/:date/items` POST so it's clearly grouped with summary-related routes (no actual route conflict either way):

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
			workLogs.PATCH("/:date/summary", wlHandler.UpdateSummary) // 新增
			// 快捷录入（今日全景）
			workLogs.POST("/:date/items", wlHandler.AddQuickEntry)
			workLogs.PATCH("/:date/items/:itemId", wlHandler.UpdateQuickEntry)
			workLogs.DELETE("/:date/items/:itemId", wlHandler.DeleteQuickEntry)
		}
```

- [ ] **Step 2: Verify build and run server smoke**

```bash
cd backend && go build ./...
```

Expected: success.

- [ ] **Step 3: Verify route registered (manual check via test or curl)**

Add a small smoke test in `backend/internal/api/handler/work_log_test.go` (or extend an existing router test if pattern exists). If no router-level test pattern exists, skip this step—the handler test in Task 8 already covers behavior.

- [ ] **Step 4: Run full backend tests**

```bash
cd backend && go test ./...
```

Expected: ALL PASS.

- [ ] **Step 5: Commit**

```bash
git add backend/internal/api/router.go
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): register PATCH /work-logs/:date/summary route"
```

---

## Phase B — Frontend (TDD)

### Task 10: Extend frontend types

**Files:**
- Modify: `frontend/src/types/index.ts:315-356`

- [ ] **Step 1: Read existing types section**

```bash
# Read frontend/src/types/index.ts lines 315-360 to see current shape
```

- [ ] **Step 2: Modify types**

In `frontend/src/types/index.ts`, find the `CreateQuickEntryInput` interface (around line 333) and replace:

```ts
// 快捷录入新建输入
export interface CreateQuickEntryInput {
  activity: string
  start_time: string
  end_time: string
  quadrant: Quadrant
  content?: string
  problem_solved?: string
  result?: string
  impact?: string
}

// 快捷录入编辑输入（部分更新）
export interface UpdateQuickEntryInput {
  activity?: string
  start_time?: string
  end_time?: string
  quadrant?: Quadrant
  content?: string
  problem_solved?: string
  result?: string
  impact?: string
}

// 新增：summary 单独更新输入
export interface UpdateSummaryInput {
  summary: string
}
```

Find the `StructuredItem` interface (somewhere in the same file) and remove the `title` field:

```ts
// AI 拆条输出（与后端 StructuredItem 对齐）
export interface StructuredItem {
  // title 字段移除——activity 由用户填，title 后端从 activity 同步
  content: string
  problem_solved: string
  result: string
  impact: string
}
```

Also add `UpdateSummaryInput` to the export list if the file uses explicit re-exports (most likely just `export interface` directly).

- [ ] **Step 3: Verify type check**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: no errors. If errors mention `title` not existing on `StructuredItem`, find usages in `WorkLog.vue` / `WorkItemList.vue` / `WorkItemEditor.vue` and remove. (Those files will be deleted in Task 16 anyway, but type check must pass NOW.)

Quick fix if vue-tsc fails on existing components: temporarily leave `title?: string` optional in StructuredItem. Will fully remove in Task 16. **Preferred: leave `title?: string` optional until old components deleted, then remove.** Adjust the StructuredItem interface accordingly:

```ts
export interface StructuredItem {
  title?: string // 废弃，待旧组件删除后移除
  content: string
  problem_solved: string
  result: string
  impact: string
}
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/types/index.ts
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): extend quick entry input types, add UpdateSummaryInput"
```

---

### Task 11: API client add updateWorkLogSummary

**Files:**
- Modify: `frontend/src/api/client.ts:101-106`

- [ ] **Step 1: Read existing client.ts work-log section**

(Already known from exploration; lines 100-106 contain `appendWorkItem` / `updateWorkItem` / `deleteWorkItem`.)

- [ ] **Step 2: Add new method**

In `frontend/src/api/client.ts`, after the existing `deleteWorkItem` line (~106), add a new method inside the `api` object. Also add `UpdateSummaryInput` to the type import at the top of the file.

Update the import (line 2):

```ts
import type { Task, TaskResponse, PomodoroSession, ClassificationResult, PrioritySuggestion, AIStatus, PomodoroSettings, AISettings, TaskTimeStats, DailySummary, TrendData, DistributionStats, ScheduleEvent, CreateScheduleDTO, UpdateScheduleDTO, MoveScheduleDTO, RescheduleResult, DailyInsights, ReviseResponse, PomodoroByTaskResult, PomodoroTrendsResult, WorkLog, WorkItem, WorkReport, WorkReportType, TodayContext, StructuredWorkLog, SaveWorkLogInput, CreateQuickEntryInput, UpdateQuickEntryInput, UpdateSummaryInput } from '@/types'
```

Add the new method in the `api` object (after `deleteWorkItem`):

```ts
  updateWorkLogSummary: (date: string, summary: string) =>
    client.patch<{ ok: boolean }>(`/work-logs/${date}/summary`, { summary }),
```

- [ ] **Step 3: Verify type check**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/api/client.ts
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add updateWorkLogSummary API method"
```

---

### Task 12: Store addWorkItemsBatch + updateSummary

**Files:**
- Modify: `frontend/src/stores/workLog.ts`
- Test: `frontend/src/stores/workLog.spec.ts`

- [ ] **Step 1: Write failing tests**

In `frontend/src/stores/workLog.spec.ts`, first extend the mock at top:

```ts
vi.mock('@/api/client', () => ({
  api: {
    listWorkLogs: vi.fn(),
    getWorkLog: vi.fn(),
    getTodayContext: vi.fn(),
    structureBrainDump: vi.fn(),
    createWorkLog: vi.fn(),
    updateWorkLog: vi.fn(),
    generateWorkReport: vi.fn(),
    listWorkReports: vi.fn(),
    getWorkReport: vi.fn(),
    appendWorkItem: vi.fn(),
    updateWorkItem: vi.fn(),
    deleteWorkItem: vi.fn(),
    updateWorkLogSummary: vi.fn(), // 新增
  },
}))

vi.mock('element-plus', () => ({
  ElMessage: { error: vi.fn(), success: vi.fn(), warning: vi.fn() }, // 加 warning
}))
```

Add to `beforeEach`:

```ts
mockApi.updateWorkLogSummary.mockResolvedValue({ data: { ok: true } })
```

Append new test cases:

```ts
describe('addWorkItemsBatch', () => {
  it('all success returns successCount=N, failureIndices=[]', async () => {
    const store = useWorkLogStore()
    mockApi.appendWorkItem.mockResolvedValue({ data: {} })
    mockApi.getWorkLog.mockResolvedValue({ data: { id: 'wl-1', date: '2026-08-03', items: [] } })

    const items = [
      { activity: 'a', start_time: '09:00', end_time: '10:00', quadrant: 1 as const },
      { activity: 'b', start_time: '10:00', end_time: '11:00', quadrant: 2 as const },
    ]
    const result = await store.addWorkItemsBatch('2026-08-03', items)
    expect(result.successCount).toBe(2)
    expect(result.failureIndices).toEqual([])
    expect(mockApi.appendWorkItem).toHaveBeenCalledTimes(2)
  })

  it('mid-failure returns failureIndices=[i], partial success count', async () => {
    const store = useWorkLogStore()
    mockApi.appendWorkItem
      .mockResolvedValueOnce({ data: {} })
      .mockRejectedValueOnce(new Error('server error'))
    mockApi.getWorkLog.mockResolvedValue({ data: { id: 'wl-1', date: '2026-08-03', items: [] } })

    const items = [
      { activity: 'a', start_time: '09:00', end_time: '10:00', quadrant: 1 as const },
      { activity: 'b', start_time: '10:00', end_time: '11:00', quadrant: 2 as const },
    ]
    const result = await store.addWorkItemsBatch('2026-08-03', items)
    expect(result.successCount).toBe(1)
    expect(result.failureIndices).toEqual([1])
  })
})

describe('updateSummary', () => {
  it('success does not show error toast', async () => {
    const store = useWorkLogStore()
    mockApi.updateWorkLogSummary.mockResolvedValue({ data: { ok: true } })
    await store.updateSummary('2026-08-03', '今日小结')
    expect(mockApi.updateWorkLogSummary).toHaveBeenCalledWith('2026-08-03', '今日小结')
    expect(mockEl.warning).not.toHaveBeenCalled()
  })

  it('failure shows warning toast but does not throw', async () => {
    const store = useWorkLogStore()
    mockApi.updateWorkLogSummary.mockRejectedValue(new Error('fail'))
    await expect(store.updateSummary('2026-08-03', 'x')).resolves.toBeUndefined()
    expect(mockEl.warning).toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd frontend && npx vitest run src/stores/workLog.spec.ts
```

Expected: tests fail with "addWorkItemsBatch is not a function" / "updateSummary is not a function".

- [ ] **Step 3: Implement store actions**

In `frontend/src/stores/workLog.ts`, after `deleteQuickEntry` function (around line 134), append two new actions:

```ts
  async function addWorkItemsBatch(
    date: string,
    items: CreateQuickEntryInput[],
  ): Promise<{ successCount: number; failureIndices: number[] }> {
    const failureIndices: number[] = []
    let successCount = 0
    for (let i = 0; i < items.length; i++) {
      try {
        await api.appendWorkItem(date, items[i])
        successCount++
      } catch {
        failureIndices.push(i)
      }
    }
    if (successCount > 0) {
      await fetchLog(date)
    }
    return { successCount, failureIndices }
  }

  async function updateSummary(date: string, summary: string): Promise<void> {
    try {
      await api.updateWorkLogSummary(date, summary)
    } catch (e: any) {
      ElMessage.warning('保存今日小结失败：' + (e?.response?.data?.error || e?.message || ''))
    }
  }
```

Also add them to the return statement at end of the store (around line 178-184):

```ts
  return {
    logs, currentLog, todayContext, reports, currentReport, selected, loading,
    todayManualItems,
    fetchInitialRange, fetchLog, fetchTodayContext, structureBrainDump,
    saveWorkLog, generateReport, fetchReports, fetchReport, selectNode,
    addQuickEntry, updateQuickEntry, deleteQuickEntry,
    addWorkItemsBatch, updateSummary, // 新增
  }
```

- [ ] **Step 4: Run store tests**

```bash
cd frontend && npx vitest run src/stores/workLog.spec.ts
```

Expected: ALL PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/stores/workLog.ts frontend/src/stores/workLog.spec.ts
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add addWorkItemsBatch + updateSummary store actions"
```

---

### Task 13: New WorkItemForm.vue component

**Files:**
- Create: `frontend/src/components/work-log/WorkItemForm.vue`
- Test: `frontend/src/components/work-log/WorkItemForm.spec.ts`

- [ ] **Step 1: Write failing test**

Create `frontend/src/components/work-log/WorkItemForm.spec.ts`:

```ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import WorkItemForm from './WorkItemForm.vue'

vi.mock('@/api/client', () => ({
  api: {
    appendWorkItem: vi.fn().mockResolvedValue({ data: {} }),
    updateWorkItem: vi.fn().mockResolvedValue({ data: { ok: true } }),
    getWorkLog: vi.fn().mockResolvedValue({ data: { id: 'l1', date: '2026-08-03', items: [] } }),
  },
}))

vi.spyOn(ElMessage, 'success').mockImplementation(() => ({}) as any)
vi.spyOn(ElMessage, 'error').mockImplementation(() => ({}) as any)

const mountOpts = { global: { plugins: [ElementPlus] } }

describe('WorkItemForm', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders required + optional fields in add mode', () => {
    const wrapper = mount(WorkItemForm, {
      props: { date: '2026-08-03', mode: 'add' },
      ...mountOpts,
    })
    expect(wrapper.find('[data-test="activity-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="start-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="end-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="quadrant-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="content-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="problem-solved-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="result-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="impact-input"]').exists()).toBe(true)
    expect(wrapper.find('[data-test="submit-btn"]').text()).toContain('添加')
  })

  it('rejects submit when activity is empty', async () => {
    const wrapper = mount(WorkItemForm, {
      props: { date: '2026-08-03', mode: 'add' },
      ...mountOpts,
    })
    await wrapper.find('[data-test="submit-btn"]').trigger('click')
    expect(ElMessage.error).toHaveBeenCalled()
    expect(wrapper.emitted('added')).toBeFalsy()
  })

  it('submits with all fields and emits added on success', async () => {
    const wrapper = mount(WorkItemForm, {
      props: { date: '2026-08-03', mode: 'add' },
      ...mountOpts,
    })
    await wrapper.find('[data-test="activity-input"]').setValue('晨会')
    await wrapper.find('[data-test="content-input"]').setValue('内容')
    await wrapper.find('[data-test="submit-btn"]').trigger('click')
    // flush
    await Promise.resolve()
    await Promise.resolve()
    expect(wrapper.emitted('added')).toBeTruthy()
  })

  it('edit mode: prefills all fields from initial prop and shows cancel button', async () => {
    const wrapper = mount(WorkItemForm, {
      props: {
        date: '2026-08-03',
        mode: 'edit',
        itemId: 'item-1',
        initial: {
          activity: '原活动',
          start_time: '09:00',
          end_time: '10:00',
          quadrant: 2,
          content: '原内容',
          problem_solved: '原问题',
          result: '原结果',
          impact: '原影响',
        },
      },
      ...mountOpts,
    })
    expect((wrapper.find('[data-test="activity-input"]').element as HTMLInputElement).value).toBe('原活动')
    expect((wrapper.find('[data-test="content-input"]').element as HTMLTextAreaElement).value).toBe('原内容')
    expect(wrapper.find('[data-test="cancel-btn"]').exists()).toBe(true)
  })

  it('edit mode: cancel emits cancel event', async () => {
    const wrapper = mount(WorkItemForm, {
      props: { date: '2026-08-03', mode: 'edit', itemId: 'x', initial: {} },
      ...mountOpts,
    })
    await wrapper.find('[data-test="cancel-btn"]').trigger('click')
    expect(wrapper.emitted('cancel')).toBeTruthy()
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd frontend && npx vitest run src/components/work-log/WorkItemForm.spec.ts
```

Expected: fails (component does not exist).

- [ ] **Step 3: Implement WorkItemForm.vue**

Create `frontend/src/components/work-log/WorkItemForm.vue`:

```vue
<template>
  <div class="work-item-form">
    <el-form :model="form" @submit.prevent="onSubmit">
      <div class="wif-required">
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
            style="width: 220px"
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
            :clearable="false"
            style="width: 100px"
          />
          <span style="margin: 0 4px">-</span>
          <el-time-select
            data-test="end-input"
            v-model="form.end_time"
            :min-time="form.start_time"
            placeholder="结束"
            start="00:30"
            step="00:30"
            end="24:00"
            :clearable="false"
            style="width: 100px"
          />
        </el-form-item>

        <el-form-item label="象限">
          <el-radio-group v-model="form.quadrant" data-test="quadrant-input">
            <el-radio-button
              v-for="q in [1, 2, 3, 4] as Quadrant[]"
              :key="q"
              :label="q"
            >
              Q{{ q }}
            </el-radio-button>
          </el-radio-group>
        </el-form-item>
      </div>

      <div class="wif-optional">
        <div class="wif-optional-title">补充详情（可选）</div>
        <div class="wif-optional-grid">
          <div class="wif-field">
            <span class="wif-label">内容</span>
            <el-input
              data-test="content-input"
              v-model="form.content"
              type="textarea"
              :rows="2"
              placeholder="一句话描述"
            />
          </div>
          <div class="wif-field">
            <span class="wif-label">解决了什么问题</span>
            <el-input
              data-test="problem-solved-input"
              v-model="form.problem_solved"
              type="textarea"
              :rows="2"
            />
          </div>
          <div class="wif-field">
            <span class="wif-label">已产生的结果</span>
            <el-input
              data-test="result-input"
              v-model="form.result"
              type="textarea"
              :rows="2"
            />
          </div>
          <div class="wif-field">
            <span class="wif-label">对后续的影响</span>
            <el-input
              data-test="impact-input"
              v-model="form.impact"
              type="textarea"
              :rows="2"
            />
          </div>
        </div>
      </div>

      <div class="wif-actions">
        <el-button
          v-if="mode === 'edit'"
          data-test="cancel-btn"
          @click="$emit('cancel')"
        >
          取消
        </el-button>
        <el-button
          data-test="submit-btn"
          type="primary"
          :loading="saving"
          @click="onSubmit"
        >
          {{ mode === 'edit' ? '保存' : '添加' }}
        </el-button>
      </div>
    </el-form>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { useWorkLogStore } from '@/stores/workLog'
import type { Quadrant } from '@/types'

interface InitialData {
  activity?: string
  start_time?: string
  end_time?: string
  quadrant?: Quadrant
  content?: string
  problem_solved?: string
  result?: string
  impact?: string
}

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
  quadrant: props.initial.quadrant ?? (2 as Quadrant),
  content: props.initial.content ?? '',
  problem_solved: props.initial.problem_solved ?? '',
  result: props.initial.result ?? '',
  impact: props.initial.impact ?? '',
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
      const ok = await store.updateQuickEntry(form.date, props.itemId, {
        activity: form.activity,
        start_time: form.start_time,
        end_time: form.end_time,
        quadrant: form.quadrant,
        content: form.content,
        problem_solved: form.problem_solved,
        result: form.result,
        impact: form.impact,
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
        content: form.content,
        problem_solved: form.problem_solved,
        result: form.result,
        impact: form.impact,
      })
      if (ok) {
        ElMessage.success('已添加')
        form.activity = ''
        form.content = ''
        form.problem_solved = ''
        form.result = ''
        form.impact = ''
        emit('added')
      }
    }
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.work-item-form {
  background: var(--bg-card, #FFFEFC);
  border: 1px solid var(--border-color, #e5e5e5);
  border-radius: var(--radius-md);
  padding: 16px 20px;
  margin-bottom: 16px;
}
.wif-required {
  display: grid;
  grid-template-columns: 140px 220px 1fr 220px;
  gap: 12px;
  align-items: end;
  padding-bottom: 14px;
  border-bottom: 1px solid var(--border-color);
  margin-bottom: 14px;
}
.wif-required :deep(.el-form-item) {
  margin-bottom: 0;
}
.wif-optional {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 12px 14px;
}
.wif-optional-title {
  font-size: 11px;
  color: var(--text-muted);
  letter-spacing: 0.5px;
  text-transform: uppercase;
  margin-bottom: 10px;
  font-weight: 500;
}
.wif-optional-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px 14px;
}
.wif-field {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.wif-label {
  font-size: 11px;
  color: var(--text-muted);
  font-weight: 500;
}
.wif-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 14px;
}
</style>
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd frontend && npx vitest run src/components/work-log/WorkItemForm.spec.ts
```

Expected: 5 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/work-log/WorkItemForm.vue frontend/src/components/work-log/WorkItemForm.spec.ts
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add WorkItemForm component (8 fields, add/edit)"
```

---

### Task 14: New BatchTableEditor.vue component

**Files:**
- Create: `frontend/src/components/work-log/BatchTableEditor.vue`
- Test: `frontend/src/components/work-log/BatchTableEditor.spec.ts`

- [ ] **Step 1: Write failing test**

Create `frontend/src/components/work-log/BatchTableEditor.spec.ts`:

```ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import { ElMessage } from 'element-plus'
import BatchTableEditor from './BatchTableEditor.vue'
import type { DraftWorkItem } from './BatchTableEditor.vue'

vi.mock('@/api/client', () => ({
  api: {
    appendWorkItem: vi.fn(),
    updateWorkLogSummary: vi.fn(),
    getWorkLog: vi.fn().mockResolvedValue({ data: { id: 'l1', date: '2026-08-03', items: [] } }),
  },
}))

vi.spyOn(ElMessage, 'success').mockImplementation(() => ({}) as any)
vi.spyOn(ElMessage, 'error').mockImplementation(() => ({}) as any)
vi.spyOn(ElMessage, 'warning').mockImplementation(() => ({}) as any)

const mountOpts = { global: { plugins: [ElementPlus] } }

const sampleDraft = (): DraftWorkItem[] => [
  {
    activity: '',
    start_time: '09:00',
    end_time: '10:00',
    quadrant: 2,
    content: '内容1',
    problem_solved: '问题1',
    result: '结果1',
    impact: '影响1',
  },
]

describe('BatchTableEditor', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('renders one row per draft item plus an add-row', () => {
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items: sampleDraft(),
        summary: '',
      },
      ...mountOpts,
    })
    // 1 draft row + 1 "+加一条" row
    expect(wrapper.findAll('[data-test="draft-row"]')).toHaveLength(1)
    expect(wrapper.find('[data-test="add-row"]').exists()).toBe(true)
  })

  it('clicking add-row emits update:items with new empty row', async () => {
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items: sampleDraft(),
        summary: '',
      },
      ...mountOpts,
    })
    await wrapper.find('[data-test="add-row"]').trigger('click')
    const emitted = wrapper.emitted('update:items')
    expect(emitted).toBeTruthy()
    expect(emitted![0][0]).toHaveLength(2)
  })

  it('delete row removes it from items', async () => {
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items: sampleDraft(),
        summary: '',
      },
      ...mountOpts,
    })
    await wrapper.find('[data-test="delete-btn"]').trigger('click')
    const emitted = wrapper.emitted('update:items')
    expect(emitted).toBeTruthy()
    expect(emitted![0][0]).toHaveLength(0)
  })

  it('batch save validates empty activity and shows error', async () => {
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items: sampleDraft(), // activity is empty
        summary: '',
      },
      ...mountOpts,
    })
    await wrapper.find('[data-test="save-btn"]').trigger('click')
    await Promise.resolve()
    expect(ElMessage.error).toHaveBeenCalled()
  })

  it('batch save with all valid triggers save emit', async () => {
    const items = sampleDraft()
    items[0].activity = '填好了'
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items,
        summary: '今日小结',
      },
      ...mountOpts,
    })
    const { api } = await import('@/api/client')
    ;(api.appendWorkItem as any).mockResolvedValue({ data: {} })
    ;(api.updateWorkLogSummary as any).mockResolvedValue({ data: { ok: true } })

    await wrapper.find('[data-test="save-btn"]').trigger('click')
    await Promise.resolve()
    await Promise.resolve()
    await Promise.resolve()

    expect(api.appendWorkItem).toHaveBeenCalledTimes(1)
    expect(wrapper.emitted('save')).toBeTruthy()
  })

  it('discard button emits discard event', async () => {
    const wrapper = mount(BatchTableEditor, {
      props: {
        date: '2026-08-03',
        items: sampleDraft(),
        summary: '',
      },
      ...mountOpts,
    })
    await wrapper.find('[data-test="discard-btn"]').trigger('click')
    expect(wrapper.emitted('discard')).toBeTruthy()
  })
})
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd frontend && npx vitest run src/components/work-log/BatchTableEditor.spec.ts
```

Expected: fails (component does not exist).

- [ ] **Step 3: Implement BatchTableEditor.vue**

Create `frontend/src/components/work-log/BatchTableEditor.vue`:

```vue
<template>
  <div class="batch-table-editor">
    <div class="bte-header">
      <h3 class="bte-title">AI 草稿 · 待补齐 ({{ items.length }} 条)</h3>
    </div>

    <table class="bte-table">
      <thead>
        <tr>
          <th style="width: 100px">日期</th>
          <th style="width: 130px">时段</th>
          <th>活动 *</th>
          <th style="width: 110px">象限</th>
          <th>内容</th>
          <th>解决了什么问题</th>
          <th>已产生的结果</th>
          <th>对后续的影响</th>
          <th style="width: 30px"></th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="(item, idx) in items"
          :key="idx"
          class="bte-row"
          :class="{ 'bte-row-error': errorIndices.includes(idx) }"
          data-test="draft-row"
        >
          <td><input class="bte-cell" v-model="item.date" :value="item.date || date" @input="onCellInput(idx, 'date', $event)" /></td>
          <td>
            <div class="bte-time-pair">
              <input class="bte-cell" :value="item.start_time" @input="onCellInput(idx, 'start_time', $event)" placeholder="09:00" />
              <span>-</span>
              <input class="bte-cell" :value="item.end_time" @input="onCellInput(idx, 'end_time', $event)" placeholder="10:00" />
            </div>
          </td>
          <td><input class="bte-cell" :value="item.activity" @input="onCellInput(idx, 'activity', $event)" placeholder="做了什么" /></td>
          <td>
            <div class="bte-quadrant">
              <button
                v-for="q in [1, 2, 3, 4]"
                :key="q"
                type="button"
                class="bte-quad-btn"
                :class="{ active: item.quadrant === q }"
                @click="onQuadrantClick(idx, q)"
              >Q{{ q }}</button>
            </div>
          </td>
          <td><input class="bte-cell" :value="item.content" @input="onCellInput(idx, 'content', $event)" /></td>
          <td><input class="bte-cell" :value="item.problem_solved" @input="onCellInput(idx, 'problem_solved', $event)" /></td>
          <td><input class="bte-cell" :value="item.result" @input="onCellInput(idx, 'result', $event)" /></td>
          <td><input class="bte-cell" :value="item.impact" @input="onCellInput(idx, 'impact', $event)" /></td>
          <td><button class="bte-delete" data-test="delete-btn" @click="onDelete(idx)">×</button></td>
        </tr>
        <tr>
          <td colspan="9" class="bte-add-cell">
            <button class="bte-add" data-test="add-row" @click="onAdd">+ 加一条</button>
          </td>
        </tr>
      </tbody>
    </table>

    <div class="bte-summary">
      <label class="bte-label">今日小结</label>
      <textarea
        class="bte-summary-input"
        :value="summary"
        @input="$emit('update:summary', ($event.target as HTMLTextAreaElement).value)"
      />
    </div>

    <div class="bte-actions">
      <button class="bte-btn bte-btn-secondary" data-test="discard-btn" @click="$emit('discard')">放弃草稿</button>
      <button class="bte-btn bte-btn-primary" data-test="save-btn" :disabled="saving" @click="onSave">
        {{ saving ? '保存中…' : '批量入库' }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useWorkLogStore } from '@/stores/workLog'
import type { Quadrant } from '@/types'

export interface DraftWorkItem {
  activity: string
  start_time: string
  end_time: string
  quadrant: Quadrant
  content: string
  problem_solved: string
  result: string
  impact: string
  date?: string
}

const props = defineProps<{
  date: string
  items: DraftWorkItem[]
  summary: string
}>()

const emit = defineEmits<{
  'update:items': [items: DraftWorkItem[]]
  'update:summary': [summary: string]
  save: []
  discard: []
}>()

const store = useWorkLogStore()
const saving = ref(false)
const errorIndices = ref<number[]>([])

function clone(): DraftWorkItem[] {
  return props.items.map(it => ({ ...it }))
}

function onCellInput(idx: number, key: keyof DraftWorkItem, ev: Event) {
  const next = clone()
  ;(next[idx] as any)[key] = (ev.target as HTMLInputElement).value
  emit('update:items', next)
}

function onQuadrantClick(idx: number, q: Quadrant) {
  const next = clone()
  next[idx].quadrant = q
  emit('update:items', next)
}

function onAdd() {
  emit('update:items', [
    ...props.items,
    {
      activity: '',
      start_time: '09:00',
      end_time: '10:00',
      quadrant: 2 as Quadrant,
      content: '',
      problem_solved: '',
      result: '',
      impact: '',
    },
  ])
}

function onDelete(idx: number) {
  emit('update:items', props.items.filter((_, i) => i !== idx))
}

function validate(): number[] {
  const bad: number[] = []
  props.items.forEach((it, idx) => {
    if (!it.activity.trim()) bad.push(idx)
    else if (!it.start_time || !it.end_time) bad.push(idx)
    else if (it.start_time >= it.end_time) bad.push(idx)
  })
  return bad
}

async function onSave() {
  const bad = validate()
  if (bad.length > 0) {
    errorIndices.value = bad
    ElMessage.error(`第 ${bad.map(i => i + 1).join(', ')} 行必填字段缺失或时段无效`)
    return
  }
  errorIndices.value = []
  saving.value = true
  try {
    const payload = props.items.map(it => ({
      activity: it.activity,
      start_time: it.start_time,
      end_time: it.end_time,
      quadrant: it.quadrant,
      content: it.content,
      problem_solved: it.problem_solved,
      result: it.result,
      impact: it.impact,
    }))
    const { successCount, failureIndices } = await store.addWorkItemsBatch(props.date, payload)
    if (failureIndices.length > 0) {
      ElMessage.error(`${successCount}/${props.items.length} 条成功，失败 ${failureIndices.length} 条`)
      // 保留失败的，移除已成功的
      const remaining = failureIndices.map(i => props.items[i])
      emit('update:items', remaining)
      return
    }
    // 全部成功，再保存 summary
    if (props.summary.trim()) {
      await store.updateSummary(props.date, props.summary)
    }
    ElMessage.success(`已入库 ${successCount} 条`)
    emit('save')
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.batch-table-editor {
  background: var(--bg-card, #FFFEFC);
  border: 1px solid var(--accent-tertiary, #D98A75);
  border-radius: var(--radius-md);
  padding: 14px 18px;
  margin-bottom: 16px;
}
.bte-header {
  margin-bottom: 10px;
}
.bte-title {
  font-family: var(--font-display);
  font-size: 15px;
  font-weight: 600;
  margin: 0;
}
.bte-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 12px;
  background: var(--bg-elevated);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  overflow: hidden;
}
.bte-table th, .bte-table td {
  padding: 6px 8px;
  text-align: left;
  border-bottom: 1px solid var(--border-color);
}
.bte-table th {
  background: var(--bg-secondary);
  font-weight: 500;
  color: var(--text-muted);
  font-size: 10px;
  letter-spacing: 0.4px;
  text-transform: uppercase;
}
.bte-row {
  transition: background 0.15s;
}
.bte-row-error {
  background: rgba(184, 69, 44, 0.05);
}
.bte-cell {
  width: 100%;
  border: 1px solid transparent;
  background: transparent;
  font-family: var(--font-body);
  font-size: 12px;
  color: var(--text-primary);
  padding: 3px 5px;
  border-radius: 2px;
  outline: none;
}
.bte-cell:focus {
  background: var(--bg-elevated);
  border-color: var(--accent-primary);
}
.bte-time-pair {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--text-muted);
}
.bte-quadrant {
  display: flex;
  gap: 2px;
}
.bte-quad-btn {
  width: 22px;
  height: 18px;
  border: 1px solid var(--border-accent);
  background: var(--bg-elevated);
  font-size: 10px;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 0;
}
.bte-quad-btn.active {
  background: var(--accent-primary);
  color: white;
  border-color: var(--accent-primary);
}
.bte-delete {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 16px;
}
.bte-delete:hover {
  color: var(--accent-primary);
}
.bte-add-cell {
  text-align: center;
  padding: 6px;
}
.bte-add {
  background: transparent;
  border: none;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 12px;
  padding: 4px 12px;
}
.bte-add:hover {
  color: var(--accent-primary);
}
.bte-summary {
  margin-top: 12px;
  padding: 10px 12px;
  background: var(--bg-secondary);
  border-radius: var(--radius-sm);
}
.bte-label {
  display: block;
  font-size: 11px;
  color: var(--text-muted);
  margin-bottom: 6px;
}
.bte-summary-input {
  width: 100%;
  min-height: 50px;
  border: none;
  background: transparent;
  font-family: var(--font-body);
  font-size: 12px;
  line-height: 1.5;
  color: var(--text-primary);
  resize: vertical;
  outline: none;
}
.bte-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  margin-top: 12px;
}
.bte-btn {
  border: none;
  border-radius: var(--radius-sm);
  padding: 6px 16px;
  font-size: 12px;
  font-family: var(--font-body);
  cursor: pointer;
}
.bte-btn-primary {
  background: var(--accent-primary);
  color: white;
}
.bte-btn-primary:hover:not(:disabled) {
  background: var(--accent-secondary);
}
.bte-btn-primary:disabled {
  background: var(--text-muted);
  cursor: not-allowed;
}
.bte-btn-secondary {
  background: transparent;
  color: var(--text-secondary);
  border: 1px solid var(--border-accent);
}
.bte-btn-secondary:hover {
  border-color: var(--accent-primary);
  color: var(--accent-primary);
}
</style>
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd frontend && npx vitest run src/components/work-log/BatchTableEditor.spec.ts
```

Expected: 6 tests PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/work-log/BatchTableEditor.vue frontend/src/components/work-log/BatchTableEditor.spec.ts
git -c user.name="lsy" -c user.email="lsy@local" commit -m "feat(work-log): add BatchTableEditor component (Excel-style batch edit)"
```

---

### Task 15: Wire WorkLog.vue to new components

**Files:**
- Modify: `frontend/src/views/WorkLog.vue`

- [ ] **Step 1: Replace template**

In `frontend/src/views/WorkLog.vue`, replace the entire `<template>` section (lines 1-69). New version uses `WorkItemForm` + `BatchTableEditor`, removes `QuickEntryForm` / `WorkItemList` references:

```vue
<template>
  <div class="work-log-page">
    <div class="page-header">
      <h1 class="page-title">工作日志</h1>
      <div class="page-actions">
        <button class="action-btn" @click="goToday">今日</button>
        <ReportActions @generate="onGenerateReport" />
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
          <WorkItemForm
            v-if="!editingItemId"
            :date="currentDate"
            mode="add"
            @added="onQuickAdded"
          />
          <WorkItemForm
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

          <TodayContextCard :context="store.todayContext" />

          <BrainDumpInput
            :loading="structuring"
            @structure="onStructure"
          />

          <BatchTableEditor
            v-if="draftItems.length"
            :date="currentDate"
            :items="draftItems"
            :summary="draftSummary"
            @update:items="draftItems = $event"
            @update:summary="draftSummary = $event"
            @save="onBatchSaved"
            @discard="onBatchDiscard"
          />
        </template>

        <!-- 报告视图 -->
        <template v-else>
          <ReportDetail :report="store.currentReport" />
        </template>
      </div>
    </div>
  </div>
</template>
```

- [ ] **Step 2: Replace script imports and editingInitial**

In the `<script setup>` block, replace the import section (lines 72-83):

```ts
import { ref, computed, onMounted, watch } from 'vue'
import { useWorkLogStore } from '@/stores/workLog'
import Timeline from '@/components/work-log/Timeline.vue'
import TodayContextCard from '@/components/work-log/TodayContextCard.vue'
import BrainDumpInput from '@/components/work-log/BrainDumpInput.vue'
import ReportActions from '@/components/work-log/ReportActions.vue'
import ReportDetail from '@/components/work-log/ReportDetail.vue'
import WorkItemForm from '@/components/work-log/WorkItemForm.vue'
import BatchTableEditor from '@/components/work-log/BatchTableEditor.vue'
import type { DraftWorkItem } from '@/components/work-log/BatchTableEditor.vue'
import TodayPanorama from '@/components/work-log/TodayPanorama.vue'
import { ElMessageBox } from 'element-plus'
import type { StructuredWorkLog, SaveWorkLogInput, WorkReportType } from '@/types'
```

Replace the `DraftItem` interface declaration (lines 85-91) with a re-export usage:

```ts
// DraftItem 来自 BatchTableEditor，无需在此重定义
```

Replace the `editingInitial` computed (lines 106-116) to include optional fields:

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
    content: it.content ?? '',
    problem_solved: it.problem_solved ?? '',
    result: it.result ?? '',
    impact: it.impact ?? '',
  }
})
```

- [ ] **Step 3: Adjust onStructure to drop title from drafts**

In `onStructure` (around line 125-136), update to handle missing `title` field in AI output:

```ts
async function onStructure(text: string) {
  structuring.value = true
  try {
    const out: StructuredWorkLog | null = await store.structureBrainDump(text)
    if (out) {
      draftItems.value = out.items.map(it => ({
        activity: '',
        start_time: '09:00',
        end_time: '10:00',
        quadrant: 2,
        content: it.content ?? '',
        problem_solved: it.problem_solved ?? '',
        result: it.result ?? '',
        impact: it.impact ?? '',
      } as DraftWorkItem))
      draftSummary.value = out.summary
    }
  } finally {
    structuring.value = false
  }
}
```

- [ ] **Step 4: Add onBatchSaved + onBatchDiscard handlers**

Find the existing `onQuickAdded` (around line 170-172) and add new handlers right after:

```ts
function onQuickAdded() {
  // store 已经 fetchLog 过，panorama 通过 computed 自动刷新
}

function onBatchSaved() {
  draftItems.value = []
  draftSummary.value = ''
}

function onBatchDiscard() {
  draftItems.value = []
  draftSummary.value = ''
}
```

- [ ] **Step 5: Remove the obsolete watch on store.currentLog**

Delete the watch around lines 229-239 (the one that writes `currentLog.items` back to `draftItems`). New design: BatchTableEditor only holds AI drafts, not loaded items.

After deletion, this watch should be gone:

```ts
// 删除这整段 watch
watch(() => store.currentLog, (log) => {
  if (!log || log.date !== currentDate.value) return
  draftItems.value = log.items.map(it => ({...}))
  draftSummary.value = log.summary
})
```

- [ ] **Step 6: Remove now-unused save bar template + style**

In the `<template>`, the `<div class="save-bar">` block (lines 55-59 in old version) was removed in step 1, but if anything remains, remove it. In `<style>`, remove `.save-bar` and `.save-btn` rules (lines 296-317 in old version) since BatchTableEditor has its own actions.

- [ ] **Step 7: Type check**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: 0 errors.

- [ ] **Step 8: Run all frontend tests**

```bash
cd frontend && npx vitest run
```

Expected: ALL PASS. (Old QuickEntryForm.spec.ts will still pass since we haven't deleted it yet; new tests pass.)

- [ ] **Step 9: Commit**

```bash
git add frontend/src/views/WorkLog.vue
git -c user.name="lsy" -c user.email="lsy@local" commit -m "refactor(work-log): wire WorkLog view to WorkItemForm + BatchTableEditor"
```

---

### Task 16: Delete old components

**Files:**
- Delete: `frontend/src/components/work-log/QuickEntryForm.vue`
- Delete: `frontend/src/components/work-log/QuickEntryForm.spec.ts`
- Delete: `frontend/src/components/work-log/WorkItemList.vue`
- Delete: `frontend/src/components/work-log/WorkItemEditor.vue`

- [ ] **Step 1: Verify no remaining references**

```bash
cd frontend && grep -rn "QuickEntryForm\|WorkItemList\|WorkItemEditor" src/
```

Expected: no matches (only `WorkItemForm` / `BatchTableEditor` should remain).

If matches found, fix those references first.

- [ ] **Step 2: Delete files**

```bash
rm frontend/src/components/work-log/QuickEntryForm.vue
rm frontend/src/components/work-log/QuickEntryForm.spec.ts
rm frontend/src/components/work-log/WorkItemList.vue
rm frontend/src/components/work-log/WorkItemEditor.vue
```

- [ ] **Step 3: Now fully remove title from StructuredItem**

In `frontend/src/types/index.ts`, find `StructuredItem` and remove the deprecated `title?: string` field:

```ts
export interface StructuredItem {
  content: string
  problem_solved: string
  result: string
  impact: string
}
```

- [ ] **Step 4: Type check and tests**

```bash
cd frontend && npx vue-tsc --noEmit
cd frontend && npx vitest run
```

Expected: 0 type errors; all tests pass.

- [ ] **Step 5: Commit**

```bash
git add -A frontend/src/components/work-log/ frontend/src/types/index.ts
git -c user.name="lsy" -c user.email="lsy@local" commit -m "refactor(work-log): remove deprecated QuickEntryForm/WorkItemList/WorkItemEditor"
```

---

## Phase C — Verification

### Task 17: Full type check + tests + manual smoke

**Files:** None (verification only)

- [ ] **Step 1: Backend full tests**

```bash
cd backend && go test ./...
```

Expected: ALL PASS, no panics.

- [ ] **Step 2: Frontend type check**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: 0 errors.

- [ ] **Step 3: Frontend full tests**

```bash
cd frontend && npx vitest run
```

Expected: ALL PASS.

- [ ] **Step 4: Frontend build check**

```bash
cd frontend && npm run build
```

Expected: success (vue-tsc + vite build both succeed).

- [ ] **Step 5: Manual smoke test (E2E)**

Start backend (note: Windows needs CGO_ENABLED=1):

```bash
# Per LRN-001 / project memory: use scripts/start.sh on Windows
bash scripts/start.sh dev
```

In a browser, open `http://localhost:5173/work-log`:

1. **Main form add**: Fill 日期=今天 / 活动=测试 / 时段=09:00-10:00 / 象限=Q2 / 内容=测试内容 → 点 添加 → 验证：
   - ElMessage "已添加"
   - TodayPanorama 出现新行（时段 09:00-10:00 / 活动 测试 / 象限 Q2 / 操作 编辑 删除）
   - 主表单 activity + 4 可选字段清空，date/time/quadrant 保留

2. **Main form edit**: 点 TodayPanorama 的 编辑 → 主表单切到编辑态，8 字段全部回填 → 改活动名 + 改内容 → 点 保存 → 验证：
   - ElMessage "已更新"
   - TodayPanorama 行内容更新
   - 主表单回到添加态

3. **AI brain dump batch entry**: 在 BrainDumpInput 输入 "今天做了 X，解决了 Y 问题，产出 Z" → 点 AI 拆条 → 验证 BatchTableEditor 出现，至少 1 行，4 个可选字段已预填，必填字段空 → 用户补 activity + 时段 → 点 批量入库 → 验证：
   - ElMessage "已入库 N 条"
   - BatchTableEditor 消失
   - TodayPanorama 出现新行

4. **Validation**: BatchTableEditor 中清空 activity → 点 批量入库 → 验证 ElMessage.error，不发请求。

5. **TodayPanorama delete**: 点某行 删除 → 弹确认 → 确认 → 行消失。

- [ ] **Step 6: Commit any smoke-test fixes**

If smoke test surfaces bugs, fix and commit them individually with descriptive messages. If no fixes needed, skip.

- [ ] **Step 7: Final summary commit (optional)**

If the implementation is split across many commits and a final wrap-up commit is desired for branch tracking:

```bash
git -c user.name="lsy" -c user.email="lsy@local" commit --allow-empty -m "chore(work-log): form merge complete — verified"
```

(Most cases skip this — the per-task commits already tell the story.)

---

## Self-Review

**Spec coverage check:**

- §3 Architecture (component tree): Task 13 (WorkItemForm) + Task 14 (BatchTableEditor) + Task 15 (wire) + Task 16 (delete old) ✓
- §4.1 Backend model unchanged: no task needed ✓
- §4.2 Backend DTO extension + title sync: Task 5 + Task 6 ✓
- §4.2.1 New PATCH /summary endpoint: Task 3 (repo) + Task 4 (service) + Task 8 (handler) + Task 9 (router) ✓
- §4.3 Frontend types: Task 10 ✓
- §4.4 Frontend store + API client: Task 11 (api) + Task 12 (store) ✓
- §4.5 Backend handler changes: Task 8 ✓
- §5 Three data flow paths: Task 13 (paths 1+2) + Task 14 (path 3) ✓
- §6 Summary placement: Task 4 + Task 8 + Task 12 (updateSummary wiring) + Task 14 (BatchTableEditor footer) ✓
- §7 AI prompt drop title: Task 7 ✓
- §8 SQLite migration: Task 1 + Task 2 ✓
- §9 Error handling: covered in Task 13 (form validation) + Task 14 (batch validation + partial failure) + Task 12 (summary warning) ✓
- §10 Testing strategy: every task is TDD with tests ✓
- §11 Out of scope: not implemented, ✓

**Type/Method consistency check:**

- `MigrateWorkItemsTitleBackfill(db *gorm.DB) error` — Task 1 def → Task 2 use ✓
- `UpdateWorkLogSummary(date, summary string) error` (repo) — Task 3 def → Task 4 use ✓
- `UpdateSummary(date, summary string) error` (service) — Task 4 def → Task 8 use ✓
- `addWorkItemsBatch` / `updateSummary` (store) — Task 12 def → Task 14 use ✓
- `DraftWorkItem` (frontend) — Task 14 def → Task 15 import ✓
- `updateWorkLogSummary(date, summary)` (api) — Task 11 def → Task 12 use ✓

No contradictions found.
