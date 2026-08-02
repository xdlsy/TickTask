# logger

## Responsibility [✓ auto]

Global structured logger singleton using Go's `log/slog` package. Provides a package-level `Logger` variable initialized at load time, with a mode-based `Init` function for runtime reconfiguration.

Key exports:
- `Logger *slog.Logger` — global logger instance (text handler, stdout)
- `Init(mode string)` — reconfigures log level: debug mode -> `LevelDebug`, default -> `LevelInfo`

## Conventions [✓ auto]

- Uses `init()` for zero-config default initialization (info level, text format, stdout)
- `Init()` called from `main.go` after config load to apply debug/release mode
- Text handler format (not JSON) — suitable for local dev and single-instance deployment
- Package-level variable pattern — not goroutine-safe to call `Init()` concurrently, but safe for reads after init

## Dependencies [✓ auto]

- Depends on: `log/slog`, `os` (standard library only)
- Depended on by: all backend packages that need structured logging
