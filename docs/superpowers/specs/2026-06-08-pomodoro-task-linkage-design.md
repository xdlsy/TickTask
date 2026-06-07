# Pomodoro-Task Linkage Design

**Date**: 2026-06-08
**Status**: Draft
**Branch**: `evolve/pomodoro-task-linkage`

## Summary

Link Pomodoro sessions to tasks with planned/completed tracking, completion reminders, and analytics. Users can see how many pomodoros a task needs, start pomodoros from task/schedule views, get reminded when pomodoros are used up, and review pomodoro statistics per task.

## Motivation

Currently, Pomodoro sessions can optionally reference a task via `task_id`, but there is no concept of "how many pomodoros does this task need" or "how many are done." Users must manually track progress. This feature makes the Pomodoro technique a first-class part of task management.

## Design Decisions

| Decision | Choice | Rationale |
|----------|--------|-----------|
| Pomodoro count calculation | `ceil(EstimatedTime / WorkDuration)` | Upward rounding ensures sufficient buffer |
| Estimation source | Reuse existing `EstimatedTime` field | No database migration, single source of truth |
| Completion reminder | Element Plus dialog popup | Explicit prompt at the critical moment |
| Extension method | Add 1 pomodoro per click | Simple and predictable |
| Architecture | Lightweight computed layer (Option A) | Zero migration, always-consistent data |
| UI style | No decorative emoji/icons in titles | Refined minimalism aesthetic |

## Data Layer

### API Response Extensions

No model changes. Task API responses gain computed fields:

```
TaskResponse:
  planned_pomodoros: int      // ceil(EstimatedTime / WorkDuration), 0 if no estimate
  completed_pomodoros: int    // count of completed work sessions for this task
  pomodoro_status: string     // "not_started" | "in_progress" | "completed" | "exceeded"
```

**Calculation rules**:
- `planned_pomodoros = 0` when `EstimatedTime` is 0 or unset
- `completed_pomodoros` = `SELECT COUNT(*) FROM pomodoro_sessions WHERE task_id = ? AND type = 'work' AND status = 'completed'`
- Status logic:
  - `planned == 0` → `"not_started"` (no estimate, free-form pomodoro)
  - `completed == 0 && planned > 0` → `"not_started"`
  - `0 < completed < planned` → `"in_progress"`
  - `completed == planned` → `"completed"`
  - `completed > planned` → `"exceeded"`

### New API Endpoints

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/analytics/pomodoro-by-task` | GET | Per-task pomodoro stats (ranking, completion rate) |
| `/api/analytics/pomodoro-trends` | GET | Daily planned vs actual pomodoro comparison |

**Query parameters** (both endpoints):
- `period`: `"week"` | `"month"` (default: `"week"`)
- `start_date`, `end_date`: optional date range override

**`pomodoro-by-task` response**:
```json
{
  "tasks": [
    {
      "task_id": "abc",
      "task_title": "Write weekly report",
      "planned_pomodoros": 4,
      "completed_pomodoros": 8,
      "total_focus_minutes": 200,
      "status": "exceeded"
    }
  ]
}
```

**`pomodoro-trends` response**:
```json
{
  "days": [
    {
      "date": "2026-06-01",
      "planned": 6,
      "actual": 5,
      "completed_tasks": 2,
      "exceeded_tasks": 1
    }
  ]
}
```

### Modified Endpoints

| Endpoint | Change |
|----------|--------|
| `GET /api/tasks` | Response includes `planned_pomodoros`, `completed_pomodoros`, `pomodoro_status` |
| `GET /api/tasks/:id` | Same enriched response |

### Backend Implementation

**Task Service** — new enrichment method:

```go
func (s *TaskService) enrichWithPomodoroInfo(tasks []*model.Task, workDuration int) {
    for _, task := range tasks {
        if task.EstimatedTime > 0 && workDuration > 0 {
            // EstimatedTime is in minutes, WorkDuration is in seconds
            task.PlannedPomodoros = int(math.Ceil(float64(task.EstimatedTime) / float64(workDuration / 60)))
        }
        task.CompletedPomodoros = s.sessionRepo.CountByTaskID(task.ID, "work", "completed")
        task.PomodoroStatus = computeStatus(task.PlannedPomodoros, task.CompletedPomodoros, task.Status)
    }
}
```

**Session Repository** — new query method:

```go
CountByTaskID(taskID string, sessionType string, status string) (int, error)
```

**Analytics Service** — two new methods for the new endpoints.

**Extension logic** (no new field): When user chooses "continue", frontend creates a new PomodoroSession with the same `task_id`. `completed_pomodoros` naturally increments. Status auto-transitions to `exceeded`.

## Completion Reminder Flow

1. User completes a work Pomodoro → frontend receives WebSocket completion event
2. Frontend checks: `completed_pomodoros >= planned_pomodoros && planned_pomodoros > 0`
3. If exactly met, show `ElMessageBox` dialog:

```
Title: 番茄钟已全部完成
Content: 任务「{task_title}」的 {planned}/{planned} 个番茄钟已完成。
Buttons: [标记任务完成] [再来一个番茄钟]
```

4. **"标记任务完成"** → `updateTask(id, { status: 'completed' })`
5. **"再来一个番茄钟"** → `createSession({ task_id, type: 'work' })`

**Edge cases**:
- No estimated time (`planned_pomodoros = 0`) → no reminder
- Task already `status: completed` → no reminder
- Already exceeded (`completed > planned`) → no reminder (only triggers at the exact boundary)
- During break sessions → no reminder (only on work session completion)

## Frontend: Task View

### Task Card (Quadrant View)

Compact single-line layout. Each card row:

```
[Task title]          2/4 番茄钟  [▶]
```

- Progress text: `completed/planned 番茄钟` (hidden if `planned == 0`)
- Circular play button (28px, accent color `#B8452C`, ▶ icon only)
- Completed tasks: no button, slight opacity
- No estimated time: show `—` in progress area, still has play button
- Active pomodoro: button shows ⏸ state

### Task Detail Dialog

Opened by clicking task title. Contains:

1. **Basic info**: title, description, deadline, quadrant
2. **Pomodoro progress section**:
   - Progress bar (accent color fill)
   - Text: `completed/planned` with percentage
   - Today's session history (plain text, no emoji): time ranges per session
   - "开始第 N 个番茄钟" button
3. **Footer stats**: "已专注 X 分钟 · 剩余约 Y 分钟"

The detail dialog is a shared component used in both Task view and Schedule view.

## Frontend: Schedule View

### Quick-Start Button

At the top of the schedule view, a **"开始番茄" button**:

- Auto-selects the nearest upcoming uncompleted task with estimated time
- If an active pomodoro exists: button shows "查看进行中" (navigates to timer)
- If no pending tasks: button is disabled with tooltip "暂无待办任务"

### Schedule Event Cards

Calendar event cards for tasks show pomodoro progress text (e.g., `2/4 番茄钟`), consistent with task cards. Clicking an event opens the same task detail dialog.

### AI-Generated Pomodoro Events

Existing AI schedule generation already creates Pomodoro-type events linked to tasks. These gain:
- Display of associated task's pomodoro progress
- Click to start/continue the task's pomodoro

## Frontend: Analytics View

Three new statistics modules, integrated into the existing Analytics page as a new section or tab.

### 1. Pomodoro Ranking

Per-task pomodoro count ranking, displayed as a horizontal bar chart.

- Each row: rank number, task title, bar (width proportional to count), count text
- Period selector: week / month
- Accent color bars on neutral background

### 2. Planned vs Actual Trend

Daily comparison bar chart over the selected period.

- Each day: two bars side by side
  - Light bar (`#e8e4df`): planned pomodoros (sum of all tasks' planned counts for that day)
  - Accent bar (`#B8452C`): actual completed pomodoros
- Legend: "计划" and "实际"
- Period selector: 7 days / 30 days

### 3. Completion Rate

Three circular indicators showing percentage breakdown:

- **按时完成**: Tasks where `completed_pomodoros == planned_pomodoros` and status is `completed`
- **超时完成**: Tasks where `completed_pomodoros > planned_pomodoros`
- **未完成**: Tasks where `planned_pomodoros > 0` and status is not `completed` and `completed_pomodoros < planned_pomodoros`

Period selector: week / month, consistent with other modules.

### Style

- No decorative emoji or icons in section titles
- Use existing design tokens (`--accent-primary: #B8452C`, `--bg-primary: #FAF9F6`)
- Section titles are plain text (e.g., "番茄钟排行榜", not "🏆 番茄钟排行榜")

## Frontend Types

Add to `frontend/src/types/index.ts`:

```typescript
interface TaskPomodoroInfo {
  plannedPomodoros: number
  completedPomodoros: number
  pomodoroStatus: 'not_started' | 'in_progress' | 'completed' | 'exceeded'
}

// Extend existing Task type with these fields
interface PomodoroByTask {
  taskId: string
  taskTitle: string
  plannedPomodoros: number
  completedPomodoros: number
  totalFocusMinutes: number
  status: string
}

interface PomodoroTrendDay {
  date: string
  planned: number
  actual: number
  completedTasks: number
  exceededTasks: number
}
```

## Scope

### In Scope
- Task API response enrichment with pomodoro computed fields
- Task card pomodoro progress display and quick-start button
- Task detail dialog with full pomodoro progress and session history
- Completion reminder dialog (ElMessageBox)
- Schedule view quick-start button and event card progress
- Analytics: ranking, planned vs actual trend, completion rate
- Two new analytics API endpoints

### Out of Scope
- Per-session custom duration when starting from task (uses global settings)
- Pomodoro count editing (users edit EstimatedTime instead)
- Push notifications for reminders (in-app dialog only)
- Historical pomodoro data migration
- Team/multi-user pomodoro features
