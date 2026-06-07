# TickTask Repository Profile

Generated: 2026-06-07 | Phase 0 Diagnostic

---

## Language Distribution

Source code only (excludes node_modules, dist, .git, .playwright-mcp).

| Language   | Files | Lines  | Code   | Comments | Blanks | Share (by code lines) |
|------------|-------|--------|--------|----------|--------|-----------------------|
| Go         | 43    | 10,897 | 8,567  | 538      | 1,792  | 38.4%                 |
| TypeScript | 38    | 8,082  | 6,655  | 112      | 1,315  | 29.8%                 |
| Vue (SFC)  | 18    | 7,861  | 6,820  | 65       | 976    | 30.6% (incl. CSS/HTML/JS blocks) |
| Shell      | 2     | 278    | 198    | 35       | 45     | 0.9%                  |
| YAML       | 1     | 18     | 15     | 0        | 3      | 0.1%                  |
| Makefile   | 1     | 60     | 38     | 11       | 11     | 0.2%                  |
| Markdown   | 9     | 231    | 3      | 162      | 66     | ~0% (prose)           |

**Primary languages**: Go (backend), TypeScript + Vue 3 (frontend).

---

## Root Modules (2)

| Path         | Type    | Files | ~Code Lines | Description                                      |
|--------------|---------|-------|-------------|--------------------------------------------------|
| `backend/`   | service | 43    | ~8,567      | Go REST API server with Gin framework, SQLite DB, WebSocket hub, AI scheduling |
| `frontend/`  | service | 56    | ~13,475     | Vue 3 SPA with Pinia stores, Element Plus UI, Vite build |

---

## Leaf Modules

### Backend (`backend/`)

| Path                              | Type     | Role                                                         |
|-----------------------------------|----------|--------------------------------------------------------------|
| `backend/cmd/server/`             | service  | Entry point — manual DI wiring (config -> DB -> repos -> services -> router) |
| `backend/internal/model/`         | library  | Domain models: Task, Session, Schedule, Setting, Analytics   |
| `backend/internal/repository/`    | library  | Data access layer (GORM/SQLite). Interfaces + private structs. 6 repos. |
| `backend/internal/service/`       | library  | Business logic: TaskService, TimerService, AIService, ScheduleService, AnalyticsService |
| `backend/internal/ai/`            | library  | AI client abstraction (OpenAI-compatible API) + prompt templates |
| `backend/internal/api/handler/`   | adapter  | HTTP handlers (thin: bind JSON -> call service -> return JSON). 8 handlers. |
| `backend/internal/api/middleware/`| adapter  | CORS middleware                                               |
| `backend/internal/websocket/`     | library  | WebSocket hub for real-time timer broadcasts                  |
| `backend/pkg/config/`             | library  | YAML config loader (server, database, CORS, AI settings)      |
| `backend/pkg/database/`           | library  | SQLite init + seed data via GORM                              |
| `backend/pkg/logger/`             | library  | Structured logger (slog-based)                                |

### Frontend (`frontend/src/`)

| Path                        | Type     | Role                                                         |
|-----------------------------|----------|--------------------------------------------------------------|
| `frontend/src/views/`       | adapter  | Page components: Dashboard, Tasks, Schedule, Timer, Analytics, Settings |
| `frontend/src/components/`  | adapter  | Reusable UI components grouped by domain: schedule/, tasks/, timer/ |
| `frontend/src/stores/`      | library  | Pinia stores: task, timer, schedule, ai, app. Call API client directly. |
| `frontend/src/api/`         | library  | Axios-based API client (singleton)                            |
| `frontend/src/utils/`       | library  | WebSocket client (singleton), time utilities                  |
| `frontend/src/router/`      | library  | Vue Router configuration                                      |
| `frontend/src/types/`       | library  | Barrel type definitions file (index.ts)                       |

### Other

| Path                | Type   | Role                                             |
|---------------------|--------|--------------------------------------------------|
| `scripts/`          | config | Build and start shell scripts                    |
| `config/`           | config | Auto-schedule skill inputs: todo.json, habit.md  |
| `docs/skills/`      | config | Claude Code skill definitions                    |

---

## Build System

| Module      | Build Tool  | Build Command                                               | Output                       |
|-------------|-------------|-------------------------------------------------------------|------------------------------|
| Backend     | Go Modules  | `cd backend && CGO_ENABLED=1 go build -o bin/ticktask-server cmd/server/main.go` | `backend/bin/ticktask-server` |
| Frontend    | Vite + vue-tsc | `cd frontend && npm run build` (= `vue-tsc && vite build`) | `frontend/dist/`            |
| Full stack  | Makefile    | `make build` or `./scripts/build.sh all`                    | Both outputs above            |

**Development mode**: `make dev` or `./scripts/start.sh dev`
- Backend: `go run cmd/server/main.go` (port 8080)
- Frontend: `npm run dev` via Vite (port 5173, HMR)
- Vite proxies `/api` -> `:8080` and `/ws` -> `ws://:8080`

**Production mode**: `./scripts/start.sh prod`
- Serves frontend dist from backend (Go binary + static files)

### Go Version & Dependencies (key)

- **Go 1.21**
- `github.com/gin-gonic/gin v1.10.0` — HTTP framework
- `gorm.io/gorm v1.25.12` + `gorm.io/driver/sqlite v1.5.7` — ORM + SQLite driver
- `github.com/gorilla/websocket v1.5.3` — WebSocket
- `github.com/google/uuid v1.6.0` — UUID generation
- `gopkg.in/yaml.v3 v3.0.1` — YAML config parsing

### Frontend Dependencies (key)

- **Vue 3.5**, **Pinia 2.2**, **Vue Router 4.4**
- **Element Plus 2.8** — UI component library
- **Axios 1.7** — HTTP client
- **Vite 5.4** — Build tool
- **TypeScript 5.6** (strict mode enabled)
- **Vitest 2.1** + `@vue/test-utils` — Testing
- **vue-tsc 2.1** — Type checking

---

## Module Dependencies (Internal)

```
backend/cmd/server/main.go
  -> backend/internal/api          (router + handlers)
       -> backend/internal/api/handler
       -> backend/internal/api/middleware
       -> backend/internal/service
            -> backend/internal/repository  (interfaces)
            -> backend/internal/ai
       -> backend/internal/websocket
  -> backend/internal/repository   (concrete repos)
  -> backend/internal/service
  -> backend/internal/websocket
  -> backend/pkg/config
  -> backend/pkg/database
  -> backend/pkg/logger

frontend/src/views/*
  -> frontend/src/components/*
  -> frontend/src/stores/*
       -> frontend/src/api/client
  -> frontend/src/router

frontend/src/stores/*
  -> frontend/src/api/client       (direct API calls)
  -> frontend/src/utils/websocket

frontend/src/App.vue
  -> frontend/src/utils/websocket   (singleton WS connection)
  -> frontend/src/router
```

**Topological order**: pkg/* -> internal/model -> internal/repository -> internal/ai -> internal/service -> internal/websocket -> internal/api -> cmd/server

**Entry point**: `backend/cmd/server/main.go` (manual DI wiring)

**No circular dependencies detected.**

---

## Test Frameworks & Commands

### Backend (Go)

- **Framework**: Go standard `testing` package
- **Test files**: 10 (`*_test.go`) across handler/ and service/
- **Mocks**: `backend/internal/api/handler/mocks_test.go` (shared mock repos using in-memory maps)
- **Commands**:
  - `cd backend && go test ./...` — All tests
  - `cd backend && go test -v ./internal/service/...` — Service tests only
  - `cd backend && go test -v ./internal/api/handler/...` — Handler tests only
  - `make test` — Runs backend tests + frontend build check

### Frontend (Vue/TS)

- **Framework**: Vitest + @vue/test-utils + jsdom
- **Config**: `frontend/vitest.config.ts` (globals: true, environment: jsdom)
- **Test files**: 26 (`*.test.ts` + `*.spec.ts`) covering stores, components, views, utils, router, api client
- **Commands**:
  - `cd frontend && npx vitest run` — All tests (single run)
  - `cd frontend && npx vitest run src/stores/ai.spec.ts` — Single test file
  - `cd frontend && npx vitest` — Watch mode
  - `cd frontend && npm run test:run` — Alias for vitest run
- **Type checking**: `cd frontend && npx vue-tsc --noEmit`

### Coverage

- Backend: `backend/coverage.out` exists (Go coverage output)
- Frontend: `@vitest/coverage-v8` installed; `frontend/coverage/` directory exists

---

## Code Style & Lint Tools

**Not detected** (no project-level linting/formatting configuration found):
- No `.eslintrc*`, `.prettierrc*`, `.editorconfig` at project root or in frontend/
- No `.golangci.yml` in backend/
- No `.clang-format`, `.clang-tidy`

**TypeScript strict mode** is enabled in `tsconfig.json`:
- `strict: true`, `noUnusedLocals: true`, `noUnusedParameters: true`, `noFallthroughCasesInSwitch: true`

---

## CI/CD

**Not detected.** No CI configuration files found:
- No `.github/workflows/`
- No `.gitlab-ci.yml`
- No `Jenkinsfile`
- No `azure-pipelines.yml`
- No `.circleci/config.yml`

[? 待确认] CI may be handled externally or not yet set up.

---

## Commit Convention

**Conventional Commits** (predominant).

Recent examples:
```
feat: add schedule revision UI (Story 1.2)
feat: add schedule revision backend API (Story 1.1)
feat: add recurring tasks support with preferred time slots and AI scheduling integration
fix: prevent null-events crash when applying schedule revision
refactor: redesign frontend with refined editorial minimalism aesthetic
docs: add bug fix workflow report for null-events crash
```

Prefixes used: `feat:`, `fix:`, `refactor:`, `docs:`

---

## Branch Strategy

- **main** — Stable/production branch
- **evolve/ai-scheduling-enhancements** — Current development branch (AI scheduling features)

---

## Additional Notes

- **Database**: SQLite (`backend/data/ticktask.db`) — single-writer, not stress-tested for concurrency
- **AI Integration**: OpenAI-compatible API (configurable base URL), used for task classification and schedule generation
- **WebSocket**: Real-time timer broadcasts via gorilla/websocket, auto-reconnect on frontend
- **Design System**: Custom CSS properties in `App.vue` — warm paper aesthetic, Playfair Display + DM Sans fonts
- **Skills**: `docs/skills/auto-schedule/` — Chinese-language daily schedule generation skill
- **Config**: AI API key stored in `backend/configs/config.yaml` (plaintext, never committed)
