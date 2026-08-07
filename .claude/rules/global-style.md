---
paths: ["**/*.go", "**/*.ts", "**/*.vue"]
---

# Code Style

## Go
- Standard conventions: `gofmt`, PascalCase exports, camelCase unexported
- Package names: short, lowercase, single-word (`model`, `service`, `repository`)
- File naming: `snake_case` with domain prefix (`task_repo.go`, `task_service.go`)
- Interfaces: exported PascalCase noun; implementations: unexported lowercase struct
- Constructors: `New*` prefix returning the interface type
- Handler DTOs: `*Input` suffix; service DTOs: `*Request`/`*DTO` suffix
- Module path: `ticktask`
- No linter configured (no `.golangci.yml`)

## TypeScript/Vue
- Vue SFC with `<script setup lang="ts">` (Composition API)
- Components: PascalCase `.vue` files; views are singular nouns
- Stores: `useXxxStore` naming, Composition API `defineStore`
- API methods: CRUD-prefix (`getTasks`, `createTask`, `updateTask`)
- CSS: kebab-case classes, design tokens as CSS custom properties in `App.vue`
- TypeScript: `strict: true`, `noUnusedLocals`, `noUnusedParameters`
- No ESLint/Prettier (strict TS compiler acts as baseline check)

## Design System — "Atelier Noir" (warm-ink editorial dark)
- `--bg-primary: #14120D` (warm ink), `--bg-card: #1E1B14`, `--accent-primary: #E6A23C` (amber)
- Status accents: `--accent-sage #8FB28C`, `--accent-gold #D6B45A`, `--accent-crimson #D86F54`, `--accent-sky #7FA8C0`; soft fills via `--*-fill` tokens
- Fonts: Fraunces (display, opsz), Geist (body), Geist Mono (data/numerals)
- Amber used sparingly; status colors only for semantics. Ambient warm glow + fine grain overlay in `App.vue`
- Element Plus dark: `element-plus/theme-chalk/dark/css-vars.css` imported in `main.ts`, `<html class="dark">` in `index.html`, `--el-*` brand overrides in `App.vue :root`
- Editorial signatures: mono uppercase eyebrow with hairline rule; Fraunces page titles & large numerals; mono pills/tags (`border-radius:999px`)

## Commits
- Conventional Commits: `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`
- Branches: `main` (stable), `evolve/*` (feature dev)
