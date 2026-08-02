# Module: api/handler (HTTP 传输层)

> Gin HTTP Handler 层，薄层设计：解析输入 -> 委托 Service -> 返回 JSON。

## Component Diagram

```mermaid
graph TB
    subgraph "api/"
        Router["router.go<br/>路由注册 + CORS + SPA fallback"]
        CORS["middleware/cors.go<br/>跨域配置"]
    end

    subgraph "api/handler/"
        TH["TaskHandler<br/>7 routes"]
        TMH["TimerHandler<br/>5 routes"]
        AIH["AIHandler<br/>7 routes + timeout"]
        SH["SettingHandler<br/>3 routes, API Key 脱敏"]
        AH["AnalyticsHandler<br/>3 routes"]
        SCH["ScheduleHandler<br/>9 routes"]
    end

    Router --> TH & TMH & AIH & SH & AH & SCH
    Router --> CORS

    TH --> TaskSvc["service.TaskService"]
    TMH --> TimerSvc["service.TimerService"]
    AIH --> AiSvc["service.AIService"]
    SCH --> SchedSvc["service.ScheduleService"]

    style Router fill:#B8452C,color:#fff
    style TH fill:#D4A843,color:#1C1B1A
    style TMH fill:#D4A843,color:#1C1B1A
    style AIH fill:#4A90D9,color:#fff
    style SH fill:#6E6A65,color:#fff
    style AH fill:#6E6A65,color:#fff
    style SCH fill:#4A90D9,color:#fff
```

## 路由表

| 方法 | 路径 | Handler | 说明 |
|------|------|---------|------|
| GET | `/api/tasks` | TaskHandler.GetTasks | 获取所有任务 |
| GET | `/api/tasks/quadrant` | TaskHandler.GetByQuadrant | 按象限分组 |
| POST | `/api/tasks` | TaskHandler.Create | 创建任务 |
| PUT | `/api/tasks/:id` | TaskHandler.Update | 更新任务 |
| DELETE | `/api/tasks/:id` | TaskHandler.Delete | 删除任务 |
| PATCH | `/api/tasks/:id/move` | TaskHandler.Move | 移动象限 |
| POST | `/api/sessions` | TimerHandler.Create | 创建计时会话 |
| POST | `/api/ai/classify` | AIHandler.Classify | AI 分类 (30s) |
| POST | `/api/ai/schedule` | AIHandler.GenerateSchedule | AI 排程 (60s) |
| POST | `/api/schedules/generate` | ScheduleHandler.GenerateFromAI | AI 生成日程 (360s) |
| POST | `/api/schedules/revise` | ScheduleHandler.Revise | AI 修订日程 (360s) |
| GET | `/ws` | Hub.WebSocketHandler | WebSocket 升级 |

## HTTP 状态码映射

| Service 错误 | HTTP 状态码 |
|-------------|------------|
| 验证失败 | 400 |
| 记录不存在 | 404 |
| 内部错误 | 500 |
| AI 未配置 | 503 |

## 关联

| 类型 | 链接 |
|------|------|
| 依赖 | [service](service.md), [websocket](websocket.md) |
| 消费模块 | `cmd/server/main.go` (路由注册) |
| 关联流程 | [Task Lifecycle](../flows/task-lifecycle.md), [AI Schedule](../flows/ai-schedule-generation.md) |
