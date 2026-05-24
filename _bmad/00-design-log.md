# TickTask - Design Log

## Phase 0: Project Setup — 2026-05-24

- **项目类型**: Brownfield（现有产品改进）
- **产品复杂度**: Web Application（复杂）
- **技术栈**: Vue 3 + Vite + Element Plus（前端）、Go（后端）、SQLite（数据库）
- **组件库**: Element Plus（跳过设计系统阶段）
- **简报深度**: 完整
- **工作关系**: 个人项目，均衡参与，产品负责人角色，推荐式建议
- **设计文件位置**: docs/

### 下一步: Phase 8 — 产品进化（改善法）

### 2026-05-24 - Phase 1: Product Brief Complete

**Agent:** Saga (Product Brief)
**Brief Level:** Complete

**Artifacts Created:**
- `A-Product-Brief/product-brief.md`
- `_progress/dialog/00-context.md`
- `_progress/dialog/client-profile.md`
- `_progress/dialog/02-vision.md`
- `_progress/dialog/07-positioning.md`
- `_progress/dialog/03-users.md`
- `_progress/dialog/04-concept.md`
- `_progress/dialog/08-success-criteria.md`
- `_progress/dialog/09-competitive-landscape.md`
- `_progress/dialog/decisions.md`
- `_progress/dialog/progress-tracker.md`

**Summary:** 确定了 TickTask 作为 AI 驱动自动排程引擎的定位，面向软件开发团队解决"靠脑记任务"导致的四大效率痛点（打断/琐事/决策瘫痪/规划成本）。AI 是核心决策大脑，形成"录入→排程→执行→调整→分析→反馈"闭环。成功标准：下班时间从凌晨12:00降至晚上8:30，日完成率≥70%。开源、非盈利、Web桌面端。

**Next:** Phase 2 - Trigger Mapping

### 2026-05-24 - Phase 2: Trigger Mapping Complete

**Agent:** Unknown (Trigger Map)
**Artifacts Created:**
- `B-Trigger-Map/00-trigger-map.md`
- `B-Trigger-Map/feature-impact-analysis.md`

**Summary:** 识别 3 个用户画像（老刘 Primary、小张 Secondary、李姐 Tertiary），Top 4 Must Have 特性全部指向"规划→执行→打断→调整→延期→提醒"闭环。确定老刘为杠杆用户。

**Next:** Phase 3 - UX Scenarios

### 2026-05-24 — Phase 3: UX Scenarios Complete

**Agent:** Claude (Scenario Outline)
**Scenarios:** 3 scenarios covering 6 pages
**Quality:** Excellent (all scenarios 7/7 across all dimensions)

**Artifacts Created:**
- `C-UX-Scenarios/00-ux-scenarios.md` — Scenario index with coverage matrix
- `C-UX-Scenarios/01-daily-workflow/01-daily-workflow.md` — 老刘的每日工作闭环
- `C-UX-Scenarios/01-daily-workflow/1.1-dashboard/1.1-dashboard.md` — Dashboard 步骤
- `C-UX-Scenarios/01-daily-workflow/1.2-timer/1.2-timer.md` — Timer 步骤
- `C-UX-Scenarios/01-daily-workflow/1.3-schedule/1.3-schedule.md` — Schedule 步骤
- `C-UX-Scenarios/01-daily-workflow/1.4-analytics/1.4-analytics.md` — Analytics 步骤
- `C-UX-Scenarios/02-task-management/02-task-management.md` — 老刘的任务录入与管理
- `C-UX-Scenarios/02-task-management/2.1-tasks/2.1-tasks.md` — Tasks 步骤
- `C-UX-Scenarios/03-settings/03-settings.md` — 老刘的系统配置
- `C-UX-Scenarios/03-settings/3.1-settings/3.1-settings.md` — Settings 步骤

**Key Decisions:**
1. 修订 01 Q8 路径：将 Tasks 移出改为 Scenario 02 独立展开，将 Analytics 加入 01 收工环节，形成完整的"执行→分析→反馈"闭环
2. 每个页面仅归属一个场景（无重复覆盖），6/6 页面全覆盖

**Summary:** 围绕老刘（Primary Persona）创建 3 个场景覆盖全部 6 个页面。01 覆盖核心执行闭环（Dashboard→Timer→Schedule→Analytics），02 聚焦任务录入与四象限管理（Tasks），03 覆盖系统配置（Settings）。全场景质量满分。

**Next:** Phase 4 — UX Design

---

## Design Loop Status

| Scenario | Page | Status | Date |
|----------|------|--------|------|
| 01-daily-workflow | 01.1 Dashboard | specified | 2026-05-24 |
| 01-daily-workflow | 01.2 Timer | specified | 2026-05-24 |
| 01-daily-workflow | 01.3 Schedule | specified | 2026-05-24 |
| 01-daily-workflow | 01.4 Analytics | specified | 2026-05-24 |
| 02-task-management | 02.1 Tasks | specified | 2026-05-24 |
| 03-settings | 03.1 Settings | specified | 2026-05-24 |

---

### 2026-05-24 — Phase 4: UX Design Complete

**Agent:** Claude (UX Design, Suggest mode)
**All 6 pages updated with:** Overview, Journey Context, Layout Sections, Key Components (已有/待加 status), Key States, Interaction Behaviors

**New Features Identified (10 total across all pages):**
1. Dashboard — AI 今日排程概览卡片
2. Dashboard — AI 优先级建议列表
3. Timer — 打断原因记录与 AI 重排程
4. Schedule — AI 排程结果采纳/拒绝
5. Schedule — AI 调整标记显示
6. Analytics — AI 每日洞察卡片
7. Tasks — AI 智能分类（创建时根据文本推荐象限）
8. Tasks — AI 批量分类按钮
9. Settings — 打断缓冲比例滑块
10. Settings — 任务类型时段偏好设置

**Next:** Phase 5 — Agentic Development (Evolution Track)

---

### 2026-05-25 — Phase 5: Agentic Development (Evolution) Complete

**Track:** Evolution（改善法）— 5 步提交
**Branch:** `evolve/ai-scheduling-enhancements`

**Commit 1 — DB + Backend API:**
- `model/session.go` — 新增 `InterruptReason` 字段
- `model/schedule.go` — 新增 `AIAdjusted`、`AdjustmentType` 字段
- `model/setting.go` — 新增 `BufferRatio`、`TaskTimePreferences` 字段，更新默认值
- `ai/prompts.go` — 3 个新 Prompt：`ClassifyByTextPrompt`、`ReschedulePrompt`、`DailyInsightsPrompt`
- `service/ai_service.go` — 扩展 AIService（sessionRepo/scheduleRepo），3 个新方法
- `handler/ai.go` — 3 个新 Handler：`ClassifyTaskByText`、`RescheduleAfterInterrupt`、`GetDailyInsights`
- `handler/timer.go` — `ControlSessionInput` 增加 `interrupt_reason`
- `service/timer_service.go` — `AbandonSession` 存储打断原因
- `router.go` — 3 条新路由 + 1 条修改

**Commit 2 — Store:**
- `types/index.ts` — 新类型 `RescheduleResult`、`DailyInsights`；扩展 `PomodoroSettings`、`PomodoroSession`、`ScheduleEvent`
- `api/client.ts` — 新方法 `classifyTaskByText`、`rescheduleAfterInterrupt`、`getDailyInsights`
- `stores/ai.ts` — 3 个新 Action

**Commit 3 — Settings UI:**
- `views/Settings.vue` — 打断缓冲比例滑块（10/20/30%）、任务类型时段偏好下拉框（管理/开发 → 上午/下午/无所谓）

**Commit 4 — Timer + Schedule:**
- `components/timer/TimerControls.vue` — 放弃计时前确认弹窗
- `views/Timer.vue` — 会话列表展示打断原因
- `stores/timer.ts` — `controlSession` 传递 `interruptReason`

**Commit 5 — Analytics + Tasks:**
- `views/Analytics.vue` — AI 每日洞察卡片（生产力评分、亮点、建议、鼓励语）
- `components/tasks/TaskForm.vue` — 使用 `classifyTaskByText` 替代 `classifyTask('temp')`

**Verification:**
- ✅ `go test ./...` — 所有测试通过
- ✅ `npx vue-tsc --noEmit` — 类型检查通过
- ✅ `vite build` — 前端构建成功
- ✅ 新端点路由正确（`/classify-task-text`、`/reschedule-after-interrupt`、`/daily-insights`）
- ✅ 打断原因正确存储（`interrupt_reason: "meeting"`）
- ✅ 已有 API 向后兼容

**Summary:** 完成 AI 排程增强功能的完整实现，覆盖后端 3 个新 AI 端点和打断原因追踪，前端 6 个页面中 4 个有实质性改进。所有变更向后兼容，已有端点无破坏性修改。

**Next:** 无强制后续步骤。Evolution 循环完成。可返回 Scenario 迭代或进入下一轮改善。

---

### 2026-05-25 — AI Provider 扩展：Claude CLI + Anthropic API

**背景:** OpenAI API 不可达，用户建议直接使用 `claude -p` CLI 命令。

**变更内容:**
- `ai/client.go` — 新增 `CLIClient`（通过 `exec.CommandContext` 调用 `claude -p`）和 `AnthropicClient`（原生 Anthropic Messages API）
- `service/ai_service.go` — `NewAIService` 根据 `cfg.Provider` 选择客户端：`"claude"` → CLIClient、`"anthropic"` → AnthropicClient、默认 → OpenAIClient

**端到端 AI 闭环验证（Claude CLI provider）:**
- ✅ `POST /classify-task-text` — "修复登录页面白屏" → 象限1（重要且紧急），理由："登录是核心功能，白屏意味着用户完全无法使用系统"
- ✅ `POST /reschedule-after-interrupt` — 返回调整后的排程 + 摘要（含task_id/时间/调整类型/原因）
- ✅ `GET /daily-insights` — 返回生产力评分82分 + 峰值时段 + 成就 + 建议 + 鼓励语
- ✅ 3 个端点均返回中文，与产品定位一致

**Commit:** `af7b741` — feat: add Claude CLI and Anthropic API client support
