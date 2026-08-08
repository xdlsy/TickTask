# config

## Responsibility [~ inferred]

YAML configuration loader providing a single `Config` struct with typed sub-configs for server, database, CORS, and AI settings. Loads from file. The AI API key is no longer part of this config — it is entered via the Settings page and stored encrypted in SQLite.

Key types:
- `Config` — top-level struct aggregating `ServerConfig`, `DatabaseConfig`, `CORSConfig`, `AIConfig`
- `ServerConfig` — host, port, mode (debug/release)
- `AIConfig` — provider, base URL, model, timeout (no API key)

Key functions:
- `Load(path string) (*Config, error)` — reads YAML file
- `LoadDefault() *Config` — sensible defaults (port 8080, gpt-4o-mini, 30s timeout)

## Conventions [~ inferred]

- API Key is no longer configured via yaml/env. Users enter it in the Settings page; it is stored encrypted in SQLite. `TT_AI_API_KEY` is read once at startup by `cmd/server/main.go` for the legacy one-time migration, then ignored.
- Defaults are hardcoded in `LoadDefault()`, not in a separate defaults file
- YAML struct tags map directly to `configs/config.yaml` keys

## Dependencies [✓ auto]

- Depends on: `gopkg.in/yaml.v3`, `os`, `time` (standard library)
- Depended on by: `cmd/server/main.go` (loads config at startup)
