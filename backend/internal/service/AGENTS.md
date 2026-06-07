# service

## Responsibility [~ inferred]

Business logic layer orchestrating operations across multiple repositories and the WebSocket hub. Services define DTO types for cross-boundary communication and encapsulate domain rules.

Services:
- `TaskService` — CRUD + partial update (pointer fields), quadrant move with auto flag calculation, status transition tracking (sets `CompletedAt`, increments analytics)
- `TimerService` — goroutine-based countdown with 1-second ticker, WebSocket broadcasts on tick/state/complete, unified session control (pause/resume/complete/abandon)
- `AIService` — lazy LLM client init (nil if no API key), JSON response parsing with markdown stripping, configurable timeouts
- `AnalyticsService` — summary/trend/distribution queries with date range filtering and gap-filling
- `ScheduleService` — time validation, auto task title fill, color mapping by type and quadrant, AI schedule generation with fallback, revision workflow (revise/apply), ICS import via `ics_parser.go`, experience validation helpers (`experience.go`)
- `ConfigWriter` — writes AI config to `config.yaml` for skill integration (`config_writer.go`)

## Conventions [~ inferred]

- Service structs hold injected dependencies as unexported fields
- Constructors: `New*Service(dep1, dep2, ...) *Service`
- DTO types defined alongside the service: `CreateTaskRequest`, `UpdateTaskRequest`, `CreateScheduleDTO`
- Business rules enforced in service (e.g., quadrant → IsImportant/IsUrgent mapping)
- Timer goroutine uses channel-based lifecycle (`stopChan`, `pauseChan`, `resumeChan`) with `defer ticker.Stop()`

## Dependencies [✓ auto]

- Depends on: `model/`, `repository/`, `ai/`, `websocket/`
- Depended on by: `api/handler/`, `cmd/server/main.go` (DI wiring)
