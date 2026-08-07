# Green Test Baseline + CI — Design Spec

- **Date:** 2026-08-08
- **Branch:** `evolve/green-baseline-ci`
- **Status:** Approved (pending implementation)
- **Approach:** A — restore a trustworthy green baseline, then guard it with CI
- **Follow-up (separate cycle):** Route B — SQLite WAL + `busy_timeout` + concurrency stress test (FEAT-20260607-001)

## Goal

Pull the frontend test suite from a chronic **“11 known-red”** baseline back to **fully green**, then add a **GitHub Actions CI** so regressions are caught automatically. The objective is **developer confidence**: a permanently-red baseline means no one can trust “tests passed,” and without CI, regressions land silently.

Backend tests are already green; this cycle does **not** change backend logic — it only brings backend under CI coverage.

## Scope

**In scope:**
1. Fix the **10 schedule event-color tests** that assert `rgb(...)` but render hex under happy-dom.
2. Remove the **stale `scheduling_strategy` tests** referencing a field that no longer exists.
3. Drop the **leftover `jsdom` devDependency** (superseded by `happy-dom`).
4. Add **`.github/workflows/ci.yml`** (Go test + vue-tsc + vitest run).
5. Refresh the **README** API table and dev-plan checklist.
6. Update the now-stale **“Known red baseline”** note in `AGENTS.md` / `CLAUDE.md`.

**Out of scope (Route B, next cycle):** SQLite WAL mode, `busy_timeout` PRAGMA, write-retry, concurrency stress test. Noted here only so it is not lost.

## Decisions (from brainstorming)

1. **Color tests assert hex, not rgb.** Components emit `backgroundColor: <hex>` and happy-dom preserves hex; the tests were wrong to assert `rgb()`. We fix the **tests**, not the production components (don’t warp product code to suit a test).
2. **`scheduling_strategy` tests are deleted, not rewritten.** The field exists in neither the frontend `PomodoroSettings` type nor the backend model — there is no contract to test.
3. **CI runs `vue-tsc --noEmit` + `vitest run`, not `npm run build`.** `build` is `vue-tsc && vite build`; vue-tsc already covers type-checking and vite build adds no signal for regression catching while slowing CI. Build stays a local/release step.
4. **`npm ci` in CI** (not `install`) for reproducibility; Go modules + npm cache enabled.
5. **Empirical baseline first.** The documented “11 red” count is a snapshot — step 0 captures the actual failing list before any fix, and scope follows the real list (the `scheduling_strategy` count in particular looks like 3 blocks, not 1).

## Workstream 1 — Fix 10 schedule color tests

**Root cause:** `DayView.vue`, `WeekView.vue`, `MonthView.vue` render event background via `backgroundColor: color` where `color` is a hex string (e.g. `#a855f7`). happy-dom does **not** normalize hex to `rgb(...)`, so assertions like `background-color: rgb(168, 85, 247)` never match.

**Fix:** change each assertion to the hex the component actually emits.

| File | rgb (current) | hex (new) | meaning |
|---|---|---|---|
| `DayView.test.ts:181` | `rgb(168, 85, 247)` | `#a855f7` | custom color |
| `DayView.test.ts:195` | `rgb(184, 149, 77)` | `#b8954d` | pomodoro default |
| `DayView.test.ts:209` | `rgb(184, 69, 44)` | `#b8452c` | task default |
| `DayView.test.ts:223` | `rgb(107, 139, 111)` | `#6b8b6f` | break default |
| `MonthView.test.ts:232` | `rgb(139, 92, 246)` | `#8b5cf6` | custom color |
| `MonthView.test.ts:246` | `rgb(184, 149, 77)` | `#b8954d` | pomodoro default |
| `MonthView.test.ts:260` | `rgb(107, 139, 111)` | `#6b8b6f` | break default |
| `WeekView.test.ts:190` | `rgb(239, 68, 68)` | `#ef4444` | custom color |
| `WeekView.test.ts:204` | `rgb(184, 149, 77)` | `#b8954d` | pomodoro default |
| `WeekView.test.ts:218` | `rgb(107, 139, 111)` | `#6b8b6f` | break default |

> **Implementation note:** the hex values above are derived from the rgb triples and must be cross-checked against the actual default-color constants used by each component (`DayView`/`WeekView` compute `color`; `MonthView` uses `getEventColor(event)`). Locate those constants and assert the literal hex they hold — do not trust the rgb→hex derivation blindly. happy-dom preserves the color string **verbatim** (it does not normalize to `rgb()` and does not change casing), so the assertion must match the **exact** casing the source emits: custom colors use the lowercase hex the test mock passes (e.g. `createMockEvent({ color: '#a855f7' })`); default colors use the literal in the component’s color constant (verify its casing before asserting).

## Workstream 2 — Remove stale `scheduling_strategy` tests

**Root cause:** `Settings.test.ts` references a `scheduling_strategy` field (mock at line 247; `it()` blocks at 261, 288, 305). Neither the frontend `PomodoroSettings` type (`src/types/index.ts:47-57`) nor the backend `model.PomodoroSettings` (`backend/internal/model/setting.go`) has this field — it was removed in an earlier cleanup (see LRN-20260802-016).

**Fix:** delete the `scheduling_strategy` test blocks and the stray mock key. The field does not exist; there is nothing to verify. (If any of the 3 blocks currently happens to pass for the wrong reason, deleting it still leaves the suite correct — we are not losing real coverage.)

## Workstream 3 — Drop leftover `jsdom` dependency

`package.json` lists both `jsdom` (^29) and `happy-dom` (^20). `vitest.config.ts` runs on `happy-dom`, and `src/` has zero `jsdom` references (confirmed by grep). jsdom is dead weight left over from the LRN-20260803-018 switch.

**Fix:** remove `jsdom` from `devDependencies`, run `npm install` to regenerate the lockfile, confirm `npx vitest run` still passes.

## Workstream 4 — GitHub Actions CI

New file `.github/workflows/ci.yml`. Triggers on push to `main` and on all pull requests. Two jobs run in parallel:

**`backend` job:**
```yaml
- uses: actions/checkout@v4
- uses: actions/setup-go@v5
  with:
    go-version: '1.21'
    cache-dependency-path: backend/go.sum
- working-directory: backend
  run: go mod download
- working-directory: backend
  run: go test ./...
```

**`frontend` job:**
```yaml
- uses: actions/checkout@v4
- uses: actions/setup-node@v4
  with:
    node-version: '20'
    cache: npm
    cache-dependency-path: frontend/package-lock.json
- working-directory: frontend
  run: npm ci
- working-directory: frontend
  run: npx vue-tsc --noEmit
- working-directory: frontend
  run: npx vitest run
```

**Notes:**
- Node 20 LTS in CI (dev machine is Node 18 per LRN-018, but the jsdom ESM issue is already resolved by happy-dom, so Node 20 is safe and Node 18 is EOL).
- No `concurrency` cancel group for now (personal project; add later if needed).
- If `frontend/package-lock.json` does not exist, `npm ci` will fail — step 0 must confirm the lockfile is committed; if not, generate it (`npm install`) and commit it as part of this work.

## Workstream 5 — Refresh README

`README.md` is stale: the API table predates schedule/analytics/work-log/work-report/data endpoints, and the “开发计划” checklist stops at the initial 8 items.

**Fix:**
- Regenerate the API endpoint table from `backend/internal/api/router.go` (the single source of truth) — add schedule, analytics, work-logs, work-reports, data groups.
- Update the dev-plan checklist to reflect shipped features (calendar, recurring tasks, analytics, work log + period reports, data import/export + clear-all).
- Leave the tech-stack and quick-start sections unchanged (still accurate).

## Edge Cases & Risks

- **Broken local `node_modules`.** In this environment `frontend/node_modules` is incomplete (vitest binary missing); `npx vitest run` currently pulls vitest@4 from the network and fails to resolve config. This is an **environment** artifact (no `npm install` run), **not** a project defect. Implementation step 0 is `npm install` to obtain the real baseline.
- **hex derivation error.** The rgb→hex table must be verified against component constants at implementation time (see Workstream 1 note).
- **CI first-run surprises.** A clean Linux runner may surface latent failures not seen locally (path/timezone/env differences). Each is triaged on its own merit — fixed if real, not papered over.
- **Lockfile missing.** `npm ci` requires `package-lock.json`; if uncommitted, generate and commit it (also needed for reproducible CI).
- **`scheduling_strategy` removal reduces test count.** Expected and acceptable — those tests had no contract behind them.

## Definition of Done

1. `cd frontend && npm install && npx vitest run` → **0 failures** (and the inflated “Test Files failed” count from unhandled Vue warns is gone).
2. `cd backend && go test ./...` → all green.
3. `cd frontend && npx vue-tsc --noEmit` → 0 errors.
4. `.github/workflows/ci.yml` pushed; both jobs green on GitHub.
5. README API table matches `router.go` line-for-line.
6. The “Known red baseline” paragraph in `AGENTS.md` / `CLAUDE.md` is **removed** and replaced with a one-line note stating the suite is green and CI guards it.

## Testing

This cycle **is** testing work, so “testing” here means verification of the verification layer:

- **Before:** capture the actual failing-test list (`npx vitest run` after `npm install`) and attach it to the implementation plan; reconcile any count drift from the documented “11.”
- **After each workstream:** re-run `npx vitest run` and confirm the targeted failures flip to green without new failures appearing.
- **CI:** the workflow itself is validated by a green run on GitHub after push.
- **No new production code** is written, so no new unit tests are required. The `jsdom` removal is verified by a clean `vitest run`.

## Files Touched

**Frontend (edited):**
- `frontend/src/components/schedule/DayView.test.ts` — 4 color assertions (hex)
- `frontend/src/components/schedule/WeekView.test.ts` — 3 color assertions (hex)
- `frontend/src/components/schedule/MonthView.test.ts` — 3 color assertions (hex)
- `frontend/src/views/Settings.test.ts` — remove `scheduling_strategy` blocks + mock key
- `frontend/package.json` — remove `jsdom` devDependency
- `frontend/package-lock.json` — regenerate (if present; create+commit if absent)

**CI (new):**
- `.github/workflows/ci.yml`

**Docs (edited):**
- `README.md` — API table + dev-plan checklist
- `AGENTS.md` / `CLAUDE.md` — remove the “Known red baseline” paragraph; add a one-line note that the suite is green and CI guards it
