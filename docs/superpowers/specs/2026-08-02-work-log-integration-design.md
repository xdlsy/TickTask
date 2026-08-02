# Work-Log 集成到 TickTask — 设计文档

- **日期**：2026-08-02
- **状态**：已通过 brainstorming，待实现计划
- **作者**：lsy + Claude（superpowers:brainstorming）
- **关联 skill**：`~/.claude/skills/work-log/`（保留，本集成不改动它）

---

## 1. 概述

把 `work-log` skill 的「脑暴 → 四维结构化日报 → 周期报告分层汇总」理念，原生集成到 TickTask 网页，形成一个独立「工作日志」页面。

核心价值：保留 skill「输入低成本、输出高结构、绝不编造」的精神；用 TickTask 自有数据（任务、番茄钟、日程）进一步降低输入成本；用 SQLite + AI service 取代 markdown 文件 + Claude Code harness 的运行形态。

## 2. 关键决策（brainstorming 中确认）

| 决策点 | 结论 |
|---|---|
| 集成深度 | **完整原生集成**——SQLite 存、复用已有 AI service、4 类周期报告全做、原 skill 不动 |
| 日报生成方式 | **AI 拆条提炼**——贴脑暴 → AI 拆四维 → 预览可编辑 → 落库 |
| 与现有数据联动 | **预填今日活动**作为脑暴提示（已完成任务 + 番茄钟会话 + 日程） |
| 周期报告范围 | **周/月/半年/年 4 类全做** |
| 报告触发 | **纯手动按钮**（"生成本周/本月/本半年/本年报告"）；不做边界自动检测弹窗 |
| 页面布局 | **左时间轴 + 右详情** |
| 数据存储粒度 | **结构化四维**——work_logs + work_items + work_reports 三表 |

## 3. 模块依赖（沿用 TickTask 拓扑）

```
internal/model/work_log.go        ← WorkLog, WorkItem, WorkReport
        ↓
internal/repository/work_log_repo.go   ← interface + 私有 struct，返回 interface
        ↓
internal/service/work_log_service.go   ← AI 拆条、周期算法、报告分层汇总
        ↑ (依赖 ai_service)
        ↓
internal/api/handler/work_log.go   ← 薄 HTTP 层
```

`work_log_service` 复用 `service.AIService`（已有 OpenAI 兼容客户端）做拆条与汇总，不重复造轮子。

## 4. 数据模型（GORM / SQLite）

```go
// internal/model/work_log.go

type WorkLog struct {
    ID           string     `gorm:"primaryKey;size:36"`
    Date         string     `gorm:"uniqueIndex;size:10"` // YYYY-MM-DD
    Summary      string     `gorm:"size:500"`            // 今日小结
    RawBrainDump string     `gorm:"type:text"`           // 用户原始脑暴
    CreatedAt    time.Time
    UpdatedAt    time.Time

    Items []WorkItem `gorm:"foreignKey:WorkLogID;references:ID"`
}

type WorkItem struct {
    ID            string `gorm:"primaryKey;size:36"`
    WorkLogID     string `gorm:"index;size:36"`
    Seq           int    // 顺序
    Title         string `gorm:"size:200"`
    Content       string `gorm:"size:1000"`       // 做了什么
    ProblemSolved string `gorm:"size:1000"`       // 解决了什么问题
    Result        string `gorm:"size:1000"`       // 已经产生了什么结果（"（待补充）" 表示缺）
    Impact        string `gorm:"size:1000"`       // 对后续的影响
}

type WorkReportType string

const (
    ReportWeekly   WorkReportType = "weekly"
    ReportMonthly  WorkReportType = "monthly"
    ReportHalfYear WorkReportType = "halfyear"
    ReportYearly   WorkReportType = "yearly"
)

type WorkReport struct {
    ID          string         `gorm:"primaryKey;size:36"`
    Type        WorkReportType `gorm:"uniqueIndex:idx_report_period,priority:1"`
    PeriodKey   string         `gorm:"uniqueIndex:idx_report_period,priority:2;size:20"` // 2026-W31 / 2026-07 / 2026-H1 / 2026
    StartDate   string         `gorm:"size:10"`
    EndDate     string         `gorm:"size:10"`
    SummaryJSON string         `gorm:"type:text"` // 结构化字段（按 type 不同 schema）
    MissingDays string         `gorm:"size:200"`  // 点名缺失天，逗号分隔
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

**关键设计点：**

- `WorkLog.Date` 唯一索引 → 一天一条；二次记录走 PUT（items 全量替换）+ summary 重写
- `WorkItem` 四维独立字段，"（待补充）"作为合法字符串值
- `WorkReport.SummaryJSON` 而非 markdown 字符串：周报 4 字段（核心工作 / 主要进展 / 遗留问题 / 下周关注）；月报/半年报/年报按需扩展
- 表名复数（`work_logs` / `work_items` / `work_reports`），与 `tasks` / `sessions` 一致

## 5. REST API

| Method | Path | 说明 |
|---|---|---|
| GET | `/api/work-logs/today/context` | 返回今日预填上下文（完成任务、番茄钟会话、日程） |
| POST | `/api/work-logs/structure` | 输入脑暴 → AI 拆条返回结构化预览，**不落库** |
| POST | `/api/work-logs` | 保存日报（含 items）；同日已存在 → 409 Conflict，响应体含 `existing_work_log` 供前端走 PUT |
| GET | `/api/work-logs?from=&to=` | 列表（左时间轴） |
| GET | `/api/work-logs/:date` | 详情 |
| PUT | `/api/work-logs/:date` | 编辑（items 全量替换） |
| POST | `/api/work-reports/generate` | 手动生成报告 `{type, periodKey?, force?}`，未 force 时遇已存在则 error |
| GET | `/api/work-reports?type=&periodKey=` | 读报告详情 |
| GET | `/api/work-reports?type=` | 报告列表（按类型） |

## 6. AI Prompt 设计

`internal/ai/work_log_prompts.go`：

**日报拆条**——输入：脑暴 + 今日上下文；输出：严格 JSON
```json
{
  "items": [
    {
      "title": "...",
      "content": "...",
      "problem_solved": "...",
      "result": "...",
      "impact": "..."
    }
  ],
  "summary": "今日小结 2-3 句"
}
```

**system prompt 关键约束**：

- 绝不编造。无法从脑暴中落实的维度，整维只输出"（待补充）"三个字；不要复述 + 括注凑数
- result 维度必须给出具体数字 / 产出 / 结论；模糊时写"（待补充）"
- title 不超过 20 字；四维各不超过 300 字

**报告汇总**——分层 prompt：周报读 JSON 化的 work_items；月报读周报 JSON；半年报读月报 JSON；年报读月报或半年报 JSON。每层 prompt 都明确"只读下一层数据"。

## 7. 周期算法（`service/work_log_calendar.go`）

```go
func WeeklyRange(t time.Time) (start, end time.Time)   // ISO 周一~周日
func WeeklyKey(t time.Time) string                      // "2026-W31"
func MonthlyRange(t time.Time) (start, end time.Time)
func MonthlyKey(t time.Time) string                     // "2026-07"
func HalfYearRange(t time.Time) (start, end time.Time)  // H1=1~6月, H2=7~12月
func HalfYearKey(t time.Time) string                    // "2026-H1"
func YearlyRange(t time.Time) (start, end time.Time)
func YearlyKey(t time.Time) string                      // "2026"
```

## 8. 分层汇总（硬规则：上一层只读下一层）

| 报告 | 读源 | AI 任务 |
|---|---|---|
| weekly | 该周 7 天的 `work_items`（SQL 直查） | 按主题归并去重 → 4 字段 |
| monthly | 该月所有 `work_reports WHERE type='weekly'`（4–5 篇） + 月内**不属于任何完整周**的日期（如月初 / 月末零散天）的 `work_items` | 月度总结（4 字段） |
| halfyear | 该半年 6 篇 `work_reports WHERE type='monthly'` | 重大成果 / 趋势 / 关键问题 |
| yearly | 12 篇月报或 2 篇半年报 | 年度总结 |

**关键不变式（service 层强制）：**

- 月报不读全部原始 work items，只读周报 + 月内不属于任何完整周的零散天 items（"孤儿 items"）
- 半年报不读周报或日报，只读月报
- 年报不读周报或日报，只读月报 / 半年报
- 违反 → service 直接返回 error，防止上下文爆炸
- 守门方式：`// INVARIANT:` 注释 + 表驱动单测

**MissingDays**：报告生成时 SQL 查周期内所有 `work_logs.date`，与日期范围 diff，写入 `WorkReport.MissingDays`。

**覆盖语义**：报告可重新生成（覆盖）。`POST /generate` 的 `force` 默认 false，遇已存在直接 error；前端先 ElMessageBox.confirm 后再带 `force=true` 重发。

## 9. 前端结构

### 路由

```ts
// frontend/src/router/index.ts
{ path: '/work-log', name: 'WorkLog', component: () => import('@/views/WorkLog.vue') }
```

### 顶部导航（`App.vue` `navItems`）

```ts
{ path: '/work-log', name: 'work-log', label: '工作日志', icon: Document }
```
插入位置：在"分析"和"设置"之间。

### Pinia store（`frontend/src/stores/workLog.ts`）

```ts
export const useWorkLogStore = defineStore('workLog', () => {
  const logs = ref<WorkLog[]>([])
  const currentLog = ref<WorkLog | null>(null)
  const todayContext = ref<TodayContext | null>(null)
  const reports = ref<Record<WorkReportType, WorkReport[]>>({
    weekly: [], monthly: [], halfyear: [], yearly: []
  })

  async function fetchLogs(from: string, to: string)
  async function fetchLog(date: string)
  async function fetchTodayContext()
  async function structureBrainDump(text: string, context: TodayContext)
  async function saveWorkLog(payload: SaveWorkLogInput)
  async function generateReport(type: WorkReportType, periodKey?: string)
  async function fetchReports(type: WorkReportType)
  async function fetchReport(type, periodKey)
})
```

### 页面布局（`frontend/src/views/WorkLog.vue`）

```
┌─────────────────────────────────────────────────────┐
│  工作日志                              [今日 ▼ 日期] │
├──────────────────┬──────────────────────────────────┤
│  时间轴 (240px)   │  详情区                           │
│                  │                                  │
│  ─── 日报 ───    │  ┌─────────────────────────────┐ │
│  8/2 周六 (今) ← │  │ 今日预填                     │ │
│  8/1 周五       │  │ ✓ 任务 A、B   🍅 3 个 (50min) │ │
│  7/31 周四      │  │ 📅 日程 2 项已完成             │ │
│  7/30 周三      │  └─────────────────────────────┘ │
│                  │                                  │
│  ─── 周报 ───    │  ┌─────────────────────────────┐ │
│  W31 (本周) ←   │  │ 脑暴输入                     │ │
│  W30           │  │ [textarea]                   │ │
│                  │  │              [AI 拆条 →]     │ │
│  ─── 月报 ───    │  └─────────────────────────────┘ │
│  7月           │                                  │
│                  │  ┌ 预览/编辑（拆条后显示）─────┐ │
│  ─── 半年报 ───  │  │ Item 1: [标题][四维表单]    │ │
│  H1 2026       │  │ Item 2: ...                 │ │
│                  │  │ [+ 加一条]                  │ │
│  ─── 年报 ───    │  │ 小结: [textarea]            │ │
│  2025          │  │              [保存日报]       │ │
│                  │  └─────────────────────────────┘ │
│                  │                                  │
│  [+ 生成报告 ▼]  │  (报告视图：从时间轴选报告节点时 │
│                  │   右侧切换为只读报告详情)        │
└──────────────────┴──────────────────────────────────┘
```

### 子组件（`frontend/src/components/work-log/`）

| 组件 | 职责 |
|---|---|
| `Timeline.vue` | 左时间轴，按类型分组；点击节点 emit `select` |
| `TodayContextCard.vue` | 预填卡片（任务 / 番茄 / 日程） |
| `BrainDumpInput.vue` | textarea + "AI 拆条"按钮，emit `structured` |
| `WorkItemEditor.vue` | 单条四维表单，v-model 双向 |
| `WorkItemList.vue` | items 列表容器，支持增删拖拽排序 |
| `ReportActions.vue` | "生成本周 / 本月 / 本半年 / 本年报告"下拉按钮 |
| `ReportDetail.vue` | 报告只读视图，按 type 切换 layout |

### 交互关键点

1. AI 拆条不直接保存——拆条后落到 `WorkItemList.vue` 的可编辑预览，用户改完点"保存日报"才发 POST
2. 同日二次保存——检测到当天已有日报时，保存按钮文案变"更新今日日报"，PUT 语义（items 全量替换）
3. 报告视图切换——左时间轴点击"周报 W31"节点 → 右侧切到 `ReportDetail.vue`；顶部不再显示脑暴区
4. 生成报告按钮——下拉菜单 4 项，每项点击后 ElMessageBox.confirm；已存在报告则提示"将覆盖"，确认后 force=true 重发

## 10. 错误处理与不变式

| 红线 | 实现位置 |
|---|---|
| 绝不编造——凑不出具体产出的维度，整维输出"（待补充）" | `ai/work_log_prompts.go` system prompt + service schema 校验 |
| 绝不静默覆盖日报 | handler `POST /api/work-logs` 检测 `WorkLog.Date` 已存在 → 400 + 返回 existing；前端走 PUT |
| 绝不静默覆盖报告 | `POST /api/work-reports/generate` 已存在 → 必须前端先 confirm；service `force` 默认 false，遇已存在直接 error |
| 月/半年/年报绝不读全部原始日报 | service 层强制 + `// INVARIANT:` 注释 + 表驱动单测守门 |
| AI 失败不 fallback 给假数据 | service 返回 502 → 前端 ElMessage.error；不写空 item 进库 |
| AI 拆条 JSON 非法 | service 严格 schema 校验，失败返回 502 + 原始响应片段供排查 |
| 日期非法（如 `2026-02-30`） | handler 解析失败返回 400 |

## 11. 测试策略

### 后端（Go standard testing）

`backend/internal/service/work_log_service_test.go`：

- 表驱动：四维提炼边界（脑暴完整 / 缺"结果"维度 / 全部缺失）
- 表驱动：周期算法（周日 / 月末 / 6·30 / 12·31 / 普通周三的 periodKey 与 range）
- 分层汇总不变式：mock repo 让月报读到 work_items → 必须 fail
- 同日二次保存：返回 existing → 前端走 PUT 路径
- AI 拆条 JSON 非法：返回 502 不写库

`backend/internal/api/handler/work_log_test.go`：

- 用既有 `mocks_test.go` 风格的 manual mock 实现 `WorkLogRepository` interface
- 覆盖每个端点的 happy path + 主错误路径

`backend/internal/repository/work_log_repo_test.go`：

- 真接 SQLite（跟随 `schedule_repo.go` 测试风格）
- CRUD + 按 date range 查询 + 报告 unique 索引

### 前端（Vitest + @vue/test-utils）

| 测试文件 | 覆盖 |
|---|---|
| `stores/workLog.spec.ts` | store 每个 action 的成功 + 错误路径 |
| `components/work-log/Timeline.spec.ts` | 渲染分组、点击 emit select |
| `components/work-log/BrainDumpInput.spec.ts` | 输入 + 拆条按钮 emit |
| `components/work-log/WorkItemEditor.spec.ts` | v-model 四维表单 |
| `views/WorkLog.spec.ts` | 日报 / 报告视图切换、保存流程 |

类型检查：`npx vue-tsc --noEmit` 必须通过。

## 12. 落地阶段（每个阶段独立 PR / 独立验证）

| 阶段 | 内容 | 验证 |
|---|---|---|
| **M1** | 后端骨架：model + repo + 三表 migration + service 占位（不含 AI）+ handler 端点骨架 + 测试 | `go test ./...` 通过；空端点可调通 |
| **M2** | 日报 AI 拆条：接 `ai_service.GenerateWorkLogItems()`，prompt 模板，`/structure` 端点 | 单测覆盖四维边界 |
| **M3** | 前端日报页：路由 + 导航入口 + store + Timeline + TodayContextCard + BrainDumpInput + WorkItemEditor + 保存流程 | `npm run build` + 手测保存 |
| **M4** | 周期算法 + 报告生成：`work_log_calendar.go` + 分层汇总 + generate 端点 + 不变式测试 | 不变式测试守门 |
| **M5** | 前端报告视图：ReportActions + ReportDetail + 报告节点接入 Timeline | 手测生成 / 覆盖报告 |
| **M6** | 收尾：E2E 走查、文档更新（`AGENTS.md` 加新模块、`docs/knowledge/` 加蓝图） | 文档 review |

## 13. 文档影响（M6 完成时更新）

- `AGENTS.md` 仓库结构 / 模块依赖图：加入 work_log 模块
- `ARCHITECTURE.md` Code map：加 3 个新条目（model / repository / service / handler）
- `frontend/src/types/index.ts`：加 `WorkLog` `WorkItem` `WorkReport` `WorkReportType` `TodayContext` 类型
- 不动 `~/.claude/skills/work-log/`（保留原 skill 给 Claude Code 用户）

## 14. 范围与非目标（YAGNI）

**不做：**

- 不做边界自动检测弹窗——纯手动按钮触发报告生成
- 不做导出为 skill markdown 格式——两套生态独立，不强行打通
- 不做多用户 / 团队共享——个人功能
- 不做后台 cron 自动生成报告——AI 额度不可控
- 不做"AI 失败时降级为手动表单"——失败提示用户重试即可，不做双模式
- 不做 undo / 回收站——PUT 全量替换，用户误删自负（用 confirm 兜底覆盖报告场景）

## 15. 后续

实现计划由 `superpowers:writing-plans` 产出，按 M1~M6 阶段逐步落地。
