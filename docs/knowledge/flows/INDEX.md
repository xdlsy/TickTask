# Flow Blueprints Index

核心业务流程的 Mermaid 时序图，展示参与模块、调用链和异常路径。

## 流程清单

| 流程 | 蓝图 | 类型 | 参与模块数 | 说明 |
|------|------|------|-----------|------|
| Task Lifecycle | [task-lifecycle.md](task-lifecycle.md) | 同步请求 | 6 | 任务 CRUD + 象限移动 + 状态流转 |
| AI Schedule Generation | [ai-schedule-generation.md](ai-schedule-generation.md) | 混合 (REST + AI 异步) | 7 | LLM 生成日程 + 修订工作流 |
| Timer Session | [timer-session.md](timer-session.md) | 异步事件 | 6 | goroutine 倒计时 + WebSocket 推送 |
| WebSocket Real-time | [websocket-realtime.md](websocket-realtime.md) | 异步事件 | 4 | 连接管理 + 消息广播 + 自动重连 |
| Schedule Revision | [schedule-revision.md](schedule-revision.md) | 混合 | 7 | 两阶段修订：预览 → 确认应用 |

## 按触发方式浏览

**用户触发 (REST)**
- [Task Lifecycle](task-lifecycle.md) — 创建/编辑/删除/移动任务
- [AI Schedule Generation](ai-schedule-generation.md) — 一键生成日程
- [Schedule Revision](schedule-revision.md) — AI 修订日程

**系统驱动 (Goroutine + WebSocket)**
- [Timer Session](timer-session.md) — 计时器倒计时推送
- [WebSocket Real-time](websocket-realtime.md) — 实时消息分发

## 流程间关联

```
AI Schedule Generation ──修订──→ Schedule Revision
Timer Session ──推送──→ WebSocket Real-time
Task Lifecycle ──状态变更──→ WebSocket Real-time (task_updated)
```

## 导航

- [知识库总览](../README.md)
- [模块蓝图](../modules/) — 按模块查阅
- [架构决策](../decisions/) — 按决策查阅
