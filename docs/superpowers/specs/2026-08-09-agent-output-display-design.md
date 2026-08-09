# Agent 输出显示优化 · 设计文档

- 日期:2026-08-09
- 状态:已通过 brainstorming,待实现
- 范围:前端 `frontend/src/components/agent/` + `frontend/src/stores/agent.ts`
- 不在范围:后端(`backend/internal/agent/**`)、语法高亮依赖

## 1. 背景与问题

Agent 抽屉(`AgentDrawer`)当前的消息显示存在四类问题(用户确认均为优化重点):

1. **Agent 文本不渲染 Markdown** —— `AgentMessageList.vue` 的 `.bubble` 直接输出 `{{ m.content }}` 纯文本,列表 / 加粗 / 代码块 / 表格都显示成原始符号(`**像这样**`)。
2. **工具卡片原始 JSON 堆叠** —— `ToolCard.vue` 把 `tool_args` 放进 `<pre>` 原样展示,`tool_result` 是扁平文本,长结果(如整张日程)糊成一片。
3. **消息扁平、无分组** —— 每个 `tool_call` / `tool_result` 都自带一行 `Agent` 眉标;缺少"一轮对话"的视觉聚合。
4. **流式 / 思考中体验弱** —— 只有 3 个闪烁点 + 逐字累加;工具执行时看不到进度。

### 范围发现(关键)

"实时工具时间线"无法仅靠 CSS 实现。`frontend/src/stores/agent.ts` 有两处占位会丢数据:

- **`onAgentTool`(77-85 行)什么都没做**:注释写着"Update or append the tool_call message locally",但没有实现。流式过程中的工具事件全部被丢弃,只有刷新页面(从后端重载)后才出现。
- **`onAgentDone`(88-94 行)把整轮压成一条 assistant 消息**:后端实际是"每个 LLM 回合一跳"就持久化一条 assistant 消息(`backend/internal/agent/service.go:133`),并和工具调用交错。当前实时视图把这层文本↔工具的交错全丢了。

后端已把数据存对(细粒度 assistant + 工具消息,靠 `parent_id` 串起来),所以 **MVP 不需要改后端** —— 但 store 的 WS 处理和消息列表渲染都得真改。

### 协议事实(来自 `backend/internal/agent/service.go:108-200`)

一轮对话内,WS 事件顺序为:
```
agent_message(文本段, 带 message_id)
  → [agent_tool started → agent_tool succeeded/failed]*   # PermRead,事件无 message_id
  或 [agent_tool pending_confirmation → (用户确认) → succeeded/rejected]  # PermWrite/Dangerous,带 message_id
  → (下一轮)agent_message ...
  → agent_done(stop)
```
- `agent_message.delta_text` 是该轮 LLM 回合的完整内容块(非逐 token)。
- 读工具事件(`PermRead`)无 `message_id`;写工具事件(`PermWrite/Dangerous`)有稳定 `message_id`(= 持久化的 `tool_call` 消息 id)。
- 同一回合内 `for _, tc := range resp.ToolCalls` 是**串行**执行。

## 2. 锁定的设计决策(brainstorming 结论)

- **骨架**:方向 A —— 按回合(turn)分组的时间线 + 字母章头像。
- **工具行**:智能摘要 + 可展开完整 JSON。
- **Markdown**:编辑式风格(Fraunces 标题、代码块带语言标签 + 复制、琥珀色引用竖线、表头 mono 大写)。
- **流式**:工具执行时时间线实时点亮;空闲用单点琥珀脉冲指示器。
- **头像**:琥珀色 Fraunces 字母章 "A"。
- **技术方案**:`marked` + `DOMPurify`(方案 A)。

## 3. 数据模型与 store 修复

### 3.1 单一事实源 + 视图派生

store 的 `messages`(扁平数组)保持不变,仍是唯一事实源(与后端一致)。视图不直接渲染扁平 messages,而是用 composable `useAgentTurns(messages, streamingText, isThinking)` 派生成 `turns`(响应式 `computed`,不引入第二份状态)。

### 3.2 Turn 结构(视图内部模型,放 `useAgentTurns.ts`,不入全局 barrel)

```ts
type Segment =
  | { kind: 'text'; message: AgentMessage }   // assistant 文本段
  | { kind: 'tool'; message: AgentMessage }   // tool_call / tool_result

interface Turn {
  id: string                 // = 该轮 user 消息 id;实时轮用 'live'
  user?: AgentMessage        // 用户的提问
  segments: Segment[]        // 按到达顺序排列
  live?: { text: string }    // 当前正在流式、尚未 flush 的文本(带光标)
}
```

分组规则:一条 user 消息 + 其后到下一条 user 消息之前的所有 assistant / tool 消息 = 一个 turn。实时进行中的那轮(还有 `streamingText` 或 `isThinking`)附加到最后一个 turn,或作为新 turn。

### 3.3 Store 修复 1 · `onAgentTool` 实例化工具消息

入口处先 `flushStreaming()`(见 §3.4),保证本工具之前的文本段落袋、顺序正确,再做下面的实例化:

```ts
onAgentTool(e) {
  if (this.currentConvId !== null && e.conversation_id !== this.currentConvId) return
  this.flushStreaming()   // 先把进行中的 assistant 文本段落袋
  if (e.status === 'pending_confirmation') {
    this.pendingConfirm = { messageId: e.message_id!, toolName: e.tool_name, args: e.args, preview: e.preview }
  }
  if (e.message_id) {
    // 写工具:按 id upsert
    const idx = this.messages.findIndex(m => m.id === e.message_id)
    const msg: AgentMessage = {
      id: e.message_id, conversation_id: this.currentConvId!, role: 'tool_call',
      content: '', tool_name: e.tool_name, tool_args: safeStringify(e.args),
      tool_result: e.result != null ? safeStringify(e.result) : (e.error ? `{"error":${JSON.stringify(e.error)}}` : undefined),
      tool_status: e.status, created_at: new Date().toISOString(),
    }
    if (idx >= 0) this.messages[idx] = { ...this.messages[idx], ...msg }
    else this.messages.push(msg)
  } else {
    // 读工具:无 message_id,按序匹配
    if (e.status === 'started') {
      this.messages.push({ /* role:'tool_call', tool_status:'started', id: 'tool-'+<seq> */ })
    } else {
      const idx = findLastIndex(this.messages,
        m => m.tool_name === e.tool_name && m.tool_status === 'started')
      if (idx >= 0) this.messages[idx] = { ...this.messages[idx], tool_status: e.status, tool_result: ... }
    }
  }
}
```

`findLastIndex` + "最近一条同名 started" 匹配依赖后端串行执行(见 §1 协议事实),当前安全。

### 3.4 Store 修复 2 · 按 `message_id` 分段 flush

```ts
onAgentMessage(e) {
  if (this.currentConvId !== null && e.conversation_id !== this.currentConvId) return
  if (this.streamingMessageId !== null && this.streamingMessageId !== e.message_id && this.streamingText) {
    this.flushStreaming()   // message_id 变了 → 先把上一段落袋
  }
  this.streamingMessageId = e.message_id
  this.streamingText += e.delta_text
}

// 在 onAgentTool 开头、onAgentDone 里调用:
flushStreaming() {
  if (this.streamingText) {
    this.messages.push({
      id: this.streamingMessageId || 'ast-' + <seq>,
      conversation_id: this.currentConvId!, role: 'assistant',
      content: this.streamingText, created_at: new Date().toISOString(),
    })
  }
  this.streamingText = ''
  this.streamingMessageId = null
}
```

`onAgentDone` 改为只调用 `flushStreaming()` + `isThinking=false`,不再"整轮压一条"。`onAgentTool` 在处理前先 `flushStreaming()`,保证文本段排在工具之前。

> 注意:`new Date()`/`Date.now()` 在主运行时可用(本约束只针对 workflow 脚本)。`<seq>` 用模块级自增计数或现有的 `Date.now()` 模式,与 `sendMessage` 一致。

### 3.5 删掉"合成流式气泡" hack

`AgentMessageList.vue` 现有的 `v-if="streamingText || isThinking"` 临时 bubble(11-18 行)删除,改成由真实分段 + turn 内 `live` 段驱动。自动滚动逻辑(未提交的那份,`root` ref + `scrollParent` + `isNearBottom` + `stickToBottom`)保留。

## 4. 组件分解 + Markdown 渲染

### 4.1 组件树

```
AgentDrawer
└─ AgentMessageList            (改造:算 turns + 渲染 AgentTurn;保留自动滚动)
   └─ AgentTurn :turn          (新:头像 + 分段聚合 + live 段)
      ├─ MarkdownText :content (新:marked + DOMPurify)
      └─ ToolRow :message      (新:替代 ToolCard)
```

### 4.2 `MarkdownText.vue`(新)

- props:`{ content: string }`
- `computed html = DOMPurify.sanitize(marked.parse(content, { gfm:true, breaks:true, mangle:false, headerIds:false }), { USE_PROFILES:{html:true} })`
  - `breaks:true`:单 `\n` → `<br>`,贴合聊天。
  - DOMPurify 杀毒(LLM 输出是 XSS 面,必做)。
- `v-html` 渲染,scoped 编辑式样式:h2/h3 Fraunces、行内 code 琥珀 pill、`pre` 升为 elevated 面板 + 语言标签 + 复制按钮、表头 mono 大写、引用琥珀竖线斜体、链接琥珀下划线。
- 复制按钮:parse 后 `nextTick` 里 `querySelectorAll('pre')`,按 `code` 的 class 读语言、挂复制按钮(复制 `textContent`)。
- 解析结果按 `content` memo,避免重复 parse。

### 4.3 `ToolRow.vue`(新,替代 ToolCard)

保留 ToolCard 语义(perm 分类 read/write/danger/failed、失败 crimson、状态图标 + chip),新增:

- 智能摘要行:工具名 + 状态点 + 一句人话摘要(argHint / resultHint,来自 `toolFormatters`)+ 内联关键参数;`⌄` 展开。
- 可展开 JSON:美化后的 args + result,`max-height` + 滚动。
- 原位确认(并入 `ToolConfirmDialog`):pending_confirmation 时直接在工具行显示**预览 + 批准 / 拒绝**,调用同一个 `store.confirmToolCall(message.id, 'approve'|'reject')`。

### 4.4 `toolFormatters.ts`(新,与 ToolRow 同目录)

`summary(toolName, args, result): { argHint?: string; resultHint?: string }` 的工具名 → 格式化器映射。覆盖真实工具注册表中的工具(如 `get_tasks` → "读取 N 条";`create_schedule` → "生成 N 时段";`create_task` → 取 title)。未命中回退:第一个标量参数 + 状态。

### 4.5 `AgentTurn.vue`(新)

- 渲染头像(琥珀 Fraunces 字母章 "A")+ 该轮 segments。
- text 段 → `<MarkdownText>`;tool 段 → `<ToolRow>`。
- `turn.live`:有文本 → `<MarkdownText>` + 闪烁光标;空 → 单点琥珀脉冲。
- user 消息:右对齐琥珀 bubble(沿用现有 `.msg.user` 样式)。

### 4.6 `AgentMessageList.vue`(改造)

- 接口不变(`messages / streamingText / isThinking`)。
- 用 `useAgentTurns` 派生 turns,`v-for` 渲染 `<AgentTurn>`。
- 删掉合成流式 bubble;保留自动滚动逻辑。

## 5. 文件增删

| 动作 | 文件 |
|---|---|
| 新增 | `MarkdownText.vue` / `ToolRow.vue` / `AgentTurn.vue` / `useAgentTurns.ts` / `toolFormatters.ts` |
| 改造 | `AgentMessageList.vue`、`stores/agent.ts`(两处修复)、`AgentDrawer.vue`(去掉 `ToolConfirmDialog`) |
| 删除 | `ToolCard.vue`、`ToolConfirmDialog.vue`(功能并入 ToolRow) |
| 测试 | `ToolCard.spec.ts` → 迁为 `ToolRow.spec.ts`;新增 `MarkdownText.spec.ts`、`useAgentTurns.spec.ts`、`stores/agent.spec.ts`(补充) |
| 依赖 | `package.json` + `marked`、`dompurify`(后者自带类型) |
| 类型 | `Segment` / `Turn` 放在 `useAgentTurns.ts` 本地,不入全局 barrel |

## 6. 流式状态机

| 阶段 | 触发 | 渲染 |
|---|---|---|
| 空闲思考 | `isThinking && !streamingText` | 单点琥珀脉冲 |
| 流式输出 | `streamingText` 增长 | MarkdownText + 闪烁光标 |
| 工具执行中 | `agent_tool` `started` | 该工具行:spinner + 发光点 + "执行中…" |
| 待确认 | `pending_confirmation` | 工具行原位:预览 + 批准 / 拒绝 |
| 完成 | `succeeded` | 状态点固化 + 摘要 |
| 失败 / 取消 | `failed` / `rejected` | crimson / 灰态(保留现有语义) |

## 7. 边界与错误处理

- 空 content(只有 tool_calls 的 assistant 消息)→ 不渲染空 bubble。
- 超长 JSON / 结果 → `max-height` + 滚动;超长 Markdown 同理,代码块已裁剪。
- Markdown XSS → DOMPurify 兜底;杀毒后空字符串不崩。
- 失败工具 → crimson 结果文案(保留 ToolCard 行为)。
- 对话中途切换 → `onAgent*` 的 `conversation_id` 守卫保留。
- 用户上滑阅读 → `isNearBottom` 尊重,不强行拉底。
- 读工具 `started` 后断连未回 `succeeded` → 维持"执行中"态;WS 重连由现有机制处理,不额外造超时。

## 8. 测试矩阵(Vitest + jsdom)

- `MarkdownText.spec.ts`:GFM 元素渲染;**XSS 输入(`<img onerror>`)被剥**;代码块语言标签 + 复制按钮;同 content 不重复 parse。
- `useAgentTurns.spec.ts`:user/assistant/tool 交错分组;live turn(有 / 无 streamingText);空消息跳过;对话切换数组替换。
- `ToolRow.spec.ts`(迁自 ToolCard):read/write/danger/failed 四态 class;智能摘要命中 + 未知工具回退;`⌄` 展开 JSON;pending 原位确认点击触发 `store.confirmToolCall(id,'approve'/'reject')`;预览显示。
- `stores/agent.spec.ts`(补充):`onAgentTool` 读 started→succeeded 匹配、写 id upsert;`onAgentMessage` 跨 message_id 分段 flush;`onAgentDone` flush 残余。
- `AgentDrawer.spec.ts`:保留 `.messages` / `agent-input` / `not-configured-hint` 断言(若类名调整同步改)。
- testid 统一:删 ToolConfirmDialog 后,确认按钮 testid 统一到 ToolRow(`tool-confirm-approve` / `tool-confirm-reject`),迁移旧断言。

## 9. 手动验证(E2E)

```
make dev
# 另开终端:
lsof -ti:8080 | xargs kill -9
cd backend && go run cmd/server/main.go
# 刷新前端页
```

在 Agent 抽屉里发"安排今天的日程",观察:实时时间线逐个点亮、Markdown 编辑式渲染、写工具原位确认流、结果智能摘要。

## 10. 后续可选(不在本次范围)

- 后端给 `PermRead` 的 `agent_tool` 事件补稳定 id,替代前端的"按序匹配"(更鲁棒,支持潜在并行)。
- 引入语法高亮依赖(`highlight.js` / `shiki`),若实际 agent 回复中代码块变多。
- 周期报告等超长回复的折叠 / 摘要策略。
