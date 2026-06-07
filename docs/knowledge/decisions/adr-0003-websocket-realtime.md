# ADR-0003: WebSocket for Real-time Timer Updates

**Status**: Accepted
**Date**: 2026-06-07
**Context**: [websocket](../modules/websocket.md) | [Timer Session](../flows/timer-session.md) | [WebSocket Real-time](../flows/websocket-realtime.md)

## Context

番茄钟计时器需要每秒向前端推送倒计时状态（剩余时间、进度百分比）。需要选择实时通信方案。

## Decision

使用 **WebSocket** 实现服务端到前端的实时推送。后端使用 gorilla/websocket 实现 Hub+Client 模式，前端使用自定义 WebSocket 客户端单例。

## Alternatives Considered

| 方案 | 优点 | 缺点 |
|------|------|------|
| **WebSocket** (选中) | 全双工、低延迟、适合高频推送 | 需要管理连接生命周期、重连逻辑 |
| Server-Sent Events (SSE) | 简单、HTTP 协议兼容 | 单向推送（足够本场景）、浏览器兼容性 |
| HTTP 轮询 | 实现最简单 | 每秒一次请求开销大、延迟高 |
| 长轮询 | 比短轮询高效 | 实现复杂、不如 WebSocket 高效 |

## POS (正面影响)

- **实时性**：1 秒 tick 延迟极低，用户体验流畅
- **双向通信**：虽然当前仅用服务端推送，但预留了双向能力
- **连接复用**：单连接承载所有实时消息（timer_tick, session_state, task_updated 等）
- **类型化消息**：通过 `type` 字段实现消息路由

## NEG (负面影响)

- **连接管理复杂**：需处理断线重连、慢客户端驱逐、后端重启恢复
- **goroutine 开销**：每个 WebSocket 连接占用 2 个 goroutine (readPump + writePump)
- **状态恢复**：后端重启时 goroutine 丢失，需 REST 兜底恢复计时器状态

## IMP (实施要点)

- 后端：Hub 维护 `map[*Client]bool`，`sync.RWMutex` 保护
- 前端：`wsClient` 单例在 App.vue `onMounted` 初始化
- 重连策略：指数退避 (1s → 2s → 4s → ... → 30s max)
- 慢客户端：非阻塞 send + 缓冲区 256，满则驱逐
- TimerStore 双路径：REST 获取初始状态 + WebSocket 接收实时更新

## REF (参考)

- [websocket 蓝图](../modules/websocket.md)
- [stores 蓝图](../modules/stores.md)
- [Timer Session 流程](../flows/timer-session.md)
- [WebSocket Real-time 流程](../flows/websocket-realtime.md)
