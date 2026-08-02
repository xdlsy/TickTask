# TickTask 修订日程功能 — 开发流程报告

**生成日期：** 2026-06-05
**分支：** `evolve/ai-scheduling-enhancements`
**状态：** Story 1.1 实现完成 + Code Review 通过，Story 1.2 待开发

---

## 一、概述

本报告记录了 TickTask "修订日程"功能的完整开发流程：从产品需求（PRD）到 Epic/Story 拆分、后端代码实现、测试、Code Review 和代码优化。整个过程由 BMad Method 工作流驱动，Claude 作为 Product Manager → Tech Lead → Developer → Reviewer 全角色协作完成。

---

## 二、完整 Skill 调用链条

```
┌──────────────────────────────────────────────────────────────────────────┐
│                      BMad Method 完整工作流链路                             │
├──────────────────────────────────────────────────────────────────────────┤
│                                                                           │
│  ① /bmad-prd                                                              │
│    ┌──────────────┐                                                       │
│    │  bmad-prd    │  Discovery → Fast Path → Finalize (6 步)               │
│    └──────┬───────┘                                                       │
│           │ prd.md + .decision-log.md                                     │
│           ▼                                                               │
│  ② /bmad-create-epics-and-stories                                         │
│    ┌──────────────────────────────┐                                       │
│    │ bmad-create-epics-and-stories│  step-01 → step-04 (4 步)              │
│    └──────────────┬───────────────┘                                       │
│                   │ epics.md (1 Epic / 2 Stories / FR 全覆盖映射)          │
│                   ▼                                                       │
│  ③ bmad-create-story "Story 1.1"                                          │
│    ┌──────────────────┐                                                   │
│    │ bmad-create-story│  discover-inputs → 代码库深度分析 → template 填充   │
│    └──────┬───────────┘                                                   │
│           │ 1-1-revise-api.md (7 AC / 7 Tasks / 25 Subtasks / Dev Notes)  │
│           ▼                                                               │
│  ④ bmad-dev-story (Story 1.1)                                             │
│    ┌──────────────────┐                                                   │
│    │ bmad-dev-story   │  Task 1 → Task 7: red-green-refactor 循环          │
│    └──────┬───────────┘                                                   │
│           │ 5 文件修改 / +200 行代码 / 9 个新测试                           │
│           ▼                                                               │
│  ⑤ code-review                                                            │
│    ┌──────────┐                                                           │
│    │code-review│  Phase 0 (diff) → Phase 1 (4 angles × agents) → Phase 2   │
│    └──────┬───┘                                                           │
│           │ 6 项发现（全部清理/简化级别，无 Bug）                              │
│           ▼                                                               │
│  ⑥ 修复                                                                    │
│    ├─ Fix #1: 移除 ~150 行死代码                                            │
│    ├─ Fix #2: 提取 currentWeekRange() 公共函数                              │
│    ├─ Fix #3: 简化 computeDiff() 消除重复遍历                                │
│    ├─ Fix #4: 合并 prompt builders → buildSkillPrompt()                    │
│    └─ Fix #5: ICS 行尾 \n → \r\n (RFC 5545)                               │
│                                                                           │
│  ✅ Story 1.1 就绪                                                        │
│                                                                           │
└──────────────────────────────────────────────────────────────────────────┘
```

### 各 Skill 职责与产出

| 阶段 | Skill | 角色 | 输入 | 核心产出 |
|------|-------|------|------|----------|
| ① | `bmad-prd` | PM | 用户需求描述 | `prd.md`（5 FR + 3 NFR + API 规格） |
| ② | `bmad-create-epics-and-stories` | Tech Lead | PRD + UX Scenarios | `epics.md`（1 Epic / 2 Stories / 全覆盖映射） |
| ③ | `bmad-create-story` | Tech Lead | Epic + 现有代码库 | `1-1-revise-api.md`（7 AC + 25 Subtasks + Dev Notes） |
| ④ | `bmad-dev-story` | Developer | Story 文件 | 工作代码 + 9 个测试 |
| ⑤ | `code-review` | Reviewer | Git diff | 6 项发现 → 全部修复 |

---

## 三、生成产物清单

| # | 产物 | 路径 | 状态 |
|---|------|------|------|
| 1 | 产品需求文档 | `_bmad/prds/prd-ticktask-2026-06-05/prd.md` | ✅ final |
| 2 | 决策日志 | `_bmad/prds/prd-ticktask-2026-06-05/.decision-log.md` | ✅ 完成 |
| 3 | Epic & Story 分解 | `_bmad/epics.md` | ✅ 完成 |
| 4 | Story 1.1 开发规格 | `_bmad/implementation-artifacts/1-1-revise-api.md` | ✅ review |
| 5 | Story 1.1 后端代码 | `backend/internal/` (5 文件) | ✅ 实现完成 |
| 6 | 开发流程报告 | `REVISE-SCHEDULE-DEVELOPMENT-REPORT.md` | ✅ 本报告 |
| 7 | Story 1.2 开发规格 | 待创建 | 🔲 待进行 |

---

## 四、需求逐层细化（信息流）

```
PRD                           EPICS                        STORY 1.1                   IMPLEMENTATION
────                          ──────                       ─────────                   ──────────────
FR1 修订按钮 ──┐              ┌─ Epic 1: 修订日程 ─┐
FR2 输入框  ──┤              │  FR1-FR5 全覆盖     │
FR3 执行流程 ──┤  全部映射到  │  NFR1-NFR3          ├─ 7 AC (Given/When/Then) ──→  5 文件 / +200 行
FR4 预览确认 ──┤ ──────────→ │  UX-DR1~5           │   7 Tasks / 25 Subtasks       9 新测试 / 0 回归
FR5 校验    ──┘              └─────────────────────┘   Dev Notes (Pipeline 对照表、
                                                       Diff 算法、ICS 格式模板、
NFR1 UX        ──┐              ┌─ 依赖关系 ─┐          代码行号引用、文件修改清单)
NFR2 数据安全  ──┤  全部映射到  │ Story 1.1  ─┤                                    → 代码实现
NFR3 向后兼容  ──┘              │    ↓       │     Story 1.2: 前端修订交互界面
                                │ Story 1.2  ─┘     (待创建)
UX-DR1~5      ──全部映射到 Story 1.2
```

**每层精度递增：**

| 层级 | 回答的问题 | 精度 |
|------|-----------|------|
| PRD | 做什么？为什么做？ | 功能边界 + 用户价值 |
| Epics | 按什么顺序做？依赖关系？ | Epic/Story 拆分 + FR 覆盖映射 |
| Story | 具体怎么做？架构约束？ | AC + 任务拆分 + 架构参考 + 代码行号 |
| Code | 可工作吗？测试覆盖？ | 可执行代码 + 9 个单元测试 |

---

## 五、关键技术决策

### 5.1 PRD 阶段（6 项）

| # | 决策 | 说明 |
|---|------|------|
| 1 | 修订范围为整周（周一至周日） | 与现有 GenerateSchedule 行为一致 |
| 2 | 预览-确认两阶段机制 | 用户选择方案 B：预览变更差异，确认后才写 DB |
| 3 | 自然语言 prompt 输入 | 灵活度最高，符合 AI-native 理念 |
| 4 | 复用 auto-schedule 校验脚本 | 保持一致性，不重复造轮子 |
| 5 | 两阶段 API 设计 | `/revise` 返回预览差异，`/revise/apply` 确认应用 |
| 6 | 按钮在"生成日程"右侧 | 工具栏自然延伸 |

### 5.2 Epics 阶段（2 项）

| # | 决策 | 说明 |
|---|------|------|
| 1 | 单 Epic（不拆分） | 5 个 FR 构成完整用户操作链，拆分会导致文件反复改动 |
| 2 | 前后端分 Story | Story 1.1（Go）和 Story 1.2（Vue/TS）文件集完全分离 |

### 5.3 Story 1.1 实现阶段（4 项）

| # | 决策 | 说明 |
|---|------|------|
| 1 | 复用 GenerateSchedule pipeline | `runClaudeStreamJSON()` 不修改，`ParseICS()` 复用 |
| 2 | 内存保存 ICS 基线 + Claude 覆写 + 对比差异 | 适配 skill 实际行为（覆写 `schedule.ics` 而非写 `schedule_revised.ics`） |
| 3 | `computeDiff()` 按 title + date 建索引 | 准确区分 moved/added/removed，跨天同名任务视为独立事件 |
| 4 | ReviseSchedule 不写 DB | 预览阶段仅返回差异数据，保证数据安全 |

### 5.4 Code Review 修复阶段（5 项）

| # | 修复 | 类型 |
|---|------|------|
| 1 | 移除 ~150 行死代码（ReadRevisedICS, CleanRevisedICS, generateWeekICS, writeICS, buildSchedulePrompt） | 清理 |
| 2 | 提取 `currentWeekRange()` 公共函数，消除 3 处重复 | 简化 |
| 3 | 提取 `indexEventsByTitleDate()`，merge 计数循环 | 简化 |
| 4 | 合并 prompt builders → `buildSkillPrompt()` | 简化 |
| 5 | ICS 行尾 `\n` → `\r\n`（RFC 5545 标准） | 规范 |

---

## 六、Story 1.1 实现详情

### 核心 API

```
POST /api/schedules/revise
  Request:  { "prompt": "把代码评审移到下午" }
  Response: { "applied": false, "summary": "共调整 3 个任务...", "changes": [...], "events": [] }

POST /api/schedules/revise/apply
  Request:  无 body
  Response: { "applied": true, "events": [...] }
```

### ReviseSchedule Pipeline 与 GenerateSchedule 的差异

```
                            GenerateSchedule              ReviseSchedule
                            ────────────────              ───────────────
1. 计算周范围                  currentWeekRange()           currentWeekRange() ← 提取公共函数
2. WebSocket 广播             ✅ 相同                       ✅ 相同
3. 写日程基线                  ❌ 无                        ✅ WriteScheduleICS()  ← NEW
4. 写 config 文件              WriteTodoJSON + HabitMD    ✅ 相同
5. Claude CLI                 auto-schedule skill          revise-schedule skill  ← 新 skill
6. 读 ICS                     单文件                       双文件（内存保存原始 + 读覆写后）← NEW
7. 差异计算                    ❌ 无                        ✅ computeDiff()         ← NEW
8. 持久化                     ✅ 立即写 DB                   ❌ 不写 DB                ← 差异
9. 返回                       events + summary             changes + summary         ← 差异
```

### 文件变更清单

| 文件 | 操作 | 新增内容 |
|------|------|----------|
| `backend/internal/service/schedule_service.go` | UPDATE | 3 类型 + 5 函数（ReviseSchedule, ApplyRevision, computeDiff, currentWeekRange, buildSkillPrompt, indexEventsByTitleDate） |
| `backend/internal/service/config_writer.go` | UPDATE | WriteScheduleICS + escapeICS |
| `backend/internal/api/handler/schedule.go` | UPDATE | ReviseWithAI + ApplyRevision |
| `backend/internal/api/router.go` | UPDATE | POST /revise + POST /revise/apply |
| `backend/internal/service/schedule_service_test.go` | UPDATE | 9 个测试（computeDiff 7 场景 + WriteScheduleICS 往返 + escapeICS） |

### 测试覆盖

```
=== RUN  TestComputeDiff_Moved          — PASS
=== RUN  TestComputeDiff_Added          — PASS
=== RUN  TestComputeDiff_Removed        — PASS
=== RUN  TestComputeDiff_Mixed          — PASS
=== RUN  TestComputeDiff_NoChanges      — PASS
=== RUN  TestComputeDiff_AllAdded       — PASS
=== RUN  TestComputeDiff_DifferentDays  — PASS
=== RUN  TestWriteScheduleICS_RoundTrip — PASS
=== RUN  TestEscapeICS_SpecialChars     — PASS
---
PASS  9/9 new tests | 0 regressions
```

---

## 七、当前状态与下一步

```
✅ 已完成
  ① bmad-prd                    → prd.md + .decision-log.md
  ② bmad-create-epics-and-stories → epics.md (1 Epic / 2 Stories)
  ③ bmad-create-story (1.1)      → 1-1-revise-api.md (25 Subtasks)
  ④ bmad-dev-story (1.1)         → 后端代码 + 9 测试 / 0 回归
  ⑤ code-review                  → 6 项发现 → 全部修复

🔲 待进行
  ⑥ bmad-create-story (1.2)      → Story 1.2 前端开发规格
  ⑦ bmad-dev-story (1.2)         → 前端代码实现
  ⑧ code-review (1.2)            → 前端代码审查
  ⑨ 端到端集成测试                → 全功能验证
```

---

## 八、快速启动

```bash
# 创建 Story 1.2（前端规格）
bmad-create-story "Story 1.2: 前端修订交互界面"

# 实现 Story 1.2
bmad-dev-story

# 运行后端测试
cd backend && go test ./... -v

# 启动开发环境
make dev

# 测试新 API
curl -X POST http://localhost:8080/api/schedules/revise \
  -H "Content-Type: application/json" \
  -d '{"prompt": "把代码评审移到下午"}'
```

---

*本报告由 Claude (BMad Method) 在 2026-06-05 自动生成 — 覆盖完整的 ① PRD → ② Epics → ③ Story → ④ Dev → ⑤ Review → ⑥ Fix 闭环*
