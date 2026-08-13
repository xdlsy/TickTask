# database

## Responsibility [~ inferred]

SQLite database initialization, schema auto-migration via GORM, and first-run seed data insertion. Provides a `*gorm.DB` instance for the entire application.

Key functions:
- `Init(path string) (*gorm.DB, error)` — opens SQLite, runs auto-migration, returns DB handle
- `AutoMigrate(db *gorm.DB) error` — migrates all 5 models (Task, PomodoroSession, Setting, DailyStats, Schedule)
- `SeedInitialData(db *gorm.DB) error` — idempotent seed: creates default pomodoro/AI settings and today's stats row (skips if data exists)

## Conventions [~ inferred]

- GORM logger set to `Silent` mode (no SQL logging in production)
- `AutoMigrate` is non-destructive — adds columns/tables but never drops
- Seed check uses `Count` on settings table — assumes empty DB on first run
- Crosses internal boundary: imports `internal/model` (acceptable for pkg/database at DI level)

## Dependencies [✓ auto]

- Depends on: `internal/model` (all 5 GORM entities), `gorm.io/gorm`, `github.com/glebarez/sqlite`（纯 Go，CGO-free）
- Depended on by: `cmd/server/main.go` (DB init + seed at startup)
