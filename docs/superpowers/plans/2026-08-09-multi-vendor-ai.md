# 多厂商 AI 模型扩展 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add DeepSeek / 通义千问 / 智谱 GLM / 月之暗面 Kimi / MiniMax as first-class vendor presets in the Settings → AI 智能助手 card, reusing the existing OpenAI- and Anthropic-compatible clients.

**Architecture:** Approach A — a frontend vendor-preset registry (`aiVendors.ts`) drives the provider dropdown, default base URLs, and model lists; the backend gains one pure function `protocolFor(provider)` that routes each vendor id to the OpenAI-, Anthropic-, or CLI-client. No new HTTP clients, no new endpoints, no data migration.

**Tech Stack:** Go 1.21 (standard `testing`) · Vue 3.5 `<script setup>` + Element Plus 2.8 · Vitest 2.1

**Spec:** `docs/superpowers/specs/2026-08-09-multi-vendor-ai-design.md`

**Branch:** `evolve/multi-vendor-ai` (already created; spec already committed at `5f35023`)

---

## File Structure

**Backend**
- Modify `backend/internal/ai/client.go` — add `protocolFor`; rewrite the `NewClientFromSettings` switch to route by protocol.
- Create `backend/internal/ai/provider_test.go` — table-driven tests for `protocolFor` + `NewClientFromSettings` routing (also the drift guard: every vendor id resolves).
- Modify `backend/internal/ai/AGENTS.md` — update the outdated module doc (currently claims only an OpenAI client exists).

**Frontend**
- Create `frontend/src/utils/aiVendors.ts` — `VendorPreset` type, `VENDOR_PRESETS` (8 entries), `getVendorPreset`, `DEFAULT_VENDOR`. Single source of truth for display metadata.
- Create `frontend/src/utils/aiVendors.spec.ts` — registry invariant tests.
- Modify `frontend/src/views/Settings.vue` — provider dropdown, `handleProviderChange`, `availableModels`, `modelPlaceholder`, base-URL field all derived from `VENDOR_PRESETS`; delete the hardcoded `openaiModels`/`anthropicModels` arrays.
- Modify `frontend/src/views/Settings.spec.ts` — add vendor-preset behavior tests.

**Untouched (per spec non-goals)**: `model.AISettings` struct, the three client implementations, API-key encryption / `api_key_preview`, TestConnection path, agent tool-calling flow.

---

## Task 1: Backend — vendor→protocol routing (TDD)

**Files:**
- Create: `backend/internal/ai/provider_test.go`
- Modify: `backend/internal/ai/client.go:585-612`

- [ ] **Step 1: Write the failing tests**

Create `backend/internal/ai/provider_test.go`:

```go
package ai

import (
	"testing"

	"ticktask/internal/model"
)

func TestProtocolFor(t *testing.T) {
	cases := []struct {
		provider string
		want     string
	}{
		{"openai", "openai"},
		{"deepseek", "openai"},
		{"qwen", "openai"},
		{"zhipu", "openai"},
		{"moonshot", "openai"},
		{"custom", "openai"},
		{"", "openai"},
		{"anthropic", "anthropic"},
		{"minimax", "anthropic"},
		{"claude", "cli"},
		{"cli", "cli"},
	}
	for _, c := range cases {
		if got := protocolFor(c.provider); got != c.want {
			t.Errorf("protocolFor(%q) = %q, want %q", c.provider, got, c.want)
		}
	}
}

func TestNewClientFromSettings_Routing(t *testing.T) {
	if got := NewClientFromSettings(nil); got != nil {
		t.Fatalf("nil settings: got %v, want nil", got)
	}

	cases := []struct {
		name     string
		provider string
		apiKey   string
		want     string // "openai" | "anthropic" | "cli" | "nil"
	}{
		{"openai-compatible vendor routes to OpenAIClient", "deepseek", "k", "openai"},
		{"anthropic-compatible vendor routes to AnthropicClient", "minimax", "k", "anthropic"},
		{"cli vendor routes to CLIClient", "claude", "k", "cli"},
		{"http provider with empty key returns nil", "openai", "", "nil"},
		{"anthropic provider with empty key returns nil", "minimax", "", "nil"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			s := &model.AISettings{Provider: c.provider, APIKey: c.apiKey, BaseURL: "http://x", Model: "m"}
			got := NewClientFromSettings(s)
			switch c.want {
			case "openai":
				if _, ok := got.(*OpenAIClient); !ok {
					t.Fatalf("got %T, want *OpenAIClient", got)
				}
			case "anthropic":
				if _, ok := got.(*AnthropicClient); !ok {
					t.Fatalf("got %T, want *AnthropicClient", got)
				}
			case "cli":
				if _, ok := got.(*CLIClient); !ok {
					t.Fatalf("got %T, want *CLIClient", got)
				}
			case "nil":
				if got != nil {
					t.Fatalf("got %T, want nil", got)
				}
			}
		})
	}
}
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd backend && go test -run 'TestProtocolFor|TestNewClientFromSettings_Routing' ./internal/ai/ -v`

Expected: FAIL — compile error `undefined: protocolFor` (the function does not exist yet).

- [ ] **Step 3: Implement `protocolFor` and rewrite the switch**

In `backend/internal/ai/client.go`, replace the existing `NewClientFromSettings` block (its doc comment + function, currently lines 585–612) with:

```go
// protocolFor maps a vendor id (AISettings.Provider) to one of the three
// implemented client protocols. Known vendors resolve to their protocol;
// unknown values — including "custom" and the empty string — fall back to
// the OpenAI-compatible client, matching historical behavior so existing
// stored settings keep working without migration.
//
// The frontend vendor-preset registry
// (frontend/src/utils/aiVendors.ts) is the display source of truth for vendor
// labels, default base URLs, and model presets; this function is the backend
// routing counterpart and must stay in sync with the preset list.
func protocolFor(provider string) string {
	switch provider {
	case "claude", "cli":
		return "cli"
	case "anthropic", "minimax":
		return "anthropic"
	default: // openai, deepseek, qwen, zhipu, moonshot, custom, "" → OpenAI-compatible
		return "openai"
	}
}

// NewClientFromSettings constructs the appropriate LLMClient for a given
// AISettings, routing by the vendor's protocol (see protocolFor). Returns nil
// if the provider needs an API key and none is set, so callers can distinguish
// "nothing configured" from "configured but rejected" by checking for nil.
//
// Extracted from cmd/server/main.go's constructLLMClient so other packages
// (notably the agent service's TestConnection temp-settings path) can build
// one-shot clients without import cycles through main. main.go's
// constructLLMClient now delegates to this function.
func NewClientFromSettings(settings *model.AISettings) LLMClient {
	if settings == nil {
		return nil
	}
	switch protocolFor(settings.Provider) {
	case "cli":
		return NewCLIClient()
	case "anthropic":
		if settings.APIKey == "" {
			return nil
		}
		return NewAnthropicClient(settings.APIKey, settings.BaseURL, settings.Model)
	default: // openai
		if settings.APIKey == "" {
			return nil
		}
		return NewOpenAIClient(settings.APIKey, settings.BaseURL, settings.Model)
	}
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd backend && go test -run 'TestProtocolFor|TestNewClientFromSettings_Routing' ./internal/ai/ -v`

Expected: PASS — both tests pass.

- [ ] **Step 5: Run the full `ai` package + agent tests (regression check)**

Run: `cd backend && go test ./internal/ai/... ./internal/agent/...`

Expected: PASS — no regressions. (The agent service's TestConnection path calls `NewClientFromSettings`; its existing tests assert the empty-key→nil contract, which is preserved.)

- [ ] **Step 6: Commit**

```bash
git add backend/internal/ai/client.go backend/internal/ai/provider_test.go
git commit -m "feat(ai): route vendor ids to protocol via protocolFor

Adds DeepSeek/Qwen/Zhipu/Moonshot (OpenAI-compatible) and MiniMax
(Anthropic-compatible) to NewClientFromSettings routing. No new clients;
behavior for openai/anthropic/custom/cli is unchanged.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 2: Frontend — vendor preset registry (TDD)

**Files:**
- Create: `frontend/src/utils/aiVendors.ts`
- Create: `frontend/src/utils/aiVendors.spec.ts`

- [ ] **Step 1: Write the failing tests**

Create `frontend/src/utils/aiVendors.spec.ts`:

```ts
import { describe, it, expect } from 'vitest'
import { VENDOR_PRESETS, getVendorPreset, DEFAULT_VENDOR } from './aiVendors'

describe('aiVendors registry', () => {
  it('exposes the 8 vendor presets in order', () => {
    expect(VENDOR_PRESETS.map((v) => v.id)).toEqual([
      'openai',
      'anthropic',
      'deepseek',
      'qwen',
      'zhipu',
      'moonshot',
      'minimax',
      'custom',
    ])
  })

  it('has unique ids', () => {
    const ids = VENDOR_PRESETS.map((v) => v.id)
    expect(new Set(ids).size).toBe(ids.length)
  })

  it('every preset except custom has a non-empty baseURL and at least one model', () => {
    for (const v of VENDOR_PRESETS) {
      if (v.id === 'custom') {
        expect(v.baseURL).toBe('')
        expect(v.models).toEqual([])
        continue
      }
      expect(v.baseURL.length).toBeGreaterThan(0)
      expect(v.models.length).toBeGreaterThan(0)
    }
  })

  it('routes MiniMax through the anthropic protocol with the documented base URL', () => {
    const mm = getVendorPreset('minimax')
    expect(mm?.protocol).toBe('anthropic')
    expect(mm?.baseURL).toBe('https://api.minimaxi.com/anthropic')
  })

  it('routes DeepSeek through the openai protocol', () => {
    expect(getVendorPreset('deepseek')?.protocol).toBe('openai')
  })

  it('returns undefined for unknown vendor ids', () => {
    expect(getVendorPreset('nope')).toBeUndefined()
  })

  it('defaults to openai', () => {
    expect(DEFAULT_VENDOR).toBe('openai')
  })
})
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd frontend && npx vitest run src/utils/aiVendors.spec.ts`

Expected: FAIL — `Failed to resolve import "./aiVendors"` (module does not exist yet).

- [ ] **Step 3: Implement the registry**

Create `frontend/src/utils/aiVendors.ts`:

```ts
/**
 * AI vendor preset registry — the frontend source of truth for the
 * 「AI 智能助手」provider dropdown. Each preset carries the display label,
 * the backend protocol it routes through (see backend protocolFor), the
 * default base URL, and a set of preset model names. The model <el-select>
 * stays allow-create/filterable so users can type any model name.
 *
 * BaseURL convention (must match the backend clients in
 * backend/internal/ai/client.go):
 *   - OpenAI-protocol vendors: baseURL INCLUDES the version path; the backend
 *     appends `/chat/completions`.
 *   - Anthropic-protocol vendors: baseURL does NOT include `/v1`; the backend
 *     appends `/v1/messages`.
 *
 * NOTE: base URLs / model names for DeepSeek/Qwen/Zhipu/Moonshot are
 * best-effort; verify against each vendor's current docs before relying on
 * them. MiniMax values are confirmed from the official Anthropic-compat docs.
 */
export type AIProtocol = 'openai' | 'anthropic'

export interface VendorPreset {
  id: string
  label: string
  protocol: AIProtocol
  baseURL: string
  models: string[]
}

export const VENDOR_PRESETS: VendorPreset[] = [
  {
    id: 'openai',
    label: 'OpenAI',
    protocol: 'openai',
    baseURL: 'https://api.openai.com/v1',
    models: ['gpt-4o-mini', 'gpt-4o', 'gpt-4.1-mini', 'gpt-4.1'],
  },
  {
    id: 'anthropic',
    label: 'Anthropic',
    protocol: 'anthropic',
    baseURL: 'https://api.anthropic.com',
    models: ['claude-sonnet-4-6', 'claude-3-5-sonnet-latest', 'claude-3-5-haiku-latest'],
  },
  {
    id: 'deepseek',
    label: 'DeepSeek',
    protocol: 'openai',
    baseURL: 'https://api.deepseek.com/v1',
    models: ['deepseek-chat', 'deepseek-reasoner'],
  },
  {
    id: 'qwen',
    label: '通义千问',
    protocol: 'openai',
    baseURL: 'https://dashscope.aliyuncs.com/compatible-mode/v1',
    models: ['qwen-plus', 'qwen-max', 'qwen-turbo', 'qwen-long'],
  },
  {
    id: 'zhipu',
    label: '智谱 GLM',
    protocol: 'openai',
    baseURL: 'https://open.bigmodel.cn/api/paas/v4',
    models: ['glm-4-plus', 'glm-4-air', 'glm-4-flash', 'glm-4.5'],
  },
  {
    id: 'moonshot',
    label: '月之暗面 Kimi',
    protocol: 'openai',
    baseURL: 'https://api.moonshot.cn/v1',
    models: ['moonshot-v1-8k', 'moonshot-v1-32k', 'moonshot-v1-128k', 'kimi-k2-0905-preview'],
  },
  {
    id: 'minimax',
    label: 'MiniMax',
    protocol: 'anthropic',
    baseURL: 'https://api.minimaxi.com/anthropic',
    models: ['MiniMax-M3', 'MiniMax-M2.7', 'MiniMax-M2.7-highspeed', 'MiniMax-M2.5'],
  },
  {
    id: 'custom',
    label: '自定义',
    protocol: 'openai',
    baseURL: '',
    models: [],
  },
]

export const DEFAULT_VENDOR = 'openai'

/** Look up a preset by vendor id. Returns undefined for unknown ids. */
export function getVendorPreset(id: string): VendorPreset | undefined {
  return VENDOR_PRESETS.find((v) => v.id === id)
}
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `cd frontend && npx vitest run src/utils/aiVendors.spec.ts`

Expected: PASS — all 7 tests pass.

- [ ] **Step 5: Commit**

```bash
git add frontend/src/utils/aiVendors.ts frontend/src/utils/aiVendors.spec.ts
git commit -m "feat(settings): add vendor preset registry

8 presets (OpenAI/Anthropic/DeepSeek/Qwen/Zhipu/Moonshot/MiniMax/custom)
with protocol, default base URL, and model lists. Frontend source of truth
for the AI provider dropdown.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 3: Frontend — wire Settings.vue to the registry (TDD)

**Files:**
- Modify: `frontend/src/views/Settings.vue` (template AI card + script `handleProviderChange`/`availableModels`/`modelPlaceholder`)
- Modify: `frontend/src/views/Settings.spec.ts` (add a describe block)

- [ ] **Step 1: Write the failing tests**

Append a new `describe` block to the end of `frontend/src/views/Settings.spec.ts` (after the closing `})` of the existing `'Settings.vue — API Key input'` describe on line 126):

```ts
describe('Settings.vue — vendor presets', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('handleProviderChange fills baseURL + default model for a known vendor', async () => {
    const { api } = await import('@/api/client')
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())

    const wrapper = mount(Settings, { global: { stubs: elStubs } })
    await flushPromises()

    ;(wrapper.vm as any).aiSettings.provider = 'minimax'
    ;(wrapper.vm as any).handleProviderChange()

    expect((wrapper.vm as any).aiSettings.base_url).toBe('https://api.minimaxi.com/anthropic')
    expect((wrapper.vm as any).aiSettings.model).toBe('MiniMax-M3')
    expect((wrapper.vm as any).availableModels).toEqual([
      'MiniMax-M3',
      'MiniMax-M2.7',
      'MiniMax-M2.7-highspeed',
      'MiniMax-M2.5',
    ])
  })

  it('handleProviderChange clears baseURL + model for custom', async () => {
    const { api } = await import('@/api/client')
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())

    const wrapper = mount(Settings, { global: { stubs: elStubs } })
    await flushPromises()

    ;(wrapper.vm as any).aiSettings.provider = 'custom'
    ;(wrapper.vm as any).handleProviderChange()

    expect((wrapper.vm as any).aiSettings.base_url).toBe('')
    expect((wrapper.vm as any).aiSettings.model).toBe('')
    expect((wrapper.vm as any).availableModels).toEqual([])
  })

  it('renders the vendor presets as provider options', async () => {
    const { api } = await import('@/api/client')
    ;(api.getSettings as any).mockResolvedValue(mockSettingsResponse())

    const wrapper = mount(Settings, { global: { stubs: elStubs } })
    await flushPromises()

    // The el-option stub exposes the id via the <option> value attribute (the
    // `label` prop is not rendered as slot text by this stub). Vendor ids are
    // unique to the provider dropdown, so membership is a robust check.
    const values = wrapper.findAll('option').map((o) => o.attributes('value'))
    for (const expected of [
      'openai',
      'anthropic',
      'deepseek',
      'qwen',
      'zhipu',
      'moonshot',
      'minimax',
      'custom',
    ]) {
      expect(values).toContain(expected)
    }
  })
})
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `cd frontend && npx vitest run src/views/Settings.spec.ts`

Expected: FAIL — the first two tests fail (current `handleProviderChange` does not know `minimax`, so `base_url` is not the MiniMax URL) and the third fails (only OpenAI/Anthropic/自定义 options exist).

- [ ] **Step 3: Wire the registry into Settings.vue — template**

In `frontend/src/views/Settings.vue`, replace the three hardcoded provider options. Replace this block (the 服务商 `<el-select>`):

```html
        <div class="form-item">
          <label>服务商</label>
          <el-select v-model="aiSettings.provider" @change="handleProviderChange" size="large">
            <el-option label="OpenAI" value="openai" />
            <el-option label="Anthropic" value="anthropic" />
            <el-option label="自定义" value="custom" />
          </el-select>
        </div>
```

with:

```html
        <div class="form-item">
          <label>服务商</label>
          <el-select v-model="aiSettings.provider" @change="handleProviderChange" size="large">
            <el-option v-for="v in VENDOR_PRESETS" :key="v.id" :label="v.label" :value="v.id" />
          </el-select>
        </div>
```

Then make the API 地址 field visible for all vendors (so any vendor can take a proxy URL). Replace this block:

```html
        <div class="form-item" v-if="aiSettings.provider === 'custom' || aiSettings.provider === 'openai'">
          <label>API 地址</label>
          <el-input
            v-model="aiSettings.base_url"
            placeholder="例如: https://api.openai.com/v1"
            size="large"
          />
          <div class="form-tip">
            {{ aiSettings.provider === 'openai' ? '默认使用 OpenAI 官方地址，如需代理可修改' : '请输入兼容 OpenAI API 的地址' }}
          </div>
        </div>
```

with:

```html
        <div class="form-item">
          <label>API 地址</label>
          <el-input
            v-model="aiSettings.base_url"
            :placeholder="baseUrlPlaceholder"
            size="large"
          />
          <div class="form-tip">{{ baseUrlHint }}</div>
        </div>
```

(The 模型 `<el-select>` block already loops over `availableModels` — leave it unchanged.)

- [ ] **Step 4: Wire the registry into Settings.vue — script**

First add the import. In the `<script setup>` import section, after the line `import type { PomodoroSettings, AISettings, ClearResult } from '@/types'`, add:

```ts
import { VENDOR_PRESETS, getVendorPreset } from '@/utils/aiVendors'
```

Then replace the hardcoded model arrays and the derived computeds/handler. Replace this whole block:

```ts
const openaiModels = ['gpt-4o-mini', 'gpt-4o', 'gpt-4-turbo', 'gpt-3.5-turbo']
const anthropicModels = ['claude-3-5-sonnet-latest', 'claude-3-5-haiku-latest', 'claude-3-opus-latest']

const availableModels = computed(() => {
  if (aiSettings.value.provider === 'openai') return openaiModels
  if (aiSettings.value.provider === 'anthropic') return anthropicModels
  return []
})

const modelPlaceholder = computed(() => {
  if (aiSettings.value.provider === 'openai') return '选择或输入 OpenAI 模型'
  if (aiSettings.value.provider === 'anthropic') return '选择或输入 Anthropic 模型'
  return '输入模型名称'
})
```

with:

```ts
const currentPreset = computed(() => getVendorPreset(aiSettings.value.provider))

const availableModels = computed(() => currentPreset.value?.models ?? [])

const modelPlaceholder = computed(() => {
  const label = currentPreset.value?.label
  return label ? `选择或输入 ${label} 模型` : '输入模型名称'
})

const baseUrlPlaceholder = computed(() =>
  currentPreset.value?.baseURL || '例如: https://api.openai.com/v1'
)

const baseUrlHint = computed(() =>
  aiSettings.value.provider === 'custom'
    ? '请输入兼容 OpenAI API 的地址'
    : '默认使用官方地址，如需代理可修改'
)
```

Then replace the `handleProviderChange` function:

```ts
function handleProviderChange() {
  if (aiSettings.value.provider === 'openai') {
    aiSettings.value.base_url = 'https://api.openai.com/v1'
    aiSettings.value.model = 'gpt-4o-mini'
  } else if (aiSettings.value.provider === 'anthropic') {
    aiSettings.value.base_url = 'https://api.anthropic.com/v1'
    aiSettings.value.model = 'claude-3-5-sonnet-latest'
  } else {
    aiSettings.value.base_url = ''
    aiSettings.value.model = ''
  }
}
```

with:

```ts
function handleProviderChange() {
  const preset = getVendorPreset(aiSettings.value.provider)
  if (preset) {
    aiSettings.value.base_url = preset.baseURL
    aiSettings.value.model = preset.models[0] ?? ''
  } else {
    aiSettings.value.base_url = ''
    aiSettings.value.model = ''
  }
}
```

> This also fixes the latent `/v1` double-path bug: the old code prefilled Anthropic's base URL as `https://api.anthropic.com/v1`, but `AnthropicClient` appends `/v1/messages`. The registry preset is `https://api.anthropic.com` (no `/v1`).

- [ ] **Step 5: Run the Settings tests to verify they pass**

Run: `cd frontend && npx vitest run src/views/Settings.spec.ts`

Expected: PASS — all tests pass (existing 3 + new 3).

- [ ] **Step 6: Type-check the frontend**

Run: `cd frontend && npx vue-tsc --noEmit`

Expected: no output, exit code 0. (Confirms no unused locals — the deleted `openaiModels`/`anthropicModels` are gone — and that `currentPreset`/`baseUrlPlaceholder`/`baseUrlHint` are valid.)

- [ ] **Step 7: Run the full frontend suite (regression)**

Run: `cd frontend && npx vitest run`

Expected: PASS — all test files green, including `Settings.test.ts` (which loads `provider: 'anthropic'` + `base_url: 'https://api.anthropic.com'`; both still valid).

- [ ] **Step 8: Commit**

```bash
git add frontend/src/views/Settings.vue frontend/src/views/Settings.spec.ts
git commit -m "feat(settings): drive AI provider dropdown from vendor presets

Provider select, handleProviderChange defaults, model list, and base-URL
hint now derive from VENDOR_PRESETS. Adds DeepSeek/Qwen/Zhipu/Moonshot/
MiniMax and fixes the Anthropic /v1 base-URL bug.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 4: Docs — update the `ai` module AGENTS.md

**Files:**
- Modify: `backend/internal/ai/AGENTS.md`

The current `AGENTS.md` is stale — it describes only `ChatCompletion` and an `OpenAIClient`. It is `@`-included into `.claude/rules/backend-internal-ai.md`, so updating it fixes the rule context too.

- [ ] **Step 1: Replace the file contents**

Overwrite `backend/internal/ai/AGENTS.md` with:

```markdown
# ai

## Responsibility [~ inferred]

LLM integration for intelligent task features. Provides an `LLMClient`
interface with two methods — `ChatCompletion` (single user-prompt convenience)
and `ChatWithTools` (multi-turn, function-calling form) — implemented by three
clients covering two HTTP protocol families plus a CLI shim.

Files:
- `client.go` — clients + unified message/tool types + vendor→protocol routing
  (`protocolFor`, `NewClientFromSettings`)
- `prompts.go` — Chinese-language prompt templates requesting JSON-only output
  for classification, scheduling, prioritization
- `work_log_prompts.go` — work-log structuring prompts

## Protocols & vendors [✓ auto]

Two real HTTP protocol families; most vendors reuse one of them:
- **OpenAI-compatible** (`OpenAIClient`): `POST {baseURL}/chat/completions`,
  `Authorization: Bearer`, `choices[].message`. Vendors: openai, deepseek, qwen,
  zhipu, moonshot, custom.
- **Anthropic-compatible** (`AnthropicClient`): `POST {baseURL}/v1/messages`,
  `x-api-key` + `anthropic-version`, `content[]` blocks. Vendors: anthropic,
  minimax (base URL `https://api.minimaxi.com/anthropic`).
- **CLI** (`CLIClient`): shells out to `claude -p`; tool-calling unsupported
  (`ErrFunctionCallNotSupported`). Vendors: claude, cli.

**BaseURL convention**: OpenAI-family baseURLs include the version path (the
client appends `/chat/completions`); Anthropic-family baseURLs do NOT include
`/v1` (the client appends `/v1/messages`). The frontend preset registry
(`frontend/src/utils/aiVendors.ts`) is the display source of truth for vendor
labels, default baseURLs, and model presets; the backend only needs the
vendor→protocol map (`protocolFor`).

## Conventions [~ inferred]

- `LLMClient` interface enables mock substitution in tests.
- `NewClientFromSettings(*model.AISettings)` routes by `protocolFor(provider)`,
  returns nil for nil settings or when an HTTP provider has no API key.
- `ChatWithTools` retries up to 3× on transient network errors / HTTP 5xx with
  exponential backoff (1s, 2s, 4s) on both HTTP clients; CLI is unsupported.
- Anthropic mapping: leading system message → top-level `system`; OpenAI
  `tools` → Anthropic `input_schema`; response `tool_use` blocks → unified
  `ToolCall`; `stop_reason` → OpenAI-style `finish_reason`.
- Prompts request JSON-only output; response parsing uses `extractJSON()` in
  `ai_service.go` to strip markdown fences.

## Dependencies [✓ auto]

- Depends on: standard library (`net/http`, `encoding/json`, `context`, `os/exec`)
- Depended on by: `service/` and `internal/agent/` (via `LLMClient` interface),
  `cmd/server/main.go` (`NewClientFromSettings`)
```

- [ ] **Step 2: Commit**

```bash
git add backend/internal/ai/AGENTS.md
git commit -m "docs(ai): refresh module doc for multi-vendor routing

Reflects the three clients, two protocol families, tool-calling support,
vendor→protocol routing, and the baseURL convention. Replaces the stale
OpenAI-only description.

Co-Authored-By: Claude <noreply@anthropic.com>"
```

---

## Task 5: Final verification

**Files:** none (verification only; commit only if something needed fixing)

- [ ] **Step 1: Full backend test suite**

Run: `cd backend && go test ./...`

Expected: PASS — all packages green, including `internal/ai` and `internal/agent`.

- [ ] **Step 2: Full frontend test suite + type check**

Run: `cd backend && cd ../frontend && npx vitest run && npx vue-tsc --noEmit`

Expected: all Vitest tests pass; `vue-tsc` exits 0 with no output.

- [ ] **Step 3: End-to-end smoke (manual, requires a real vendor API key)**

Per spec acceptance criterion 2–3. Skip if no vendor key available — the unit/integration tests + type check already cover routing correctness.

```bash
lsof -ti:8080 | xargs -r kill -9
cd backend && go run cmd/server/main.go &
```

Then in the frontend (refresh after backend is up):
1. Settings → AI 智能助手 → 服务商 dropdown shows 8 entries.
2. Select MiniMax → API 地址 auto-fills `https://api.minimaxi.com/anthropic`, model defaults to `MiniMax-M3`.
3. Paste a real MiniMax API key → 「测试连接」 → success toast.
4. Select Anthropic → API 地址 is `https://api.anthropic.com` (no double `/v1`) → 「测试连接」 succeeds.

Expected: connection succeeds for at least one vendor with a valid key; no 404/double-`/v1` errors.

- [ ] **Step 4: Commit only if smoke uncovered a fix**

If Step 3 surfaced a base-URL/model-name error that you corrected in `aiVendors.ts`, commit it:

```bash
git add frontend/src/utils/aiVendors.ts
git commit -m "fix(settings): correct <vendor> base URL / model name

Co-Authored-By: Claude <noreply@anthropic.com>"
```

Otherwise nothing to commit — the feature is complete.

---

## Done criteria

- [ ] `go test ./...` green; new `protocolFor` / routing tests pass.
- [ ] `npx vitest run` green; `npx vue-tsc --noEmit` clean.
- [ ] Settings provider dropdown lists 8 vendors; selecting one fills base URL + default model.
- [ ] MiniMax selectable and routes through the Anthropic client (covered by `TestNewClientFromSettings_Routing`).
- [ ] Anthropic base URL no longer has a double `/v1`.
- [ ] `backend/internal/ai/AGENTS.md` reflects the current module.
- [ ] All work committed on `evolve/multi-vendor-ai`.
