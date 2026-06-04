---
title: "PRD: 修订日程功能"
status: final
created: 2026-06-05
updated: 2026-06-05
---

# PRD：修订日程功能

## 概述

在 TickTask 日程界面添加"修订日程"选项，允许用户在当前已生成的七日日程基础上进行针对性微调，而非重新生成整个日程。该功能调用新增的 `revise-schedule` skill，通过 Claude CLI 在现有 `schedule.ics` 基础上执行修订。

**核心价值**：用户对已生成的日程基本满意，仅需局部优化（如移动某任务、为紧急事项腾时间、优化某时段安排），避免全量重新生成带来的不确定性。

---

## 功能需求

### FR-1: 修订日程按钮

**描述**：在日程界面工具栏中，"生成日程"按钮右侧添加"修订日程"按钮。

**位置**：[frontend/src/views/Schedule.vue](frontend/src/views/Schedule.vue) 工具栏区域，现有"生成日程"按钮（el-button，含 MagicStick 图标）的右侧。

**行为**：
- 按钮文案：「修订日程」
- 图标：使用 Element Plus 的 `Edit` 或 `Brush` 图标
- 当 `scheduleStore.loading` 为 `true` 时按钮禁用
- 当 `events` 为空（无当前日程）时按钮禁用，悬停提示"请先生成日程"

### FR-2: 修订指令输入

**描述**：点击按钮后弹出输入对话框，引导用户描述修订需求。

**对话框内容**：
- 标题：「修订日程」
- 输入框（el-input type="textarea"）：4-6 行，placeholder 为引导性文案，如"描述你想如何调整日程，例如：把代码评审移到下午、优化上午的深度工作安排、为紧急任务腾出 2 小时……"
- 提示文字：显示当前日程覆盖的日期范围（如"修订范围：2026-06-08（周一）至 2026-06-14（周日）"）
- 操作按钮：「取消」和「开始修订」

### FR-3: 修订执行流程

**描述**：提交修订指令后，前端调用后端 API 执行 revise-schedule skill。

**流程**：
1. 前端向 `POST /api/schedules/revise` 发送请求，body 包含 `{ prompt: string }`
2. 后端广播 `terminal_status: "started"` 并通过 WebSocket 推送实时 Claude 输出（复用现有 TerminalOverlay 组件）
3. 后端执行 `docs/skills/revise-schedule` skill：
   - 读取现有 `config/schedule.ics`（当前日程基线）
   - 读取 `config/todo.json` 和 `config/habit.md`
   - 根据用户 prompt 执行针对性修订
   - 生成修订后的 ICS 写入临时文件 `config/schedule_revised.ics`
4. Claude 退出后，后端解析原始 ICS 和修订后 ICS，计算**变更差异**
5. 广播 `terminal_status: "completed"`，TerminalOverlay 自动关闭
6. 前端展示修订预览对话框

**差异计算**（后端）：
- 逐事件比对两个 ICS 文件
- 识别三类变更：
  - `moved`：任务时间被调整（标题相同，时间段不同）
  - `added`：新增任务
  - `removed`：移除的任务
- 生成变更摘要文字，如"共调整 3 个任务：2 个移动，1 个新增"

### FR-4: 修订预览与确认

**描述**：修订完成后，展示变更预览对话框，用户确认后才应用。

**对话框内容**：
- 标题：「修订预览」
- 变更摘要：顶部显示变更统计（如"3 个任务将被调整"）
- 变更列表：逐条展示，每条包含：
  - 变更类型标签（移动/新增/移除，使用不同颜色）
  - 任务标题
  - 原始时间 → 新时间（移动类型）
  - 新时间（新增类型）
- 操作按钮：「取消」（不应用修订）和「确认应用」
- 「确认应用」后：用修订后的 ICS 替换原有日程，刷新日历视图，显示成功通知

### FR-5: 校验流程

**描述**：与生成日程保持一致，修订后运行校验脚本。

复用 `auto-schedule` skill 的校验脚本：
```
python3 docs/skills/auto-schedule/scripts/validate_schedule.py --tasks config/todo.json --ics config/schedule_revised.ics
```
- 校验不通过时，skill 内部自动重试（最多 2 次）
- 2 次仍不通过，记录到 `learning.md`，前端显示"修订完成但存在部分冲突，请查看"的警告

---

## 非功能需求

### NFR-1: 用户体验
- 终端覆盖层（TerminalOverlay）必须展示修订的实时进度，与生成流程体验一致
- 修订预览对话框中，变更差异必须清晰可读，时间格式使用 `MM/DD 星期X HH:mm`
- 修订失败时，原有日程不被覆盖

### NFR-2: 数据安全
- 修订仅在预览确认后才写回数据库
- 临时 ICS 文件（`schedule_revised.ics`）在校验通过但用户取消后清理
- 修订过程中原有 `schedule.ics` 不被修改，直到用户确认

### NFR-3: 向后兼容
- 不影响现有"生成日程"按钮行为
- 不影响现有 `auto-schedule` skill
- 新增 API 端点，不修改现有 `/api/schedules/generate`

---

## 技术约束

- 修订通过 Claude CLI 子进程执行 `docs/skills/revise-schedule` skill
- `revise-schedule` SKILL.md 已存在，位于 `docs/skills/revise-schedule/`
- 后端复用现有 ICS 解析逻辑和 WebSocket 推送机制
- 前端复用现有 TerminalOverlay 组件展示实时进度

---

## API 规格

### `POST /api/schedules/revise`

**Request:**
```json
{
  "prompt": "把代码评审移到下午，优化上午的深度工作安排"
}
```

**Response (200):**
```json
{
  "applied": false,
  "summary": "共调整 3 个任务：2 个移动，1 个新增",
  "changes": [
    {
      "type": "moved",
      "title": "代码评审",
      "original_start": "2026-06-09T10:00:00",
      "original_end": "2026-06-09T11:00:00",
      "new_start": "2026-06-09T14:00:00",
      "new_end": "2026-06-09T15:00:00"
    },
    {
      "type": "added",
      "title": "紧急修复",
      "new_start": "2026-06-09T10:00:00",
      "new_end": "2026-06-09T11:30:00"
    }
  ],
  "events": []
}
```

### `POST /api/schedules/revise/apply`

**Request:** 无 body（或 `{ "confirm": true }`）

**Response (200):**
```json
{
  "applied": true,
  "events": [...]
}
```

---

## 待确认问题

1. **Q: 修订是否支持跨周？** 如果当前视图是某周，修订某天后，会不会影响下一周？[ASSUMPTION] 修订范围始终是当前显示的整周（周一至周日），与 generate 保持一致。

2. **Q: 如果用户在修订后又手动移动了某个任务（拖拽），再执行修订，以哪个为准？** [ASSUMPTION] 以当前数据库中实际存储的日程为准（即手动修改后的状态会反映到 ICS 中作为修订基线）。

---

## 成功指标

- 用户可在 3 次点击内完成一次修订（点击按钮 → 输入 prompt → 确认应用）
- 修订预览中的变更差异与实际应用结果一致率 100%
- 修订失败时不破坏原有日程数据
