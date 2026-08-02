# router

## Responsibility [✓ auto]

Vue Router configuration with HTML5 history mode and lazy-loaded route components. Defines 6 page routes plus a root redirect.

Routes:
- `/` — redirects to `/dashboard`
- `/dashboard` — Dashboard page
- `/timer` — Pomodoro timer page
- `/tasks` — Task management page
- `/schedule` — Calendar schedule page
- `/analytics` — Usage analytics page
- `/settings` — Application settings page

## Conventions [~ inferred]

- All route components use dynamic `() => import(...)` for code splitting
- Route names match PascalCase component names (e.g., `name: 'Dashboard'`)
- HTML5 history mode (no hash-based routing)
- No route guards or middleware configured
- Single flat route array — no nested routes
- Co-located test: `index.test.ts`

## Dependencies [✓ auto]

- Depends on: `vue-router`, `@/views/*.vue` (lazy imports)
- Depended on by: `App.vue` (mounts router), `main.ts` (app setup)
