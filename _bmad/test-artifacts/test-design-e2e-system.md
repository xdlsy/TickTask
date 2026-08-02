---
name: 'TickTask System-Level E2E Test Design'
mode: 'system-level'
date: '2026-06-07'
author: 'Master Test Architect'
status: 'Draft'
inputDocuments:
  - _bmad/prds/prd-ticktask-2026-06-06/prd.md
  - _bmad/architecture.md
  - _bmad/A-Product-Brief/product-brief.md
  - _bmad/C-UX-Scenarios/00-ux-scenarios.md
  - ARCHITECTURE.md
  - _bmad/epics.md
detectedStack: 'fullstack'
testFramework: 'Playwright 1.60 + Go testing'
---

# TickTask — 系统级 E2E 测试设计

**Date:** 2026-06-07
**Author:** Master Test Architect
**Status:** Draft
**Scope:** 覆盖 TickTask 全部功能模块的 E2E 测试用例设计

---

## Executive Summary

**Scope:** 系统级 E2E 测试设计，覆盖 TickTask 全部 6 个功能模块（Dashboard、Timer、Schedule、Analytics、Tasks、Settings）+ 后端 API 层 + WebSocket 实时通信，基于 Playwright 浏览器自动化 + API 级别验证。

**Risk Summary:**

- Total risks identified: 21
- Critical (Score 9): 1
- High (Score 6): 8
- Medium (Score 4-5): 6
- Low (Score 1-3): 6

**Coverage Summary:**

- P0 E2E scenarios: 32（~40-55 hours）
- P1 E2E scenarios: 28（~25-35 hours）
- P2 E2E scenarios: 18（~10-18 hours）
- **Total**: 78 scenarios, ~75-108 hours（~2-3 weeks）

---

## Testability Review

### 🚨 Testability Concerns

| # | Concern | Impact | Mitigation |
|---|---------|--------|------------|
| T01 | AI 功能依赖外部 LLM API，E2E 测试中不可控 | 高 | E2E 中仅测试 AI 状态检查 + UI 交互；AI 生成结果通过 API mock 验证 |
| T02 | WebSocket 实时推送需要后端运行，并行测试间端口冲突 | 中 | 每个测试套件使用独立后端实例 + 测试 DB |
| T03 | 番茄钟计时器涉及真实时间等待 | 中 | 使用短周期（25s work / 5s break）+ API 直接控制状态 |
| T04 | SQLite 单写锁，并发 E2E 测试写入冲突 | 中 | 每个测试 worker 使用独立 DB 文件 |
| T05 | 前端 HMR/路由懒加载导致首次导航延迟 | 低 | 增加导航超时到 30s，使用 `waitForURL` 代替固定等待 |

### ✅ Testability Strengths

| # | Strength | Detail |
|---|----------|--------|
| TS01 | REST API 设计清晰 | 所有功能均有 HTTP 端点，可通过 API 直接设置测试状态 |
| TS02 | 已有 Playwright 基础设施 | playwright.config.ts + factories + api-client 已就绪 |
| TS03 | 工厂模式支持 | TaskFactory / ScheduleFactory 已实现自动清理 |
| TS04 | 前后端分离 | 可独立启动后端进行 API 测试，前端 HMR 热更新 |
| TS05 | 类型系统完备 | TypeScript strict 模式 + 单一类型文件，便于构造测试数据 |

### ASRs (Architecturally Significant Requirements)

| ASR | Description | Status |
|-----|-------------|--------|
| ASR-01 | WebSocket 实时推送与 HTTP API 状态一致性 | ACTIONABLE — 需跨层验证 |
| ASR-02 | AI 排程结果→Schedule 数据→Calendar 渲染端到端一致性 | ACTIONABLE — 需跨层验证 |
| ASR-03 | 番茄钟会话状态机（running→paused→completed）的持久化 | ACTIONABLE — 刷新后恢复 |
| ASR-04 | 任务 CRUD→AI 归类→象限分配→日程生成的数据流 | FYI — 可通过 API 分步验证 |

---

## Risk Assessment

### Critical & High-Priority Risks (Score ≥ 6)

| Risk ID | Category | Description | P | I | Score | Mitigation | Owner | Timeline |
|---------|----------|-------------|---|---|-------|------------|-------|----------|
| R01 | BUS | AI 日程重排返回错误结果，后续日程全部错乱 | 3 | 3 | **9** | E2E 验证：重排前后日程对比 + 用户确认机制 + 回滚 | Dev | 立即 |
| R02 | DATA | 番茄钟状态页面刷新后丢失 | 2 | 3 | **6** | E2E 验证：创建会话→刷新→验证恢复 | Dev | Sprint 1 |
| R03 | BUS | 打断记录失败，AI 无法触发重排 | 2 | 3 | **6** | E2E 验证：打断流程完整性 + 失败降级 | Dev | Sprint 1 |
| R04 | BUS | Dashboard 加载失败时空白无反馈 | 2 | 3 | **6** | E2E 验证：空数据/error 三态处理 | Dev | Sprint 1 |
| R05 | DATA | AI 调整数据未持久化 | 2 | 3 | **6** | E2E 验证：接受调整→刷新→验证持久化 | Dev | Sprint 1 |
| R06 | BUS | AI 象限归类持续错误，用户信任崩塌 | 2 | 3 | **6** | E2E 验证：归类流程 + 手动修正降级 | Dev | Sprint 2 |
| R07 | SEC | API Key 明文暴露 | 2 | 3 | **6** | E2E 验证：设置页面脱敏 + API 响应审计 | Dev | 立即 |
| R08 | TECH | WebSocket 断连后状态同步失败 | 2 | 3 | **6** | E2E 验证：模拟断连→重连→状态一致 | Dev | Sprint 1 |
| R09 | OPS | AI API 不可用时核心功能全部失效 | 2 | 3 | **6** | E2E 验证：mock AI 不可用→降级 UI | Dev | Sprint 2 |

### Medium-Priority Risks (Score 4-5)

| Risk ID | Category | Description | P | I | Score | Mitigation |
|---------|----------|-------------|---|---|-------|------------|
| R10 | BUS | AI 效率建议基于错误分析数据 | 2 | 2 | **4** | 数据校验 + 置信度标注 |
| R11 | PERF | 分析数据查询随时间推移性能下降 | 2 | 2 | **4** | 分页 + 索引验证 |
| R12 | DATA | Day/Week/Month 视图切换数据不一致 | 2 | 2 | **4** | 视图间数据一致性断言 |
| R13 | PERF | AI 分类 debounce 失败致过多 API 调用 | 2 | 2 | **4** | debounce 行为验证 |
| R14 | TECH | 浏览器本地存储与服务器状态不一致 | 2 | 2 | **4** | localStorage 与 API 数据对比 |
| R15 | BUS | Schedule 批量操作静默失败 | 2 | 2 | **4** | 操作结果 Toast 断言 |

### Low-Priority Risks (Score 1-3)

| Risk ID | Category | Description | P | I | Score | Action |
|---------|----------|-------------|---|---|-------|--------|
| R16 | BUS | 日期导航跳转逻辑错误 | 1 | 2 | **2** | 单元测试覆盖 |
| R17 | TECH | 番茄钟计时器精度偏差 | 1 | 2 | **2** | 短周期验证 |
| R18 | DATA | 任务删除后数据残留 | 1 | 2 | **2** | API 验证 |
| R19 | DATA | 设置保存静默失败 | 1 | 2 | **2** | Toast 反馈断言 |
| R20 | BUS | 重复创建任务 | 1 | 1 | **1** | 去重验证 |
| R21 | PERF | 大量任务时象限视图渲染缓慢 | 1 | 2 | **2** | 性能基线测试 |

### Risk Category Legend

- **TECH**: Technical/Architecture — 系统集成、可扩展性
- **SEC**: Security — 访问控制、数据暴露
- **PERF**: Performance — 响应时间、资源限制
- **DATA**: Data Integrity — 数据丢失、不一致
- **BUS**: Business Impact — 用户体验、逻辑错误
- **OPS**: Operations — 部署、外部依赖

---

## NFR Planning

| NFR Category | Requirement / Threshold | Risk Link | Planned Validation | Evidence |
|--------------|------------------------|-----------|-------------------|----------|
| Security | API Key 不在前端明文暴露 | R07 | E2E: 检查设置页面 DOM + API 响应 | Playwright 截图 + 响应审计 |
| Security | 所有 API 通信经 CORS 白名单 | R07 | API 测试: 跨域请求验证 | 测试报告 |
| Performance | 分析查询 < 500ms (1K records) | R11 | E2E: 批量创建数据→查询耗时 | Playwright trace |
| Performance | AI 分类 debounce 300ms | R13 | E2E: 连续输入→观察 API 调用次数 | Network 拦截 |
| Reliability | WebSocket 断连自动重连 < 5s | R08 | E2E: 强制断连→验证重连 | 事件监听 |
| Reliability | AI 不可用时降级行为 | R09 | E2E: mock 503→验证降级 UI | Playwright snapshot |
| Data Integrity | 任务/日程 CRUD 数据一致 | R05, R18 | E2E: 操作→刷新→验证持久化 | DB 查询验证 |
| Accessibility | 核心页面键盘可达 | — | E2E: Tab 导航 + ARIA 验证 | axe-core 扫描 |

**Unknown thresholds (需确认):**
- AI API 响应超时具体 SLA（当前 Go 代码中为 300s）
- WebSocket 心跳间隔和重连最大次数
- 前端性能预算（首次加载、路由切换时间）

---

## Entry Criteria

- [x] Playwright 1.60 + Chromium 已配置
- [x] 后端 dev server 可启动（`make dev`）
- [ ] 测试数据库种子脚本就绪
- [ ] AI mock 服务或 flag 控制（E2E 中不依赖真实 AI）
- [ ] Session factory 补充（当前仅有 task + schedule factories）

## Exit Criteria

- [ ] 所有 P0 E2E 测试通过（32 scenarios）
- [ ] P1 通过率 ≥ 95%
- [ ] R01 (Score 9) 缓解措施已验证
- [ ] 所有 Score ≥ 6 风险有对应 E2E 验证场景
- [ ] API Key 安全性（R07）100% 验证通过
- [ ] WebSocket 重连（R08）100% 验证通过

---

## Test Coverage Plan

> P0/P1/P2 = 优先级/风险分级，不是执行时机。

### E2E Test File Organization

```
frontend/tests/e2e/
├── smoke.spec.ts              # P0 — 基础冒烟测试（已有，扩展）
├── task-crud.spec.ts          # P0 — 任务 CRUD 全流程
├── task-quadrant.spec.ts      # P0 — 四象限视图与操作
├── timer-workflow.spec.ts     # P0 — 番茄钟完整工作流
├── timer-interrupt.spec.ts    # P0 — 打断与 AI 重排
├── schedule-views.spec.ts     # P0 — 日程视图与切换
├── schedule-crud.spec.ts      # P0 — 日程 CRUD
├── schedule-revision.spec.ts  # P0 — AI 日程修订
├── analytics-dashboard.spec.ts # P1 — 分析仪表板
├── settings.spec.ts           # P0 — 设置管理
├── navigation.spec.ts         # P1 — 全局导航
├── websocket.spec.ts          # P1 — WebSocket 实时通信
├── dashboard.spec.ts          # P0 — 仪表板数据展示
├── cross-module.spec.ts       # P1 — 跨模块集成流程
├── error-handling.spec.ts     # P1 — 错误处理与降级
└── accessibility.spec.ts      # P2 — 可访问性基线
```

---

### P0 (Critical) — 32 Scenarios

**Criteria**: 阻塞核心用户旅程 + 高风险 (≥6) + 无替代方案

#### 1. Dashboard 仪表板 (4 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| DASH-E2E-001 | 用户打开首页，Dashboard 正确渲染今日日程时间线 | R04 | 页面可见日程卡片 + 时间轴 |
| DASH-E2E-002 | Dashboard 显示今日统计概览（番茄钟数、完成任务数、专注时长） | R04 | 概览卡片数值与 API 一致 |
| DASH-E2E-003 | Dashboard 空状态：无日程时显示引导 CTA | R04 | 空状态文案 + "开始第一个番茄钟"按钮可见 |
| DASH-E2E-004 | Dashboard 快捷操作：点击 CTA 跳转到 Timer 并自动创建会话 | — | 页面跳转 + Timer 显示运行中 |

#### 2. Tasks 任务管理 (6 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| TASK-E2E-001 | 用户创建任务→任务出现在对应象限 | R06 | API 创建→UI 验证象限位置 |
| TASK-E2E-002 | 用户拖拽任务跨象限→象限更新持久化 | R06 | 拖拽操作→刷新页面→位置保持 |
| TASK-E2E-003 | 用户更新任务标题/描述→变更保存成功 | R18 | 编辑→保存→刷新验证 |
| TASK-E2E-004 | 用户删除任务→任务从列表消失→API 确认 | R18 | 删除→列表无该任务→API 404 |
| TASK-E2E-005 | 创建任务时触发 AI 分类→象限自动分配 | R06 | 创建后→象限与 AI 返回一致 |
| TASK-E2E-006 | AI 分类失败时提供手动选择象限的降级路径 | R06 | Mock AI 500→手动选择可用 |

#### 3. Timer 番茄钟 (6 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| TMR-E2E-001 | 用户启动番茄钟→倒计时正常渲染→状态机正确流转 | R02 | work→completed 状态流转 |
| TMR-E2E-002 | 用户暂停/继续番茄钟→计时正确暂停和恢复 | — | 暂停→时间不变→继续→继续倒计 |
| TMR-E2E-003 | 用户提前完成番茄钟→会话标记为 completed | — | 完成操作→API 状态 = completed |
| TMR-E2E-004 | 番茄钟运行中刷新页面→状态从服务端恢复 | R02 | 刷新→会话仍在运行 |
| TMR-E2E-005 | 用户点击打断→选择原因→AI 触发重排 | R01, R03 | 打断→原因选择→重排触发 |
| TMR-E2E-006 | 完成番茄钟后自动更新关联任务状态 | — | 完成会话→任务 status 更新 |

#### 4. Schedule 日程管理 (8 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| SCH-E2E-001 | 日程 Day 视图正确渲染当日日程块 | R12 | 日程卡片位置、时长与数据一致 |
| SCH-E2E-002 | Day/Week/Month 视图切换→数据一致性 | R12 | 切换后同一日程在不同视图中一致 |
| SCH-E2E-003 | 用户创建日程→出现在日历上 | — | API 创建→UI 可见 |
| SCH-E2E-004 | 用户拖拽移动日程→时间更新持久化 | R05 | 拖拽→刷新→位置保持 |
| SCH-E2E-005 | AI 日程修订→预览变更摘要→确认应用 | R01 | 修订→摘要可见→确认→日程更新 |
| SCH-E2E-006 | AI 修订预览中用户可拒绝→原日程不变 | R01 | 拒绝→刷新→日程原样 |
| SCH-E2E-007 | "接受全部调整"→所有 AI 调整持久化 | R05 | 接受→刷新→调整保持 |
| SCH-E2E-008 | AI 生成日程→7天日程出现在日历 | R01 | 生成→Week 视图可见7天日程 |

#### 5. Settings 设置 (4 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| SET-E2E-001 | 设置页面加载→显示当前配置 | — | 表单值与 API 返回一致 |
| SET-E2E-002 | 修改番茄钟设置→保存成功→即时生效 | — | 修改→保存→新番茄钟使用新设置 |
| SET-E2E-003 | API Key 字段始终密码遮蔽显示 | R07 | DOM 检查 input type="password" |
| SET-E2E-004 | 保存 AI 设置→API 返回中 Key 脱敏 | R07 | 保存→GET 验证仅后4位 |

#### 6. Backend API 验证 (4 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| API-E2E-001 | 任务 CRUD 全生命周期 API 正确 | R18 | Create→Read→Update→Delete 流转 |
| API-E2E-002 | 日程 CRUD + Move API 正确 | R05 | Create→Read→Move→Update→Delete |
| API-E2E-003 | 番茄钟会话状态机 API 正确 | R02 | Create→Pause→Resume→Complete |
| API-E2E-004 | Analytics API 返回正确聚合数据 | R10 | 创建数据→查询验证聚合 |

**P0 Total**: 32 scenarios, ~40-55 hours

---

### P1 (High) — 28 Scenarios

**Criteria**: 关键路径 + 中/高风险

#### 1. Dashboard (3 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| DASH-E2E-005 | Dashboard 加载中显示 loading 骨架屏 | R04 | loading 状态→骨架屏可见 |
| DASH-E2E-006 | Dashboard API 失败时显示错误 + 重试按钮 | R04 | Mock 500→错误提示+重试 |
| DASH-E2E-007 | Dashboard 效率趋势迷你卡片渲染 | R10 | 趋势图可见+数据点正确 |

#### 2. Tasks (4 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| TASK-E2E-007 | 任务列表视图切换→数据一致 | R18 | 四象限↔列表视图切换 |
| TASK-E2E-008 | 任务状态变更为 completed→completed_at 时间戳记录 | R18 | 完成→验证 completed_at |
| TASK-E2E-009 | 批量 AI 分类→所有任务正确归类 | R06 | 多任务→批量分类→结果一致 |
| TASK-E2E-010 | 任务标签正确显示和筛选 | — | 标签→筛选→结果正确 |

#### 3. Timer (4 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| TMR-E2E-007 | 番茄钟 Short Break / Long Break 自动开始 | — | work 完成→break 自动启动 |
| TMR-E2E-008 | 当前任务信息在 Timer 页面显示 | — | 任务标题+象限+预估时间 |
| TMR-E2E-009 | 今日番茄钟历史记录展示 | — | 完成多个→历史列表显示 |
| TMR-E2E-010 | 打断原因选择后的 AI 重排结果展示 | R03 | 打断→重排→新日程可见 |

#### 4. Schedule (5 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| SCH-E2E-009 | Week 视图正确渲染一周日程 | R12 | 7天日程块位置正确 |
| SCH-E2E-010 | Month 视图正确渲染月历 | R12 | 月历网格+日程标记 |
| SCH-E2E-011 | 日程颜色按类型区分显示 | — | task/pomodoro/break 颜色不同 |
| SCH-E2E-012 | 日程编辑弹窗→修改标题/时间/描述 | — | 编辑→保存→验证更新 |
| SCH-E2E-013 | 删除日程→确认弹窗→删除成功 | — | 删除→确认→列表无该日程 |

#### 5. Analytics (4 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| ANALY-E2E-001 | Analytics 概览卡片指标正确 | R10 | 番茄钟数/专注时间/完成任务数 |
| ANALY-E2E-002 | 时间分布图表渲染 | — | 象限分布饼图可见 |
| ANALY-E2E-003 | 趋势图表按时间范围切换 | — | 7天/30天切换→数据更新 |
| ANALY-E2E-004 | 空数据状态显示引导文案 | — | 无数据时→引导提示 |

#### 6. Cross-Module (4 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| CROSS-E2E-001 | 任务创建→AI 排程→日程生成→Timer 执行全流程 | R01, R09 | 端到端全链路 |
| CROSS-E2E-002 | Settings 修改番茄钟时长→Timer 使用新时长 | — | 修改设置→新会话用新时长 |
| CROSS-E2E-003 | Timer 完成会话→Analytics 数据更新 | R10 | 完成番茄钟→分析数据增加 |
| CROSS-E2E-004 | Schedule 修改→Dashboard 时间线同步更新 | — | 修改日程→Dashboard 反映 |

#### 7. Error & Edge Cases (4 scenarios)

| ID | Scenario | Risk | Validation |
|----|----------|------|------------|
| ERR-E2E-001 | AI API 不可用时 Schedule 页面降级提示 | R09 | Mock 503→降级提示可见 |
| ERR-E2E-002 | WebSocket 断连→自动重连→状态恢复 | R08 | 断连→UI 提示→重连→状态一致 |
| ERR-E2E-003 | 网络错误时创建任务→显示错误提示 | — | Mock 网络错误→Toast 提示 |
| ERR-E2E-004 | 并发操作（两个 Tab 同时编辑任务）不产生数据丢失 | R05 | 双 Tab 编辑→保存→验证 |

**P1 Total**: 28 scenarios, ~25-35 hours

---

### P2 (Medium) — 18 Scenarios

**Criteria**: 次要流程 + 低/中风险

#### 1. Navigation & Routing (3 scenarios)

| ID | Scenario | Validation |
|----|----------|------------|
| NAV-E2E-001 | 浏览器前进/后退导航正确 | 前进后退→页面状态正确 |
| NAV-E2E-002 | 刷新任意页面→正确恢复 | 各页面刷新→数据恢复 |
| NAV-E2E-003 | 直接访问深链接→正确渲染 | 直接 URL 访问→页面正常 |

#### 2. Settings Extended (3 scenarios)

| ID | Scenario | Validation |
|----|----------|------------|
| SET-E2E-005 | 设置未保存提示→离开确认 | 修改→导航→确认弹窗 |
| SET-E2E-006 | 重置设置为默认值 | 重置→确认→值恢复默认 |
| SET-E2E-007 | 缓冲比例滑块+预览 | 调整→预览更新→保存 |

#### 3. Accessibility (4 scenarios)

| ID | Scenario | Validation |
|----|----------|------------|
| A11Y-E2E-001 | 所有页面 Tab 键可达主要交互元素 | Tab 序列完整 |
| A11Y-E2E-002 | 任务创建表单键盘可操作 | 无鼠标完成创建 |
| A11Y-E2E-003 | 导航栏键盘可达 | Tab 到各链接 |
| A11Y-E2E-004 | 番茄钟控制按钮有 ARIA 标签 | screen reader 可识别 |

#### 4. Performance (3 scenarios)

| ID | Scenario | Validation |
|----|----------|------------|
| PERF-E2E-001 | 100 个任务时象限视图渲染 < 2s | 批量创建→渲染计时 |
| PERF-E2E-002 | Schedule Week 视图 50 个日程块渲染 < 2s | 批量创建→渲染计时 |
| PERF-E2E-003 | Dashboard 首屏加载 < 3s | 清空数据→加载计时 |

#### 5. Data Edge Cases (5 scenarios)

| ID | Scenario | Validation |
|----|----------|------------|
| EDGE-E2E-001 | 日程时间跨天（23:00-01:00）正确显示 | 创建跨天日程→显示正确 |
| EDGE-E2E-002 | 任务 deadline 过期→视觉标记 | 过期任务→红色标记 |
| EDGE-E2E-003 | 空任务列表→引导 CTA | 无任务→引导文案 |
| EDGE-E2E-004 | 任务描述含特殊字符不破坏布局 | XSS 安全 + 布局正确 |
| EDGE-E2E-005 | 长标题任务在卡片中正确截断显示 | 长文本→省略号 |

**P2 Total**: 18 scenarios, ~10-18 hours

---

## NFR Coverage and Evidence Plan

| NFR | Validation Scenario | Tool/Level | Evidence |
|-----|---------------------|------------|----------|
| Security: API Key 脱敏 | SET-E2E-003, SET-E2E-004 | Playwright E2E | DOM 快照 + API 响应审计 |
| Security: XSS 防护 | EDGE-E2E-004 | Playwright E2E | 注入脚本不执行 |
| Performance: 渲染速度 | PERF-E2E-001~003 | Playwright trace | Trace 时序分析 |
| Reliability: WS 重连 | ERR-E2E-002 | Playwright E2E | 断连→重连事件链 |
| Reliability: AI 降级 | ERR-E2E-001 | Playwright E2E + Route mock | Mock 503→降级 UI |
| Data Integrity | CROSS-E2E-003, SCH-E2E-007 | Playwright + API | 操作→刷新→断言 |
| Accessibility | A11Y-E2E-001~004 | Playwright E2E | Tab 序列 + ARIA 检查 |

---

## Detailed E2E Test Specifications

> 以下为每个 P0 场景的详细 Given-When-Then 规格，可直接转化为 Playwright 测试代码。

### DASH-E2E-001: Dashboard 渲染今日日程时间线

```typescript
// 前置：通过 API 创建今日日程
test('Dashboard 渲染今日日程时间线', async ({ page, scheduleFactory }) => {
  // Given 今日有 3 个日程
  const today = new Date().toISOString().split('T')[0]
  const schedules = await Promise.all([
    scheduleFactory.create({ title: '晨会', start_time: `${today}T09:00:00`, end_time: `${today}T09:30:00` }),
    scheduleFactory.create({ title: '编码', start_time: `${today}T10:00:00`, end_time: `${today}T12:00:00` }),
    scheduleFactory.create({ title: '午餐', start_time: `${today}T12:00:00`, end_time: `${today}T13:00:00` }),
  ])

  // When 用户访问 Dashboard
  await page.goto('/')

  // Then 日程时间线可见，3 个日程卡片按时间排列
  await expect(page.getByText('晨会')).toBeVisible()
  await expect(page.getByText('编码')).toBeVisible()
  await expect(page.getByText('午餐')).toBeVisible()
})
```

### TASK-E2E-001: 创建任务→象限分配

```typescript
test('创建任务后出现在对应象限', async ({ page, taskFactory }) => {
  // Given 用户在 Tasks 页面
  await page.goto('/tasks')

  // When 创建一个 Q1（重要紧急）任务
  const task = await taskFactory.create({
    title: '修复生产环境 Bug',
    quadrant: 1,
    is_important: true,
    is_urgent: true,
  })

  // Then 任务出现在 Q1 象限
  await page.reload()
  await expect(page.getByText('修复生产环境 Bug')).toBeVisible()
  // 验证任务在 Q1 区域内
  const quadrant1 = page.getByTestId('quadrant-1')
  await expect(quadrant1.getByText('修复生产环境 Bug')).toBeVisible()
})
```

### TASK-E2E-002: 拖拽任务跨象限

```typescript
test('拖拽任务跨象限→更新持久化', async ({ page, taskFactory }) => {
  // Given Q2 中有一个任务
  const task = await taskFactory.create({ title: '学习新框架', quadrant: 2 })
  await page.goto('/tasks')
  await expect(page.getByText('学习新框架')).toBeVisible()

  // When 拖拽到 Q1
  const taskCard = page.getByText('学习新框架')
  const quadrant1 = page.getByTestId('quadrant-1')
  await taskCard.dragTo(quadrant1)

  // Then 刷新后任务仍在 Q1
  await page.reload()
  const q1 = page.getByTestId('quadrant-1')
  await expect(q1.getByText('学习新框架')).toBeVisible()
})
```

### TMR-E2E-001: 番茄钟完整工作流

```typescript
test('番茄钟完整工作流：启动→倒计时→完成', async ({ page, apiClient }) => {
  // Given 用户在 Timer 页面
  await page.goto('/timer')

  // When 启动一个 25 秒的 work session（测试用短周期）
  const session = await apiClient.createSession({
    type: 'work',
    planned_duration: 25,
  })

  // Then 倒计时显示，状态为 running
  await expect(page.getByText(/\d{2}:\d{2}/)).toBeVisible()
  await expect(page.getByTestId('timer-status')).toHaveText(/running|进行中/)

  // When 等待 26 秒后倒计时结束
  // （或通过 API 直接控制完成）
  await apiClient.controlSession(session.id, 'complete')

  // Then 状态变为 completed
  await expect(page.getByTestId('timer-status')).toHaveText(/completed|已完成/)
})
```

### TMR-E2E-004: 番茄钟刷新恢复

```typescript
test('番茄钟运行中刷新页面→状态恢复', async ({ page, apiClient }) => {
  // Given 有一个运行中的番茄钟
  const session = await apiClient.createSession({
    type: 'work',
    planned_duration: 1500,
  })

  // When 刷新 Timer 页面
  await page.goto('/timer')
  await expect(page.getByTestId('timer-status')).toHaveText(/running|进行中/)
  await page.reload()

  // Then 番茄钟仍在运行
  await expect(page.getByTestId('timer-status')).toHaveText(/running|进行中/)
  await expect(page.getByText(/\d{2}:\d{2}/)).toBeVisible()
})
```

### TMR-E2E-005: 打断流程

```typescript
test('打断番茄钟→选择原因→AI 重排', async ({ page, apiClient }) => {
  // Given 有一个运行中的番茄钟
  const session = await apiClient.createSession({
    type: 'work',
    planned_duration: 1500,
  })
  await page.goto('/timer')

  // When 点击打断按钮
  await page.getByTestId('interrupt-btn').click()

  // Then 显示打断原因选择弹窗
  await expect(page.getByText(/打断原因|选择原因/)).toBeVisible()

  // When 选择原因 "会议"
  await page.getByText(/会议|meeting/i).click()
  await page.getByRole('button', { name: /确认|确定/ }).click()

  // Then 会话标记为 interrupted，AI 重排触发
  await expect(page.getByTestId('timer-status')).toHaveText(/interrupted|已打断/)
  // AI 重排结果出现（依赖 AI mock 或真实服务）
})
```

### SCH-E2E-005: AI 日程修订预览

```typescript
test('AI 日程修订→预览变更摘要→确认应用', async ({ page, scheduleFactory }) => {
  // Given 今日有多个日程
  const today = new Date().toISOString().split('T')[0]
  await scheduleFactory.create({ title: '任务A', start_time: `${today}T09:00:00`, end_time: `${today}T10:00:00` })
  await scheduleFactory.create({ title: '任务B', start_time: `${today}T10:00:00`, end_time: `${today}T11:00:00` })

  // When 用户在 Schedule 页面点击 "修订日程"
  await page.goto('/schedule')
  await page.getByRole('button', { name: /修订|revise/i }).click()

  // Then 显示修订对话框
  await expect(page.getByText(/修订日程|调整日程/)).toBeVisible()

  // When 输入修订指令并提交
  await page.getByPlaceholder(/输入修订/).fill('把任务A移到下午')
  await page.getByRole('button', { name: /提交|生成/ }).click()

  // Then 显示变更预览摘要
  await expect(page.getByText(/变更摘要|调整预览/)).toBeVisible()
  // 变更高亮标记可见
  await expect(page.getByTestId('ai-adjustment-highlight')).toBeVisible()

  // When 确认应用
  await page.getByRole('button', { name: /确认|应用/ }).click()

  // Then 日程已更新
  await expect(page.getByText('任务A')).toBeVisible()
  // 刷新后保持
  await page.reload()
  await expect(page.getByText('任务A')).toBeVisible()
})
```

### CROSS-E2E-001: 端到端全链路

```typescript
test('任务创建→AI排程→日程生成→Timer执行 全流程', async ({
  page, taskFactory, apiClient
}) => {
  // Step 1: 创建任务
  const task = await taskFactory.create({
    title: '完成项目报告',
    quadrant: 1,
    estimated_time: 60,
  })

  // Step 2: AI 排程（如 AI 可用）
  const aiStatus = await apiClient.getAIStatus()
  if (aiStatus.configured) {
    // When 触发日程生成
    await page.goto('/schedule')
    await page.getByRole('button', { name: /生成日程|AI 排程/ }).click()

    // Then 日程出现在日历
    await expect(page.getByText('完成项目报告')).toBeVisible({ timeout: 60000 })
  }

  // Step 3: 在 Timer 中执行
  await page.goto('/timer')
  const session = await apiClient.createSession({
    task_id: task.id,
    type: 'work',
    planned_duration: 25,
  })
  await expect(page.getByText('完成项目报告')).toBeVisible()

  // Step 4: 完成会话
  await apiClient.controlSession(session.id, 'complete')
  await expect(page.getByTestId('timer-status')).toHaveText(/completed|已完成/)

  // Step 5: 验证 Analytics 更新
  await page.goto('/analytics')
  await expect(page.getByText(/1|完成/)).toBeVisible()
})
```

---

## Execution Strategy

### PR 级别 (每次提交)

- P0 Smoke subset（DASH-E2E-001, TASK-E2E-001, TMR-E2E-001, SCH-E2E-001）
- API 级别快速验证（API-E2E-001~004）
- 运行时间：< 5 分钟

### Nightly (每日)

- 全部 P0 E2E 场景
- 运行时间：~20-30 分钟

### Weekly (每周)

- 全量 P0 + P1 + P2
- 性能基线测试
- 覆盖率报告
- 运行时间：~45-60 分钟

### E2E 测试运行命令

```bash
# PR 级别（smoke）
cd frontend && npx playwright test --grep "@smoke"

# 全量 P0
cd frontend && npx playwright test --grep "@p0"

# 全量 E2E
cd frontend && npx playwright test

# 带 trace 和截图
cd frontend && npx playwright test --trace on --screenshot on
```

---

## Resource Estimates

| Priority | Count | Est. Hours |
|----------|-------|------------|
| P0 | 32 | ~40-55 |
| P1 | 28 | ~25-35 |
| P2 | 18 | ~10-18 |
| **Total** | **78** | **~75-108** |

**Timeline**: ~2-3 weeks（单人全职），含测试基础设施完善（SessionFactory、AI mock、并行隔离）。

---

## Quality Gates

- P0 pass rate = **100%**
- P1 pass rate ≥ **95%**
- P2 pass rate ≥ **90%**（参考性）
- High-risk mitigations (R01-R09): **100% covered by E2E**
- Security test (R07): **100% pass**
- WebSocket reliability (R08): **100% pass**
- AI degradation (R09): **100% pass**

---

## Test Infrastructure Requirements

### 需新增的 Test Support

| Item | Description | Priority |
|------|-------------|----------|
| SessionFactory | PomodoroSession 的 Playwright 工厂类 | P0 |
| AI Mock Server | 拦截 AI API 调用返回预设结果 | P0 |
| Test DB Isolation | 每个测试套件使用独立 SQLite 文件 | P0 |
| Tag Annotations | `@smoke`, `@p0`, `@p1`, `@p2` 标签 | P1 |
| Visual Regression | Playwright screenshot comparison | P2 |
| Axe-core Integration | 可访问性自动化扫描 | P2 |

### 需扩展的 ApiClient 方法

```typescript
// 缺失的 ApiClient 方法需补充：
async updatePomodoroSettings(settings: Partial<PomodoroSettings>)
async updateAISettings(settings: Partial<AISettings>)
async deleteAllSchedules()
async generateSchedule(startTime: string, endTime: string)
async reviseSchedule(prompt: string)
async applyRevision()
async getAnalyticsTrend(days: number)
async getAnalyticsDistribution(start: string, end: string)
async batchClassifyTasks(taskIds: string[])
async classifyTaskText(title: string, description: string)
```

---

## Mitigation Plans

### R01: AI 重排错误结果 (Score: 9)

1. **E2E 验证**: 修订前后日程数据对比断言（时间块不重叠、总时长合理）
2. **预览机制**: 用户确认前 AI 调整仅为建议状态
3. **回滚能力**: 拒绝修订→原日程恢复
4. **测试覆盖**: SCH-E2E-005, SCH-E2E-006, SCH-E2E-007

### R08: WebSocket 断连 (Score: 6)

1. **E2E 验证**: 模拟网络断连→验证自动重连→状态恢复
2. **测试覆盖**: ERR-E2E-002
3. **实现**: Playwright `page.context().setOffline(true)` → `setOffline(false)`

### R09: AI 不可用降级 (Score: 6)

1. **E2E 验证**: Playwright route mock AI 503→验证降级 UI 提示
2. **测试覆盖**: ERR-E2E-001
3. **实现**: `page.route('**/api/ai/**', route => route.fulfill({ status: 503 }))`

---

## Interworking & Regression Matrix

| Source → Target | Interaction | Regression Tests |
|-----------------|-------------|------------------|
| Tasks → Schedule | 任务创建影响 AI 排程 | CROSS-E2E-001 |
| Timer → Schedule | 打断触发 AI 重排 | TMR-E2E-005, SCH-E2E-005 |
| Settings → Timer | 番茄钟参数影响计时 | CROSS-E2E-002 |
| Timer → Analytics | 完成会话更新分析数据 | CROSS-E2E-003 |
| Schedule → Dashboard | 日程修改同步到 Dashboard | CROSS-E2E-004 |
| Settings → AI | AI 设置影响排程结果 | SET-E2E-002 |

---

## Assumptions and Dependencies

### Assumptions

1. E2E 测试运行在有 AI 配置的环境中（或使用 mock）
2. SQLite 适合 E2E 测试场景，每个 worker 使用独立 DB
3. Playwright Chromium 足以覆盖桌面端浏览器
4. 后端 dev server (`make dev`) 可在测试前启动

### Dependencies

1. Playwright 1.60 已安装（`npx playwright install`）
2. Go 后端可编译运行
3. Node.js 环境（`npm install` 成功）

### Risks to Plan

- **Risk**: AI API 响应时间不稳定导致 E2E 超时
  - **Contingency**: AI 相关场景使用 mock + 增加 timeout 到 60s

---

## Appendix: Existing Test Baseline

### Backend Tests (Go)

| File | Module | Status |
|------|--------|--------|
| `ai_service_test.go` | AI Service | Keep |
| `schedule_service_test.go` | Schedule Service | Keep |
| `task_service_test.go` | Task Service | Keep |
| `timer_service_test.go` | Timer Service | Keep |
| `ai_test.go` | AI Handler | Keep |
| `schedule_test.go` | Schedule Handler | Keep |
| `setting_test.go` | Settings Handler | Keep |
| `task_test.go` | Task Handler | Keep |
| `timer_test.go` | Timer Handler | Keep |

### Frontend Tests (Vitest)

| File | Module | Status |
|------|--------|--------|
| Dashboard.test.ts | Dashboard View | Keep, extend |
| Tasks.test.ts | Tasks View | Keep, extend |
| Schedule.test.ts | Schedule View | Keep, extend |
| Analytics.test.ts | Analytics View | Keep, extend |
| Settings.test.ts | Settings View | Keep, extend |
| TimerControls.test.ts | Timer | Keep |
| TimerDisplay.test.ts | Timer | Keep |
| QuadrantView.test.ts | Tasks | Keep |
| TaskCard.test.ts | Tasks | Keep |
| TaskForm.test.ts | Tasks | Keep |
| ai.spec.ts | AI Store | Keep |
| schedule.test.ts | Schedule Store | Keep |
| task.test.ts | Task Store | Keep |
| timer.test.ts | Timer Store | Keep |
| websocket.test.ts | WebSocket | Keep |

### E2E Tests (Playwright)

| File | Status |
|------|--------|
| `example.spec.ts` | Keep, extend to smoke suite |

---

**Generated by**: Master Test Architect — BMad TEA `bmad-testarch-test-design` workflow
**Workflow**: System-Level Mode (Steps 1-5)
**Version**: 1.0
