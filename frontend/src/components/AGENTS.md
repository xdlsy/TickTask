# components

## Responsibility [~ inferred]

Vue 3 SFC components organized by feature domain. Components use `<script setup lang="ts">` with Composition API, receive data from Pinia stores, and emit events upward to views.

Feature groups:
- `tasks/` — `QuadrantView` (Eisenhower Matrix grid with drag-and-drop), `ListView` (flat task list), `TaskCard` (individual task display with actions), `TaskForm` (create/edit dialog)
- `timer/` — `TimerDisplay` (circular progress with remaining time), `TimerControls` (play/pause/stop buttons with session type selector)
- `schedule/` — `DayView`, `WeekView`, `MonthView` (calendar views with time-slot grid), `EventForm` (event create/edit dialog)

## Conventions [~ inferred]

- All components use `<script setup lang="ts">` with Composition API
- Props defined with `defineProps<T>()`, emits with `defineEmits<T>()`
- Data flows down from stores via `storeToRefs()` or direct store access
- Events flow up: components emit, views handle (e.g., `@save`, `@close`, `@drag-start`)
- Element Plus components used throughout (dialogs, forms, buttons, tags, date pickers)
- CSS: scoped `<style scoped>`, kebab-case class names, design tokens from `App.vue` global styles
- Task card color-coding by quadrant (red=Q1, orange=Q2, blue=Q3, gray=Q4)

## Dependencies [✓ auto]

- Depends on: `stores/`, `types/`, `element-plus`, `vue`
- Depended on by: `views/`
