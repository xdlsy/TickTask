# views

## Responsibility [~ inferred]

Page-level Vue SFC components representing the top-level routes. Each view orchestrates Pinia stores and composes feature components from `components/`. Views handle page layout, navigation, and cross-component coordination.

Pages:
- `Dashboard.vue` — landing page with overview widgets, task summary, recent activity
- `Tasks.vue` — task management page, hosts QuadrantView/ListView components
- `Timer.vue` — Pomodoro timer page with timer display, controls, session history
- `Schedule.vue` — calendar schedule page with day/week/month views, AI generation, revision workflow
- `Analytics.vue` — usage analytics with charts for focus time, completion rates, trends
- `Settings.vue` — configuration page for pomodoro settings, AI API settings

## Conventions [~ inferred]

- Views use `<script setup lang="ts">` with Composition API
- Store initialization: views call store actions (e.g., `fetchTasks()`) in `onMounted` or via watchers
- Co-located test files: `*.test.ts` per view for unit/integration tests
- Views are the only place that directly import multiple stores and coordinate cross-store logic
- Lazy-loaded via Vue Router `() => import(...)` — code-split per page

## Dependencies [✓ auto]

- Depends on: `stores/`, `components/`, `types/`, `router/`, `element-plus`
- Depended on by: `router/` (lazy imports)
