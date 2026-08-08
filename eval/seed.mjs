// Seeds today's schedules + tasks with distinctive titles so L2 assertions can
// reference stable keywords. Idempotent: clears prior seed data first.
// Run against a live backend:  AGENT_BASE_URL=http://localhost:8080 node seed.mjs
const BASE = process.env.AGENT_BASE_URL || 'http://localhost:8080';
const today = new Date().toISOString().slice(0, 10); // YYYY-MM-DD
const rfc = (hhmm) => `${today}T${hhmm}:00Z`; // time.RFC3339, UTC

const SCHEDULE_TITLES = ['审查 PR-1234', '和 Alice 1:1', '发布 v2.3'];
const TASK_TITLES = ['整理周报', '修复登录 bug'];

const SCHEDULES = [
  { title: '审查 PR-1234', start_time: rfc('09:00'), end_time: rfc('10:00'), type: 'task' },
  { title: '和 Alice 1:1', start_time: rfc('14:00'), end_time: rfc('14:30'), type: 'task' },
  { title: '发布 v2.3', start_time: rfc('17:00'), end_time: rfc('17:30'), type: 'task' },
];
const TASKS = [
  { title: '整理周报', quadrant: 2 },
  { title: '修复登录 bug', quadrant: 1 },
];

async function json(method, path, body) {
  const res = await fetch(`${BASE}${path}`, {
    method,
    headers: body ? { 'Content-Type': 'application/json' } : undefined,
    body: body ? JSON.stringify(body) : undefined,
  });
  if (!res.ok) throw new Error(`${method} ${path} -> ${res.status}: ${await res.text().catch(() => '')}`);
  return res.status === 204 ? null : res.json().catch(() => null);
}

async function clearSeed() {
  // Schedules: backend exposes DELETE /api/schedules (DeleteAll) — acceptable
  // for an eval DB we fully control.
  await json('DELETE', '/api/schedules').catch(() => {});
  // Tasks: delete only our seed titles to stay idempotent without nuking all.
  const tasks = await json('GET', '/api/tasks');
  for (const t of tasks || []) {
    if (TASK_TITLES.includes(t.title)) await json('DELETE', `/api/tasks/${t.id}`).catch(() => {});
  }
}

async function main() {
  await clearSeed();
  for (const s of SCHEDULES) {
    const ev = await json('POST', '/api/schedules', s);
    console.log('schedule:', ev && ev.title);
  }
  for (const tk of TASKS) {
    const task = await json('POST', '/api/tasks', tk);
    console.log('task:', task && task.title);
  }
  console.log(`seeded ${SCHEDULES.length} schedules + ${TASKS.length} tasks for ${today}`);
}

main().catch((e) => { console.error(e); process.exit(1); });
