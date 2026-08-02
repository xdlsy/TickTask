---
paths: ["**/*_test.go", "**/*.test.ts", "**/*.spec.ts"]
---

# Testing Guidelines

## Backend (Go)
- Framework: standard `testing` package (no testify/gomock)
- Test files co-located: `*_test.go`
- Mock repos: manual structs with in-memory `map[string]*Model`, implement full interface
- Shared mocks: `backend/internal/api/handler/mocks_test.go`
- Table-driven test patterns for combinatorial logic
- Mocks return `repository.ErrNotFound` for missing records
- Coverage: `cd backend && go test -coverprofile=coverage.out`

## Frontend (Vitest)
- Framework: Vitest 2.1 + `@vue/test-utils` + jsdom
- Test files: `*.test.ts` and `*.spec.ts`
- Pinia isolation: `setActivePinia(createPinia())` in each `beforeEach`
- Mocking: `vi.mock()` for stores, API client, router, WebSocket, `ElMessage`
- Type checking: `cd frontend && npx vue-tsc --noEmit`
- Coverage: `@vitest/coverage-v8` installed
