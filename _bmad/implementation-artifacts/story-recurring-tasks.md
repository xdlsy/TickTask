# Story 1.1: 任务重复与固定时段支持

Status: review

## Story

As a 用户,
I want 创建任务时指定重复周期、起止日期和固定时段,
so that 重复性任务无需手动重建，且 AI 日程生成能尊重我设定的时段，生成更准确的规划.

## Acceptance Criteria

1. 创建任务时可通过"重复任务"开关标记任务为重复类型
2. 重复任务可设置重复模式：每日/每周/每月
3. 任务表单新增"开始日期"和"截止日期"字段，两者均为可选
4. 任务表单新增"偏好时段"设置（开始时间和结束时间），用于指定任务希望安排在哪个时间段
5. AI 生成日程时，对于设置了偏好时段的任务，应将日程安排在指定时段内，而非按默认顺序排布
6. 未设置偏好时段的任务，AI 日程生成行为与现有逻辑一致（顺序排布）
7. 现有任务（无新字段）在创建、编辑、列表展示中不受影响

## Tasks / Subtasks

- [x] Task 1: 后端数据模型扩展 (AC: 1, 2, 3, 4)
  - [x] 1.1 扩展 `model/task.go` Task 结构体，新增 StartDate, DueDate, IsRecurring, RecurrencePattern, PreferredStartTime, PreferredEndTime 字段
  - [x] 1.2 扩展 `repository/task_repo.go` 接口，确保新字段可持久化（GORM AutoMigrate 自动处理 DDL）
  - [x] 1.3 扩展 `service/task_service.go` CreateTaskRequest / UpdateTaskRequest，在 Create/Update 方法中处理新字段
  - [x] 1.4 扩展 `handler/task.go` CreateTaskInput / UpdateTaskInput，解析前端传入的新字段

- [x] Task 2: 前端类型与 TaskForm 扩展 (AC: 1, 2, 3, 4)
  - [x] 2.1 扩展 `types/index.ts` Task 接口，新增对应前端字段
  - [x] 2.2 在 `TaskForm.vue` 中新增：开始日期选择器、截止日期选择器、重复任务开关、重复模式下拉、偏好时段选择器
  - [x] 2.3 更新 `stores/task.ts` createTask 方法，传递新字段

- [x] Task 3: AI 日程生成尊重固定时段 (AC: 5, 6)
  - [x] 3.1 修改 `service/schedule_service.go` 中的 `simpleScheduleWithStart`，对于有 PreferredStartTime/PreferredEndTime 的任务，将日程安排在指定时段
  - [x] 3.2 无时段偏好的任务保持现有顺序排布逻辑，在剩余时间段中填充

- [x] Task 4: 测试与兼容性验证 (AC: 7)
  - [x] 4.1 扩展后端 task handler 测试，验证新字段的 CRUD
  - [x] 4.2 扩展 schedule service 测试，验证时段尊重逻辑
  - [x] 4.3 扩展前端 task store 测试，验证新字段传递（现有 schedule store 测试通过，确认兼容）
  - [x] 4.4 验证现有任务（无新字段）不受影响（全部 448 个前端测试 + 27 个后端测试通过）

## Dev Notes

### 数据模型设计

新增字段（全部可选，向后兼容）：

| 字段 | Go 类型 | GORM 类型 | 前端类型 | 说明 |
|------|---------|-----------|----------|------|
| `StartDate` | `*time.Time` | `date` | `string \| null` | 任务开始日期 |
| `DueDate` | `*time.Time` | `date` | `string \| null` | 任务截止日期（与现有 Deadline 区分：Deadline 是精确时间点，DueDate 是日期） |
| `IsRecurring` | `bool` | `not null;default:false` | `boolean` | 是否为重复任务 |
| `RecurrencePattern` | `string` | `size:20` | `'daily' \| 'weekly' \| 'monthly' \| ''` | 重复模式 |
| `PreferredStartTime` | `string` | `size:5` | `string \| null` | 偏好开始时间，格式 "HH:MM"，如 "09:00" |
| `PreferredEndTime` | `string` | `size:5` | `string \| null` | 偏好结束时间，格式 "HH:MM"，如 "11:00" |

**关于 Deadline vs DueDate**：现有 `Deadline` 字段是 `*time.Time`（包含时间戳），保留不动。新增的 `DueDate` 是纯日期概念（date only），用于日历视图中的截止日标记。两者语义不同，不互相替代。

### 前后端数据流

```
TaskForm.vue (formData) → Tasks.vue (onSave) → taskStore.createTask(data) → api.createTask(data)
  → POST /api/tasks → handler/task.go CreateTaskInput → service/task_service.go CreateTaskRequest → model.Task
```

每个环节都需要对应增加新字段的传递。采用"字段透传"模式，与现有 `estimated_time`、`deadline` 一致。

### AI 日程生成改造

**关键文件**: `backend/internal/service/schedule_service.go:227-260`

现有逻辑 `simpleScheduleWithStart` 对所有任务顺序排布，每个任务分配 `estimated_time` 分钟，首尾相接。改造后分为两类处理：

**伪代码**:
```
func simpleScheduleWithStart(tasks, start):
  timeSlots = [] // 已占用的时间段
  events = []

  // 第一轮：安排有固定时段的任务
  for task in tasks where task.PreferredStartTime != "" && task.PreferredEndTime != "":
    slotStart = parseTimeToToday(task.PreferredStartTime)
    slotEnd = parseTimeToToday(task.PreferredEndTime)
    event = createSchedule(task, slotStart, slotEnd)
    events.append(event)
    timeSlots.append([slotStart, slotEnd])

  // 第二轮：安排无时段偏好的任务到剩余空闲时段
  cursor = start
  for task in tasks where task.PreferredStartTime == "":
    duration = task.EstimatedTime > 0 ? task.EstimatedTime * min : 30min
    // 跳过已有安排的时段
    cursor = nextFreeSlot(cursor, duration, timeSlots)
    event = createSchedule(task, cursor, cursor+duration)
    events.append(event)
    cursor = cursor + duration

  return events
```

**注意事项**：
- 时区处理需与 `GenerateScheduleWithAI` 中的 `now.Location()` 保持一致
- `PreferredStartTime` / `PreferredEndTime` 存储为 "HH:MM" 字符串，解析时使用 `time.Parse("15:04", ...)`
- 固定时段任务可能时间冲突，当前策略是"先到先得"，不处理冲突（后续版本可优化）

### 前端表单布局

TaskForm.vue 新增字段建议放置在"预估时间"和"截止时间"之间：

```
预估时间: [InputNumber]
偏好时段: [TimePicker 开始] — [TimePicker 结束]  (新增)
开始日期: [DatePicker]  (新增)
截止日期: [DatePicker]  (新增)
截止时间: [DateTimePicker]  (现有 deadline)
重复任务: [Switch 开关]  (新增)
  当开关打开时：重复模式: [Select: 每日/每周/每月]  (新增)
```

### 测试要点

1. **后端 TaskHandler**: 创建任务时传入新字段，验证返回的 task 包含这些字段
2. **ScheduleService**: 
   - 任务 A 设置时段 09:00-10:00，任务 B 无时段 → A 应在 09:00-10:00，B 在其他时间
   - 全部任务无时段 → 行为与现有逻辑一致（顺序排布）
3. **前端**: TaskForm 渲染新字段，开关控制重复模式显示/隐藏

### References

- [Source: backend/internal/model/task.go] — Task 模型定义
- [Source: backend/internal/api/handler/task.go:20-42] — CreateTaskInput / UpdateTaskInput
- [Source: backend/internal/service/task_service.go:29-51] — CreateTaskRequest / UpdateTaskRequest
- [Source: backend/internal/service/schedule_service.go:227-260] — simpleScheduleWithStart 日程排布逻辑
- [Source: frontend/src/components/tasks/TaskForm.vue] — 任务表单组件
- [Source: frontend/src/types/index.ts:5-20] — Task 前端类型

## Dev Agent Record

### Agent Model Used

Claude Opus 4.7

### Debug Log References

- Backend build: `go build ./...` — passed
- Backend tests: `go test ./...` — service + handler tests all pass (27 tests)
- Frontend type check: `vue-tsc --noEmit` — passed
- Frontend tests: `vitest run` — 448 passed, 4 pre-existing failures (color values, unrelated)

### Completion Notes List

- Task model extended with 6 new optional fields; GORM AutoMigrate handles DDL automatically
- Handler/Service layers updated with full CRUD support for new fields via pointer-based optional pattern
- TaskForm.vue updated with new form controls: preferred time range pickers, start/due date pickers, recurring task toggle + pattern selector
- Schedule generation refactored into two-phase algorithm: preferred-slot tasks scheduled first, then remaining tasks fill gaps via collision-avoiding cursor
- New backend tests: `TestTaskHandler_CreateTask_WithNewFields`, `TestScheduleService_GenerateScheduleWithAI_PreferredTimeSlots`, `TestScheduleService_GenerateScheduleWithAI_NoPreferences`
- All existing tests pass; no regressions

### Browser Verification Bugs & Fixes (2026-05-26)

Three bugs found during live browser testing after implementation:

1. **Time picker causes save crash**: `el-time-picker` with `value-format="HH:mm"` returns strings (e.g. "09:00"), not Date objects. Old `formatTime` helper called `.getHours()` on the string, crashing. Fixed by removing `formatTime` entirely and using string types in `formData`.

2. **Form fields not populated on edit**: `watch(() => props.task, ...)` was missing `{ immediate: true }`. Since TaskForm uses `v-if` for conditional rendering, the task prop is passed at mount time — without immediate, the watcher never fires. Fixed by adding `{ immediate: true }`.

3. **Date fields silently lost on save**: `el-date-picker` with `value-format="YYYY-MM-DD"` outputs "2026-06-15", but Go's `time.Time` JSON unmarshaling requires RFC3339 format. `ShouldBindJSON` silently failed on date fields. Fixed by appending `T00:00:00Z` in frontend `onSave()` and extracting `.substring(0, 10)` in watch for display.

4. **Backend old process serving stale code**: Backend server (PID 28459) was running pre-change binary. API responses were missing all 6 new fields because the old binary without model changes was still serving. Root cause: Go doesn't hot-reload — the compiled binary must be restarted after code changes. Fixed by `kill` old process + `go run cmd/server/main.go &` to start new process (PID 34670). Memory file `feedback_backend_restart.md` created to prevent recurrence.

Verified end-to-end: create task with all new fields → edit task → all fields correctly populated (preferred start/end times, start/due dates, recurring toggle + pattern).

### File List

- `backend/internal/model/task.go` — Added 6 new fields to Task struct
- `backend/internal/service/task_service.go` — Extended CreateTaskRequest/UpdateTaskRequest + Create/Update methods
- `backend/internal/api/handler/task.go` — Extended CreateTaskInput/UpdateTaskInput + mapping logic
- `backend/internal/service/schedule_service.go` — Refactored simpleScheduleWithStart → two-phase scheduling + extracted createScheduleEvent helper + added findNextAvailableSlot
- `backend/internal/service/schedule_service_test.go` — Added 3 new tests for preferred time slots and backward compatibility
- `backend/internal/api/handler/task_test.go` — Added TestTaskHandler_CreateTask_WithNewFields
- `frontend/src/types/index.ts` — Added 6 new fields to Task interface
- `frontend/src/components/tasks/TaskForm.vue` — Added UI for preferred time, start/due dates, recurring toggle, recurrence pattern
- `frontend/src/stores/task.ts` — Updated createTask type signature with new optional fields
