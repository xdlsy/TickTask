# ARCHITECTURE

## Bird's-eye view

TickTask is a personal time-management application combining the Pomodoro Technique, Eisenhower Matrix prioritization, and AI-powered scheduling. Users create tasks classified by importance and urgency into four quadrants, run Pomodoro timers with real-time WebSocket state streaming, and optionally use an OpenAI-compatible LLM to classify tasks, rank priorities, and generate or revise daily schedules. The stack is a Go backend (Gin + GORM + SQLite + WebSocket) and a Vue 3 frontend (Pinia + Element Plus), communicating via REST and WebSocket with manual DI wiring in a single entry point.

## Code map

### `backend/cmd/server/main.go`
Application entry point. Manually wires all dependencies in order: loads YAML config, initializes SQLite via GORM, runs auto-migration and seed data, creates repositories, then services, then WebSocket hub, then passes everything to the router and starts the Gin HTTP server. No framework or DI container — all construction is explicit.

### `backend/internal/model/`
GORM entity models defining the database schema. Central types: `Task` (Eisenhower Matrix fields, status lifecycle, estimated time, recurring task fields, date range, tags), `PomodoroSession` (work/break types, timer state, interruption count), `Schedule` (calendar events with time ranges, AI adjustment tracking), `Setting` (key-value config with typed convenience structs), `DailyStats` (aggregated analytics), `WorkLog`/`WorkItem`/`WorkReport` (work-log domain with `WorkReportType` enum: weekly/monthly/halfyear/yearly). All models define `TableName()` and use GORM struct tags. Pointer fields for nullable values; tags stored as JSON strings. Work-log tables use string PKs (caller-generated UUIDs) and composite unique indexes (`uniqueIndex:idx_work_log_date` etc.).

### `backend/internal/repository/`
Data access via GORM. Each repository follows the interface + private struct pattern: exported interface, unexported GORM-backed implementation, constructor returning the interface. Six repos: `TaskRepository` (8 methods, `GetAllByQuadrant` guarantees all 4 quadrants present, `GetCompletedTasksInRange` for work-log context), `SessionRepository` (6 methods, active session lookup, `GetCompletedWorkByDateRange`), `SettingRepository` (6 methods, JSON marshal/unmarshal for typed settings), `AnalyticsRepository` (8 methods, atomic counter increments via `gorm.Expr`), `ScheduleRepository` (11 methods, time-range overlap queries, revision workflow support), `WorkLogRepository` (8 methods: CreateWorkLog, GetWorkLogByDate, GetWorkLogsInRange, UpsertWorkLog, ReplaceItems, CreateWorkReport, UpdateWorkReport, GetWorkReportByTypeAndPeriod, ListWorkReports; date-range queries use string-compare on YYYY-MM-DD). Shared `ErrNotFound` sentinel error.

### `backend/internal/service/`
Business logic orchestration. Services coordinate across repositories and the WebSocket hub. `TaskService` handles CRUD, partial updates with pointer fields, quadrant moves with auto flag calculation, and status transitions with analytics tracking. `TimerService` runs a goroutine-based countdown with 1-second ticker and typed WebSocket broadcasts, using channel-based lifecycle (`stopChan`, `pauseChan`, `resumeChan`). `AIService` lazily initializes the LLM client, strips markdown from JSON responses, and provides classification/scheduling/prioritization; `CallLLM(ctx, system, user)` concatenates prompts for use by other services. `ScheduleService` handles time validation, color mapping, AI schedule generation with fallback, revision workflow (revise/apply), and ICS import. `WorkLogService` orchestrates brain-dump AI structuring (fills missing dimensions with "（待补充）"), today-context aggregation from task/session repos, daily log save/update (POST-then-PUT-on-409), and period report generation with **layered invariant**: weekly reads items, monthly reads weeklies + orphan items, halfyear/yearly read only monthlies. `ConfigWriter` writes AI config to `config.yaml` for skill integration. DTO types (`CreateTaskRequest`, `UpdateTaskRequest`, `BrainDumpInput`, `SaveWorkLogInput`, `GenerateReportInput`, `ReportSummary`) defined alongside services.

### `backend/internal/service/work_log_calendar.go`
Pure-function period algorithm helpers (no struct state). `WeeklyRange`/`MonthlyRange`/`HalfYearRange`/`YearlyRange` return `[start, end)` time ranges (end exclusive). `WeeklyKey`/`MonthlyKey`/`HalfYearKey`/`YearlyKey` produce canonical period keys (`2026-W31`, `2026-07`, `2026-H1`, `2026`). `RangeForType`/`KeyForType` dispatch by `WorkReportType`. `DateRangeToYMD` converts ranges to YYYY-MM-DD strings with inclusive end. `MissingDays` returns comma-joined set difference. Table-driven tests cover ISO week boundaries, Sunday rollover, H1/H2 split.

### `backend/internal/api/`
HTTP transport layer. `router.go` registers all routes under `/api` with CORS middleware and SPA fallback for non-API routes. Handlers are thin: `ShouldBindJSON` to parse input, delegate to service, return JSON. Route groups: tasks, sessions, AI, settings, analytics, schedules, work-logs (6 endpoints), work-reports (3 endpoints). WebSocket at `GET /ws`. Handlers map service errors to HTTP status codes (400 validation, 404 not found, 409 duplicate work log/report, 500 internal, 502 AI failure, 503 AI not configured). Settings handler masks API key in responses. AI handler uses timeout wrappers (30s/60s deadlines). Work-log handler extracts `existing_work_log` payload on 409 to surface the conflicting record.

### `backend/internal/ai/`
OpenAI-compatible LLM integration. `LLMClient` interface with single `ChatCompletion(ctx, prompt)` method enables mock substitution. `OpenAIClient` implementation uses standard `net/http`. Chinese-language prompt templates in `prompts.go` explicitly request JSON-only output. `work_log_prompts.go` adds 7 templates (brain-dump structuring system/user + 4 report types) encoding the "（待补充）" non-fabrication rule for missing dimensions. Constructor `NewOpenAIClient` with sensible defaults (GPT-4o-mini).

### `backend/internal/websocket/`
Real-time hub for timer state streaming. `Hub` maintains a `map[*Client]bool` protected by `sync.RWMutex`. `Client` wraps a gorilla/websocket connection with a buffered send channel (capacity 256). Non-blocking send evicts slow clients. Typed broadcast methods for timer tick, session state, timer complete, task updated, terminal output, and terminal status. `readPump` drains inbound messages (no application-level inbound handling); `writePump` writes from channel. `WebSocketHandler` upgrades HTTP to WS with permissive origin check.

### `backend/pkg/config/`
YAML configuration loader. `Config` struct aggregates `ServerConfig`, `DatabaseConfig`, `CORSConfig`, `AIConfig`. `Load(path)` reads YAML and applies `TT_AI_API_KEY` environment variable override. `LoadDefault()` provides sensible defaults (port 8080, GPT-4o-mini, 30s timeout).

### `backend/pkg/database/`
SQLite initialization. `Init(path)` opens SQLite, runs GORM auto-migration for all 5 models, returns DB handle. `SeedInitialData` is idempotent: creates default pomodoro and AI settings plus today's stats row, skipping if data already exists. AutoMigrate is non-destructive (adds columns/tables, never drops).

### `backend/pkg/logger/`
Global structured logger using Go's `log/slog`. Package-level `Logger` variable initialized via `init()` at info level with text handler. `Init(mode)` reconfigures to debug level when server mode is "debug".

### `frontend/src/views/`
Page-level components: Dashboard, Tasks, Timer, Schedule, Analytics, WorkLog, Settings. Each view orchestrates Pinia stores and composes feature components. Store actions called in `onMounted` or via watchers. Views are the only place that directly import and coordinate multiple stores. Lazy-loaded via Vue Router dynamic imports for code splitting. `WorkLog.vue` is the work-log page: left Timeline + right detail area (TodayContextCard + BrainDumpInput + WorkItemList editor, or ReportDetail when a report node is selected), with client-side period-key computation and 409 → ElMessageBox.confirm for force-overwrite.

### `frontend/src/components/`
Feature-domain components: `tasks/` (QuadrantView with drag-and-drop, ListView, TaskCard with quadrant color-coding, TaskForm dialog), `timer/` (TimerDisplay with circular progress, TimerControls with session type selector), `schedule/` (DayView, WeekView, MonthView with time-slot grid, EventForm), `work-log/` (TodayContextCard, BrainDumpInput, WorkItemEditor, WorkItemList, Timeline, ReportActions, ReportDetail). All use `<script setup lang="ts">`, receive data from stores, emit events upward. Element Plus components used throughout. Scoped CSS with design tokens from App.vue global styles. WorkItemEditor renders the 4-dimensional form (content/problem_solved/result/impact); Timeline highlights active selection and shows the "今" badge for today.

### `frontend/src/stores/`
Pinia state management (Composition API `defineStore`). Six stores: `useTaskStore` (dual flat + grouped-by-quadrant representation), `useTimerStore` (dual REST + WebSocket update path, registers WebSocket listeners via `setupWebSocket`), `useScheduleStore` (calendar CRUD, view mode switching, date navigation), `useAiStore` (classification/scheduling/prioritization, config status check on mount), `useAppStore` (UI state, notification queue), `useWorkLogStore` (logs/currentLog/todayContext/reports/currentReport/selected with 9 actions; `saveWorkLog` POST then PUT on 409; `selectNode` discriminated union drives right-pane view). Async actions follow `try/catch/finally` with `loading` flags. Stores call API client directly.

### `frontend/src/api/`
Single Axios client with `baseURL: '/api'`, 60s default timeout. Typed API methods organized by domain: tasks (7), timer (5), AI (7), settings (3), analytics (3), schedule (9), work-log (9: getTodayContext, structureBrainDump @ 120s, listWorkLogs, getWorkLog, createWorkLog, updateWorkLog, generateWorkReport @ 180s, listWorkReports, getWorkReport). Long-running AI endpoints override timeout to 360s. Vite dev server proxies `/api` and `/ws` to backend. Co-located test file.

### `frontend/src/types/`
Single barrel file (`index.ts`) — the only place for shared TypeScript types. Contains: task domain types (`Task`, `Quadrant`, `TaskStatus`, `QUADRANT_INFO`), timer types, WebSocket discriminated union (`WSMessage`), AI types, analytics types, schedule types with DTOs, work-log types (`WorkItem`, `WorkLog`, `WorkReportType`, `WorkReport`, `TodayContext`, `StructuredWorkLog`, `SaveWorkLogInput`, `ReportSummary`). No runtime code.

### `frontend/src/utils/`
Two utilities: `websocket.ts` (pub/sub `WebSocketClient` singleton with type-based message routing and exponential backoff reconnection, exported as `wsClient`) and `time.ts` (Chinese-locale formatting functions: `formatTime`, `formatDuration`, `formatDateTime`, `getRemainingTime`).

### `frontend/src/router/`
Vue Router with HTML5 history mode. Seven lazy-loaded routes plus root redirect to `/dashboard`. Route names match PascalCase component names. No route guards or nested routes.

## Cross-cutting concerns

### Error handling [~ inferred]
- **Backend:** Repository errors bubble up through service to handler. Handlers map to HTTP status: 400 (validation), 404 (not found), 500 (internal), 503 (AI not configured). `gin.Recovery()` catches panics. Timer goroutine uses channel-based lifecycle with `defer ticker.Stop()`. `context.WithTimeout` for AI calls.
- **Frontend:** Stores use `try/catch/finally` with `loading` flags. Axios interceptor logs errors to console. `ElMessage.error` for user-facing errors. No global error boundary.

### Observability [~ inferred]
- **Backend:** Structured logging via `pkg/logger` (slog-based, text format, stdout). Log level configurable via server mode (debug/release). No metrics, tracing, or APM libraries.
- **Frontend:** Console-based logging only. No error tracking service.
<!-- HUMAN_REVIEW: add logging format standards, metrics dashboards, monitoring approach, alerting -->

### Testing strategy [✓ auto]
- **Backend:** Standard `testing` package, no external assertion/mocking libraries. Manual mock repositories using in-memory `map[string]*Model` implementing full interfaces. Shared mocks in `handler/mocks_test.go`. Table-driven test patterns for combinatorial logic. 10 test files across handler and service layers. Co-located `*_test.go`.
- **Frontend:** Vitest 2.1 + `@vue/test-utils` + jsdom. `vi.mock()` for stores, API client, router, WebSocket, Element Plus. Pinia isolation via `setActivePinia(createPinia())` in each `beforeEach`. Both `.test.ts` and `.spec.ts` naming. 26 test files covering stores, components, views, router, API client, and utilities. `vue-tsc --noEmit` for type checking.

### Build & deploy [✓ auto]
- **Dev:** `make dev` starts backend on `:8080` (Go) and frontend on `:5173` (Vite HMR). Vite proxies `/api` and `/ws` to backend.
- **Prod:** `make prod` builds frontend to `dist/`, then backend serves static files on `:8080` with SPA fallback.
- **Build:** `make build` compiles Go binary (`CGO_ENABLED=1`) and runs `vue-tsc` + Vite build.
- No CI/CD pipeline detected. [? 待审核]
<!-- HUMAN_REVIEW: confirm CI/CD status and deployment process -->

### Security [? review]
<!-- HUMAN_REVIEW: add authentication/authorization design, threat model, security review process -->
- AI API key stored in `backend/configs/config.yaml` (plaintext YAML), never committed to git. Environment variable `TT_AI_API_KEY` can override.
- No authentication layer present — application assumes single-user local deployment.
- Settings handler masks API key in responses (first 4 + last 4 characters).
- CORS middleware with config-based origin allowlist. WebSocket `CheckOrigin` returns true.
- SQLite file (`backend/data/ticktask.db`) has no encryption.

### Performance [? review]
<!-- HUMAN_REVIEW: add performance SLAs, profiling methodology, known bottlenecks, load testing results -->
- Timer goroutine broadcasts at 1-second tick rate to all WebSocket clients.
- AI endpoints use `context.WithTimeout` (30s/60s backend, 360s frontend for schedule generation).
- SQLite single-writer: concurrent write behavior under timer sessions not stress-tested.
- Frontend uses lazy-loaded routes for code splitting. No virtual scrolling for large task lists.
- `GetByTimeRange` in ScheduleRepository uses 3-condition OR query — performance with large datasets unknown.

### Internationalization [~ inferred]
- Backend AI prompts are Chinese-language, requesting JSON-only output.
- Frontend time formatting uses `zh-CN` locale.
- No i18n framework; UI strings are hardcoded in Chinese and English.
<!-- HUMAN_REVIEW: confirm i18n strategy if multi-language support is planned -->
