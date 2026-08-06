# 配置/数据 导入导出 — 设计文档

- **日期**: 2026-08-07
- **状态**: 已批准(待实现计划)
- **分支**: `evolve/work-log-quick-entry`(当前工作分支;实现时按约定切 `evolve/*` 特性分支)

## 1. 目标

为 TickTask 增加一个**全量数据导入导出**功能:用户可一键把全部业务数据导出为单个 JSON 文件,并在另一台机器(或清库后)通过**冲突感知的导入流程**把数据导入回来。

导入时若当前库已有数据,系统按模块检测冲突,弹出解决界面由用户人工裁定(设置类逐字段 diff,集合类按模块策略 + 可选逐条覆盖)。

## 2. 范围

### 纳入(全部用户源数据)

| 模块 | model / 表 | 主键 | 外键 |
|---|---|---|---|
| 任务 | `Task` / tasks | `id` (uuid) | — |
| 番茄钟会话 | `PomodoroSession` / sessions | `id` | `task_id` → Task(可空) |
| 日程 | `Schedule` / schedules | `id` | `task_id` → Task(可空) |
| 设置 | `Setting` / settings | `key` | — (组装为 `pomodoro` + `ai`) |
| 工作日志 | `WorkLog` / work_logs | `id` | has_many `WorkItem` |
| 工作日志条目 | `WorkItem` / work_items | `id` | `work_log_id` → WorkLog(**非空**) |
| 周期报告 | `WorkReport` / work_reports | `id` | — |

### 排除

- **`DailyStats`(daily_stats)** — 派生聚合数据,可重算,不纳入导出。
- **`backend/configs/config.yaml`**(server/database/cors 部署配置)— 本就不在 UI 编辑,且含明文 AI key,不做应用内导入导出。

## 3. 背景

本项目存在两类「配置」:
- **DB 用户设置**(settings 表,UI 可改) — 本期纳入。
- **文件部署配置**(config.yaml) — 本期排除。

用户明确选择「全部数据」而非仅设置,理由是个人工具数据量小,无需拆细。

## 4. 关键决策

| 决策点 | 选择 | 备选 / 理由 |
|---|---|---|
| 文件格式 | **JSON 应用层转储**(带 `schema_version` + 各表分节) | 否决原始 SQLite 文件拷贝(不透明、绑 schema 版本)与 SQL dump(GORM 映射成本高) |
| 导入流程 | **两阶段:预览(只读 diff)→ 应用(单事务)** | 否决前端自算 diff(业务逻辑前移,违背架构);否决服务端导入会话(过度设计) |
| 设置序列化 | 组装态 `{pomodoro, ai}` 导出;导入时拆回 settings 行 | 对齐 `GetSettings` 与 UI,diff 友好 |
| work_items | **嵌套在 `work_logs.items`** | 贴合模型 has_many,保证父子绑定与一致策略 |
| 新 repo | 新建专用 `BackupRepository`,不改 6 个现有领域 repo | 横跨全表的特性,专用 repo 更内聚、改动更局部 |
| 冲突粒度 | 设置逐字段 diff;集合类「每模块策略 + 可选逐条 override」 | 用户明确选择;否决「全部逐条审阅」(数据多时太累) |

## 5. 架构与组件

### 5.1 后端新增

**`internal/repository/data_repo.go`** — 专用 `BackupRepository`(接口 + 私有 `dataRepository` 实现 + `NewDataRepository`),用 GORM 直接对各 model 做整表读 + 事务内批量写/替换。现有领域 repo 不动。

**`internal/service/data_service.go`** — 接口 `DataService` + 实现 `dataService` + `NewDataService(repo repository.BackupRepository)`:
- `Export() (*BackupData, error)` — 组装全量导出结构
- `PreviewImport(data *BackupData) (*ImportPreview, error)` — 只读 diff,产出冲突报告
- `ApplyImport(req *ApplyImportRequest) (*ApplyResult, error)` — 按解决意图单事务落库

**`internal/api/handler/data.go`** — 三个薄 handler(bind → service → JSON),router.go 新增 `/data` group:
- `GET  /api/data/export` — 返回 JSON,`Content-Disposition: attachment; filename=ticktask-backup-YYYYMMDD-HHMMSS.json`(浏览器直接下载)
- `POST /api/data/import/preview` — 接收上传 JSON(multipart `file`),返回 `ImportPreview`
- `POST /api/data/import/apply` — 接收 `ApplyImportRequest`,返回 `ApplyResult`

`cmd/server/main.go` 按现有手动 DI 风格注入 `BackupRepository` → `DataService` → `DataHandler`。

### 5.2 前端新增

- **`Settings.vue` 新增卡片「数据管理」**:两个按钮 `导出全部数据` / `导入数据`。
- **`components/settings/ImportWizard.vue`**(新)— 导入向导对话框,三步:① 选文件 → ② 预览(模块总览 + 设置字段级 diff + 每模块策略 + 可展开冲突清单)→ ③ 确认应用。
- **API client**(`src/api/client.ts`)加三方法:`exportData()` / `previewImport(file)` / `applyImport(payload)`。
- 状态留在向导组件本地(不新建 Pinia store)。

## 6. 数据结构

### 6.1 导出文件信封(`BackupData`)

```json
{
  "app": "ticktask",
  "schema_version": 1,
  "exported_at": "2026-08-07T00:00:00Z",
  "data": {
    "tasks":       [ {Task}, ... ],
    "sessions":    [ {PomodoroSession}, ... ],
    "schedules":   [ {Schedule}, ... ],
    "settings":    { "pomodoro": {PomodoroSettings}, "ai": {AISettings} },
    "work_logs":   [ {WorkLog, "items": [ {WorkItem}, ... ]} ],
    "work_reports":[ {WorkReport}, ... ]
  }
}
```

- `schema_version = 1`(当前版本)。导入时若不一致 → 告警;本期不做自动迁移。
- 各 model 的字段序列化沿用其现有 `json` tag,无需新定义。

### 6.2 预览结果(`ImportPreview`)

```json
{
  "schema_version": 1,
  "schema_warning": "",
  "modules": {
    "tasks":     { "new": 2, "identical": 5, "conflict": 1, "orphan": 0,
                   "conflicts": [ { "id": "...", "fields": [ {"field":"status","current":"todo","imported":"done"} ] } ] },
    "sessions":  { "new": ..., "identical": ..., "conflict": ..., "orphan": ...,
                   "conflicts": [ { "id": "...", "fields": [ ... ] } ] },
    "schedules": { ... },
    "work_logs": { "new": ..., "identical": ..., "conflict": ..., "orphan": ...,
                   "conflicts": [ { "id": "...", "fields": [ ... ] } ] },
    "work_reports": { ... },
    "settings":  { "conflicts": [
        { "section": "pomodoro", "field": "work_duration", "current": 1500, "imported": 1800 },
        { "section": "ai",       "field": "model",         "current": "gpt-4o-mini", "imported": "gpt-4o" }
    ] }
  }
}
```

- 集合类每模块四桶计数:`new` / `identical` / `conflict` / `orphan`;`conflicts` 仅列冲突记录明细(含逐字段 current vs imported)。
- `identical` 判定:对当前记录与文件记录各自做 canonical JSON 序列化后逐字节相等。
- 设置模块不适用桶,改为 `pomodoro`/`ai` 逐字段冲突列表;`api_key` 字段冲突照列(前端掩码显示)。
- **work_logs 的冲突解决粒度是「整条日志原子」**:一条日志的冲突判定涵盖其标量字段**与任一嵌套 item**(任一项不同即为 conflict)。解决时整条日志(含其全部 items)一起取 file 或 current 版本,不做 item 级单独 override,保证父子绑定一致。

### 6.3 应用请求(`ApplyImportRequest`)

```json
{
  "schema_version": 1,
  "modules": {
    "tasks":     { "policy": "merge_file", "overrides": { "<id>": "current" } },
    "sessions":  { "policy": "add_new_only", "overrides": {} },
    "schedules": { "policy": "merge_current", "overrides": {} },
    "work_logs": { "policy": "replace", "overrides": {} },
    "work_reports": { "policy": "merge_file", "overrides": {} },
    "settings":  { "pomodoro": { "work_duration": 1800, "...": "..." },
                   "ai":       { "model": "gpt-4o", "api_key": "<chosen value>" } }
  }
}
```

- 每模块 `policy` ∈ `add_new_only` | `merge_file` | `merge_current` | `replace`。
- `overrides` 仅作用于**冲突记录**,值 ∈ `file` | `current`,盖过模块策略(orphan 删除仅由 `replace` 决定,不被 override 影响)。对 work_logs,override 作用于整条日志(含其全部 items)。
- `settings` 携带**完整**的 pomodoro/ai 对象:冲突字段用用户选定值,非冲突字段沿用当前值,后端对完整对象按 key upsert 回 settings 表。

### 6.4 应用结果(`ApplyResult`)

```json
{ "applied": { "tasks": {"inserted":2,"updated":1,"deleted":0}, "sessions": {...}, "...": "..." } }
```

## 7. 数据流

### 7.1 导出

`导出按钮 → GET /api/data/export → DataService.Export()`
- `BackupRepository.ReadAll()` 读全表(work_items 预加载进 work_logs;settings 行组装为 `{pomodoro,ai}`)
- 组信封 → handler 设 attachment 头 → 浏览器下载
- 空库产出合法空信封,不报错

### 7.2 导入 · 阶段一:预览(只读)

`选文件 → POST /api/data/import/preview → DataService.PreviewImport()`
- 校验信封(`app`/`schema_version`/结构)。
- 加载当前库同形态数据,逐模块按主键比对 → 四桶归类 + 冲突明细;设置逐字段 diff。
- 返回 `ImportPreview`,**全程不写库**。

### 7.3 导入 · 阶段二:应用(单事务)

`收集解决意图 → POST /api/data/import/apply → DataService.ApplyImport()`

**每模块策略语义:**

| policy | 新增 | 冲突 | orphan |
|---|---|---|---|
| `add_new_only` 只加新的 | 插入 | 跳过 | 保留 |
| `merge_file` 文件优先 | 插入 | 用文件值 | 保留 |
| `merge_current` 当前优先 | 插入 | 保留当前 | 保留 |
| `replace` 整模块覆盖 | 插入 | 用文件值 | **删除** |

- 逐条 `overrides` 对冲突记录强制 file/current,盖过策略。
- **开一个 GORM 事务**,按 FK 顺序写入:`tasks → (sessions, schedules) → work_logs(含嵌套 work_items) → work_reports → settings`。任一步失败 → 整体回滚。
- `task_id` 引用:SQLite 默认不强制 FK,dangling 引用保留原值不报错(预览阶段已警告「N 条引用了不存在的任务」),不清理、不丢数据。
- settings:字段级选定值拆回 settings 表行(upsert by key)。
- 返回 `ApplyResult`;前端 success → 重载 settings store + 提供「刷新页面」入口。

## 8. 密钥与安全

- **导出**:`包含 AI API Key` 勾选框(默认✅)。取消则 `api_key` 导出为空串。
- **预览 diff**:`api_key` 一律掩码显示(`••••`),只给「用当前/用文件」开关,不露明文。
- **应用**:写入用户选定值。
- **安全网**:选 `replace` 时强制二次确认弹窗;应用前提供「先导出当前数据」入口。本期不做服务端自动快照(YAGNI)。

## 9. 错误处理与边界

**预览校验**
- 非 JSON / `app != "ticktask"` / 缺 `data` → `400`("文件格式无效" / "不是有效的 TickTask 备份文件")
- `schema_version` 不识别 → 仍返回预览 + `schema_warning`;结构完全不兼容才 `400`
- 文件过大 → 设上限(50MB)提前拒绝,防 OOM

**应用校验**
- 非法 `policy` 枚举 / override id 不在冲突清单 → `400`;未知的 override id 静默忽略
- 事务失败 → 回滚 + `500`;前端「导入失败,数据未改动」
- 错误信息不回显 `api_key`

**并发 / SQLite**
- 导入持一个写事务,期间其他写短暂阻塞;个人单用户、用户主动触发、耗时短 → 可接受。计时器会话写入理论上可能冲突,不做特殊处理,文档标注此限制。

**前端 UX**
- 导出失败 → `ElMessage.error`
- 预览失败 → 报错 + 停在选文件步
- 应用失败 → 报错 + 向导不关
- 应用成功 → `ElMessage.success` + 重载 settings store + 「刷新页面」入口

## 10. 测试策略

沿用项目约定(Go 标准 `testing` + 手写 in-memory mock + 表驱动;Vitest + `@vue/test-utils`)。编号前缀:`S`=后端 service、`H`=后端 handler、`F`=前端、`M`=手工。

### 后端 · `data_service_test.go`(mock `BackupRepository`)

**导出**

| # | 场景 | 预期 |
|---|---|---|
| S-1 | 空库导出 | 合法信封,各数组空,settings 为默认 pomodoro/ai |
| S-2 | 有数据导出 | 各表记录数与 mock 一致 |
| S-3 | settings 组装 | 表行正确组装为 `{pomodoro, ai}` |
| S-4 | work_items 嵌套 | item 归入所属 work_log.items |
| S-5 | 排除 DailyStats | 信封无 daily_stats |
| S-6 | 信封元信息 | `app`/`schema_version=1`/`exported_at` 非空 |

**预览(只读 diff)**

| # | 场景 | 预期 |
|---|---|---|
| S-7 | 全 new | 每模块 new=N,其余 0 |
| S-8 | 全 identical | identical=N,其余 0 |
| S-9 | 单条冲突 | conflict=1,conflicts 含逐字段 diff |
| S-10 | orphan 存在 | orphan 计数正确 |
| S-11 | 四桶混合 | 四桶计数全对 |
| S-12 | settings 逐字段 diff | 每个不同字段进 conflicts |
| S-13 | settings api_key 冲突 | 列入 conflicts(后端不掩码) |
| S-14 | work_logs 标量冲突 | 该 log 判 conflict |
| S-15 | work_logs 嵌套 item 冲突 | 整条 log 判 conflict(原子) |
| S-16 | work_logs 全相同(含 items) | identical |
| S-17 | schema_version 不匹配 | 返回预览 + schema_warning 非空 |
| S-18 | identical 判定边界 | canonical JSON 逐字节相等才判 identical |

**应用(单事务)**

| # | 场景 | 预期 |
|---|---|---|
| S-19 | `add_new_only` | 只插 new,inserted=N |
| S-20 | `merge_file` | 插 new + 冲突用文件值,orphan 保留 |
| S-21 | `merge_current` | 插 new + 冲突保留当前,orphan 保留 |
| S-22 | `replace` | 插 new + 冲突用文件值 + 删 orphan |
| S-23 | override=file 盖过 merge_current | 那条用文件值 |
| S-24 | override=current 盖过 merge_file | 那条保留当前 |
| S-25 | override 作用 orphan | 不生效 |
| S-26 | work_logs replace | 整条 log+items 用文件版 |
| S-27 | settings 字段选定值落表 | 正确 upsert |
| S-28 | **事务回滚不变式** | 中途写失败 → 零提交 |
| S-29 | FK 写入顺序 | work_items 随 work_log 写入 |
| S-30 | 全空操作 | 无写,counts 全 0 |

**校验**

| # | 场景 | 预期 |
|---|---|---|
| S-31 | 坏 JSON | 「文件格式无效」 |
| S-32 | `app != ticktask` | 「不是有效的 TickTask 备份文件」 |
| S-33 | 缺 `data` | 错误 |
| S-34 | 文件 >50MB | 拒绝 |
| S-35 | 非法 policy 枚举 | 错误 |
| S-36 | 未知 override id | 静默忽略 |

### 后端 · `data_test.go`(handler · 复用 `mocks_test.go`)

| # | 场景 | 预期 |
|---|---|---|
| H-1 | `GET /api/data/export` | 200 + attachment 头 + 合法 JSON |
| H-2 | export 空库 | 200 + 空信封 |
| H-3 | `POST /import/preview` 合法 | 200 + ImportPreview |
| H-4 | `POST /import/preview` 坏文件 | 400 |
| H-5 | `POST /import/apply` 合法 | 200 + ApplyResult |
| H-6 | `POST /import/apply` 非法 policy | 400 |
| H-7 | `POST /import/apply` 服务层错误 | 500 |

### 前端 · Vitest

**`ImportWizard.vue`**

| # | 场景 | 预期 |
|---|---|---|
| F-1 | 步骤流转 | 选文件→预览→应用 正确推进 |
| F-2 | 渲染模块计数 | 四桶计数正确 |
| F-3 | settings 字段 diff | 列出 current vs imported |
| F-4 | api_key 掩码 | `••••`,不露明文 |
| F-5 | 策略下拉变更 | 更新 apply payload |
| F-6 | 逐条 override 切换 | 更新 overrides |
| F-7 | 选 replace | 弹二次确认 |
| F-8 | 取消二次确认 | 不发起 apply |
| F-9 | 收集 payload | 调 applyImport,形态正确 |
| F-10 | apply 成功 | 关闭向导 + 触发重载 |
| F-11 | apply 失败 | ElMessage.error + 向导不关 |
| F-12 | 预览失败 | ElMessage.error + 停选文件步 |

**`Settings.vue` + api client**

| # | 场景 | 预期 |
|---|---|---|
| F-13 | 「数据管理」卡片 | 渲染,含两按钮 |
| F-14 | 点导出 | 调 api.exportData |
| F-15 | 点导入 | 打开 ImportWizard |
| F-16 | exportData() | GET,下载/blob 正确 |
| F-17 | previewImport(file) | POST multipart,字段名 file |
| F-18 | applyImport(payload) | POST JSON,形态正确 |

### 手工验证(非自动化 · verify 流程)

| # | 场景 | 预期 |
|---|---|---|
| M-1 | 导出→清库→导入(replace)→比对 | 与原库完全一致 |
| M-2 | 含 api_key 导入→设置页 AI 卡 | api_key 正确还原 |
| M-3 | dangling task_id 导入 | 不报错,预览有警告 |

## 11. 未来扩展(本期不做)

- `schema_version` 不一致时的自动迁移器
- `DailyStats` 重算入口(导入后一键重建分析聚合)
- 导入进度条(超大文件场景)
- 选择性模块导出(只导任务/只导日志)
