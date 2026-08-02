---
title: 'Fix schedule revision null-events crash'
type: 'bugfix'
created: '2026-06-06'
status: 'done'
route: 'one-shot'
---

# Fix schedule revision null-events crash

## Intent

**Problem:** When users click "确定" to apply a schedule revision, the frontend crashes with a generic error "应用修订失败，请重试" if the backend returns `"events": null` instead of `"events": []` in the JSON response. Go's `nil` slice serializes to JSON `null`, and the frontend TypeScript calls `.map()` on `null`, throwing a TypeError that bypasses the proper error message display.

**Approach:** Fix the root cause in two layers — backend initializes empty slices with `make([]T, 0)` instead of `var x []T` (nil slice), and frontend adds null-coalescing guard (`?? []`) before array operations. Apply the same fix pattern to both `ApplyRevision` and `GenerateSchedule` functions.

## Suggested Review Order

1. [backend/internal/service/schedule_service.go:758](backend/internal/service/schedule_service.go#L758) — `ApplyRevision`: root cause — nil slice serialized as JSON `null`
2. [backend/internal/service/schedule_service.go:492](backend/internal/service/schedule_service.go#L492) — `GenerateSchedule`: same nil-slice pattern
3. [frontend/src/stores/schedule.ts:220](frontend/src/stores/schedule.ts#L220) — `applyRevision`: defensive null guard
4. [frontend/src/stores/schedule.ts:144](frontend/src/stores/schedule.ts#L144) — `generateScheduleFromTasks`: same null-vulnerability pattern
