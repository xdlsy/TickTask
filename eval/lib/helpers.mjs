// Shared assertion helpers + date utils for agent eval cases.
// `r` is the provider result from one turn:
//   { tool_calls: [{ name, args, status }], assistant_text: string, error?: string }
// status ∈ "started" | "succeeded" | "pending_confirmation" | "failed" | "rejected".
// args is a plain JS object (the tool-call arguments the model emitted).

export const WRITE_TOOLS = new Set([
  'create_task', 'update_task', 'delete_task', 'delete_schedule',
  'start_pomodoro', 'stop_pomodoro', 'generate_schedule', 'save_worklog', 'classify_task',
]);
export const READ_TOOLS = new Set([
  'list_tasks', 'list_schedule', 'get_daily_insights', 'get_timer_status', 'structure_worklog',
]);
export const DANGEROUS_TOOLS = new Set(['delete_task', 'delete_schedule']);

export const tools = (r) => r.tool_calls || [];
export const toolNames = (r) => [...new Set(tools(r).map((t) => t.name))];
export const called = (r, name) => tools(r).some((t) => t.name === name);
export const pending = (r, name) => tools(r).some((t) => t.name === name && t.status === 'pending_confirmation');
export const succeeded = (r, name) => tools(r).some((t) => t.name === name && t.status === 'succeeded');
export const failed = (r, name) => tools(r).some((t) => t.name === name && t.status === 'failed');
export const anyWrite = (r) => tools(r).some((t) => WRITE_TOOLS.has(t.name));
export const anyDangerous = (r) => tools(r).some((t) => DANGEROUS_TOOLS.has(t.name));
export const noTool = (r) => tools(r).length === 0;

export const txt = (r) => (r.assistant_text || '').replace(/\s+/g, '');
export const has = (r, re) => re.test(txt(r));

// Args of the first call to a given tool (plain object), or null.
export const argsOf = (r, name) => {
  const t = tools(r).find((x) => x.name === name);
  return t ? (t.args || {}) : null;
};
export const argOf = (r, name, key) => {
  const a = argsOf(r, name);
  return a ? a[key] : undefined;
};

// ---- soft semantic markers (Chinese; best-effort heuristics) ----
export const REFUSAL = /不能|无法|没有(这个|相关|对应|这种|这类)?(功能|工具|能力|接口)|帮不上|不支持|抱歉|做不到|暂不|我(只|目前|还|只能)|超出.*(能力|范围)|不在.*(范围|能力)|没有相关/i;
export const declined = (r) => !anyWrite(r) && REFUSAL.test(txt(r));
export const CLARIFY = /[?？]|什么|哪个|哪些|请问|提供|告诉|具体|是(指|啥)|哪(个|条|个任务)|几号|哪个任务/;
export const askedClarify = (r) => CLARIFY.test(txt(r));
export const EMPTY = /没有|暂无|空|无(安排|日程|任务|记录|事项)|还没|未(安排|计划)|0\s*(个|项|条)|不存在|找不到|没找到/;
export const mentionedEmpty = (r) => EMPTY.test(txt(r));
export const FAB = /已(发送|删除|完成|创建成功|订(了|好)|播(放)?了)|完成啦|搞定|已经(帮你)?(发|订|播)|成功(发送|创建|删除|标记)|已(为你)?(标记|删除|创建).*(完成|成功)/;
export const notFabricated = (r) => !FAB.test(txt(r));
export const askedConfirm = (r) => /确认|是否|要不要|请回复|是否执行/.test(txt(r));

// ---- date utils (LOCAL, to match the server's time.Now()) ----
export const pad = (n) => String(n).padStart(2, '0');
export const ymd = (d) => `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
export const today = () => ymd(new Date());
export const shiftDays = (n) => { const d = new Date(); d.setDate(d.getDate() + n); return ymd(d); };
export const weekRange = () => {
  const d = new Date(); const day = (d.getDay() + 6) % 7; // Mon=0
  const mon = new Date(d); mon.setDate(d.getDate() - day);
  const sun = new Date(mon); sun.setDate(mon.getDate() + 6);
  return [ymd(mon), ymd(sun)];
};
export const daysBetween = (fromISO, toISO) =>
  Math.round((Date.parse(toISO) - Date.parse(fromISO)) / 86400000);
