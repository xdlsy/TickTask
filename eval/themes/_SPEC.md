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
