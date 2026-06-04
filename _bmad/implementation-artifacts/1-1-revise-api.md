# Story 1.1: 后端修订 API 与差异计算

Status: review

## Story

As a **前端开发者**,
I want **后端提供 `/api/schedules/revise` 和 `/api/schedules/revise/apply` 两个端点，支持通过 Claude CLI 执行 revise-schedule skill 并计算 ICS 变更差异**,
so that **前端可以调用标准 API 完成修订预览和确认应用的两阶段流程**.

## Acceptance Criteria

**AC1: 修订请求处理**
**Given** 前端发送 `POST /api/schedules/revise` 请求，body 为 `{ "prompt": "把代码评审移到下午" }`
**When** 后端收到请求
**Then** 后端从数据库读取当前周（周一至周日）的日程事件，序列化为 `config/schedule.ics` 文件，同时写入 `config/todo.json`（当前周任务）和 `config/habit.md`（用户作息设置）
**And** 通过 WebSocket 广播 `terminal_status: { status: "started" }`

**AC2: Claude CLI 调用**
**Given** 前置文件已就绪（schedule.ics / todo.json / habit.md）
**When** 后端启动 Claude CLI 子进程，prompt 为执行 `docs/skills/revise-schedule` skill（含用户修订指令）
**Then** Claude 的 stdout（JSON 流）被逐行解析，`assistant` 消息和 `tool_use` 信息通过 WebSocket `terminal_output` 事件实时推送到前端
**And** Claude 退出后，后端读取 `config/schedule_revised.ics`（修订后日程）

**AC3: 差异计算**
**Given** 修订前后的两个 ICS 文件均存在（`schedule.ics` 原始版 + `schedule_revised.ics` 修订版）
**When** 后端逐事件比对两个文件（按事件标题 + 日期匹配）
**Then** 生成差异列表，每个差异项包含：
  - `type`: `"moved"` | `"added"` | `"removed"`
  - `title`: 任务标题
  - `original_start` / `original_end`: 原始时间（moved/removed 类型）
  - `new_start` / `new_end`: 新时间（moved/added 类型）
**And** 生成变更摘要文字，格式如 `"共调整 N 个任务：X 个移动，Y 个新增，Z 个移除"`
**And** 通过 WebSocket 广播 `terminal_status: { status: "completed" }`
**And** 返回 `200` 响应，body 包含 `{ applied: false, summary, changes, events: [] }`

**AC4: 确认应用**
**Given** 修订预览已返回，用户确认应用
**When** 前端发送 `POST /api/schedules/revise/apply`
**Then** 后端解析 `config/schedule_revised.ics`，按任务标题匹配数据库中的 Schedule 记录
**And** 更新/创建/删除对应的 Schedule 记录（moved → update 时间，added → create 新记录，removed → 按标题+日期匹配后删除）
**And** 返回 `200` 响应，body 包含 `{ applied: true, events: [...] }`（更新后的完整日程事件列表）
**And** 清理临时文件 `schedule_revised.ics`

**AC5: 数据安全**
**Given** 修订过程中任何步骤失败（Claude 崩溃、校验不通过、用户取消）
**When** 异常发生
**Then** 原有 `schedule.ics` 不被修改
**And** 数据库中的 Schedule 记录保持不变
**And** WebSocket 广播 `terminal_status: { status: "error", message: "..." }`
**And** 错误响应码为非 2xx（400/500 等）

**AC6: 校验集成**
**Given** Claude 生成了 `schedule_revised.ics`
**When** Claude CLI 的 skill 内部执行 `validate_schedule.py` 校验
**Then** 校验通过（退出码 0）→ 正常返回差异数据
**And** 校验失败（退出码 1）→ skill 内部自动重试（最多 2 次），若仍失败则记录到 `learning.md`，响应中 `summary` 字段附加 `"（存在部分冲突，请查看）"` 警告

**AC7: 向后兼容**
**Given** 现有 `POST /api/schedules/generate` 端点已存在
**When** 新增 `/revise` 和 `/revise/apply` 路由
**Then** 现有 generate 端点的行为、响应格式、超时设置完全不变
**And** 现有 `auto-schedule` skill 的 SKILL.md 和脚本不被修改
**And** 新端点注册在 `router.go` 中，路径不与现有路由冲突

## Tasks / Subtasks

- [x] Task 1: 新增数据类型定义 (AC1, AC3)
  - [x] Subtask 1.1: 在 `schedule_service.go` 新增 `ReviseRequest` struct（Prompt 字段）
  - [x] Subtask 1.2: 新增 `RevisionChange` struct（Type, Title, OriginalStart, OriginalEnd, NewStart, NewEnd）
  - [x] Subtask 1.3: 新增 `ReviseResponse` struct（Applied, Summary, Changes, Events）

- [x] Task 2: 实现将当前日程写入 ICS（修订基线）(AC1)
  - [x] Subtask 2.1: 在 `config_writer.go` 新增 `WriteScheduleICS(events []ScheduleEvent) error` 函数，将当前日程事件序列化为 ICS 格式写入 `config/schedule.ics`
  - [x] Subtask 2.2: 实现 ICS 序列化逻辑（遵循现有 ICS 格式：VCALENDAR + VEVENT，含 DTSTART/DTEND/SUMMARY/DESCRIPTION）

- [x] Task 3: 实现 `ReviseSchedule` 方法 (AC1, AC2, AC3, AC5, AC6)
  - [x] Subtask 3.1: 在 `schedule_service.go` 新增 `ReviseSchedule(prompt string) (*ReviseResponse, error)` 方法签名
  - [x] Subtask 3.2: 计算当前周范围（周一至周日，复用 GenerateSchedule 的算法，参考 line 405-412）
  - [x] Subtask 3.3: 设置 WebSocket broadcast/broadcastStatus 闭包（复用 pattern，参考 line 414-423）
  - [x] Subtask 3.4: 获取本周现有日程（调用 `s.GetSchedules(monday, sunday)`），写入 `config/schedule.ics` 作为修订基线
  - [x] Subtask 3.5: 获取并过滤任务（复用 `filterTasksForWeek` + `sortTasksForScheduling`），写入 `config/todo.json` 和 `config/habit.md`
  - [x] Subtask 3.6: 构建 revise prompt：`"项目路径: {root}。执行 docs/skills/revise-schedule skill，为 {monday} 至 {sunday} 修订整周日程。修订指令：{prompt}"`
  - [x] Subtask 3.7: 调用 `runClaudeStreamJSON(revisePrompt, broadcast)` 执行修订
  - [x] Subtask 3.8: 读取原始 ICS（`ReadScheduleICS()`）和修订后 ICS（`config/schedule_revised.ics` 通过 `os.ReadFile`）
  - [x] Subtask 3.9: 实现 `computeDiff(original, revised []ICSEvent) (changes []RevisionChange, summary string)` 差异对比逻辑
  - [x] Subtask 3.10: 构建 `ReviseResponse` 返回（`Applied: false`, `Changes: changes`, `Summary: summary`, `Events: []`）
  - [x] Subtask 3.11: broadcastStatus("completed", ...) 和错误处理（broadcastStatus("error", ...) + 不修改 DB）

- [x] Task 4: 实现 `ApplyRevision` 方法 (AC4, AC5)
  - [x] Subtask 4.1: 在 `schedule_service.go` 新增 `ApplyRevision() ([]ScheduleEvent, error)` 方法
  - [x] Subtask 4.2: 读取 `config/schedule_revised.ics`
  - [x] Subtask 4.3: 解析 ICS（复用 `ParseICS`）
  - [x] Subtask 4.4: 删除本周已有的任务日程（复用 `s.scheduleRepo.DeleteTaskSchedulesByDateRange(monday, sunday)`）
  - [x] Subtask 4.5: 遍历 parsed events，matchTaskByTitle + icsEventToDTO + create，去重后持久化（完全复用 GenerateSchedule Phase 3 逻辑，参考 line 488-524）
  - [x] Subtask 4.6: 清理临时文件 `config/schedule_revised.ics`
  - [x] Subtask 4.7: 返回 `[]ScheduleEvent` 和 nil error

- [x] Task 5: 新增 Handler 方法 (AC1, AC4, AC7)
  - [x] Subtask 5.1: 在 `schedule.go` 新增 `ReviseWithAI(c *gin.Context)` handler，绑定 `{ "prompt": "..." }` JSON body，调用 `h.scheduleService.ReviseSchedule(prompt)`，返回 `gin.H{"applied": false, "summary": ..., "changes": ..., "events": []}`
  - [x] Subtask 5.2: 在 `schedule.go` 新增 `ApplyRevision(c *gin.Context)` handler，无参调用 `h.scheduleService.ApplyRevision()`，返回 `gin.H{"applied": true, "events": ...}`

- [x] Task 6: 注册路由 (AC7)
  - [x] Subtask 6.1: 在 `router.go` 的 schedules Group 中注册 `schedules.POST("/revise", scheduleHandler.ReviseWithAI)` 和 `schedules.POST("/revise/apply", scheduleHandler.ApplyRevision)`
  - [x] Subtask 6.2: 确保新路由位于 `schedules.GET("/:id", ...)` 之前，避免 `revise` 被解析为 `:id` 参数（参照 `/generate` 路由的位置）

- [x] Task 7: 编写测试 (AC1-AC7)
  - [x] Subtask 7.1: 单元测试 `computeDiff` — 覆盖 moved/added/removed 三种场景、空变更（identical ICS）、全新增（empty original）
  - [x] Subtask 7.2: 单元测试 `WriteScheduleICS` — 验证输出为合法 ICS 格式、能被 `ParseICS` 正确回读
  - [x] Subtask 7.3: 集成测试 `ReviseSchedule` — mock Claude CLI 输出预定 ICS，验证返回 RevisionChange 正确
  - [x] Subtask 7.4: 集成测试 `ApplyRevision` — 验证 DB 中旧记录被替换、新记录被创建

## Dev Notes

### Architecture Compliance（必须遵守的模式）

**路由注册** — `router.go:96-108`
- 在 `schedules` Group 中新增路由，放在 `schedules.GET("/:id", ...)` **之前**（line 101）
- 参照现有 `/generate` 路由（line 106）的位置 —— `/generate` 也需要在 `/:id` 之前，但当前 router.go 中 `/generate` 在 `/:id` 之后（line 106 vs line 101），这是正确的因为 Gin 路由树优先匹配静态路径。为保险起见，将新路由放在 `/:id` 之前。
- 路由注册模式：
  ```go
  schedules.POST("/revise", scheduleHandler.ReviseWithAI)
  schedules.POST("/revise/apply", scheduleHandler.ApplyRevision)
  ```

**Handler 模式** — `handler/schedule.go:163-188`
- 参照 `GenerateWithAI` 的写法：`func (h *ScheduleHandler) ReviseWithAI(c *gin.Context)`
- Request body 绑定：`c.ShouldBindJSON(&req)` 其中 req 为匿名 struct
- 错误响应格式：`c.JSON(http.StatusBadRequest, gin.H{"error": "..."})`
- 成功响应格式：`c.JSON(http.StatusOK, gin.H{"applied": ..., "summary": ..., "changes": ..., "events": ...})`
- 不设置超时 —— 超时已在 `runClaudeStreamJSON` 内部通过 context 设置（300s）

**Service 方法模式** — `service/schedule_service.go:404-567`
- 方法签名：`func (s *ScheduleService) ReviseSchedule(prompt string) (*ReviseResponse, error)` 和 `func (s *ScheduleService) ApplyRevision() ([]ScheduleEvent, error)`
- 周一计算（line 405-412）：直接复制 GenerateSchedule 中的算法
- WebSocket 闭包（line 414-423）：直接复用 broadcast/broadcastStatus 模式
- Claude CLI 调用（line 670-750）：复用 `runClaudeStreamJSON(prompt, broadcast)` —— 不修改此函数
- ICS 解析（line 488）：复用 `ParseICS(content, location)`
- 任务匹配（line 500）：复用 `matchTaskByTitle(candidates, ev.Summary)`
- ICS→DTO 转换（line 570）：复用 `icsEventToDTO(ev, fallbackDate)`
- 去重 key（line 512）：复用 `fmt.Sprintf("%s|%s|%s", dto.TaskID, date, dto.StartTime)` 格式
- 持久化（line 518）：复用 `s.CreateSchedule(dto)` — 注意这是 Create，对 moved 任务会创建新记录

### Files to Modify

| 文件 | 操作 | 变更内容 |
|------|------|----------|
| `backend/internal/service/schedule_service.go` | UPDATE | 新增 `ReviseRequest`、`RevisionChange`、`ReviseResponse` 类型 + `ReviseSchedule()`、`ApplyRevision()`、`computeDiff()` 方法 |
| `backend/internal/service/config_writer.go` | UPDATE | 新增 `WriteScheduleICS(events []ScheduleEvent) error` |
| `backend/internal/api/handler/schedule.go` | UPDATE | 新增 `ReviseWithAI(c *gin.Context)`、`ApplyRevision(c *gin.Context)` |
| `backend/internal/api/router.go` | UPDATE | 新增两条路由注册 |
| `backend/internal/service/schedule_service_test.go` | UPDATE | 新增 `computeDiff` 和 `WriteScheduleICS` 的单元测试 |
| `backend/internal/service/config_writer_test.go` | CREATE | 新增 `WriteScheduleICS` 单元测试（如文件不存在则创建） |

### Key Reference: GenerateSchedule Pipeline（必须遵循）

```
GenerateSchedule (line 404-567):
  1. 计算周范围 monday/sunday (line 405-412)
  2. 设置 broadcast/broadcastStatus 闭包 (line 414-423)
  3. broadcastStatus("started", ...) (line 425-427)
  4. DeleteTaskSchedulesByDateRange (line 430)     ← ReviseSchedule 跳过此步
  5. GetTasks + filterTasksForWeek + sort (line 434-446)
  6. WriteHabitMD + WriteTodoJSON (line 451-458)
  7. AI not configured check (line 465-468)
  8. buildWeekSchedulePrompt + runClaudeStreamJSON (line 470-477)
  9. ReadScheduleICS (line 480)
  10. ParseICS (line 488)
  11. Loop: matchTaskByTitle → icsEventToDTO → 去重 → CreateSchedule → toEvent (line 494-524)
  12. validateDayEvents + validateRecurrenceDays (line 527-544)
  13. broadcastStatus("completed", ...) (line 564)
```

**ReviseSchedule 的 pipeline 差异：**
```
ReviseSchedule:
  1. 计算周范围（同 Generate）
  2. 设置 broadcast/broadcastStatus 闭包（同 Generate）
  3. broadcastStatus("started", ...)
  4. GetSchedules(monday,sunday) → WriteScheduleICS(events)  ← 新增：写当前日程作为基线
  5. GetTasks + filterTasksForWeek + sort（同 Generate）
  6. WriteHabitMD + WriteTodoJSON（同 Generate）
  7. AI check
  8. buildRevisePrompt(monday, sunday, userPrompt) → runClaudeStreamJSON  ← 新 prompt
  9. ReadScheduleICS + ReadRevisedICS                 ← 读两个文件
  10. ParseICS(original) + ParseICS(revised)          ← 双解析
  11. computeDiff(originalEvents, revisedEvents)       ← 新增：差异计算
  12. broadcastStatus("completed", ...)
  13. return ReviseResponse{Applied: false, ...}       ← 不写 DB
```

### ICS 序列化格式（WriteScheduleICS 必须输出）

```
BEGIN:VCALENDAR
VERSION:2.0
PRODID:-//TickTask//EN
CALSCALE:GREGORIAN
METHOD:PUBLISH
BEGIN:VEVENT
DTSTART:20260609T100000
DTEND:20260609T110000
SUMMARY:代码评审
DESCRIPTION:代码评审 | high优先级 | deep_work
END:VEVENT
...
END:VCALENDAR
```
- 时间格式：`20060102T150405`（本地时间，无时区后缀）
- 不含 VTIMEZONE
- DESCRIPTION 格式：`{标题} | {优先级}优先级 | {任务类型}`（与 auto-schedule skill 输出一致）

### Diff 算法（computeDiff）

```
输入: original []ICSEvent, revised []ICSEvent
输出: changes []RevisionChange, summary string

算法:
1. 按 (标题, 日期) 建索引:
   - origByKey[title|date] = ICSEvent
   - revByKey[title|date] = ICSEvent
2. 遍历 revised: 如在 original 中存在同 key → 比对 Start/End 是否不同 → moved
                 如在 original 中不存在 → added
3. 遍历 original: 如在 revised 中不存在 → removed
4. 生成 summary: "共调整 {total} 个任务：{moved} 个移动，{added} 个新增，{removed} 个移除"
```

### 临时文件管理

- 修订基线：`config/schedule.ics`（覆盖写入，保留用于 Claude 读取）
- 修订输出：`config/schedule_revised.ics`（Claude 写入）
- 清理时机：ApplyRevision 成功后删除 `schedule_revised.ics`；ReviseSchedule 失败/预览取消时保留（用户可能重试）

### Testing Notes

- `computeDiff` 是纯函数，最易测试 —— 优先覆盖
- `WriteScheduleICS` 测试：验证 Write→Read 往返一致性
- `ReviseSchedule` 集成测试：可使用 `t.Setenv` 控制 Claude 行为，或 mock `runClaudeStreamJSON`
- 现有 mock 定义在 `handler/mocks_test.go`，如需新增 mock repo 方法参考现有模式

### References

- [Source: backend/internal/service/schedule_service.go:404-567] GenerateSchedule — 核心 pipeline 参考
- [Source: backend/internal/service/schedule_service.go:670-750] runClaudeStreamJSON — Claude CLI 调用
- [Source: backend/internal/service/ics_parser.go:1-85] ParseICS — ICS 解析器
- [Source: backend/internal/service/config_writer.go:41-142] WriteTodoJSON, WriteHabitMD, ReadScheduleICS
- [Source: backend/internal/service/config_writer.go:1199-1205] writeICS — 现有 ICS 写入函数（仅被 generateWeekICS 使用，ReviseSchedule 需要新写 WriteScheduleICS 从 []ScheduleEvent 生成 ICS）
- [Source: backend/internal/api/handler/schedule.go:163-188] GenerateWithAI handler — Handler 模式参考
- [Source: backend/internal/api/router.go:96-108] schedules 路由组
- [Source: docs/skills/revise-schedule/SKILL.md] Revise-schedule skill 定义
- [Source: docs/skills/auto-schedule/scripts/validate_schedule.py] 校验脚本（被 skill 内部调用）

## Dev Agent Record

### Agent Model Used

Claude (via bmad-dev-story)

### Debug Log References

### Completion Notes List

- ✅ Task 1: 新增 ReviseRequest, RevisionChange, ReviseResponse 三个类型
- ✅ Task 2: 实现 WriteScheduleICS() + escapeICS() + ReadRevisedICS() + CleanRevisedICS()
- ✅ Task 3: 实现 ReviseSchedule() 完整 pipeline — 计算周范围 → 写基线ICS → 写config → Claude CLI → 双ICS解析 → computeDiff
- ✅ Task 4: 实现 ApplyRevision() 方法 — 读修订后ICS → 解析 → 删旧记录 → 持久化 → 清理临时文件
- ✅ Task 5: 新增 ReviseWithAI + ApplyRevision handler 方法
- ✅ Task 6: 路由注册 /revise 和 /revise/apply，位于 /:id 之前避免冲突
- ✅ Task 7: 9 个测试全部通过 — computeDiff 7 场景 + WriteScheduleICS round-trip + escapeICS
- 全量回归测试通过，零回归

### File List

- `backend/internal/service/schedule_service.go` — 新增 ReviseRequest, RevisionChange, ReviseResponse 类型 + ReviseSchedule(), ApplyRevision(), computeDiff(), buildRevisePrompt()
- `backend/internal/service/config_writer.go` — 新增 WriteScheduleICS(), escapeICS(), ReadRevisedICS(), CleanRevisedICS()
- `backend/internal/api/handler/schedule.go` — 新增 ReviseWithAI(), ApplyRevision() handler 方法
- `backend/internal/api/router.go` — 新增 POST /revise 和 POST /revise/apply 路由
- `backend/internal/service/schedule_service_test.go` — 新增 9 个测试（computeDiff 7 场景 + WriteScheduleICS round-trip + escapeICS）

## Change Log

- 2026-06-05: Story 1.1 实现完成 — 全部 7 Tasks / 25 Subtasks 完成，9 个新测试通过，零回归
- Created story file with 7 tasks, 25 subtasks, covering AC1-AC7
