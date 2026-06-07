# ADR-0001: SQLite as Primary Database

**Status**: Accepted
**Date**: 2026-06-07
**Context**: [repository](../modules/repository.md) | [Timer Session](../flows/timer-session.md)

## Context

TickTask 是一款个人时间管理工具，需要持久化任务、计时会话、日程、设置和分析数据。需要选择数据库方案。

## Decision

使用 **SQLite** 作为唯一数据库，通过 GORM ORM 访问。数据库文件存储在 `backend/data/ticktask.db`。

## Alternatives Considered

| 方案 | 优点 | 缺点 |
|------|------|------|
| **SQLite** (选中) | 零配置、单文件部署、无需数据库服务器、GORM 原生支持 | 单写入者、不适合高并发写入、无网络访问 |
| PostgreSQL | 成熟稳定、支持并发写入、丰富数据类型 | 需要独立服务、配置复杂、过度设计 |
| MySQL | 广泛使用、社区支持大 | 需要独立服务、配置复杂 |
| LevelDB/BoltDB | 嵌入式、高性能 KV 存储 | 无 SQL 查询能力、GORM 不支持 |

## POS (正面影响)

- **部署简单**：单文件 `ticktask.db`，无需安装数据库服务
- **开发体验**：GORM AutoMigrate 非破坏性迁移，开箱即用
- **性能足够**：单用户场景，读多写少，SQLite 性能绰绰有余
- **备份便捷**：复制一个文件即可完成备份
- **CGO_ENABLED=1** 构建启用 SQLite 驱动

## NEG (负面影响)

- **单写入者限制**：计时器 goroutine 每秒写入 + 其他 CRUD 可能有锁竞争
- **并发压力未验证**：Timer Session 期间的并发写入行为未经压测
- **无网络访问**：不支持远程数据库连接（仅限本机）

## IMP (实施要点)

- GORM AutoMigrate 用于 schema 演进（只增不删）
- `pkg/database/Init()` 初始化连接 + `SeedInitialData()` 幂等种子数据
- 5 张表：tasks, pomodoro_sessions, schedules, settings, daily_stats

## REF (参考)

- [repository 蓝图](../modules/repository.md)
- [model 蓝图](../modules/model.md)
- GORM SQLite driver: `gorm.io/driver/sqlite v1.5.7`
