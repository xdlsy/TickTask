# Agent 端到端验证设计

- **日期**: 2026-08-09
- **状态**: Draft（待实现计划）
- **主题**: 如何验证 TickTask 对话式 Agent 的真实端到端行为（例：「我一会有啥安排吗？」→ 拿到今天的安排）

## 1. 背景与问题

TickTask 已有一个完整的对话式 Agent（`backend/internal/agent/`）：

- 8 个 REST 端点（`/api/agent/*`）；`POST /chat` 返回 202，后台 goroutine 跑 `runTurn`。
- 13 个工具（任务/番茄钟/日程/分析/工作日志），三级权限 `PermRead`（自动执行）/`PermWrite`（确认）/`PermDangerous`（二次确认）。
- WebSocket 流式事件：`agent_message`（文本增量）、`agent_tool`（工具调用与状态）、`agent_done`（回合结束）。
- 决策发生在 **Go 进程内**：`runTurn`（`service.go:85`）调用 `LLM.ChatWithTools` 后在进程内分发工具，外部仅能通过 WS 广播观测。

现有测试覆盖：

- **L0**（工具单元，`tools/*_test.go`）与 **L1**（编排，`service_test.go` 用 `mockLLM` + `mockHub` + 真实 SQLite）都已存在。
- L1 用预设的 `ToolResponse` 队列驱动，**LLM 是假的**——它证明「程序写对了吗」，但证明不了「模型是否会把『我一会有啥安排吗？』翻译成 `list_schedule(date=today)`」。
- **没有任何 Agent 的端到端测试。**

因此真正缺失的是验证 **「自然语言 → 正确工具 + 参数」** 这一层。这是 LLM Agent 验证中最棘手的一层：模型决策天生概率性，无法 100% 确定性断言。

## 2. 目标与非目标

### 目标

- 建立分层验证金字塔，覆盖从确定性管线到真实模型决策再到浏览器纵向的全链路。
- 让「我一会有啥安排吗？」这类典型意图可以被**重复、可观测、可打分**地验证。
- 引入单一内部录点（`TraceRecorder`），同时服务「断言」（L2）与「可观测」（L3）。
- 复用业界专门工具（Promptfoo / Langfuse），避免手搓其弱化版。

### 非目标

- 不追求 100% 确定性断言模型自由文本输出（不可能）。
- 不引入 Python 生态工具（Inspect AI / DeepEval）——语言与形状错配，胶水多于价值。
- 不引入 SaaS（LangSmith）——选 Langfuse 自托管，避免数据外送与账号依赖。
- 不在本期实现大批量自动生成的 eval 数据集——12 条手写意图变体足够。
- 不改动 Agent 的运行时业务行为；`TraceRecorder` 对生产路径是 nil 安全的旁路。

## 3. 总体架构：四层金字塔

```
                    ┌──────────────────────────────────────────────┐
   每次 PR（确定性）  │ L0 工具单元 (Go)    L1 编排 mockLLM (Go)       │ 守「程序写对了吗」
                    └──────────────────────────────────────────────┘
                    ┌──────────────────────────────────────────────┐
   改 prompt/工具    │ L2  Promptfoo (JS)                            │ 守「模型听懂人话了吗」
                    │  自定义 WS provider + 12 条意图 + 断言 tool call │
                    └──────────────────────────────────────────────┘
                    ┌──────────────────────────────────────────────┐
   任何时候（可观测） │ L3  TraceRecorder → Langfuse（jsonTracer 后备）│ runTurn 内录点
                    └──────────────────────────────────────────────┘
                    ┌──────────────────────────────────────────────┐
   想肉眼看全链路    │ 纵向 Playwright (gated)  浏览器→LLM→DB→WS       │
                    └──────────────────────────────────────────────┘
```

**核心洞察**：对 tool-calling agent，断言「工具调用」这个结构化事实比断言自由文本稳定一个数量级。例：「模型调用了 `list_schedule` 且 `from ≤ today ≤ to`」是近确定性的；而「你今天有 3 个安排…」每次措辞都变。

**工具选型理由**（每个工具按 TickTask 真实接缝评估）：

| 工具 | 接缝 | 裁决 |
|---|---|---|
| Inspect AI | Python solver 驱动 agent；本 Agent 是远端 Go 服务，需写 HTTP/WS solver 胶水 | ❌ 形状不对 |
| DeepEval | pytest / Python | ❌ 语言错配 |
| **Promptfoo** | JS，驱动 HTTP boundary，原生断言 tool call，跨 run 打分 diff；仓库已有 node | ✅ **L2** |
| **Langfuse** | 你主动 emit span 的 trace 后端；自托管开源 | ✅ **L3 sink** |
| LangSmith | SaaS trace 后端 | ❌ 数据外送，选 Langfuse |
| record/replay | 模式，非工具；被 jsonTracer 文件 + Langfuse dataset 覆盖 | ◻️ 一种模式 |

## 4. 组件设计

### 4.1 `TraceRecorder`（L3 录点；唯一 Go 运行时改动）

复刻现有 `HubBroadcaster`（`service.go:19`）的注入模式，在 `agent` 包新增：

**新文件 `backend/internal/agent/trace.go`**

```go
type TraceRecorder interface {
    RecordTurn(convID string, trace TurnTrace)
}

type TraceStep struct {
    ToolCall   ai.ToolCall   // 模型请求的 ID/Name/Args —— 决策证据
    Permission ToolPermission
    Status     string        // started | pending_confirmation | rejected | succeeded | failed
    Result     any
    Error      string
}

type TurnTrace struct {
    ConversationID string
    UserText       string
    AssistantText  string   // 跨多次 ChatWithTools 的内容拼接
    Steps          []TraceStep
}
```

**默认实现 `noopTracer`**：`RecordTurn` 空实现。`AgentDeps.Tracer` 为 nil 时服务不记录，**现有 L1 测试零改动**（与 `SettingsRepo` 同样的 nil 安全约定）。

**接线（`service.go`）**：

- `AgentDeps`（`service.go:23`）新增字段 `Tracer TraceRecorder`。
- `NewAgentService`（`service.go:68`）在 `Tracer == nil` 时注入 `noopTracer{}`。
- `runTurn`（`service.go:85`）在入口处声明局部 `var trace TurnTrace`，`UserText` 取自已加载历史（`service.go:95` 的 `LoadRecentMessages` 最近一条 user 消息，避免改签名）；`defer` 中 `s.Tracer.RecordTurn(convID, trace)`，保证所有返回路径（stop / error / max_tools / ctx 取消）都落 trace。
- 在工具分发循环（`service.go:119-184`）内，每个 `resp.ToolCalls[i]` 产生一个 `TraceStep`：`ToolCall`/`Permission` 在决策点填入，`Status`/`Result`/`Error` 在对应执行/确认分支填入。
- 多次 `resp.Content`（`service.go:105`）拼接到 `trace.AssistantText`。

**两个实现**：

- `jsonTracer`（Phase 1，随 L2 交付）：把每次 turn 的 `TurnTrace` 以 JSONL 追加写到 `backend/testdata/traces/<conversation_id>.jsonl`（每个 turn 一行，利于跨 run diff）。零运维，**同时是将来 record/replay 的 cassette 源**。
- `langfuseTracer`（Phase 2）：纯 HTTP POST 到 Langfuse ingestion API（`/api/public/ingestion`），**不引入 Go SDK**，靠 env `LANGFUSE_PUBLIC_KEY` / `LANGFUSE_SECRET_KEY` / `LANGFUSE_BASE_URL` gate。把每个 `TraceStep` 映射为一个嵌套 span。

**接口前向兼容说明（去歧义）**：v1 接口为「整条 `TurnTrace` 在 turn 结束时一次性 `RecordTurn`」。Langfuse 的实时 span 流式推送是未来演进方向；若 Phase 2 需要，再扩展 `RecordStep`，不破坏 v1 调用方。

**`main.go` 选择策略**：按 env 优先级 `langfuseTracer` > `jsonTracer` > `noopTracer`（默认）。

### 4.2 L2：Promptfoo eval（顶层新目录 `eval/`）

```
eval/
├── package.json          # 依赖：promptfoo, ws
├── promptfooconfig.yaml  # 自定义 provider + 12 条 case + 断言规则
├── provider.mjs          # 自定义 WS provider
├── seed.mjs              # 前置 seed 今天的日程/任务（强特征标题）
└── README.md             # 如何运行
```

**`provider.mjs` —— 自定义 provider 契约（黑盒观测 WS）**：

Promptfoo 调用 provider 的 `callApi(prompt)`，返回供断言的 JSON。流程：

1. `POST http://localhost:8080/api/agent/conversations` → 解析 `{ id }`。
2. 开 `ws://localhost:8080/ws`；只处理 `payload.conversation_id === id` 的事件。
3. `POST http://localhost:8080/api/agent/chat`，body `{ conversation_id: id, text: prompt }`（返回 202）。
4. 累积：
   - `assistant_text` ← 每个 `agent_message` 事件的 `delta_text` 拼接。
   - `tool_calls[]` ← 每个 `agent_tool` 事件 `{ name: tool_name, args, status }`。
5. 收到匹配的 `agent_done` → 关 WS，返回 `JSON.stringify({ tool_calls, assistant_text })`。
6. 超时（建议 60s）未收到 `agent_done` → 返回 `{ tool_calls, assistant_text, error: "timeout" }`，让断言失败而非挂死。

**`seed.mjs` —— 前置数据**：通过现有 schedule / task REST 接口创建今天的强特征数据，使断言可引用关键词。固定标题（确定性）：

- 日程：`审查 PR-1234`（09:00）、`和 Alice 1:1`（14:00）、`发布 v2.3`（17:00）。
- 任务：`整理周报`（todo）、`修复登录 bug`（in_progress）。

具体 REST 路径在实现计划阶段确认（schedule / task handler 已存在 CRUD）。

**断言策略**：Promptfoo custom assert（JS 函数）+ 内置 `icontains`。

- 结构断言（custom assert）：检查 `tool_calls` 含某工具名，且参数满足条件（例：`list_schedule` 的 `args.from ≤ today ≤ args.to`）。
- 文本弱断言（`icontains`）：`assistant_text` 命中 ≥1 个 seeded 关键词；且不含道歉性词（`没有|无法|不知道|未配置`）。

**12 条意图 case**（canonical + 11）：

| # | 用户输入 | 期望工具 | 权限行为 | 关键断言 |
|---|---|---|---|---|
| 1 | 我一会有啥安排吗？ | `list_schedule` | 自动执行 | tool 日期覆盖今天 + 文本含 seeded 关键词 |
| 2 | 今天还剩什么安排？ | `list_schedule` | 自动执行 | 同上，from=today |
| 3 | 我有哪些没做的任务？ | `list_tasks` | 自动执行 | status=todo/in_progress |
| 4 | 下一个任务是啥？ | `list_tasks` | 自动执行 | 工具被调用 |
| 5 | 我今天专注了多久？ | `get_daily_insights` | 自动执行 | 工具被调用 |
| 6 | 番茄钟现在啥状态？ | `get_timer_status` | 自动执行 | 工具被调用 |
| 7 | 帮我建个任务：周报 | `create_task` | 待确认 | status=`pending_confirmation` |
| 8 | 把「整理周报」标完成 | `update_task` | 待确认 | status=`pending_confirmation` |
| 9 | 删掉任务「修复登录 bug」 | `delete_task` | 二次确认 | status=`pending_confirmation` |
| 10 | 开始一个番茄钟 | `start_pomodoro` | 待确认 | status=`pending_confirmation` |
| 11 | 我今天有啥洞察？ | `get_daily_insights` | 自动执行 | 工具被调用 |
| 12 | 帮我把这段工作记成日志：... | `structure_worklog` | 自动执行 | 工具被调用 |

**打分与门槛建议**：每条 case 三类断言全中算 pass；suite 输出 pass-rate，门槛建议 ≥0.9。**不进默认 `go test` / CI**——改 prompt 或改工具时手动跑，或 nightly。

### 4.3 L3：Langfuse（分阶段）

- **Phase 1**（随 L2 交付）：`jsonTracer`，trace 文件即观测产物 + 将来 cassette 源。零运维。
- **Phase 2**（想要 UI/dataset 时）：`langfuseTracer` + 根目录新增 `docker-compose.langfuse.yml`（postgres + langfuse）。在 Langfuse UI 中查看每次 turn 的工具调用 / 结果 / 耗时，建 dataset，对 trace 打分。

### 4.4 纵向 e2e：Playwright（gated）

**新文件 `frontend/tests/e2e/agent-llm.spec.ts`**：

- `process.env.LLM_E2E !== '1'` 时 `test.skip`（默认不在 e2e 套件里跑）。
- 前置：后端运行 + AI key 已配 + `seed.mjs` 已 seed。
- 步骤：打开 AgentDrawer → 输入「我一会有啥安排吗？」→ 等待助手消息出现 → 断言消息引用 ≥1 个 seeded 关键词。
- 复用现有 Playwright 配置（`frontend/playwright.config.ts`，baseURL、webServer 已就绪）。
- 性质：唯一证明「浏览器→后端→LLM→DB→WS→浏览器」全打通的层；flaky/slow/花钱，故手动触发。

## 5. 数据流（canonical 示例）

「我一会有啥安排吗？」完整路径：

```
用户输入(Promptfoo/Playwright/真实前端)
  → POST /api/agent/chat {conversation_id, text}        (handler, 返回 202)
  → goroutine: SendMessage → AppendMessage(user) → runTurn
  → ChatWithTools(真 LLM)                                ← 模型决策:list_schedule(date≈today)
  → PermRead 分支: broadcastTool(started) → tool.Execute(读 DB)
  → broadcastTool(succeeded, result) → AppendMessage(tool_result)
  → ChatWithTools(真 LLM) → resp.Content                 ← 模型生成自然语言答复
  → AppendMessage(assistant) + broadcast(agent_message) + broadcastDone
  └─ TraceRecorder.RecordTurn(TurnTrace{...})            ← L3 落 trace(json/langfuse)

观测侧:
  Promptfoo provider: 从 WS agent_tool/agent_message 重建 {tool_calls, assistant_text} → 断言
  Playwright:         断言 UI 出现助手消息 + 关键词
  Langfuse:           从 TraceRecorder 看完整 turn span
```

## 6. 错误处理

每一层在失败时都必须「让验证可观测地失败」，而非挂死或假绿。

| 失败情形 | 各层行为 |
|---|---|
| LLM 未配置 / key 无效 | `runTurn` 已广播 `finish_reason=error`（`service.go:88`）；provider 收到 `agent_done(error)` → 返回 `{error}`，断言失败；trace 记录空 Steps + error |
| 工具执行失败 | trace 记 `Status=failed`+Error；WS 已广播 `failed`；provider 的 `tool_calls` 含该失败项，断言按 case 判断 |
| WS 在 provider 侧断连 | provider 60s 超时 → 返回 `{error:"timeout"}` → 断言失败（不挂死） |
| PermWrite/Dangerous 超时未确认 | `service.go:165` 已 `rejected`；trace 记 `rejected`；对应 case（7–10）断言看到 `pending_confirmation`（注：eval 不驱动 `/confirm`，只验证「触发了确认」这个事实） |
| Langfuse 不可达 | `langfuseTracer` 对 POST 失败仅 warn-log，**绝不阻塞 runTurn**（trace 是旁路，不能拖垮生产路径） |
| seed 失败 | `seed.mjs` 非零退出，`promptfoo eval` 前置失败 |

## 7. 验证「验证器」本身（元层）

本系统是验证设施，需自证可信：

- **TraceRecorder 不破坏生产路径**：现有 `service_test.go` 全量绿（noop 默认）；新增一条 L1 测试，注入一个 recording tracer + mockLLM 触发一次工具调用，断言 `TurnTrace` 字段正确填充。
- **Promptfoo provider 正确**：写一个「假后端」冒烟——provider 对一个 mock 的 `/api/agent/*` + WS（返回固定事件序列）应正确重建 `{tool_calls, assistant_text}`。可用一个最小 node WS server 脚本验证，避免依赖真 LLM。
- **断言逻辑正确**：custom assert 函数单测（纯函数，输入 `{tool_calls}` 输出 bool）。
- **seed 幂等**：`seed.mjs` 重跑不重复创建（先清理或 upsert）。

## 8. 分阶段交付

| 阶段 | 内容 | 价值 |
|---|---|---|
| **P1** | `TraceRecorder`（noop/json）+ 接线 + L1 recording 测试；`eval/` 骨架 + provider + seed + 12 case + Promptfoo 跑通 canonical | 拿到「能重复验证『我一会有啥安排吗』」+ 零运维 trace |
| **P2** | Langfuse tracer + docker-compose；Playwright gated 纵向 spec | UI 可观测 + 浏览器纵向信心 |

P1 独立可交付、独立产生价值；P2 是增强。

## 9. 如何运行

| 场景 | 命令 | 前置 |
|---|---|---|
| 每次 PR（确定性） | `cd backend && go test ./...` | 无 |
| 改 prompt/工具（L2） | `make dev` → 设置页配 key → `node eval/seed.mjs` → `cd eval && npx promptfoo eval` | 活后端 + key |
| 肉眼看全链路（纵向） | `cd frontend && LLM_E2E=1 npx playwright test agent-llm` | 活后端 + key + seed |
| 看 trace UI（L3 Phase 2） | 设 `LANGFUSE_*` env，`docker compose -f docker-compose.langfuse.yml up` | Langfuse 实例 |

## 10. 风险与权衡

- **Promptfoo 需活后端 + seed**：相较「Go 进程内 eval 用内存 SQLite」多一套 test-fixture 基建。代价可接受，换取业界专门工具的打分/diff/报告 UX。
- **真 LLM 非确定性**：L2 是打分制（pass-rate + 阈值），不是二值闸门；偶发 miss 不挂 CI，但记录回归。
- **模型升级/换厂商**：近期 `evolve/multi-vendor-ai` 引入多厂商，不同厂商 tool-calling 遵从度不同——L2 可按 provider 分桶报分。
- **trace 改动 `service.go`**：surgical，受现有 L1 测试 + 新增 recording 测试双重守卫。

## 11. 显式不做（带理由）

- Inspect AI / DeepEval：语言与形状错配（真理由，非偷懒）。
- LangSmith：SaaS，选 Langfuse 自托管避免数据外送。
- 独立 record/replay 工具：是模式，已被 `jsonTracer` 文件 + Langfuse dataset 覆盖。
- 大批量自动生成数据集：12 条手写意图变体边际收益足够。
- 改动 Agent 业务运行时行为：`TraceRecorder` 是 nil 安全旁路。
