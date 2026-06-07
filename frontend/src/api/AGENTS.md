# api

## Responsibility [✓ auto]

Axios-based HTTP client singleton providing typed API methods for all backend endpoints. Configured with `/api` base URL (proxied to `:8080` in dev), 60s default timeout, and JSON content type.

API method groups:
- **Tasks** (6): `getTasks`, `getTasksByQuadrant`, `getTask`, `createTask`, `updateTask`, `deleteTask`, `moveTask`
- **Timer** (5): `getActiveSession`, `getRecentSessions`, `getTodayTaskStats`, `createSession`, `controlSession`
- **AI** (7): `getAIStatus`, `classifyTask`, `classifyTasks`, `classifyTaskByText`, `generateSchedule`, `rescheduleAfterInterrupt`, `getPrioritySuggestions`, `getDailyInsights`
- **Settings** (3): `getSettings`, `updatePomodoroSettings`, `updateAISettings`
- **Analytics** (3): `getAnalyticsSummary`, `getAnalyticsTrend`, `getAnalyticsDistribution`
- **Schedule** (9): CRUD + `generateScheduleFromTasks` (360s timeout), `deleteAllSchedules`, `reviseSchedule` (360s), `applyRevision`

## Conventions [~ inferred]

- All methods return typed Axios responses via generics (e.g., `client.get<Task[]>('/tasks')`)
- Long-running AI endpoints override default timeout: `generateScheduleFromTasks` and `reviseSchedule` use 360s
- Request/response interceptors for logging (minimal — just console.error on failure)
- Co-located test: `client.test.ts`
- Stores call `api.*` methods directly — no intermediate service layer

## Dependencies [✓ auto]

- Depends on: `axios`, `@/types` (all response/request type imports)
- Depended on by: `stores/` (all stores import `api` object)
