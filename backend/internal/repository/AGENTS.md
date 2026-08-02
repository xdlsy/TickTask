# repository

## Responsibility [~ inferred]

Data access layer implementing CRUD operations against SQLite via GORM. Each repository follows the **interface + private struct** pattern: exported interface defines the contract, unexported struct holds the `*gorm.DB` reference, constructor returns the interface type.

Repositories:
- `TaskRepository` — 8 methods including `GetAllByQuadrant()` that guarantees all 4 quadrants present in the map
- `SessionRepository` — 6 methods with active session lookup (`GetActive()` queries by `status IN (running, paused)`)
- `SettingRepository` — 6 methods; typed settings stored as JSON strings in key-value store
- `AnalyticsRepository` — 8 methods; uses `gorm.Expr` for atomic counter increments
- `ScheduleRepository` — 11 methods; `GetByTimeRange()` with 3-condition OR query for overlapping intervals, `DeleteTaskSchedulesByDateRange()` for revision workflow, `DeleteAll()` for full reset

## Conventions [~ inferred]

- All repositories return `repository.ErrNotFound` sentinel for missing records
- GORM errors propagated directly (no wrapping)
- Constructor always `New*Repository(db *gorm.DB) *Repository` returning the interface type
- Atomic updates use `gorm.Expr("column + ?", n)` for counter operations
- Settings repository handles JSON marshal/unmarshal internally (typed API, string storage)

## Dependencies [✓ auto]

- Depends on: `model/`, `gorm.io/gorm`
- Depended on by: `service/`, `cmd/server/main.go` (DI wiring)
