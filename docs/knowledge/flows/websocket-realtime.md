# Flow: WebSocket Real-time Updates (实时更新)

> WebSocket 连接建立、消息广播和自动重连的完整流程。

## Sequence Diagram

```mermaid
sequenceDiagram
    participant App as App.vue (mount)
    participant WS as utils/websocket<br/>(wsClient singleton)
    participant Hub as websocket.Hub
    participant Timer as TimerService
    participant Store as useTimerStore

    rect rgb(74, 144, 217, 0.1)
        Note over App,Hub: 连接建立
        App->>WS: wsClient.connect()
        WS->>Hub: GET /ws (HTTP Upgrade)
        Hub->>Hub: NewClient(conn) → register
        Hub-->>WS: Connection established
    end

    rect rgb(184, 69, 44, 0.1)
        Note over Timer,Store: 消息广播 (计时器运行中)
        loop 每秒
            Timer->>Hub: BroadcastTimerTick(remaining, total)
            Hub->>Hub: JSON marshal → 遍历 clients
            Hub-->>WS: {"type":"timer_tick","data":{...}}
        end
        Timer->>Hub: BroadcastSessionState("running")
        Hub-->>WS: {"type":"session_state","data":{...}}
    end

    rect rgb(212, 168, 67, 0.1)
        Note over WS,Store: 前端消息路由
        WS->>WS: wsClient.onMessage(msg)
        WS->>WS: JSON parse → 按 type 路由
        WS->>Store: callback("timer_tick", data)
        Store->>Store: 更新 remaining / progress
    end

    rect rgb(110, 106, 101, 0.1)
        Note over App,Hub: 断线重连
        Note over WS: 连接断开
        WS->>WS: 指数退避等待 (1s, 2s, 4s, 8s, max 30s)
        WS->>Hub: GET /ws (重新 Upgrade)
        Hub->>Hub: register new client
        WS->>Store: 触发 REST fetchActiveSession 恢复状态
    end
```

## 消息类型

| type | 触发时机 | 数据字段 |
|------|----------|----------|
| `timer_tick` | 每秒 (计时器运行中) | `remaining`, `total` |
| `session_state` | 暂停/恢复/开始 | `state` (running/paused/completed) |
| `timer_complete` | 计时自然结束 | `sessionType` (work/short_break/long_break) |
| `task_updated` | 任务状态变更 | `task` (完整 Task 对象) |
| `terminal_output` | 终端输出流 | `chunk`, `isStderr` |
| `terminal_status` | 终端状态 | `status`, `message` |

## 参与模块

| 模块 | 角色 | 蓝图 |
|------|------|------|
| App.vue | 连接初始化 | - |
| utils/websocket | 客户端单例 | - |
| websocket/Hub | 服务端 Hub | [websocket](../modules/websocket.md) |
| stores/timer | 消费 tick 消息 | [stores](../modules/stores.md) |

## 关键设计

- **单例连接**：`wsClient` 在 App.vue 的 `onMounted` 中初始化一次
- **类型化路由**：消息按 `type` 字段分发给对应 callback
- **指数退避重连**：1s → 2s → 4s → 8s → ... → 30s max
- **慢客户端驱逐**：Hub 端非阻塞 send，缓冲满则断开

## 异常路径

| 场景 | 处理方式 |
|------|----------|
| 后端重启 | WebSocket 断开 → 前端自动重连 → REST 恢复状态 |
| 网络波动 | 指数退避重连 |
| 消息解析失败 | console.error + 丢弃消息 |
| 多 Tab 打开 | 每个 Tab 独立 WebSocket 连接，Hub 广播到所有 |

## 关联

| 类型 | 链接 |
|------|------|
| 关联模块 | [websocket](../modules/websocket.md), [stores](../modules/stores.md) |
| 关联流程 | [Timer Session](timer-session.md) |
| 关联 ADR | [ADR-0003](../decisions/adr-0003-websocket-realtime.md) |
