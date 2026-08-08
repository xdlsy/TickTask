# API Key 安全加固设计

**日期**：2026-08-08
**作者**：lsy + Claude（brainstorming session）
**状态**：设计已确认，待实现

## 概述

TickTask 当前的"AI 智能助手"API Key 存在多个暴露面（数据库明文、备份导出默认包含、掩码往返 bug），本设计在不引入"主口令"等反业界惯例的前提下，对 API Key 的存储、传输、使用做端到端加固。

## 背景：现状与威胁模型

### 现状（API Key 全链路）

- **存储双轨**：① `backend/configs/config.yaml` 的 `ai.api_key`（明文，已被 `.gitignore`，但启动后实际不再被使用）；② **SQLite `settings` 表 `key='ai.settings'` 的 JSON `value` 中 `api_key` 字段，明文**。前端 Settings 页保存的 key 进数据库，agent 每轮通过 `llmFactory` 重新查 DB 构造 client。
- **使用**：`OpenAIClient`/`AnthropicClient` 把 key 放进 `Authorization: Bearer ...` / `x-api-key: ...` header，发往用户配置的 `base_url`。代码层面不打日志。
- **环境变量** `TT_AI_API_KEY` 在 config 加载时覆盖 yaml 的 key（`config.go:52`）—— 但运行中没人读这条路径，等同于历史遗留。

### 暴露面（按严重程度）

1. **备份导出默认包含 API Key 明文**：`GET /api/data/export` 默认 `include_api_key=true`（`data.go:29`），一次点击就把明文 key 写进 `ticktask-backup-<ts>.json` 落盘到用户下载目录。Settings.vue 的"导出全部数据"按钮还显式带 `?include_api_key=true`。
2. **掩码往返 bug（功能性 + 安全副作用）**：`GET /api/settings` 返回掩码后的 key（`sk-ab****wxyz`），前端把它当成真 key 装进 `aiSettings.value`。**用户改 model 点保存 → 掩码字符串被回写到 DB，key 当场失效**。`testAIConnection` 也会先保存掩码再测，必然失败。
3. **数据库明文**：`./data/ticktask.db` 文件直接 `sqlite3` 打开就能看到 key。文件被复制、同步云盘、git 误提交都会泄露。
4. **config.yaml 明文**：用户本地填进去后，文件本身就是泄露面（备份/分享/同步）。
5. **HTTP 传输**：dev 跨端口走 HTTP，生产取决于是否反代 TLS（不在代码内）。
6. **CLIClient**：`claude -p <prompt>` 进程参数中带 prompt（数据隐私，非 key 泄露）。

### 威胁模型

**部署形态**：自用但多机同步（DB 文件 / 备份 JSON 会在多台机器、云盘、NAS 之间同步）。

**威胁**：
- 云盘被动泄露（同步服务的账户被盗、共享链接意外公开）
- 备份 JSON 意外外流（分享、上传到 issue、误发到聊天）
- DB 文件被 git 误提交（个人仓库不小心公开）

**非威胁**：
- 本机被恶意软件入侵（任何本地明文/加密都挡不住）
- 服务端被 RCE（同上）
- 物理接触机器

## 设计目标

1. DB 文件被同步/泄露后，API Key 不可直接读
2. 备份 JSON 在任何参数下都不含 API Key 明文
3. 掩码往返 bug 在结构上不可能复现
4. 不引入"主口令"等反业界惯例
5. 换机器时重填 API Key 的体验对标 `gh login` / `aws configure`

## 决策清单（核心 4 项）

| 决策 | 选择 | 理由 |
|------|------|------|
| KEK 来源 | **本地 `.keyvault` 文件**（随机 32 字节，权限 0600） | 简单、故障面小；文件不同步 → DB 同步到另一台机器、没有这个文件 → key 解不开 → 重填 |
| 备份导出 | **导出永不含 Key** | 即使 `?include_api_key=true` 也不导出，最安全 |
| yaml 路径 | **废弃 `ai.api_key`** | 减少泄露面，统一从 Settings 页输入 |
| 测试连接 | **不写库、用临时 key** | 避免掩码往返 bug 的温床；用户不点保存就能验证 |

## 架构与组件

### 新增/修改的文件

```
backend/pkg/vault/                              ← 新增包
├── vault.go                                    ← Vault 接口 + 文件实现
└── vault_test.go
backend/internal/repository/setting_repo.go     ← 修改：加解密 + 迁移
backend/internal/api/handler/setting.go         ← 修改：preview 字段替代掩码
backend/internal/api/handler/agent_handler.go   ← 修改：/test 接受 body
backend/internal/service/data_service.go        ← 修改：导出剔除 encrypted
backend/pkg/config/config.go                    ← 修改：删除 ai.api_key 路径
backend/cmd/server/main.go                      ← 修改：启动期迁移 + vault 注入
backend/.gitignore                              ← 加 .keyvault
backend/configs/config.yaml.example             ← 删 ai.api_key 行
backend/.env.example                            ← 删 TT_AI_API_KEY
frontend/src/views/Settings.vue                 ← 修改：input + 测试连接
frontend/src/types/index.ts                     ← 修改：AISettings 类型
frontend/src/api/agent.ts                       ← 修改：test() 接受 body
```

### 组件职责

| 组件 | 职责 | 接口 |
|------|------|------|
| `vault.Vault` | 加载/生成 `.keyvault`，提供 `Encrypt(plaintext) (nonce, ciphertext, error)` / `Decrypt(nonce, ciphertext) (plaintext, error)` | AES-256-GCM；32 字节 key 文件；首次缺失自动生成 (0600) |
| `SettingRepository` | DB 读写时透明加解密 `api_key`；调用方拿到的 `AISettings.APIKey` 永远是明文（或空） | 现有 4 个方法签名不变；新增迁移方法 `MigrateLegacyAPIKey(vault)` |
| `SettingHandler` | GET 返回 `api_key_set: bool` + `api_key_preview: "sk-ab****wxyz"`；PUT 接受"空=保留原值，非空=更新" | DTO 字段调整 |
| `AgentHandler.testConnection` | `POST /api/agent/test` body `{provider, api_key, base_url, model}` → 构造一次性 client → ping LLM → 不落库 | 改 body 解析 |
| `vault.Init` (启动钩子) | 启动期：①首次生成 .keyvault ②跑 legacy 明文 key 迁移 ③失败 → fail-fast | 在 main.go 调用 |

### 数据流（4 条核心）

**(1) 启动期**：`main.go` → `vault.Init(dataDir)` → `settingRepo.MigrateLegacyAPIKey(vault)` → 注入 vault 到 `settingRepo` 实例。

**(2) 保存 key**：前端 PUT body `{api_key: "sk-xxx", ...}` → handler → repo 检测 `api_key` 非空 → `vault.Encrypt` → DB 存 `api_key_encrypted: {ct, nonce}`。空字段 → 不动 DB 里的 encrypted 字段。

**(3) 读 key**：repo.GetAISettings → 读 `api_key_encrypted` → `vault.Decrypt` → 返回明文给 service/agent。解密失败 → 返回空 key + warn（不崩）。

**(4) 测试连接**：前端 → `POST /api/agent/test` body 含临时 key → handler 用临时 settings 构造 client → 调 `ChatCompletion("ping")` → 返回 ok/latency/error → **完全不碰 DB**。

## DB Schema 变更

**关键设计：`model.AISettings` Go struct 不变，DB JSON shape 解耦。**

```go
// model.AISettings —— 不变，APIKey 字段永远是明文或空
type AISettings struct {
    Provider string `json:"provider"`
    APIKey   string `json:"api_key"`           // 内存中：明文
    BaseURL  string `json:"base_url"`
    Model    string `json:"model"`
}
```

```jsonc
// DB 中 settings.value（key='ai.settings'）—— 形态变了
// 旧（legacy）：
{"provider":"openai","api_key":"sk-xxx","base_url":"...","model":"..."}
// 新：
{"provider":"openai","base_url":"...","model":"...","api_key_encrypted":{"ct":"<b64>","nonce":"<b64>"}}
```

- repo 层 marshal 时**剔除明文 `api_key`**、追加 `api_key_encrypted`
- repo 层 unmarshal 时**忽略 DB 中残留的明文 `api_key`**（防回滚攻击）、解密 `api_key_encrypted`、填到 struct
- 调用方（Schedule/Agent/WorkLog）拿到的 `AISettings.APIKey` 永远是明文 —— **零改动**

### 意外适配：BackupRepository 的 raw JSON 路径

`dataRepository.ReadAll` 当前直接 `json.Unmarshal` DB raw JSON 到 `model.AISettings` struct（绕过 `settingRepository`），这意味着：

- 新版 DB JSON 是 `{"api_key_encrypted": {...}, ...}`，没有 `api_key` 字段
- struct `AISettings{APIKey string}` 没有 `APIKeyEncrypted` 字段，所以 `api_key_encrypted` 被 unmarshal 忽略
- **`BackupData.Settings.AI.APIKey` 天然为空**——导出路径自动满足"导出永不含 key"

因此 `dataService.Export` 当前那段 `if !includeAPIKey && data.Settings.AI != nil { data.Settings.AI.APIKey = "" }` 变成无效但无害的死代码，应该清理掉。`include_api_key` query 参数也可移除（或保留为 no-op 兼容旧前端）。

## 迁移逻辑（启动期一次性）

```
MigrateLegacyAPIKey(vault):
  1. SELECT value FROM settings WHERE key='ai.settings'
  2. unmarshal → map[string]any
  3. if "api_key" in map && map["api_key"] != "":
       - 加密 → 写回 "api_key_encrypted"
       - 删除 "api_key" 字段
       - UPDATE settings SET value=? WHERE key='ai.settings'
       - logger.Info("migrated legacy api_key to encrypted form")
  4. else: 跳过
  5. 失败 → 返回 error，main.go fail-fast
```

只在启动期跑一次。幂等（迁移后再跑会跳过）。

### Import 旧版备份的即时迁移

旧版备份 JSON 含明文 `api_key`。用户通过 `/api/data/import/apply` 导入后，DB 写入的是明文形态（因为 `dataRepository.Apply` 走 raw JSON marshal）。如果等下次启动才迁移，用户在"导入完成 → 重启服务"窗口内调用 AI 功能会失败（settingRepo 新版逻辑忽略明文 `api_key` 字段）。

**解决**：`dataService.ApplyImport` 在事务提交后、返回结果前，调用 `settingRepo.MigrateLegacyAPIKey(vault)`。这样导入完成时 DB 已是加密形态。失败时返回 warning（不阻断 import，因为 key 不是导入的核心目的）。

## 错误处理矩阵

| 情况 | 行为 |
|------|------|
| `.keyvault` 不存在（首次启动） | 生成 32 字节随机 → 写 0600 → 继续 |
| `.keyvault` 存在但权限过松 | warn（不 fail-fast，本机用户独占没强威胁） |
| `.keyvault` 存在但 DB 中 `api_key_encrypted` 用别的 vault 加密（多机同步过来） | repo 返回 `DefaultAISettings()` + warn log "key decryption failed, please re-enter in Settings"；**服务正常启动**，AI 功能不可用直到用户重填 |
| 迁移期写 DB 失败 | fail-fast（DB 未被修改，下次启动重试） |
| `.keyvault` 误同步到云盘 / 误入 git | 文档警告；建议 rotate key；`.gitignore` 已加 |
| `vault.Encrypt` 失败 | 极罕见 → handler 返回 500 |
| `vault.Decrypt` 失败（运行中） | 同"多机同步过来"路径：返回空 key，warn log |

## 掩码往返 bug 的彻底修复

不只是"修 bug"，而是**让 bug 在结构上不可能**：

```ts
// 旧 GET /api/settings 返回：
{ai: {provider, api_key: "sk-ab****wxyz", base_url, model}}
// 前端把这个掩码字符串当真 key 装进 input → 改 model 点保存 → 掩码字符串被回写

// 新 GET /api/settings 返回：
{ai: {provider, base_url, model, api_key_set: true, api_key_preview: "sk-ab****wxyz"}}
// 没有 api_key 字段；前端 input 留空 + placeholder 显示 preview
```

PUT 时 handler 语义：`api_key` 字段缺失或空字符串 = **保留原值**；非空 = **更新**。

额外防线：handler 检测 `api_key` 是否含 `****` 字面量 → 拒绝（防 bug 复发）。

## 前端交互细节（Settings.vue）

```vue
<!-- API Key 输入框：从「填入掩码字符串」改成「空 + placeholder 显示 preview」 -->
<el-input
  v-model="aiSettings.api_key"
  type="password"
  :placeholder="apiKeyPlaceholder"
  show-password
/>
<!-- apiKeyPlaceholder computed:
     - 已配置：'已设置 (sk-ab****wxyz)，留空保留；输入新值覆盖'
     - 未配置：'请输入 API Key' -->
```

```ts
// saveAISettings：api_key 为空 = 保留原值（后端语义已对齐）
async function saveAISettings() {
  await api.updateAISettings(aiSettings.value) // api_key 空字段 → 后端保留
}

// testAIConnection：用当前编辑中的 key（含可能的空），走新 body 接口
async function testAIConnection() {
  if (!aiSettings.value.api_key && !aiSettingsPreview.value) {
    ElMessage.warning('请先输入 API Key'); return
  }
  // 关键：不调 updateAISettings，直接用 form 中的临时 key 测试
  const r = await api.agent.test({
    provider: aiSettings.value.provider,
    api_key: aiSettings.value.api_key,        // 空时后端从 DB 读
    base_url: aiSettings.value.base_url,
    model: aiSettings.value.model,
  })
  // ...显示 ok/error
}
```

**测试连接的兜底语义**：body.api_key 为空时，后端用 DB 中已存的 key（方便用户"已保存的 key 测一下"）；非空时用 body 中的（用户编辑了但还没保存）。两种路径都不写库。

### 类型变更（types/index.ts）

```ts
// 旧
interface AISettings {
  provider: string
  api_key: string      // 掩码往返 bug 的根因
  base_url: string
  model: string
}

// 新
interface AISettings {
  provider: string
  api_key: string              // 仅用于"输入新 key"，GET 时不返回此字段
  base_url: string
  model: string
  api_key_set?: boolean        // GET 时返回
  api_key_preview?: string     // GET 时返回，如 "sk-ab****wxyz"
}
```

## yaml 路径废弃

### 改动

- `backend/configs/config.yaml.example`：删除 `ai.api_key` 行，保留 `provider/base_url/model/timeout`
- `backend/.env.example`：删除 `TT_AI_API_KEY`
- `pkg/config/config.go`：删除 `os.Getenv("TT_AI_API_KEY")` 覆盖逻辑；`AIConfig.APIKey` 字段移除
- `pkg/config/AGENTS.md`：更新文档说明 key 只从 Settings 页面输入

### 对现有用户的影响

| 用户类型 | 影响 |
|---------|------|
| 没填过 yaml key（绝大多数） | 无感，按提示从 Settings 页输入即可 |
| 已在 yaml/env 填了 key 的用户 | 升级后启动期迁移会把 yaml 里的 key 加密落 DB，然后**忽略 yaml 字段**。文档提示"已迁移到加密存储，可从 yaml/env 删除" |

**安全网**：启动期迁移时，如果 yaml/env 中 `TT_AI_API_KEY` 仍存在且 DB 中无 key，**最后再尊重一次**这条路径（写入加密 DB + warn 提示用户删除），避免一刀切破坏既有部署。

### 历史泄露的不可补救性

本设计能阻止**未来的**明文泄露，但**已泄露的**明文 key 无法回收：

- 用户之前点过"导出全部数据"（旧版默认含 key）→ 那份 JSON 文件可能已存档/同步/分享
- 用户之前在 git 中误提交过 `config.yaml` → 历史 commit 中明文 key
- 用户之前在聊天/issue 中粘贴过掩码前的 key

**文档提示**：升级到本设计后，建议**主动 rotate 一次 API Key**（在 LLM 服务商后台撤销旧 key、生成新 key、在 Settings 页填入新 key）。这能让所有历史泄露的 key 失效。

## 范围边界（明确**不做**的事）

| 不做 | 理由 |
|------|------|
| OS keychain 集成 | 多机同步场景下不工作；本地 .keyvault 已够用 |
| 主口令模式 | 业界异类，对个人时间管理工具过度设计 |
| TLS/HTTPS | 部署层责任（反代或 Caddy），不在代码内 |
| 用户认证 | 个人单机/内网工具，不在本任务范围 |
| API Key 旋转/过期 | 个人 key 用户自己 rotate，无需代码支持 |
| 审计日志（谁改了 key） | 单用户场景没意义；多用户时再加 |
| CLIClient 的 prompt 进程参数泄露 | 是数据隐私问题（非 key 泄露），单独议题 |
| `.keyvault` 在 Windows 上的 ACL | 文件权限 0600 在 Windows 效果有限，但本机用户独占场景下没强威胁；后续可加 `_chmod` syscall |
| 加密其他敏感设置 | 当前只有 API Key 一个，Pomodoro/排程偏好非敏感 |

## 测试矩阵

| 文件 | 用例 |
|------|------|
| `backend/pkg/vault/vault_test.go` | ①首次生成文件 ②读取往返 ③Encrypt→Decrypt 还原 ④损坏文件 fail-fast ⑤不同 vault 解不开 |
| `backend/internal/repository/setting_repo_test.go` | ①UpdateAISettings 写入后 DB JSON 不含明文 `api_key` ②GetAISettings 解密往返 ③解密失败返回 default ④MigrateLegacyAPIKey 正向 ⑤迁移幂等 |
| `backend/internal/api/handler/setting_test.go` | ①GET 返回 `api_key_set`/`api_key_preview`、不含 `api_key` ②PUT 空字段=保留 ③PUT 非空=更新 ④PUT 含掩码字符串 `****` → 拒绝 |
| `backend/internal/api/handler/agent_handler_test.go` | `/test` 接受 body、不写库、用临时 key ping（mock LLMClient） |
| `backend/internal/service/data_service_test.go` | ①Export 在任何参数下 `BackupData.Settings.AI.APIKey` 都为空（断言：导出 JSON 无 `api_key` 与 `api_key_encrypted` 键） ②Import 旧版备份（含明文 `api_key`）后调用 ApplyImport，事务结束时 DB 已是加密形态 |
| `frontend/src/views/Settings.spec.ts` | ①加载后 input 为空、placeholder 显示 preview ②空保存成功 ③测试连接走新 body |
| `frontend/src/api/agent.spec.ts` | `agent.test()` 接受 settings body |

## 失败回滚策略

| 失败点 | 回滚 |
|--------|------|
| 迁移期写 DB 失败 | DB 未被修改，下次启动重试 |
| vault 生成失败 | main.go fail-fast，DB 未触碰 |
| 单测发现 schema 回退 | git revert + 重跑迁移 |

## 实现顺序（粗略，等写 plan 时细化）

1. `pkg/vault/` 包 + 单测
2. `setting_repo.go` 加解密 + 迁移方法 + 单测
3. `setting.go` handler DTO + body 语义 + 单测
4. `agent_handler.go` /test body 接受 + 单测
5. `data_service.go` 导出剔除 encrypted + 单测
6. `main.go` vault.Init + 启动期迁移
7. `pkg/config/config.go` 删除 yaml 路径
8. 前端 Settings.vue + types + api client + 单测
9. `.gitignore` + yaml.example + 文档更新
10. 端到端验证：保存/读取/测试/导出/多机同步模拟
