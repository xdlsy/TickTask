// Multi-turn e2e for Bug 4: after a PermWrite tool (create_task / create_schedule)
// is used and confirmed, a subsequent turn in the SAME conversation must still
// produce a real summary (previously the malformed history hung every later
// turn). Runs one scenario per write tool and reports each.
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

// One Bug-4 scenario: create an entity via a write tool, confirm it, then ask a
// follow-up in the SAME conversation and check the agent still answers coherently
// (mentioning the entity). `writeTool` is the tool we expect to reach
// pending_confirmation on turn 1.
async function scenario({ label, writeTool, createPrompt, name, followUp }) {
  const conv = await newConv();

  // Turn 1: the write tool
  const r1 = await run(conv, createPrompt);
  console.log(`[${label}] turn1 tools:`, r1.toolCalls.map(t => `${t.name}(${t.status})`).join(','), '| pending:', !!r1.pendingMsgId);
  if (r1.error) return { label, ok: false, reason: `turn1 error: ${r1.error}` };
  const reached = r1.toolCalls.some(t => t.name === writeTool && t.status === 'pending_confirmation');
  if (!reached) return { label, ok: false, reason: `${writeTool} did not reach pending_confirmation` };

  // confirm
  const conf = await fetch(`${BASE}/api/agent/confirm`, { method: 'POST', headers: { 'Content-Type': 'application/json' }, body: JSON.stringify({ message_id: r1.pendingMsgId, decision: 'approve' }) });
  console.log(`[${label}] confirm status:`, conf.status);
  // (after confirm the agent finishes the turn; we don't need to collect the tail)

  // Turn 2: a FOLLOW-UP in the SAME conversation — this is where Bug 4 hung
  const r2 = await run(conv, followUp);
  console.log(`[${label}] turn2 tools:`, r2.toolCalls.map(t => `${t.name}(${t.status})`).join(','));
  console.log(`[${label}] turn2 text:`, (r2.text || '').slice(0, 160).replace(/\n/g, ' '));

  const ok = r2.done && !r2.error && r2.text.includes(name);
  return { label, ok, reason: ok ? 'follow-up answered and mentioned the entity' : 'follow-up did not mention the entity or errored' };
}

const stamp = Date.now();
const results = [];

// Original Bug-4 guard: create_task
results.push(await scenario({
  label: 'create_task',
  writeTool: 'create_task',
  createPrompt: `帮我建个任务：e2e验证任务_${stamp}`,
  name: `e2e验证任务_${stamp}`,
  followUp: '我刚才让你建的是什么任务？',
}));

// New-tool guard: create_schedule (start+end+title given so the agent calls the
// tool directly rather than asking for missing times)
results.push(await scenario({
  label: 'create_schedule',
  writeTool: 'create_schedule',
  createPrompt: `帮我在日历上加个会：明天 15:00-15:30「产品评审_${stamp}」`,
  name: `产品评审_${stamp}`,
  followUp: '我刚才让你加的是什么日程？',
}));

console.log('\n=== Bug4 multi-turn e2e ===');
let allOk = true;
for (const r of results) {
  console.log(`${r.ok ? 'PASS' : 'FAIL'} [${r.label}] :: ${r.reason}`);
  if (!r.ok) allOk = false;
}
if (!allOk) process.exit(1);
