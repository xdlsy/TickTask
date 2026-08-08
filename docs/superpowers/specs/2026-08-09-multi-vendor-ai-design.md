# 多厂商 AI 模型扩展 — Design

- **Date**: 2026-08-09
- **Status**: Approved (pending spec review)
- **Scope**: Settings → AI 智能助手
- **Approach chosen**: 方案 A — 前端预设表 + 后端最小协议映射

## 背景与动机

当前「AI 智能助手」配置只暴露三个服务商：OpenAI、Anthropic、自定义。服务商下拉、`handleProviderChange` 的默认值、模型预设列表都硬编码在前端 `Settings.vue` 里；后端 `ai.NewClientFromSettings` 用一个 `switch settings.Provider` 把 `openai`/`custom` 路由到 `OpenAIClient`、`anthropic` 路由到 `AnthropicClient`、`claude`/`cli` 路由到 `CLIClient`。

这套机制只支持两家大厂，且每加一家都要在前端再堆一组 if/else 和数组——这正是用户说的「配置太简单」。

**关键洞察**：后端只有**两套真实的 HTTP 协议族**——OpenAI 兼容（`/chat/completions` + `Authorization: Bearer` + `choices[].message`）和 Anthropic 兼容（`/v1/messages` + `x-api-key` + `anthropic-version` + `content[]` blocks），外加不走 HTTP 的 `CLIClient`。绝大多数主流模型厂商都复用这两套协议：

- DeepSeek / 通义千问(Qwen) / 智谱(GLM) / 月之暗面(Kimi) → **OpenAI 兼容**
- MiniMax → **Anthropic 兼容**（官方文档：base URL `https://api.minimaxi.com/anthropic`，`/v1/messages` + `x-api-key` + `anthropic-version`，完整支持 `tool_use`）

因此「多厂商」主要是**预设注册表**（展示名 + 协议 + 默认 baseURL + 模型列表）叠加在现有两个客户端之上，而非 N 个独立客户端。

### 顺手修掉的 latent bug

前端 `handleProviderChange` 把 Anthropic 的 `base_url` 填成 `https://api.anthropic.com/v1`，而 `AnthropicClient.ChatCompletion` 会再拼 `/v1/messages` → 出现双 `/v1`。后端 `NewAnthropicClient` 的默认值 `https://api.anthropic.com`（不带 `/v1`）本就正确，所以 bug 仅在 UI 选 Anthropic 时触发。本设计用注册表统一管理 baseURL，该 bug 自然消除。

## 目标

1. 新增 5 家国内主流厂商为一等公民预设：**DeepSeek、通义千问、智谱 GLM、月之暗面 Kimi、MiniMax**（保留 OpenAI / Anthropic / 自定义）。
2. 选中厂商即自动填好默认 baseURL + 预设模型下拉，仍允许手填自定义模型名。
3. 单一全局模型——所有 AI 功能（任务分类、排程、Agent 对话、工作日志）共用一份配置，仅扩展可选厂商。
4. 后端零新增客户端，仅加一个协议派生函数；前端用一张预设表驱动 UI，替换散落的硬编码。
5. 向后兼容：存量用户存储的 `openai`/`anthropic`/`custom` 值继续可用，无需数据迁移。

## 非目标（明确不做）

- **按功能分配不同模型**（用户已选「单一全局模型」）。
- **动态拉取厂商 `/models` 列表**：预设模型静态维护（YAGNI，模型名会漂移，动态拉取还要处理鉴权/缓存）。
- **自定义协议开关**：`custom` 维持 OpenAI 兼容（与今天一致）。MiniMax 证明 Anthropic 兼容的自定义端点确有需求，但当前不实现；日后若需要，给 `AISettings` 加一个可选 `protocol` 字段，`protocolFor` 优先读它即可。

## 设计

### § 1. 数据模型与存储

`model.AISettings` 字段不变，但 `Provider` 语义从「协议名」升级为「厂商 id」：

```go
type AISettings struct {
    Provider string `json:"provider"` // 厂商 id: openai|anthropic|deepseek|qwen|zhipu|moonshot|minimax|custom|cli
    APIKey   string `json:"api_key"`
    BaseURL  string `json:"base_url"`
    Model    string `json:"model"`
}
```

- 协议**不存储**，由 `protocolFor(provider)` 派生。
- `DefaultAISettings()` 保持 `openai` + `gpt-4o-mini` 不变。
- **向后兼容**：`openai`/`anthropic`/`custom`/`claude`/`cli` 在派生表中全部有效。
- **BaseURL 约定**（写进注册表文件注释，前后端共同遵守）：
  - **OpenAI 系**：baseURL **带**版本路径，客户端拼 `/chat/completions`（如 `https://api.openai.com/v1`）。
  - **Anthropic 系**：baseURL **不带** `/v1`，客户端拼 `/v1/messages`（如 `https://api.anthropic.com`、`https://api.minimaxi.com/anthropic`）。

前端 `AISettings` 类型（`types/index.ts`）保持 `provider: string`，无需改类型；语义文档化即可。

### § 2. 后端：协议路由（`backend/internal/ai/client.go`）

新增派生函数，`NewClientFromSettings` 改为按派生协议选客户端：

```go
// protocolFor 把厂商 id 映射到三套实现之一。已知厂商走预设协议；
// 未知值（含 custom 与空串）回退到 OpenAI 兼容，与历史行为一致。
func protocolFor(provider string) string {
    switch provider {
    case "claude", "cli":
        return "cli"
    case "anthropic", "minimax":
        return "anthropic"
    default: // openai, deepseek, qwen, zhipu, moonshot, custom, "" → OpenAI 兼容
        return "openai"
    }
}

func NewClientFromSettings(settings *model.AISettings) LLMClient {
    if settings == nil {
        return nil
    }
    switch protocolFor(settings.Provider) {
    case "cli":
        return NewCLIClient()
    case "anthropic":
        if settings.APIKey == "" {
            return nil
        }
        return NewAnthropicClient(settings.APIKey, settings.BaseURL, settings.Model)
    default: // openai
        if settings.APIKey == "" {
            return nil
        }
        return NewOpenAIClient(settings.APIKey, settings.BaseURL, settings.Model)
    }
}
```

`OpenAIClient` / `AnthropicClient` / `CLIClient` **零改动**。`NewClientFromSettings` 的「nil 表示未配置」「空 key 返回 nil」语义保持不变——这同时守住了 agent 服务的 TestConnection 临时客户端路径（`main.go` 注释提到的复用点）。

> `/v1` bug 修复在前端（§3），后端默认值本就正确。

### § 3. 前端：厂商预设注册表

新文件 `frontend/src/utils/aiVendors.ts`：

```ts
export type AIProtocol = 'openai' | 'anthropic'

export interface VendorPreset {
  id: string            // 存入 AISettings.provider 的厂商 id
  label: string         // 下拉展示名
  protocol: AIProtocol
  baseURL: string       // 按 §1 约定
  models: string[]      // 预设模型；模型选择框仍 allow-create + filterable
  hint?: string         // baseURL 输入框下方的提示文案
}

export const VENDOR_PRESETS: VendorPreset[] = [ /* §4 的 8 项 */ ]
export const DEFAULT_VENDOR = 'openai'

export function getVendorPreset(id: string): VendorPreset | undefined {
  return VENDOR_PRESETS.find(v => v.id === id)
}
```

`Settings.vue` 改动（AI 智能助手卡片内）：

- 服务商 `el-select` 的 options 由 `VENDOR_PRESETS` 生成；删除模板里手写的三个 `el-option`。
- `handleProviderChange()` 改为查 `getVendorPreset(provider)`：存在则填入 `baseURL` + `models[0]`（默认模型），不存在（custom）则清空两者。
- `availableModels` 由 `getVendorPreset(provider)?.models ?? []` 派生；删除 `openaiModels` / `anthropicModels` 数组。
- `modelPlaceholder` 按厂商 label 拼提示。
- baseURL 输入框**对所有厂商可见**（方便填代理），`v-if` 条件放宽；提示文案用 preset.hint 区分。
- Anthropic 的 baseURL 默认改为 `https://api.anthropic.com`（修 `/v1` bug）。
- 模型选择框保持 `allow-create + filterable`，可手填任意模型名。
- 密码框、测试连接、空 key = 保留原 key 的语义、`api_key_preview` 机制——全部不动。

### § 4. 预设内容（8 项）

| id | label | protocol | baseURL | 预设模型 |
|---|---|---|---|---|
| `openai` | OpenAI | openai | `https://api.openai.com/v1` | gpt-4o-mini, gpt-4o, gpt-4.1-mini, gpt-4.1 |
| `anthropic` | Anthropic | anthropic | `https://api.anthropic.com` | claude-sonnet-4-6, claude-3-5-sonnet-latest, claude-3-5-haiku-latest |
| `deepseek` | DeepSeek | openai | `https://api.deepseek.com/v1` | deepseek-chat, deepseek-reasoner |
| `qwen` | 通义千问 | openai | `https://dashscope.aliyuncs.com/compatible-mode/v1` | qwen-plus, qwen-max, qwen-turbo, qwen-long |
| `zhipu` | 智谱 GLM | openai | `https://open.bigmodel.cn/api/paas/v4` | glm-4-plus, glm-4-air, glm-4-flash, glm-4.5 |
| `moonshot` | 月之暗面 Kimi | openai | `https://api.moonshot.cn/v1` | moonshot-v1-8k, moonshot-v1-32k, moonshot-v1-128k, kimi-k2-0905-preview |
| `minimax` | MiniMax | anthropic | `https://api.minimaxi.com/anthropic` | MiniMax-M3, MiniMax-M2.7, MiniMax-M2.7-highspeed, MiniMax-M2.5 |
| `custom` | 自定义 | openai | （空） | （空，手填） |

> **校验要求**：MiniMax 的 baseURL/模型来自官方文档（已确认）。DeepSeek/Qwen/智谱/Kimi 的 baseURL 与模型名是最佳努力值，**实现时对照各厂商当前文档逐一校验**（模型名会随版本漂移）。Anthropic 默认模型对齐后端 `NewAnthropicClient` 的 `claude-sonnet-4-6`。

## 实现清单（文件级）

**后端**
- `backend/internal/ai/client.go` — 新增 `protocolFor`；改写 `NewClientFromSettings` 的 switch。
- `backend/internal/ai/*_test.go`（扩展或新增）— `protocolFor` 表驱动测试 + `NewClientFromSettings` 类型断言。
- `backend/internal/ai/AGENTS.md` — 更新（现仅写 OpenAI 客户端，已过期）：补 Anthropic/CLI 客户端、工具调用、厂商协议路由、baseURL 约定。

**前端**
- `frontend/src/utils/aiVendors.ts`（新）— `VendorPreset` 类型、`VENDOR_PRESETS`、`getVendorPreset`。
- `frontend/src/views/Settings.vue` — 服务商下拉 / `handleProviderChange` / `availableModels` / `modelPlaceholder` / baseURL 框可见性，全部改为读注册表。
- `frontend/src/views/Settings.spec.ts` + `Settings.test.ts` — 更新厂商下拉、自动填充、模型列表、custom 手填的用例。

**不动**
- `model/setting.go` 的 `AISettings` 结构（语义升级，字段不变）。
- `OpenAIClient` / `AnthropicClient` / `CLIClient` 实现。
- API key 加密存储、`api_key_preview`、TestConnection 路径、Agent 服务的工具调用流程。

## 测试计划

**后端**（Go，扩展现有 `ai` 包测试）
- 表驱动 `protocolFor`：覆盖 `openai`/`anthropic`/`deepseek`/`qwen`/`zhipu`/`moonshot`/`minimax`/`custom`/`claude`/`cli`/`""` → 预期协议。
- `NewClientFromSettings`：对每个厂商 id 断言返回的客户端类型（`*OpenAIClient` / `*AnthropicClient` / `*CLIClient`），以及空 key 返回 nil。这条测试同时充当「协议表覆盖所有已知厂商」的漂移守卫——加厂商时若忘了进 `protocolFor` 会在此失败。

**前端**（Vitest，更新现有 Settings 测试）
- 服务商下拉渲染 8 项。
- 选中某厂商（如 minimax）→ baseURL 与默认模型自动填好，模型下拉匹配预设。
- 选 `custom` → baseURL/模型为空，模型框可手填（allow-create）。
- 现有用例（保存、测试连接、空 key 保留）回归通过。

## 验收标准

1. 设置页服务商下拉出现 8 项；选中任一预设厂商，baseURL 与默认模型自动填入。
2. 任选一家国内厂商 + 填入其真实 API Key，点「测试连接」返回成功（端到端：`lsof -ti:8080 | xargs kill -9` → `go run` → 刷新前端 → 选 MiniMax 测试）。
3. 选 Anthropic 后 baseURL 为 `https://api.anthropic.com`（无双 `/v1`），测试连接成功。
4. `go test ./internal/ai/...` 与前端 `npx vitest run` 全绿。
5. 存量配置（provider=openai/anthropic/custom）无需迁移即正常工作。

## 风险与备注

- **模型名漂移**：预设模型名可能过期。缓解：模型框 allow-create 允许手填；实现时对照官方文档校验；非目标里已排除动态拉取。
- **baseURL 约定分歧**：两套协议对 baseURL 是否带 `/v1` 的要求相反，配错会 404。缓解：约定写进注册表注释 + 后端测试隐式覆盖（每厂商按其协议返回正确客户端）。
- **MiniMax 工具调用多轮**：官方要求把完整 assistant 消息（含 thinking/text/tool_use 块）原样回带以保持思维链连续。当前 Agent 服务的消息重建逻辑是否已满足，需在实现/验收时针对 MiniMax 复核一次（属 Agent 流程，不在本次结构改动范围，但列入验收核对项）。
