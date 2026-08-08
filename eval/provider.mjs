import { WebSocket } from 'ws';

// Default env-driven entry point Promptfoo calls. Returns Promptfoo's
// { output: <json string> } shape; assertions parse output back to JSON.
export async function callApi(prompt) {
  const baseUrl = process.env.AGENT_BASE_URL || 'http://localhost:8080';
  const wsUrl = process.env.AGENT_WS_URL || 'ws://localhost:8080/ws';
  const result = await collect({ baseUrl, wsUrl, prompt, timeoutMs: 60_000 });
  return { output: JSON.stringify(result) };
}

// Core logic, exported for unit testing with mock servers. Creates a
// conversation, opens WS, posts the chat, collects agent_* events for that
// conversation until agent_done (or timeout), and returns the reconstruction.
export function collect({ baseUrl, wsUrl, prompt, timeoutMs }) {
  return (async () => {
    const conv = await fetch(`${baseUrl}/api/agent/conversations`, { method: 'POST' }).then((r) => r.json());
    const convId = conv.id;

    return new Promise((resolve) => {
      const toolCalls = [];
      let assistantText = '';
      let settled = false;

      const finish = (extra) => {
        if (settled) return;
        settled = true;
        clearTimeout(timer);
        try { ws.close(); } catch {}
        resolve({ tool_calls: toolCalls, assistant_text: assistantText, ...(extra || {}) });
      };

      const ws = new WebSocket(wsUrl);
      const timer = setTimeout(() => finish({ error: 'timeout' }), timeoutMs);

      ws.on('open', () => {
        fetch(`${baseUrl}/api/agent/chat`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ conversation_id: convId, text: prompt }),
        }).catch((e) => finish({ error: `chat POST failed: ${e.message}` }));
      });
      ws.on('message', (data) => {
        let m;
        try { m = JSON.parse(data.toString()); } catch { return; }
        if (m.conversation_id !== convId) return;
        if (m.type === 'agent_message') {
          assistantText += m.delta_text || '';
        } else if (m.type === 'agent_tool') {
          toolCalls.push({ name: m.tool_name, args: m.args, status: m.status });
        } else if (m.type === 'agent_done') {
          finish(m.error ? { error: m.error } : {});
        }
      });
      ws.on('error', (e) => finish({ error: `ws error: ${e.message}` }));
    });
  })();
}

export default { callApi };
