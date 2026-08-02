# Story 1.2: 前端修订交互界面

Status: ready-for-dev

## Story

As a **TickTask 用户（老刘）**,
I want **在日程界面点击"修订日程"按钮，输入自然语言指令，看到 AI 实时修订进度，预览变更差异后决定是否应用**,
so that **我可以对已生成的日程做针对性微调而无需重新生成整周安排**.

## Acceptance Criteria

**AC1: 修订按钮**
- 工具栏中"生成日程"按钮右侧显示「修订日程」按钮（Edit 图标）
- `scheduleStore.loading` 为 `true` 时按钮禁用
- `events` 为空时按钮禁用，tooltip "请先生成日程"

**AC2: 修订指令输入对话框**
- el-dialog，标题「修订日程」，textarea（4-6行，placeholder 含引导示例）
- 底部显示日期范围提示（灰色小字）
- 「取消」和「开始修订」按钮（primary accent 色）
- 空 prompt 时不允许提交

**AC3: 修订执行与 TerminalOverlay**
- 提交后关闭输入框，TerminalOverlay 弹出（复用现有，`cliToolName` 设为 "日程修订引擎"）
- WebSocket 实时展示 Claude 输出
- 完成后 TerminalOverlay 自动关闭 → 弹出预览

**AC4: 修订预览对话框**
- 顶部变更统计摘要（如 "✨ 共调整 3 个任务：2 个移动，1 个新增"）
- 变更列表：el-tag 彩色标签（moved=orange, added=green, removed=gray）+ 时间箭头
- 「取消」和「确认应用」
- 空变更时显示"无需调整"

**AC5: 确认应用**
- POST /revise/apply → 刷新 events → ElMessage.success
- 取消 → 不调用 API，日程不变

**AC6: 错误处理**
- TerminalOverlay 显示错误 + ElMessage.error
- 原有 events 不变

## Tasks / Subtasks

- [ ] Task 1: 新增前端类型定义 (AC3, AC4)
  - [ ] Subtask 1.1: 在 `types/index.ts` 新增 `RevisionChange` 和 `ReviseResponse` 接口

- [ ] Task 2: 新增 API 客户端方法 (AC3, AC5)
  - [ ] Subtask 2.1: 在 `api/client.ts` 新增 `reviseSchedule(prompt)` 和 `applyRevision()` 方法

- [ ] Task 3: Store 新增修订状态和 Actions (AC3, AC4, AC5)
  - [ ] Subtask 3.1: 新增 `revisionChanges`, `revisionSummary` state refs
  - [ ] Subtask 3.2: 新增 `reviseSchedule(prompt)` action（复用 TerminalOverlay + WebSocket）
  - [ ] Subtask 3.3: 新增 `applyRevision()` action（调用 API 后刷新 events）

- [ ] Task 4: 修订按钮 + 输入对话框 (AC1, AC2)
  - [ ] Subtask 4.1: Schedule.vue 工具栏添加「修订日程」按钮
  - [ ] Subtask 4.2: Schedule.vue 添加修订输入 el-dialog

- [ ] Task 5: 修订预览对话框 (AC4, AC5, AC6)
  - [ ] Subtask 5.1: Schedule.vue 添加修订预览 el-dialog（变更摘要 + 变更列表 + 取消/确认按钮）
  - [ ] Subtask 5.2: 空变更和错误状态处理

- [ ] Task 6: 端到端验证 (AC1-AC6)
  - [ ] Subtask 6.1: npm run build 验证编译通过
  - [ ] Subtask 6.2: 手动验证完整修订流程

## Dev Notes

### API 契约（后端已实现）

```
POST /api/schedules/revise
  Request:  { "prompt": "..." }
  Response: { "applied": false, "summary": "共调整 N 个任务...", "changes": [...], "events": [] }

POST /api/schedules/revise/apply
  Request:  无 body
  Response: { "applied": true, "events": [...] }
```

### 修改文件清单

| 文件 | 操作 |
|------|------|
| `frontend/src/types/index.ts` | UPDATE — 新增 RevisionChange, ReviseResponse |
| `frontend/src/api/client.ts` | UPDATE — 新增 reviseSchedule, applyRevision |
| `frontend/src/stores/schedule.ts` | UPDATE — 新增 state refs + 2 actions |
| `frontend/src/views/Schedule.vue` | UPDATE — 按钮 + 2 对话框 |

### 设计约束

- 按钮样式：`el-button` + `Edit` 图标（`@element-plus/icons-vue`），与"生成日程"一致
- 对话框按钮：`--accent-primary: #B8452C`（burnt umber）
- 变更标签：moved=`warning`(orange), added=`success`(green), removed=`info`(gray)
- TerminalOverlay 复用现有组件，无需修改
- 时间格式：`MM/DD 星期X HH:mm`

## Dev Agent Record

### Agent Model Used

Claude (via bmad-dev-story)

### Debug Log References

### Completion Notes List

### File List

## Change Log

- Created story file
