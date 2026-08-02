# Module: stores (Pinia 状态管理)

> 前端 Pinia 状态管理层，Composition API 模式，直接调用 API 客户端。

## Component Diagram

```mermaid
graph TB
    subgraph "frontend/src/stores/"
        Task["useTaskStore<br/>任务 CRUD<br/>双表示: flat + byQuadrant"]
        Timer["useTimerStore<br/>计时器生命周期<br/>REST + WS 双路径"]
        Schedule["useScheduleStore<br/>日历 CRUD<br/>日/周/月视图切换"]
        AI["useAiStore<br/>AI 分类/排程/优先级<br/>配置状态检查"]
        App["useAppStore<br/>UI 状态<br/>通知队列"]
    end

    Task -->|api.getTasks 等| APIC["api/client"]
    Timer -->|api.createSession 等| APIC
    Schedule -->|api.generateSchedule 等| APIC
    AI -->|api.classifyTask 等| APIC
    Timer -->|"wsClient.on('timer_tick')"| WS["utils/websocket"]

    style Task fill:#B8452C,color:#fff
    style Timer fill:#D4A843,color:#1C1B1A
    style Schedule fill:#4A90D9,color:#fff
    style AI fill:#4A90D9,color:#fff
    style App fill:#6E6A65,color:#fff
```

## 对外接口

| Store | 关键 State | 关键 Actions |
|-------|-----------|--------------|
| useTaskStore | `tasks`, `tasksByQuadrant`, `loading` | `fetchTasks`, `createTask`, `updateTask`, `deleteTask`, `moveTask` |
| useTimerStore | `activeSession`, `timerState`, `remaining` | `startSession`, `pauseSession`, `resumeSession`, `setupWebSocket` |
| useScheduleStore | `schedules`, `currentDate`, `viewMode` | `fetchSchedules`, `generateFromAI`, `reviseSchedule`, `applyRevision` |
| useAiStore | `aiStatus`, `loading` | `classifyTask`, `generateSchedule`, `checkConfig` |
| useAppStore | `currentView`, `sidebarOpen`, `notifications` | `addNotification`, `toggleSidebar` |

## 关键设计

- 所有 state 使用 `ref()`，不使用 `reactive()`
- 异步 action 遵循 `try/catch/finally` + `loading` flag 模式
- **TimerStore** 双路径更新：REST 获取初始状态，WebSocket 接收实时推送
- Store 直接调用 API 客户端，无中间 service 层
- 测试中每个 `beforeEach` 必须 `setActivePinia(createPinia())`

## 关联

| 类型 | 链接 |
|------|------|
| 依赖 | [api-client](api-client.md), `types/`, `utils/websocket` |
| 消费模块 | `views/`, `components/`, `App.vue` |
| 关联流程 | [Task Lifecycle](../flows/task-lifecycle.md), [Timer Session](../flows/timer-session.md), [AI Schedule](../flows/ai-schedule-generation.md) |
