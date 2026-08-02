---
stepsCompleted: ['step-01-preflight', 'step-02-select-framework', 'step-03-scaffold-framework', 'step-04-docs-and-scripts']
lastStep: 'step-04-docs-and-scripts'
lastSaved: '2026-06-07'
---

## Step 1 Output: Preflight Checks

- **Detected stack**: fullstack (frontend/package.json + backend/go.mod)
- **Frontend**: Vue 3.5 + Vite 5.4 + TypeScript 5.6 (strict) + Element Plus 2.8
- **Backend**: Go 1.21 + Gin 1.10 + GORM 1.25 + SQLite
- **Existing E2E**: None
- **Architecture docs**: ARCHITECTURE.md, docs/knowledge/README.md
- **Dev server**: Frontend :5173 (proxies /api → :8080, /ws → ws://:8080)
- **Prerequisites**: PASS

## Step 2 Output: Framework Selection

- **Framework**: Playwright
- **Rationale**: Fullstack coverage, WebSocket support, multi-browser, TypeScript-first, CI speed, no conflict with Vitest
- **Backend**: No change (go test with existing mocks)

## Step 3 Output: Scaffold

- **Directory structure**: frontend/tests/{e2e,support/{fixtures,helpers,factories,page-objects}}
- **Config**: frontend/playwright.config.ts (baseURL :5173, chromium, webServer auto-start)
- **Fixtures**: test-fixtures.ts (taskFactory, scheduleFactory, apiClient with auto-cleanup)
- **Factories**: task.factory.ts, schedule.factory.ts (Faker-based, cleanup tracking)
- **Helpers**: api-client.ts (typed backend API), timer.ts (WebSocket/timer utilities)
- **Sample test**: example.spec.ts (5 smoke tests)
- **Dependencies**: @playwright/test, @faker-js/faker installed
- **Scripts**: test:e2e, test:e2e:headed, test:e2e:debug added to package.json
- **Makefile**: test-e2e target added

## Step 4 Output: Documentation

- **README**: frontend/tests/README.md (setup, running, architecture, best practices, troubleshooting)
- **Scripts**: package.json and Makefile updated
