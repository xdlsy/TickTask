# stores

## Responsibility [~ inferred]

Pinia state management layer using Composition API pattern (`defineStore` + `ref()`/`computed()`). Stores contain all business logic for data fetching, mutation, and computed state — views are thin orchestrators that consume store state.

Stores:
- `useTaskStore` — task CRUD, dual data representation (flat list + `tasksByQuadrant` grouped dictionary), quadrant helpers
- `useTimerStore` — Pomodoro session lifecycle, dual REST+WebSocket update path, computed status/progress
- `useScheduleStore` — calendar events CRUD, view mode switching (day/week/month), date navigation with delta logic
- `useAgentStore` — agent conversation lifecycle, tool execution (`runTool`), AI status check, drawer state; embedded AI buttons (TaskForm/TaskCard/Dashboard/Analytics) call `runTool` and cast the bare result; WS events are dispatched via a self-registered listener in `utils/websocket`
- `useAppStore` — current view tracking, sidebar toggle, auto-dismissing notification queue

## Conventions [~ inferred]

- All state uses `ref()`, not `reactive()`
- Async actions follow `try/catch/finally` with `loading` flag management
- Stores call API client directly (no intermediate repository/service layer)
- Stores use `useXxxStore` naming, Composition API `defineStore('id', () => { ... })`
- Error handling: `console.error` in store, `ElMessage.error` propagated for UI
- Timer store registers WebSocket listeners via `setupWebSocket()` called from `App.vue`

## Dependencies [✓ auto]

- Depends on: `api/` (Axios client), `types/`, `utils/websocket` (timer store), `pinia`
- Depended on by: `views/`, `components/`, `App.vue`
