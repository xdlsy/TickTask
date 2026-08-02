# Flow: Schedule Revision (日程修订)

> AI 日程修订工作流：用户提出修改需求 → LLM 生成修订方案 → 用户确认 → 应用修订。

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

    rect rgb(250, 249, 246)
        Note over User,Repo: 阶段 1: 请求修订
        User->>View: 输入修订指令 (如"把下午的任务提前")
        View->>Store: reviseSchedule(prompt)
        Store->>API: POST /api/schedules/revise (360s timeout)
        API->>Handler: ReviseSchedule
        Handler->>Svc: ReviseSchedule(prompt)

        Svc->>Repo: GetByTimeRange(today)
        Repo-->>Svc: currentSchedules
        Note over Svc: 构建修订 prompt:<br/>当前日程 + 用户指令
        Svc->>AISvc: ChatCompletion(revisePrompt)
        AISvc->>LLM: ChatCompletion(ctx, prompt)
        LLM-->>AISvc: JSON (修订后日程)
        AISvc->>AISvc: extractJSON()
        AISvc-->>Svc: revised events

        Note over Svc: 存储修订到内存<br/>(不写入 DB)
        Svc-->>Handler: ReviseResponse {events, summary}
        Handler-->>API: JSON 200
        API-->>Store: response
        Store-->>View: 显示修订预览
    end

    rect rgb(184, 69, 44, 0.1)
        Note over User,Repo: 阶段 2: 确认应用
        User->>View: 确认修订结果
        View->>Store: applyRevision()
        Store->>API: POST /api/schedules/apply-revision
        API->>Handler: ApplyRevision
        Handler->>Svc: ApplyRevision()

        Svc->>Repo: DeleteAll()
        Note over Svc: 清除当日所有日程
        Svc->>Repo: Create (批量写入修订后日程)
        Note over Svc: 清空内存中的修订缓存
        Svc-->>Handler: applied events
        Handler-->>API: JSON 200
        API-->>Store: response
        Store-->>View: 更新日历视图
    end

    rect rgb(110, 106, 101, 0.1)
        Note over User,View: 阶段 3: 拒绝修订 (可选)
        User->>View: 取消修订
        View->>Store: discardRevision()
        Note over Store: 清空修订预览状态
    end
```

## 参与模块

| 模块 | 角色 | 蓝图 |
|------|------|------|
| views/Schedule | UI 入口 (修订对话框) | - |
| stores/schedule | 状态管理 (预览 vs 正式) | [stores](../modules/stores.md) |
| api/client | HTTP (360s 超时) | [api-client](../modules/api-client.md) |
| handler/ScheduleHandler | 请求处理 | [api-handler](../modules/api-handler.md) |
| service/ScheduleService | 修订逻辑 (内存缓存) | [service](../modules/service.md) |
| service/AIService | LLM 调用 | [service](../modules/service.md) |
| repository/ScheduleRepo | 日程 CRUD | [repository](../modules/repository.md) |

## 关键设计

- **两阶段提交**：修订结果先缓存在 Service 内存中，用户确认后才持久化
- **全量替换**：应用修订时 DeleteAll + 批量 Create，非增量更新
- **长时间请求**：前端 API 超时 360 秒，适配 LLM 响应延迟

## 异常路径

| 场景 | 处理方式 |
|------|----------|
| AI 未配置 | Handler 返回 503 |
| LLM 返回无效 JSON | extractJSON 失败 → 500 |
| 修订过期 (后端重启) | 内存缓存丢失 → 提示用户重新修订 |
| 并发修订请求 | 后提交覆盖前一次 (Last Write Wins) |
| 空修订结果 | 返回空列表 → 前端显示"无需修改" |

## 关联

| 类型 | 链接 |
|------|------|
| 前置流程 | [AI Schedule Generation](ai-schedule-generation.md) |
| 关联模块 | [service](../modules/service.md), [ai](../modules/ai.md) |
| 关联 ADR | [ADR-0004](../decisions/adr-0004-openai-compatible-ai.md) |
