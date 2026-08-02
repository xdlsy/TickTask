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

## Design System
- `--bg-primary: #FAF9F6`, `--bg-card: #FFFEFC`, `--accent-primary: #B8452C`
- Fonts: Playfair Display (display), DM Sans (body), JetBrains Mono (mono)
- No gradients, no glow shadows, no bounce/scale animations — refined minimalism

## Commits
- Conventional Commits: `feat:`, `fix:`, `refactor:`, `docs:`, `chore:`
- Branches: `main` (stable), `evolve/*` (feature dev)
