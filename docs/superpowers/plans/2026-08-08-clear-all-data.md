# Clear All Data Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a “清空全部数据” action to Settings → 数据管理 that wipes all user data in one atomic transaction while preserving the `Setting` table (Pomodoro + AI config, including the API key).

**Architecture:** A dedicated `ClearAll()` walks the existing `BackupRepository`, deleting every user-data table (and `DailyStats`) in a single GORM transaction, leaving `Setting` untouched. It surfaces through the `DataService` and a `DELETE /api/data/all` handler. The frontend adds a danger-styled button whose click flow is: ask-backup → type `清空` → call API → reload.

**Tech Stack:** Go 1.21 / Gin / GORM / SQLite (backend) · Vue 3 + TypeScript + Element Plus / Vitest (frontend)

---

## File Structure

**Backend (Go):**
- `internal/model/backup.go` — add `ClearResult` struct (the per-table deleted counts).
- `internal/repository/data_repo.go` — add `ClearAll()` to the `BackupRepository` interface + implementation + a `clearTable` helper.
- `internal/repository/data_repo_test.go` — repo-level test against in-memory SQLite (real clearing logic lives here).
- `internal/service/data_service.go` — add `ClearAll()` to the `DataService` interface + passthrough implementation.
- `internal/service/data_service_test.go` — extend `mockBackupRepo` with `ClearAll`; service passthrough test.
- `internal/api/handler/data.go` — add `ClearAll` handler.
- `internal/api/handler/data_test.go` — extend `mockDataService` with `ClearAll`; handler test.
- `internal/api/router.go` — register `data.DELETE("/all", …)`.

**Frontend (Vue/TS):**
- `src/types/index.ts` — add `ClearResult` interface.
- `src/api/client.ts` — add `clearAll()` method.
- `src/views/Settings.vue` — danger-zone button + `clearAllData()` flow + `clearing` ref + `ElMessageBox` import + `.clear-zone` styles.
- `src/views/Settings.test.ts` — add `clearAll` to the api mock + `prompt` to the ElMessageBox mock + a clear-button test.

**Interface rule (Go):** Growing an interface requires updating every implementor in the same change or the package won't compile. Each task below grows exactly one interface and updates its implementors (the real struct + the test mock) together.

---

## Task 1: Repository `ClearAll` + `ClearResult` + repo test

**Files:**
- Modify: `backend/internal/model/backup.go` (append `ClearResult`)
- Modify: `backend/internal/repository/data_repo.go` (interface + impl + helper)
- Modify: `backend/internal/service/data_service_test.go` (add `ClearAll` to `mockBackupRepo` so the service test package still compiles)
- Test: `backend/internal/repository/data_repo_test.go`

- [ ] **Step 1: Write the failing repo test**

Append to `backend/internal/repository/data_repo_test.go` (add `"time"` to the import block):

```go
func TestBackupRepo_ClearAll_KeepsSettings(t *testing.T) {
	db := newDataTestDB(t)
	// Seed one row in each user-data table + two setting rows.
	db.Create(&model.Task{ID: "t1", Title: "x", Status: model.StatusTodo, Quadrant: model.Quadrant1})
	db.Create(&model.PomodoroSession{ID: "s1", Type: model.SessionWork, PlannedDuration: 1500})
	db.Create(&model.Schedule{ID: "sch1", Title: "s", StartTime: time.Now(), EndTime: time.Now()})
	db.Create(&model.WorkLog{ID: "wl1", Date: "2026-08-07"})
	db.Create(&model.WorkItem{ID: "wi1", WorkLogID: "wl1", Seq: 1, Source: "ai"})
	db.Create(&model.WorkReport{ID: "wr1", Type: "weekly", PeriodKey: "2026-W31", StartDate: "2026-08-01", EndDate: "2026-08-07"})
	db.Create(&model.DailyStats{})
	db.Create(&model.Setting{Key: "pomodoro.settings", Value: "{}"})
	db.Create(&model.Setting{Key: "ai.settings", Value: "{}"})

	repo := NewDataRepository(db)
	res, err := repo.ClearAll()
	if err != nil {
		t.Fatalf("clearall: %v", err)
	}
	if res.Tasks != 1 || res.Sessions != 1 || res.Schedules != 1 ||
		res.WorkLogs != 1 || res.WorkReports != 1 || res.DailyStats != 1 {
		t.Errorf("counts wrong: %+v", res)
	}

	// Every user-data table is now empty.
	for _, dest := range []any{
		&model.Task{}, &model.PomodoroSession{}, &model.Schedule{},
		&model.WorkLog{}, &model.WorkItem{}, &model.WorkReport{}, &model.DailyStats{},
	} {
		var n int64
		db.Model(dest).Count(&n)
		if n != 0 {
			t.Errorf("%T not cleared: %d rows remain", dest, n)
		}
	}

	// Settings are RETAINED.
	var settingCount int64
	db.Model(&model.Setting{}).Count(&settingCount)
	if settingCount != 2 {
		t.Errorf("settings should be retained (2 rows), got %d", settingCount)
	}
}
```

- [ ] **Step 2: Run test to verify it fails (compile error)**

Run: `cd backend && go test ./internal/repository/ -run TestBackupRepo_ClearAll -v`
Expected: compile error — `repo.ClearAll undefined (type BackupRepository has no field or method ClearAll)`.

- [ ] **Step 3: Add `ClearResult` to the model**

Append to `backend/internal/model/backup.go`:

```go
// ClearResult 各用户数据表的清除计数(Setting 不在内,配置保留)。
type ClearResult struct {
	Tasks       int64 `json:"tasks"`
	Sessions    int64 `json:"sessions"`
	Schedules   int64 `json:"schedules"`
	WorkLogs    int64 `json:"work_logs"`
	WorkReports int64 `json:"work_reports"`
	DailyStats  int64 `json:"daily_stats"`
}
```

- [ ] **Step 4: Grow the `BackupRepository` interface + implement `ClearAll`**

In `backend/internal/repository/data_repo.go`, add `ClearAll() (*model.ClearResult, error)` to the interface:

```go
type BackupRepository interface {
	ReadAll() (*model.BackupData, error)
	Apply(plan model.ApplyPlan) error
	ClearAll() (*model.ClearResult, error)
}
```

Add the implementation + helper (anywhere in the file after `writeSetting`):

```go
// ClearAll 单事务清空全部用户数据;Setting 表保留(配置不丢)。
func (r *dataRepository) ClearAll() (*model.ClearResult, error) {
	res := &model.ClearResult{}
	err := r.db.Transaction(func(tx *gorm.DB) error {
		var err error
		// 子表先删:WorkItem 属于 WorkLog(不计入 ClearResult)。
		if _, err = clearTable(tx, &model.WorkItem{}); err != nil {
			return err
		}
		if res.WorkLogs, err = clearTable(tx, &model.WorkLog{}); err != nil {
			return err
		}
		if res.WorkReports, err = clearTable(tx, &model.WorkReport{}); err != nil {
			return err
		}
		if res.Tasks, err = clearTable(tx, &model.Task{}); err != nil {
			return err
		}
		if res.Sessions, err = clearTable(tx, &model.PomodoroSession{}); err != nil {
			return err
		}
		if res.Schedules, err = clearTable(tx, &model.Schedule{}); err != nil {
			return err
		}
		if res.DailyStats, err = clearTable(tx, &model.DailyStats{}); err != nil {
			return err
		}
		// Setting 故意不清。
		return nil
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// clearTable 先计数再删全表,返回删除行数。
func clearTable(tx *gorm.DB, dest any) (int64, error) {
	var n int64
	if err := tx.Model(dest).Count(&n).Error; err != nil {
		return 0, err
	}
	if err := tx.Where("1 = 1").Delete(dest).Error; err != nil {
		return 0, err
	}
	return n, nil
}
```

- [ ] **Step 5: Add `ClearAll` to `mockBackupRepo` (service test package)**

In `backend/internal/service/data_service_test.go`, extend the struct + add the method so the service package still compiles:

```go
type mockBackupRepo struct {
	snapshot    *model.BackupData
	lastPlan    *model.ApplyPlan
	applyErr    error
	clearResult *model.ClearResult
	clearErr    error
}

func (m *mockBackupRepo) ClearAll() (*model.ClearResult, error) {
	return m.clearResult, m.clearErr
}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `cd backend && go test ./internal/repository/ ./internal/service/ -run 'ClearAll|Data' -v`
Expected: PASS (the new repo test passes; existing data tests still pass).

- [ ] **Step 7: Commit**

```bash
cd backend && git add internal/model/backup.go internal/repository/data_repo.go internal/repository/data_repo_test.go internal/service/data_service_test.go
git commit -m "feat(data): add ClearAll to backup repository"
```

---

## Task 2: Service `ClearAll` passthrough + service test

**Files:**
- Modify: `backend/internal/service/data_service.go` (interface + impl)
- Modify: `backend/internal/api/handler/data_test.go` (add `ClearAll` to `mockDataService` so the handler test package still compiles)
- Test: `backend/internal/service/data_service_test.go`

- [ ] **Step 1: Write the failing service test**

Append to `backend/internal/service/data_service_test.go`:

```go
func TestDataService_ClearAll_Passthrough(t *testing.T) {
	want := &model.ClearResult{Tasks: 3, Sessions: 5, WorkLogs: 1}
	svc := NewDataService(&mockBackupRepo{clearResult: want})
	got, err := svc.ClearAll()
	if err != nil {
		t.Fatalf("clearall: %v", err)
	}
	if got != want {
		t.Errorf("ClearAll should pass the repo result through: got %+v, want %+v", got, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails (compile error)**

Run: `cd backend && go test ./internal/service/ -run TestDataService_ClearAll -v`
Expected: compile error — `svc.ClearAll undefined`.

- [ ] **Step 3: Grow the `DataService` interface + implement passthrough**

In `backend/internal/service/data_service.go`, add `ClearAll() (*model.ClearResult, error)` to the interface:

```go
type DataService interface {
	Export(includeAPIKey bool) (*model.BackupEnvelope, error)
	PreviewImport(file *model.BackupData, fileVersion int) (*model.ImportPreview, error)
	ApplyImport(req *model.ApplyImportRequest) (*model.ApplyResult, error)
	ClearAll() (*model.ClearResult, error)
}
```

Add the implementation (after `ApplyImport`):

```go
func (s *dataService) ClearAll() (*model.ClearResult, error) {
	return s.repo.ClearAll()
}
```

- [ ] **Step 4: Add `ClearAll` to `mockDataService` (handler test package)**

In `backend/internal/api/handler/data_test.go`, extend the struct + add the method:

```go
type mockDataService struct {
	exportEnvelop   *model.BackupEnvelope
	exportErr       error
	previewResult   *model.ImportPreview
	previewErr      error
	applyResult     *model.ApplyResult
	applyErr        error
	clearResult     *model.ClearResult
	clearErr        error
	lastFileVersion int
	lastIncludeKey  bool
}

func (m *mockDataService) ClearAll() (*model.ClearResult, error) {
	return m.clearResult, m.clearErr
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd backend && go test ./internal/service/ ./internal/api/handler/ -run 'ClearAll|Data' -v`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
cd backend && git add internal/service/data_service.go internal/service/data_service_test.go internal/api/handler/data_test.go
git commit -m "feat(data): add ClearAll to data service"
```

---

## Task 3: Handler `ClearAll` + `DELETE /api/data/all` route + handler test

**Files:**
- Modify: `backend/internal/api/handler/data.go` (handler)
- Modify: `backend/internal/api/router.go` (route)
- Test: `backend/internal/api/handler/data_test.go`

- [ ] **Step 1: Write the failing handler test**

Append to `backend/internal/api/handler/data_test.go`:

```go
func TestDataHandler_ClearAll(t *testing.T) {
	h := NewDataHandler(&mockDataService{clearResult: &model.ClearResult{Tasks: 2, Sessions: 1}})
	r := setupTestRouter()
	r.DELETE("/api/data/all", h.ClearAll)

	req, _ := http.NewRequest("DELETE", "/api/data/all", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status %d body %s", w.Code, w.Body.String())
	}
	var res model.ClearResult
	if err := json.Unmarshal(w.Body.Bytes(), &res); err != nil {
		t.Fatalf("unmarshal: %v body %s", err, w.Body.String())
	}
	if res.Tasks != 2 || res.Sessions != 1 {
		t.Errorf("counts not returned: %+v", res)
	}
}

func TestDataHandler_ClearAll_ServiceError(t *testing.T) {
	h := NewDataHandler(&mockDataService{clearErr: errBoom})
	r := setupTestRouter()
	r.DELETE("/api/data/all", h.ClearAll)

	req, _ := http.NewRequest("DELETE", "/api/data/all", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("expected 500, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails (compile error)**

Run: `cd backend && go test ./internal/api/handler/ -run TestDataHandler_ClearAll -v`
Expected: compile error — `h.ClearAll undefined`.

- [ ] **Step 3: Add the handler**

Append to `backend/internal/api/handler/data.go`:

```go
// ClearAll DELETE /api/data/all → 清空全部用户数据(保留配置)
func (h *DataHandler) ClearAll(c *gin.Context) {
	res, err := h.svc.ClearAll()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, res)
}
```

- [ ] **Step 4: Register the route**

In `backend/internal/api/router.go`, inside the `/data` group (after the `import/apply` line):

```go
			data.DELETE("/all", dataHandler.ClearAll)
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `cd backend && go test ./... `
Expected: PASS (all backend tests).

- [ ] **Step 6: Commit**

```bash
cd backend && git add internal/api/handler/data.go internal/api/handler/data_test.go internal/api/router.go
git commit -m "feat(data): add DELETE /api/data/all endpoint"
```

---

## Task 4: Frontend `ClearResult` type + `clearAll` API client

**Files:**
- Modify: `frontend/src/types/index.ts`
- Modify: `frontend/src/api/client.ts`

- [ ] **Step 1: Add the `ClearResult` type**

Append to `frontend/src/types/index.ts`:

```ts
export interface ClearResult {
  tasks: number
  sessions: number
  schedules: number
  work_logs: number
  work_reports: number
  daily_stats: number
}
```

- [ ] **Step 2: Add `clearAll` to the API client**

In `frontend/src/api/client.ts`:

Add `ClearResult` to the type import on line 1 (append `ClearResult` inside the existing `import type { ... } from '@/types'`).

In the `api` object, inside the data section (after `applyImport`), add:

```ts
  clearAll: () => client.delete<ClearResult>('/data/all'),
```

- [ ] **Step 3: Verify the type check passes**

Run: `cd frontend && npx vue-tsc --noEmit`
Expected: no errors.

- [ ] **Step 4: Commit**

```bash
cd frontend && git add src/types/index.ts src/api/client.ts
git commit -m "feat(data): add ClearResult type and clearAll api client"
```

---

## Task 5: Frontend Settings — clear-all button + flow + test

**Files:**
- Modify: `frontend/src/views/Settings.vue` (template + script + style)
- Test: `frontend/src/views/Settings.test.ts`

- [ ] **Step 1: Write the failing test**

In `frontend/src/views/Settings.test.ts`:

1. Update the imports (line 2) to include `flushPromises`:
```ts
import { mount, flushPromises } from '@vue/test-utils'
```

2. Add `clearAll` to the mocked `api` (in the `vi.mock('@/api/client', ...)` block):
```ts
vi.mock('@/api/client', () => ({
  api: {
    getSettings: vi.fn(),
    updatePomodoroSettings: vi.fn(),
    updateAISettings: vi.fn(),
    previewImport: vi.fn(),
    applyImport: vi.fn(),
    clearAll: vi.fn()
  }
}))
```

3. Add `prompt` to the mocked `element-plus`:
```ts
vi.mock('element-plus', () => ({
  ElMessage: { success: vi.fn(), error: vi.fn(), warning: vi.fn() },
  ElMessageBox: { confirm: vi.fn(), prompt: vi.fn() }
}))
```

4. Add the test (inside the top-level `describe('Settings.vue', …)` block):
```ts
  describe('Clear All Data', () => {
    it('calls clearAll after backup-confirm and type-confirm', async () => {
      vi.useFakeTimers()
      const { api } = await import('@/api/client')
      const { ElMessage, ElMessageBox } = await import('element-plus')
      ;(api.getSettings as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { pomodoro: mockPomodoroSettings, ai: mockAISettings }
      })
      ;(api.clearAll as ReturnType<typeof vi.fn>).mockResolvedValue({
        data: { tasks: 1, sessions: 0, schedules: 0, work_logs: 0, work_reports: 0, daily_stats: 0 }
      })
      ;(ElMessageBox.confirm as ReturnType<typeof vi.fn>).mockResolvedValue('confirm')
      ;(ElMessageBox.prompt as ReturnType<typeof vi.fn>).mockResolvedValue({ value: '清空' })

      const wrapper = mount(Settings, { global: { stubs: { ...elStubs, ImportWizard: true } } })
      await flushPromises()

      await wrapper.find('[data-test="clear-btn"]').trigger('click')
      // advance the async chain: confirm → (export) → prompt → api.clearAll
      await flushPromises()
      await flushPromises()
      await flushPromises()

      expect(api.clearAll).toHaveBeenCalled()
      expect(ElMessage.success).toHaveBeenCalled()
      vi.useRealTimers()
    })
  })
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd frontend && npx vitest run src/views/Settings.test.ts`
Expected: FAIL — `Unable to find [data-test="clear-btn"]` (button does not exist yet).

- [ ] **Step 3: Add the danger-zone markup**

In `frontend/src/views/Settings.vue`, inside the 数据管理 card's `.card-content` (right after the `<ImportWizard … />` line, still inside the card):

```html
        <ImportWizard v-model="importVisible" @applied="onImported" />

        <div class="clear-zone">
          <div class="clear-text">
            <span class="clear-title">清空全部数据</span>
            <span class="clear-desc">删除所有任务、番茄记录、日程与工作日志(配置与 AI Key 保留)。操作不可恢复。</span>
          </div>
          <el-button type="danger" size="large" data-test="clear-btn" :loading="clearing" @click="clearAllData">清空全部数据</el-button>
        </div>
```

- [ ] **Step 4: Add the `clearAllData` flow + `clearing` ref + `ElMessageBox` import**

In the `<script setup>`:

1. Update the element-plus import (line 294) to include `ElMessageBox`:
```ts
import { ElMessage, ElMessageBox } from 'element-plus'
```

2. Add the `ClearResult` type to the existing type import (line 298):
```ts
import type { PomodoroSettings, AISettings, ClearResult } from '@/types'
```

3. Add the `clearing` ref next to `exporting` (line 451-452):
```ts
const exporting = ref(false)
const importVisible = ref(false)
const clearing = ref(false)
```

4. Add the `clearAllData` function (right after `exportData`, before `onImported`):
```ts
async function clearAllData() {
  // 1. 是否先备份:confirm = 先备份再清空;cancel = 直接清空;close(X/Esc) = 取消
  let backup = false
  try {
    await ElMessageBox.confirm(
      '此操作将清空所有任务、番茄记录、日程与工作日志,且不可恢复。配置与 AI Key 会保留。',
      '清空全部数据',
      {
        confirmButtonText: '先备份再清空',
        cancelButtonText: '直接清空',
        distinguishCancelAndClose: true,
        type: 'warning'
      }
    )
    backup = true
  } catch (action) {
    if (action === 'cancel') {
      backup = false
    } else {
      return // close → 用户放弃
    }
  }

  if (backup) {
    exportData() // 触发与「导出全部数据」相同的下载
  }

  // 2. 最终确认:输入「清空」
  try {
    await ElMessageBox.prompt('请输入「清空」以确认。', '最终确认', {
      confirmButtonText: '清空全部数据',
      cancelButtonText: '取消',
      inputPlaceholder: '清空',
      inputValidator: (v: string) => v === '清空' || '请输入「清空」以确认',
      type: 'error'
    })
  } catch {
    return
  }

  // 3. 执行
  clearing.value = true
  try {
    const { data } = await api.clearAll()
    const total = data.tasks + data.sessions + data.schedules + data.work_logs + data.work_reports + data.daily_stats
    ElMessage.success(`已清空 ${total} 条记录`)
    setTimeout(() => location.reload(), 600)
  } catch {
    ElMessage.error('清空失败,请重试')
  } finally {
    clearing.value = false
  }
}
```

- [ ] **Step 5: Add the `.clear-zone` styles**

In the `<style scoped>` block of `frontend/src/views/Settings.vue`, add (e.g. right after the `.card-actions .el-button { … }` rule):

```css
.clear-zone {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 18px;
  margin-top: 22px;
  padding-top: 20px;
  border-top: 1px solid var(--border-color);
}

.clear-text {
  display: flex;
  flex-direction: column;
  gap: 5px;
}

.clear-title {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 40;
  font-size: 15px;
  font-weight: 440;
  color: var(--accent-crimson);
  letter-spacing: -0.01em;
}

.clear-desc {
  font-size: 12.5px;
  color: var(--text-muted);
  line-height: 1.5;
}

.clear-zone .el-button {
  flex-shrink: 0;
}
```

- [ ] **Step 6: Run the test to verify it passes**

Run: `cd frontend && npx vitest run src/views/Settings.test.ts`
Expected: PASS (the new Clear All Data test passes; existing Settings tests still pass).

- [ ] **Step 7: Run the full frontend check**

Run: `cd frontend && npx vue-tsc --noEmit && npx vitest run`
Expected: vue-tsc clean; vitest at the documented 11-failure baseline (no new failures).

- [ ] **Step 8: Commit**

```bash
cd frontend && git add src/views/Settings.vue src/views/Settings.test.ts
git commit -m "feat(data): add clear-all-data action to Settings"
```

---

## Task 6: End-to-end manual verification

The backend changed (Go), so the running binary will NOT pick it up — it must be restarted.

- [ ] **Step 1: Restart the backend**

```bash
lsof -ti:8080 | xargs kill -9 2>/dev/null
cd backend && go run cmd/server/main.go &
```

(Or run `make dev` from the repo root.)

- [ ] **Step 2: Verify the endpoint directly**

With some data present:
```bash
curl -s -X DELETE http://localhost:8080/api/data/all
```
Expected: `{"tasks":N,"sessions":N,"schedules":N,"work_logs":N,"work_reports":N,"daily_stats":N}` and HTTP 200. A follow-up `curl http://localhost:8080/api/data/export` should return empty arrays but still contain `pomodoro` + `ai` settings (config retained).

- [ ] **Step 3: Verify the UI flow**

1. Open the app (`http://localhost:5173`), go to **设置 → 数据管理**.
2. Click **清空全部数据** → choose 先备份再清空 → a `ticktask-backup-*.json` downloads → the type-`清空` prompt appears → type `清空` → confirm.
3. Confirm: success toast with the count, then the page reloads; tasks/timer/schedule/work-log are now empty; **Settings (Pomodoro + AI key) are unchanged**.
4. Re-test the 取消 path (X on the first dialog aborts) and the 直接清空 path.

---

## Self-Review

**Spec coverage:**
- Scope (clear user data, keep Setting) → Task 1 repo test asserts both. ✓
- `ClearResult` model + counts → Task 1. ✓
- Service passthrough → Task 2. ✓
- `DELETE /api/data/all` handler + route → Task 3. ✓
- Frontend type + client → Task 4. ✓
- Danger button + ask-backup + type-`清空` + reload → Task 5 (`clearAllData`). ✓
- Edge cases (atomic transaction, empty DB returns 0, settings retained) → Task 1 test; active-timer note covered by reload. ✓
- Tests (repo/service/handler/frontend) → Tasks 1/2/3/5. ✓

**Placeholder scan:** None — every code step contains complete code; every test step contains complete test code; commands include expected output.

**Type consistency:** `ClearAll() (*model.ClearResult, error)` consistent across interface (repo + service), impl, mock, handler. `ClearResult` field names (`tasks, sessions, schedules, work_logs, work_reports, daily_stats`) match between Go struct JSON tags and the TS `ClearResult` interface. `api.clearAll()` returns `client.delete<ClearResult>`, destructured as `{ data }` in `clearAllData`. Mock returns `{ data: {…} }` to match. ✓
