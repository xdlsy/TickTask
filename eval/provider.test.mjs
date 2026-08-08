import { test } from 'node:test';
import assert from 'node:assert/strict';
import http from 'node:http';
import { WebSocketServer, WebSocket } from 'ws';
import { collect } from './provider.mjs';

// Mocks the agent REST + WS surface with a scripted event sequence and asserts
// the provider reconstructs {tool_calls, assistant_text} from WS events. This
// is the "verify the verifier" check — it does not call any real LLM.
test('collect reconstructs tool_calls and assistant_text from WS events', async () => {
  const toSend = [
    { type: 'agent_tool', conversation_id: 'conv-1', tool_name: 'list_schedule', args: { from: '2026-08-09', to: '2026-08-09' }, status: 'succeeded' },
    { type: 'agent_message', conversation_id: 'conv-1', delta_text: '你今天有 3 个安排' },
    { type: 'agent_done', conversation_id: 'conv-1', finish_reason: 'stop' },
  ];

  const httpServer = http.createServer((req, res) => {
    if (req.method === 'POST' && req.url === '/api/agent/conversations') {
      res.setHeader('Content-Type', 'application/json');
      res.end(JSON.stringify({ id: 'conv-1' }));
    } else if (req.method === 'POST' && req.url === '/api/agent/chat') {
      res.statusCode = 202;
      res.end();
      // give the WS client a moment to be registered before broadcasting
      setTimeout(() => {
        for (const c of wss.clients) {
          if (c.readyState === WebSocket.OPEN) toSend.forEach((e) => c.send(JSON.stringify(e)));
        }
      }, 50);
    } else {
      res.statusCode = 404;
      res.end();
    }
  });

  const wss = new WebSocketServer({ noServer: true });
  httpServer.on('upgrade', (req, socket, head) => {
    wss.handleUpgrade(req, socket, head, () => {});
  });

  await new Promise((r) => httpServer.listen(0, r));
  const port = httpServer.address().port;

  try {
    const r = await collect({
      baseUrl: `http://localhost:${port}`,
      wsUrl: `ws://localhost:${port}/ws`,
      prompt: '我一会有啥安排吗？',
      timeoutMs: 2000,
    });
    assert.equal(r.error, undefined, `unexpected error: ${r.error}`);
    assert.equal(r.tool_calls.length, 1);
    assert.equal(r.tool_calls[0].name, 'list_schedule');
    assert.equal(r.tool_calls[0].status, 'succeeded');
    assert.equal(r.assistant_text, '你今天有 3 个安排');
  } finally {
    wss.close();
    httpServer.close();
  }
});

test('collect surfaces a timeout error when agent_done never arrives', async () => {
  const httpServer = http.createServer((req, res) => {
    if (req.method === 'POST' && req.url === '/api/agent/conversations') {
      res.end(JSON.stringify({ id: 'conv-2' }));
    } else if (req.method === 'POST' && req.url === '/api/agent/chat') {
      res.statusCode = 202;
      res.end(); // never broadcast done
    } else {
      res.statusCode = 404;
      res.end();
    }
  });
  const wss = new WebSocketServer({ noServer: true });
  httpServer.on('upgrade', (req, socket, head) => wss.handleUpgrade(req, socket, head, () => {}));
  await new Promise((r) => httpServer.listen(0, r));
  const port = httpServer.address().port;
  try {
    const r = await collect({
      baseUrl: `http://localhost:${port}`,
      wsUrl: `ws://localhost:${port}/ws`,
      prompt: 'x',
      timeoutMs: 200,
    });
    assert.equal(r.error, 'timeout');
  } finally {
    wss.close();
    httpServer.close();
  }
});
