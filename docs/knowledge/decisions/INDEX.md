# Architecture Decision Records Index

架构决策记录（ADR），使用 MADR 格式 + POS/NEG/ALT/IMP/REF 编码要点。

## ADR 清单

| 编号 | 决策 | 状态 | 日期 | 关联模块 | 关联流程 |
|------|------|------|------|----------|----------|
| [ADR-0001](adr-0001-sqlite-as-db.md) | SQLite 作为主数据库 | Accepted | 2026-06-07 | repository, model | Timer Session |
| [ADR-0002](adr-0002-manual-di.md) | 手动依赖注入 | Accepted | 2026-06-07 | service, api/handler | - |
| [ADR-0003](adr-0003-websocket-realtime.md) | WebSocket 实时推送 | Accepted | 2026-06-07 | websocket, stores | Timer Session, WebSocket Real-time |
| [ADR-0004](adr-0004-openai-compatible-ai.md) | OpenAI 兼容 AI 集成 | Accepted | 2026-06-07 | ai, service | AI Schedule Generation, Schedule Revision |

## 按主题分类

**数据层**
- [ADR-0001: SQLite as DB](adr-0001-sqlite-as-db.md)

**架构模式**
- [ADR-0002: Manual DI](adr-0002-manual-di.md)

**通信**
- [ADR-0003: WebSocket Real-time](adr-0003-websocket-realtime.md)

**AI/ML 集成**
- [ADR-0004: OpenAI-Compatible AI](adr-0004-openai-compatible-ai.md)

## 导航

- [知识库总览](../README.md)
- [模块蓝图](../modules/) — 按模块查阅
- [流程蓝图](../flows/) — 按流程查阅
