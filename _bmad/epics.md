---
stepsCompleted: [step-01-validate-prerequisites, step-02-design-epics, step-03-create-stories, step-04-final-validation]
inputDocuments:
  - _bmad/prds/prd-ticktask-2026-06-05/prd.md
  - _bmad/C-UX-Scenarios/00-ux-scenarios.md
  - _bmad/C-UX-Scenarios/01-daily-workflow/1.3-schedule/1.3-schedule.md
---

# TickTask - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for the **修订日程（Schedule Revision）** feature, decomposing the requirements from the PRD and UX scenarios into implementable stories.

## Requirements Inventory

### Functional Requirements

FR1: 修订日程按钮 — 在日程界面工具栏中，"生成日程"按钮右侧添加"修订日程"按钮，文案为「修订日程」，使用 Edit/Brush 图标；当 `scheduleStore.loading` 为 true 或 `events` 为空时按钮禁用，悬停提示"请先生成日程"

FR2: 修订指令输入 — 点击按钮后弹出输入对话框，标题为「修订日程」，包含 4-6 行 textarea（placeholder 含引导示例）、当前日期范围提示文字，以及「取消」和「开始修订」按钮

FR3: 修订执行流程 — 提交后调用 `POST /api/schedules/revise`，后端通过 Claude CLI 执行 `docs/skills/revise-schedule` skill（读取现有 `schedule.ics` 作为基线 + `todo.json` + `habit.md`），生成修订后的 ICS 写入 `config/schedule_revised.ics`，TerminalOverlay 实时展示 Claude 输出；后端对比原始和修订后的 ICS 计算变更差异（moved/added/removed），完成后返回差异数据，TerminalOverlay 自动关闭

FR4: 修订预览与确认 — 显示「修订预览」对话框，顶部展示变更统计摘要（如"3 个任务将被调整"），变更列表逐条展示类型标签（移动/新增/移除配不同颜色）、任务标题、时间变更；「取消」不应用修订，「确认应用」后调用 `POST /api/schedules/revise/apply` 写回数据库，刷新日历视图，显示成功通知

FR5: 校验流程 — 修订完成后运行 `validate_schedule.py` 校验，不通过时 skill 内部自动重试（最多 2 次），2 次仍失败则记录到 `learning.md`，前端显示警告"修订完成但存在部分冲突，请查看"

### NonFunctional Requirements

NFR1: 用户体验 — TerminalOverlay 必须展示修订实时进度（与生成流程体验一致）；修订预览对话框中时间格式统一使用 `MM/DD 星期X HH:mm`；修订失败时原有日程数据不被覆盖

NFR2: 数据安全 — 修订仅在预览确认后才写入数据库；临时文件 `schedule_revised.ics` 在校验通过但用户取消后清理；修订过程中原有 `schedule.ics` 不被修改直到用户确认

NFR3: 向后兼容 — 不影响现有"生成日程"按钮行为；不影响 `auto-schedule` skill；新增 API 端点（`/api/schedules/revise` 和 `/api/schedules/revise/apply`），不修改现有 `/api/schedules/generate`

### Additional Requirements

- 无 Architecture 文档，无额外技术约束
- Backend: 复用现有 ICS 解析逻辑（`schedule_service.go`）和 WebSocket 推送机制（`hub.go`）
- Backend: 新增 handler 端点需在 `router.go` 注册，复用现有 `schedule_handler` 或新建 `revise_handler`
- Frontend: 复用现有 `TerminalOverlay.vue` 组件展示实时 Claude 输出
- Frontend: 新增变更预览对话框组件，遵循现有设计系统（`App.vue` CSS 自定义属性）
- `revise-schedule` SKILL.md 已存在于 `docs/skills/revise-schedule/SKILL.md`，`learning.md` 已初始化
- Claude CLI 子进程调用模式复用 `GenerateSchedule()` 中已有的实现

### UX Design Requirements

UX-DR1: 修订按钮与现有工具栏风格一致 — 遵循 `Schedule.vue` 工具栏中 el-button 的排列方式，紧邻"生成日程"（MagicStick 图标）右侧；图标使用 Element Plus `Edit`，保持精炼极简主义风格（无渐变、无发光、无弹跳动画）

UX-DR2: 修订输入对话框设计 — 使用 el-dialog 组件，标题「修订日程」，textarea 4-6 行带引导性 placeholder（如"描述你想如何调整日程，例如：把代码评审移到下午、优化上午的深度工作安排……"）；底部显示当前日程日期范围提示（灰色小字）；按钮组「取消」（secondary）和「开始修订」（primary accent 色）

UX-DR3: 修订预览对话框设计 — 使用 el-dialog 组件，标题「修订预览」；顶部变更统计摘要条（如 "✨ 共调整 3 个任务：2 个移动，1 个新增"）；变更列表使用 el-tag 标记类型（moved=橙色、added=绿色、removed=灰色），每条显示任务标题 + 时间变化（箭头表示移动）；按钮组「取消」和「确认应用」

UX-DR4: 修订预览对话框空状态 — 当 AI 判断无需修订时（changes 为空），显示空状态："当前日程已是最优安排，无需调整"，仅有关闭按钮

UX-DR5: 修订失败状态 — 当网络错误或 Claude 执行失败时，TerminalOverlay 显示错误信息并保持 5 秒可见（复用现有错误展示逻辑）；原有日程不被覆盖，用户可重试

### FR Coverage Map

| FR | Epic | 描述 |
|----|------|------|
| FR1 | Epic 1 | 修订日程按钮 |
| FR2 | Epic 1 | 修订指令输入对话框 |
| FR3 | Epic 1 | 修订执行流程（API + Claude CLI + TerminalOverlay） |
| FR4 | Epic 1 | 修订预览与确认对话框 |
| FR5 | Epic 1 | 校验流程 |

| NFR | Epic | 描述 |
|------|------|------|
| NFR1 | Epic 1 | 用户体验（实时进度、时间格式、失败不覆盖） |
| NFR2 | Epic 1 | 数据安全（确认后写DB、临时文件清理） |
| NFR3 | Epic 1 | 向后兼容（不影响generate/auto-schedule/现有API） |

| UX-DR | Epic | 描述 |
|--------|------|------|
| UX-DR1 | Epic 1 | 按钮与工具栏风格一致 |
| UX-DR2 | Epic 1 | 输入对话框设计 |
| UX-DR3 | Epic 1 | 预览对话框设计 |
| UX-DR4 | Epic 1 | 预览空状态 |
| UX-DR5 | Epic 1 | 失败状态 |

## Epic List

### Epic 1: 修订日程

用户对已生成的七日日程基本满意时，可通过自然语言指令进行针对性微调，实时看到 AI 修订过程，预览变更差异后再确认应用——全程无需重新生成整个日程。

**FRs covered:** FR1, FR2, FR3, FR4, FR5

---

## Epic 1: 修订日程

用户对已生成的七日日程基本满意时，可通过自然语言指令进行针对性微调，实时看到 AI 修订过程，预览变更差异后再确认应用——全程无需重新生成整个日程。

### Story 1.1: 后端修订 API 与差异计算

As a **前端开发者**,
I want **后端提供 `/api/schedules/revise` 和 `/api/schedules/revise/apply` 两个端点，支持通过 Claude CLI 执行 revise-schedule skill 并计算 ICS 变更差异**,
So that **前端可以调用标准 API 完成修订预览和确认应用的两阶段流程**.

**Acceptance Criteria:**

**AC1: 修订请求处理**
**Given** 前端发送 `POST /api/schedules/revise` 请求，body 为 `{ "prompt": "把代码评审移到下午" }`
**When** 后端收到请求
**Then** 后端从数据库读取当前周（周一至周日）的日程事件，序列化为 `config/schedule.ics` 文件，同时写入 `config/todo.json`（当前周任务）和 `config/habit.md`（用户作息设置）
**And** 通过 WebSocket 广播 `terminal_status: { status: "started" }`

**AC2: Claude CLI 调用**
**Given** 前置文件已就绪（schedule.ics / todo.json / habit.md）
**When** 后端启动 Claude CLI 子进程，prompt 为执行 `docs/skills/revise-schedule` skill
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

**Requirements covered:** FR3, FR5, NFR2, NFR3

---

### Story 1.2: 前端修订交互界面

As a **TickTask 用户（老刘）**,
I want **在日程界面点击"修订日程"按钮，输入自然语言指令，看到 AI 实时修订进度，预览变更差异后决定是否应用**,
So that **我可以对已生成的日程做针对性微调而无需重新生成整周安排，并且有完全的掌控感——看到改了什么再确认**.

**Acceptance Criteria:**

**AC1: 修订按钮**
**Given** 用户位于日程界面（`/schedule`），当前周有日程事件（`events` 不为空）
**When** 页面加载完成
**Then** 工具栏中"生成日程"按钮右侧显示「修订日程」按钮
**And** 按钮使用 Element Plus `Edit` 图标
**And** 按钮样式与"生成日程"一致（el-button，无特殊动画，符合精炼极简主义风格）
**Given** `scheduleStore.loading` 为 `true`（正在生成/修订中）
**When** 用户查看按钮状态
**Then** 「修订日程」按钮处于 disabled 状态
**Given** 当前周无日程事件（`events` 为空）
**When** 用户鼠标悬停在按钮上
**Then** 按钮 disabled 并显示 tooltip："请先生成日程"

**AC2: 修订指令输入对话框**
**Given** 用户点击「修订日程」按钮
**When** 按钮被点击
**Then** 弹出 el-dialog 对话框，标题为「修订日程」
**And** 对话框包含一个 el-input（type="textarea"，4-6 行），placeholder 为引导文案如"描述你想如何调整日程，例如：把代码评审移到下午、优化上午的深度工作安排、为紧急任务腾出 2 小时……"
**And** 对话框底部显示当前日程覆盖的日期范围提示（灰色小字），格式如"修订范围：2026-06-08（周一）至 2026-06-14（周日）"
**And** 底部包含两个按钮：「取消」（secondary 样式）和「开始修订」（primary accent 色，`--accent-primary: #B8452C`）
**Given** 用户未输入任何内容
**When** 用户点击「开始修订」
**Then** 按钮 disabled（或点击后提示"请输入修订指令"），不允许提交空 prompt
**Given** 用户点击「取消」或对话框外部区域
**When** 对话框关闭
**Then** 不发起任何 API 请求，textarea 内容清空

**AC3: 修订执行与 TerminalOverlay**
**Given** 用户在输入对话框中输入了修订指令（如"把代码评审移到下午"）并点击「开始修订」
**When** 请求发送
**Then** 输入对话框关闭
**And** TerminalOverlay 全屏覆盖层弹出，显示 `$ claude -p "执行 skill: docs/skills/revise-schedule"`
**And** TerminalOverlay 实时展示从 WebSocket 推送的 Claude stdout/stderr 文本行，光标闪烁
**And** 终端状态指示器显示为 "started"
**Given** 修订执行中（TerminalOverlay 可见）
**When** Claude 执行完成且 WebSocket 收到 `terminal_status: { status: "completed" }`
**Then** TerminalOverlay 显示完成状态，2 秒后自动关闭

**AC4: 修订预览对话框**
**Given** TerminalOverlay 关闭，`POST /api/schedules/revise` 返回了差异数据
**When** 前端收到响应
**Then** 弹出 el-dialog 对话框，标题为「修订预览」
**And** 顶部显示变更统计摘要，格式如 "✨ 共调整 3 个任务：2 个移动，1 个新增，0 个移除"
**And** 变更列表逐条展示，每项包含：
  - `el-tag` 类型标签：`moved` = 橙色、`added` = 绿色、`removed` = 灰色（使用设计系统中已有的配色）
  - 任务标题（粗体）
  - 时间变化：移动类型显示 `原始时间 → 新时间`（箭头），新增类型显示 `新时间`，移除类型显示 `原时间（将被移除）`
  - 时间格式统一为 `MM/DD 星期X HH:mm`（如 `06/09 周一 10:00 → 06/09 周一 14:00`）
**And** 底部包含两个按钮：「取消」（secondary 样式）和「确认应用」（primary accent 色）

**AC5: 预览空状态**
**Given** `POST /api/schedules/revise` 返回 `changes: []`（AI 判断无需修订）
**When** 前端收到响应
**Then** 预览对话框显示空状态："当前日程已是最优安排，无需调整"
**And** 仅显示「关闭」按钮（无「确认应用」按钮）

**AC6: 确认应用**
**Given** 用户在预览对话框中看到变更列表并点击「确认应用」
**When** 前端调用 `POST /api/schedules/revise/apply`
**Then** 响应返回 `{ applied: true, events: [...] }`
**And** 预览对话框关闭
**And** `scheduleStore.events` 更新为返回的最新日程事件
**And** 日历视图（Day/Week/Month）自动刷新显示新日程
**And** `ElMessage.success("日程修订成功")` 提示显示
**Given** 用户在预览对话框中点击「取消」
**When** 对话框关闭
**Then** 不调用 `/revise/apply`，日程保持修订前状态不变
**And** 无任何成功或错误提示

**AC7: 错误处理**
**Given** 修订过程中发生错误（网络错误、服务器 500、Claude 执行失败）
**When** 错误被捕获
**Then** TerminalOverlay 显示错误信息（复用现有错误展示逻辑），保持 5 秒可见
**And** `ElMessage.error("日程修订失败，请重试")` 提示显示
**And** 原有日程数据不被修改（events 数组不变）
**And** 用户可再次点击「修订日程」重试

**AC8: API 客户端与 Store**
**Given** `frontend/src/api/client.ts` 文件
**When** 新增 API 方法
**Then** `api.reviseSchedule(prompt: string)` 方法发送 `POST /api/schedules/revise`，超时 360 秒（与 generate 一致）
**And** `api.applyRevision()` 方法发送 `POST /api/schedules/revise/apply`
**Given** `frontend/src/stores/schedule.ts` store
**When** 新增 actions
**Then** `reviseSchedule(prompt: string)` action 复用 `setupTerminalListener()` 注册 WebSocket 监听，设置 `aiGenerating = true`，调用 `api.reviseSchedule(prompt)`，返回差异数据
**And** `applyRevision()` action 调用 `api.applyRevision()`，更新 `events` 数组（按 `task_id` 去重合并，保留自定义事件），设置 `aiGenerating = false`

**Requirements covered:** FR1, FR2, FR3, FR4, FR5, NFR1, UX-DR1, UX-DR2, UX-DR3, UX-DR4, UX-DR5
