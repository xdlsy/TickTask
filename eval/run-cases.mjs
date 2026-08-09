// Enhanced runner for the 197-case suite. Supports execution modes beyond a
// single turn, selected by case fields:
//   turns: [p1, p2, ...]        → multi-turn (one conversation, sequential turns)
//   confirm: 'approve'|'reject' → after pending_confirmation, POST /confirm and
//                                 keep collecting until agent_done
//   runs: N                     → repeat the single turn N times (determinism)
//   maxMs: number               → timing (ctx.ms measured; check asserts)
//   dbVerify: 'tasks'|'schedules'|'sessions' → query API after, ctx.dbState
// check signature: check(r, ctx) where r = {tool_calls, assistant_text, error}
// and ctx = {history?, runs?, dbState?, ms}. Old check(r) still works.
//
// Still SKIP: fault-injection (needs infra to drop WS / 429 / malformed) and
// llm-judge (subjective) — by note, plus the `needs ` placeholder fallback.
import { WebSocket } from 'ws';
import { ALL_CASES, CATEGORIES } from './all-cases.mjs';

const BASE = process.env.AGENT_BASE_URL || 'http://localhost:8080';
const WS = process.env.AGENT_WS_URL || 'ws://localhost:8080/ws';

// Drive one turn within an existing conversation. If `confirm` is set, on the
// first pending_confirmation it POSTs /confirm and keeps collecting; otherwise
// it early-stops on pending_confirmation (single-turn mode).
function runInConv(convId, prompt, { confirm, timeoutMs = 60_000 } = {}) {
  return new Promise((resolve) => {
    const tcs = []; let text = ''; let settled = false; let confirmed = false;
    const finish = (x) => { if (settled) return; settled = true; clearTimeout(t); try { ws.close(); } catch {} resolve({ tool_calls: tcs, assistant_text: text, ...(x || {}) }); };
    const ws = new WebSocket(WS);
    const t = setTimeout(() => finish({ error: 'timeout' }), timeoutMs);
    ws.on('open', () => fetch(`${BASE}/api/agent/chat`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ conversation_id: convId, text: prompt }) }).catch((e) => finish({ error: `chat:${e.message}` })));
    ws.on('message', (data) => {
      let m; try { m = JSON.parse(data.toString()); } catch { return; }
      if (m.conversation_id !== convId) return;
      if (m.type === 'agent_message') text += m.delta_text || '';
      else if (m.type === 'agent_tool') {
        tcs.push({ name: m.tool_name, args: m.args, status: m.status });
        if (m.status === 'pending_confirmation' && !confirmed) {
          confirmed = true;
          if (confirm) {
            fetch(`${BASE}/api/agent/confirm`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ message_id: m.message_id, decision: confirm }) }).catch(() => {});
          } else { finish({}); } // single-turn: early-stop on pending
        }
      } else if (m.type === 'agent_done') finish(m.error ? { error: m.error } : {});
    });
    ws.on('error', (e) => finish({ error: `ws:${e.message}` }));
  });
}

async function newConv() {
  return (await fetch(`${BASE}/api/agent/conversations`, { method: 'POST' }).then((r) => r.json())).id;
}

async function queryDb(type) {
  try {
    if (type === 'tasks') return await fetch(`${BASE}/api/tasks`).then((r) => r.json());
    if (type === 'schedules') return await fetch(`${BASE}/api/schedules`).then((r) => r.json());
    if (type === 'sessions') return await fetch(`${BASE}/api/sessions`).then((r) => r.json());
  } catch (e) { return { error: e.message }; }
  return null;
}

// Execute one case per its mode → returns { r, ctx }.
async function executeCase(c) {
  const ctx = {};
  const t0 = Date.now();

  // multi-turn: one conversation, sequential turns
  if (Array.isArray(c.turns)) {
    const convId = await newConv();
    const history = [];
    for (const tp of c.turns) history.push(await runInConv(convId, tp, { confirm: c.confirm }));
    ctx.ms = Date.now() - t0;
    ctx.history = history;
    // combined r across all turns
    const tool_calls = history.flatMap((h) => h.tool_calls || []);
    const assistant_text = history.map((h) => h.assistant_text || '').join('\n');
    const err = history.find((h) => h.error);
    return { r: { tool_calls, assistant_text, error: err ? err.error : undefined }, ctx };
  }

  // N-run (determinism): N independent conversations
  if (c.runs && c.runs > 1) {
    const runs = [];
    for (let k = 0; k < c.runs; k++) {
      const convId = await newConv();
      runs.push(await runInConv(convId, c.prompt, { confirm: c.confirm }));
    }
    ctx.ms = Date.now() - t0;
    ctx.runs = runs;
    return { r: runs[runs.length - 1], ctx };
  }

  // single-turn (optionally confirm-flow)
  const convId = await newConv();
  const r = await runInConv(convId, c.prompt, { confirm: c.confirm });
  ctx.ms = Date.now() - t0;
  if (c.dbVerify) ctx.dbState = await queryDb(c.dbVerify);
  return { r, ctx };
}

const cases = ALL_CASES;
const byCat = Object.fromEntries(CATEGORIES.map((c) => [c, { pass: 0, fail: 0, skip: 0, fails: [] }]));
let totalPass = 0, totalSkip = 0;

for (let i = 0; i < cases.length; i++) {
  const c = cases[i];
  const cat = c.cat;
  const label = c.prompt ? c.prompt.slice(0, 26) : (c.turns ? c.turns.join(' / ').slice(0, 26) : '?');

  // still SKIP: fault-injection (needs infra) + llm-judge (subjective)
  if (c.note && /needs[- ]?fault|inject |restart backend|llm-judge/i.test(c.note)) {
    totalSkip++; byCat[cat].skip++;
    console.log(`${String(i + 1).padStart(3)}/${cases.length} [SKIP] (${cat.padEnd(13)}) ${label.padEnd(28)} :: ${c.note}`);
    continue;
  }

  let r, ctx, execErr;
  try { ({ r, ctx } = await executeCase(c)); }
  catch (e) { execErr = e.message; }

  let verdict = 'FAIL', reason = '';
  if (execErr) { reason = `exec error: ${execErr}`; }
  else if (r && r.error) { reason = `ERROR: ${r.error}`; }
  else {
    try {
      const out = c.check(r, ctx);
      const pass = Array.isArray(out) ? out[0] : out;
      const why = Array.isArray(out) ? out[1] : '';
      reason = String(why ?? '');
      if (pass && /^needs /i.test(reason)) verdict = 'SKIP';
      else if (pass) verdict = 'PASS';
      else verdict = 'FAIL';
    } catch (e) { reason = `check threw: ${e.message}`; verdict = 'FAIL'; }
  }

  const tn = [...new Set((r?.tool_calls || []).map((t) => t.name))].join(',') || '-';
  if (verdict === 'PASS') { totalPass++; byCat[cat].pass++; }
  else if (verdict === 'SKIP') { totalSkip++; byCat[cat].skip++; }
  else { byCat[cat].fail++; byCat[cat].fails.push(c.prompt || c.turns?.[0]); }
  console.log(`${String(i + 1).padStart(3)}/${cases.length} [${verdict}] (${cat.padEnd(13)}) ${label.padEnd(28)} tools=[${tn}]${reason ? ' :: ' + reason.slice(0, 60) : ''}`);
}

console.log('\n================= by category =================');
for (const c of CATEGORIES) {
  const s = byCat[c]; const n = s.pass + s.fail;
  if (n + s.skip === 0) continue;
  const rate = n ? ((s.pass / n) * 100).toFixed(0).padStart(3) : '  -';
  console.log(`  ${c.padEnd(16)} ${rate}%  (pass ${s.pass}/${n})${s.skip ? ` skip ${s.skip}` : ''}${s.fail ? '  ✗ ' + s.fails.slice(0, 4).map((f) => `"${(f || '').slice(0, 16)}"`).join(', ') : ''}`);
}
const ran = cases.length - totalSkip;
console.log(`\n=== PASS ${totalPass}/${ran} (${ran ? ((totalPass / ran) * 100).toFixed(0) : 0}% of runnable) | SKIP ${totalSkip}/${cases.length} | TOTAL ${cases.length} ===`);
