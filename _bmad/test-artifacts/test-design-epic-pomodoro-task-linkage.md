---
workflowStatus: 'completed'
totalSteps: 5
stepsCompleted: ['step-01-detect-mode', 'step-02-load-context', 'step-03-risk-and-testability', 'step-04-coverage-plan', 'step-05-generate-output']
lastStep: 'step-05-generate-output'
nextStep: ''
lastSaved: '2026-06-08'
inputDocuments:
  - 'docs/superpowers/specs/2026-06-08-pomodoro-task-linkage-design.md'
  - 'CLAUDE.md'
  - 'frontend/tests/support/fixtures/test-fixtures.ts'
  - 'frontend/tests/support/factories/session-factory.ts'
  - 'frontend/tests/support/factories/task-factory.ts'
---

# Test Design: Epic - Pomodoro-Task Linkage

**Date:** 2026-06-08
**Author:** Master Test Architect
**Status:** Draft

---

## Executive Summary

**Scope:** Epic-level test design for the Pomodoro-Task Linkage feature

**Risk Summary:**

- Total risks identified: 8
- High-priority risks (score >= 6): 3
- Critical categories: DATA, TECH, BUS

**Coverage Summary:**

- P0 scenarios: 12 (~24-36 hours)
- P1 scenarios: 17 (~17-25 hours)
- P2 scenarios: 12 (~6-12 hours)
- P3 scenarios: 4 (~1-2 hours)
- **Total effort**: ~48-75 hours (~1.5-2.5 weeks)

---

## Not in Scope

| Item | Reasoning | Mitigation |
|------|-----------|------------|
| Per-session custom duration | Uses global settings only | Covered by existing settings tests |
| Push notifications | Out of feature scope (in-app dialog only) | Monitor for future enhancement |
| Multi-user pomodoro | Single-user application | N/A |
| Historical data migration | No existing data to migrate | N/A |
| Performance stress testing | Small dataset expected | R-006 monitors for degradation |

---

## Risk Assessment

### High-Priority Risks (Score >= 6)

| Risk ID | Category | Description | Probability | Impact | Score | Mitigation | Owner | Timeline |
|---------|----------|-------------|-------------|--------|-------|------------|-------|----------|
| R-001 | DATA | Pomodoro progress calculation inconsistent -- aggregate query may miscount due to session status/type filter errors | 2 | 3 | 6 | API tests validate count across all session status/type combinations | QA | Pre-merge |
| R-002 | TECH | WebSocket timing issue for completion reminder -- event arrives before task data refreshes | 2 | 3 | 6 | E2E tests use WebSocket wait strategy, verify reminder only fires at correct moment | QA | Pre-merge |
| R-003 | BUS | Completion reminder fires at wrong time -- completed tasks or tasks without estimate still trigger dialog | 2 | 3 | 6 | Boundary tests cover all no-reminder scenarios | QA | Pre-merge |

### Medium-Priority Risks (Score 3-4)

| Risk ID | Category | Description | Probability | Impact | Score | Mitigation | Owner |
|---------|----------|-------------|-------------|--------|-------|------------|-------|
| R-004 | TECH | Task card and detail dialog show inconsistent pomodoro data | 2 | 2 | 4 | Shared data source component, E2E consistency check | QA |
| R-005 | DATA | Schedule quick-start picks wrong "nearest task" -- time comparison ignores completed tasks | 1 | 3 | 3 | API tests validate task filtering logic | QA |
| R-006 | TECH | Analytics API degrades with large session counts | 1 | 3 | 3 | Monitor; add index if needed | Dev |

### Low-Priority Risks (Score 1-2)

| Risk ID | Category | Description | Probability | Impact | Score | Action |
|---------|----------|-------------|-------------|--------|-------|--------|
| R-007 | OPS | New API endpoints slow without index | 1 | 2 | 2 | Add index when perf issue observed |
| R-008 | BUS | Empty state for analytics when no pomodoro data | 1 | 1 | 1 | Boundary test for UI |

### Risk Category Legend

- **TECH**: Technical/Architecture (flaws, integration, scalability)
- **SEC**: Security (access controls, auth, data exposure)
- **PERF**: Performance (SLA violations, degradation, resource limits)
- **DATA**: Data Integrity (loss, corruption, inconsistency)
- **BUS**: Business Impact (UX harm, logic errors, revenue)
- **OPS**: Operations (deployment, config, monitoring)

---

## Entry Criteria

- [ ] Pomodoro-Task Linkage feature backend API implemented and deployed
- [ ] Task API returns `planned_pomodoros`, `completed_pomodoros`, `pomodoro_status` fields
- [ ] Analytics API endpoints (`pomodoro-by-task`, `pomodoro-trends`) available
- [ ] Frontend task cards, detail dialog, and schedule view updated with pomodoro UI
- [ ] Completion reminder dialog implemented
- [ ] Test environment provisioned with backend + frontend running
- [ ] Task factory and session factory updated with pomodoro-related fields

## Exit Criteria

- [ ] All P0 tests passing (100%)
- [ ] All P1 tests passing or failures triaged (>=95%)
- [ ] No open high-priority / high-severity bugs
- [ ] Test coverage agreed as sufficient
- [ ] Risk mitigations R-001, R-002, R-003 verified

---

## Test Coverage Plan

> **Note:** P0/P1/P2/P3 = priority and risk level, NOT execution timing. See Execution Strategy for timing.

### P0 (Critical)

**Criteria**: Blocks core journey + High risk (>=6) + No workaround

| Test ID | Requirement | Test Level | Risk Link | Scenario | Notes |
|---------|-------------|-----------|-----------|----------|-------|
| PTL-E2E-001 | Task API returns correct pomodoro fields | API | R-001 | Task with EstimatedTime=100, WorkDuration=1500s returns plannedPomodoros=4 | Verify ceil calculation |
| PTL-E2E-002 | Task API returns correct pomodoro fields | API | R-001 | Task with EstimatedTime=0 returns plannedPomodoros=0, pomodoroStatus="not_started" | No estimate case |
| PTL-E2E-003 | Task API returns correct pomodoro fields | API | R-001 | Task with 3 completed work sessions returns completedPomodoros=3 | Only count work+completed |
| PTL-E2E-004 | Task API pomodoro status transitions | API | R-001 | Verify all 4 status transitions: not_started -> in_progress -> completed -> exceeded | Status logic |
| PTL-E2E-005 | Pomodoro status excludes non-work sessions | API | R-001 | Break sessions do not increment completedPomodoros | Type filter |
| PTL-E2E-006 | Completion reminder fires at correct time | E2E | R-002, R-003 | Complete last planned pomodoro -> ElMessageBox dialog appears | Core flow |
| PTL-E2E-007 | Completion reminder does NOT fire for no-estimate task | E2E | R-003 | Complete pomodoro on task with plannedPomodoros=0 -> no dialog | Boundary |
| PTL-E2E-008 | Completion reminder does NOT fire for completed task | E2E | R-003 | Task status=completed -> complete pomodoro -> no dialog | Boundary |
| PTL-E2E-009 | "Mark complete" action works | E2E | R-003 | Click "标记任务完成" -> task status changes to completed | Branch A |
| PTL-E2E-010 | "Continue" action creates new pomodoro | E2E | R-003 | Click "再来一个番茄钟" -> new session created, completedPomodoros increments | Branch B |
| PTL-E2E-011 | Start pomodoro from task card button | E2E | R-004 | Click play button on task card -> pomodoro session starts with correct task_id | Quick start |
| PTL-E2E-012 | Task card shows pomodoro progress | E2E | R-004 | Task with 2/4 pomodoros shows "2/4 番茄钟" text on card | Display |

**Total P0**: 12 tests, ~24-36 hours

### P1 (High)

**Criteria**: Important features + Medium risk (3-4) + Common workflows

| Test ID | Requirement | Test Level | Risk Link | Scenario | Notes |
|---------|-------------|-----------|-----------|----------|-------|
| PTL-E2E-013 | Task detail dialog shows progress bar | E2E | R-004 | Open detail dialog -> progress bar fills to correct percentage | Visual |
| PTL-E2E-014 | Task detail dialog shows session history | E2E | R-004 | Dialog shows today's completed sessions with time ranges | History list |
| PTL-E2E-015 | Task detail dialog "start Nth pomodoro" button | E2E | R-004 | Button shows "开始第 3 个番茄钟" for 2/4 task | Context-aware |
| PTL-E2E-016 | Task detail dialog footer stats | E2E | R-004 | Footer shows "已专注 50 分钟 · 剩余约 50 分钟" | Calculation |
| PTL-E2E-017 | "Mark complete" changes task status | E2E | R-003 | After reminder "标记完成" -> task card shows completed state | Full flow |
| PTL-E2E-018 | "Continue" extends pomodoro count | E2E | R-003 | After reminder "继续" -> status becomes "exceeded" after next completion | Extension flow |
| PTL-E2E-019 | Reminder does not re-fire after extension | E2E | R-003 | After extending, complete next pomodoro -> no dialog (already exceeded) | Dedup |
| PTL-E2E-020 | Reminder does not fire during break | E2E | R-002 | Complete a break session -> no reminder | Session type check |
| PTL-E2E-021 | Schedule quick-start selects nearest task | E2E | R-005 | Click "开始番茄" -> starts pomodoro for nearest upcoming task | Auto-select |
| PTL-E2E-022 | Schedule quick-start when active pomodoro exists | E2E | R-005 | Active session running -> button shows "查看进行中" | State |
| PTL-E2E-023 | Schedule quick-start when no pending tasks | E2E | R-005 | No tasks available -> button disabled with tooltip | Empty state |
| PTL-E2E-024 | Schedule event card shows pomodoro progress | E2E | - | Calendar event shows "2/4 番茄钟" text | Display |
| PTL-E2E-025 | Schedule event click opens task detail | E2E | - | Click event -> same detail dialog as task view | Shared component |
| PTL-E2E-026 | Analytics pomodoro-by-task API returns ranking | API | R-004 | Multiple tasks with sessions -> sorted by pomodoro count desc | Ranking |
| PTL-E2E-027 | Analytics pomodoro-trends API returns daily data | API | R-004 | Daily planned vs actual comparison for 7-day period | Trend data |
| PTL-E2E-028 | Analytics completion rate calculation | API | R-004 | Three categories: on-time, exceeded, incomplete percentages | Math |
| PTL-E2E-029 | Analytics period filter works | API | R-004 | Switch week/month -> correct date range in response | Filtering |

**Total P1**: 17 tests, ~17-25 hours

### P2 (Medium)

**Criteria**: Secondary features + Low risk (1-2) + Edge cases

| Test ID | Requirement | Test Level | Risk Link | Scenario | Notes |
|---------|-------------|-----------|-----------|----------|-------|
| PTL-E2E-030 | No-estimate task starts free pomodoro | E2E | - | Task with EstimatedTime=0 -> start button works, no progress shown | Free mode |
| PTL-E2E-031 | No-estimate task never triggers reminder | E2E | - | Complete multiple pomodoros on no-estimate task -> no dialog | No boundary |
| PTL-E2E-032 | No-estimate task card shows "—" progress | E2E | - | Card displays "—" in progress area | Placeholder |
| PTL-E2E-033 | Completed task has no start button | E2E | R-003 | Task status=completed -> play button hidden, slight opacity | UI state |
| PTL-E2E-034 | Completed task card shows check mark | E2E | R-003 | Card shows "N/N ✓" with completed status | Final state |
| PTL-E2E-035 | Exceeded status shows on API | API | R-001 | completedPomodoros > plannedPomodoros -> pomodoroStatus="exceeded" | Overflow |
| PTL-E2E-036 | Exceeded task still shows start button | API | R-001 | exceeded status -> card still has play button for extra pomodoros | Continue |
| PTL-E2E-037 | Non-completed sessions not counted | API | R-001 | Abandoned/paused sessions excluded from completedPomodoros | Status filter |
| PTL-E2E-038 | Analytics empty state | E2E | R-008 | No pomodoro data -> analytics shows appropriate empty state | Zero data |
| PTL-E2E-039 | Analytics ranking with single task | E2E | R-008 | One task -> ranking shows single entry | Minimum data |
| PTL-E2E-040 | Exceeded tasks in completion rate | API | R-004 | Task with completedPomodoros > plannedPomodoros -> counted in "exceeded" bucket | Classification |
| PTL-E2E-041 | Incomplete tasks in completion rate | API | R-004 | Task with plannedPomodoros=4, completedPomodoros=2 -> counted as "incomplete" | Classification |

**Total P2**: 12 tests, ~6-12 hours

### P3 (Low)

**Criteria**: Nice-to-have + Exploratory

| Test ID | Requirement | Test Level | Test Count | Notes |
|---------|-------------|-----------|-----------|-------|
| PTL-E2E-042 | Pomodoro state syncs across pages | E2E | - | Start on Tasks page, switch to Schedule -> active session visible | Cross-page |
| PTL-E2E-043 | Pomodoro state syncs back from Schedule | E2E | - | Start on Schedule, check Tasks page -> progress updated | Reverse sync |
| PTL-E2E-044 | Multiple consecutive pomodoro completions | E2E | - | Complete 4 pomodoros in sequence -> reminder fires only at planned boundary | Sequential |
| PTL-E2E-045 | Long-running task with 10+ pomodoros | E2E | - | Task with EstimatedTime=300min -> 12 planned pomodoros, verify all display correctly | Scale |

**Total P3**: 4 tests, ~1-2 hours

---

## Execution Strategy

### PR (Every Commit)

All P0 + P1 functional tests: ~29 tests, ~8-12 minutes with Playwright parallelization.

Philosophy: run everything that is fast. Playwright parallelization handles hundreds of tests in 10-15 minutes.

### Nightly

P2 edge case tests: ~12 tests, ~3-5 minutes.

### On-demand

P3 exploratory: ~4 tests, run manually before releases.

---

## Resource Estimates

### Test Development Effort

| Priority | Count | Estimated Range | Notes |
|----------|-------|----------------|-------|
| P0 | 12 | ~24-36 hours | Complex setup, WebSocket timing, API validation |
| P1 | 17 | ~17-25 hours | Standard E2E coverage, shared component testing |
| P2 | 12 | ~6-12 hours | Simple boundary and edge cases |
| P3 | 4 | ~1-2 hours | Exploratory scenarios |
| **Total** | **45** | **~48-75 hours** | **~1.5-2.5 weeks** |

### Prerequisites

**Test Data:**

- Task factory: add `estimated_time` field support
- Session factory: add `task_id` linkage support
- Pomodoro settings fixture: configure WorkDuration for tests

**Tooling:**

- Playwright (existing setup)
- Custom test fixtures (extend existing `test-fixtures.ts`)
- API client (extend existing helper with new analytics endpoints)

**Environment:**

- Backend running with SQLite
- Frontend dev server (Vite)
- WebSocket connection functional

---

## Quality Gate Criteria

### Pass/Fail Thresholds

- **P0 pass rate**: 100% (no exceptions)
- **P1 pass rate**: >=95% (waivers required for failures)
- **P2/P3 pass rate**: >=90% (informational)
- **High-risk mitigations**: R-001, R-002, R-003 must be verified complete

### Coverage Targets

- **Critical paths** (pomodoro counting, reminder, start from card): >=80%
- **Data integrity** (API calculations): 100%
- **Edge cases**: >=50%

### Non-Negotiable Requirements

- [ ] All P0 tests pass
- [ ] No high-risk (>=6) items unmitigated
- [ ] Data integrity tests (DATA category) pass 100%
- [ ] Pomodoro count calculations verified for all boundary conditions

---

## Mitigation Plans

### R-001: Pomodoro Progress Calculation Inconsistent (Score: 6)

**Mitigation Strategy:**
1. API tests cover all session type/status combinations (work+completed, work+abandoned, break+completed)
2. Verify ceil calculation for various EstimatedTime/WorkDuration pairs
3. Verify status transitions with precise count assertions
**Owner:** QA
**Timeline:** Pre-merge
**Status:** Planned
**Verification:** PTL-E2E-001 through PTL-E2E-005 pass

### R-002: WebSocket Timing for Completion Reminder (Score: 6)

**Mitigation Strategy:**
1. Use existing timer helper's WebSocket wait strategy
2. Tests verify reminder appears only after data refresh completes
3. Add explicit wait for task data update before asserting dialog
**Owner:** QA
**Timeline:** Pre-merge
**Status:** Planned
**Verification:** PTL-E2E-006 passes reliably without flakiness

### R-003: Completion Reminder Fires at Wrong Time (Score: 6)

**Mitigation Strategy:**
1. Explicit tests for all no-reminder scenarios (no estimate, completed task, break session, exceeded)
2. Test both dialog branches (mark complete / continue) produce correct state changes
3. Verify reminder fires exactly once at the planned boundary
**Owner:** QA
**Timeline:** Pre-merge
**Status:** Planned
**Verification:** PTL-E2E-007 through PTL-E2E-010, PTL-E2E-019, PTL-E2E-020 pass

---

## Assumptions and Dependencies

### Assumptions

1. Task API will add computed fields without schema migration (lightweight computed layer)
2. Session factory can create sessions linked to specific tasks via task_id
3. WebSocket events for session completion are reliable (tested in existing timer tests)
4. Pomodoro settings (WorkDuration) can be configured per test via settings API

### Dependencies

1. Backend task service enrichment with pomodoro fields - Required before test development
2. Frontend task card component update with progress display - Required before E2E tests
3. Completion reminder dialog implementation - Required before E2E tests
4. Analytics API endpoints - Required before analytics tests

### Risks to Plan

- **Risk**: Backend API not ready when test development starts
  - **Impact**: E2E tests blocked, API tests can proceed with mocked responses
  - **Contingency**: Write API tests first against mock, validate against real backend later

---

## Interworking & Regression

| Service/Component | Impact | Regression Scope |
|-------------------|--------|------------------|
| **Task API** | New computed fields in response | Existing task CRUD tests must still pass |
| **Timer Service** | Session creation with task linkage | Existing timer workflow tests must still pass |
| **Analytics** | New endpoints and UI modules | Existing analytics tests must still pass |
| **WebSocket** | No protocol changes | Existing WebSocket tests must still pass |
| **Schedule View** | New quick-start button and event progress | Existing schedule tests must still pass |

---

## Appendix

### Test ID Convention

All tests use prefix `PTL-E2E-` (Pomodoro-Task Linkage, E2E).

### Knowledge Base References

- `risk-governance.md` - Risk classification framework
- `probability-impact.md` - Risk scoring methodology
- `test-levels-framework.md` - Test level selection
- `test-priorities-matrix.md` - P0-P3 prioritization

### Related Documents

- Design Spec: `docs/superpowers/specs/2026-06-08-pomodoro-task-linkage-design.md`
- Project Instructions: `CLAUDE.md`
- Existing Timer Tests: `frontend/tests/e2e/timer-workflow.spec.ts`
- Test Fixtures: `frontend/tests/support/fixtures/test-fixtures.ts`

---

**Generated by**: Master Test Architect
**Workflow**: `bmad-testarch-test-design`
**Mode**: Epic-Level
