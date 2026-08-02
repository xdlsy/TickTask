# Module: repository (数据访问层)

> 通过 GORM 实现的 SQLite 数据访问层，每个仓储遵循 interface + private struct 模式。

## Component Diagram

```mermaid
graph TB
    subgraph "repository/"
        TR["TaskRepository<br/>8 methods<br/>GetAllByQuadrant 保证 4 象限"]
        SeR["SessionRepository<br/>6 methods<br/>Active Session 查询"]
        SrR["SettingRepository<br/>6 methods<br/>JSON 序列化类型化设置"]
        AR["AnalyticsRepository<br/>8 methods<br/>gorm.Expr 原子计数"]
        SchR["ScheduleRepository<br/>11 methods<br/>时间区间重叠查询"]
        ERR["ErrNotFound<br/>哨兵错误"]
    end

    DB[("SQLite<br/>via GORM")]

    TR --> DB
    SeR --> DB
    SrR --> DB
    AR --> DB
    SchR --> DB

    style TR fill:#B8452C,color:#fff
    style SeR fill:#D4A843,color:#1C1B1A
    style SrR fill:#4A90D9,color:#fff
    style AR fill:#6E6A65,color:#fff
    style SchR fill:#4A90D9,color:#fff
```

## 对外接口

| 仓储 | 关键方法 | 说明 |
|------|----------|------|
| TaskRepository | GetAll, GetAllByQuadrant, GetByID, Create, Update, Delete, Move | `GetAllByQuadrant` 保证返回全部 4 个象限 |
| SessionRepository | GetActive, GetRecent, Create, Update, Delete | `GetActive` 查询 `status IN (running, paused)` |
| SettingRepository | GetByKey, GetAll, Upsert, Delete | JSON marshal/unmarshal 内部处理 |
| AnalyticsRepository | GetDailyStats, IncrementCompleted, IncrementFocusTime | `gorm.Expr("column + ?", n)` 原子更新 |
| ScheduleRepository | GetByTimeRange, Create, Update, Delete, DeleteByDateRange | 3 条件 OR 查询处理时间区间重叠 |

## 关键设计

- 所有构造函数返回**接口类型**，不暴露具体实现
- 统一 `ErrNotFound` 哨兵错误，上层通过 `errors.Is()` 判断
- GORM 错误直接传播（不 wrapping）
- 测试中使用 in-memory `map[string]*Model` 实现 mock 接口

## 关联

| 类型 | 链接 |
|------|------|
| 依赖 | [model](model.md), GORM |
| 消费模块 | [service](service.md) |
| 关联 ADR | [ADR-0001](../decisions/adr-0001-sqlite-as-db.md) |
