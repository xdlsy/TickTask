# api

## Responsibility [~ inferred]

HTTP transport layer: Gin router setup, request binding/validation, response formatting, and CORS middleware. Handlers are thin — they parse input, delegate to services, and return JSON responses.

Structure:
- `router.go` — route registration grouped by domain under `/api`, SPA fallback for non-API routes
- `handler/` — per-domain handler files (task, timer, ai, setting, analytics, schedule)
- `middleware/cors.go` — origin-based CORS with config allowlist, OPTIONS preflight short-circuit

## Conventions [~ inferred]

- Each handler struct holds a `*service.XxxService` pointer, created via `New*Handler(svc)`
- Handlers are not singletons — instantiated per route registration inline in `router.go`
- Input binding: `c.ShouldBindJSON(&input)` with `binding:"required"` struct tags
- Response format: `gin.H{"error": err.Error()}` on failure, result directly on success
- HTTP status code mapping: 400 (validation), 404 (not found), 500 (internal), 503 (AI not configured)
- AI handler uses `service.GetAIServiceWithTimeout()` with 30s/60s deadlines
- Settings handler masks API key in responses (first 4 + last 4 characters)

## Dependencies [✓ auto]

- Depends on: `service/`, `websocket/`, `repository/` (for settings), `github.com/gin-gonic/gin`
- Depended on by: `cmd/server/main.go` (router setup and server start)
