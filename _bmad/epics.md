---
stepsCompleted: [1, 2, 3, 4]
inputDocuments:
  - _bmad/prds/prd-ticktask-2026-06-06/prd.md
  - _bmad/C-UX-Scenarios/00-ux-scenarios.md
  - _bmad/architecture.md
  - _bmad/A-Product-Brief/product-brief.md
  - docs/superpowers/specs/2026-06-08-pomodoro-task-linkage-design.md
  - _bmad/test-artifacts/test-design-epic-pomodoro-task-linkage.md
---

# TickTask - Epic Breakdown

## Overview

This document provides the complete epic and story breakdown for **AI Agent 可配置能力**，从 PRD、UX Scenarios、Architecture、Product Brief 中提取需求并分解为可实施的 Story。

## Requirements Inventory

### Functional Requirements

FR1: **工作时间偏好配置** — 用户可在设置界面配置每日工作起止时间（如 09:00–18:00）、午休时间范围（AI 排程保护该时段不放入任务）、深度/浅层工作时间偏好（上午/下午/无偏好）

FR2: **精力匹配规则配置** — 用户可启用/禁用精力匹配开关（高专注 deep_work 任务优先安排在深度工作时间，低专注 shallow_work 安排在浅层时间）；周五下午保护开关（避免安排高难度任务）

FR3: **任务时长偏好配置** — 用户可配置默认任务时长（分钟，默认 60）、最小任务时长（分钟，默认 30，AI 不生成短于此值的日程块）、任务间缓冲时间（分钟，默认 15，相邻任务最小间隔）

FR4: **Prompt 模板编辑** — 用户可在 textarea 中查看和编辑完整 Prompt 模板（基于现有 `auto-schedule/SKILL.md` 提炼默认值）；支持变量占位符 `{{tasks}}` `{{habits}}` `{{week_start}}` `{{week_end}}` `{{learning_log}}`；提供「重置为默认值」按钮；修改后的 Prompt 对日程生成和修订均生效

FR5: **AI 后端类型选择** — 用户可在三种后端之间切换：Claude CLI（`claude -p`）、OpenCode CLI（`opencode run`）、API 直连；后端选择对日程生成和修订统一生效

FR6: **Claude CLI 后端配置** — 配置 Claude 可执行文件路径（默认 `claude`，PATH 查找）、附加参数（默认 `--output-format stream-json --verbose --permission-mode acceptEdits --dangerously-skip-permissions`，用户可覆盖）；提供「测试连接」按钮执行 `claude --version`

FR7: **OpenCode CLI 后端配置** — 配置 OpenCode 可执行文件路径（默认 `opencode`）、模型选择（下拉）、附加参数（自定义）；提供「测试连接」按钮

FR8: **API 直连后端配置** — Provider 选择（OpenAI / Anthropic / 自定义）；API Key（密码输入框）；Base URL（Provider 默认值，自定义可自由填写）；Model（下拉+可自定义输入，Provider 提供预设列表）；「测试连接」按钮发送最小请求验证 Key

FR9: **后端切换行为** — 切换后端时旧配置保留不丢失；日程生成和修订统一使用当前选中后端；升级后默认 Claude CLI，现有用户无需任何改动

FR10: **配置导出** — 设置页面「导出配置」按钮；导出 JSON 文件（含习惯、Prompt、后端配置、版本号、导出时间，不含 API Key）；文件名 `ticktask-config-YYYY-MM-DD.json`

FR11: **配置导入** — 设置页面「导入配置」按钮；选择 JSON 文件后预览变更内容；用户确认后原子性写入（全部成功或全部拒绝）；导入失败显示明确错误（格式错误/版本不兼容/字段缺失）

### NonFunctional Requirements

NFR1: **API Key 安全性** — Key 存储在本地 SQLite DB 中；导出文件不包含 Key；界面中始终密码遮蔽显示

NFR2: **向后兼容性** — 升级后现有用户无需迁移操作；已有 `config/habit.md` 手写内容首次进入设置时解析回填 UI；默认后端 Claude CLI

NFR3: **用户体验** — 配置修改保存后即时生效无需重启；测试连接 5 秒内返回；导出/导入 3 次点击内完成

NFR4: **错误处理** — AI 后端不可用时（CLI 未安装、API Key 无效），日程生成界面显示明确错误提示；生成失败已有日程不受影响

### Additional Requirements

- **AgentRunner 接口抽象**：三种后端统一为后端 Go interface，接受 Prompt 字符串返回流式输出；`ScheduleService` 面向接口编程，解耦硬编码 `claude -p`
- **Prompt 变量替换引擎**：后端负责将 `{{tasks}}` `{{habits}}` `{{week_start}}` `{{week_end}}` `{{learning_log}}` 占位符替换为实际数据
- **JSON 导出 Schema 版本化**：导出 JSON 含 `version` 字段，供未来配置格式升级兼容判断
- **CLI 子进程管理**：Claude CLI 和 OpenCode CLI 通过子进程调用，需要统一超时控制、输出流解析（stream-json）、错误处理
- **技术栈复用**：Go + Gin、SQLite；扩展 `PUT /api/settings` 系列接口增加 agent/habits/prompt 端点；复用 WebSocket `terminal_output` / `terminal_status` 实时推送

### UX Design Requirements

UX-DR1: **设置页面 AI 配置分节** — Settings.vue 当前有「Pomodoro Settings」+「AI Settings」两个卡；需重构/新增为三个清晰配置区：Agent 后端选择 + 习惯偏好 + Prompt 模板编辑

UX-DR2: **配置即时生效体验** — 遵循「设一次偏好，AI 越排越准」的 UX 理念；修改自动保存或明确保存按钮 + 成功反馈；遵循产品 Brief 定义的「干练高效、温和可靠」语调

UX-DR3: **测试连接反馈** — 点击后内联展示 loading → 成功/失败结果（非弹窗）；失败时给出可操作的提示（如"未检测到 Claude CLI，请确认已安装"）；5 秒内返回

UX-DR4: **配置导入预览** — 导入 JSON 后在对话框中预览变更（文件名、导出时间、配置摘要），用户确认后写入；导入失败显示具体原因

### FR Coverage Map

| FR | Epic | 描述 |
|----|------|------|
| FR1 | Epic 2 | 工作时间偏好配置 |
| FR2 | Epic 2 | 精力匹配规则配置 |
| FR3 | Epic 2 | 任务时长偏好配置 |
| FR4 | Epic 2 | Prompt 模板编辑 |
| FR5 | Epic 1 | AI 后端类型选择 |
| FR6 | Epic 1 | Claude CLI 后端配置 |
| FR7 | Epic 1 | OpenCode CLI 后端配置 |
| FR8 | Epic 1 | API 直连后端配置 |
| FR9 | Epic 1 | 后端切换行为 |
| FR10 | Epic 1 | 配置导出 |
| FR11 | Epic 1 | 配置导入 |

## Epic List

### Epic 1: Agent 后端可配置

用户可以选择并切换 AI 执行后端（Claude CLI / OpenCode CLI / API 直连），测试连接验证可用性，并导出/导入配置以便备份和迁移。

**FRs covered:** FR5, FR6, FR7, FR8, FR9, FR10, FR11

---

### Story 1.1: AgentRunner 接口 + 数据模型 + 设置 API

As a **后端开发者**,
I want **定义 `AgentRunner` Go interface 并扩展 `AISettings` 数据模型以支持三种后端（Claude CLI、OpenCode CLI、API 直连），同时提供完整的设置 CRUD API**,
So that **`ScheduleService` 可以面向接口编程，不再硬编码 `claude -p` 调用**.

**Acceptance Criteria:**

**Given** `backend/internal/ai/` 目录
**When** 新增 `runner.go` 文件
**Then** 定义 `AgentRunner` 接口，包含 `Run(ctx context.Context, prompt string) (<-chan StreamChunk, error)` 方法
**And** `StreamChunk` 结构体包含 `Text string` 和 `IsStderr bool` 字段
**And** 接口支持流式输出（channel-based），与 WebSocket 推送机制兼容

**Given** `backend/internal/model/setting.go` 中的 `AISettings` 结构体
**When** 扩展模型
**Then** 新增 `AgentConfig` 结构体，包含 `Backend`、`ClaudeCLIConfig`、`OpenCodeCLIConfig`、`APIConfig`、`HabitsConfig`、`PromptTemplate` 字段
**And** 所有字段有 JSON tag 和合理的默认值（`Backend` 默认 `"claude_cli"`，CLI 路径默认 `"claude"` / `"opencode"`）

**Given** 现有 SQLite 数据库 `ticktask.db`，`settings` 表存有 `ai.settings` key
**When** 后端启动时执行迁移
**Then** 新增 `agent.config` setting key，首次初始化时写入默认 `AgentConfig` JSON
**And** 现有 `ai.settings` 配置迁移到 `agent.config.api` 字段中
**And** 迁移为幂等操作，失败不阻塞启动仅 WARN 日志

**Given** 前端请求获取当前 Agent 配置
**When** `GET /api/settings/agent` 被调用
**Then** 返回 `200`，body 包含完整 `AgentConfig` JSON（`api_key` 仅返回后 4 位脱敏）
**And** 如果数据库无配置，返回默认配置

**Given** 前端发送更新 Agent 配置的请求
**When** `PUT /api/settings/agent` 被调用
**Then** 验证 `backend` 字段必须在 `["claude_cli", "opencode_cli", "api"]` 中
**And** 验证通过后写入数据库，返回 `200` 及更新后的完整配置
**And** 验证失败返回 `400` + 具体错误字段说明

**Given** 用户点击「测试连接」按钮
**When** `POST /api/settings/agent/test` 被调用
**Then** 后端 5 秒超时内执行连接测试（CLI 执行 `--version`，API 发送最小请求验证 Key）
**And** 成功返回 `{ "ok": true, "message": "..." }`，失败返回 `{ "ok": false, "message": "..." }`

**Given** 现有 TickTask 用户升级到新版本
**When** 后端首次启动且数据库中无 `agent.config`
**Then** 默认配置 `backend = "claude_cli"`，现有日程生成/修订功能正常运行（行为与升级前一致）

**Requirements covered:** FR5, FR9, NFR1, NFR2

---

### Story 1.2: CLI 后端实现 + 日程生成集成

As a **用户（老刘）**,
I want **使用 Claude CLI 或 OpenCode CLI 作为 AI 后端来生成和修订日程**,
So that **我可以继续使用本地 CLI 工具，同时在未来灵活切换到其他后端**.

**Acceptance Criteria:**

**Given** `backend/internal/ai/` 目录
**When** 新增 `claude_cli_runner.go`（改造现有 CLIClient）
**Then** 实现 `AgentRunner` 接口，`Run()` 方法构建 `{path} -p {prompt} {extra_args}` 并通过 `exec.CommandContext` 执行
**And** 现有 `runClaudeStreamJSON` 核心逻辑迁移到此实现中
**And** 复用 WebSocket `terminal_output` / `terminal_status` 实时推送

**Given** `backend/internal/ai/` 目录
**When** 新增 `opencode_cli_runner.go`
**Then** 实现 `AgentRunner` 接口，构建 `{path} run {prompt}` 执行
**And** 输出流解析适配 opencode 格式（stream-json），复用相同的 WebSocket 推送逻辑

**Given** `AgentConfig.Backend` 选择了任意后端类型
**When** `ScheduleService` 需要执行 AI 命令
**Then** `NewAgentRunnerFactory(config)` 根据 `Backend` 字段返回对应的 `AgentRunner` 实例（`claude_cli` / `opencode_cli` / `api`）

**Given** `ScheduleService` 当前硬编码 `claude -p`
**When** 改造 `GenerateSchedule()` 和 `ReviseSchedule()`
**Then** 使用 `AgentRunnerFactory` 获取 runner，调用 `runner.Run(ctx, skillPrompt)`
**And** 原有 config 文件写入、ICS 解析、校验流程保持不变
**And** `ClaudeCLIRunner` 产生的行为与原来硬编码完全一致

**Given** 配置的 CLI 路径不可执行（未安装、路径错误）
**When** `runner.Run()` 被调用
**Then** 返回明确错误（含路径信息），WebSocket 广播 `terminal_status: error`，前端可见

**Requirements covered:** FR6, FR7, FR9, NFR4

---

### Story 1.3: API 直连后端 + 日程生成集成

As a **用户（老刘）**,
I want **使用自己的 API Key 直接调用 OpenAI / Anthropic 等 AI 服务来生成和修订日程**,
So that **我不依赖本地 CLI 工具，也能享受 AI 排程**.

**Acceptance Criteria:**

**Given** `backend/internal/ai/` 目录
**When** 新增 `api_runner.go`
**Then** 实现 `AgentRunner` 接口，根据 `AgentConfig.APIConfig` 的 `Provider` 选择 OpenAI / Anthropic / Custom 格式调用
**And** `Run()` 支持 SSE 流式响应（`text/event-stream`），chunk delta 通过 `StreamChunk` channel 推送

**Given** Schedule skill prompt（当前通过 CLI 执行）
**When** 通过 API 直连调用
**Then** 将 skill prompt 转换为标准 `messages` 数组（system + user），API 响应 ICS 文本写入 `schedule.ics` → 校验
**And** 非流式响应一次性写入后返回

**Given** API 直连 SSE 流式响应
**When** 每个 chunk 到达
**Then** 通过 `hub.BroadcastTerminalOutput(chunk, false)` 实时推送
**And** TerminalOverlay 体验与 CLI 后端一致，流结束时广播 `terminal_status: completed`

**Given** API Key 无效或过期
**When** `runner.Run()` 被调用
**Then** 返回明确错误 "API Key 无效"，WebSocket 广播 `terminal_status: error`
**And** 已有日程数据不受影响

**Given** API 调用超过 300 秒无响应
**When** 超时
**Then** 自动取消，返回超时错误，前端展示 "连接超时，请检查网络或 API 地址"

**Requirements covered:** FR8, NFR4

---

### Story 1.4: 前端 Agent 配置 UI

As a **用户（老刘）**,
I want **在设置页面可视化选择 AI 后端、填写各后端参数、一键测试连接**,
So that **我不需要编辑配置文件或数据库就能完全掌控 AI 行为**.

**Acceptance Criteria:**

**Given** 用户进入设置页面 `/settings`
**When** 页面加载
**Then** 新增/重构「AI Agent」配置卡片，包含三个配置区：后端选择（Radio/Select 切换 Claude CLI / OpenCode CLI / API 直连）+ 动态表单 + 测试连接按钮

**Given** 用户选择 Claude CLI 后端
**When** 表单渲染
**Then** 显示可执行文件路径（默认 `claude`）、附加参数（默认展示当前参数+灰色提示文字）、测试连接按钮（内联反馈结果）

**Given** 用户选择 OpenCode CLI 后端
**When** 表单渲染
**Then** 显示可执行文件路径（默认 `opencode`）、模型下拉选择、附加参数、测试连接按钮

**Given** 用户选择 API 直连后端
**When** 表单渲染
**Then** 显示 Provider 选择（OpenAI/Anthropic/自定义）、API Key（密码遮蔽）、Base URL（Provider 切换默认值）、Model（可筛选+自定义输入）

**Given** 用户修改了任意 Agent 配置
**When** 点击「保存配置」
**Then** 调用 `PUT /api/settings/agent` 保存，成功后 `ElMessage.success` 即时生效
**And** 切换后端时旧配置独立保留不丢失

**Given** 用户进入设置页面
**When** `GET /api/settings/agent` 请求中
**Then** 显示 loading 状态，失败显示错误 + 重试按钮

**Requirements covered:** FR5, FR6, FR7, FR8, FR9, NFR3, UX-DR1, UX-DR2, UX-DR3

---

### Story 1.5: 配置导出/导入

As a **用户（老刘）**,
I want **将我的 AI 配置导出为 JSON 文件备份，并在需要时一键导入恢复**,
So that **换设备或重装后不用重新配置，迁移零 friction**.

**Acceptance Criteria:**

**Given** 用户点击「导出配置」
**When** `GET /api/settings/agent/export` 被调用
**Then** 后端读取 `AgentConfig`，序列化为 JSON（含 `version` + `exported_at`，**不含 `api_key`**），返回文件下载 `ticktask-config-YYYY-MM-DD.json`

**Given** 用户选择 JSON 配置并确认导入
**When** `POST /api/settings/agent/import` 被调用
**Then** 后端验证 JSON 格式、版本兼容性、必填字段 → 原子性写入 DB（全部成功或全部拒绝）
**And** 失败返回 `400` + 具体原因

**Given** 用户在 Settings 页面 AI Agent 配置区域
**When** 查看页面
**Then** 底部显示「导出配置」按钮（Download 图标）+「导入配置」按钮（Upload 图标）

**Given** 用户点击「导入配置」并选择 `.json` 文件
**When** 文件选择器返回
**Then** 弹出「导入配置预览」el-dialog，展示文件名、导出时间、配置摘要 + "API Key 不会被导入"警告
**And** 用户确认后导入成功 → 表单刷新；失败 → 对话框内显示错误可关闭

**Requirements covered:** FR10, FR11, NFR1, UX-DR4

---

## Epic 2: 习惯与 Prompt 定制

用户可在设置界面可视化配置个人工作习惯和 AI Prompt 模板，替代手动编辑 `config/habit.md` 文件的体验，让 AI 排程更精准贴合个人偏好。

**FRs covered:** FR1, FR2, FR3, FR4

---

### Story 2.1: 习惯配置数据模型 + API

As a **后端开发者**,
I want **在 `AgentConfig` 中定义 `HabitsConfig` 结构体并提供读写 API，首次启动时自动解析已有 `config/habit.md` 回填到数据库**,
So that **用户的习惯数据有结构化存储，不再依赖手动编辑 Markdown 文件**.

**Acceptance Criteria:**

**Given** `backend/internal/model/setting.go`
**When** 扩展 `AgentConfig` 结构体
**Then** 新增 `HabitsConfig` 字段（`WorkStart`/`WorkEnd`/`LunchStart`/`LunchEnd`/`DeepWorkPreference`/`ShallowWorkPreference`/`EnergyMatchingEnabled`/`FridayAfternoonProtection`/`DefaultDurationMinutes`/`MinDurationMinutes`/`BufferMinutes`），全部带 JSON tag 和合理默认值

**Given** 后端首次启动且 DB 中 `habits` 为默认值，`config/habit.md` 文件存在
**When** 启动迁移逻辑执行
**Then** 解析 `habit.md` 结构化内容回填到 `AgentConfig.HabitsConfig` 并写入 DB
**And** 已回填的数据库不重复解析（`habits_migrated` flag）

**Given** 前端请求习惯配置
**When** `GET /api/settings/agent/habits` 被调用
**Then** 返回 `200` + `HabitsConfig` JSON

**Given** 前端发送更新习惯配置
**When** `PUT /api/settings/agent/habits` 被调用
**Then** 验证时间格式（`HH:MM`）、枚举值、数值范围 → 写入 DB + 同步写入 `config/habit.md`
**And** 验证失败返回 `400` + 具体错误字段

**Requirements covered:** FR1, FR2, FR3, NFR2

---

### Story 2.2: Prompt 模板数据模型 + 变量替换引擎

As a **后端开发者**,
I want **在 `AgentConfig` 中存储可自定义的 Prompt 模板并提供变量替换引擎**,
So that **用户自定义的 Prompt 能在日程生成/修订时动态注入实际数据**.

**Acceptance Criteria:**

**Given** `AgentConfig` 结构体
**When** 扩展模型
**Then** 新增 `PromptTemplate string` 字段，默认值从 `auto-schedule/SKILL.md` 提炼为 Go 常量 `DefaultPromptTemplate`

**Given** `backend/internal/service/` 目录
**When** 新增 `prompt_engine.go`
**Then** 实现 `BuildPrompt(template string, data PromptData) (string, error)` —— 遍历 `{{tasks}}` `{{habits}}` `{{week_start}}` `{{week_end}}` `{{learning_log}}` 占位符并替换为实际值，未匹配的保留原样+WARN 日志

**Given** 前端请求 Prompt 模板
**When** `GET /api/settings/agent/prompt` 被调用
**Then** 返回 `{ prompt_template, is_default }`（`is_default` 标识是否修改过）

**Given** 前端更新 Prompt 模板
**When** `PUT /api/settings/agent/prompt` 被调用
**Then** 验证非空 → 写入 DB，返回更新后模板

**Given** 用户点击「重置为默认值」
**When** `POST /api/settings/agent/prompt/reset` 被调用
**Then** 重置为 `DefaultPromptTemplate`，返回默认模板

**Requirements covered:** FR4

---

### Story 2.3: 前端习惯 + Prompt 配置 UI

As a **用户（老刘）**,
I want **在设置页面可视化配置我的工作习惯和 Prompt 模板**,
So that **我不用手动编辑 `config/habit.md` 文件，在 UI 上就能让 AI 理解我的偏好**.

**Acceptance Criteria:**

**Given** 用户在 Settings 页面「AI Agent」配置区域
**When** 切换到「工作习惯」Tab
**Then** 显示表单：工作起止时间（el-time-select）+ 午休时间 + 深度/浅层工作偏好（el-radio-group）+ 精力匹配开关 + 周五保护开关 + 默认/最小/间隔时长（el-input-number）+ 「保存习惯」按钮
**And** 保存后 `ElMessage.success` 即时生效

**Given** 用户在 Settings 页面
**When** 切换到「Prompt 模板」Tab
**Then** 显示：引导文字 + 10+ 行 textarea（当前模板）+ 变量参考卡片（`{{tasks}}` `{{habits}}` 等只读展示）+ 「重置为默认值」+ 「保存模板」按钮
**And** 重置弹出确认对话框 → 调用 API → textarea 刷新；保存调用 API → success 提示

**Given** 用户进入 Settings 页面
**When** 页面 mount
**Then** 一次性 `GET /api/settings/agent` 获取完整配置，`habits` 和 `prompt_template` 分发填充，loading/error 状态处理

**Requirements covered:** FR1, FR2, FR3, FR4, NFR3, UX-DR1, UX-DR2

---

### Story 2.4: 日程生成 Prompt 动态注入

As a **用户（老刘）**,
I want **日程生成/修订时自动使用我配置的习惯和 Prompt 模板**,
So that **我在设置中的定制真正影响到 AI 的排程行为**.

**Acceptance Criteria:**

**Given** 用户修改了 Prompt 模板和习惯配置
**When** 点击「生成日程」或「修订日程」
**Then** `ScheduleService` 从 DB 读取 `AgentConfig.PromptTemplate`，调用 `BuildPrompt()` 替换 `{{tasks}}` `{{habits}}` `{{week_start}}` `{{week_end}}` `{{learning_log}}` 占位符
**And** 替换后的完整 Prompt 传给 `AgentRunner.Run()`
**And** 当模板为默认值（`is_default = true`）时，行为与升级前完全一致

**Given** 用户通过 UI 修改了习惯配置
**When** 保存后
**Then** `config/habit.md` 文件同步更新，CLI skill 读到的数据与 UI 一致

**Given** 用户定制了习惯 + Prompt（如深度工作改下午、代码评审放 4 点后）
**When** 生成日程
**Then** AI 生成的七日 ICS 反映这些定制约束，且三种后端（CLI/API）均生效

**Requirements covered:** FR4, NFR2

---

## Epic 3: 番茄钟-任务关联

将番茄钟（Pomodoro）与任务深度关联，实现任务级番茄钟计划/进度追踪、完成提醒、以及跨界面（任务/日程/分析）的番茄钟统计。用户可以直观地看到每个任务需要多少番茄钟、完成了多少、并在用完后获得完成/延长提醒。

**FRs covered:** FR1, FR2, FR3, FR4, FR5, FR6, FR7, FR8
**Design Spec:** `docs/superpowers/specs/2026-06-08-pomodoro-task-linkage-design.md`
**Test Design:** `_bmad/test-artifacts/test-design-epic-pomodoro-task-linkage.md`

---

### Story 3.1: 后端 API — 番茄钟计算字段 + 分析端点

As a **后端开发者**,
I want **在 Task API 响应中添加番茄钟计算字段（planned/completed/status），并新增两个分析 API 端点**,
So that **前端可以获取每个任务的番茄钟进度，分析页面可以展示番茄钟排行、趋势和完成率统计**.

**Acceptance Criteria:**

**Given** `backend/internal/repository/session_repo.go` 中的 `SessionRepository` 接口
**When** 新增 `CountByTaskID(taskID string, sessionType string, status string) (int, error)` 方法
**Then** 返回 `SELECT COUNT(*) FROM pomodoro_sessions WHERE task_id = ? AND type = ? AND status = ?` 的结果
**And** 查询使用现有 SQLite 索引，无 N+1 问题

**Given** `backend/internal/service/task_service.go`
**When** 新增 `enrichWithPomodoroInfo(tasks []*model.Task, workDuration int)` 方法
**Then** 对每个 task 计算 `PlannedPomodoros = ceil(EstimatedTime / (WorkDuration / 60))`（EstimatedTime 单位分钟，WorkDuration 单位秒）
**And** `CompletedPomodoros` 通过 `CountByTaskID(taskID, "work", "completed")` 获取
**And** 只计算 `type='work' AND status='completed'` 的 session，排除 break 和 abandoned

**Given** 任务的番茄钟状态逻辑
**When** 计算 `PomodoroStatus`
**Then** `planned == 0` → `"not_started"`
**And** `completed == 0 && planned > 0` → `"not_started"`
**And** `0 < completed < planned` → `"in_progress"`
**And** `completed == planned` → `"completed"`
**And** `completed > planned` → `"exceeded"`

**Given** `GET /api/tasks` 和 `GET /api/tasks/:id` 端点
**When** 返回任务列表或单个任务
**Then** 响应 JSON 中包含 `planned_pomodoros`、`completed_pomodoros`、`pomodoro_status` 字段
**And** 读取当前用户的 Pomodoro Settings（`WorkDuration`）用于计算

**Given** `backend/internal/service/analytics_service.go`
**When** 新增 `GetPomodoroByTask(period string)` 方法
**Then** 返回按任务聚合的番茄钟统计列表，包含 `task_id`、`task_title`、`planned_pomodoros`、`completed_pomodoros`、`total_focus_minutes`、`status`
**And** 按 `completed_pomodoros` 降序排列

**Given** `GET /api/analytics/pomodoro-by-task` 端点
**When** 请求包含 `period=week|month` 参数
**Then** 返回该时间范围内每个任务的番茄钟统计
**And** 默认 `period=week`

**Given** `backend/internal/service/analytics_service.go`
**When** 新增 `GetPomodoroTrends(period string)` 方法
**Then** 返回每日番茄钟计划 vs 实际对比数据，包含 `date`、`planned`、`actual`、`completed_tasks`、`exceeded_tasks`

**Given** `GET /api/analytics/pomodoro-trends` 端点
**When** 请求包含 `period=week|month` 参数
**Then** 返回该时间范围内每日的趋势数据
**And** `planned` = 当日所有待办任务的 `planned_pomodoros` 之和
**And** `actual` = 当日实际完成的 work 类型 session 数

**Given** `backend/internal/api/handler/analytics_handler.go`
**When** 新增 `GetPomodoroByTask` 和 `GetPomodoroTrends` handler
**Then** 注册到 `GET /api/analytics/pomodoro-by-task` 和 `GET /api/analytics/pomodoro-trends` 路由
**And** 参数验证：`period` 仅接受 `week` 或 `month`

**Requirements covered:** FR1, FR8, NFR1

---

### Story 3.2: 番茄钟完成提醒流程

As a **用户**,
I want **当一个任务的所有计划番茄钟用完时，系统弹出提醒让我选择标记完成或继续计时**,
So that **我不会忘记关闭已完成的任务，也可以灵活地延长工作时间**.

**Acceptance Criteria:**

**Given** 前端 Timer Store 收到 WebSocket 番茄钟完成事件
**When** 完成的是一个 `type='work'` 的 session
**Then** 前端检查当前 session 关联的 task 的 `completed_pomodoros >= planned_pomodoros && planned_pomodoros > 0`
**And** 仅在刚好达到计划数量时触发提醒（`completed == planned`）

**Given** 番茄钟刚好用完
**When** 前端触发提醒
**Then** 弹出 `ElMessageBox` 对话框
**And** 标题：`番茄钟已全部完成`
**And** 内容：`任务「{task_title}」的 {planned}/{planned} 个番茄钟已完成。`
**And** 两个按钮：`标记任务完成`（confirmButtonText）和 `再来一个番茄钟`（cancelButtonText）

**Given** 用户点击「标记任务完成」
**When** `ElMessageBox` 返回 confirm
**Then** 调用 `updateTask(taskId, { status: 'completed' })`
**And** 任务卡片更新为已完成状态（无启动按钮，轻微透明）

**Given** 用户点击「再来一个番茄钟」
**When** `ElMessageBox` 返回 cancel
**Then** 调用 `createSession({ task_id: taskId, type: 'work' })` 创建新的番茄钟
**And** 新 session 开始计时，`completed_pomodoros` 自然递增

**Given** 无预估时间的任务（`planned_pomodoros = 0`）
**When** 任意番茄钟完成
**Then** 不触发提醒对话框

**Given** 已完成的任务（`task.status = 'completed'`）
**When** 番茄钟完成
**Then** 不触发提醒对话框

**Given** 已经超出计划的任务（`pomodoro_status = 'exceeded'`）
**When** 后续番茄钟完成
**Then** 不触发提醒对话框（只在刚好达到边界时触发一次）

**Given** 完成的是 break 类型的 session
**When** session 结束
**Then** 不触发提醒（只在 work session 完成时检查）

**Requirements covered:** FR2, NFR2

---

### Story 3.3: 任务视图 — 卡片进度 + 详情弹窗

As a **用户**,
I want **在任务视图中看到每个任务的番茄钟进度，一键启动番茄钟，点击任务标题查看完整进度和历史**,
So that **我可以直观地追踪任务进展，快速开始专注**.

**Acceptance Criteria:**

**Given** `frontend/src/types/index.ts`
**When** 扩展 Task 类型
**Then** 新增 `plannedPomodoros: number`、`completedPomodoros: number`、`pomodoroStatus: 'not_started' | 'in_progress' | 'completed' | 'exceeded'` 字段
**And** 新增 `PomodoroByTask` 和 `PomodoroTrendDay` 类型用于分析

**Given** 四象限视图中的任务卡片
**When** 任务有预估时间（`plannedPomodoros > 0`）
**Then** 卡片右侧显示 `completed/planned 番茄钟` 文字（如 `2/4 番茄钟`）
**And** 旁边显示 28px 圆形启动按钮（▶，accent color `#B8452C`）
**And** 布局为单行紧凑：`[任务标题] [进度文字] [▶]`

**Given** 任务卡片上的启动按钮
**When** 用户点击 ▶
**Then** 调用 `createSession({ task_id: task.id, type: 'work' })` 启动关联该任务的番茄钟
**And** 按钮变为 ⏸ 状态（如果已有活跃番茄钟）

**Given** 无预估时间的任务（`plannedPomodoros = 0`）
**When** 渲染卡片
**Then** 进度区域显示 `—`，但仍显示启动按钮（自由番茄钟）

**Given** 已完成的任务（`status = 'completed'`）
**When** 渲染卡片
**Then** 无启动按钮，卡片轻微透明
**And** 进度文字显示 `N/N ✓`

**Given** `frontend/src/components/tasks/TaskPomodoroDetail.vue` 新组件
**When** 用户点击任务标题
**Then** 打开详情弹窗，包含：
1. 基本信息（标题、描述、截止日期、象限）
2. 番茄钟进度区域：
   - 进度条（accent color 填充，宽度 = completed/planned 百分比）
   - 文字：`completed/planned` 及百分比
   - 今日历史记录（纯文字，无 emoji）：每个番茄钟的时间段
   - 「开始第 N 个番茄钟」按钮（N = completed + 1）
3. 底部统计：`已专注 X 分钟 · 剩余约 Y 分钟`

**Given** 详情弹窗中的「开始第 N 个番茄钟」按钮
**When** 用户点击
**Then** 创建新番茄钟 session 并自动关闭弹窗

**Given** 详情弹窗组件
**When** 被引用
**Then** 在任务视图和日程视图中共享使用（同一组件）

**Requirements covered:** FR3, FR4, NFR3

---

### Story 3.4: 日程视图 — 快捷启动 + 事件进度

As a **用户**,
I want **在日程视图中快速启动最近任务的番茄钟，并在日历事件上看到番茄钟进度**,
So that **我无需切换页面就能开始专注工作**.

**Acceptance Criteria:**

**Given** 日程视图顶部操作栏
**When** 页面渲染
**Then** 显示「开始番茄」按钮

**Given** 点击「开始番茄」按钮
**When** 无活跃番茄钟且有待办任务
**Then** 自动查找距离当前时间最近的、`status != 'completed'` 且 `estimated_time > 0` 的任务
**And** 启动该任务的番茄钟 session

**Given** 点击「开始番茄」按钮
**When** 已有活跃番茄钟
**Then** 按钮文本变为「查看进行中」
**And** 点击后导航到 Timer 页面

**Given** 点击「开始番茄」按钮
**When** 无待办任务（所有任务已完成或无预估时间）
**Then** 按钮置灰，tooltip 显示「暂无待办任务」

**Given** 日历上的任务类型事件卡片
**When** 事件关联了任务（`event.task_id` 存在）
**Then** 事件卡片上显示番茄钟进度文字（如 `2/4 番茄钟`），与任务卡片样式一致

**Given** 日历事件卡片
**When** 用户点击事件
**Then** 打开 `TaskPomodoroDetail.vue` 详情弹窗（复用 Story 3.3 的共享组件）
**And** 弹窗内可以启动该任务的番茄钟

**Given** AI 生成的 Pomodoro 类型日程事件
**When** 渲染
**Then** 显示关联任务的番茄钟进度
**And** 点击可启动/继续该任务的番茄钟

**Requirements covered:** FR5, FR6

---

### Story 3.5: 分析视图 — 番茄钟统计模块

As a **用户**,
I want **在分析页面查看番茄钟排行榜、计划 vs 实际趋势对比、以及番茄钟完成率**,
So that **我可以了解自己的专注模式，发现哪些任务花费了最多精力**.

**Acceptance Criteria:**

**Given** 分析页面（`Analytics.vue`）
**When** 页面加载
**Then** 在现有统计模块之后新增一个「番茄钟统计」区域，包含三个子模块

**Given** 「番茄钟排行榜」子模块
**When** 渲染
**Then** 显示水平条形图：每行为排名序号 + 任务标题 + 进度条 + 番茄钟数
**And** 条形宽度与番茄钟数成比例
**And** 使用 accent color `#B8452C` 作为条形颜色，`#e8e4df` 作为背景
**And** 支持周/月周期切换

**Given** 「计划 vs 实际趋势」子模块
**When** 渲染
**Then** 显示每日对比柱状图：每天两根柱子
**And** 浅色柱（`#e8e4df`）= 计划番茄钟数
**And** 深色柱（`#B8452C`）= 实际完成数
**And** 包含图例：「计划」和「实际」
**And** 支持最近 7 天 / 30 天切换

**Given** 「番茄钟完成率」子模块
**When** 渲染
**Then** 显示三个环形指标：
  - 按时完成：`completed_pomodoros == planned_pomodoros` 且 `status = 'completed'` 的任务占比
  - 超时完成：`completed_pomodoros > planned_pomodoros` 的任务占比
  - 未完成：`planned_pomodoros > 0 && status != 'completed' && completed_pomodoros < planned_pomodoros` 的任务占比
**And** 支持周/月周期切换

**Given** 分析 API 端点
**When** 前端请求番茄钟统计数据
**Then** 调用 `GET /api/analytics/pomodoro-by-task?period=week` 获取排行榜数据
**And** 调用 `GET /api/analytics/pomodoro-trends?period=week` 获取趋势数据
**And** 完成率从 `pomodoro-by-task` 返回的数据中计算

**Given** 无番茄钟数据时
**When** 请求返回空列表
**Then** 各子模块显示合适的空状态提示（非空白）

**Given** 所有统计模块标题
**When** 渲染
**Then** 不使用装饰性 emoji 或图标，保持纯文字标题（如「番茄钟排行榜」而非「🏆 番茄钟排行榜」）

**Requirements covered:** FR7
