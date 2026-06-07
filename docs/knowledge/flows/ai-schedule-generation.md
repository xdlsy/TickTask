# Flow: AI Schedule Generation (AI 日程生成)

> 从任务列表出发，通过 LLM 生成每日日程的完整流程，含修订工作流。

## Sequence Diagram

```mermaid
sequenceDiagram
    actor User
    participant View as views/Schedule.vue
    participant Store as useScheduleStore
    participant API as api/client
    participant Handler as ScheduleHandler
    participant Svc as ScheduleService
    participant AISvc as AIService
    participant LLM as ai.LLMClient
    participant Repo as ScheduleRepository
    participant AIProv as AI Provider

    rect rgb(250, 249, 246)
        Note over User,AIProv: 阶段 1: AI 生成日程
        User->>View: 点击"AI 生成日程"
        View->>Store: generateFromAI(startTime, endTime)
        Store->>API: POST /api/schedules/generate (360s timeout)
        API->>Handler: GenerateSchedule
        Handler->>Svc: GenerateScheduleWithAI(start, end)
        Svc->>Repo: DeleteTaskSchedulesByDateRange()
        Note over Svc: 清除旧的任务类日程
        Svc->>AISvc: GenerateSchedule(tasks, start, end)
        AISvc->>LLM: ChatCompletion(ctx, prompt)
        LLM->>AIProv: HTTPS POST (JSON prompt)
        AIProv-->>LLM: JSON response
        LLM-->>AISvc: JSON string
        AISvc->>AISvc: extractJSON() 剥离 markdown
        AISvc-->>Svc: []ScheduleEvent
        Svc->>Repo: Create (批量写入)
        Repo-->>Svc: OK
        Svc-->>Handler: events
        Handler-->>API: JSON 200
        API-->>Store: response
        Store-->>View: 更新日历视图
    end

    rect rgb(212, 168, 67, 0.1)
        Note over User,AIProv: 阶段 2: 日程修订 (可选)
        User->>View: 输入修订指令 (中文)
        View->>Store: reviseSchedule(prompt)
        Store->>API: POST /api/schedules/revise (360s timeout)
        API->>Handler: ReviseSchedule
        Handler->>Svc: ReviseSchedule(prompt)
        Svc->>AISvc: 发送当前日程 + 修订指令
        AISvc->>LLM: ChatCompletion(ctx, revisePrompt)
        LLM->>AIProv: HTTPS POST
        AIProv-->>LLM: JSON (修订后日程)
        LLM-->>Svc: ReviseResponse {events, summary}
        Svc-->>Handler: ReviseResponse (不持久化)
        User->>View: 确认应用修订
        View->>Store: applyRevision()
        Store->>Svc: ApplyRevision()
        Svc->>Repo: DeleteAll + Create (替换日程)
    end
```

## 参与模块

| 模块 | 角色 | 蓝图 |
|------|------|------|
| views/Schedule | UI 入口 | - |
| stores/schedule | 状态管理 | [stores](../modules/stores.md) |
| api/client | HTTP (360s 超时) | [api-client](../modules/api-client.md) |
| handler/ScheduleHandler | 请求处理 | [api-handler](../modules/api-handler.md) |
| service/ScheduleService | 业务逻辑 | [service](../modules/service.md) |
| service/AIService | LLM 调用 | [service](../modules/service.md) |
| ai/LLMClient | HTTP LLM 客户端 | [ai](../modules/ai.md) |

## 异常路径

| 场景 | 处理方式 |
|------|----------|
| AI 未配置 (无 API Key) | AIService 返回 nil → Handler 返回 503 |
| LLM 超时 | `context.WithTimeout` (60s 后端, 360s 前端) → 503 |
| LLM 返回非 JSON | `extractJSON()` 尝试剥离 markdown fence；失败 → 500 |
| LLM JSON 解析失败 | ScheduleService 返回错误 → Handler 500 |
| 时间区间无效 | ScheduleService 验证 → 400 |

## 关联

| 类型 | 链接 |
|------|------|
| 关联模块 | [ai](../modules/ai.md), [service](../modules/service.md) |
| 关联 ADR | [ADR-0004](../decisions/adr-0004-openai-compatible-ai.md) |
