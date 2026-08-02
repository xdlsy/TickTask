# config

## Responsibility [~ inferred]

YAML configuration loader providing a single `Config` struct with typed sub-configs for server, database, CORS, and AI settings. Loads from file with environment variable override for the AI API key.

Key types:
- `Config` — top-level struct aggregating `ServerConfig`, `DatabaseConfig`, `CORSConfig`, `AIConfig`
- `ServerConfig` — host, port, mode (debug/release)
- `AIConfig` — provider, API key, base URL, model, timeout

Key functions:
- `Load(path string) (*Config, error)` — reads YAML file, applies env override (`TT_AI_API_KEY`)
- `LoadDefault() *Config` — sensible defaults (port 8080, gpt-4o-mini, 30s timeout)

## Conventions [~ inferred]

- Environment variable `TT_AI_API_KEY` overrides config file value
- Defaults are hardcoded in `LoadDefault()`, not in a separate defaults file
- YAML struct tags map directly to `configs/config.yaml` keys
- AI API key stored in plaintext in config — never committed to git

## Dependencies [✓ auto]

- Depends on: `gopkg.in/yaml.v3`, `os`, `time` (standard library)
- Depended on by: `cmd/server/main.go` (loads config at startup)
