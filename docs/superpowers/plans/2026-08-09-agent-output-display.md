# Agent Output Display Optimization — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the agent drawer's flat plain-text message list with a turn-grouped timeline that renders Markdown, shows smart-summary tool rows with collapsible JSON + inline confirm, and lights up tools live as they run.

**Architecture:** Source of truth stays the flat `store.messages` array. A `useAgentTurns` composable derives an ordered `Turn[]` (user prompt → interleaved assistant text + tool segments) for the view. Two store fixes materialize tool events live and flush assistant text per-`message_id` so text↔tool interleaving survives streaming. New presentational components: `MarkdownText` (marked + DOMPurify), `ToolRow` (smart summary + collapsible JSON + inline confirm), `AgentTurn` (avatar + segments + live indicator). `ToolCard` and `ToolConfirmDialog` are deleted; their behavior folds into `ToolRow`.

**Tech Stack:** Vue 3.5 `<script setup lang="ts">` + Pinia, Element Plus 2.8, Vitest 2.1 + jsdom + @vue/test-utils, `marked` + `dompurify` (new deps). Design tokens are CSS custom properties in `App.vue` (Atelier Noir warm-ink dark).

**Spec:** `docs/superpowers/specs/2026-08-09-agent-output-display-design.md`

**Working branch:** `evolve/agent-output-display` (already created; spec already committed).

**Conventions (from `AGENTS.md` / `.claude/rules/`):**
- Run a single test file: `cd frontend && npx vitest run src/path/file.spec.ts`
- Type check: `cd frontend && npx vue-tsc --noEmit`
- Full suite: `cd frontend && npm run test:run`
- Pinia isolation in every test `beforeEach`: `setActivePinia(createPinia())`
- Types live in `src/types/index.ts` — but view-local models (`Segment`/`Turn`) stay in `useAgentTurns.ts`
- Conventional Commits; end commit messages with `Co-Authored-By: Claude <noreply@anthropic.com>`

**Tool permission map (from `backend/internal/agent/tools/*.go` — drives `ToolRow` coloring):**
- **danger (crimson):** `delete_task`, `delete_schedule`, `apply_schedule_revision`
- **write (gold):** `create_task`, `update_task`, `move_task`, `generate_schedule`, `update_schedule`, `create_schedule`, `revise_schedule`, `start_pomodoro`, `stop_pomodoro`, `control_pomodoro`, `save_worklog`, `generate_work_report`, `update_worklog`, `update_worklog_summary`, `add_worklog_entry`
- **read (sage):** everything else (`list_tasks`, `list_schedule`, `get_timer_status`, `get_pomodoro_stats`, `get_daily_insights`, `get_analytics`, `get_settings`, `classify_task`, `structure_worklog`, `get_worklog`, `list_worklogs`, `get_work_report`)

---

## Task 1: Add `marked` and `dompurify` dependencies

**Files:**
- Modify: `frontend/package.json`, `frontend/package-lock.json`

- [ ] **Step 1: Install the deps**

```bash
cd frontend && npm install marked dompurify && npm install -D @types/dompurify
```

> `dompurify` ships its own types in recent versions; if `@types/dompurify` errors as already-present, drop it — the install command is safe to run as-is and npm will reconcile.

- [ ] **Step 2: Verify the imports resolve and DOMPurify works under jsdom**

Create a throwaway check (delete after):

```bash
cd frontend && cat > /tmp/md-check.spec.ts <<'EOF'
import { describe, it, expect } from 'vitest'
import { marked } from 'marked'
import DOMPurify from 'dompurify'
describe('md check', () => {
  it('parses + sanitizes', () => {
    const raw = marked.parse('**hi**', { gfm: true, breaks: true, mangle: false, headerIds: false }) as string
    expect(DOMPurify.sanitize(raw, { USE_PROFILES: { html: true } })).toContain('<strong>hi</strong>')
  })
})
EOF
cp /tmp/md-check.spec.ts src/__mdcheck.spec.ts && npx vitest run src/__mdcheck.spec.ts; rm src/__mdcheck.spec.ts
```

Expected: PASS (1 test).

- [ ] **Step 3: Commit**

```bash
cd frontend && git add package.json package-lock.json
git commit -m "chore(agent-ui): add marked + dompurify deps

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: `toolFormatters.ts` — smart-summary formatter (TDD)

Pure, dependency-free module. Produces a one-line human summary for a tool message.

**Files:**
- Create: `frontend/src/components/agent/toolFormatters.ts`
- Test: `frontend/src/components/agent/toolFormatters.spec.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/components/agent/toolFormatters.spec.ts`:

```ts
import { describe, it, expect } from 'vitest'
import { summarizeTool, classifyPermission } from './toolFormatters'
import type { AgentMessage } from '@/types'

function msg(over: Partial<AgentMessage>): AgentMessage {
  return {
    id: 'm', conversation_id: 'c', role: 'tool_call', content: '',
    tool_name: 'list_tasks', tool_args: '{}', tool_status: 'succeeded', created_at: '',
    ...over,
  }
}

describe('classifyPermission', () => {
  it('delete_task is danger', () => {
    expect(classifyPermission('delete_task')).toBe('danger')
  })
  it('create_task is write', () => {
    expect(classifyPermission('create_task')).toBe('write')
  })
  it('list_tasks is read', () => {
    expect(classifyPermission('list_tasks')).toBe('read')
  })
  it('unknown tool defaults to read', () => {
    expect(classifyPermission('something_new')).toBe('read')
  })
})

describe('summarizeTool', () => {
  it('array result → count hint with noun', () => {
    const s = summarizeTool(msg({ tool_name: 'list_tasks', tool_result: '[{"id":1},{"id":2}]' }))
    expect(s.resultHint).toBe('2 条任务')
  })
  it('create_schedule array result uses 时段 noun', () => {
    const s = summarizeTool(msg({ tool_name: 'create_schedule', tool_result: '[1,2,3]' }))
    expect(s.resultHint).toBe('3 个时段')
  })
  it('create_task surfaces title as argHint', () => {
    const s = summarizeTool(msg({ tool_name: 'create_task', tool_args: '{"title":"写文档"}', tool_status: 'pending_confirmation' }))
    expect(s.argHint).toBe('title=写文档')
  })
  it('unknown tool falls back to first scalar arg + generic count', () => {
    const s = summarizeTool(msg({ tool_name: 'mystery_tool', tool_args: '{"q":"x"}', tool_result: '[1,2]' }))
    expect(s.argHint).toBe('q=x')
    expect(s.resultHint).toBe('2 项')
  })
  it('failed status yields no resultHint', () => {
    const s = summarizeTool(msg({ tool_name: 'list_tasks', tool_status: 'failed', tool_result: '{"error":"oops"}' }))
    expect(s.resultHint).toBeUndefined()
  })
  it('start_pomodoro succeeded → 完成 hint', () => {
    const s = summarizeTool(msg({ tool_name: 'start_pomodoro', tool_result: '{"started":true}' }))
    expect(s.resultHint).toBe('完成')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/components/agent/toolFormatters.spec.ts
```

Expected: FAIL — `Failed to resolve import "./toolFormatters"`.

- [ ] **Step 3: Write the implementation**

Create `frontend/src/components/agent/toolFormatters.ts`:

```ts
import type { AgentMessage, ToolStatus } from '@/types'

export type Permission = 'read' | 'write' | 'danger'

// Mirrors backend/internal/agent/tools/*.go Permission values.
const DANGER_TOOLS = new Set(['delete_task', 'delete_schedule', 'apply_schedule_revision'])
const WRITE_TOOLS = new Set([
  'create_task', 'update_task', 'move_task',
  'generate_schedule', 'update_schedule', 'create_schedule', 'revise_schedule',
  'start_pomodoro', 'stop_pomodoro', 'control_pomodoro',
  'save_worklog', 'generate_work_report', 'update_worklog', 'update_worklog_summary', 'add_worklog_entry',
])

export function classifyPermission(toolName?: string): Permission {
  if (!toolName) return 'read'
  if (DANGER_TOOLS.has(toolName)) return 'danger'
  if (WRITE_TOOLS.has(toolName)) return 'write'
  return 'read'
}

// Per-tool display overrides: which arg to surface, and the noun for a count.
const LABELS: Record<string, { argKey?: string; noun?: string; doneHint?: string }> = {
  list_tasks: { noun: '条任务' },
  list_schedule: { noun: '个时段' },
  list_worklogs: { noun: '篇日志' },
  create_schedule: { noun: '个时段' },
  generate_schedule: { noun: '个时段' },
  create_task: { argKey: 'title' },
  update_task: { argKey: 'title' },
  delete_task: { argKey: 'task_id' },
  start_pomodoro: { doneHint: '完成' },
}

const ARG_PRIORITY = ['title', 'name', 'date', 'task_id', 'id', 'date_str']

function parseArgs(m: AgentMessage): Record<string, unknown> {
  try { return JSON.parse(m.tool_args || '{}') } catch { return {} }
}
function parseResult(m: AgentMessage): unknown {
  try { return JSON.parse(m.tool_result || '') } catch { return m.tool_result || undefined }
}

function firstScalarArg(args: Record<string, unknown>): { key: string; value: string } | undefined {
  for (const k of ARG_PRIORITY) {
    if (args[k] != null) return { key: k, value: String(args[k]) }
  }
  for (const [k, v] of Object.entries(args)) {
    if (typeof v === 'string' || typeof v === 'number') return { key: k, value: String(v) }
  }
  return undefined
}

export interface ToolSummary {
  argHint?: string
  resultHint?: string
}

export function summarizeTool(m: AgentMessage): ToolSummary {
  const args = parseArgs(m)
  const result = parseResult(m)
  const cfg = LABELS[m.tool_name || '']
  const status: ToolStatus | undefined = m.tool_status

  // argHint
  let argHint: string | undefined
  if (cfg?.argKey && args[cfg.argKey] != null) {
    argHint = `${cfg.argKey}=${args[cfg.argKey]}`
  } else {
    const s = firstScalarArg(args)
    if (s) argHint = `${s.key}=${s.value}`
  }

  // resultHint (only on success — failures are surfaced by ToolRow via tool_result)
  let resultHint: string | undefined
  if (status === 'succeeded') {
    if (cfg?.doneHint) {
      resultHint = cfg.doneHint
    } else if (Array.isArray(result)) {
      resultHint = `${result.length} ${cfg?.noun || '项'}`
    } else if (result && typeof result === 'object' && 'count' in (result as Record<string, unknown>)) {
      resultHint = `${(result as Record<string, unknown>).count} ${cfg?.noun || '项'}`
    }
  }

  return { argHint, resultHint }
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd frontend && npx vitest run src/components/agent/toolFormatters.spec.ts
```

Expected: PASS (10 tests).

- [ ] **Step 5: Commit**

```bash
cd frontend && git add src/components/agent/toolFormatters.ts src/components/agent/toolFormatters.spec.ts
git commit -m "feat(agent-ui): add tool summary formatters + permission classifier

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: `MarkdownText.vue` — sanitized Markdown renderer (TDD)

**Files:**
- Create: `frontend/src/components/agent/MarkdownText.vue`
- Test: `frontend/src/components/agent/MarkdownText.spec.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/components/agent/MarkdownText.spec.ts`:

```ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import MarkdownText from './MarkdownText.vue'

describe('MarkdownText', () => {
  beforeEach(() => {
    // jsdom has no clipboard by default
    Object.assign(navigator, { clipboard: { writeText: vi.fn().mockResolvedValue(undefined) } })
  })

  it('renders bold + list as real HTML', () => {
    const w = mount(MarkdownText, { props: { content: '**hi**\n- a\n- b' } })
    const html = w.html()
    expect(html).toContain('<strong>hi</strong>')
    expect(html).toContain('<ul>')
  })

  it('strips an XSS payload', () => {
    const w = mount(MarkdownText, { props: { content: '<img src=x onerror="alert(1)">' } })
    expect(w.html()).not.toContain('onerror')
    expect(w.html()).not.toContain('alert')
  })

  it('renders a code block with a copy button + language label', async () => {
    const w = mount(MarkdownText, { props: { content: '```bash\necho hi\n```' } })
    await w.vm.$nextTick()
    const btn = w.find('button.md-copy')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toContain('bash')
  })

  it('copy button writes code text to clipboard', async () => {
    const w = mount(MarkdownText, { props: { content: '```js\nconst a = 1\n```' } })
    await w.vm.$nextTick()
    const btn = w.find('button.md-copy')
    expect(btn.exists()).toBe(true)
    expect(btn.text()).toContain('js')
    await btn.trigger('click')
    expect(navigator.clipboard.writeText).toHaveBeenCalledWith('const a = 1\n')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/components/agent/MarkdownText.spec.ts
```

Expected: FAIL — `Failed to resolve import "./MarkdownText.vue"`.

- [ ] **Step 3: Write the implementation**

Create `frontend/src/components/agent/MarkdownText.vue`:

```vue
<template>
  <div ref="root" class="md" data-testid="md" v-html="html" />
</template>

<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { marked } from 'marked'
import DOMPurify from 'dompurify'

const props = defineProps<{ content: string }>()
const root = ref<HTMLElement | null>(null)

const html = computed(() => {
  const raw = marked.parse(props.content ?? '', {
    gfm: true, breaks: true, mangle: false, headerIds: false,
  }) as string
  return DOMPurify.sanitize(raw, { USE_PROFILES: { html: true } })
})

// Inject a copy button + language label into each <pre><code> after every render.
function wireCodeBlocks() {
  const el = root.value
  if (!el) return
  el.querySelectorAll('pre').forEach((pre) => {
    if (pre.querySelector('button.md-copy')) return
    const code = pre.querySelector('code')
    if (!code) return
    const lang = (code.className.match(/language-([\w-]+)/) || [])[1] || ''
    const btn = document.createElement('button')
    btn.type = 'button'
    btn.className = 'md-copy'
    btn.textContent = lang ? `${lang} ⧉` : '⧉'
    btn.addEventListener('click', () => {
      navigator.clipboard?.writeText(code.textContent || '')
    })
    pre.appendChild(btn)
  })
}

watch(html, () => nextTick(wireCodeBlocks), { immediate: true })
</script>

<style scoped>
.md {
  font-family: var(--font-body);
  font-size: 13px;
  line-height: 1.65;
  color: var(--text-primary);
  word-wrap: break-word;
}
.md :deep(h1),
.md :deep(h2),
.md :deep(h3) {
  font-family: var(--font-display);
  font-variation-settings: 'opsz' 60;
  font-weight: 500;
  letter-spacing: -0.01em;
  color: var(--text-primary);
  margin: 12px 0 6px;
  line-height: 1.3;
}
.md :deep(h1) { font-size: 18px; }
.md :deep(h2) { font-size: 16px; }
.md :deep(h3) { font-size: 14px; }
.md :deep(p) { margin: 0 0 8px; }
.md :deep(p:last-child) { margin-bottom: 0; }
.md :deep(strong) { color: var(--accent-secondary); font-weight: 600; }
.md :deep(em) { color: var(--text-secondary); }
.md :deep(ul),
.md :deep(ol) { margin: 0 0 8px; padding-left: 18px; }
.md :deep(li) { margin: 3px 0; }
.md :deep(ol li::marker) { font-family: var(--font-mono); color: var(--text-muted); font-size: 11px; }
.md :deep(a) { color: var(--accent-primary); text-decoration: underline; text-underline-offset: 2px; }
.md :deep(code) {
  font-family: var(--font-mono);
  font-size: 11px;
  background: rgba(230, 162, 60, 0.12);
  color: var(--accent-secondary);
  padding: 1px 5px;
  border-radius: 4px;
}
.md :deep(pre) {
  position: relative;
  background: var(--bg-elevated);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 9px 11px;
  margin: 0 0 8px;
  overflow-x: auto;
}
.md :deep(pre > code) {
  display: block;
  background: transparent;
  color: var(--text-secondary);
  padding: 0;
  font-size: 11px;
  white-space: pre;
}
.md :deep(blockquote) {
  margin: 0 0 8px;
  padding: 4px 0 4px 11px;
  border-left: 2px solid var(--accent-primary);
  color: var(--text-secondary);
  font-family: var(--font-display);
  font-style: italic;
  font-size: 12.5px;
}
.md :deep(table) {
  width: 100%;
  border-collapse: collapse;
  margin: 0 0 8px;
  font-size: 11.5px;
}
.md :deep(th),
.md :deep(td) {
  text-align: left;
  padding: 4px 7px;
  border-bottom: 1px solid var(--border-color);
}
.md :deep(th) {
  font-family: var(--font-mono);
  font-size: 9.5px;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--text-muted);
  font-weight: 400;
}
.md :deep(td) { color: var(--text-primary); }
.md :deep(hr) { border: 0; border-top: 1px solid var(--border-color); margin: 10px 0; }
:deep(.md-copy) {
  position: absolute;
  top: 4px;
  right: 4px;
  font-family: var(--font-mono);
  font-size: 9px;
  color: var(--text-muted);
  background: var(--bg-card);
  border: 1px solid var(--border-color);
  border-radius: 4px;
  padding: 1px 6px;
  cursor: pointer;
}
:deep(.md-copy:hover) { color: var(--text-primary); }
</style>
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd frontend && npx vitest run src/components/agent/MarkdownText.spec.ts
```

Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
cd frontend && git add src/components/agent/MarkdownText.vue src/components/agent/MarkdownText.spec.ts
git commit -m "feat(agent-ui): add MarkdownText (marked + DOMPurify) with copy buttons

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: `useAgentTurns.ts` — turn-grouping composable (TDD)

**Files:**
- Create: `frontend/src/components/agent/useAgentTurns.ts`
- Test: `frontend/src/components/agent/useAgentTurns.spec.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/components/agent/useAgentTurns.spec.ts`:

```ts
import { describe, it, expect } from 'vitest'
import { groupIntoTurns } from './useAgentTurns'
import type { AgentMessage } from '@/types'

function user(id: string, content: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'user', content, created_at: '' }
}
function ast(id: string, content: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'assistant', content, created_at: '' }
}
function tool(id: string, name: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'tool_call', content: '', tool_name: name, tool_args: '{}', tool_status: 'succeeded', created_at: '' }
}

describe('groupIntoTurns', () => {
  it('groups a user message with its following assistant + tool messages', () => {
    const turns = groupIntoTurns([user('u1', 'hi'), ast('a1', 'ok'), tool('t1', 'list_tasks')], '', false)
    expect(turns).toHaveLength(1)
    expect(turns[0].user?.id).toBe('u1')
    expect(turns[0].segments.map((s) => s.kind)).toEqual(['text', 'tool'])
  })

  it('starts a new turn at the next user message', () => {
    const turns = groupIntoTurns([user('u1', 'hi'), ast('a1', 'ok'), user('u2', 'again')], '', false)
    expect(turns).toHaveLength(2)
    expect(turns[1].user?.id).toBe('u2')
  })

  it('skips empty assistant segments (no empty bubble)', () => {
    const turns = groupIntoTurns([user('u1', 'hi'), ast('a1', '')], '', false)
    expect(turns[0].segments).toHaveLength(0)
  })

  it('attaches live streaming text to the last turn', () => {
    const turns = groupIntoTurns([user('u1', 'hi')], 'think', true)
    expect(turns[0].live?.text).toBe('think')
  })

  it('shows a live (empty) indicator when thinking but no text yet', () => {
    const turns = groupIntoTurns([user('u1', 'hi')], '', true)
    expect(turns[0].live?.text).toBe('')
  })

  it('no live when neither streaming nor thinking', () => {
    const turns = groupIntoTurns([user('u1', 'hi'), ast('a1', 'done')], '', false)
    expect(turns[0].live).toBeUndefined()
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/components/agent/useAgentTurns.spec.ts
```

Expected: FAIL — `Failed to resolve import "./useAgentTurns"`.

- [ ] **Step 3: Write the implementation**

Create `frontend/src/components/agent/useAgentTurns.ts`:

```ts
import { computed, type ComputedRef, type Ref } from 'vue'
import type { AgentMessage } from '@/types'

export type Segment =
  | { kind: 'text'; message: AgentMessage }
  | { kind: 'tool'; message: AgentMessage }

export interface Turn {
  id: string
  user?: AgentMessage
  segments: Segment[]
  live?: { text: string }
}

/**
 * Group a flat message list into turns. A turn = one user message plus every
 * following non-user message up to the next user message. The in-flight
 * streaming state (streamingText/isThinking) attaches a `live` segment to the
 * last turn so the view can render the typing indicator / streaming bubble.
 */
export function groupIntoTurns(
  messages: AgentMessage[],
  streamingText: string,
  isThinking: boolean,
): Turn[] {
  const turns: Turn[] = []
  let current: Turn | null = null

  for (const m of messages) {
    if (m.role === 'user') {
      current = { id: m.id, user: m, segments: [] }
      turns.push(current)
      continue
    }
    if (!current) {
      // Orphan assistant/tool before any user (defensive): give it its own turn.
      current = { id: 'orphan-' + m.id, segments: [] }
      turns.push(current)
    }
    if (m.role === 'assistant') {
      if (!m.content) continue // skip empty bubbles
      current.segments.push({ kind: 'text', message: m })
    } else {
      // tool_call / tool_result
      current.segments.push({ kind: 'tool', message: m })
    }
  }

  if (streamingText || isThinking) {
    let last = turns[turns.length - 1]
    if (!last) {
      last = { id: 'live', segments: [] }
      turns.push(last)
    }
    last.live = { text: streamingText }
  }

  return turns
}

/** Reactive wrapper for use in components. */
export function useAgentTurns(
  messages: Ref<AgentMessage[]> | ComputedRef<AgentMessage[]>,
  streamingText: Ref<string>,
  isThinking: Ref<boolean>,
): ComputedRef<Turn[]> {
  return computed(() => groupIntoTurns(messages.value, streamingText.value, isThinking.value))
}
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd frontend && npx vitest run src/components/agent/useAgentTurns.spec.ts
```

Expected: PASS (6 tests).

- [ ] **Step 5: Commit**

```bash
cd frontend && git add src/components/agent/useAgentTurns.ts src/components/agent/useAgentTurns.spec.ts
git commit -m "feat(agent-ui): add useAgentTurns composable for turn grouping

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Store fixes — live tool materialization + segment flush (TDD)

Extend `frontend/src/stores/agent.spec.ts`. Two fixes: `onAgentTool` materializes tool messages; `onAgentMessage`/`onAgentDone` flush assistant text per-`message_id`.

**Files:**
- Modify: `frontend/src/stores/agent.ts`
- Test: `frontend/src/stores/agent.spec.ts`

- [ ] **Step 1: Add the failing tests**

In `frontend/src/stores/agent.spec.ts`, add these tests inside the existing `describe('useAgentStore', ...)` block (after the existing `runTool` test, before the closing `})`):

```ts
  it('flushes an assistant segment when message_id changes', () => {
    const s = useAgentStore()
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'hello' })
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm2', delta_text: 'world' })
    expect(s.messages).toHaveLength(1)
    expect(s.messages[0]).toMatchObject({ id: 'm1', role: 'assistant', content: 'hello' })
    expect(s.streamingText).toBe('world')
  })

  it('materializes a read tool started -> succeeded (no message_id, matched by order)', () => {
    const s = useAgentStore()
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'reading' })
    s.onAgentTool({ type: 'agent_tool', conversation_id: 'c1', tool_name: 'list_tasks', args: {}, status: 'started' })
    s.onAgentTool({ type: 'agent_tool', conversation_id: 'c1', tool_name: 'list_tasks', args: {}, status: 'succeeded', result: [{ id: 1 }, { id: 2 }] })
    const tools = s.messages.filter((m) => m.tool_name === 'list_tasks')
    expect(tools).toHaveLength(1)
    expect(tools[0].tool_status).toBe('succeeded')
    expect(tools[0].tool_result).toContain('2') // serialized array
  })

  it('upserts a write tool by message_id and clears pendingConfirm on resolution', () => {
    const s = useAgentStore()
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'go' })
    s.onAgentTool({ type: 'agent_tool', conversation_id: 'c1', message_id: 'tc1', tool_name: 'create_task', args: { title: 'x' }, status: 'pending_confirmation', preview: { title: 'x' } })
    expect(s.pendingConfirm?.messageId).toBe('tc1')
    s.onAgentTool({ type: 'agent_tool', conversation_id: 'c1', message_id: 'tc1', tool_name: 'create_task', args: { title: 'x' }, status: 'succeeded', result: { id: 9 } })
    const tc = s.messages.find((m) => m.id === 'tc1')
    expect(tc?.tool_status).toBe('succeeded')
    expect(s.pendingConfirm).toBeNull()
  })

  it('flushes remaining streaming text on agent_done', () => {
    const s = useAgentStore()
    s.onAgentMessage({ type: 'agent_message', conversation_id: 'c1', message_id: 'm1', delta_text: 'final' })
    s.onAgentDone({ type: 'agent_done', conversation_id: 'c1', finish_reason: 'stop' })
    expect(s.streamingText).toBe('')
    expect(s.messages.find((m) => m.id === 'm1')).toMatchObject({ role: 'assistant', content: 'final' })
    expect(s.isThinking).toBe(false)
  })
```

- [ ] **Step 2: Run tests to verify the new ones fail**

```bash
cd frontend && npx vitest run src/stores/agent.spec.ts
```

Expected: the 4 new tests FAIL (tools/segments not materialized/flushed). The pre-existing "appends streaming tokens" test still PASSES (same-`message_id` deltas still accumulate).

- [ ] **Step 3: Implement the store fixes**

In `frontend/src/stores/agent.ts`:

3a. Add a module-level id counter near the top (after the `unwrap` helper, before `defineStore`):

```ts
// Monotonic id for locally-synthesized messages (read-tool events carry no id).
let localSeq = 0
const nextLocalId = (prefix: string) => `${prefix}-${++localSeq}`
```

3b. Add a `flushStreaming` action and rewrite the three WS handlers. Replace the existing `onAgentMessage`, `onAgentTool`, and `onAgentDone` actions with:

```ts
    onAgentMessage(e: Extract<AgentWsEvent, { type: 'agent_message' }>) {
      if (this.currentConvId !== null && e.conversation_id !== this.currentConvId) return
      // A new message_id means the prior assistant segment is complete.
      if (this.streamingMessageId !== null && this.streamingMessageId !== e.message_id && this.streamingText) {
        this.flushStreaming()
      }
      this.streamingMessageId = e.message_id
      this.streamingText += e.delta_text
    },
    onAgentTool(e: Extract<AgentWsEvent, { type: 'agent_tool' }>) {
      if (this.currentConvId !== null && e.conversation_id !== this.currentConvId) return
      // Tool arrives after the preceding text segment: flush it so order is text → tool.
      this.flushStreaming()
      if (e.status === 'pending_confirmation') {
        this.pendingConfirm = {
          messageId: e.message_id!, toolName: e.tool_name, args: e.args, preview: e.preview,
        }
      } else if (this.pendingConfirm && e.message_id && e.message_id === this.pendingConfirm.messageId) {
        this.pendingConfirm = null
      }
      const base = {
        role: 'tool_call' as const,
        conversation_id: this.currentConvId!,
        content: '',
        tool_name: e.tool_name,
        tool_args: safeStringify(e.args),
        tool_status: e.status,
        created_at: new Date().toISOString(),
      }
      if (e.message_id) {
        const idx = this.messages.findIndex((m) => m.id === e.message_id)
        const msg: AgentMessage = {
          id: e.message_id, ...base,
          tool_result: e.result != null ? safeStringify(e.result) : (e.error ? `{"error":${JSON.stringify(e.error)}}` : undefined),
        }
        if (idx >= 0) this.messages[idx] = { ...this.messages[idx], ...msg }
        else this.messages.push(msg)
      } else {
        // Read tool: started creates a row; a terminal status updates the most recent
        // same-named row still in 'started' (backend runs tools serially within a turn).
        if (e.status === 'started') {
          this.messages.push({ id: nextLocalId('tool'), ...base })
        } else {
          for (let i = this.messages.length - 1; i >= 0; i--) {
            const m = this.messages[i]
            if (m.tool_name === e.tool_name && m.tool_status === 'started') {
              this.messages[i] = {
                ...m, tool_status: e.status,
                tool_result: e.result != null ? safeStringify(e.result) : (e.error ? `{"error":${JSON.stringify(e.error)}}` : m.tool_result),
              }
              break
            }
          }
        }
      }
    },
    onAgentDone(e: Extract<AgentWsEvent, { type: 'agent_done' }>) {
      if (this.currentConvId !== null && e.conversation_id !== this.currentConvId) return
      this.flushStreaming()
      this.isThinking = false
    },
    flushStreaming() {
      if (this.streamingText) {
        this.messages.push({
          id: this.streamingMessageId || nextLocalId('ast'),
          conversation_id: this.currentConvId!,
          role: 'assistant',
          content: this.streamingText,
          created_at: new Date().toISOString(),
        })
      }
      this.streamingText = ''
      this.streamingMessageId = null
    },
```

3c. Add the `safeStringify` helper at module scope (near `unwrap`):

```ts
function safeStringify(v: unknown): string {
  try { return JSON.stringify(v) } catch { return String(v) }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd frontend && npx vitest run src/stores/agent.spec.ts
```

Expected: PASS (all, including the pre-existing 3 + 4 new = 7).

- [ ] **Step 5: Commit**

```bash
cd frontend && git add src/stores/agent.ts src/stores/agent.spec.ts
git commit -m "fix(agent): materialize tool events live + flush assistant segments per message_id

Previously onAgentTool dropped events and onAgentDone collapsed the whole
turn into one message, losing text<->tool interleaving in the live view.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 6: `ToolRow.vue` — smart-summary tool row with inline confirm (TDD)

Replaces `ToolCard.vue`. Keeps ToolCard's perm/status semantics; adds smart summary, collapsible JSON, and absorbs `ToolConfirmDialog`'s inline confirm + preview.

**Files:**
- Create: `frontend/src/components/agent/ToolRow.vue`
- Test: `frontend/src/components/agent/ToolRow.spec.ts` (migrated from `ToolCard.spec.ts`)

- [ ] **Step 1: Write the failing test**

Create `frontend/src/components/agent/ToolRow.spec.ts`:

```ts
import { describe, it, expect, beforeEach, vi } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import ToolRow from './ToolRow.vue'
import { useAgentStore } from '@/stores/agent'

const baseMsg: any = {
  id: 'm1', conversation_id: 'c1', role: 'tool_call', content: '',
  tool_name: 'list_tasks', tool_args: '{}', tool_status: 'succeeded', created_at: '',
}

describe('ToolRow', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('read succeeded gets read class', () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_status: 'succeeded' } }, global: { plugins: [ElementPlus] } })
    expect(w.classes()).toContain('read')
  })
  it('write pending shows write class + inline confirm buttons', () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_name: 'create_task', tool_args: '{"title":"x"}', tool_status: 'pending_confirmation' } }, global: { plugins: [ElementPlus] } })
    expect(w.classes()).toContain('write')
    expect(w.find('[data-testid="tool-confirm-approve"]').exists()).toBe(true)
  })
  it('danger pending shows danger class', () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_name: 'delete_task', tool_status: 'pending_confirmation' } }, global: { plugins: [ElementPlus] } })
    expect(w.classes()).toContain('danger')
  })
  it('failed surfaces the error message', () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_status: 'failed', tool_result: '{"error":"oops"}' } }, global: { plugins: [ElementPlus] } })
    expect(w.text()).toContain('oops')
  })
  it('shows smart summary (count hint) for a succeeded read', () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_name: 'list_tasks', tool_result: '[1,2,3]' } }, global: { plugins: [ElementPlus] } })
    expect(w.text()).toContain('3 条任务')
  })
  it('toggles JSON detail on chevron click', async () => {
    const w = mount(ToolRow, { props: { message: { ...baseMsg, tool_args: '{"a":1}' } }, global: { plugins: [ElementPlus] } })
    expect(w.find('[data-testid="tool-json"]').exists()).toBe(false)
    await w.find('[data-testid="tool-toggle"]').trigger('click')
    expect(w.find('[data-testid="tool-json"]').exists()).toBe(true)
  })
  it('approve click calls store.confirmToolCall', async () => {
    const store = useAgentStore()
    const spy = vi.spyOn(store, 'confirmToolCall').mockResolvedValue(undefined as never)
    const w = mount(ToolRow, { props: { message: { ...baseMsg, id: 'tc1', tool_name: 'create_task', tool_args: '{"title":"x"}', tool_status: 'pending_confirmation' } }, global: { plugins: [ElementPlus] } })
    await w.find('[data-testid="tool-confirm-approve"]').trigger('click')
    expect(spy).toHaveBeenCalledWith('tc1', 'approve')
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/components/agent/ToolRow.spec.ts
```

Expected: FAIL — `Failed to resolve import "./ToolRow.vue"`.

- [ ] **Step 3: Write the implementation**

Create `frontend/src/components/agent/ToolRow.vue`:

```vue
<template>
  <div :class="['tool-row', perm, statusClass]" data-testid="tool-row">
    <span class="dot" />
    <div class="main">
      <div class="head">
        <code class="name">{{ message.tool_name }}</code>
        <span v-if="summary.argHint" class="arg">{{ summary.argHint }}</span>
        <span class="status" :class="statusClass">{{ statusLabel }}</span>
        <span v-if="summary.resultHint" class="res">{{ summary.resultHint }}</span>
        <button class="toggle" data-testid="tool-toggle" @click="open = !open">⌄</button>
      </div>
      <div v-if="message.tool_status === 'failed' && errorText" class="err">{{ errorText }}</div>

      <div v-if="message.tool_status === 'pending_confirmation'" class="confirm">
        <pre v-if="previewText" class="preview">{{ previewText }}</pre>
        <div class="actions">
          <el-button size="small" type="primary" data-testid="tool-confirm-approve" @click="decide('approve')">✓ 批准</el-button>
          <el-button size="small" data-testid="tool-confirm-reject" @click="decide('reject')">✕ 拒绝</el-button>
        </div>
      </div>

      <pre v-if="open" class="json" data-testid="tool-json">{{ prettyArgs }}<template v-if="message.tool_result">
{{ prettyResult }}</template></pre>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useAgentStore } from '@/stores/agent'
import type { AgentMessage } from '@/types'
import { classifyPermission, summarizeTool } from './toolFormatters'

const props = defineProps<{ message: AgentMessage }>()
const store = useAgentStore()
const open = ref(false)

const perm = computed(() => {
  if (props.message.tool_status === 'failed') return 'failed'
  return classifyPermission(props.message.tool_name)
})
const statusClass = computed(() => `s-${props.message.tool_status || 'unknown'}`)
const summary = computed(() => summarizeTool(props.message))

const STATUS_LABELS: Record<string, string> = {
  started: '执行中',
  pending_confirmation: '待确认',
  succeeded: '已执行',
  failed: '失败',
  rejected: '已取消',
}
const statusLabel = computed(() => STATUS_LABELS[props.message.tool_status || ''] || props.message.tool_status || '')

const errorText = computed(() => {
  const raw = props.message.tool_result ?? ''
  if (raw.startsWith('{')) {
    try {
      const p = JSON.parse(raw) as { error?: string; message?: string }
      if (typeof p.error === 'string') return p.error
      if (typeof p.message === 'string') return p.message
    } catch { /* fall through */ }
  }
  return raw
})

const previewText = computed(() => {
  const pc = store.pendingConfirm
  if (!pc || pc.messageId !== props.message.id) return ''
  const p = pc.preview
  if (p == null || p === '') return ''
  if (typeof p === 'string') return p
  try { return JSON.stringify(p, null, 2) } catch { return String(p) }
})

function pretty(s: string | undefined): string {
  if (!s) return ''
  try { return JSON.stringify(JSON.parse(s), null, 2) } catch { return s }
}
const prettyArgs = computed(() => pretty(props.message.tool_args))
const prettyResult = computed(() => pretty(props.message.tool_result))

function decide(d: 'approve' | 'reject') {
  store.confirmToolCall(props.message.id, d)
}
</script>

<style scoped>
.tool-row {
  position: relative;
  display: flex;
  gap: 8px;
  padding: 2px 0 2px 14px;
  border-left: 1.5px dashed var(--border-color);
  font-family: var(--font-mono);
  font-size: 12px;
}
.tool-row .dot {
  position: absolute;
  left: -6px;
  top: 6px;
  width: 9px;
  height: 9px;
  border-radius: 50%;
  background: var(--text-muted);
}
.tool-row.read .dot { background: var(--accent-sage); }
.tool-row.write .dot { background: var(--accent-gold); }
.tool-row.danger .dot { background: var(--accent-crimson); }
.tool-row.failed .dot { background: var(--accent-crimson); }
.tool-row.s-pending_confirmation .dot { background: var(--accent-gold); }
.tool-row.s-started .dot {
  background: var(--accent-gold);
  animation: trow-glow 1.2s infinite ease-out;
}
@keyframes trow-glow {
  0%, 100% { box-shadow: 0 0 0 0 rgba(214, 180, 90, 0.5); }
  50% { box-shadow: 0 0 0 4px rgba(214, 180, 90, 0); }
}
.main { flex: 1; min-width: 0; }
.head { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.name { color: var(--text-primary); font-size: 11.5px; letter-spacing: 0.02em; }
.arg { color: var(--text-muted); font-size: 10px; }
.status { font-size: 9px; color: var(--text-muted); }
.status.s-succeeded { color: var(--accent-sage); }
.status.s-failed { color: var(--accent-crimson); }
.status.s-rejected { color: var(--text-muted); }
.status.s-pending_confirmation { color: var(--accent-gold); }
.res { color: var(--accent-sage); font-size: 9.5px; }
.toggle {
  margin-left: auto;
  background: transparent;
  border: 0;
  color: var(--text-muted);
  cursor: pointer;
  font-size: 11px;
  padding: 0 2px;
}
.err { color: var(--accent-crimson); font-size: 10.5px; margin-top: 2px; }
.confirm { margin-top: 6px; display: flex; flex-direction: column; gap: 6px; }
.preview {
  margin: 0;
  padding: 6px 8px;
  background: rgba(214, 180, 90, 0.06);
  border: 1px solid rgba(214, 180, 90, 0.3);
  border-radius: var(--radius-sm);
  color: var(--text-secondary);
  font-size: 10px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 140px;
  overflow: auto;
}
.actions { display: flex; gap: 6px; }
.json {
  margin: 4px 0 0;
  padding: 6px 8px;
  background: rgba(239, 231, 215, 0.04);
  border-radius: 4px;
  color: var(--text-secondary);
  font-size: 10px;
  white-space: pre-wrap;
  word-break: break-all;
  max-height: 180px;
  overflow: auto;
}
</style>
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd frontend && npx vitest run src/components/agent/ToolRow.spec.ts
```

Expected: PASS (8 tests).

- [ ] **Step 5: Commit**

```bash
cd frontend && git add src/components/agent/ToolRow.vue src/components/agent/ToolRow.spec.ts
git commit -m "feat(agent-ui): add ToolRow (smart summary + collapsible JSON + inline confirm)

Replaces ToolCard; absorbs ToolConfirmDialog's inline confirm + preview.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 7: `AgentTurn.vue` — avatar + segments + live indicator (TDD)

**Files:**
- Create: `frontend/src/components/agent/AgentTurn.vue`
- Test: `frontend/src/components/agent/AgentTurn.spec.ts`

- [ ] **Step 1: Write the failing test**

Create `frontend/src/components/agent/AgentTurn.spec.ts`:

```ts
import { describe, it, expect, beforeEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import ElementPlus from 'element-plus'
import AgentTurn from './AgentTurn.vue'
import type { AgentMessage } from '@/types'
import type { Turn } from './useAgentTurns'

function user(id: string, content: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'user', content, created_at: '' }
}
function ast(id: string, content: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'assistant', content, created_at: '' }
}
function tool(id: string): AgentMessage {
  return { id, conversation_id: 'c', role: 'tool_call', content: '', tool_name: 'list_tasks', tool_args: '{}', tool_status: 'succeeded', tool_result: '[1]', created_at: '' }
}

describe('AgentTurn', () => {
  beforeEach(() => setActivePinia(createPinia()))

  it('renders the user bubble on the right', () => {
    const turn: Turn = { id: 'u1', user: user('u1', 'hi'), segments: [] }
    const w = mount(AgentTurn, { props: { turn }, global: { plugins: [ElementPlus] } })
    expect(w.find('.msg.user').exists()).toBe(true)
    expect(w.text()).toContain('hi')
  })

  it('renders text + tool segments inside the agent block', () => {
    const turn: Turn = { id: 'u1', user: user('u1', 'hi'), segments: [
      { kind: 'text', message: ast('a1', '**ok**') },
      { kind: 'tool', message: tool('t1') },
    ] }
    const w = mount(AgentTurn, { props: { turn }, global: { plugins: [ElementPlus] } })
    expect(w.findComponent({ name: 'MarkdownText' }).exists() || w.find('.md').exists()).toBe(true)
    expect(w.find('[data-testid="tool-row"]').exists()).toBe(true)
  })

  it('shows the pulsing indicator when live has no text', () => {
    const turn: Turn = { id: 'u1', user: user('u1', 'hi'), segments: [], live: { text: '' } }
    const w = mount(AgentTurn, { props: { turn }, global: { plugins: [ElementPlus] } })
    expect(w.find('.pulse').exists()).toBe(true)
  })

  it('shows streaming text + caret when live has text', () => {
    const turn: Turn = { id: 'u1', user: user('u1', 'hi'), segments: [], live: { text: 'think' } }
    const w = mount(AgentTurn, { props: { turn }, global: { plugins: [ElementPlus] } })
    expect(w.find('.live-stream').exists()).toBe(true)
    expect(w.find('.caret').exists()).toBe(true)
  })
})
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd frontend && npx vitest run src/components/agent/AgentTurn.spec.ts
```

Expected: FAIL — `Failed to resolve import "./AgentTurn.vue"`.

- [ ] **Step 3: Write the implementation**

Create `frontend/src/components/agent/AgentTurn.vue`:

```vue
<template>
  <div class="turn">
    <div v-if="turn.user" class="msg user">
      <div class="role">YOU</div>
      <div class="bubble">{{ turn.user.content }}</div>
    </div>

    <div v-if="turn.segments.length || turn.live" class="agent-block">
      <div class="ava">A</div>
      <div class="agent-body">
        <template v-for="s in turn.segments" :key="s.message.id">
          <MarkdownText v-if="s.kind === 'text'" :content="s.message.content" class="bubble" />
          <ToolRow v-else :message="s.message" />
        </template>

        <div v-if="turn.live && turn.live.text" class="bubble live-stream">
          <MarkdownText :content="turn.live.text" />
          <span class="caret" />
        </div>
        <div v-else-if="turn.live" class="pulse"><span /></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import MarkdownText from './MarkdownText.vue'
import ToolRow from './ToolRow.vue'
import type { Turn } from './useAgentTurns'
defineProps<{ turn: Turn }>()
</script>

<style scoped>
.turn { display: flex; flex-direction: column; gap: 10px; }
.msg { display: flex; flex-direction: column; gap: 4px; }
.msg.user { align-items: flex-end; }
.role {
  font-family: var(--font-mono);
  font-size: 10px;
  letter-spacing: 0.12em;
  text-transform: uppercase;
  color: var(--text-muted);
}
.bubble {
  background: var(--bg-secondary);
  border: 1px solid var(--border-color);
  border-radius: var(--radius-sm);
  padding: 10px 14px;
  color: var(--text-primary);
  font-size: 13px;
  line-height: 1.6;
  max-width: 90%;
}
.msg.user .bubble {
  background: var(--accent-primary);
  color: var(--bg-primary);
  border-color: transparent;
}
.agent-block { display: flex; gap: 10px; }
.ava {
  width: 26px;
  height: 26px;
  flex-shrink: 0;
  border-radius: 50%;
  background: var(--bg-elevated);
  border: 1px solid rgba(230, 162, 60, 0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  font-family: var(--font-display);
  font-size: 13px;
  color: var(--accent-primary);
}
.agent-body {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 2px;
}
.agent-body :deep(.tool-row) { margin-left: 2px; }
.live-stream { display: flex; flex-direction: column; }
.caret {
  display: inline-block;
  width: 6px;
  height: 13px;
  background: var(--accent-primary);
  margin-top: 2px;
  animation: caret-blink 1s steps(1) infinite;
}
@keyframes caret-blink { 0%, 100% { opacity: 1; } 50% { opacity: 0; } }
.pulse { display: flex; align-items: center; padding: 6px 0; }
.pulse span {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--accent-primary);
  animation: pulse-glow 1.2s infinite ease-out;
}
@keyframes pulse-glow {
  0%, 100% { box-shadow: 0 0 0 0 rgba(230, 162, 60, 0.5); }
  50% { box-shadow: 0 0 0 5px rgba(230, 162, 60, 0); }
}
</style>
```

- [ ] **Step 4: Run test to verify it passes**

```bash
cd frontend && npx vitest run src/components/agent/AgentTurn.spec.ts
```

Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
cd frontend && git add src/components/agent/AgentTurn.vue src/components/agent/AgentTurn.spec.ts
git commit -m "feat(agent-ui): add AgentTurn (avatar + segment grouping + live indicator)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 8: Rework `AgentMessageList.vue` to render turns (TDD)

Keep the existing auto-scroll logic; swap the flat `v-for` + synthetic streaming bubble for `useAgentTurns` + `<AgentTurn>`.

**Files:**
- Modify: `frontend/src/components/agent/AgentMessageList.vue`

- [ ] **Step 1: Read the current file to preserve the scroll logic**

```bash
cd frontend && cat src/components/agent/AgentMessageList.vue
```

Note the `scrollParent` / `isNearBottom` / `stickToBottom` logic and the three `watch` calls — they are kept verbatim; only the `watch` signals and the `<template>` change.

- [ ] **Step 2: Replace the file contents**

Overwrite `frontend/src/components/agent/AgentMessageList.vue` with:

```vue
<template>
  <div ref="root" class="messages" data-testid="messages">
    <AgentTurn v-for="t in turns" :key="t.id" :turn="t" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import type { AgentMessage } from '@/types'
import AgentTurn from './AgentTurn.vue'
import { useAgentTurns } from './useAgentTurns'

const props = defineProps<{
  messages: AgentMessage[]
  streamingText: string
  isThinking: boolean
}>()

const turns = useAgentTurns(
  computed(() => props.messages),
  computed(() => props.streamingText),
  computed(() => props.isThinking),
)

const root = ref<HTMLElement | null>(null)

// `.messages` itself isn't the scroll container — Element Plus' `.el-drawer__body`
// is. Walk up to the first scrollable ancestor so this stays decoupled from the
// drawer's layout and class names.
function scrollParent(): HTMLElement | null {
  let node: HTMLElement | null = root.value
  while (node) {
    const { overflowY } = window.getComputedStyle(node)
    if (overflowY === 'auto' || overflowY === 'scroll') return node
    node = node.parentElement
  }
  return null
}

const STICK_THRESHOLD_PX = 80

function isNearBottom(): boolean {
  const sc = scrollParent()
  if (!sc) return true
  return sc.scrollHeight - sc.scrollTop - sc.clientHeight <= STICK_THRESHOLD_PX
}

function stickToBottom() {
  const sc = scrollParent()
  if (sc) sc.scrollTop = sc.scrollHeight
}

// Conversation switched/cleared → array replaced. Jump to bottom unconditionally.
watch(
  () => props.messages,
  () => nextTick(stickToBottom),
  { flush: 'post' },
)

// New committed message arrived. User's own input always pins; agent/tool replies
// follow only if the user is still watching the tail.
watch(
  () => props.messages.length,
  (newLen, oldLen) => {
    if (newLen <= (oldLen ?? 0)) return
    const last = props.messages[newLen - 1]
    if (last?.role === 'user' || isNearBottom()) nextTick(stickToBottom)
  },
  { flush: 'post' },
)

// Streaming tokens + typing indicator: follow only while already at the bottom.
watch(
  () => [props.streamingText, props.isThinking],
  () => {
    if (isNearBottom()) nextTick(stickToBottom)
  },
  { flush: 'post' },
)

onMounted(() => nextTick(stickToBottom))
</script>

<style scoped>
.messages {
  display: flex;
  flex-direction: column;
  gap: 18px;
  padding: 4px 0 16px;
}
</style>
```

- [ ] **Step 3: Update `AgentDrawer.spec.ts` to match the new root testid**

The drawer test asserts `.messages` exists. That class is still present, so it passes as-is — but add a tighter assertion. In `frontend/src/components/agent/AgentDrawer.spec.ts`, in the "renders header/input/messages when open and configured" test, after the existing `expect(w.find('.messages').exists()).toBe(true)` line, no change is required. Verify by running:

```bash
cd frontend && npx vitest run src/components/agent/AgentDrawer.spec.ts
```

Expected: PASS (3 tests).

- [ ] **Step 4: Run the drawer spec**

```bash
cd frontend && npx vitest run src/components/agent/AgentDrawer.spec.ts
```

Expected: PASS (3 tests). (AgentMessageList has no spec of its own; its behavior is covered by the AgentTurn / useAgentTurns specs.)

- [ ] **Step 5: Commit**

```bash
cd frontend && git add src/components/agent/AgentMessageList.vue
git commit -m "refactor(agent-ui): render AgentTurn list via useAgentTurns; drop synthetic bubble

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 9: Remove `ToolCard` + `ToolConfirmDialog`; wire `AgentDrawer`

**Files:**
- Modify: `frontend/src/components/agent/AgentDrawer.vue`
- Delete: `frontend/src/components/agent/ToolCard.vue`, `frontend/src/components/agent/ToolConfirmDialog.vue`, `frontend/src/components/agent/ToolCard.spec.ts`

- [ ] **Step 1: Drop `ToolConfirmDialog` from `AgentDrawer.vue`**

In `frontend/src/components/agent/AgentDrawer.vue`:

1a. Remove the import line:
```ts
import ToolConfirmDialog from './ToolConfirmDialog.vue'
```
1b. Remove the usage in the template:
```html
      <ToolConfirmDialog v-if="store.pendingConfirm" />
```
(Leave `<AgentMessageList ... />` exactly as is — `ToolRow` now renders pending confirmation inline.)

- [ ] **Step 2: Delete the obsolete files**

```bash
cd frontend && git rm src/components/agent/ToolCard.vue src/components/agent/ToolConfirmDialog.vue src/components/agent/ToolCard.spec.ts
```

- [ ] **Step 3: Type check + run the agent specs**

```bash
cd frontend && npx vue-tsc --noEmit && npx vitest run src/components/agent/ src/stores/agent.spec.ts
```

Expected: vue-tsc exits 0; all agent specs PASS (MarkdownText, useAgentTurns, toolFormatters, ToolRow, AgentTurn, AgentDrawer) + agent store spec PASS.

- [ ] **Step 4: Commit**

```bash
cd frontend && git add src/components/agent/AgentDrawer.vue
git commit -m "refactor(agent-ui): remove ToolCard + ToolConfirmDialog (folded into ToolRow)

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 10: Final verification — type check, full suite, manual E2E

**Files:** none (verification only)

- [ ] **Step 1: Full type check**

```bash
cd frontend && npx vue-tsc --noEmit
```

Expected: exits 0, no errors.

- [ ] **Step 2: Full unit suite**

```bash
cd frontend && npm run test:run
```

Expected: all agent tests PASS. Compare any other failures against the known-red baseline (see memory `frontend-test-baseline-green.md`: ~13 unit + 22 e2e are pre-existing on `main`). Only NEW failures introduced by this work are blockers — none expected outside the agent files touched here.

- [ ] **Step 3: Manual E2E (real app)**

```bash
# from repo root
lsof -ti:8080 | xargs kill -9 2>/dev/null; cd backend && go run cmd/server/main.go &
cd ../frontend && npm run dev
```

Open the app, open the Agent drawer (🤖), ensure AI is configured (Settings → API Key or CLI provider), and send: `安排今天的日程`. Verify:

- Idle state shows a single pulsing amber dot.
- Assistant text streams with a blinking caret and renders Markdown (bold, lists, code block with copy button, tables if present).
- The tool timeline lights up live: `list_tasks` (sage ✓ with "N 条任务" hint) then `generate_schedule`/`create_schedule` (gold, "执行中…" then ✓ "N 个时段").
- If a write tool needs confirmation, the approve/reject buttons + preview appear inline on that tool row.
- Scrolling up stops auto-follow; sending a new message snaps back to bottom.

- [ ] **Step 4: Stop the dev servers**

```bash
lsof -ti:8080 | xargs kill -9 2>/dev/null
# stop the vite dev server (Ctrl-C in its terminal)
```

- [ ] **Step 5: Final commit if any verification tweaks were needed**

If steps 1–2 surfaced fixes, commit them. Otherwise this task produces no commit. The feature branch `evolve/agent-output-display` is now ready for review/merge.

---

## Notes for the implementer

- **Read the spec first:** `docs/superpowers/specs/2026-08-09-agent-output-display-design.md`. It explains *why* the two store fixes are needed (the live tool timeline is impossible without them).
- **TDD discipline:** every code task writes the failing test first, watches it fail, then implements. Do not skip the "run to verify it fails" step — it catches typos in test imports.
- **`marked.parse` is synchronous** by default and returns a `string`; the `as string` cast is intentional. Do not enable `async: true`.
- **DOMPurify in jsdom:** Vitest's jsdom environment provides `window`, which DOMPurify auto-detects. No special setup needed.
- **Read-tool correlation** (`onAgentTool` for events with no `message_id`) relies on the backend executing tools serially within a turn (`backend/internal/agent/service.go:149`). If the backend ever parallelizes tool calls, add stable ids to `agent_tool` events (listed as a follow-up in the spec) and key on those instead.
- **testids** are the contract for the E2E layer: `tool-row`, `tool-toggle`, `tool-json`, `tool-confirm-approve`, `tool-confirm-reject`, `md`, `messages`, `agent-input`, `agent-drawer`.
