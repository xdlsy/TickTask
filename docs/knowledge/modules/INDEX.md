# Module Blueprints Index

C4 Level 3 (Component) 蓝图，展示每个核心模块的内部架构、对外接口和依赖关系。

## 模块清单

| 模块 | 蓝图 | 职责 | 关联流程数 |
|------|------|------|-----------|
| service | [service.md](service.md) | 业务逻辑编排：任务、计时器、AI、日程、分析 | 3 |
| ai | [ai.md](ai.md) | OpenAI 兼容 LLM 客户端抽象 | 1 |
| repository | [repository.md](repository.md) | GORM/SQLite 数据访问层（5 个仓储） | 1 |
| api/handler | [api-handler.md](api-handler.md) | Gin HTTP 传输层（6 个 Handler） | 2 |
| websocket | [websocket.md](websocket.md) | gorilla/websocket Hub+Client 实时通信 | 2 |
| stores | [stores.md](stores.md) | Pinia 状态管理（5 个 Store） | 3 |
| model | [model.md](model.md) | GORM 领域模型（5 个实体） | 1 |
| api/client | [api-client.md](api-client.md) | Axios HTTP 客户端单例 | - |

## 依赖拓扑

```
model ← repository ← ai ← service ← api/handler
                        ← websocket ← service
stores → api/client → (Backend API)
```

## 按领域浏览

**后端 (Go)**
- 数据层：[model](model.md) → [repository](repository.md)
- 业务层：[ai](ai.md) → [service](service.md)
- 传输层：[api/handler](api-handler.md)
- 实时层：[websocket](websocket.md)

**前端 (Vue/TS)**
- 数据层：[api/client](api-client.md)
- 状态层：[stores](stores.md)

## 导航

- [知识库总览](../README.md)
- [流程蓝图](../flows/) — 按业务流程查阅
- [架构决策](../decisions/) — 按决策查阅
