# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```bash
make dev              # Full stack dev: backend :8080 + frontend :5173 (hot reload)
make build            # Build both
make test             # Backend tests + frontend build check

# Backend (Go)
cd backend && go test ./...                              # All tests
cd backend && go test -v ./internal/service/...          # Single package
cd backend && go run cmd/server/main.go                  # Start backend only

# Frontend (Vue/TS)
cd frontend && npx vue-tsc --noEmit                      # Type check
cd frontend && npx vitest run                            # All tests
cd frontend && npx vitest run src/stores/ai.spec.ts      # Single test file
cd frontend && npm run build                             # Production build
```

## Architecture

**Backend** is manual DI in `cmd/server/main.go`: config → DB → repos → services → WebSocket hub → router. Every layer programs against interfaces. Repos are interface+private struct (constructors return the interface). Handlers are thin (bind JSON → call service → return JSON). Timer runs a goroutine with 1-second ticker broadcasting typed messages via WebSocket.

**Frontend** stores call the API client directly — no intermediate service layer. Components receive data from stores via `computed`, emit events upward. Vite proxies `/api` → `:8080` and `/ws` → `ws://:8080` in dev.

## 重启规则（关键经验）

- **后端（Go）**：每次修改 Go 代码后必须重启后端才能生效。`go run cmd/server/main.go` 或 `make dev`。
- **前端（Vue/TS）**：Vite 开发服务器支持 HMR 热更新，修改组件/store/样式会自动生效。仅当修改 `vite.config.ts` 或新增依赖时需要重启 `npm run dev`。
- **前后端数据流**：后端重启后 WebSocket 连接会断开，前端会自动重连。重启后端前建议刷新页面以清空前端状态。
- **端到端测试 AI 功能**：先 `lsof -ti:8080 | xargs kill -9` 杀旧进程，再 `cd backend && go run cmd/server/main.go` 启动后端，然后刷新前端页面。

## Key invariants

- `backend/internal/` packages are compiler-enforced private — nothing outside `backend/` can import them
- Frontend types live in a single barrel file: `src/types/index.ts` — add new types there, not alongside components
- Repository constructors return interface types; mock repos in tests use in-memory `map[string]*Model` stores
- WebSocket connection is established once in `App.vue` on mount via the singleton `wsClient`
- Pinia stores must be isolated in tests: `setActivePinia(createPinia())` in each `beforeEach`
- Go test mocks live in `handler/mocks_test.go` (shared across handler tests)
- AI API key is stored in `backend/configs/config.yaml` (plaintext) — never committed
- SQLite is a single-writer database; concurrent writes during timer sessions have not been stress-tested

## Design system

Frontend uses CSS custom properties defined in `App.vue`:
- `--bg-primary: #FAF9F6` (warm paper), `--bg-card: #FFFEFC`
- `--accent-primary: #B8452C` (burnt umber — used sparingly for primary CTAs only)
- `--text-primary: #1C1B1A`, `--text-secondary: #6E6A65`, `--text-muted: #9C9893`
- `--border-color: rgba(0,0,0,0.06)` (nearly invisible)
- Fonts: Playfair Display (display), DM Sans (body), JetBrains Mono (mono)
- No gradient backgrounds, no glow shadows, no bounce/scale hover animations — refined minimalism

## Skills

- `docs/skills/auto-schedule/` — 日程序自动生成 skill。用户用中文说"安排日程"、"生成日程"、"排一下任务"时可触发。读取 `config/todo.json` + `config/habit.md`，通过 `scripts/generate_schedule.py` 生成 `config/schedule.ics` 日历文件。
