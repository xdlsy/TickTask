# TickTask

## Project Overview [~ inferred]

TickTask is a personal time-management tool combining the Pomodoro Technique, Eisenhower Matrix (four-quadrant prioritization), and AI-powered intelligent recommendations. Full-stack application with a Go backend (Gin + GORM + SQLite) and a Vue 3 + TypeScript + Element Plus frontend, communicating via REST and WebSocket.

Key features:
- Pomodoro timer with real-time WebSocket state streaming (1-second ticker goroutine)
- Four-quadrant (Eisenhower Matrix) task prioritization
- Task CRUD with status lifecycle (todo -> in_progress -> completed/cancelled)
- AI-powered task classification, priority ranking, and daily schedule generation (OpenAI-compatible API)
- Calendar schedule management with day/week/month views, schedule revision workflow
- Recurring tasks with preferred time slots and AI scheduling integration
- Usage analytics (focus time, completion rates, quadrant distribution)

**Tech stack**: Go 1.21 / Gin 1.10 / GORM 1.25 / SQLite | Vue 3.5 / Pinia 2.2 / Element Plus 2.8 / Vite 5.4 / TypeScript 5.6 (strict)

## Repository Structure [✓ auto]

```
TickTask/
├── backend/                    # Go backend (Gin + GORM + SQLite)
│   ├── cmd/server/main.go      # Entry point — manual DI wiring
│   ├── internal/
│   │   ├── model/              # Domain models: Task, Session, Schedule, Setting, Analytics
│   │   ├── repository/         # Data access (GORM/SQLite), interface + private struct, 5 repos + errors.go
│   │   ├── service/            # Business logic: Task, Timer, AI, Schedule, Analytics
│   │   ├── ai/                 # OpenAI-compatible client + prompt templates
│   │   ├── api/
│   │   │   ├── handler/        # Thin HTTP handlers (6), bind JSON -> service -> JSON
│   │   │   └── middleware/     # CORS middleware
│   │   └── websocket/          # WebSocket hub for real-time timer broadcasts
│   ├── pkg/                    # Shared: config (YAML), database (SQLite+seed), logger (slog)
│   └── configs/                # config.yaml — server, DB, CORS, AI settings
├── frontend/                   # Vue 3 + TypeScript + Element Plus SPA
│   ├── src/
│   │   ├── api/                # Axios HTTP client (singleton)
│   │   ├── components/         # Feature components: schedule/, tasks/, timer/
│   │   ├── views/              # Pages: Dashboard, Tasks, Schedule, Timer, Analytics, Settings
│   │   ├── stores/             # Pinia stores: task, timer, schedule, ai, app
│   │   ├── router/             # Vue Router (lazy-loaded, HTML5 history)
│   │   ├── types/              # Single barrel file: index.ts (ALL shared types)
│   │   └── utils/              # WebSocket client (singleton) + time utilities
│   └── vitest.config.ts
├── scripts/                    # Build (build.sh) and start (start.sh) shell scripts
├── config/                     # Auto-schedule skill: todo.json + habit.md
├── docs/skills/                # Claude Code skill definitions
└── Makefile                    # Top-level build/dev/prod/test targets
```

### Module Dependency Graph [✓ auto]

```
pkg/* -> internal/model -> internal/repository -> internal/ai -> internal/service -> internal/websocket -> internal/api -> cmd/server
```

No circular dependencies detected.

## Build & Test Commands [✓ auto]

```bash
# --- Full stack ---
make dev              # Dev: backend :8080 + frontend :5173 (HMR), Vite proxies /api + /ws
make prod             # Production: backend serves frontend dist on :8080
make build            # Build both (Go binary + Vite build)
make test             # Backend tests + frontend build check

# --- Backend (Go 1.21) ---
cd backend && go run cmd/server/main.go                     # Start backend only
cd backend && go test ./...                                 # All Go tests
cd backend && go test -v ./internal/service/...             # Service tests
cd backend && go test -v ./internal/api/handler/...         # Handler tests
cd backend && CGO_ENABLED=1 go build -o bin/ticktask-server cmd/server/main.go  # Build binary

# --- Frontend (Vue/TS) ---
cd frontend && npm install               # Install deps
cd frontend && npm run dev               # Dev server on :5173
cd frontend && npm run build             # vue-tsc + vite build -> dist/
cd frontend && npx vue-tsc --noEmit      # Type check only
cd frontend && npx vitest run            # All tests (single run)
cd frontend && npx vitest run src/stores/ai.spec.ts  # Single test file
cd frontend && npx vitest                # Watch mode
```

### Restart Rules (critical for dev flow) [✓ auto]

- **Backend (Go)**: Must restart after any Go code change. `go run cmd/server/main.go` or `make dev`.
- **Frontend (Vue/TS)**: HMR auto-updates on component/store/style changes. Only restart `npm run dev` if `vite.config.ts` changes or new deps added.
- **WebSocket**: Backend restart breaks WS connections; frontend auto-reconnects. Refresh frontend page before backend restart to clear stale state.
- **E2E testing AI features**: `lsof -ti:8080 | xargs kill -9` then `cd backend && go run cmd/server/main.go` then refresh frontend.

## Coding Style [~ inferred]

**Go:**
- Standard Go conventions: `gofmt`, PascalCase exports, camelCase unexported
- Package names: short, lowercase, single-word (`model`, `service`, `repository`, `handler`)
- File naming: `snake_case` with domain prefix (`task_repo.go`, `task_service.go`)
- Interfaces: exported PascalCase noun; implementations: unexported lowercase struct
- Constructors: `New*` prefix returning the interface type
- Handler DTOs: `*Input` suffix; service DTOs: `*Request`/`*DTO` suffix
- Module path: `ticktask`
- No linting tool configured (no `.golangci.yml`)

**TypeScript/Vue:**
- Vue SFC with `<script setup lang="ts">` (Composition API)
- Components: PascalCase `.vue` files; views are singular nouns
- Stores: `useXxxStore` naming, Composition API `defineStore`
- API methods: CRUD-prefix (`getTasks`, `createTask`, `updateTask`)
- CSS: kebab-case classes, design tokens as CSS custom properties in `App.vue`
- TypeScript: `strict: true`, `noUnusedLocals`, `noUnusedParameters`
- No ESLint/Prettier configured (strict TS compiler acts as baseline check)

## Testing Guidelines [✓ auto]

**Backend (Go):**
- Framework: standard `testing` package (no testify/gomock)
- Test files co-located: `*_test.go` (10 files across handler/ and service/)
- Mock repos: manual structs with in-memory `map[string]*Model`, implement full interface
- Shared mocks: `backend/internal/api/handler/mocks_test.go`
- Table-driven test patterns for combinatorial logic (e.g., quadrant calculation)
- Mocks return `repository.ErrNotFound` for missing records
- Coverage: `go test -coverprofile=coverage.out`

**Frontend (Vitest):**
- Framework: Vitest 2.1 + `@vue/test-utils` + jsdom
- Test files co-located: `*.test.ts` and `*.spec.ts` (26 files)
- Pinia isolation: `setActivePinia(createPinia())` in each `beforeEach`
- Mocking: `vi.mock()` for stores, API client, router, WebSocket, `ElMessage`
- Components: test rendering, emits, computed properties, method behavior
- Stores: test initial state, each action (success + error paths), computed
- Type checking: `npx vue-tsc --noEmit`
- Coverage: `@vitest/coverage-v8` installed

## Commit & PR Conventions [✓ auto]

**Conventional Commits** format:
```
feat: add schedule revision UI (Story 1.2)
fix: prevent null-events crash when applying schedule revision
refactor: redesign frontend with refined editorial minimalism aesthetic
docs: add bug fix workflow report for null-events crash
```
Prefixes: `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`

**Branch strategy**:
- `main` — stable/production
- `evolve/*` — feature development branches

**CI/CD**: Not configured (no `.github/workflows/` or equivalent). <!-- HUMAN_REVIEW: 确认 CI 是否外部管理或尚未设置 -->

## Do Not / Gotchas [? review]

**Auto-detected invariants:**
- `backend/internal/` packages are Go compiler-enforced private — nothing outside `backend/` can import them
- Repository constructors return interface types, not concrete structs — always program against the interface
- Frontend types live in a single barrel file `src/types/index.ts` — add new types there, never alongside components
- WebSocket connection established once in `App.vue` on mount via singleton `wsClient`
- Pinia stores must be isolated in tests: `setActivePinia(createPinia())` in each `beforeEach`
- Go test mocks live in `handler/mocks_test.go` (shared across handler tests)
- AI API key stored in `backend/configs/config.yaml` (plaintext) — never committed to git
- SQLite is single-writer; concurrent writes during timer sessions are not stress-tested
- Frontend stores call the API client directly — no intermediate service layer
- Topological dependency order must be respected when adding new cross-cutting packages

<!-- HUMAN_REVIEW: 请补充以下内容：
- 数据库迁移规则（GORM AutoMigrate 的使用限制、是否允许 destructive migration）
- AI API 调用的 rate limiting 和 error handling 策略
- 前端 dist/ 的部署流程（是否有 nginx 配置、CDN 策略）
- Element Plus 组件的自定义覆盖规则（避免全局样式污染）
- 已知的性能瓶颈（大量任务时的查询性能、WebSocket 消息频率上限）
-->

## 架构文档 [~ inferred]

详见 [ARCHITECTURE.md](ARCHITECTURE.md)

Three-section format (matklad style):
- Bird's-eye view: project purpose and stack summary
- Code map: 16 module entries with key types, conventions, and architectural roles
- Cross-cutting concerns: error handling, observability, testing, build/deploy, security, performance, i18n

## 知识库 [~ inferred]

详见 [Knowledge Base](docs/knowledge/README.md)

Mermaid + C4 模型可视化知识库：
- C4 Context/Container 系统全景图 (2 张)
- 模块蓝图：8 个核心模块的 Component 图
- 流程蓝图：5 条核心业务流程的时序图
- ADR：4 篇架构决策记录 (SQLite/DI/WebSocket/AI)
- 横切关注点索引

## 领域能力 [~ inferred]

- **领域能力**：`docs/skills/AGENTS.md` — 可复用自动化操作（7 个 skill）

> 遇到"安排日程"、"修订日程"、"实现变更"→ 查 docs/skills/。

### 日程管理
- [auto-schedule](docs/skills/auto-schedule/SKILL.md) — 触发词："安排日程"、"生成日程"、"排一下任务"、"帮我规划"
- [revise-schedule](docs/skills/revise-schedule/SKILL.md) — 触发词："修订日程"、"调整日程"、"优化日程"、"重新安排"

### OpenSpec 变更管理
- [openspec-propose](docs/skills/openspec-propose/SKILL.md) — 提出新变更
- [openspec-apply-change](docs/skills/openspec-apply-change/SKILL.md) — 实现变更任务
- [openspec-archive-change](docs/skills/openspec-archive-change/SKILL.md) — 归档已完成变更
- [openspec-explore](docs/skills/openspec-explore/SKILL.md) — 浏览规范结构
- [openspec-sync-specs](docs/skills/openspec-sync-specs/SKILL.md) — 同步规范文件
