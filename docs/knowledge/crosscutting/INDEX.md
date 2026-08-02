# Crosscutting Concerns Index

跨模块的横切关注点索引。当前仅为索引级别，待后续深度文章生成时展开。

## 关注点清单

| 关注点 | 涉及模块 | 当前状态 | 说明 |
|--------|----------|----------|------|
| **Error Handling** | 全部后端模块 | [? 待审核] | Repo 错误冒泡 → Service → Handler HTTP 状态码映射。前端 try/catch/finally + ElMessage.error |
| **Observability** | 全部后端模块 | [? 待审核] | slog 结构化日志，配置 mode 控制级别。无 metrics/tracing/APM。前端仅 console.log |
| **Configuration** | pkg/config, configs/ | 已覆盖 | YAML 配置 + 环境变量覆盖。详见 ADR-0004 |
| **Security** | api/middleware, handler | [? 待审核] | 无认证层（单用户本地部署）。API Key 脱敏响应。CORS 配置化。详见 ARCHITECTURE.md |
| **Testing** | handler, service, stores | 已覆盖 | Go 标准 testing + 手动 mock。Vitest + jsdom。详见 ARCHITECTURE.md Testing 章节 |
| **Build & Deploy** | Makefile, scripts/ | 已覆盖 | `make dev`/`make prod`/`make build`。无 CI/CD [? 待审核] |
| **Performance** | websocket, repository | [? 待审核] | 1s tick 广播。SQLite 单写入者。无虚拟滚动。GetByTimeRange OR 查询性能未知 |
| **i18n** | ai/prompts, utils/time | [? 待审核] | AI prompt 中文。时间格式 zh-CN。无 i18n 框架 |

## 参考

- [ARCHITECTURE.md](../../ARCHITECTURE.md) — Cross-cutting concerns 章节
- [ADR-0001](../decisions/adr-0001-sqlite-as-db.md) — 数据库选型影响并发性能
- [ADR-0003](../decisions/adr-0003-websocket-realtime.md) — WebSocket 连接管理

## 导航

- [知识库总览](../README.md)
