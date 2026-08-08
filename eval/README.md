# TickTask Agent 端到端意图验证（L2 / Promptfoo）

验证「自然语言 → 正确工具」的命中率。打分制，非 CI 闸门；改 prompt 或工具时手动跑。

## 前置

1. 后端运行且 AI 已配 key：`make dev`，在设置页配置并测试连接成功。
2. 安装依赖：`cd eval && npm install`

## 跑一次

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