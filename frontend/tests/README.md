# TickTask E2E Tests

End-to-end tests powered by [Playwright](https://playwright.dev/).

## Prerequisites

- **Node.js** 18+
- **Backend running** on `:8080` (for API-dependent tests)
- **Playwright browsers** installed (first-time setup)

## Setup

```bash
# From project root
cd frontend
npm install
npx playwright install          # Download browser binaries (first time only)
cp .env.example .env            # Configure test environment
```

## Running Tests

```bash
# Run all E2E tests (headless)
npm run test:e2e

# Run with browser visible
npm run test:e2e:headed

# Run with Playwright Inspector (step-through debugging)
npm run test:e2e:debug

# Run a single test file
npx playwright test tests/e2e/example.spec.ts

# Run tests matching a pattern
npx playwright test -g "dashboard"
```

> **Note:** API-dependent tests require the backend running on `:8080`. Start it with `cd backend && go run cmd/server/main.go` or `make dev` from the project root.

## Architecture

```
tests/
├── e2e/                    # Test specs (*.spec.ts)
├── support/
│   ├── fixtures/           # Playwright test fixtures
│   │   ├── index.ts        # Public API — import { test, expect } from here
│   │   └── test-fixtures.ts # Fixture definitions (taskFactory, apiClient, etc.)
│   ├── factories/          # Data factories with auto-cleanup
│   │   ├── task.factory.ts
│   │   └── schedule.factory.ts
│   ├── helpers/            # Test utilities
│   │   ├── api-client.ts   # Typed backend API client
│   │   └── timer.ts        # Timer/WebSocket test helpers
│   └── page-objects/       # (Add as needed)
```

### How It Fits Together

1. **Spec files** import `test` and `expect` from `support/fixtures`
2. **Fixtures** inject typed helpers (`taskFactory`, `scheduleFactory`, `apiClient`) into each test
3. **Factories** generate realistic test data via Faker and auto-cleanup after each test
4. **Helpers** wrap common interactions (API calls, timer controls, WebSocket waits)

## Best Practices

| Practice | How |
|----------|-----|
| **Selectors** | Prefer `data-testid` attributes over CSS classes or text |
| **Test data** | Use factories — they auto-cleanup via fixture teardown |
| **Structure** | Given / When / Then comments in each test |
| **No flake** | Never use `page.waitForTimeout()`. Use `expect(...).toBeVisible()` |
| **Isolation** | Each test creates its own data; never depend on test order |
| **Assertions** | Use Playwright's async matchers (`expect(...).toBeVisible()`) |

## CI Integration

```bash
# CI runs with retries and restricted workers
CI=true npm run test:e2e
```

Artifacts (traces, screenshots, videos) are captured on failure only and stored in `test-results/`.

## Troubleshooting

| Issue | Fix |
|-------|-----|
| `Error: browserType.launch` | Run `npx playwright install` |
| Tests timeout on startup | Ensure backend is running on `:8080` |
| `BASE_URL` errors | Copy `.env.example` to `.env` and verify values |
| Port already in use | `lsof -ti:8080 \| xargs kill -9` to free the port |
