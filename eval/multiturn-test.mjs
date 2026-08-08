// Multi-turn e2e for Bug 4: after a PermWrite tool (create_task) is used and
// confirmed, a subsequent turn must still produce a real summary (previously
// the malformed history hung every later turn).
import { WebSocket } from 'ws';
const BASE = process.env.AGENT_BASE_URL || 'http://localhost:8080';
const WS = process.env.AGENT_WS_URL || 'ws://localhost:8080/ws';

async function newConv() {
  return (await fetch(`${BASE}/api/agent/conversations`, { method: 'POST' }).then(r => r.json())).id;
}

// Collects events for convId until agent_done OR pending_confirmation (returns
// {text, toolCalls, pendingMsgId, done, error}).
function run(convId, prompt, timeoutMs = 60_000) {
  return new Promise((resolve) => {
    const toolCalls = []; let text = ''; let settled = false;
    const finish = (x) => { if (settled) return; settled = true; clearTimeout(t); try{ws.close()}catch{}; resolve({text, toolCalls, ...(x||{})}); };
    const ws = new WebSocket(WS);
    const t = setTimeout(() => finish({error:'timeout'}), timeoutMs);
    ws.on('open', () => fetch(`${BASE}/api/agent/chat`, {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({conversation_id: convId, text: prompt})}).catch(e=>finish({error:`chat:${e.message}`})));
    ws.on('message', (data) => {
      let m; try{ m = JSON.parse(data.toString()); } catch { return; }
      if (m.conversation_id !== convId) return;
      if (m.type === 'agent_message') text += m.delta_text || '';
      else if (m.type === 'agent_tool') { toolCalls.push({name:m.tool_name, status:m.status}); if (m.status === 'pending_confirmation') finish({pendingMsgId: m.message_id}); }
      else if (m.type === 'agent_done') finish({done:true, error: m.error});
    });
    ws.on('error', e => finish({error:`ws:${e.message}`}));
  });
}

const conv = await newConv();

// Turn 1: write tool (create_task)
const TASK = 'e2e验证任务_' + Date.now();
let r1 = await run(conv, `帮我建个任务：${TASK}`);
console.log('turn1 tools:', r1.toolCalls.map(t=>`${t.name}(${t.status})`).join(','), '| pending:', !!r1.pendingMsgId);
if (r1.error) { console.log('FAIL turn1 error:', r1.error); process.exit(1); }
if (!r1.pendingMsgId) {
  console.log('FAIL: create_task did not reach pending_confirmation'); process.exit(1);
}
// confirm
const conf = await fetch(`${BASE}/api/agent/confirm`, {method:'POST', headers:{'Content-Type':'application/json'}, body: JSON.stringify({message_id: r1.pendingMsgId, decision:'approve'})});
console.log('confirm status:', conf.status);
// (after confirm the agent finishes the turn; we don't need to collect the tail)

// Turn 2: a FOLLOW-UP in the SAME conversation — this is where Bug 4 hung
const r2 = await run(conv, '我刚才让你建的是什么任务？');
console.log('turn2 tools:', r2.toolCalls.map(t=>`${t.name}(${t.status})`).join(','));
console.log('turn2 text:', (r2.text||'').slice(0,160).replace(/\n/g,' '));

const ok = r2.done && !r2.error && r2.text.includes(TASK);
console.log(`\n=== Bug4 e2e: ${ok ? 'PASS' : 'FAIL'} (follow-up after write tool produced a real answer mentioning the task) ===`);
if (!ok) process.exit(1);
