---
workflowStatus: 'completed'
totalSteps: 5
stepsCompleted: ['step-01-detect-mode', 'step-02-load-context', 'step-03-risk-and-testability', 'step-04-coverage-plan', 'step-05-generate-output']
lastStep: 'step-05-generate-output'
nextStep: ''
lastSaved: '2026-05-25'
---

# Test Design: Epic 1 - TickTask 全功能模块

**Date:** 2026-05-25
**Author:** Master Test Architect
**Status:** Draft

---

## Executive Summary

**Scope:** Epic-level test design covering 6 functional modules — Dashboard, Timer, Schedule, Analytics, Tasks, Settings, plus backend (Go) API layer.

**Risk Summary:**

- Total risks identified: 19
- High-priority risks (≥6): 9 (1 Critical Score=9, 8 High Score=6)
- Critical categories: BUS (5), DATA (3), SEC (1), TECH (1), OPS (1)

**Coverage Summary:**

- P0 scenarios: 27 (~30-45 hours)
- P1 scenarios: 30 (~20-30 hours)
- P2 scenarios: 22 (~8-15 hours)
- **Total effort**: ~58-90 hours (~2-3 weeks)

---

## Not in Scope

| Item | Reasoning | Mitigation |
|------|-----------|------------|
| E2E 浏览器测试 (Playwright/Cypress) | 项目未配置浏览器自动化工具，当前使用 Vitest + jsdom | 关键用户路径通过 Component 测试 + API 集成测试覆盖；后续可引入 Playwright |
| 契约测试 (Pact) | 项目为单体架构，无微服务间通信 | 通过 API 集成测试覆盖服务契约验证 |
| 移动端/响应式测试 | 产品定位为 Web 桌面端 | 当前仅验证桌面视口 |
| 第三方集成 (Stripe 等) | 无支付/外部服务集成 | N/A |

---

## Risk Assessment

### High-Priority Risks (Score ≥6)

| Risk ID | Category | Description | P | I | Score | Mitigation | Owner | Timeline |
|---------|----------|-------------|---|---|-------|------------|-------|----------|
| R01 | BUS | AI 打断重排返回错误结果，导致后续日程全部错乱 | 3 | 3 | **9** | 重排结果验证层 + 用户确认机制 + 回滚能力 | Dev | 立即 |
| R02 | DATA | 番茄钟状态页面刷新后丢失，工作会话数据丢失 | 2 | 3 | **6** | 状态持久化到 localStorage + 服务端同步 | Dev | Sprint 1 |
| R03 | BUS | 打断记录失败，AI 无法触发重排，用户卡住 | 2 | 3 | **6** | 打断记录离线队列 + 重试机制 + 手动重排入口 | Dev | Sprint 1 |
| R04 | BUS | AI 排程数据加载失败，Dashboard 空白，用户无法开始一天 | 2 | 3 | **6** | Loading/Empty/Error 三态处理 + 降级显示手动排程入口 | Dev | Sprint 1 |
| R05 | DATA | AI 调整数据未持久化，刷新后调整丢失 | 2 | 3 | **6** | 调整数据实时保存 + 未确认状态标记持久化 | Dev | Sprint 1 |
| R06 | BUS | AI 象限归类持续错误，用户信任崩塌 | 2 | 3 | **6** | 归类置信度阈值 + 低置信度时让用户手动选择 + 反馈机制 | Dev | Sprint 2 |
| R07 | SEC | API Key 在前端明文存储/泄露 | 2 | 3 | **6** | 后端加密存储 + 前端仅存 masked 版本 + HTTPS 传输 | Dev | 立即 |
| R08 | TECH | 后端（Go）零测试覆盖，所有 API 无自动化保障 | 3 | 2 | **6** | 补充 Go 后端单元测试 + 集成测试 + CI 集成 | Dev | Sprint 1 |
| R09 | OPS | Claude API 不可用时无降级方案，核心功能全部失效 | 2 | 3 | **6** | 规则引擎 fallback + API 状态监控 + 用户提示 | Dev | Sprint 2 |

### Medium-Priority Risks (Score 4-5)

| Risk ID | Category | Description | P | I | Score | Mitigation | Owner |
|---------|----------|-------------|---|---|-------|------------|-------|
| R10 | BUS | AI 效率建议基于错误分析数据给出误导建议 | 2 | 2 | **4** | 数据校验层 + 建议置信度标注 | Dev |
| R11 | PERF | 历史分析数据查询随时间推移性能下降 | 2 | 2 | **4** | 数据分页/聚合 + 查询性能监控 | Dev |
| R12 | DATA | Day/Week/Month 视图切换时数据不一致 | 2 | 2 | **4** | 统一数据源 + 视图间一致性校验 | Dev |
| R13 | PERF | AI 分类 debounce 失败导致过多 API 调用 | 2 | 2 | **4** | 前端 debounce 机制 + API 调用频率限制 | Dev |
| R14 | TECH | WebSocket 断连后状态同步失败 | 2 | 2 | **4** | 自动重连 + 状态全量同步 + 断连提示 | Dev |

### Low-Priority Risks (Score 1-3)

| Risk ID | Category | Description | P | I | Score | Action |
|---------|----------|-------------|---|---|-------|--------|
| R15 | BUS | 日期导航跳转逻辑错误 | 1 | 2 | **2** | 单元测试覆盖 |
| R16 | TECH | 番茄钟计时器精度偏差 | 1 | 2 | **2** | 精度校验测试 |
| R17 | BUS | Schedule "接受全部调整"按钮静默失败 | 1 | 3 | **3** | 操作结果 Toast 反馈 |
| R18 | DATA | 任务保存失败数据丢失 | 1 | 3 | **3** | 保存前本地缓存 + 失败重试 |
| R19 | DATA | 设置保存静默失败 | 1 | 2 | **2** | 保存状态明确反馈 |

### Risk Category Legend

- **TECH**: Technical/Architecture (flaws, integration, scalability)
- **SEC**: Security (access controls, auth, data exposure)
- **PERF**: Performance (degradation, resource limits)
- **DATA**: Data Integrity (loss, corruption, inconsistency)
- **BUS**: Business Impact (UX harm, logic errors, trust)
- **OPS**: Operations (deployment, config, monitoring, external dependencies)

---

## NFR Planning

| NFR Category | Requirement / Threshold | Risk Link | Planned Validation | Evidence Needed |
|--------------|------------------------|-----------|-------------------|----------------|
| Security | API Key 加密存储，不可前端明文暴露 | R07 | 集成测试：验证 API Key 存储/传输加密 | 测试报告 + 代码审计 |
| Security | HTTPS 传输所有 API 通信 | R07 | 集成测试：验证 TLS 配置 | 安全扫描报告 |
| Performance | 分析数据查询响应 <500ms (1000条记录) | R11 | Go benchmark + 前端性能测试 | benchmark 报告 |
| Performance | AI API 分类 debounce 有效性 | R13 | 组件测试：验证 debounce 行为 | 测试报告 |
| Reliability | AI API (Claude) 不可用时的降级行为 | R09 | 集成测试：mock AI 不可用 → fallback 触发 | 测试报告 |
| Reliability | WebSocket 断连自动重连 | R14 | 集成测试：模拟断连 → 重连 → 状态同步 | 测试报告 |
| Data Integrity | 任务/日程 CRUD 数据一致性 | R05, R18 | 集成测试：CRUD 操作后验证数据库状态 | 测试报告 |

**Unknown thresholds:**
- 分析数据查询性能具体 SLA（需确认数据量预期）
- AI API 响应超时阈值（需确认 Claude API 典型延迟）
- WebSocket 重连最大尝试次数（需产品确认）

---

## Entry Criteria

- [ ] 测试环境（本地 dev server + test DB）可访问
- [ ] 测试数据工厂（task/schedule factories）就绪
- [ ] Vitest 配置完成（含 jsdom 环境）
- [ ] Go 测试框架就绪（标准 testing + testify）
- [ ] Mock AI API 服务可用

## Exit Criteria

- [ ] 所有 P0 测试通过（27 scenarios）
- [ ] P1 测试通过率 ≥95%
- [ ] R01 (Score 9) 缓解措施已验证
- [ ] 所有 Score ≥6 风险有缓解方案并实施
- [ ] 后端测试覆盖从 0 提升至 ≥60%
- [ ] 前端整体测试覆盖率 ≥80%

---

## Test Coverage Plan

> P0/P1/P2 = priority/risk classification, NOT execution timing.

### P0 (Critical)

**Criteria**: Blocks core journey + High risk (≥6) + No workaround

| Module | Requirement | Test Level | Risk Link | Count | Owner |
|--------|-------------|------------|-----------|-------|-------|
| Dashboard | AI 排程日程时间线渲染 | Component | R04 | 1 | Dev |
| Dashboard | CTA 开始第一个番茄钟 → Timer | Component | — | 1 | Dev |
| Dashboard | 获取今日日程 API | Integration | R04 | 1 | Dev |
| Timer | 番茄钟倒计时渲染 + 状态机 | Component+Unit | — | 2 | Dev |
| Timer | 打断按钮 → 原因选择 → AI 重排 | Component | R01, R03 | 2 | Dev |
| Timer | 打断 API + 重排结果验证 | Integration | R01 | 1 | Dev |
| Timer | 计时状态页面刷新持久化 | Integration | R02 | 1 | Dev |
| Schedule | AI 调整摘要 + 高亮标记 | Component | R01 | 2 | Dev |
| Schedule | "接受全部调整"持久化 | Integration | R05 | 1 | Dev |
| Schedule | Day 视图渲染 | Component | — | 1 | Dev |
| Analytics | 概览卡片指标 | Component | R10 | 1 | Dev |
| Analytics | 获取分析数据 API | Integration | R10 | 1 | Dev |
| Tasks | 四象限视图 + 任务录入 | Component | R06 | 2 | Dev |
| Tasks | 创建任务 API + AI 归类 API | Integration | R06 | 2 | Dev |
| Settings | API Key 脱敏显示 | Component | R07 | 1 | Dev |
| Settings | 保存设置 API | Integration | — | 1 | Dev |
| Backend | 任务 CRUD 逻辑 | Unit | R08 | 1 | Dev |
| Backend | 日程生成/调整逻辑 | Unit | R08 | 1 | Dev |
| Backend | 番茄钟会话管理 | Unit | R08 | 1 | Dev |
| Backend | AI API Client 集成 | Integration | R08, R09 | 1 | Dev |
| Backend | AI API 降级 Fallback | Integration | R09 | 1 | Dev |

**Total P0**: 27 scenarios, ~30-45 hours

### P1 (High)

| Module | Requirement | Test Level | Risk Link | Count | Owner |
|--------|-------------|------------|-----------|-------|-------|
| Dashboard | Loading/Empty/Error 三态 | Component | R04 | 3 | Dev |
| Dashboard | 效率趋势迷你卡片 | Component | R10 | 1 | Dev |
| Dashboard | 空日程 API | Integration | R04 | 1 | Dev |
| Timer | 暂停/继续/提前完成 | Component | — | 3 | Dev |
| Timer | 当前任务信息显示 | Component | — | 1 | Dev |
| Timer | 打断 API 失败离线队列 | Integration | R03 | 1 | Dev |
| Timer | 完成番茄钟更新任务状态 | Integration | — | 1 | Dev |
| Schedule | Week/Month 视图 + 切换 | Component | — | 3 | Dev |
| Schedule | "撤销 AI 调整" + API | Component+Integration | R01 | 2 | Dev |
| Schedule | Day/Week/Month 数据一致性 | Integration | R12 | 1 | Dev |
| Analytics | 时间分布/趋势图表 | Component | — | 2 | Dev |
| Analytics | 时间筛选刷新 | Component | — | 1 | Dev |
| Analytics | AI 建议 API | Integration | R10 | 1 | Dev |
| Tasks | 确认 AI 归类 → 入象限 | Component | R06 | 1 | Dev |
| Tasks | 拖拽跨象限覆盖 | Component | — | 1 | Dev |
| Tasks | AI 归类失败手动降级 | Integration | R06 | 1 | Dev |
| Settings | 番茄钟/工作时间表单 | Component | — | 2 | Dev |
| Settings | 缓冲比例滑块 + 预览 | Component | — | 1 | Dev |
| Settings | 加载设置 API | Integration | — | 1 | Dev |
| Backend | 分析数据计算逻辑 | Unit | R08 | 1 | Dev |
| Backend | WebSocket 连接管理 | Integration | R14 | 1 | Dev |
| Backend | 数据库迁移验证 | Integration | R08 | 1 | Dev |

**Total P1**: 30 scenarios, ~20-30 hours

### P2 (Medium)

| Module | Requirement | Test Level | Risk Link | Count | Owner |
|--------|-------------|------------|-----------|-------|-------|
| Dashboard | 概览卡片计数 / 日期导航 | Unit | R15 | 2 | Dev |
| Timer | 今日会话历史 / 计时精度 | Component+Unit | R16 | 2 | Dev |
| Schedule | 空日程状态 / 日期跳转逻辑 | Component+Unit | — | 2 | Dev |
| Analytics | 打断分析 / 空数据引导 / 骨架屏 | Component | — | 3 | Dev |
| Analytics | 效率评分 / 时间占比 / AI建议API | Unit+Integration | R10 | 3 | Dev |
| Tasks | 列表视图 / 视图切换 / 空状态 / 删除API | Component+Integration | R18 | 4 | Dev |
| Settings | 未保存指示 / 成功Toast / 重置 / 导入导出 | Component+Integration | R19 | 4 | Dev |

**Total P2**: 22 scenarios (no P3 for this epic), ~8-15 hours

---

## NFR Coverage and Planned Evidence

| NFR Category | Requirement | Validation Scenario | Tool/Level | Evidence |
|--------------|-------------|---------------------|------------|----------|
| Security | API Key 加密存储 | SETT-INT-003: 验证 API Key 在 DB 中加密、前端仅 masked 版本 | Go test | 测试报告 |
| Performance | 分析查询 <500ms | 1000条记录下查询响应时间 | Go benchmark | benchmark JSON |
| Performance | AI 分类 debounce | TASK-UNIT-002: debounce 300ms 行为 | Vitest | 测试报告 |
| Reliability | AI 不可用 Fallback | BACK-INT-003: mock AI 超时 → 规则引擎 fallback | Go test | 测试报告 |
| Reliability | WebSocket 重连 | BACK-INT-002: 模拟断连 → 自动重连 → 状态同步 | Go test | 测试报告 |
| Data Integrity | CRUD 一致性 | 创建/更新/删除后验证 DB 状态 | Go integration | 测试报告 |

---

## Execution Strategy

### PR (每次提交)

所有 Unit + Component 测试（Vitest, <5 min）。

### Nightly (每日)

全部 P0 + P1 集成测试 + Go backend 测试（~15-20 min）。

### Weekly (每周)

全量 P0-P2 + Go benchmark + 覆盖率报告。

**Philosophy**: 优先在 PR 中运行所有快速测试；仅将耗时/昂贵的测试推迟到 Nightly/Weekly。

---

## Resource Estimates

| Priority | Count | Est. Hours |
|----------|-------|------------|
| P0 | 27 | ~30-45 |
| P1 | 30 | ~20-30 |
| P2 | 22 | ~8-15 |
| **Total** | **79** | **~58-90** |

**Timeline**: ~2-3 周（单人全职），含测试基础设施搭建。

---

## Quality Gate Criteria

- P0 pass rate = **100%**
- P1 pass rate ≥ **95%**
- P2 pass rate ≥ **90%** (informational)
- High-risk mitigations (R01-R09): **100% complete** or approved waivers
- Frontend coverage ≥ **80%**
- Backend coverage ≥ **60%** (from 0%)
- Security test (R07): **100% pass**

---

## Mitigation Plans

### R01: AI 打断重排返回错误结果 (Score: 9)

**Strategy:**
1. 后端实现重排结果验证层（检查时间块不重叠、总时长合理）
2. 前端展示调整摘要，用户确认后才生效
3. 提供"撤销"按钮，支持回滚到打断前排程
4. 记录每次 AI 调整日志用于排查

**Owner:** Dev
**Timeline:** Sprint 1
**Status:** Planned
**Verification:** TIMER-INT-001, SCHED-COMP-002, SCHED-INT-001

### R08: 后端零测试覆盖 (Score: 6)

**Strategy:**
1. 引入 Go testing + testify 框架
2. 为所有 API handler 编写单元测试
3. 为 service 层编写集成测试（含 SQLite 测试 DB）
4. 在 CI 中集成 Go test

**Owner:** Dev
**Timeline:** Sprint 1
**Status:** Planned
**Verification:** BACK-UNIT-001~004, BACK-INT-001~003

### R09: Claude API 不可用无降级 (Score: 6)

**Strategy:**
1. 实现规则引擎 fallback（基于优先级+预估时间简单排程）
2. API 调用添加超时 + 重试机制
3. 前端在 AI 不可用时展示降级提示

**Owner:** Dev
**Timeline:** Sprint 2
**Status:** Planned
**Verification:** BACK-INT-003

---

## Assumptions and Dependencies

### Assumptions

1. AI API (Claude) 可用性 ≥99%，延迟 <5s
2. SQLite 适合当前单用户场景，无需迁移至 PostgreSQL
3. 用户数据量在 10K 任务/年 以内
4. 当前 Vitest + jsdom 足以覆盖 UI 组件测试

### Dependencies

1. Claude API key 可用 — 已有
2. Go 开发环境就绪 — 已有
3. CI 环境支持 Go test + Vitest — 待配置

### Risks to Plan

- **Risk**: Claude API 响应格式变更
  - **Impact**: AI 排程/归类功能异常
  - **Contingency**: 后端响应解析层加版本校验 + fallback

---

## Interworking & Regression

| Service/Component | Impact | Regression Scope |
|-------------------|--------|------------------|
| Timer → Schedule | 打断后 AI 调整影响日程视图 | Schedule 视图测试 + AI 调整 API 测试 |
| Tasks → Schedule | 任务入池后影响 AI 排程 | AI 排程生成测试 |
| Settings → Timer | 番茄钟参数影响计时器行为 | Timer 状态机测试 |
| Settings → AI | 缓冲比例影响排程结果 | AI 排程参数化测试 |

---

## Appendix A: Existing Tests (Baseline)

| File | Module | Status |
|------|--------|--------|
| `DayView.test.ts` | Schedule | Keep, may extend |
| `WeekView.test.ts` | Schedule | Keep, may extend |
| `MonthView.test.ts` | Schedule | Keep, may extend |
| `EventForm.test.ts` | Schedule | Keep |
| `TimerDisplay.test.ts` | Timer | Keep, add interrupt scenarios |
| `TimerControls.test.ts` | Timer | Keep |
| `QuadrantView.test.ts` | Tasks | Keep, add AI classify scenarios |
| `ListView.test.ts` | Tasks | Keep |
| `TaskCard.test.ts` | Tasks | Keep |
| `TaskForm.test.ts` | Tasks | Keep, simplify to minimal fields |
| `task.test.ts / task.spec.ts` | Tasks | Keep, add AI API scenarios |
| `timer.test.ts / timer.spec.ts` | Timer | Keep, add interrupt flow |
| `ai.spec.ts` | AI | Keep, add classification/interrupt tests |
| `schedule.test.ts` | Schedule | Keep, add adjustment scenarios |
| `app.test.ts` | App | Keep |
| `websocket.test.ts` | WebSocket | Keep, add reconnect scenarios |
| `client.test.ts` | API | Keep |

## Appendix B: Knowledge Base References

- `risk-governance.md` — Risk scoring matrix, gate decision engine
- `probability-impact.md` — Probability × Impact scale and thresholds
- `test-levels-framework.md` — Unit / Integration / E2E selection guide
- `test-priorities-matrix.md` — P0–P3 prioritization with risk alignment

---

**Generated by**: BMad TEA Agent — Test Architect Module
**Workflow**: `bmad-testarch-test-design` (Epic-Level Mode)
**Version**: 4.0 (BMad v6)
