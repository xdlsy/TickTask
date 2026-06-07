# types

## Responsibility [✓ auto]

Single barrel file (`index.ts`) containing ALL shared TypeScript type definitions for the frontend application. This is the single source of truth for interfaces, type aliases, and constants used across stores, components, and views.

Key type groups:
- **Task domain**: `Task`, `Quadrant`, `TaskStatus`, `QuadrantInfo`, `QUADRANT_INFO` (constant)
- **Timer domain**: `PomodoroSession`, `SessionType`, `SessionStatus`, `PomodoroSettings`
- **WebSocket**: `WSMessage` (discriminated union of 5 message types), `WSMessageType`
- **AI domain**: `ClassificationResult`, `PrioritySuggestion`, `AIStatus`, `AISettings`, `DailyInsights`
- **Analytics**: `DailySummary`, `TrendData`, `DistributionStats`, `TaskTimeStats`
- **Schedule**: `ScheduleEvent`, `ScheduleType`, `ScheduleStatus`, `CreateScheduleDTO`, `UpdateScheduleDTO`, `MoveScheduleDTO`, `ReviseResponse`, `RevisionChange`

## Conventions [✓ auto]

- ALL shared types must be added here — never create `.ts` type files alongside components
- Uses `interface` for object shapes, `type` for unions and aliases
- Discriminated union pattern for WebSocket messages (`type` field as discriminant)
- `QUADRANT_INFO` constant exported alongside the `Quadrant` type
- Nullable fields use `string | null` (not `string | undefined`)
- DTOs for API requests: `CreateScheduleDTO`, `UpdateScheduleDTO`, `MoveScheduleDTO`

## Dependencies [✓ auto]

- Depends on: nothing (pure type definitions, no runtime imports)
- Depended on by: every other frontend module (stores, components, views, api, utils)
