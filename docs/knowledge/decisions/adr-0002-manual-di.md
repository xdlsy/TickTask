# ADR-0002: Manual Dependency Injection

**Status**: Accepted
**Date**: 2026-06-07
**Context**: [service](../modules/service.md) | [api-handler](../modules/api-handler.md) | `cmd/server/main.go`

## Context

TickTask 后端需要一种依赖注入方式来组装各层（Repository → Service → Handler）。需要选择 DI 策略。

## Decision

使用**手动依赖注入**（Manual DI），在 `cmd/server/main.go` 中显式构造所有依赖并按拓扑顺序传递。不使用 DI 框架（如 Wire、Dig）。

## Alternatives Considered

| 方案 | 优点 | 缺点 |
|------|------|------|
| **Manual DI** (选中) | 零依赖、依赖关系透明、易于调试 | 大量构造代码集中在 main.go |
| Google Wire | 编译时注入、类型安全 | 引入额外工具、学习成本 |
| Uber Dig | 运行时注入、反射实现 | 运行时错误、调试困难 |
| Gin 框架注入 | 与 HTTP 框架集成 | 仅限 HTTP 层，Service/Repository 需另行处理 |

## POS (正面影响)

- **零额外依赖**：不需要 DI 框架或代码生成工具
- **依赖关系一目了然**：main.go 按拓扑顺序构造，清晰的构造链路
- **易于调试**：所有构造逻辑集中，设置断点即可
- **编译时检查**：Go 类型系统保证依赖完整性

## NEG (负面影响)

- **构造代码冗长**：main.go 包含所有 new + pass 逻辑
- **扩展成本**：新增模块需手动更新 main.go 的构造链
- **无自动生命周期管理**：需手动管理 goroutine 启停

## IMP (实施要点)

构造拓扑顺序：
```
config → database → repositories → services → websocket hub → router
```
- 每层构造函数 `New*()` 返回接口类型（Repository）或具体类型（Service）
- Handler 在 router.go 中按需构造（非单例）
- WebSocket Hub 通过 `go hub.Run()` 启动后台 goroutine

## REF (参考)

- [service 蓝图](../modules/service.md)
- [api-handler 蓝图](../modules/api-handler.md)
- `backend/cmd/server/main.go` — DI 入口
