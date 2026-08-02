# 工作日志 - 快捷录入与今日全景

**Spec date**: 2026-08-02
**Branch**: evolve/work-log-integration (后续切到 evolve/work-log-quick-entry)
**Status**: 设计已确认，待写实施计划

## 背景与动机

现有「工作日志」模块围绕 `脑暴输入 → AI 拆条成 4 维 items → 保存日报` 流程构建，WorkItem 是 AI 结构化的叙事条目（title/content/problem_solved/result/impact）。用户希望在工作日志页面**随时随地把刚发生的活动记下来**，不要每次都跑 AI 流程。需求字段是轻量的「日期 + 活动 + 时间段 + 象限」，录入后立即看到今天的工作全景。

字段层 mismatch：
- 现有 WorkItem 4 维叙事字段 → 适合事后复盘，不适合实时记录
- 用户新需求字段（activity / start_time / end_time / quadrant）→ 适合实时记录，不强制叙事

「今天全景」与现有 `TodayContextCard`（拉 task + pomodoro session）解耦，是页面里新增的独立表格，按时段列出当天的 manual 条目。

## 设计目标 / 非目标

**目标**：
- 工作日志页面顶部新增「快捷录入表单」+「今日全景表」
- WorkItem 模型扩展 5 字段，复用同一张表
- AI 流程与 manual 流程互不擦除（关键不变式）
- 支持条目的编辑 / 删除

**非目标**：
- 不重构现有脑暴 → AI 拆条流程
- 不改 `TodayContextCard`（task + session 全景）
- 不支持跨午夜时段（end > start 校验拒绝）
- 不做时段重叠约束（允许并行条目）
- 不引入 Periodic Report 衍生（仍按现有周期报告逻辑读 WorkItem）

## 设计方案

### 数据模型

`backend/internal/model/work_log.go` 中的 `WorkItem` 扩展：

```go
type WorkItem struct {
    ID         string  `gorm:"primaryKey"`
    WorkLogID  string  `gorm:"index"`
    Seq        int

    // 新增字段（manual 录入必填，ai 录入为 null）
    Activity   *string `gorm:"column:activity"`
    StartTime  *string `gorm:"column:start_time"`    // "HH:MM"
    EndTime    *string `gorm:"column:end_time"`      // "HH:MM"
    Quadrant   *int    `gorm:"column:quadrant"`      // 1-4
    Source     string  `gorm:"column:source;default:'ai'"` // 'manual' | 'ai'

    // 现有 4 维叙事字段（ai 必填，manual 为空字符串）
    Title          string
    Content        string
    ProblemSolved  string
    Result         string
    Impact         string

    CreatedAt time.Time
    UpdatedAt time.Time
}
```

**关键决策**：
- 新字段全部用指针 → 区分「未设置 (nil)」和「零值」
- `Source` 显式优于「字段是否为空」推断（推断在 ai 偶然填了 activity 时会崩）
- `Source` default `'ai'` → 现有数据自动获得 `'ai'` 值，无需独立迁移脚本
- 不加唯一约束 → 同一时段允许多条（用户可能并行做多件事）

### 后端 API

**新增路由**（挂在现有 `/api/work-logs` group，注册于 `backend/internal/api/router.go`）：

| Method | Path | 用途 |
|---|---|---|
| `POST` | `/api/work-logs/:date/items` | 追加单条 manual 条目（WorkLog 不存在则自动建空壳） |
| `PATCH` | `/api/work-logs/:date/items/:itemId` | 编辑 manual 条目 |
| `DELETE` | `/api/work-logs/:date/items/:itemId` | 删除 manual 条目 |

**复用**：`GET /api/work-logs/:date` 不变，前端按 `source === 'manual'` 过滤 items 即可。

**Handler DTO**（`backend/internal/api/handler/work_log.go`）：

```go
type CreateQuickEntryInput struct {
    Activity  string `json:"activity" binding:"required"`
    StartTime string `json:"start_time" binding:"required"`
    EndTime   string `json:"end_time" binding:"required"`
    Quadrant  int    `json:"quadrant" binding:"required,min=1,max=4"`
}

type UpdateQuickEntryInput struct {
    Activity  *string `json:"activity" binding:"omitempty"`
    StartTime *string `json:"start_time" binding:"omitempty"`
    EndTime   *string `json:"end_time" binding:"omitempty"`
    Quadrant  *int    `json:"quadrant" binding:"omitempty,min=1,max=4"`
}
```

**Repo 接口扩展**（`backend/internal/repository/work_log_repo.go`）：

```go
type WorkLogRepository interface {
    // 现有方法保持...
    AppendItem(workLogID string, item WorkItem) error
    UpdateItem(workLogID string, itemID string, updates map[string]any) error
    DeleteItem(workLogID string, itemID string) error
}
```

**关键改造 — `UpsertWorkLog` 内部 `ReplaceItems`**：
现有全量替换改成「保留 manual」语义，先 `DELETE FROM work_items WHERE work_log_id=? AND source='ai'`，再插 ai items。manual 条目天然不被擦除。

**Service 层**（`backend/internal/service/work_log_service.go`）：

```go
func (s *workLogService) AddQuickEntry(ctx context.Context, date string, in CreateQuickEntryInput) (*WorkItem, error) {
    // 1. 找/建当天 WorkLog（GetWorkLogByDate → 不存在则 CreateWorkLog 空壳）
    // 2. 校验 start_time < end_time
    // 3. seq = max(existing seq) + 1
    // 4. 组装 WorkItem{Source:'manual', Activity:&in.Activity, ...}
    // 5. repo.AppendItem
    // 6. 返回新增 item
}
```

校验规则（service 层）：
- `start_time < end_time`（字符串比较 "HH:MM" 即可）
- `date` 格式 `YYYY-MM-DD`
- `activity` 非空 / `quadrant` 1-4（binding 已守）

**自动建空 WorkLog**：当天没 WorkLog 时，第一次快捷录入会自动建一个空 WorkLog（`raw_brain_dump=''`、`summary=''`、items=[新条目]）。后续 AI 流程走 `UpdateWorkLog` 把它当已存在的当天 log 处理。

### 前端组件

**新组件 1：`frontend/src/components/work-log/QuickEntryForm.vue`**

紧凑单行 / 两行表单，Element Plus 风格，镜像现有 `TaskForm.vue`：

```
[日期📅] [活动________] [开始🕒]-[结束🕒] [Q1 Q2 Q3 Q4] [添加]
```

- 字段：`date`(el-date-picker，默认今天) / `activity`(el-input) / `start_time`+`end_time`(el-time-select，30 分钟步长) / `quadrant`(el-radio-group 4 个 el-radio-button)
- 复用 `QUADRANT_INFO`（已存在的 4 象限颜色/名称常量）
- 提交成功：`ElMessage.success` + 重置 activity（保留日期/时段/象限方便连记）+ 触发全景刷新
- 手动校验：activity 非空 + end > start（跟 TaskForm 一致的非 rules 风格）
- 通过 `:mode` prop 区分新增 / 编辑（编辑时按钮文案改"保存"，预填字段）

**新组件 2：`frontend/src/components/work-log/TodayPanorama.vue`**

简易表格，按时段升序：

| 时段 | 活动 | 象限 | 操作 |
|---|---|---|---|
| 09:00-10:00 | 晨会 | Q1（红 tag） | ✏️ 🗑️ |
| 10:00-12:00 | 写设计文档 | Q2（蓝 tag） | ✏️ 🗑️ |

- 数据源：`store.todayManualItems`（getter，过滤 `source==='manual'`，按 `start_time` 升序）
- 空状态：`<el-empty description="今天还没有记录" />`
- 编辑：弹出 dialog 包装的 `QuickEntryForm`（mode='edit'，预填）
- 删除：`el-popconfirm` 二次确认 → `store.deleteQuickEntry`

**`WorkLog.vue` 改造**（在 `Timeline` 下、`TodayContextCard` 上插入）：

```vue
<QuickEntryForm :date="selectedDate" />
<TodayPanorama :date="selectedDate" />
<!-- 现有 -->
<TodayContextCard />
<BrainDumpInput />
<WorkItemList />
```

**Store 扩展**（`frontend/src/stores/workLog.ts`）：

```ts
state: {
  // 现有...
  quickEntrySaving: false,
}

getters: {
  todayManualItems: (state) => {
    const items = state.currentLog?.items ?? []
    return items
      .filter(i => i.source === 'manual')
      .sort((a, b) => (a.start_time ?? '').localeCompare(b.start_time ?? ''))
  }
}

actions: {
  async addQuickEntry(payload) {
    // 1. POST /api/work-logs/:date/items
    // 2. 成功后 await this.fetchLog(date) 重新拉 WorkLog
    //    （不调 fetchTodayContext —— manual items 与 task/session 解耦）
  },
  async updateQuickEntry(itemId, payload) { /* PATCH + refetch */ },
  async deleteQuickEntry(itemId) { /* DELETE + refetch */ },
}
```

**API client 扩展**（`frontend/src/api/client.ts`）：

```ts
appendWorkItem: (date, data) => api.post(`/work-logs/${date}/items`, data),
updateWorkItem: (date, itemId, data) => api.patch(`/work-logs/${date}/items/${itemId}`, data),
deleteWorkItem: (date, itemId) => api.delete(`/work-logs/${date}/items/${itemId}`),
```

**Types 扩展**（`frontend/src/types/index.ts`）：

```ts
export interface WorkItem {
  // 现有字段保持...
  activity?: string | null
  start_time?: string | null
  end_time?: string | null
  quadrant?: Quadrant | null
  source?: 'manual' | 'ai'
}
```

**关键交互不变式**：
- form 的 `date` prop 是受控的，panorama 跟随同一个 date
- 默认今天，用户改 date 时全景切到那一天
- 添加 / 编辑 / 删除成功后只 `fetchLog(date)`，**不**触发 `fetchTodayContext`

### 错误处理

| 场景 | HTTP | 错误信息 |
|---|---|---|
| `date` 格式非法 | 400 | `invalid date format` |
| `activity` 空 | 400 | binding 错误 |
| `start_time >= end_time` | 400 | `end_time must be after start_time` |
| `quadrant` 不在 1-4 | 400 | binding 错误（min/max） |
| PATCH/DELETE 时 item 不存在 | 404 | `item not found` |
| PATCH/DELETE 命中 `source='ai'` 条目 | 403 | `cannot modify ai item via quick entry endpoint` |
| Repo 写入失败 | 500 | 透传 `ErrInternal` |

**新增 repo 错误**（`backend/internal/repository/errors.go`）：

```go
var ErrItemNotFound = errors.New("work item not found")
var ErrItemNotEditable = errors.New("work item is not editable via this endpoint")
```

### 边界场景

1. **跨天添加**：form 选历史/未来日期 → 允许；panorama 跟随 form 的 date
2. **时段重叠**：不做唯一约束，允许并行条目
3. **AI 流程后跑**：用户先快捷录入 N 条 → 脑暴 → AI 保存时只擦 `source='ai'`，manual 全保留（关键不变式）
4. **WorkLog 空了**：删最后一条 manual 且当天没有 ai items → WorkLog 仍存在（空 items 数组），不删除（避免破坏周期报告对当天的引用）
5. **跨午夜时段**（如 23:00-01:00）：本期不支持，`end > start` 校验拒绝
6. **并发写**：SQLite 单写者，由 GORM 默认事务兜底，本期不加固

### 迁移安全

- GORM `AutoMigrate` 自动 `ALTER TABLE work_items ADD COLUMN ...`，新列 NULLABLE
- 现有数据：新列 = NULL，`source` 通过 GORM `default:'ai'` 自动回填
- 不写独立 migration 脚本

## 测试计划

### 后端

**`repository/work_log_repo_test.go`**（新增）：
- `AppendItem` 添加到已存在 WorkLog ✓
- `AppendItem` WorkLog 不存在 → `ErrNotFound` ✓
- `UpdateItem` 命中 ai item → `ErrItemNotEditable` ✓
- `DeleteItem` 删 manual ✓ / 删 ai → `ErrItemNotEditable` ✓
- **关键回归**：`UpsertWorkLog` 后 `source='manual'` 条目保留 ✓

**`service/work_log_service_test.go`**（扩展）：
- `AddQuickEntry` 自动建 WorkLog ✓
- `AddQuickEntry` start >= end → error ✓
- `UpdateQuickEntry` / `DeleteQuickEntry` happy path ✓

**`api/handler/work_log_test.go`**（扩展，复用 `mocks_test.go`）：
- POST/PATCH/DELETE happy path + 4xx 错误路径

### 前端

**`stores/workLog.spec.ts`**（扩展）：
- `addQuickEntry` 调 api + refetch `currentLog` ✓
- `todayManualItems` getter 过滤 + 排序 ✓
- 失败路径：api reject 时不更新 state ✓

**`components/work-log/QuickEntryForm.spec.ts`**（新增）：
- 渲染默认值（date=today）✓
- 提交空 activity → 不发请求 ✓
- 提交成功 → emit + 重置 activity ✓

**`components/work-log/TodayPanorama.spec.ts`**（新增）：
- 空状态渲染 ✓
- 渲染 manual items 按时段排序 ✓
- 编辑按钮 emit / 删除确认后调 store action ✓

## 实施顺序（粗粒度）

1. 后端 model + repo + service + handler + 路由 + 测试（自底向上）
2. 前端 types + api client + store + 测试
3. 前端 QuickEntryForm 组件 + 测试
4. 前端 TodayPanorama 组件 + 测试
5. WorkLog.vue 接线
6. E2E 手测：开 dev → 加条目 → 跑 AI 流程 → 确认 manual 保留

## 风险与开放问题

- **AI 流程的 `ReplaceItems` 改造**是最大风险点：如果忘了 `WHERE source='ai'` 过滤，会擦掉用户数据。回归测试是关键守门。
- `:date` 路径参数 vs `:id`：现有 `GetWorkLog` 路由参数命名需在实施时确认（推测是 `:date`，需读 router.go 验证）。如实际是 `:id`，则新路由也应保持一致风格。
- 跨午夜时段本期不支持 — 后续若有需求，需要把 `end_time` 改成 `end_datetime` 或加 `crosses_midnight` 标记。

## 参考

- 现有 TaskForm 模式：`frontend/src/components/tasks/TaskForm.vue`
- 现有 WorkLog 流程：`backend/internal/service/work_log_service.go`
- 象限常量：`frontend/src/types/index.ts` `QUADRANT_INFO`，`backend/internal/model/task.go` `Quadrant` 枚举
- GORM AutoMigrate：`backend/pkg/database/`
