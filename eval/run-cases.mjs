// Runs the comprehensive case suite (all-cases.mjs) against the live backend via
// collect(). Resolves a turn on agent_done OR pending_confirmation (so write/
// dangerous cases don't wait the 30-min ConfirmationTimeout).
//
// Case handling:
//  - cases with `turns` (array) are multi-turn → SKIP (needs a multi-turn runner)
//  - a check returning [true, "<reason>"] where reason starts with "needs " → SKIP
//    (placeholder for confirm/timing/fault-injection/llm-judge/DB runners)
//  - otherwise [true,…] → PASS, [false,…] → FAIL
import { WebSocket } from 'ws';
import { ALL_CASES, CATEGORIES } from './all-cases.mjs';

const BASE = process.env.AGENT_BASE_URL || 'http://localhost:8080';
const WS = process.env.AGENT_WS_URL || 'ws://localhost:8080/ws';

function runTurn(prompt, timeoutMs = 60_000) {
  return (async () => {
    const conv = await fetch(`${BASE}/api/agent/conversations`, { method: 'POST' }).then((r) => r.json());
    const id = conv.id;
    return new Promise((resolve) => {
      const tcs = []; let text = ''; let settled = false;
      const finish = (x) => { if (settled) return; settled = true; clearTimeout(t); try{ws.close()}catch{}; resolve({tool_calls: tcs, assistant_text: text, ...(x||{})}); };
      const ws = new WebSocket(WS);
      const t = setTimeout(() => finish({error:'timeout'}), timeoutMs);
      ws.on('open', () => fetch(`${BASE}/api/agent/chat`, {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({conversation_id: id, text: prompt})}).catch(e=>finish({error:`chat:${e.message}`})));
      ws.on('message', (data) => {
        let m; try { m = JSON.parse(data.toString()); } catch { return; }
        if (m.conversation_id !== id) return;
        if (m.type === 'agent_message') text += m.delta_text || '';
        else if (m.type === 'agent_tool') { tcs.push({name:m.tool_name, args:m.args, status:m.status}); if (m.status === 'pending_confirmation') finish({}); }
        else if (m.type === 'agent_done') finish(m.error ? {error:m.error} : {});
      });
      ws.on('error', e => finish({error:`ws:${e.message}`}));
    });
  })();
}

const cases = ALL_CASES;
const byCat = Object.fromEntries(CATEGORIES.map(c => [c, {pass:0, fail:0, skip:0, fails:[]}]));
let totalPass = 0, totalSkip = 0;

for (let i = 0; i < cases.length; i++) {
  const c = cases[i];
  const cat = c.cat;
  const label = c.prompt ? c.prompt.slice(0,26) : (c.turns ? c.turns.join(' / ').slice(0,26) : '?');

  // multi-turn cases: skip
  if (Array.isArray(c.turns)) {
    totalSkip++; byCat[cat].skip++;
    console.log(`${String(i+1).padStart(3)}/${cases.length} [SKIP] (${cat.padEnd(13)}) ${label.padEnd(28)} :: multi-turn`);
    continue;
  }
  // cases needing a non-single-turn runner (confirm-flow, DB verify, llm-judge,
  // N-runs, fault injection, timing, multi-turn) — not evaluable here, skip.
  if (c.note && /needs[- ]?confirm|needs[- ]?timing|needs[- ]?fault|inject |restart backend|llm-judge|run 5x|run twice|run-n|after confirm|approve|reject|cancel|needs db|db verify|db count|multi-turn|long-context/i.test(c.note)) {
    totalSkip++; byCat[cat].skip++;
    console.log(`${String(i+1).padStart(3)}/${cases.length} [SKIP] (${cat.padEnd(13)}) ${label.padEnd(28)} :: ${c.note}`);
    continue;
  }

  const r = await runTurn(c.prompt);
  let verdict = 'FAIL', reason = '';
  if (r.error) { reason = `ERROR: ${r.error}`; }
  else {
    try { const [pass, why] = c.check(r); reason = String(why ?? '');
      if (pass && /^needs /i.test(reason)) { verdict = 'SKIP'; }
      else if (pass) { verdict = 'PASS'; }
      else { verdict = 'FAIL'; }
    } catch (e) { reason = `check threw: ${e.message}`; verdict = 'FAIL'; }
  }

  const tn = [...new Set((r.tool_calls||[]).map(t=>t.name))].join(',') || '-';
  if (verdict === 'PASS') { totalPass++; byCat[cat].pass++; }
  else if (verdict === 'SKIP') { totalSkip++; byCat[cat].skip++; }
  else { byCat[cat].fail++; byCat[cat].fails.push(c.prompt); }
  console.log(`${String(i+1).padStart(3)}/${cases.length} [${verdict}] (${cat.padEnd(13)}) ${label.padEnd(28)} tools=[${tn}]${reason?' :: '+reason.slice(0,60):''}`);
}

console.log('\n================= by category =================');
for (const c of CATEGORIES) {
  const s = byCat[c]; const n = s.pass + s.fail;
  if (n + s.skip === 0) continue;
  const rate = n ? ((s.pass / n) * 100).toFixed(0).padStart(3) : '  -';
  console.log(`  ${c.padEnd(16)} ${rate}%  (pass ${s.pass}/${n})${s.skip?` skip ${s.skip}`:''}${s.fail?'  ✗ '+s.fails.slice(0,4).map(f=>`"${(f||'').slice(0,16)}"`).join(', '):''}`);
}
const ran = cases.length - totalSkip;
console.log(`\n=== PASS ${totalPass}/${ran} (${ran?((totalPass/ran)*100).toFixed(0):0}% of runnable) | SKIP ${totalSkip}/${cases.length} (need special runners) | TOTAL ${cases.length} ===`);
