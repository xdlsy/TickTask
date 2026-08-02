# Module: service (业务逻辑层)

> 后端核心业务逻辑编排层，协调 Repository、AI 客户端和 WebSocket Hub。

## Component Diagram

```mermaid
graph TB
    subgraph "service/"
        TS["TaskService<br/>任务 CRUD、象限移动、状态流转"]
        TMS["TimerService<br/>Goroutine 倒计时、频道控制"]
        AIS["AIService<br/>LLM 懒初始化、JSON 解析"]
        SS["ScheduleService<br/>日程验证、AI 生成、修订、ICS"]
        AS["AnalyticsService<br/>汇总/趋势/分布查询"]
        CW["ConfigWriter<br/>AI 配置回写 config.yaml"]
    end

    TS -->|任务状态变更| AS
    TMS -->|Broadcast*()| WS["websocket.Hub"]
    AIS -->|ChatCompletion()| AI["ai.LLMClient"]
    SS -->|AI 生成/修订| AIS
    SS -->|时间区间查询| SR["repository.ScheduleRepo"]

    style TS fill:#B8452C,color:#fff
    style TMS fill:#D4A843,color:#1C1B1A
    style AIS fill:#4A90D9,color:#fff
    style SS fill:#4A90D9,color:#fff
    style AS fill:#6E6A65,color:#fff
    style CW fill:#9C9893,color:#fff
```

## 对外接口

| 服务 | 入站方法 | 说明 |
|------|----------|------|
| TaskService | GetAll/GetByQuadrant/GetByID/Create/Update/Delete/Move | 任务 CRUD + 象限移动 |
| TimerService | CreateSession/ControlSession(pause/resume/complete/abandon) | 计时器生命周期 |
| AIService | ClassifyTask/GenerateSchedule/GetPrioritySuggestions | AI 功能（降级：无 Key 时返回 nil） |
| ScheduleService | Create/Update/Delete/GetByRange/GenerateFromAI/Revise/Apply | 日程管理 + AI 排程 |
| AnalyticsService | GetSummary/GetTrend/GetDistribution | 分析数据查询 |

## 关键设计

- **TimerService** 使用 goroutine + 1秒 ticker，通过 `stopChan/pauseChan/resumeChan` 控制生命周期
- **AIService** 懒初始化：首次调用时检查 API Key，无 Key 则返回 nil（降级模式）
- **DTO 类型**与 Service 共存：`CreateTaskRequest`, `UpdateTaskRequest`, `CreateScheduleDTO`

## 关联

| 类型 | 链接 |
|------|------|
| 依赖模块 | [repository](repository.md), [ai](ai.md), [websocket](websocket.md), [model](model.md) |
| 消费模块 | [api/handler](api-handler.md) |
| 关联流程 | [Task Lifecycle](../flows/task-lifecycle.md), [AI Schedule](../flows/ai-schedule-generation.md), [Timer Session](../flows/timer-session.md) |
| 关联 ADR | [ADR-0002](../decisions/adr-0002-manual-di.md) |
