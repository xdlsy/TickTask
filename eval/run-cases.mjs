// Runs the comprehensive case suite (cases.mjs) against the live backend via
// collect(). Resolves a turn on agent_done OR pending_confirmation (so write/
// dangerous cases don't wait the 30-min ConfirmationTimeout). Reports per-
// category and overall pass-rate.
import { WebSocket } from 'ws';
import { buildCases, CATEGORIES } from './cases.mjs';

const BASE = process.env.AGENT_BASE_URL || 'http://localhost:8080';
const WS = process.env.AGENT_WS_URL || 'ws://localhost:8080/ws';

function runTurn(prompt, timeoutMs = 60_000) {
  return (async () => {
    const conv = await fetch(`${BASE}/api/agent/conversations`, { method: 'POST' }).then(r => r.json());
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

const cases = buildCases();
const byCat = Object.fromEntries(CATEGORIES.map(c => [c, {pass:0, fail:0, fails:[]}]));
let totalPass = 0;

for (let i = 0; i < cases.length; i++) {
  const { cat, prompt, check } = cases[i];
  const r = await runTurn(prompt);
  let pass = false, reason = '';
  if (r.error) { reason = `ERROR: ${r.error}`; }
  else { try { [pass, reason] = check(r); } catch (e) { reason = `check threw: ${e.message}`; } }
  const tag = pass ? 'PASS' : 'FAIL';
  if (pass) { totalPass++; byCat[cat].pass++; }
  else { byCat[cat].fail++; byCat[cat].fails.push(prompt); }
  const tn = [...new Set((r.tool_calls||[]).map(t=>t.name))].join(',') || '-';
  console.log(`${String(i+1).padStart(2)}/${cases.length} [${tag}] (${cat.padEnd(12)}) ${prompt.slice(0,28).padEnd(30)} tools=[${tn}]${reason?' :: '+reason.slice(0,70):''}`);
}

console.log('\n================= by category =================');
for (const c of CATEGORIES) {
  const s = byCat[c]; const n = s.pass + s.fail;
  if (n === 0) continue;
  const rate = ((s.pass / n) * 100).toFixed(0).padStart(3);
  console.log(`  ${c.padEnd(14)} ${rate}%  (${s.pass}/${n})${s.fail?'  ✗ '+s.fails.map(f=>`"${f.slice(0,16)}"`).join(', '):''}`);
}
console.log(`\n=== OVERALL ${totalPass}/${cases.length} (${((totalPass/cases.length)*100).toFixed(0)}%) ===`);
