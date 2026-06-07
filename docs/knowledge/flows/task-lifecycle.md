# Flow: Task Lifecycle (任务生命周期)

> 任务从创建到完成/取消的完整生命周期。

## Sequence Diagram

```mermaid
sequenceDiagram
    actor User
    participant View as views/Tasks.vue
    participant Store as useTaskStore
    participant API as api/client
    participant Handler as TaskHandler
    participant Svc as TaskService
    participant Repo as TaskRepository
    participant DB as SQLite

    User->>View: 创建/编辑/移动任务
    View->>Store: createTask(data) / updateTask(id, data)
    Store->>API: POST /api/tasks / PUT /api/tasks/:id
    API->>Handler: ShouldBindJSON → CreateTaskRequest
    Handler->>Svc: CreateTask(req) / UpdateTask(id, req)
    Svc->>Repo: Create(task) / Update(task)
    Repo->>DB: GORM Create/Save
    DB-->>Repo: OK
    Repo-->>Svc: task / nil
    Svc-->>Handler: task / error
    Handler-->>API: JSON 201/200 / error
    API-->>Store: response
    Store->>Store: 更新 local state
    Store-->>View: reactive update

    Note over User,View: 象限移动
    User->>View: 拖拽任务到新象限
    View->>Store: moveTask(id, quadrant)
    Store->>API: PATCH /api/tasks/:id/move
    API->>Handler: MoveTask
    Handler->>Svc: MoveTask(id, targetQuadrant)
    Note over Svc: 自动计算 IsImportant/IsUrgent
    Svc->>Repo: Update(task)
    Svc->>Repo: 更新 analytics (如状态变更)
```

## 参与模块

| 模块 | 角色 | 蓝图 |
|------|------|------|
| views/Tasks | UI 入口 | - |
| stores/task | 状态管理 | [stores](../modules/stores.md) |
| api/client | HTTP 通信 | [api-client](../modules/api-client.md) |
| handler/TaskHandler | 请求处理 | [api-handler](../modules/api-handler.md) |
| service/TaskService | 业务逻辑 | [service](../modules/service.md) |
| repository/TaskRepo | 数据持久化 | [repository](../modules/repository.md) |

## 异常路径

| 场景 | 处理方式 |
|------|----------|
| 任务不存在 | Repo 返回 `ErrNotFound` → Handler 返回 404 |
| 验证失败 (空标题等) | `ShouldBindJSON` 失败 → Handler 返回 400 |
| 并发写冲突 | SQLite 单写入器，GORM 返回错误 → 500 |
| 网络中断 | Store catch error → `ElMessage.error` 提示用户 |

## 状态流转

```
todo → in_progress → completed
                   → cancelled
```
