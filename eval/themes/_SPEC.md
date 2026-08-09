# Agent eval 用例编写 SPEC（子 agent 必读）

你负责写**一个主题**的用例文件 `eval/themes/<theme>.mjs`。先读本文件，再按分派给你的主题用例清单落代码。

## 文件格式

```js
import {
  called, pending, succeeded, failed, anyWrite, anyDangerous, noTool,
  tools, toolNames, argsOf, argOf, txt, has,
  declined, askedClarify, mentionedEmpty, notFabricated, askedConfirm,
  today, shiftDays, weekRange, daysBetween,
} from '../lib/helpers.mjs';

// 每条用例：{ cat, prompt, check, note? }
// check(r) 返回 [pass: boolean, reason: string]。r 见下。note 标执行方式（可选）。
export const CASES = [
  { cat: '<theme>', prompt: '...', check: (r) => [called(r, 'list_schedule'), 'list_schedule'] },
  // ...
];
```

## `r` 的结构（一轮对话的重建结果）

```js
{
  tool_calls: [{ name: 'list_schedule', args: { from: '2026-08-09', to: '2026-08-09' }, status: 'succeeded' }],
  assistant_text: '你今天有 3 个安排…',
  error: undefined,            // 有则代表本轮出错/超时
}
// status ∈ 'started' | 'succeeded' | 'pending_confirmation' | 'failed' | 'rejected'
// args 是普通 JS 对象（模型发出的工具参数）
```

## helper API（全部从 '../lib/helpers.mjs' 导入）

判定工具调用：
- `called(r, name)` — 是否调过某工具
- `pending(r, name)` — 该工具是否处于 pending_confirmation（写/危险工具触发确认）
- `succeeded(r, name)` / `failed(r, name)` — 该工具执行成功/失败
- `anyWrite(r)` / `anyDangerous(r)` / `noTool(r)` — 调过任意写工具 / 危险工具 / 没调任何工具
- `tools(r)` / `toolNames(r)` — 工具调用数组 / 去重工具名
- `argsOf(r, name)` — 该工具的 args 对象（无则 null）
- `argOf(r, name, key)` — 该工具某参数值

文本软断言（中文启发式，best-effort）：
- `txt(r)` — 去空白后的答复文本；`has(r, /re/)` — 是否匹配正则
- `declined(r)` — 诚实拒绝（无写工具 + 文本含拒绝词）
- `askedClarify(r)` — 追问澄清（含 ?/什么/哪个…）
- `mentionedEmpty(r)` — 表达"没有/空/不存在"
- `notFabricated(r)` — 没谎报已做（不含"已发送/已删除/搞定…"）
- `askedConfirm(r)` — 用自然语言要确认

日期（本地，与服务端 time.Now() 对齐）：
- `today()` / `shiftDays(n)` / `weekRange()` → 'YYYY-MM-DD'（weekRange 返回 [mon, sun]）
- `daysBetween(fromISO, toISO)` — 相差天数

## Agent 上下文（写 check 时参考）

14 个工具 + 权限：
- 读（自动执行，不确认）：`list_tasks` `list_schedule` `get_daily_insights` `get_timer_status` `structure_worklog`
- 写（需确认 pending）：`create_task` `update_task` `start_pomodoro` `stop_pomodoro` `generate_schedule` `save_worklog` `classify_task`
- 危险（需确认 pending）：`delete_task` `delete_schedule`

system prompt 约束：简洁友好；**修改类操作要确认**；**只能用已有工具，做不到的诚实说明、绝不假装**。

已知行为/坑：
- 读工具自动跑；写/危险工具 → pending_confirmation；30 分钟不确认 → 自动 rejected。
- 多轮工具调用已修好（assistant tool_calls 持久化 + 链接）。
- **日期坑**：seed 用 UTC，服务端用本地时间——"今天"要确认二者对齐。

## 输出规则

1. 只写你被分派的那一个 `eval/themes/<theme>.mjs`，别动别的文件。
2. 从 `'../lib/helpers.mjs'` 按需 import，**不要**重复定义 helper。
3. 每条用例 `cat` 填你的主题名；`check` 用 helper 写，返回 `[bool, reason]`。
4. 需要多轮/确认流/计时/故障注入/LLM-judge/DB核验的用例，加 `note` 字段说明（如 `note: 'multi-turn'`、`note: 'needs-confirm'`、`note: 'llm-judge'`），`check` 仍写"单轮能验的部分"或留 `()=>[true,'needs <x> runner']` 占位。
5. 写完用 `node --check eval/themes/<theme>.mjs` 验证语法通过。**不要跑后端、不要跑用例。**
6. 报告：文件路径、用例数、`node --check` 结果。

---

# 增强版 runner（run-cases.mjs）——重写占位 check 时用

runner 现在支持多种执行模式，由 case 字段选择。**check 签名升级为 `check(r, ctx)`**，旧 `check(r)` 仍兼容（ctx 被忽略）。

## 模式字段（加到 case 上）

- `turns: ['p1', 'p2', ...]` → 多轮：同一会话顺序跑每轮。`ctx.history = [r1, r2, ...]`（每轮一个 `{tool_calls, assistant_text, error}`）；传入 check 的 `r` 是**合并**后的（所有轮 tool_calls 拼接、assistant_text 换行拼接）。
- `confirm: 'approve' | 'reject'` → 确认流：单轮，遇到 `pending_confirmation` 自动 POST `/confirm`（按 approve/reject），**继续收集**到 agent_done。`r.tool_calls` 会含工具确认后的状态（succeeded/rejected）。
- `runs: N` → N 次重复：N 个独立会话各跑一次同一 prompt。`ctx.runs = [r1, ..., rN]`；`r` = 最后一次。
- `maxMs: 数字` → 计时：`ctx.ms` = 该 case 端到端毫秒；check 断言 `ctx.ms < maxMs`。
- `dbVerify: 'tasks' | 'schedules' | 'sessions'` → 跑完查 API：`ctx.dbState` = GET `/api/<type>` 的结果。

## ctx 形态

```js
ctx = {
  ms: number,                 // 总有
  history?: [...],            // 仅多轮
  runs?: [...],               // 仅 N 次
  dbState?: {...},            // 仅 dbVerify
}
```

## 重写规则（把占位改成真断言）

- **多轮**：用 `ctx.history`。如代词指代「建任务X」→「把它的截止日期改周五」：第二轮应 `pending(update_task)`，断言 `(ctx.history[1])` 里 update_task pending。
- **确认流 approve**：`confirm:'approve'`，断言 `succeeded(r, 'create_task')`（确认后工具真执行）+ 可选 `ctx.dbState`（dbVerify:'tasks'）里有该任务。
- **确认流 reject**：`confirm:'reject'`，断言工具**未** succeeded（被拒）+ dbState 不变。
- **N 次（确定性）**：`runs:5`，断言 `ctx.runs` 每次路由一致（如每次都 `called(ri,'list_schedule')`）。
- **计时**：`maxMs:10000`，断言 `ctx.ms < 10000`（注意 LLM 延迟，给余量；可标 `note:'flaky-timing'`）。

**仍 SKIP（runner 不支持，保留占位 `()=>[true,'needs ...']`）**：故障注入（needs-fault/inject/restart backend）、llm-judge。

