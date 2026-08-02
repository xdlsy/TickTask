# 工作日志 · 合并表单（Add + AI Batch Entry）

**Spec date**: 2026-08-03
**Branch**: `evolve/work-log-quick-entry`（沿用当前分支）
**Status**: 设计已确认，进入实施

---

## 1. 背景与动机

工作日志界面当前有两套独立录入路径：

- **顶部 `QuickEntryForm`** — 单行表单，必填 `日期/活动/时段/象限`，"添加"立即落库（写入 `source='manual'` WorkItem，刷新 TodayPanorama 表）
- **底部 `WorkItemList` + `WorkItemEditor`** — 来自 AI 脑暴拆条的草稿列表，含 `标题/内容/问题/结果/影响` 5 字段，"保存日报"按钮整体落库

两条路径写到 WorkItem 的不同字段集合，是独立保存语义。用户希望"两者合一"：每条工作用一份统一表单录入，包含 4 必填（日期/时段/活动/象限）+ 4 可选（内容/问题/结果/影响）字段；标题与活动合并（一个字段）；AI 脑暴产出的多条走单独的批量入口，由用户补齐必填字段后批量入库。

## 2. 设计目标 / 非目标

**目标**：

- 一个统一的主表单 `WorkItemForm.vue`（布局 C：卡片分组），支持 `add` / `edit` 双模式
- 一个表格式批量编辑组件 `BatchTableEditor.vue`，承接 AI 脑暴拆条后的多条草稿
- 删除 `QuickEntryForm.vue` / `WorkItemList.vue` / `WorkItemEditor.vue` 三个旧组件
- 主表单：4 必填 + 4 可选字段；活动字段同时写入 `WorkItem.activity` 与 `WorkItem.title`
- AI prompt 调整：拆条输出去掉 `title`，只生成 4 维（content/problem_solved/result/impact）
- 一次性 SQLite migration：旧 manual items 的 `title` 从 `activity` 回填

**非目标**：

- 不改 WorkItem 表 schema（`title` / `activity` 都保留）
- 不改周期报告生成逻辑
- 不引入新的批量入库 API（复用现有 POST `/api/work-logs/:date/items` 循环调用）
- 不做"今日小结"独立卡片（summary 保留在 BatchTableEditor 底部，与 AI 草稿同期保存）
- 不改 `TodayContextCard` / `Timeline` / `BrainDumpInput` 三个组件

## 3. 架构（组件树）

```
WorkLog.vue (orchestrator)
├── Timeline.vue                      [复用，不变]
├── WorkItemForm.vue                  [新建·取代 QuickEntryForm]
│   └── props: date, mode('add'|'edit'), itemId?, initial?
│       emits: added, saved, cancel
├── TodayPanorama.vue                 [复用，不变]
├── TodayContextCard.vue              [复用，不变]
├── BrainDumpInput.vue                [复用，不变]
├── BatchTableEditor.vue              [新建·取代 WorkItemList + WorkItemEditor]
│   └── props: items (DraftItem[]), summary
│       emits: update:items, update:summary, save, discard
└── ReportDetail.vue                  [复用，不变]
```

**删除清单**：
- `frontend/src/components/work-log/QuickEntryForm.vue`
- `frontend/src/components/work-log/QuickEntryForm.spec.ts`
- `frontend/src/components/work-log/WorkItemList.vue`
- `frontend/src/components/work-log/WorkItemEditor.vue`

**编辑入口**：TodayPanorama 的"编辑"按钮 → WorkLog.vue 设置 `editingItemId` → 顶部 WorkItemForm 切换到 `edit` 模式（机制已存在，无新增状态）。

## 4. 数据模型与字段映射

### 4.1 后端 model（不动 schema）

`backend/internal/model/work_log.go` 中的 `WorkItem` 字段全部保留。语义重定义：

| 字段 | 旧语义 | 新语义 |
|---|---|---|
| `Title` | AI 拆条生成的 20 字标题 | `Activity` 的镜像（service 层同步写入） |
| `Content` | AI 拆条生成 | 用户在主表单或批量表中填的"内容"（可选） |
| `ProblemSolved`/`Result`/`Impact` | AI 拆条生成 | 同上（可选） |
| `Activity` | 仅 manual 录入填 | 主字段，所有来源都填（manual 主表单 / 批量入库） |
| `StartTime`/`EndTime`/`Quadrant` | 仅 manual | 同上 |
| `Source` | `'manual'` 或 `'ai'` | 主表单 = `'manual'`；批量入库 = `'manual'`（用户已补齐必填，等同于手动录入） |

> **关键决定**：AI 批量入库的 items `source` 设为 `'manual'`（不是 `'ai'`），因为用户已补齐必填字段、能进入 TodayPanorama。旧 AI items（`source='ai'`，无 activity）保留原状，不出现在 TodayPanorama（继续被 `todayManualItems` 过滤掉），但仍在 WorkLog.items 中参与报告生成。

### 4.2 后端 DTO 扩展（不重命名，只加字段）

`backend/internal/service/work_log_service.go`：

```go
// 名称保留（避免 churn），只加 4 个可选字段
type CreateQuickEntryInput struct {
    Activity      string `json:"activity" binding:"required"`
    StartTime     string `json:"start_time" binding:"required"`
    EndTime       string `json:"end_time" binding:"required"`
    Quadrant      int    `json:"quadrant" binding:"required,min=1,max=4"`
    Content       string `json:"content"`        // 新增·可选
    ProblemSolved string `json:"problem_solved"` // 新增·可选
    Result        string `json:"result"`         // 新增·可选
    Impact        string `json:"impact"`         // 新增·可选
}

type UpdateQuickEntryInput struct {
    Activity      *string `json:"activity,omitempty"`
    StartTime     *string `json:"start_time,omitempty"`
    EndTime       *string `json:"end_time,omitempty"`
    Quadrant      *int    `json:"quadrant,omitempty" binding:"omitempty,min=1,max=4"`
    Content       *string `json:"content,omitempty"`        // 新增·可选
    ProblemSolved *string `json:"problem_solved,omitempty"` // 新增·可选
    Result        *string `json:"result,omitempty"`         // 新增·可选
    Impact        *string `json:"impact,omitempty"`         // 新增·可选
}
```

**Service 层关键不变式**：
- `AddQuickEntry`：构造 `WorkItem` 时 `Title: in.Activity`（同步）+ 4 个可选字段直传
- `UpdateQuickEntry`：若 `in.Activity != nil`，updates map 同时写 `title` 和 `activity`；4 个可选字段使用指针语义（nil = 不动，`*ptr = ""` = 清空）
- 新增 service 方法 `UpdateSummary(date, summary string) error`：仅更新 WorkLog.summary 字段（避免 PUT 全量替换 items）

### 4.2.1 新增 API 端点

`PATCH /api/work-logs/:date/summary`

```jsonc
// 请求
{ "summary": "今日 2-3 句小结" }

// 响应
{ "ok": true }
```

`backend/internal/api/handler/work_log.go` 新增 handler `UpdateSummary`，路由注册加一行 `router.PATCH("/work-logs/:date/summary", ...)`。

Service 实现：

```go
func (s *WorkLogService) UpdateSummary(date string, summary string) error {
    if _, err := time.Parse("2006-01-02", date); err != nil {
        return fmt.Errorf("invalid date: %w", err)
    }
    return s.repo.UpdateWorkLogSummary(date, summary)
}
```

Repository 新增 `UpdateWorkLogSummary(date, summary string) error`，若 WorkLog 不存在则返回 `ErrNotFound`（前端可在批量入库前先 ensure WorkLog 存在，或 service 自动建）。

### 4.3 前端 types（`frontend/src/types/index.ts`）

```ts
// 名称保留（与后端 DTO 对齐），只加可选字段
export interface CreateQuickEntryInput {
  activity: string
  start_time: string
  end_time: string
  quadrant: Quadrant
  content?: string       // 新增·可选
  problem_solved?: string // 新增·可选
  result?: string        // 新增·可选
  impact?: string        // 新增·可选
}

export interface UpdateQuickEntryInput {
  activity?: string
  start_time?: string
  end_time?: string
  quadrant?: Quadrant
  content?: string       // 新增·可选
  problem_solved?: string // 新增·可选
  result?: string        // 新增·可选
  impact?: string        // 新增·可选
}

// WorkItem 接口字段不变（注释更新说明 title = activity 镜像）

// StructuredItem（AI 拆条输出）去掉 title 字段
export interface StructuredItem {
  // title 移除——activity 由用户在批量表中补齐，title 后端从 activity 同步
  content: string
  problem_solved: string
  result: string
  impact: string
}

// 新增·用于 PATCH /work-logs/:date/summary
export interface UpdateSummaryInput {
  summary: string
}
```

### 4.4 前端 store + API client

`frontend/src/stores/workLog.ts`：

- 名称保留：`addQuickEntry` / `updateQuickEntry` / `deleteQuickEntry` 不改名
- 新增 `addWorkItemsBatch(date, items: CreateQuickEntryInput[])`：循环调用 `api.appendWorkItem`（顺序、不并发，避免 SQLite 写冲突），返回 `{successCount: number, failureIndices: number[]}`；失败的 items 保留在调用方 draftItems
- 新增 `updateSummary(date, summary: string)`：调用 `api.updateWorkLogSummary(date, summary)`，失败 ElMessage.warning（不阻塞主流程）

`frontend/src/api/client.ts`：

- 类型签名扩展（`CreateQuickEntryInput` / `UpdateQuickEntryInput` 加可选字段，类型同名）
- 新增 `updateWorkLogSummary(date, summary)` → `client.patch('/work-logs/${date}/summary', { summary })`

### 4.5 后端 handler 调整

`backend/internal/api/handler/work_log.go`：

- DTO 类型 `createQuickEntryInput` / `updateQuickEntryInput` 保留名称，加 4 个可选字段
- 函数名 `AddQuickEntry` / `UpdateQuickEntry` / `DeleteQuickEntry` 保留（HTTP 路由不变）
- 新增 handler `UpdateSummary`，处理 `PATCH /api/work-logs/:date/summary`
- 路由 `router.go` 增加一行注册：`workLogRoutes.PATCH("/work-logs/:date/summary", workLogHandler.UpdateSummary)`

## 5. 数据流（三条保存路径）

### Path 1：主表单添加（WorkItemForm mode='add'）

```
用户填表（4 必填 + 0~4 可选）
  → 点击 "+添加"
  → store.addQuickEntry(date, payload)
  → POST /api/work-logs/:date/items
  → service.AddQuickEntry: 自动建 WorkLog（若不存在）+ 写入 WorkItem(title=activity)
  → fetchLog(date) 刷新 currentLog
  → TodayPanorama 通过 todayManualItems computed 自动刷新
  → WorkItemForm emit('added') → form reset（保留 date/start_time/end_time/quadrant 默认值，activity 清空）
```

### Path 2：主表单编辑（WorkItemForm mode='edit'）

```
TodayPanorama 点编辑 → emit('edit', itemId)
  → WorkLog.vue 设置 editingItemId
  → WorkItemForm 切到 edit 模式，通过 initial prop 回填字段（从 store.currentLog.items 找）
  → 用户改 → 点击"保存"
  → store.updateQuickEntry(date, itemId, payload)
  → PATCH /api/work-logs/:date/items/:itemId
  → service.UpdateQuickEntry: activity 变更时同步 title
  → fetchLog(date)
  → emit('saved') → WorkLog.vue 清空 editingItemId → WorkItemForm 切回 add 模式
```

### Path 3：AI 批量入库（BatchTableEditor）

```
用户在 BrainDumpInput 输入脑暴 → 点击 "AI 拆条"
  → store.structureBrainDump(text)
  → POST /api/work-logs/structure (AI 不落库)
  → 返回 StructuredWorkLog { items: [{content, problem_solved, result, impact}], summary }
  → WorkLog.vue 写入 draftItems（每条 default: {activity:'', start_time:'09:00', end_time:'10:00', quadrant:2, ...}）
  → BatchTableEditor 渲染 N 行表格
  → 用户补齐 activity/start_time/end_time/quadrant（必填）+ 编辑可选字段（已预填）
  → 点击"批量入库"
  → 校验全部行：activity 非空 + start<end + quadrant 1~4
  → store.addWorkItemsBatch(date, items)
  → 循环 POST /api/work-logs/:date/items（顺序，不并发，避免 SQLite 写冲突）
  → store.updateSummary(date, summary)（独立调用 PATCH /work-logs/:date/summary）
  → 全部成功：emit('save') → WorkLog.vue 清空 draftItems + draftSummary → fetchLog 刷新
  → 部分失败：ElMessage 报错，失败的 indices 在 BatchTableEditor 中保留红色边框，已成功的从 draftItems 移除
```

## 6. 今日小结（Summary）归属

**决定**：summary 保留在 BatchTableEditor 底部（沿用 WorkItemList 现有位置），通过独立端点保存。

- 用户脑暴后，AI 返回 summary 预填
- BatchTableEditor 底部 textarea 显示 summary，可编辑
- 点击"批量入库"时：先循环 POST items，再 PATCH `/api/work-logs/:date/summary` 保存 summary（两步独立，互不阻塞）
- 纯 manual 录入路径（不脑暴）：summary 不展示；WorkLog.summary 保持空字符串
- 移除 `WorkLog.vue` 中 watch `store.currentLog` 把 currentLog.items 写回 draftItems 的逻辑（旧流程痕迹，新设计中 BatchTableEditor 只承接 AI 草稿，不承接已落库 items）
- 新端点 PATCH `/api/work-logs/:date/summary` 仅更新 WorkLog.summary 字段，不动 items，避免覆盖当天已添加的 manual items

## 7. AI Prompt 调整

`backend/internal/ai/work_log_prompts.go` 中的 `WorkLogStructureSystem`：

- 输出格式去掉 `"title": "20 字以内的标题"`
- items 元素改为：`{content, problem_solved, result, impact}` 4 字段
- 拆条原则补充："每条目应可被 5~15 字的活动名概括（用户会在批量入库时填写 activity 字段，AI 不生成）"

`backend/internal/service/work_log_service.go` 中的 `StructuredItem` DTO：

```go
type StructuredItem struct {
    // Title 字段移除
    Content       string `json:"content"`
    ProblemSolved string `json:"problem_solved"`
    Result        string `json:"result"`
    Impact        string `json:"impact"`
}
```

`StructureBrainDump` 服务方法：返回时不再 marshal title 字段。

`backend/internal/service/work_log_ai_client_test.go`：调整断言（不再期望 title 字段）。

## 8. 一次性 SQLite Migration

`backend/pkg/database/migrate_work_items_title.go`（新文件）：

```go
// MigrateWorkItemsTitleBackfill 启动时执行一次，幂等
// 把 source='manual' 且 title 为空但 activity 非空的 items 的 title 回填为 activity
func MigrateWorkItemsTitleBackfill(db *gorm.DB) error {
    return db.Exec(`
        UPDATE work_items
        SET title = activity
        WHERE source = 'manual'
          AND (title = '' OR title IS NULL)
          AND activity IS NOT NULL
    `).Error
}
```

**调用点**：`backend/pkg/database/database.go` 的 `Init` 函数中，在 `AutoMigrate` 之后调用一次。带 WHERE 条件保证幂等，重启不会重复执行有效操作。

**测试**：
- 建表 + 插入混合数据（manual+空 title、manual+有 title、ai+空 activity）→ 调用 → 断言 manual+空 title 的被回填，其他不动
- 调用两次 → 第二次无变化（幂等）

## 9. 错误处理

| 场景 | 处理 |
|---|---|
| 主表单 add：activity 空 / start≥end | 前端 ElMessage.error，不发请求 |
| 主表单 add：网络失败 / 5xx | store.addQuickEntry 抛错，WorkItemForm 不 reset，ElMessage.error |
| 主表单 edit：同上 | 不清空 editingItemId，用户可重试 |
| 批量入库：必填校验失败 | 不发请求，BatchTableEditor 在失败行加错误样式，ElMessage 报"第 N 行 activity 必填" |
| 批量入库：部分 API 失败 | 已成功的从 draftItems 移除，失败的保留并标红，ElMessage 报"N/M 条成功" |
| 批量入库：summary 保存失败 | items 已落库不可回滚；summary PATCH 失败单独 ElMessage.warning（不阻塞主流程），draftSummary 保留供用户重试 |
| TodayPanorama 删除：弹窗取消 | 无操作（沿用现有 el-popconfirm） |

## 10. 测试策略

### 10.1 后端测试

**`backend/internal/service/work_log_service_test.go`（扩展）**：
- `TestAddQuickEntry_WithOptionalFields`：传入 4 可选字段，断言落库 WorkItem 包含全部字段 + `Title == Activity`
- `TestAddQuickEntry_NoOptionalFields`：可选字段空字符串，断言落库 OK
- `TestUpdateQuickEntry_ActivitySyncsTitle`：更新 activity，断言 title 同步
- `TestUpdateQuickEntry_OptionalFields`：更新 4 可选字段，断言部分更新 map
- `TestUpdateQuickEntry_ClearOptionalField`：传 `*""` 字符串指针，断言字段被清空（区别于 nil 不动）
- `TestUpdateSummary_*`：新方法，更新 WorkLog.summary，WorkLog 不存在时返回 ErrNotFound

**`backend/internal/service/work_log_ai_client_test.go`（调整）**：
- 修改 mock AI 响应，去掉 title 字段
- 断言 StructureBrainDump 返回的 items 不含 title

**`backend/internal/api/handler/work_log_test.go`（扩展）**：
- `TestAddQuickEntry_WithOptionalFields`：POST body 含 4 可选字段，断言绑定成功 + service 调用参数包含可选字段
- `TestUpdateQuickEntry_OptionalFields`：PATCH body 同上
- `TestUpdateSummary_*`：新端点 PATCH `/work-logs/:date/summary`，断言绑定 + service 调用 + 错误码映射

**`backend/internal/repository/work_log_repo_test.go`（扩展）**：
- `TestUpdateWorkLogSummary_*`：新方法，断言 WorkLog.summary 更新；WorkLog 不存在返回 ErrNotFound

**`backend/pkg/database/migrate_work_items_title_test.go`（新文件）**：
- 幂等性测试（调用两次第二次无变化）
- 混合数据测试：manual+空 title / manual+有 title / ai+空 activity 各类样本验证只回填符合条件者

### 10.2 前端测试

**`frontend/src/components/work-log/WorkItemForm.spec.ts`（新建，替代 QuickEntryForm.spec.ts）**：
- add 模式：必填校验（activity 空、start≥end）、提交触发 store.addQuickEntry、成功后 emit('added') + form 部分 reset
- add 模式：4 可选字段空字符串也能提交
- edit 模式：通过 initial prop 回填 8 字段、提交触发 store.updateQuickEntry、cancel emit('cancel')
- 字段一致性：date 在 edit 模式 disabled

**`frontend/src/components/work-log/BatchTableEditor.spec.ts`（新建）**：
- 渲染 N 行（draftItems 长度）+ "+加一条" 行
- 每行 activity 必填校验
- 编辑行触发 update:items emit
- 删除行触发 update:items emit
- "批量入库" 校验全过 → 触发 store.addWorkItemsBatch → 全部成功 emit('save')
- "批量入库" 校验失败 → 不发请求 + 错误标记
- "放弃草稿" 触发 emit('discard')

**`frontend/src/stores/workLog.spec.ts`（扩展）**：
- `addWorkItemsBatch` 全部成功：调 N 次 api.appendWorkItem，返回 successCount=N，failureIndices=[]
- `addWorkItemsBatch` 中间一条失败：已调用 i+1 次（前 i+1 次成功 + 第 i+1 次失败抛错停止），failureIndices=[i]，draftItems 中前 i+1 条已移除
- `updateSummary` 成功：调 api.updateWorkLogSummary，不弹错误 toast
- `updateSummary` 失败：ElMessage.warning，不抛错（不阻塞主流程）

**`frontend/src/views/WorkLog.vue`（无独立测试，依赖组件级覆盖）**：
- 集成验证靠手动 / E2E（项目暂无 E2E 框架）

### 10.3 类型检查

`npx vue-tsc --noEmit` 必须 0 error（项目无 ESLint，strict TS 是基线）。

## 11. Out of Scope

- 跨午夜时段支持（end > start 仍校验拒绝）
- 时段重叠约束（仍允许并行条目）
- TodayContextCard 重构
- 周期报告生成逻辑改动
- "今日小结"独立卡片（summary 留在 BatchTableEditor 底部）
- 批量入库的事务性（部分失败不回滚已成功的；后续可加 transactional endpoint）
- 编辑模式的批量编辑（TodayPanorama 只支持单条编辑）
