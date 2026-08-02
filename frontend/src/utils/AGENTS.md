# utils

## Responsibility [✓ auto]

Utility modules providing cross-cutting functionality: WebSocket client singleton and time formatting helpers.

### websocket.ts
Singleton `WebSocketClient` class managing the real-time connection to the backend hub. Features:
- Type-based message routing via `on(messageType, handler)` / `off(messageType, handler)` pattern
- Exponential backoff reconnection (1s base, max 5 attempts)
- Exported as `wsClient` singleton — one connection per app lifecycle
- Established in `App.vue` on mount

### time.ts
Pure time formatting functions:
- `formatTime(seconds)` — `MM:SS` display format
- `formatDateTime(dateStr)` — localized `zh-CN` datetime
- `formatDate(dateStr)` — localized `zh-CN` date
- `getRemainingTime(startTime, totalSeconds)` — calculates live countdown
- `formatDuration(seconds)` — human-readable Chinese duration (e.g., "1小时30分钟")

## Conventions [✓ auto]

- WebSocket: handler-based pub/sub pattern, not event emitter
- Time functions use Chinese locale (`zh-CN`) formatting
- `wsClient` exported as singleton — never instantiate `WebSocketClient` directly
- Co-located tests: `websocket.test.ts`, `time.spec.ts`

## Dependencies [✓ auto]

- Depends on: `@/types` (WSMessage, WSMessageType)
- Depended on by: `stores/timer` (WebSocket listeners), `App.vue` (connection lifecycle), views/components (time formatting)
