# model

## Responsibility [~ inferred]

GORM entity models defining the database schema and domain types for the TickTask application. All models use GORM struct tags for column mapping and define explicit `TableName()` methods.

Key types:
- `Task` — central entity with Eisenhower Matrix fields (`Quadrant`, `IsImportant`, `IsUrgent`), status lifecycle, estimated time, deadline, recurring task fields (`IsRecurring`, `RecurrencePattern`, `RecurrenceDay`, `PreferredStartTime`, `PreferredEndTime`), date range (`StartDate`, `DueDate`), and tags
- `PomodoroSession` — timer session with type (work/short_break/long_break), status, planned/actual duration, and interruption count
- `Schedule` — calendar event with time range, type (task/pomodoro/break/custom), status, color, AI adjustment tracking (`AIAdjusted`, `AdjustmentType`)
- `Setting` — key-value configuration store with typed convenience structs (`PomodoroSettings`, `AISettings`)
- `DailyStats` — aggregated daily analytics (completed pomodoros, focus time, task counts)

## Conventions [~ inferred]

- PascalCase struct names, GORM tags for column mapping (`gorm:"primaryKey;size:36"`)
- Enum-like types as custom types with string constants (e.g., `Quadrant1`, `StatusTodo`, `SessionWork`)
- Pointer fields for nullable timestamps and IDs (`*time.Time`, `*string`)
- JSON serialization for complex sub-fields (tags stored as JSON string, settings serialized to JSON)
- `TableName()` method on every model for explicit table naming
- Factory functions for default settings (`DefaultPomodoroSettings()`, `DefaultAISettings()`)

## Dependencies [✓ auto]

- Depends on: `gorm.io/gorm` (struct tags only, no GORM logic in model package)
- Depended on by: `repository/`, `service/`, `api/handler/`
