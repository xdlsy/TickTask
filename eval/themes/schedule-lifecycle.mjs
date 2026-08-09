import { called, tools } from '../lib/helpers.mjs';

// queryDb('schedules') returns whatever GET /api/schedules returns. Normalize
// to the events array whether the endpoint returns a bare array or wraps it
// (e.g. { events: [...] } / { items: [...] }).
const scheduleEvents = (ctx) => {
  const d = ctx && ctx.dbState;
  if (!d) return [];
  if (Array.isArray(d)) return d;
  if (Array.isArray(d.events)) return d.events;
  if (Array.isArray(d.items)) return d.items;
  return [];
};

// Real-behavior cases for the new schedule/timer/worklog/analytics tools.
// Drives the live backend; uses confirm + dbVerify where noted.
// NOTE: adapt the dbVerify `check(r, ctx)` access to match run-cases.mjs's real
// signature (read run-cases.mjs and post-action-verify.mjs first).
export const CASES = [
  // --- routing ---
  { cat: 'schedule-lifecycle', prompt: '把审查 PR-1234 那个日程标记完成',
    check: r => [called(r, 'update_schedule'), 'update_schedule'] },
  { cat: 'schedule-lifecycle', prompt: '明天下午3点加个会和产品对一下',
    check: r => [called(r, 'create_schedule'), 'create_schedule'] },
  { cat: 'schedule-lifecycle', prompt: '继续番茄钟',
    check: r => [called(r, 'control_pomodoro'), 'control_pomodoro'] },
  { cat: 'schedule-lifecycle', prompt: '今天我专注了多久了',
    check: r => [called(r, 'get_pomodoro_stats') || called(r, 'get_daily_insights'), 'focus stats'] },
  { cat: 'schedule-lifecycle', prompt: '看看我昨天的工作日志',
    check: r => [called(r, 'get_worklog') || called(r, 'list_worklogs'), 'worklog read'] },
  { cat: 'schedule-lifecycle', prompt: '帮我生成本周的周报',
    check: r => [called(r, 'generate_work_report'), 'generate_work_report'] },
  { cat: 'schedule-lifecycle', prompt: '最近一周专注趋势怎么样',
    check: r => [called(r, 'get_analytics') || called(r, 'get_daily_insights'), 'analytics'] },

  // --- ORIGINAL-BUG REGRESSION GUARD ---
  // Referring to a schedule + "已完成" must route to update_schedule and NOT
  // misuse update_task (the failure that motivated this whole change).
  { cat: 'schedule-lifecycle', prompt: '审查 PR-1234 这个我已经完成了',
    check: r => {
      const usedUpdateSchedule = called(r, 'update_schedule');
      const misusedUpdateTask = tools(r).some(t => t.name === 'update_task');
      const ok = usedUpdateSchedule && !misusedUpdateTask;
      return [ok, ok ? 'routed to update_schedule, not update_task' : 'regressed: misused update_task or missed update_schedule'];
    },
    note: 'original-bug regression guard' },

  // --- confirm + dbVerify (real post-action DB state) ---
  { cat: 'schedule-lifecycle', prompt: '把审查 PR-1234 标记为已完成',
    confirm: 'approve', dbVerify: 'schedules',
    check: (r, ctx) => {
      const scheds = scheduleEvents(ctx);
      const target = scheds.find(s => (s.title || '').includes('审查 PR-1234'));
      const ok = !!target && target.status === 'completed';
      return [ok, ok ? 'schedule status=completed in DB' : `status=${target && target.status}`];
    },
    note: 'dbVerify: update_schedule → completed' },
  { cat: 'schedule-lifecycle', prompt: '把审查 PR-1234 标记为已完成',
    confirm: 'reject', dbVerify: 'schedules',
    check: (r, ctx) => {
      const scheds = scheduleEvents(ctx);
      const target = scheds.find(s => (s.title || '').includes('审查 PR-1234'));
      const ok = !!target && target.status !== 'completed';
      return [ok, ok ? 'rejected: status unchanged' : 'regressed: reject still applied'];
    },
    note: 'dbVerify: reject leaves DB unchanged' },

  // --- revise two-step (multi-turn) ---
  { cat: 'schedule-lifecycle',
    turns: ['优化一下今天的安排', '就按这个改吧'],
    confirm: 'approve', dbVerify: 'schedules',
    check: (r, ctx) => {
      const ok = called(r, 'revise_schedule') && called(r, 'apply_schedule_revision');
      return [ok, ok ? 'revised + applied' : 'did not complete revise→apply'];
    },
    note: 'two-step revise→apply (turns)' },
];
