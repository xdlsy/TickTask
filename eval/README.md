# TickTask Agent 端到端意图验证（L2 / Promptfoo）

验证「自然语言 → 正确工具」的命中率。打分制，非 CI 闸门；改 prompt 或工具时手动跑。

## 前置

1. 后端运行且 AI 已配 key：`make dev`，在设置页配置并测试连接成功。
2. 安装依赖：`cd eval && npm install`

## 综合用例套（重点）：`run-cases.mjs` + `cases/`

**197 条真 LLM 用例，25 个类别**。`cases.mjs`（原 44 条 / 10 类）+ `themes/`（15 个主题文件，153 条）。主题覆盖：工具矩阵完整性、参数保真、实体消歧、确认全生命周期、多轮状态、幂等/副作用、输入边界/对抗参数、安全/内容策略、输出质量、i18n/时区、领域逻辑、动作后自验（防假成功）、确定性/flaky、性能 SLO、韧性/故障注入。共享断言 helper 在 `lib/helpers.mjs`。

```bash
AGENT_BASE_URL=http://localhost:8080 node seed.mjs      # 先 seed
AGENT_BASE_URL=http://localhost:8080 npm run cases       # 跑全量（~20 分钟，真 LLM）
```

**当前基线（minimax）：161/184 = 88%（增强 runner 可评估集）| SKIP 13/197 | 总 197。**

runner 现支持多模式（`run-cases.mjs`）：多轮 `turns[]`、确认流 `confirm:'approve'|'reject'`（自动 `/confirm` 续跑）、N 次重复 `runs`、计时 `maxMs`、DB 核验 `dbVerify`。runnable 因此从 124 → 184，SKIP 从 73 → 13（只剩故障注入 + llm-judge，需额外基建）。

- **13 条 SKIP**：故障注入（WS 断/429/畸形，需操控后端）+ llm-judge（主观，需第二个 LLM 打分）——后续基建项。
- **23 条残余 FAIL（真实信号，非测试/runner bug）**，集中在两类**模型行为问题**：
  1. **narrate 代替 tool 确认流**：部分写工具 prompt（开始/停番茄钟、保存日志、创建任务），模型用自然语言"确认?"而非调用工具触发 `pending_confirmation`（`tools=[-]`），导致确认流用例拿不到 succeeded。这是 agent 一致性问题（没可靠用上自己的确认机制）。
  2. **Bug#38 谎报成功 ~80%**：`标记不存在的任务完成` 跑 5 次，4/5 次 model 谎报"已完成"（determinism 类量化）。原 Bug#38 的间歇性，现在更明显。
  - 这两条是 **agent prompt/行为的改进靶点**（让模型可靠走工具确认流 + 不对不存在的操作谎报成功），不是测试缺陷。
- 其余残余：多轮复杂序列的模型行为方差（模型用"删了重建"代替"update"、长上下文边界等）+ 个别 flaky（AI 排程超时）。

> 这套用单源 + 自定义 runner（`collect()` 直连），比 promptfoo 的单轮 YAML 覆盖广得多（含"该拒绝/该说空/该追问/该报错/防假成功"这类软断言）。promptfooconfig 保留作 promptfoo-UI 子集。

## 跑一次（promptfoo 子集）

```bash
# 1. seed 今天的强特征数据（幂等）
AGENT_BASE_URL=http://localhost:8080 node seed.mjs

# 2. 跑 eval（每条 case 都真打一次 LLM）
AGENT_BASE_URL=http://localhost:8080 npx promptfoo eval

# 3. 看报告
npx promptfoo view
```

## provider 契约

`provider.mjs` 黑盒驱动 `/api/agent/*`：建会话 → 开 WS → POST /chat → 收集该会话的
`agent_message`/`agent_tool`/`agent_done` → 返回 `{ tool_calls, assistant_text }`。
断言据此判断工具路由是否正确（如 `list_schedule` 覆盖今天）。

## 不进默认测试

L2 是打分制（建议通过率 ≥0.9），偶发 miss 不挂 CI。默认 `go test` / `make test` 不触发它。

## 多轮 e2e（写工具后接着聊）

promptfooconfig 每条 case 是**单轮**，覆盖不到「用了写工具（建/改/删/番茄钟）确认后，下一轮是否还能正常答」——这条曾经是真实 bug（多轮工具调用断裂，每用一次写工具后续就卡死）。`multiturn-test.mjs` 守这条：建任务 → `/confirm` approve → 同一会话追问 → 断言答复真的提到该任务。

```bash
AGENT_BASE_URL=http://localhost:8080 node multiturn-test.mjs
```

## 相关

- spec：`docs/superpowers/specs/2026-08-09-agent-e2e-verification-design.md`
- 计划：`docs/superpowers/plans/2026-08-09-agent-e2e-verification-p1.md`