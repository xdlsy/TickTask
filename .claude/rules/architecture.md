---
paths: []
---

# Architecture Invariants

- `backend/internal/` packages are Go compiler-enforced private — nothing outside `backend/` can import them
- Repository constructors return interface types, not concrete structs — always program against the interface
- Frontend types live in a single barrel file `src/types/index.ts` — add new types there, never alongside components
- WebSocket connection established once in `App.vue` on mount via singleton `wsClient`
- Frontend stores call the API client directly — no intermediate service layer
- Topological dependency order must be respected when adding new cross-cutting packages
- SQLite is single-writer; concurrent writes during timer sessions are not stress-tested
- AI API key stored in `backend/configs/config.yaml` (plaintext) — never committed to git
- Go test mocks live in `handler/mocks_test.go` (shared across handler tests)
- Pinia stores must be isolated in tests: `setActivePinia(createPinia())` in each `beforeEach`

<!-- HUMAN_REVIEW: 补充数据库迁移规则、AI API rate limiting 策略、Element Plus 自定义覆盖规则 -->
