# Clear All Data — Design Spec

- **Date:** 2026-08-08
- **Branch:** `evolve/data-import-export`
- **Status:** Approved (pending implementation)
- **Approach:** A — dedicated transactional endpoint

## Goal

Add a **“清空全部数据” (Clear All Data)** action to the **Settings → 数据管理** card. It wipes all user/business data while **preserving configuration** (Pomodoro settings, AI settings, and the AI API key), so the app returns to an empty-but-configured state without re-setup.

## Scope

**Cleared (user data):**
- `Task`, `PomodoroSession`, `Schedule`, `WorkLog`, `WorkItem`, `WorkReport`
- `DailyStats` (derived analytics aggregates — cleared for consistency)

**Preserved (configuration):**
- `Setting` table rows: `pomodoro.settings`, `ai.settings` (includes the AI API key)

**Out of scope:** schema changes, table drops, granular/per-module clearing, server restart.

## Decisions (from brainstorming)

1. **Scope** — data only; configuration is kept (user confirmed: “只清空数据，不清空配置”).
2. **Safety net** — before each clear, ask whether to back up first (user confirmed: “每次询问是否备份”). Choosing backup downloads a full JSON export, then clearing proceeds.
3. **Final gate** — type-to-confirm (`清空`) before the irreversible wipe.
4. **Post-clear reset** — `location.reload()` to reset all Pinia stores and the WebSocket cleanly.
5. **Atomicity** — a single DB transaction; all-or-nothing.

## Backend

### Endpoint
`DELETE /api/data/all` → `200` with body:
```json
{ "tasks": 12, "sessions": 48, "schedules": 3, "work_logs": 5, "work_reports": 1, "daily_stats": 7 }
```

### Repository (`internal/repository/data_repo.go`)
Add to `BackupRepository`:
```go
ClearAll() (*model.ClearResult, error)
```
Implementation — one `db.Transaction`:
- For each user-data model (`Task`, `PomodoroSession`, `Schedule`, `WorkItem`, `WorkLog`, `WorkReport`, `DailyStats`): count rows, then `tx.Where("1=1").Delete(&model.X{})`.
- **Do not touch `Setting`.**
- Return `ClearResult` populated with the per-table counts.
- On any error, the transaction rolls back (atomic).

Delete order is irrelevant for full-table clears, but children (`WorkItem`) are listed before parents (`WorkLog`) for cleanliness.

### Model (`internal/model/backup.go`)
```go
type ClearResult struct {
    Tasks       int64 `json:"tasks"`
    Sessions    int64 `json:"sessions"`
    Schedules   int64 `json:"schedules"`
    WorkLogs    int64 `json:"work_logs"`
    WorkReports int64 `json:"work_reports"`
    DailyStats  int64 `json:"daily_stats"`
}
```

### Service (`internal/service/data_service.go`)
Add to `DataService`:
```go
ClearAll() (*model.ClearResult, error)
```
Thin pass-through to `repo.ClearAll()`.

### Handler (`internal/api/handler/data.go`)
```go
func (h *DataHandler) ClearAll(c *gin.Context) {
    res, err := h.svc.ClearAll()
    if err != nil {
        c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
        return
    }
    c.JSON(http.StatusOK, res)
}
```

### Router (`internal/api/router.go`)
Inside the `/data` group:
```go
data.DELETE("/all", dataHandler.ClearAll)
```

## Frontend

### Types (`src/types/index.ts`)
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

### API client (`src/api/client.ts`)
```ts
clearAll: () => client.delete<ClearResult>('/data/all'),
```

### Settings.vue — 数据管理 card
Add a danger zone below the existing export/import actions:
- A divider/subtle separator.
- A **danger-styled `el-button`** “清空全部数据” (`data-test="clear-btn"`, `type="danger"`, `:loading="clearing"`).

### Click flow — `clearAllData()`
1. **Ask backup** — `ElMessageBox.confirm`:
   - Message warns the action is irreversible and will keep configuration.
   - Buttons: `[先备份再清空]` (confirm) / `[直接清空]` (alt) / `[取消]` (cancel).
   - “先备份再清空” → `await exportData()` (existing download), then continue.
   - “直接清空” → continue.
   - “取消” → abort, no-op.
2. **Final gate** — `ElMessageBox.prompt`: requires the user to type exactly **`清空`** (`inputValidator: v => v === '清空' || '请输入「清空」以确认'`).
3. **Execute** — `await api.clearAll()`; set `clearing=true` during the call.
   - Success → `ElMessage.success` with a short summary (e.g. deleted counts), then `location.reload()`.
   - Error → `ElMessage.error`, stay on page (no reload).

## Edge Cases & Error Handling

- **Active timer session** — clearing deletes a running `PomodoroSession`; the backend ticker goroutine’s next update affects 0 rows (no-op). After `location.reload()`, the frontend fetches no active session → clean state. The UI warning text notes that clearing implicitly abandons any in-progress pomodoro.
- **Mid-transaction failure** — atomic rollback; handler returns `500`; frontend shows an error and does **not** reload. DB is unchanged.
- **Empty database** — clearing returns `0` counts, still `200`.
- **Concurrent import** — single-user app; the transaction isolates. Not a concern.
- **WebSocket** — reconnects fresh on reload; no stale timer state surfaces.

## Testing

### Backend
- **Service/repo test** (`data_service_test.go` or repo-level): seed tasks/sessions/settings; call `ClearAll`; assert all user tables are empty, **`Setting` rows are retained** (pomodoro + ai settings still present), and returned counts match seeded totals.
- **Handler test** (`data_test.go`): `DELETE /api/data/all` returns `200` with counts (mock the service).

### Frontend (`Settings.test.ts`)
- The clear button renders with `data-test="clear-btn"`.
- Clicking it drives the flow: mock `ElMessageBox` (confirm + prompt) to auto-proceed, mock `api.clearAll` → assert it is called; mock `location.reload` to a no-op spy.

## Files Touched

**Backend (new/edited):**
- `internal/repository/data_repo.go` — `ClearAll()` + interface
- `internal/model/backup.go` — `ClearResult`
- `internal/service/data_service.go` — `ClearAll()` + interface
- `internal/api/handler/data.go` — `ClearAll` handler
- `internal/api/router.go` — route
- `internal/repository/data_repo_test.go` / `internal/service/data_service_test.go` / `internal/api/handler/data_test.go` — tests (match existing co-located patterns)

**Frontend (new/edited):**
- `src/types/index.ts` — `ClearResult`
- `src/api/client.ts` — `clearAll()`
- `src/views/Settings.vue` — danger-zone button + `clearAllData()` flow
- `src/views/Settings.test.ts` — clear-button test
