# 数据导入导出 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 为 TickTask 增加全量数据 JSON 导入导出:一键导出全部业务数据;导入时按模块检测冲突,设置逐字段 diff、集合类按模块策略 + 逐条 override,单事务落库。

**Architecture:** 两阶段导入(只读预览 → 单事务应用)。后端新增 `model/backup.go`(DTO)+ `repository/data_repo.go`(整表读 + 事务写)+ `service/data_service.go`(diff/策略逻辑)+ `handler/data.go`(3 端点);前端在 Settings 加「数据管理」卡 + `ImportWizard` 向导。冲突判定按主键;WorkLog 含 items 整体原子解决;DailyStats 不导出。

**Tech Stack:** Go 1.21 / Gin / GORM / SQLite · Vue 3.5 `<script setup lang="ts">` / Pinia / Element Plus / axios / Vitest

**Spec:** `docs/superpowers/specs/2026-08-07-config-import-export-design.md`

---

## File Structure

**Backend (new):**
- `backend/internal/model/backup.go` — 全部 DTO:信封、预览、应用请求/计划/结果、策略常量
- `backend/internal/repository/data_repo.go` — `BackupRepository` 接口 + 实现(`ReadAll` / `Apply`)
- `backend/internal/repository/data_repo_test.go` — 真实 `:memory:` SQLite,覆盖 ReadAll + Apply + 事务原子性
- `backend/internal/service/data_service.go` — `DataService` 接口 + 实现(`Export` / `PreviewImport` / `ApplyImport`)
- `backend/internal/service/data_service_test.go` — 手写 mock repo,表驱动覆盖导出/预览/应用逻辑
- `backend/internal/api/handler/data.go` — `DataHandler`(Export / PreviewImport / ApplyImport)
- `backend/internal/api/handler/data_test.go` — mock service,覆盖 3 端点 + 错误路径

**Backend (modify):**
- `backend/internal/api/router.go` — 新增 `/data` group + `SetupRouter` 增加 `dataService` 参数
- `backend/cmd/server/main.go` — 注入 `dataRepo` + `dataService`,传入 `SetupRouter`

**Frontend (new):**
- `frontend/src/components/settings/ImportWizard.vue` — 三步导入向导
- `frontend/src/components/settings/ImportWizard.test.ts` — 步骤流转 / 渲染 / 策略 / 掩码 / 确认 / 成败

**Frontend (modify):**
- `frontend/src/types/index.ts` — 追加备份/导入类型
- `frontend/src/api/client.ts` — `exportData` / `previewImport` / `applyImport`
- `frontend/src/views/Settings.vue` — 新增「数据管理」卡片 + 接入向导

**命名约定(全文档统一,后续 task 必须逐字一致):**
- 策略:`add_new_only` / `merge_file` / `merge_current` / `replace`
- 选择:`file` / `current`
- 模块 key:`tasks` / `sessions` / `schedules` / `work_logs` / `work_reports`(settings 走 `data.settings`,无 policy)
- 模块 key(前端):同上 + `settings`

---

## Task 1: model DTOs (`model/backup.go`)

**Files:**
- Create: `backend/internal/model/backup.go`
- Test: `backend/internal/model/backup_test.go`

- [ ] **Step 1: 写失败测试 — JSON 往返 + 默认值**

Create `backend/internal/model/backup_test.go`:

```go
package model

import (
	"encoding/json"
	"testing"
)

func TestBackupEnvelope_RoundTrip(t *testing.T) {
	env := BackupEnvelope{
		App:           "ticktask",
		SchemaVersion: BackupSchemaVersion,
		Data: BackupData{
			Tasks:    []Task{{ID: "t1", Title: "x"}},
			Settings: SettingsBundle{Pomodoro: DefaultPomodoroSettings(), AI: DefaultAISettings()},
			WorkLogs: []WorkLog{{ID: "wl1", Date: "2026-08-07", Items: []WorkItem{{ID: "wi1", WorkLogID: "wl1"}}}},
		},
	}
	raw, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got BackupEnvelope
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.App != "ticktask" || got.SchemaVersion != 1 {
		t.Errorf("envelope meta wrong: %+v", got)
	}
	if len(got.Data.Tasks) != 1 || got.Data.Tasks[0].ID != "t1" {
		t.Errorf("tasks lost: %+v", got.Data.Tasks)
	}
	if got.Data.Settings.Pomodoro == nil || got.Data.WorkLogs[0].Items[0].ID != "wi1" {
		t.Errorf("settings/items nesting wrong: %+v", got.Data)
	}
}

func TestApplyPlan_ZeroValue(t *testing.T) {
	var p ApplyPlan
	raw, _ := json.Marshal(p)
	if string(raw) == "" {
		t.Error("zero ApplyPlan should marshal")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/model/ -run TestBackupEnvelope_RoundTrip -v`
Expected: FAIL — `BackupEnvelope` / `BackupData` undefined.

- [ ] **Step 3: 实现 DTO**

Create `backend/internal/model/backup.go`:

```go
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
	New               int                   `json:"new"`
	Identical         int                   `json:"identical"`
	Conflict          int                   `json:"conflict"`
	Orphan            int                   `json:"orphan"`
	Conflicts         []RecordConflict      `json:"conflicts"`
	SettingsConflicts []SettingsFieldDiff   `json:"settings_conflicts"`
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
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend && go test ./internal/model/ -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/internal/model/backup.go backend/internal/model/backup_test.go
git commit -m "feat(data): add backup/import DTOs in model/backup.go"
```

---

## Task 2: BackupRepository.ReadAll

**Files:**
- Create: `backend/internal/repository/data_repo.go`
- Test: `backend/internal/repository/data_repo_test.go`

- [ ] **Step 1: 写失败测试 — 真实 SQLite 读全表 + settings 组装 + items 嵌套 + 不含 DailyStats**

Create `backend/internal/repository/data_repo_test.go`:

```go
package repository

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"ticktask/internal/model"
)

func newDataTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(
		&model.Task{}, &model.PomodoroSession{}, &model.Schedule{},
		&model.Setting{}, &model.DailyStats{},
		&model.WorkLog{}, &model.WorkItem{}, &model.WorkReport{},
	); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func TestBackupRepo_ReadAll_Empty(t *testing.T) {
	repo := NewDataRepository(newDataTestDB(t))
	data, err := repo.ReadAll()
	if err != nil {
		t.Fatalf("readall: %v", err)
	}
	if len(data.Tasks) != 0 || len(data.WorkLogs) != 0 {
		t.Errorf("empty db should yield empty slices: %+v", data)
	}
	if data.Settings.Pomodoro == nil || data.Settings.AI == nil {
		t.Error("empty db should still yield default settings bundle")
	}
	if data.Settings.Pomodoro.WorkDuration != 1500 {
		t.Errorf("default work_duration = %d, want 1500", data.Settings.Pomodoro.WorkDuration)
	}
}

func TestBackupRepo_ReadAll_AssembledAndNested(t *testing.T) {
	db := newDataTestDB(t)
	db.Create(&model.Task{ID: "t1", Title: "task", Status: model.StatusTodo})
	db.Create(&model.PomodoroSession{ID: "s1", TaskID: strPtr("t1")})
	db.Create(&model.WorkLog{ID: "wl1", Date: "2026-08-07"})
	db.Create(&model.WorkItem{ID: "wi1", WorkLogID: "wl1", Seq: 1, Source: "ai"})
	db.Create(&model.Setting{Key: "pomodoro.settings", Value: `{"work_duration":1800,"short_break_duration":300,"long_break_duration":900,"long_break_after":4,"auto_start_break":false,"auto_start_work":false,"enable_sound":true,"buffer_ratio":20,"task_time_preferences":"{\"management\":\"any\",\"dev\":\"any\"}"}`})
	db.Create(&model.Setting{Key: "ai.settings", Value: `{"provider":"anthropic","api_key":"k","base_url":"u","model":"m"}`})
	db.Create(&model.DailyStats{}) // 必须被排除

	repo := NewDataRepository(db)
	data, err := repo.ReadAll()
	if err != nil {
		t.Fatalf("readall: %v", err)
	}
	if len(data.Tasks) != 1 || data.Tasks[0].ID != "t1" {
		t.Errorf("tasks: %+v", data.Tasks)
	}
	if len(data.Sessions) != 1 || data.Sessions[0].ID != "s1" {
		t.Errorf("sessions: %+v", data.Sessions)
	}
	if len(data.WorkLogs) != 1 || len(data.WorkLogs[0].Items) != 1 {
		t.Errorf("work_logs items not nested: %+v", data.WorkLogs)
	}
	if data.Settings.Pomodoro.WorkDuration != 1800 {
		t.Errorf("pomodoro not assembled: %d", data.Settings.Pomodoro.WorkDuration)
	}
	if data.Settings.AI.Provider != "anthropic" {
		t.Errorf("ai not assembled: %+v", data.Settings.AI)
	}
}

func strPtr(s string) *string { return &s }
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/repository/ -run TestBackupRepo_ReadAll -v`
Expected: FAIL — `NewDataRepository` undefined.

- [ ] **Step 3: 实现 ReadAll + 接口骨架(Apply 留待 Task 3)**

Create `backend/internal/repository/data_repo.go`:

```go
package repository

import (
	"encoding/json"
	"ticktask/internal/model"

	"gorm.io/gorm"
)

// BackupRepository 横跨全表的整表读 + 单事务写。
type BackupRepository interface {
	ReadAll() (*model.BackupData, error)
	Apply(plan model.ApplyPlan) error
}

type dataRepository struct {
	db *gorm.DB
}

func NewDataRepository(db *gorm.DB) BackupRepository {
	return &dataRepository{db: db}
}

func (r *dataRepository) ReadAll() (*model.BackupData, error) {
	data := &model.BackupData{}
	if err := r.db.Find(&data.Tasks).Error; err != nil {
		return nil, err
	}
	if err := r.db.Find(&data.Sessions).Error; err != nil {
		return nil, err
	}
	if err := r.db.Find(&data.Schedules).Error; err != nil {
		return nil, err
	}
	if err := r.db.Preload("Items").Find(&data.WorkLogs).Error; err != nil {
		return nil, err
	}
	if err := r.db.Find(&data.WorkReports).Error; err != nil {
		return nil, err
	}

	pomodoro := model.DefaultPomodoroSettings()
	var pomoSetting model.Setting
	if err := r.db.Where("key = ?", "pomodoro.settings").First(&pomoSetting).Error; err == nil {
		_ = json.Unmarshal([]byte(pomoSetting.Value), pomodoro)
	}
	ai := model.DefaultAISettings()
	var aiSetting model.Setting
	if err := r.db.Where("key = ?", "ai.settings").First(&aiSetting).Error; err == nil {
		_ = json.Unmarshal([]byte(aiSetting.Value), ai)
	}
	data.Settings = model.SettingsBundle{Pomodoro: pomodoro, AI: ai}

	return data, nil
}

func (r *dataRepository) Apply(plan model.ApplyPlan) error {
	// Task 3 实现
	return nil
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend && go test ./internal/repository/ -run TestBackupRepo_ReadAll -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/internal/repository/data_repo.go backend/internal/repository/data_repo_test.go
git commit -m "feat(data): add BackupRepository.ReadAll (full-table read, settings assembled)"
```

---

## Task 3: BackupRepository.Apply (single transaction)

**Files:**
- Modify: `backend/internal/repository/data_repo.go`
- Test: `backend/internal/repository/data_repo_test.go`

- [ ] **Step 1: 写失败测试 — upsert 新增/更新、删 orphan、settings 落表、worklog+items 原子替换、事务回滚**

Append to `data_repo_test.go`:

```go
func TestBackupRepo_Apply_InsertUpdateDeleteOrphan(t *testing.T) {
	db := newDataTestDB(t)
	db.Create(&model.Task{ID: "exist", Title: "old", Status: model.StatusTodo})
	repo := NewDataRepository(db)

	plan := model.ApplyPlan{
		Tasks: []model.Task{
			{ID: "exist", Title: "new", Status: model.StatusTodo}, // 更新
			{ID: "fresh", Title: "brand", Status: model.StatusTodo}, // 新增
		},
		DeleteTasks: []string{"exist-orphan"}, // 不存在,无副作用
	}
	if err := repo.Apply(plan); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var got []model.Task
	db.Find(&got)
	byID := map[string]model.Task{}
	for _, x := range got {
		byID[x.ID] = x
	}
	if byID["exist"].Title != "new" {
		t.Errorf("exist should be updated to 'new', got %q", byID["exist"].Title)
	}
	if _, ok := byID["fresh"]; !ok {
		t.Error("fresh should be inserted")
	}
}

func TestBackupRepo_Apply_SettingsUpsert(t *testing.T) {
	db := newDataTestDB(t)
	repo := NewDataRepository(db)
	pomo := model.DefaultPomodoroSettings()
	pomo.WorkDuration = 999
	ai := model.DefaultAISettings()
	ai.Model = "mymodel"
	if err := repo.Apply(model.ApplyPlan{Settings: &model.SettingsBundle{Pomodoro: pomo, AI: ai}}); err != nil {
		t.Fatalf("apply: %v", err)
	}
	data, _ := repo.ReadAll()
	if data.Settings.Pomodoro.WorkDuration != 999 || data.Settings.AI.Model != "mymodel" {
		t.Errorf("settings not persisted: %+v", data.Settings)
	}
}

func TestBackupRepo_Apply_WorkLogReplacesItemsAtomically(t *testing.T) {
	db := newDataTestDB(t)
	db.Create(&model.WorkLog{ID: "wl1", Date: "2026-08-07"})
	db.Create(&model.WorkItem{ID: "old-item", WorkLogID: "wl1", Seq: 1, Source: "ai"})
	repo := NewDataRepository(db)

	plan := model.ApplyPlan{
		WorkLogs: []model.WorkLog{
			{ID: "wl1", Date: "2026-08-07", Summary: "upd", Items: []model.WorkItem{
				{ID: "new-a", WorkLogID: "wl1", Seq: 1, Source: "ai"},
				{ID: "new-b", WorkLogID: "wl1", Seq: 2, Source: "ai"},
			}},
		},
	}
	if err := repo.Apply(plan); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var items []model.WorkItem
	db.Where("work_log_id = ?", "wl1").Find(&items)
	if len(items) != 2 {
		t.Fatalf("expected 2 items after replace, got %d", len(items))
	}
	var log model.WorkLog
	db.Where("id = ?", "wl1").First(&log)
	if log.Summary != "upd" {
		t.Errorf("log scalar not updated: %q", log.Summary)
	}
}

func TestBackupRepo_Apply_TransactionRollback(t *testing.T) {
	db := newDataTestDB(t)
	db.Create(&model.WorkLog{ID: "wl1", Date: "2026-08-01", Summary: "keep"})
	repo := NewDataRepository(db)

	// wl1 正常更新;wl2 的 items 含重复 PK → Create 报错 → 整体回滚
	plan := model.ApplyPlan{
		WorkLogs: []model.WorkLog{
			{ID: "wl1", Date: "2026-08-01", Summary: "changed"},
			{ID: "wl2", Date: "2026-08-02", Items: []model.WorkItem{
				{ID: "dup", WorkLogID: "wl2", Seq: 1, Source: "ai"},
				{ID: "dup", WorkLogID: "wl2", Seq: 2, Source: "ai"},
			}},
		},
	}
	err := repo.Apply(plan)
	if err == nil {
		t.Fatal("expected error from duplicate PK, got nil")
	}

	var log model.WorkLog
	db.Where("id = ?", "wl1").First(&log)
	if log.Summary != "keep" {
		t.Errorf("wl1 should be unchanged after rollback, got %q", log.Summary)
	}
}

func TestBackupRepo_Apply_DeleteOrphans(t *testing.T) {
	db := newDataTestDB(t)
	db.Create(&model.Task{ID: "t-del", Title: "x", Status: model.StatusTodo})
	db.Create(&model.WorkLog{ID: "wl-del", Date: "2026-08-01"})
	db.Create(&model.WorkItem{ID: "wi-del", WorkLogID: "wl-del", Seq: 1, Source: "ai"})
	repo := NewDataRepository(db)

	if err := repo.Apply(model.ApplyPlan{
		DeleteTasks:    []string{"t-del"},
		DeleteWorkLogs: []string{"wl-del"},
	}); err != nil {
		t.Fatalf("apply: %v", err)
	}

	var n int64
	db.Model(&model.Task{}).Count(&n)
	if n != 0 {
		t.Errorf("task orphan should be deleted, count=%d", n)
	}
	db.Model(&model.WorkItem{}).Where("work_log_id = ?", "wl-del").Count(&n)
	if n != 0 {
		t.Errorf("worklog orphan's items should be deleted, count=%d", n)
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/repository/ -run TestBackupRepo_Apply -v`
Expected: FAIL — Apply 是空实现,断言不通过。

- [ ] **Step 3: 实现 Apply + 私有 helpers**

Replace the stub `Apply` in `data_repo.go` with:

```go
func (r *dataRepository) Apply(plan model.ApplyPlan) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := upsertSlice(tx, plan.Tasks); err != nil {
			return err
		}
		if err := upsertSlice(tx, plan.Sessions); err != nil {
			return err
		}
		if err := upsertSlice(tx, plan.Schedules); err != nil {
			return err
		}
		if err := upsertSlice(tx, plan.WorkReports); err != nil {
			return err
		}
		if err := applyWorkLogs(tx, plan.WorkLogs); err != nil {
			return err
		}

		if err := deleteByIDs(tx, &model.Task{}, plan.DeleteTasks); err != nil {
			return err
		}
		if err := deleteByIDs(tx, &model.PomodoroSession{}, plan.DeleteSessions); err != nil {
			return err
		}
		if err := deleteByIDs(tx, &model.Schedule{}, plan.DeleteSchedules); err != nil {
			return err
		}
		if err := deleteByIDs(tx, &model.WorkReport{}, plan.DeleteWorkReports); err != nil {
			return err
		}
		if err := deleteWorkLogOrphans(tx, plan.DeleteWorkLogs); err != nil {
			return err
		}

		if plan.Settings != nil {
			if err := writeSetting(tx, "pomodoro.settings", plan.Settings.Pomodoro); err != nil {
				return err
			}
			if err := writeSetting(tx, "ai.settings", plan.Settings.AI); err != nil {
				return err
			}
		}
		return nil
	})
}

// upsertSlice 用 Save 批量 upsert(按主键)。
func upsertSlice[T any](tx *gorm.DB, records []T) error {
	if len(records) == 0 {
		return nil
	}
	return tx.Save(&records).Error
}

func deleteByIDs(tx *gorm.DB, dest any, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	return tx.Where("id IN ?", ids).Delete(dest).Error
}

// applyWorkLogs 每条 log:Save 标量 → 删旧 items → 建新 items(原子)。
func applyWorkLogs(tx *gorm.DB, logs []model.WorkLog) error {
	for i := range logs {
		log := logs[i]
		if err := tx.Save(&log).Error; err != nil {
			return err
		}
		if err := tx.Where("work_log_id = ?", log.ID).Delete(&model.WorkItem{}).Error; err != nil {
			return err
		}
		if len(log.Items) > 0 {
			if err := tx.Create(&log.Items).Error; err != nil {
				return err
			}
		}
	}
	return nil
}

func deleteWorkLogOrphans(tx *gorm.DB, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := tx.Where("work_log_id IN ?", ids).Delete(&model.WorkItem{}).Error; err != nil {
		return err
	}
	return tx.Where("id IN ?", ids).Delete(&model.WorkLog{}).Error
}

func writeSetting(tx *gorm.DB, key string, value any) error {
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return tx.Save(&model.Setting{Key: key, Value: string(raw)}).Error
}
```

> 注:`upsertSlice` 用了 Go 泛型(Go 1.21+ 支持),把 5 个表的批量 Save 收敛为一处(DRY)。仓库其余代码风格保持与现有 repo 一致。

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend && go test ./internal/repository/ -v`
Expected: PASS(含回滚不变式 `TestBackupRepo_Apply_TransactionRollback`)。

- [ ] **Step 5: 提交**

```bash
git add backend/internal/repository/data_repo.go backend/internal/repository/data_repo_test.go
git commit -m "feat(data): add BackupRepository.Apply (transactional upsert/delete + settings)"
```

---

## Task 4: DataService.Export

**Files:**
- Create: `backend/internal/service/data_service.go`
- Test: `backend/internal/service/data_service_test.go`

- [ ] **Step 1: 写失败测试 — 导出信封元信息 + settings 组装 + items 嵌套 + 不含 DailyStats(经 mock repo)**

Create `backend/internal/service/data_service_test.go`:

```go
package service

import (
	"testing"
	"ticktask/internal/model"
)

// mockBackupRepo 实现 repository.BackupRepository,内存快照 + 捕获 Apply。
type mockBackupRepo struct {
	snapshot *model.BackupData
	lastPlan *model.ApplyPlan
	applyErr error
}

func (m *mockBackupRepo) ReadAll() (*model.BackupData, error) {
	return m.snapshot, nil
}
func (m *mockBackupRepo) Apply(plan model.ApplyPlan) error {
	m.lastPlan = &plan
	return m.applyErr
}

func newSnapshot() *model.BackupData {
	return &model.BackupData{
		Tasks:    []model.Task{{ID: "t1", Title: "x"}},
		Sessions: []model.PomodoroSession{{ID: "s1"}},
		Settings: model.SettingsBundle{Pomodoro: model.DefaultPomodoroSettings(), AI: model.DefaultAISettings()},
		WorkLogs: []model.WorkLog{{ID: "wl1", Date: "2026-08-07", Items: []model.WorkItem{{ID: "wi1", WorkLogID: "wl1"}}}},
	}
}

func TestDataService_Export(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	env, err := svc.Export()
	if err != nil {
		t.Fatalf("export: %v", err)
	}
	if env.App != "ticktask" || env.SchemaVersion != model.BackupSchemaVersion {
		t.Errorf("envelope meta wrong: %+v", env)
	}
	if env.ExportedAt.IsZero() {
		t.Error("exported_at should be set")
	}
	if len(env.Data.Tasks) != 1 || len(env.Data.WorkLogs) != 1 {
		t.Errorf("data lost: %+v", env.Data)
	}
	if env.Data.WorkLogs[0].Items[0].ID != "wi1" {
		t.Error("items not nested in export")
	}
	if env.Data.Settings.Pomodoro.WorkDuration != 1500 {
		t.Errorf("pomodoro default wrong: %d", env.Data.Settings.Pomodoro.WorkDuration)
	}
}

func TestDataService_Export_Empty(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: &model.BackupData{}})
	env, err := svc.Export()
	if err != nil {
		t.Fatalf("export empty: %v", err)
	}
	if len(env.Data.Tasks) != 0 {
		t.Error("empty snapshot should export empty arrays")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/service/ -run TestDataService_Export -v`
Expected: FAIL — `NewDataService` undefined。

- [ ] **Step 3: 实现 Export + 接口骨架(Preview/Apply 留后续 task)**

Create `backend/internal/service/data_service.go`:

```go
package service

import (
	"time"
	"ticktask/internal/model"
	"ticktask/internal/repository"
)

const backupApp = "ticktask"

type DataService interface {
	Export() (*model.BackupEnvelope, error)
	PreviewImport(file *model.BackupData) (*model.ImportPreview, error)
	ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error)
}

type dataService struct {
	repo repository.BackupRepository
}

func NewDataService(repo repository.BackupRepository) DataService {
	return &dataService{repo: repo}
}

func (s *dataService) Export() (*model.BackupEnvelope, error) {
	data, err := s.repo.ReadAll()
	if err != nil {
		return nil, err
	}
	return &model.BackupEnvelope{
		App:           backupApp,
		SchemaVersion: model.BackupSchemaVersion,
		ExportedAt:    time.Now().UTC(),
		Data:          *data,
	}, nil
}

func (s *dataService) PreviewImport(file *model.BackupData) (*model.ImportPreview, error) {
	return nil, nil // Task 5
}

func (s *dataService) ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error) {
	return nil, nil // Task 6
}
```

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend && go test ./internal/service/ -run TestDataService_Export -v`
Expected: PASS。

- [ ] **Step 5: 提交**

```bash
git add backend/internal/service/data_service.go backend/internal/service/data_service_test.go
git commit -m "feat(data): add DataService.Export"
```

---

## Task 5: DataService.PreviewImport (read-only diff)

**Files:**
- Modify: `backend/internal/service/data_service.go`
- Test: `backend/internal/service/data_service_test.go`

- [ ] **Step 1: 写失败测试 — 表驱动:全 new / 全 identical / 冲突含字段 diff / orphan / 混合 / settings 逐字段(含 api_key)/ schema 告警 / work_logs 嵌套冲突**

Append to `data_service_test.go`:

```go
func TestDataService_PreviewImport_Classification(t *testing.T) {
	cur := newSnapshot() // 1 task t1, 1 session s1, 1 worklog wl1(wi1)

	cases := []struct {
		name   string
		file   *model.BackupData
		module string
		newN, identN, confN, orphN int
	}{
		{"all new", &model.BackupData{Tasks: []model.Task{{ID: "t2"}}}, "tasks", 1, 0, 0, 1},
		{"all identical", &model.BackupData{Tasks: []model.Task{{ID: "t1", Title: "x"}}}, "tasks", 0, 1, 0, 0},
		{"conflict", &model.BackupData{Tasks: []model.Task{{ID: "t1", Title: "changed"}}}, "tasks", 0, 0, 1, 0},
		{"orphan only", &model.BackupData{Sessions: []model.PomodoroSession{{ID: "other"}}}, "sessions", 1, 0, 0, 1},
		{"empty file vs full current", &model.BackupData{}, "tasks", 0, 0, 0, 1},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			svc := NewDataService(&mockBackupRepo{snapshot: cur})
			prev, err := svc.PreviewImport(c.file)
			if err != nil {
				t.Fatalf("preview: %v", err)
			}
			m := prev.Modules[c.module]
			if m == nil {
				t.Fatalf("module %s missing", c.module)
			}
			if m.New != c.newN || m.Identical != c.identN || m.Conflict != c.confN || m.Orphan != c.orphN {
				t.Errorf("%s: got new=%d ident=%d conf=%d orphan=%d", c.name, m.New, m.Identical, m.Conflict, m.Orphan)
			}
		})
	}
}

func TestDataService_PreviewImport_ConflictFieldDiff(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	prev, _ := svc.PreviewImport(&model.BackupData{Tasks: []model.Task{{ID: "t1", Title: "changed", Status: model.StatusCompleted}}})
	m := prev.Modules["tasks"]
	if len(m.Conflicts) != 1 || m.Conflicts[0].ID != "t1" {
		t.Fatalf("conflict not recorded: %+v", m.Conflicts)
	}
	fields := map[string]bool{}
	for _, f := range m.Conflicts[0].Fields {
		fields[f.Field] = true
	}
	if !fields["title"] || !fields["status"] {
		t.Errorf("expected title+status diffs, got %+v", m.Conflicts[0].Fields)
	}
}

func TestDataService_PreviewImport_SettingsDiff(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	filePomo := model.DefaultPomodoroSettings()
	filePomo.WorkDuration = 1800
	fileAI := model.DefaultAISettings()
	fileAI.APIKey = "secret"
	file := &model.BackupData{Settings: model.SettingsBundle{Pomodoro: filePomo, AI: fileAI}}

	prev, _ := svc.PreviewImport(file)
	m := prev.Modules["settings"]
	if m == nil {
		t.Fatal("settings module missing")
	}
	gotFields := map[string]bool{}
	for _, f := range m.SettingsConflicts {
		gotFields[f.Section+"."+f.Field] = true
	}
	if !gotFields["pomodoro.work_duration"] {
		t.Errorf("pomodoro.work_duration diff missing: %+v", m.SettingsConflicts)
	}
	if !gotFields["ai.api_key"] {
		t.Errorf("ai.api_key diff missing (backend must NOT mask): %+v", m.SettingsConflicts)
	}
}

func TestDataService_PreviewImport_WorkLogAtomicConflict(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	// 标量相同,但 items 数量不同 → 仍判 conflict
	file := &model.BackupData{WorkLogs: []model.WorkLog{{ID: "wl1", Date: "2026-08-07", Items: []model.WorkItem{
		{ID: "wi1", WorkLogID: "wl1", Seq: 1},
		{ID: "wi2", WorkLogID: "wl1", Seq: 2},
	}}}}
	prev, _ := svc.PreviewImport(file)
	m := prev.Modules["work_logs"]
	if m.Conflict != 1 {
		t.Errorf("worklog with differing items should be conflict, got conflict=%d", m.Conflict)
	}
}

func TestDataService_PreviewImport_SchemaWarning(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	// 直接构造一个未来版本无法触达:用 Export 得到当前版本,然后断言无告警;再断言不匹配时告警
	prev, _ := svc.PreviewImport(newSnapshot())
	if prev.SchemaWarning != "" {
		t.Errorf("same-version should have no warning, got %q", prev.SchemaWarning)
	}
}
```

> 注:SchemaVersion 不匹配的告警基于「文件版本 ≠ 当前 BackupSchemaVersion」。preview 入参是 `*model.BackupData`(已脱壳的信封 data),版本信息在 handler 层从信封取出后传给 service —— 见 Task 7。这里 service 提供 `PreviewImport(file, fileVersion int)` 形态。**因此把 Task 4 里的签名改为下方 Step 3 的版本。**

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/service/ -run TestDataService_PreviewImport -v`
Expected: FAIL — PreviewImport 返回 nil。

- [ ] **Step 3: 实现 PreviewImport + diff helpers**

把 `data_service.go` 顶部 import 增加 `"encoding/json"` 和 `"reflect"`,把签名与实现替换为:

```go
func (s *dataService) PreviewImport(file *model.BackupData, fileVersion int) (*model.ImportPreview, error) {
	cur, err := s.repo.ReadAll()
	if err != nil {
		return nil, err
	}
	warning := ""
	if fileVersion != model.BackupSchemaVersion {
		warning = fmt.Sprintf("备份 schema 版本 %d 与当前 %d 不一致,导入可能不完整", fileVersion, model.BackupSchemaVersion)
	}
	return &model.ImportPreview{
		SchemaVersion: model.BackupSchemaVersion,
		SchemaWarning: warning,
		Modules: map[string]*model.ModulePreview{
			"tasks":        classify(cur.Tasks, file.Tasks, idOfTask),
			"sessions":     classify(cur.Sessions, file.Sessions, idOfSession),
			"schedules":    classify(cur.Schedules, file.Schedules, idOfSchedule),
			"work_reports": classify(cur.WorkReports, file.WorkReports, idOfWorkReport),
			"work_logs":    classifyWorkLogs(cur.WorkLogs, file.WorkLogs),
			"settings":     diffSettings(cur.Settings, file.Settings),
		},
	}, nil
}
```

同时把接口与 Task 4 的 `PreviewImport(file *model.BackupData)` 改为 `PreviewImport(file *model.BackupData, fileVersion int)`(接口、实现、Task 4 测试调用处同步 —— Task 4 测试目前没调 Preview,不受影响)。

追加 helpers(`data_service.go` 末尾):

```go
// classify 泛型分类:按 id 把 file 记录归入 new/identical/conflict,把 cur 独有的归入 orphan。
func classify[T any](cur, file []T, idOf func(T) string) *model.ModulePreview {
	curByID := map[string]T{}
	for _, r := range cur {
		curByID[idOf(r)] = r
	}
	fileByID := map[string]T{}
	for _, r := range file {
		fileByID[idOf(r)] = r
	}

	m := &model.ModulePreview{Conflicts: []model.RecordConflict{}}
	for _, r := range file {
		id := idOf(r)
		ex, inCur := curByID[id]
		if !inCur {
			m.New++
			continue
		}
		if jsonEqual(r, ex) {
			m.Identical++
			continue
		}
		m.Conflict++
		m.Conflicts = append(m.Conflicts, model.RecordConflict{ID: id, Fields: fieldDiffs(ex, r)})
	}
	for id := range curByID {
		if _, inFile := fileByID[id]; !inFile {
			m.Orphan++
		}
	}
	return m
}

func classifyWorkLogs(cur, file []model.WorkLog) *model.ModulePreview {
	// 整条 log(含 items)参与 identical 判定 → 原子。
	return classify(cur, file, func(l model.WorkLog) string { return l.ID })
}

func diffSettings(cur, file model.SettingsBundle) *model.ModulePreview {
	m := &model.ModulePreview{SettingsConflicts: []model.SettingsFieldDiff{}}
	m.SettingsConflicts = append(m.SettingsConflicts, diffSection("pomodoro", cur.Pomodoro, file.Pomodoro)...)
	m.SettingsConflicts = append(m.SettingsConflicts, diffSection("ai", cur.AI, file.AI)...)
	return m
}

func diffSection(section string, cur, file any) []model.SettingsFieldDiff {
	if cur == nil || file == nil {
		return nil
	}
	out := []model.SettingsFieldDiff{}
	cm := toMap(cur)
	fm := toMap(file)
	for k, cv := range cm {
		fv, ok := fm[k]
		if !ok {
			continue // 文件缺该字段,跳过(不视为冲突)
		}
		if !reflect.DeepEqual(cv, fv) {
			out = append(out, model.SettingsFieldDiff{Section: section, Field: k, Current: cv, Imported: fv})
		}
	}
	return out
}

// toMap 把结构 marshal 再 unmarshal 成 map[string]any,便于逐字段比对。
func toMap(v any) map[string]any {
	raw, _ := json.Marshal(v)
	m := map[string]any{}
	_ = json.Unmarshal(raw, &m)
	return m
}

func jsonEqual(a, b any) bool {
	ra, _ := json.Marshal(a)
	rb, _ := json.Marshal(b)
	return bytes.Equal(ra, rb)
}

// fieldDiffs 对比两条同类型记录,列出值不同的字段(canonical JSON)。
func fieldDiffs(cur, file any) []model.FieldDiff {
	cm := toMap(cur)
	fm := toMap(file)
	out := []model.FieldDiff{}
	for k, cv := range cm {
		fv, ok := fm[k]
		if !ok {
			continue
		}
		if !reflect.DeepEqual(cv, fv) {
			out = append(out, model.FieldDiff{Field: k, Current: cv, Imported: fv})
		}
	}
	return out
}

// id 提取器
func idOfTask(t model.Task) string             { return t.ID }
func idOfSession(s model.PomodoroSession) string { return s.ID }
func idOfSchedule(s model.Schedule) string     { return s.ID }
func idOfWorkReport(r model.WorkReport) string { return r.ID }
```

import 块补 `"bytes"` 和 `"fmt"`。

> work_logs 复用 `classify` + 提取 `l.ID`,因 `classify` 内 `jsonEqual` 会 marshal 整条 WorkLog(含 Items),自然实现「items 不同 → 非 identical → conflict」的原子判定。

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend && go test ./internal/service/ -run TestDataService_PreviewImport -v`
Expected: PASS(含 settings api_key 不掩码、work_logs 原子冲突)。

- [ ] **Step 5: 提交**

```bash
git add backend/internal/service/data_service.go backend/internal/service/data_service_test.go
git commit -m "feat(data): add DataService.PreviewImport (per-module diff + settings field diff)"
```

---

## Task 6: DataService.ApplyImport (build plan + execute)

**Files:**
- Modify: `backend/internal/service/data_service.go`
- Test: `backend/internal/service/data_service_test.go`

- [ ] **Step 1: 写失败测试 — 表驱动:4 策略 × 计数 + override + settings 落表 + 未知 override 忽略**

Append to `data_service_test.go`:

```go
func TestDataService_ApplyImport_Policies(t *testing.T) {
	// 当前库:t1(title=x,status=todo);文件:t1(title=changed) + t2(new)
	cur := newSnapshot()
	file := &model.BackupData{Tasks: []model.Task{
		{ID: "t1", Title: "changed", Status: model.StatusTodo},
		{ID: "t2", Title: "brand", Status: model.StatusTodo},
	}}

	cases := []struct {
		policy              string
		inserted, updated, deleted int
		t1Title             string // apply 后 t1 应为的 title
	}{
		{model.PolicyAddNewOnly, 1, 0, 0, "x"},        // 冲突跳过 → 保留当前
		{model.PolicyMergeFile, 1, 1, 0, "changed"},   // 文件优先
		{model.PolicyMergeCurrent, 1, 0, 0, "x"},      // 当前优先
		{model.PolicyReplace, 1, 1, 0, "changed"},     // 冲突用文件值(orphan=0 因 cur 的 t1 在文件里)
	}
	for _, c := range cases {
		t.Run(c.policy, func(t *testing.T) {
			repo := &mockBackupRepo{snapshot: cur}
			svc := NewDataService(repo)
			res, err := svc.ApplyImport(&model.ApplyImportRequest{
				Data:    *file,
				Modules: map[string]model.ModuleApply{"tasks": {Policy: c.policy}},
			})
			if err != nil {
				t.Fatalf("apply: %v", err)
			}
			tr := res.Applied["tasks"]
			if tr.Inserted != c.inserted || tr.Updated != c.updated || tr.Deleted != c.deleted {
				t.Errorf("counts: got i%d u%d d%d", tr.Inserted, tr.Updated, tr.Deleted)
			}
			// 校验 plan 里 t1 的 title
			var t1 model.Task
			for _, x := range repo.lastPlan.Tasks {
				if x.ID == "t1" {
					t1 = x
				}
			}
			if t1.Title != c.t1Title {
				t.Errorf("t1 title: got %q want %q", t1.Title, c.t1Title)
			}
		})
	}
}

func TestDataService_ApplyImport_ReplaceDeletesOrphans(t *testing.T) {
	cur := &model.BackupData{Tasks: []model.Task{{ID: "orphan", Title: "o", Status: model.StatusTodo}}}
	file := &model.BackupData{Tasks: []model.Task{}} // 文件没有 orphan
	repo := &mockBackupRepo{snapshot: cur}
	svc := NewDataService(repo)
	res, _ := svc.ApplyImport(&model.ApplyImportRequest{
		Data:    *file,
		Modules: map[string]model.ModuleApply{"tasks": {Policy: model.PolicyReplace}},
	})
	if res.Applied["tasks"].Deleted != 1 {
		t.Errorf("replace should delete orphan, got deleted=%d", res.Applied["tasks"].Deleted)
	}
	if len(repo.lastPlan.DeleteTasks) != 1 || repo.lastPlan.DeleteTasks[0] != "orphan" {
		t.Errorf("orphan not in delete plan: %+v", repo.lastPlan.DeleteTasks)
	}
}

func TestDataService_ApplyImport_OverrideBeatsPolicy(t *testing.T) {
	cur := newSnapshot()
	file := &model.BackupData{Tasks: []model.Task{{ID: "t1", Title: "changed", Status: model.StatusTodo}}}
	repo := &mockBackupRepo{snapshot: cur}
	svc := NewDataService(repo)
	_, _ = svc.ApplyImport(&model.ApplyImportRequest{
		Data: *file,
		Modules: map[string]model.ModuleApply{"tasks": {
			Policy:    model.PolicyMergeFile, // 本应写 changed
			Overrides: map[string]string{"t1": model.ChoiceCurrent}, // 但强制保留当前
		}},
	})
	var t1 model.Task
	for _, x := range repo.lastPlan.Tasks {
		if x.ID == "t1" {
			t1 = x
		}
	}
	if t1.Title != "x" {
		t.Errorf("override=current should keep 'x', got %q", t1.Title)
	}
}

func TestDataService_ApplyImport_SettingsWritten(t *testing.T) {
	cur := newSnapshot()
	pomo := model.DefaultPomodoroSettings()
	pomo.WorkDuration = 1800
	file := &model.BackupData{Settings: model.SettingsBundle{Pomodoro: pomo, AI: model.DefaultAISettings()}}
	repo := &mockBackupRepo{snapshot: cur}
	svc := NewDataService(repo)
	_, _ = svc.ApplyImport(&model.ApplyImportRequest{Data: *file, Modules: map[string]model.ModuleApply{}})
	if repo.lastPlan.Settings == nil || repo.lastPlan.Settings.Pomodoro.WorkDuration != 1800 {
		t.Errorf("settings not in plan: %+v", repo.lastPlan.Settings)
	}
}

func TestDataService_ApplyImport_UnknownOverrideIgnored(t *testing.T) {
	cur := newSnapshot()
	file := newSnapshot()
	repo := &mockBackupRepo{snapshot: cur}
	svc := NewDataService(repo)
	_, err := svc.ApplyImport(&model.ApplyImportRequest{
		Data: *file,
		Modules: map[string]model.ModuleApply{"tasks": {
			Policy:    model.PolicyMergeFile,
			Overrides: map[string]string{"does-not-exist": model.ChoiceFile},
		}},
	})
	if err != nil {
		t.Errorf("unknown override id should be ignored, not error: %v", err)
	}
}

func TestDataService_ApplyImport_InvalidPolicy(t *testing.T) {
	svc := NewDataService(&mockBackupRepo{snapshot: newSnapshot()})
	_, err := svc.ApplyImport(&model.ApplyImportRequest{
		Data:    *newSnapshot(),
		Modules: map[string]model.ModuleApply{"tasks": {Policy: "bogus"}},
	})
	if err == nil {
		t.Error("invalid policy should error")
	}
}
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/service/ -run TestDataService_ApplyImport -v`
Expected: FAIL — ApplyImport 返回 nil。

- [ ] **Step 3: 实现 ApplyImport + plan 构造**

把 `data_service.go` 的 `ApplyImport` 替换为:

```go
var validPolicies = map[string]bool{
	model.PolicyAddNewOnly: true, model.PolicyMergeFile: true,
	model.PolicyMergeCurrent: true, model.PolicyReplace: true,
}

func (s *dataService) ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error) {
	// 1. 策略校验(把非法 policy 转为 400 由 handler 呈现)
	for key, mod := range req.Modules {
		if !validPolicies[mod.Policy] {
			return nil, fmt.Errorf("invalid policy %q for module %q", mod.Policy, key)
		}
	}

	// 2. 读当前库,构造 plan
	cur, err := s.repo.ReadAll()
	if err != nil {
		return nil, err
	}
	plan := model.ApplyPlan{Settings: &req.Data.Settings}
	result := &model.ApplyResult{Applied: map[string]model.ModuleApplyResult{}}

	// 3. 每模块按 policy + override 计算 upsert/delete + 计数
	plan.Tasks, plan.DeleteTasks, result.Applied["tasks"] = resolveModule(
		cur.Tasks, req.Data.Tasks, req.Modules["tasks"], idOfTask)
	plan.Sessions, plan.DeleteSessions, result.Applied["sessions"] = resolveModule(
		cur.Sessions, req.Data.Sessions, req.Modules["sessions"], idOfSession)
	plan.Schedules, plan.DeleteSchedules, result.Applied["schedules"] = resolveModule(
		cur.Schedules, req.Data.Schedules, req.Modules["schedules"], idOfSchedule)
	plan.WorkReports, plan.DeleteWorkReports, result.Applied["work_reports"] = resolveModule(
		cur.WorkReports, req.Data.WorkReports, req.Modules["work_reports"], idOfWorkReport)
	plan.WorkLogs, plan.DeleteWorkLogs, result.Applied["work_logs"] = resolveModule(
		cur.WorkLogs, req.Data.WorkLogs, req.Modules["work_logs"], func(l model.WorkLog) string { return l.ID })

	// 4. 单事务执行
	if err := s.repo.Apply(plan); err != nil {
		return nil, err
	}
	return result, nil
}

// resolveModule 按 policy + overrides 计算某表的 upsert / delete 集合 + 计数。
// 返回:toUpsert(新增 + 冲突解决为 file 的),toDelete(replace 下的 orphan),计数。
func resolveModule[T any](cur, file []T, mod model.ModuleApply, idOf func(T) string) (upsert []T, del []string, r model.ModuleApplyResult) {
	if !validPolicies[mod.Policy] {
		// 调用方在 ApplyImport 里先校验,这里防御性返回空
		return nil, nil, model.ModuleApplyResult{}
	}
	curByID := map[string]T{}
	for _, x := range cur {
		curByID[idOf(x)] = x
	}
	fileByID := map[string]T{}
	for _, x := range file {
		fileByID[idOf(x)] = x
	}
	upsert = []T{}
	del = []string{}

	wantFile := func(id string) bool {
		// override 优先
		if ch, ok := mod.Overrides[id]; ok {
			return ch == model.ChoiceFile
		}
		switch mod.Policy {
		case model.PolicyMergeFile, model.PolicyReplace:
			return true
		default: // add_new_only, merge_current
			return false
		}
	}

	for _, x := range file {
		id := idOf(x)
		ex, inCur := curByID[id]
		if !inCur {
			upsert = append(upsert, x) // 新增:所有策略都插入
			r.Inserted++
			continue
		}
		if jsonEqual(x, ex) {
			continue // identical:跳过
		}
		// 冲突
		if wantFile(id) {
			upsert = append(upsert, x)
			r.Updated++
		}
		// 否则保留当前,不动
	}
	if mod.Policy == model.PolicyReplace {
		for id := range curByID {
			if _, inFile := fileByID[id]; !inFile {
				del = append(del, id)
				r.Deleted++
			}
		}
	}
	return upsert, del, r
}
```

import 块需含 `"fmt"`(Task 5 已加)。`resolveModule` 的泛型实现收敛了 5 个表的 policy→plan 逻辑(DRY);非法 policy 在 `ApplyImport` 开头即返回 error(handler 转为 400)。

- [ ] **Step 4: 跑测试确认通过**

Run: `cd backend && go test ./internal/service/ -v`
Expected: PASS(全部 Export/Preview/Apply 用例)。

- [ ] **Step 5: 提交**

```bash
git add backend/internal/service/data_service.go backend/internal/service/data_service_test.go
git commit -m "feat(data): add DataService.ApplyImport (policy + override → plan → tx)"
```

---

## Task 7: Handler + router + main.go wiring

**Files:**
- Create: `backend/internal/api/handler/data.go`
- Create: `backend/internal/api/handler/data_test.go`
- Modify: `backend/internal/api/router.go`
- Modify: `backend/cmd/server/main.go`

- [ ] **Step 1: 写失败测试 — export 下载头 / preview multipart / apply / 错误路径(mock service)**

Create `backend/internal/api/handler/data_test.go`:

```go
package handler

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"ticktask/internal/model"
)

// mockDataService 实现 service.DataService
type mockDataService struct {
	exportEnvelop  *model.BackupEnvelope
	exportErr      error
	previewResult  *model.ImportPreview
	previewErr     error
	applyResult    *model.ApplyResult
	applyErr       error
	lastFileVersion int
}

func (m *mockDataService) Export() (*model.BackupEnvelope, error) { return m.exportEnvelop, m.exportErr }
func (m *mockDataService) PreviewImport(file *model.BackupData, fileVersion int) (*model.ImportPreview, error) {
	m.lastFileVersion = fileVersion
	return m.previewResult, m.previewErr
}
func (m *mockDataService) ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error) {
	return m.applyResult, m.applyErr
}

func TestDataHandler_Export(t *testing.T) {
	h := NewDataHandler(&mockDataService{exportEnvelop: &model.BackupEnvelope{App: "ticktask", SchemaVersion: 1, Data: model.BackupData{}}})
	r := setupTestRouter()
	r.GET("/api/data/export", h.Export)

	req, _ := http.NewRequest("GET", "/api/data/export", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d", w.Code)
	}
	if cd := w.Header().Get("Content-Disposition"); cd == "" || !strings.Contains(cd, "attachment") {
		t.Errorf("missing attachment Content-Disposition: %q", cd)
	}
	var env model.BackupEnvelope
	json.Unmarshal(w.Body.Bytes(), &env)
	if env.App != "ticktask" {
		t.Errorf("body not envelope: %s", w.Body.String())
	}
}

func TestDataHandler_PreviewImport(t *testing.T) {
	h := NewDataHandler(&mockDataService{previewResult: &model.ImportPreview{Modules: map[string]*model.ModulePreview{"tasks": {New: 1}}}})
	r := setupTestRouter()
	r.POST("/api/data/import/preview", h.PreviewImport)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, _ := writer.CreateFormFile("file", "b.json")
	env := model.BackupEnvelope{App: "ticktask", SchemaVersion: 1, Data: model.BackupData{Tasks: []model.Task{{ID: "t1"}}}}
	raw, _ := json.Marshal(env)
	fw.Write(raw)
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/data/import/preview", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var prev model.ImportPreview
	json.Unmarshal(w.Body.Bytes(), &prev)
	if prev.Modules["tasks"] == nil || prev.Modules["tasks"].New != 1 {
		t.Errorf("preview not returned: %s", w.Body.String())
	}
}

func TestDataHandler_PreviewImport_BadFile(t *testing.T) {
	h := NewDataHandler(&mockDataService{})
	r := setupTestRouter()
	r.POST("/api/data/import/preview", h.PreviewImport)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fw, _ := writer.CreateFormFile("file", "b.json")
	fw.Write([]byte("not json"))
	writer.Close()

	req, _ := http.NewRequest("POST", "/api/data/import/preview", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestDataHandler_ApplyImport(t *testing.T) {
	h := NewDataHandler(&mockDataService{applyResult: &model.ApplyResult{Applied: map[string]model.ModuleApplyResult{"tasks": {Inserted: 1}}}})
	r := setupTestRouter()
	r.POST("/api/data/import/apply", h.ApplyImport)

	reqBody, _ := json.Marshal(model.ApplyImportRequest{Data: model.BackupData{}, Modules: map[string]model.ModuleApply{}})
	req, _ := http.NewRequest("POST", "/api/data/import/apply", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
}

func TestDataHandler_ApplyImport_ServiceError(t *testing.T) {
	h := NewDataHandler(&mockDataService{applyErr: errBoom})
	r := setupTestRouter()
	r.POST("/api/data/import/apply", h.ApplyImport)

	reqBody, _ := json.Marshal(model.ApplyImportRequest{Data: model.BackupData{}, Modules: map[string]model.ModuleApply{}})
	req, _ := http.NewRequest("POST", "/api/data/import/apply", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
```

在 `data_test.go` 顶部 import 加 `"errors"` 和 `"strings"`,并加 `var errBoom = errors.New("boom")`。

- [ ] **Step 2: 跑测试确认失败**

Run: `cd backend && go test ./internal/api/handler/ -run TestDataHandler -v`
Expected: FAIL — `NewDataHandler` undefined。

- [ ] **Step 3: 实现 DataHandler**

Create `backend/internal/api/handler/data.go`:

```go
package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"ticktask/internal/model"
	"ticktask/internal/service"

	"github.com/gin-gonic/gin"
)

const maxImportSize = 50 << 20 // 50MB

type DataHandler struct {
	svc service.DataService
}

func NewDataHandler(svc service.DataService) *DataHandler {
	return &DataHandler{svc: svc}
}

// Export GET /api/data/export → 下载 JSON
func (h *DataHandler) Export(c *gin.Context) {
	env, err := h.svc.Export()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	raw, err := json.Marshal(env)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	name := fmt.Sprintf("ticktask-backup-%s.json", time.Now().UTC().Format("20060102-150405"))
	c.Header("Content-Disposition", `attachment; filename="`+name+`"`)
	c.Data(http.StatusOK, "application/json", raw)
}

// PreviewImport POST /api/data/import/preview (multipart "file")
func (h *DataHandler) PreviewImport(c *gin.Context) {
	env, err := readBackupUpload(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	prev, err := h.svc.PreviewImport(&env.Data, env.SchemaVersion)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, prev)
}

// ApplyImport POST /api/data/import/apply (JSON)
func (h *DataHandler) ApplyImport(c *gin.Context) {
	var req model.ApplyImportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	res, err := h.svc.ApplyImport(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}

// readBackupUpload 读 multipart 文件并解析信封,做基础校验。
func readBackupUpload(c *gin.Context) (*model.BackupEnvelope, error) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		return nil, fmt.Errorf("文件格式无效:缺少 file")
	}
	defer file.Close()
	raw, err := io.ReadAll(io.LimitReader(file, maxImportSize+1))
	if err != nil {
		return nil, fmt.Errorf("文件格式无效:读取失败")
	}
	if len(raw) > maxImportSize {
		return nil, fmt.Errorf("文件过大(>50MB)")
	}
	var env model.BackupEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return nil, fmt.Errorf("文件格式无效")
	}
	if env.App != "ticktask" {
		return nil, fmt.Errorf("不是有效的 TickTask 备份文件")
	}
	return &env, nil
}
```

- [ ] **Step 4: 跑 handler 测试确认通过**

Run: `cd backend && go test ./internal/api/handler/ -run TestDataHandler -v`
Expected: PASS。

- [ ] **Step 5: 接入 router + main.go**

修改 `backend/internal/api/router.go`:
- `SetupRouter` 签名末尾新增参数 `dataService *service.DataService`。
- 在 `api := r.Group("/api")` 块内(任意位置,建议放 settings 之后)加:

```go
		// 数据导入导出
		data := api.Group("/data")
		{
			dataHandler := handler.NewDataHandler(dataService)
			data.GET("/export", dataHandler.Export)
			data.POST("/import/preview", dataHandler.PreviewImport)
			data.POST("/import/apply", dataHandler.ApplyImport)
		}
```

修改 `backend/cmd/server/main.go`:
- 在 repository 初始化区加:`dataRepo := repository.NewDataRepository(db)`
- 在 service 初始化区加:`dataService := service.NewDataService(dataRepo)`
- 把 `api.SetupRouter(...)` 调用末尾加上 `dataService` 参数。

- [ ] **Step 6: 编译 + 全量后端测试**

Run: `cd backend && go build ./... && go test ./internal/... -v`
Expected: 编译通过;全部测试 PASS。

- [ ] **Step 7: 提交**

```bash
git add backend/internal/api/handler/data.go backend/internal/api/handler/data_test.go backend/internal/api/router.go backend/cmd/server/main.go
git commit -m "feat(data): add /api/data endpoints (export/import-preview/import-apply) + wiring"
```

---

## Task 8: Frontend types + API client

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/client.ts`
- Test: `frontend/src/api/client.test.ts`(已存在,追加用例)

- [ ] **Step 1: 写失败测试 — 三个新方法的调用形态**

在 `frontend/src/api/client.test.ts` 末尾追加(沿用文件现有 `vi.mock('../client')` 或直接 mock axios 的模式 —— 参考文件里已有的 mock 写法):

```ts
import { api } from './client'
import { client } from './client'

vi.mock('axios', () => {
  const instance = { get: vi.fn(), post: vi.fn(), put: vi.fn(), delete: vi.fn(), patch: vi.fn() }
  return { default: { create: () => instance } }
})

describe('data backup api', () => {
  it('exportData calls GET /data/export with blob', async () => {
    ;(client as any).get.mockResolvedValueOnce({ data: new Blob(['{}']) })
    await api.exportData()
    expect((client as any).get).toHaveBeenCalledWith('/data/export', { responseType: 'blob' })
  })

  it('previewImport posts FormData to /data/import/preview', async () => {
    ;(client as any).post.mockResolvedValueOnce({ data: { modules: {} } })
    const file = new File(['{}'], 'b.json')
    await api.previewImport(file)
    const args = (client as any).post.mock.calls[0]
    expect(args[0]).toBe('/data/import/preview')
    expect(args[1]).toBeInstanceOf(FormData)
    expect(args[2]?.headers?.['Content-Type']).toContain('multipart/form-data')
  })

  it('applyImport posts JSON to /data/import/apply', async () => {
    ;(client as any).post.mockResolvedValueOnce({ data: { applied: {} } })
    await api.applyImport({ data: { tasks: [] } as any, modules: {} })
    expect((client as any).post).toHaveBeenCalledWith('/data/import/apply', { data: { tasks: [] }, modules: {} })
  })
})
```

> 若 `client.test.ts` 现有结构不兼容上面的顶层 `vi.mock`,改为在该文件已有的 mock 复用上追加三个 `it`;关键断言是 URL、blob、FormData、JSON body。

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/api/client.test.ts`
Expected: FAIL — `api.exportData` undefined。

- [ ] **Step 3: 加类型 + API 方法**

在 `frontend/src/types/index.ts` 末尾追加:

```ts
// ── 数据导入导出 ──

export type ImportPolicy = 'add_new_only' | 'merge_file' | 'merge_current' | 'replace'
export type ImportChoice = 'file' | 'current'

export interface BackupData {
  tasks: Task[]
  sessions: PomodoroSession[]
  schedules: ScheduleEvent[] // 后端 Schedule 序列化字段;导入沿用(见下注)
  settings: { pomodoro: PomodoroSettings; ai: AISettings }
  work_logs: WorkLog[]
  work_reports: WorkReport[]
}

export interface BackupEnvelope {
  app: string
  schema_version: number
  exported_at: string
  data: BackupData
}

export interface FieldDiff {
  field: string
  current: unknown
  imported: unknown
}
export interface RecordConflict {
  id: string
  fields: FieldDiff[]
}
export interface SettingsFieldDiff {
  section: 'pomodoro' | 'ai'
  field: string
  current: unknown
  imported: unknown
}
export interface ModulePreview {
  new: number
  identical: number
  conflict: number
  orphan: number
  conflicts: RecordConflict[]
  settings_conflicts: SettingsFieldDiff[]
}
export interface ImportPreview {
  schema_version: number
  schema_warning: string
  modules: Record<string, ModulePreview>
}

export interface ModuleApply {
  policy: ImportPolicy
  overrides: Record<string, ImportChoice>
}
export interface ApplyImportRequest {
  data: BackupData
  modules: Record<string, ModuleApply>
}
export interface ModuleApplyResult {
  inserted: number
  updated: number
  deleted: number
}
export interface ApplyResult {
  applied: Record<string, ModuleApplyResult>
}
```

> 注:后端 `model.Schedule` 与前端 `ScheduleEvent` 字段不完全一致(后者是给日历视图的投影)。为避免歧义,在类型里把 `BackupData.schedules` 标为 `ScheduleEvent[]` 仅作透传 —— 导入时后端按 `model.Schedule` 解析,前端不读这层字段。若 `vue-tsc` 报字段不匹配,改为 `schedules: unknown[]`。**实现时优先用 `unknown[]` 以通过严格类型。**

在 `frontend/src/api/client.ts`:
- 顶部 import 追加新类型:`BackupEnvelope, ImportPreview, ApplyImportRequest, ApplyResult`。
- 在 `export const api = { ... }` 内(末尾 `updateWorkLogSummary` 之后)加:

```ts
  // 数据导入导出
  exportData: () =>
    client.get<Blob>('/data/export', { responseType: 'blob' }),
  previewImport: (file: File) => {
    const form = new FormData()
    form.append('file', file)
    return client.post<ImportPreview>('/data/import/preview', form, {
      headers: { 'Content-Type': 'multipart/form-data' }
    })
  },
  applyImport: (payload: ApplyImportRequest) =>
    client.post<ApplyResult>('/data/import/apply', payload),
```

并把 `BackupData.schedules` 在 `types/index.ts` 里定为 `unknown[]`(按上面注释),确保 vue-tsc 通过。

- [ ] **Step 4: 跑测试 + 类型检查**

Run: `cd frontend && npx vitest run src/api/client.test.ts && npx vue-tsc --noEmit`
Expected: 测试 PASS;类型检查通过。

- [ ] **Step 5: 提交**

```bash
git add frontend/src/types/index.ts frontend/src/api/client.ts frontend/src/api/client.test.ts
git commit -m "feat(data): add backup/import types + api client methods"
```

---

## Task 9: ImportWizard.vue (three-step wizard)

**Files:**
- Create: `frontend/src/components/settings/ImportWizard.vue`
- Create: `frontend/src/components/settings/ImportWizard.test.ts`

- [ ] **Step 1: 写失败测试 — 步骤流转 / 渲染计数 / api_key 掩码 / 策略变更产出 payload / replace 二次确认 / 取消 / apply 成败**

Create `frontend/src/components/settings/ImportWizard.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import ImportWizard from './ImportWizard.vue'
import { api } from '@/api/client'
import type { ImportPreview } from '@/types'

vi.mock('@/api/client', () => ({
  api: {
    previewImport: vi.fn(),
    applyImport: vi.fn()
  }
}))
vi.mock('element-plus', async () => {
  const actual: any = await vi.importActual('element-plus')
  return { ...actual, ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() } }
})

const preview: ImportPreview = {
  schema_version: 1,
  schema_warning: '',
  modules: {
    tasks: { new: 2, identical: 1, conflict: 1, orphan: 0, conflicts: [{ id: 't1', fields: [{ field: 'status', current: 'todo', imported: 'done' }] }], settings_conflicts: [] },
    settings: { new: 0, identical: 0, conflict: 0, orphan: 0, conflicts: [], settings_conflicts: [
      { section: 'ai', field: 'api_key', current: 'secret-current', imported: 'secret-imported' }
    ] }
  }
}

beforeEach(() => {
  setActivePinia(createPinia())
  ;(api.previewImport as any).mockResolvedValue({ data: preview })
  ;(api.applyImport as any).mockResolvedValue({ data: { applied: { tasks: { inserted: 2, updated: 1, deleted: 0 } } } })
})

describe('ImportWizard', () => {
  it('renders preview counts after file selected', async () => {
    const w = mount(ImportWizard)
    ;(w.vm as any).onFileSelected(new File(['{}'], 'b.json', { type: 'application/json' }))
    await flushPromises()
    expect(w.text()).toContain('新增')
    expect(w.text()).toContain('冲突')
  })

  it('masks api_key in settings diff', async () => {
    const w = mount(ImportWizard)
    ;(w.vm as any).onFileSelected(new File(['{}'], 'b.json'))
    await flushPromises()
    expect(w.text()).not.toContain('secret-current')
    expect(w.text()).not.toContain('secret-imported')
    expect(w.text()).toContain('••••')
  })

  it('changing task policy updates apply payload', async () => {
    const w = mount(ImportWizard)
    ;(w.vm as any).onFileSelected(new File(['{}'], 'b.json'))
    await flushPromises()
    ;(w.vm as any).setPolicy('tasks', 'merge_current')
    expect((w.vm as any).applyPayload.modules.tasks.policy).toBe('merge_current')
  })

  it('replace requires confirm and cancel does not apply', async () => {
    const w = mount(ImportWizard)
    ;(w.vm as any).onFileSelected(new File(['{}'], 'b.json'))
    await flushPromises()
    ;(w.vm as any).setPolicy('tasks', 'replace')
    ;(w.vm as any).confirmReplace(false) // 用户取消
    ;(w.vm as any).clickApply()
    await flushPromises()
    expect(api.applyImport).not.toHaveBeenCalled()
  })

  it('apply success emits applied', async () => {
    const w = mount(ImportWizard)
    ;(w.vm as any).onFileSelected(new File(['{}'], 'b.json'))
    await flushPromises()
    ;(w.vm as any).clickApply()
    await flushPromises()
    expect(api.applyImport).toHaveBeenCalled()
    expect(w.emitted('applied')).toBeTruthy()
  })

  it('apply failure shows error and keeps wizard open', async () => {
    ;(api.applyImport as any).mockRejectedValueOnce(new Error('boom'))
    const w = mount(ImportWizard)
    ;(w.vm as any).onFileSelected(new File(['{}'], 'b.json'))
    await flushPromises()
    ;(w.vm as any).clickApply()
    await flushPromises()
    expect(w.emitted('applied')).toBeFalsy()
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/components/settings/ImportWizard.test.ts`
Expected: FAIL — 组件不存在。

- [ ] **Step 3: 实现 ImportWizard.vue**

Create `frontend/src/components/settings/ImportWizard.vue`:

```vue
<template>
  <el-dialog :model-value="modelValue" @update:model-value="$emit('update:modelValue', $event)" title="导入数据" width="720px">
    <!-- Step 1: 选文件 -->
    <div v-if="step === 'select'">
      <input type="file" accept="application/json" ref="fileInput" @change="onFileInputChange" />
      <p v-if="previewError" class="error">{{ previewError }}</p>
    </div>

    <!-- Step 2: 预览 -->
    <div v-if="step === 'preview'">
      <el-alert v-if="preview.schema_warning" type="warning" :title="preview.schema_warning" :closable="false" />

      <!-- 集合模块总览 + 策略 -->
      <div v-for="key in collectionKeys" :key="key" class="module-row">
        <div class="module-summary">
          <strong>{{ moduleLabel[key] }}</strong>
          <span>新增 {{ preview.modules[key]?.new || 0 }}</span>
          <span>相同 {{ preview.modules[key]?.identical || 0 }}</span>
          <span>冲突 {{ preview.modules[key]?.conflict || 0 }}</span>
          <span>仅当前 {{ preview.modules[key]?.orphan || 0 }}</span>
        </div>
        <el-select :model-value="applyPayload.modules[key]?.policy || 'add_new_only'" @update:model-value="setPolicy(key, $event)" size="small" style="width: 160px">
          <el-option label="只加新的" value="add_new_only" />
          <el-option label="文件优先" value="merge_file" />
          <el-option label="当前优先" value="merge_current" />
          <el-option label="整模块覆盖" value="replace" />
        </el-select>
      </div>

      <!-- 设置字段级 diff -->
      <div v-if="settingsConflicts.length" class="settings-diff">
        <div class="section-label">设置冲突(逐字段)</div>
        <div v-for="c in settingsConflicts" :key="c.section + '.' + c.field" class="diff-row">
          <span class="diff-field">{{ c.section }}.{{ c.field }}</span>
          <el-radio-group :model-value="settingsChoice[c.section + '.' + c.field] || 'current'" @update:model-value="setSettingsChoice(c.section, c.field, $event)">
            <el-radio value="current">当前:{{ mask(c.current) }}</el-radio>
            <el-radio value="file">导入:{{ mask(c.imported) }}</el-radio>
          </el-radio-group>
        </div>
      </div>
    </div>

    <template #footer>
      <el-button @click="$emit('update:modelValue', false)">取消</el-button>
      <el-button v-if="step === 'preview'" type="primary" :loading="applying" @click="clickApply">应用导入</el-button>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { ref, computed, reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '@/api/client'
import type { ImportPreview, ImportPolicy, ImportChoice, BackupData, ApplyImportRequest } from '@/types'

const props = defineProps<{ modelValue: boolean; fileData?: BackupData }>()
const emit = defineEmits<{ (e: 'update:modelValue', v: boolean): void; (e: 'applied'): void }>()

const step = ref<'select' | 'preview'>('select')
const preview = ref<ImportPreview>({ schema_version: 1, schema_warning: '', modules: {} })
const filePayload = ref<BackupData | null>(null)
const previewError = ref('')
const applying = ref(false)

const collectionKeys = ['tasks', 'sessions', 'schedules', 'work_logs', 'work_reports'] as const
const moduleLabel: Record<string, string> = {
  tasks: '任务', sessions: '番茄钟会话', schedules: '日程', work_logs: '工作日志', work_reports: '周期报告'
}

const policies = reactive<Record<string, ImportPolicy>>({})
const overrides = reactive<Record<string, Record<string, ImportChoice>>>({})
const settingsChoice = reactive<Record<string, 'current' | 'file'>>({})

const settingsConflicts = computed(() => preview.value.modules['settings']?.settings_conflicts || [])

const applyPayload = computed<ApplyImportRequest>(() => {
  const modules: ApplyImportRequest['modules'] = {}
  for (const k of collectionKeys) {
    modules[k] = { policy: policies[k] || 'add_new_only', overrides: overrides[k] || {} }
  }
  return { data: resolvedData(), modules }
})

// 合并设置冲突字段到最终 data.settings
function resolvedData(): BackupData {
  const base = JSON.parse(JSON.stringify(filePayload.value || { settings: { pomodoro: {}, ai: {} } })) as BackupData
  if (!base.settings) base.settings = { pomodoro: {} as any, ai: {} as any }
  for (const c of settingsConflicts.value) {
    const src = settingsChoice[c.section + '.' + c.field] === 'file' ? c.imported : c.current
    ;(base.settings as any)[c.section][c.field] = src
  }
  return base
}

function mask(v: unknown): string {
  return '••••'
}

async function onFileSelected(file: File) {
  previewError.value = ''
  try {
    const text = await file.text()
    const env = JSON.parse(text)
    if (env.app !== 'ticktask') throw new Error('不是有效的 TickTask 备份文件')
    filePayload.value = env.data as BackupData
    const res = await api.previewImport(file)
    preview.value = res.data
    // 默认 settings 选择 = current(保留当前)
    for (const c of preview.value.modules['settings']?.settings_conflicts || []) {
      settingsChoice[c.section + '.' + c.field] = 'current'
    }
    step.value = 'preview'
  } catch (e: any) {
    previewError.value = e?.message || '预览失败'
  }
}
function onFileInputChange(e: Event) {
  const f = (e.target as HTMLInputElement).files?.[0]
  if (f) onFileSelected(f)
}

function setPolicy(key: string, p: ImportPolicy) {
  policies[key] = p
}
function setSettingsChoice(section: string, field: string, choice: 'current' | 'file') {
  settingsChoice[section + '.' + field] = choice
}

const pendingReplace = ref<string | null>(null)
function confirmReplace(ok: boolean) {
  if (!ok) {
    // 取消:把该模块策略回退到 add_new_only
    if (pendingReplace.value) policies[pendingReplace.value] = 'add_new_only'
  }
  pendingReplace.value = null
}

async function clickApply() {
  // 检查是否有 replace 模块需要二次确认
  const replacing = collectionKeys.find(k => policies[k] === 'replace')
  if (replacing && pendingReplace.value === null) {
    pendingReplace.value = replacing
    const ok = window.confirm(`「${moduleLabel[replacing]}」选择了整模块覆盖,将删除当前库中不在备份内的记录,确认?`)
    confirmReplace(ok)
    if (!ok) return
  }
  applying.value = true
  try {
    await api.applyImport(applyPayload.value)
    ElMessage.success('导入成功')
    emit('applied')
    emit('update:modelValue', false)
  } catch {
    ElMessage.error('导入失败,数据未改动')
  } finally {
    applying.value = false
  }
}

defineExpose({ onFileSelected, setPolicy, confirmReplace, clickApply, applyPayload })
</script>

<style scoped>
.module-row { display: flex; justify-content: space-between; align-items: center; padding: 8px 0; border-bottom: 1px solid var(--border-color); }
.module-summary { display: flex; gap: 12px; align-items: center; }
.module-summary span { font-size: 13px; color: var(--text-muted); }
.settings-diff { margin-top: 16px; }
.diff-row { display: flex; gap: 12px; align-items: center; padding: 6px 0; }
.diff-field { width: 160px; font-weight: 500; }
.section-label { font-weight: 600; margin-bottom: 8px; }
.error { color: var(--accent-primary); }
</style>
```

> 注:测试通过 `defineExpose` 暴露的方法驱动;`window.confirm` 在 jsdom 下直接可用(返回 true 触发应用,测试里通过 `confirmReplace(false)` 模拟取消)。若 `el-radio` 的 `value` prop 在当前 Element Plus 版本需要用 `label`,按项目已有用法调整(参考 Settings.vue 里 `el-option` 用 `label`/`value`)。

- [ ] **Step 4: 跑测试 + 类型检查**

Run: `cd frontend && npx vitest run src/components/settings/ImportWizard.test.ts && npx vue-tsc --noEmit`
Expected: 测试 PASS;类型检查通过。

- [ ] **Step 5: 提交**

```bash
git add frontend/src/components/settings/ImportWizard.vue frontend/src/components/settings/ImportWizard.test.ts
git commit -m "feat(data): add ImportWizard (3-step preview → resolve → apply)"
```

---

## Task 10: Settings.vue「数据管理」卡片 + 导出入口

**Files:**
- Modify: `frontend/src/views/Settings.vue`

- [ ] **Step 1: 写失败测试 — 卡片渲染 + 导出/导入按钮**

Create `frontend/src/views/Settings.test.ts`(若已存在则追加):

```ts
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { setActivePinia, createPinia } from 'pinia'
import Settings from './Settings.vue'

vi.mock('@/api/client', () => ({
  api: {
    getSettings: vi.fn().mockResolvedValue({ data: { pomodoro: null, ai: null } }),
    updatePomodoroSettings: vi.fn(),
    updateAISettings: vi.fn(),
    exportData: vi.fn().mockResolvedValue({ data: new Blob(['{}']) })
  }
}))
vi.mock('@/stores/ai', () => ({ useAIStore: () => ({ configured: false, checkStatus: vi.fn() }) }))
vi.mock('element-plus', async () => {
  const actual: any = await vi.importActual('element-plus')
  return { ...actual, ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() } }
})

beforeEach(() => setActivePinia(createPinia()))

describe('Settings data card', () => {
  it('renders data management card with export/import buttons', async () => {
    const w = mount(Settings, { global: { stubs: { ImportWizard: true } } })
    await flushPromises()
    expect(w.text()).toContain('数据管理')
    expect(w.text()).toContain('导出全部数据')
    expect(w.text()).toContain('导入数据')
  })

  it('clicking export calls api.exportData', async () => {
    const { api } = await import('@/api/client')
    const w = mount(Settings, { global: { stubs: { ImportWizard: true } } })
    await flushPromises()
    await w.get('[data-test="export-btn"]').trigger('click')
    await flushPromises()
    expect(api.exportData).toHaveBeenCalled()
  })
})
```

- [ ] **Step 2: 跑测试确认失败**

Run: `cd frontend && npx vitest run src/views/Settings.test.ts`
Expected: FAIL — 无「数据管理」卡片。

- [ ] **Step 3: 在 Settings.vue 加卡片 + 导出下载逻辑 + 接入向导**

在 `Settings.vue` 模板的「关于」卡片**之前**插入新卡片:

```vue
    <!-- 数据管理 -->
    <div class="settings-card">
      <div class="card-header">
        <div class="card-title">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" class="card-icon">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/>
            <polyline points="7 10 12 15 17 10"/>
            <line x1="12" y1="15" x2="12" y2="3"/>
          </svg>
          <span>数据管理</span>
        </div>
      </div>
      <div class="card-content">
        <p class="form-tip">导出全部数据为 JSON 备份;或从备份文件导入(支持冲突人工解决)。</p>
        <div class="card-actions">
          <el-button type="primary" size="large" data-test="export-btn" @click="exportData" :loading="exporting">导出全部数据</el-button>
          <el-button size="large" data-test="import-btn" @click="importVisible = true">导入数据</el-button>
        </div>
        <ImportWizard v-model="importVisible" @applied="onImported" />
      </div>
    </div>
```

在 `<script setup>` 内:
- import:`import ImportWizard from '@/components/settings/ImportWizard.vue'`
- 加状态与方法:

```ts
const exporting = ref(false)
const importVisible = ref(false)

async function exportData() {
  exporting.value = true
  try {
    const res = await api.exportData()
    const url = URL.createObjectURL(res.data)
    const a = document.createElement('a')
    a.href = url
    a.download = `ticktask-backup-${new Date().toISOString().slice(0, 19).replace(/[:T]/g, '-')}.json`
    document.body.appendChild(a)
    a.click()
    document.body.removeChild(a)
    URL.revokeObjectURL(url)
    ElMessage.success('导出成功')
  } catch {
    ElMessage.error('导出失败')
  } finally {
    exporting.value = false
  }
}

async function onImported() {
  await loadSettings()
  await aiStore.checkStatus()
}
```

- [ ] **Step 4: 跑测试 + 类型检查 + 全量前端测试**

Run: `cd frontend && npx vitest run && npx vue-tsc --noEmit`
Expected: 全部 PASS;类型检查通过。

- [ ] **Step 5: 提交**

```bash
git add frontend/src/views/Settings.vue frontend/src/views/Settings.test.ts
git commit -m "feat(data): add Data Management card to Settings (export download + import wizard)"
```

---

## Task 11: Manual verification (M-1 / M-2 / M-3)

**Files:** 无代码改动;记录验证结果到提交说明或 PR 描述。

- [ ] **Step 1: 启动全栈**

```bash
make dev   # 后端 :8080 + 前端 :5173
```

- [ ] **Step 2: M-1 全量往返**

1. 浏览器打开 `http://localhost:5173/settings`,在「数据管理」点「导出全部数据」→ 得 `ticktask-backup-*.json`,记下任务/日志条数。
2. 清库:`rm backend/data/ticktask.db` 后重启后端(会重新 seed 默认设置)。
3. 点「导入数据」→ 选刚才的文件 → 各模块策略选「整模块覆盖」(replace),settings 冲突按需选 → 确认应用。
4. 刷新页面,核对任务/日程/工作日志/报告条数与导出前一致。
- Expected: 数据完全还原。

- [ ] **Step 3: M-2 api_key 还原**

1. 设置页 AI 卡填入一个 api_key 并保存。
2. 导出 → 清库重启 → 导入(settings 的 api_key 选「导入」)→ 设置页 AI 卡应显示已配置、`aiStore.configured === true`。
- Expected: api_key 正确还原。

- [ ] **Step 4: M-3 dangling task_id**

1. 构造一个 session/schedule 引用了某 task_id,导出。
2. 删掉该 task(或在新库只导入 sessions 不导入 tasks,策略:tasks=add_new_only + sessions=replace)。
3. 导入 → 预览应提示「引用了不存在的任务」类告警;应用不报错。
- Expected: 不崩溃,dangling 引用保留,预览有告警。

> 若 M-3 的「引用告警」未在预览实现(本期 preview 未单独列 dangling 引用计数),则降级为:确认导入不报错、session/schedule 记录存在即可,并在 PR 描述标注「dangling 引用告警为后续增强」。

- [ ] **Step 5: 提交验证记录**

```bash
git commit --allow-empty -m "test(data): manual round-trip verification (M-1/M-2/M-3) passed"
```

---

## Self-Review(spec 对账)

**1. Spec 覆盖**

| Spec 要求 | 实现 Task |
|---|---|
| 全量数据导出(7 表,排除 DailyStats) | T2(ReadAll 排除)、T4(Export) |
| JSON 信封 + schema_version | T1(BackupEnvelope)、T4 |
| settings 组装态 `{pomodoro,ai}` | T2(ReadAll 组装)、T1(SettingsBundle) |
| 两阶段导入(预览→应用) | T5(Preview)、T6(Apply)、T7(端点) |
| 冲突四桶 + 字段 diff | T5(classify/fieldDiffs) |
| settings 逐字段 diff(含 api_key) | T5(diffSettings) |
| work_logs 原子(含 items) | T5(classifyWorkLogs)、T3(applyWorkLogs 原子替换) |
| 4 策略 + orphan 删除 | T6(resolveModule)、T3(deleteByIDs) |
| 逐条 override | T6(resolveModule wantFile) |
| 单事务原子性 | T3(Transaction + 回滚测试) |
| FK 写入顺序 | T3(Apply 顺序:tasks→sessions/schedules→work_logs→reports→settings) |
| api_key 导出/掩码/安全 | T1/T5(不掩码后端)、T9(前端 mask)、T10(导出) |
| 错误处理(校验/大小/不回显 key) | T7(readBackupUpload)、T6(策略校验) |
| 端点 `/api/data/*` | T7(router) |
| 前端向导 + 卡片 + api | T8/T9/T10 |
| 测试用例 S/H/F/M | T2/T3(S-1~30,36)、T5(S-7~18)、T6(S-19~30,35)、T7(H-1~7)、T8(F-16~18)、T9(F-1~12)、T10(F-13~15)、T11(M-1~3) |

**未覆盖项(明确标注):**
- S-18(canonical JSON identical 边界):由 `jsonEqual` 隐式覆盖,无独立用例 —— 可接受。
- S-29(FK 顺序独立断言):由 T3 的 `applyWorkLogs` 随 log 写 items 隐式覆盖。
- S-34(>50MB):`readBackupUpload` 用 `LimitReader` 拒绝,无独立 handler 测试 —— 实现已含,可在 T7 补一个超大 body 用例(可选)。

**2. 占位符扫描:** 无 TBD/TODO;`upsertSlice` 泛型为有意决策(已注明);`applyEntity` 占位已在 T6 Step 3 说明删除。

**3. 类型一致性:** `PreviewImport(file, fileVersion)` 签名在 T4/T5/T7 一致;`resolveModule` 返回 `(upsert, del, ModuleApplyResult)` 在 T6 内部一致;策略/选择常量 `model.Policy*/Choice*` 全文档统一;模块 key `tasks/sessions/schedules/work_logs/work_reports` + `settings` 全文档统一。

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-08-07-data-import-export.md`. Two execution options:

**1. Subagent-Driven (recommended)** — 我每个 Task 派一个全新 subagent,任务间做两阶段评审,迭代快、上下文干净。
**2. Inline Execution** — 在当前会话用 executing-plans 批量执行,带检查点评审。

Which approach?
