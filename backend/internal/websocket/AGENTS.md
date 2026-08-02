# websocket

## Responsibility [~ inferred]

Real-time WebSocket communication hub for streaming timer state to connected frontend clients. Uses the standard Go hub+client pattern with goroutine-per-client read/write pumps.

Files:
- `hub.go` — `Hub` struct (client registry with `sync.RWMutex`), `Client` struct (connection + buffered send channel), typed broadcast methods for timer events

## Conventions [~ inferred]

- Hub owns a `map[*Client]bool` protected by `sync.RWMutex`
- Clients have a buffered `send chan []byte` (capacity 256)
- Non-blocking send with slow-client eviction: `select { case client.send <- msg: default: unregister }`
- Typed broadcast methods (`BroadcastTimerTick`, `BroadcastSessionState`, `BroadcastTimerComplete`, `BroadcastTaskUpdated`, `BroadcastTerminalOutput`, `BroadcastTerminalStatus`) — each marshals a specific message struct to JSON
- Terminal broadcast methods added for terminal output streaming (`BroadcastTerminalOutput` with chunk + stderr flag, `BroadcastTerminalStatus` with status/message/detail)
- `readPump()` drains inbound messages (no application-level inbound handling); `writePump()` writes from channel
- `WebSocketHandler` upgrades HTTP to WS via `gorilla/websocket` Upgrader with `CheckOrigin` returning true

## Dependencies [✓ auto]

- Depends on: `github.com/gorilla/websocket`
- Depended on by: `service/` (TimerService calls broadcast methods), `api/` (router registers `/ws` endpoint)
