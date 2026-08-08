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
- Prompts request JSON-only output; callers strip markdown fences before
  parsing (e.g. `extractJSON` in `internal/agent/tools/classify_helpers.go`).

## Dependencies [✓ auto]

- Depends on: standard library (`net/http`, `encoding/json`, `context`, `os/exec`)
- Depended on by: `service/` and `internal/agent/` (via `LLMClient` interface),
  `cmd/server/main.go` (`NewClientFromSettings`)
