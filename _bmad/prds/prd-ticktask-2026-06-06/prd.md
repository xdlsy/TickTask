---
title: "PRD: AI Agent 可配置能力"
status: final
created: 2026-06-06
updated: 2026-06-06
---

# PRD：AI Agent 可配置能力

## 概述

TickTask 当前在日程生成和修订时**硬编码调用 `claude -p` CLI 命令**，用户无法切换 AI 后端、无法定制 Prompt、也无法在 UI 中管理自己的习惯偏好（只能手动编辑 `config/habit.md` 文件）。本 PRD 将 AI 相关配置全面开放给用户，在设置界面提供统一的可视化管理入口。

**核心目标**：让用户完全掌控 TickTask 的 AI 行为——用什么后端、写什么 Prompt、配什么习惯，一切可改、可导出。

---

## 功能需求

### 功能组 A：习惯与偏好配置

用户可在设置界面可视化配置个人习惯，替代手动编辑 `config/habit.md` 的体验。

**FR-A1：工作时间偏好**
- 每日默认工作起止时间（如 09:00–18:00）
- 午休时间范围（如 12:00–13:00），AI 安排日程时保护该时段不放入任务
- 每日深度工作时间偏好（上午/下午/无偏好）
- 浅层工作时间偏好（上午/下午/无偏好）

**FR-A2：精力匹配规则**
- 是否启用精力匹配（开关）：开启后，高专注任务（deep_work）优先安排在深度工作时间，低专注任务（shallow_work）安排在浅层时间
- 周五下午保护（开关）：周五下午尽量避免安排高难度任务

**FR-A3：任务时长偏好**
- 默认任务时长（分钟，默认 60）
- 最小任务时长（分钟，默认 30）——AI 不会生成短于此值的日程块
- 任务间缓冲时间（分钟，默认 15）——相邻任务之间的最小间隔

**FR-A4：Prompt 模板编辑**
- 提供默认 Prompt 模板（基于现有 `auto-schedule/SKILL.md` 中的系统指令提炼）
- 用户可在 textarea 中完整编辑 Prompt 模板
- 模板支持变量占位符，AI 执行时自动替换：
  - `{{tasks}}` — 当前任务列表
  - `{{habits}}` — 用户习惯配置
  - `{{week_start}}` / `{{week_end}}` — 当前周起止日期
  - `{{learning_log}}` — 历史学习记录
- 「重置为默认值」按钮，恢复出厂 Prompt
- Prompt 模板与日程生成和修订共用

### 功能组 B：Agent 后端配置

用户可选择 AI 执行后端，一个版本支持三种方式。

**FR-B1：后端类型选择**
- 三种后端可选：
  - **Claude CLI**（`claude -p`）——当前默认方式，通过本地安装的 Claude CLI 执行
  - **OpenCode CLI**（`opencode run`）——通过本地安装的 OpenCode CLI 执行
  - **API 直连**——用户填写 API Key 和 Endpoint，直接 HTTP 调用
- 后端选择对日程生成和修订均生效（统一由该后端执行）

**FR-B2：Claude CLI 配置**
- Claude 可执行文件路径（默认 `claude`，即从 PATH 查找）
- 附加参数（默认 `--output-format stream-json --verbose --permission-mode acceptEdits --dangerously-skip-permissions`，用户可覆盖）
- 「测试连接」按钮：执行 `claude --version`，检查是否可用

**FR-B3：OpenCode CLI 配置**
- OpenCode 可执行文件路径（默认 `opencode`）
- 模型选择（下拉，具体选项取决于 opencode 支持的模型列表）
- 附加参数（用户自定义）
- 「测试连接」按钮：执行 `opencode --version` 或等效命令

**FR-B4：API 直连配置**
- Provider 选择：OpenAI / Anthropic / 自定义
- API Key（密码输入框）
- Base URL（OpenAI 默认 `https://api.openai.com/v1`，Anthropic 默认 `https://api.anthropic.com/v1`，自定义可自由填写）
- Model（下拉+可自定义输入，根据 Provider 提供预设列表）
- 「测试连接」按钮：发送最小请求验证 API Key 有效

**FR-B5：后端切换行为**
- 切换后端时，之前后端的配置保留不丢失
- 日程生成和修订统一使用当前选中的后端
- 现有仅安装了 Claude CLI 的用户升级后无需任何改动（Claude CLI 为默认后端）

### 功能组 C：配置导出/导入

**FR-C1：导出配置**
- 在设置页面提供「导出配置」按钮
- 导出内容为一个 JSON 文件，包含：
  - 习惯配置（FR-A1 至 FR-A3）
  - Prompt 模板（FR-A4）
  - Agent 后端配置（FR-B1 至 FR-B4，**不含 API Key**）
  - 导出元信息（版本号、导出时间）
- 文件名格式：`ticktask-config-YYYY-MM-DD.json`
- API Key 明确不导出，导入后需重新填写

**FR-C2：导入配置**
- 在设置页面提供「导入配置」按钮
- 选择 JSON 文件后，预览将要覆盖的配置（展示新旧值对比或摘要）
- 用户确认后应用
- 导入失败（格式错误、版本不兼容）时显示明确错误信息，不回滚已导入的部分 [ASSUMPTION: 导入为原子操作——全部成功或全部拒绝]

---

## 非功能需求

**NFR-1：安全性**
- API Key 存储在本地 SQLite 数据库中，明文字段保持现有设计 [ASSUMPTION: 本地单用户应用，数据库文件级别安全即可]
- 导出文件不包含 API Key
- API Key 在设置界面中始终以密码遮蔽方式显示

**NFR-2：向后兼容**
- 升级后现有用户无需任何迁移操作，默认后端为 Claude CLI
- 现有 `config/habit.md` 在手写后仍可被 AI 读取（手动文件优先于 UI 配置）[ASSUMPTION: 当 `habit.md` 存在且用户未在 UI 中配置时，沿用文件内容；UI 配置后写入 `habit.md` 覆盖]

**NFR-3：用户体验**
- 所有配置修改即时生效，无需重启后端
- 「测试连接」按钮 5 秒内返回结果
- 导出/导入操作在 3 次点击内完成

**NFR-4：错误处理**
- AI 后端不可用时（如 CLI 未安装、API Key 无效），日程生成界面显示明确的错误提示
- 回退机制：生成失败时已有日程数据不受影响

---

## 技术约束

- 三种后端统一抽象为一个 `AgentRunner` 接口，接受 Prompt 字符串、返回流式输出，`ScheduleService` 面向接口编程而非硬编码 `claude -p`
- Prompt 模板变量替换发生在后端，前端所见即所得
- 导出 JSON 的 schema 需要版本号字段，便于未来兼容
- 复用现有设置页面的 `PUT /api/settings` 系列接口，扩展新字段

---

## API 规格（新增/修改）

### `PUT /api/settings/agent`（新增）

```json
{
  "backend": "claude_cli | opencode_cli | api",
  "claude_cli": { "path": "claude", "extra_args": "..." },
  "opencode_cli": { "path": "opencode", "model": "...", "extra_args": "..." },
  "api": { "provider": "anthropic", "api_key": "sk-ant-...", "base_url": "...", "model": "claude-sonnet-4-6" },
  "habits": {
    "work_start": "09:00",
    "work_end": "18:00",
    "lunch_start": "12:00",
    "lunch_end": "13:00",
    "deep_work_preference": "morning",
    "shallow_work_preference": "afternoon",
    "energy_matching_enabled": true,
    "friday_afternoon_protection": true,
    "default_duration_minutes": 60,
    "min_duration_minutes": 30,
    "buffer_minutes": 15
  },
  "prompt_template": "You are a schedule assistant... {{tasks}} ... {{habits}} ..."
}
```

### `POST /api/settings/agent/test`（新增）

```json
// Request
{ "backend": "claude_cli", "claude_cli": { "path": "claude" } }

// Response 200
{ "ok": true, "message": "claude version 2.0.0" }

// Response 200
{ "ok": false, "message": "claude: command not found" }
```

### `GET /api/settings/agent/export`（新增）

返回完整配置 JSON 文件下载（不含 API Key）。

### `POST /api/settings/agent/import`（新增）

接受 JSON 文件上传，验证后写入数据库。

---

## 待确认问题

1. **Q: `habit.md` 迁移策略？** 已有用户手写了 `config/habit.md`，UI 配置上线后如何处理？[ASSUMPTION] 首次进入设置页面时，如果数据库无习惯配置但 `habit.md` 存在，后端解析 `habit.md` 回填到 UI 中，用户后续通过 UI 修改。

2. **Q: CLI 路径默认值？** `claude` 和 `opencode` 默认从 PATH 查找，但 Windows 用户可能安装在不同位置。[ASSUMPTION] 默认 `claude` / `opencode`，用户可按需填绝对路径。

---

## 成功指标

- 用户首次完成 Agent 配置（从打开设置到「测试连接」成功）不超过 2 分钟
- 配置导出/导入往返无数据丢失（除 API Key 外）
- 三种后端均能成功生成一份有效的七日 ICS 日程
