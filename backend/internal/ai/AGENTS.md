# ai

## Responsibility [~ inferred]

OpenAI-compatible LLM integration for intelligent task features. Provides an `LLMClient` interface with a single `ChatCompletion(ctx, prompt) (string, error)` method, implemented by `OpenAIClient` using standard `net/http`.

Files:
- `client.go` — HTTP client with configurable base URL + API key + model, error handling for non-200 status and API errors
- `prompts.go` — Chinese-language prompt templates requesting JSON-only output for classification, scheduling, and prioritization

## Conventions [~ inferred]

- `LLMClient` interface enables mock substitution in tests
- Constructor `NewOpenAIClient(apiKey, baseURL, model)` with sensible defaults (GPT-4o-mini)
- All requests use `context.Context` for cancellation/timeout
- Prompts explicitly request JSON-only output ("只返回 JSON，不要包含任何其他文字")
- Response parsing uses `extractJSON()` helper in `ai_service.go` to strip markdown code fences

## Dependencies [✓ auto]

- Depends on: standard library (`net/http`, `encoding/json`, `context`)
- Depended on by: `service/` (via `LLMClient` interface, not concrete type)
