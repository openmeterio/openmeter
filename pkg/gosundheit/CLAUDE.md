# gosundheit

<!-- archie:ai-start -->

> Thin adapter that bridges AppsFlyer go-sundheit health checks to the project's `*slog.Logger`, providing a `health.CheckListener` that logs registration/start/completion. Wired by app/common.

## Patterns

**slog-backed CheckListener** — `NewLogger(logger *slog.Logger)` returns a `health.CheckListener`; failures (result.Error != nil) log at Warn, successes at Debug, using structured slog.String/slog.Any fields (`c.logger.Warn("health check failed", slog.String("check", name), slog.Any("error", result.Error))`)
**Injected logger, no default fallback** — The logger is a required constructor argument — consistent with the repo rule against slog.Default() fallbacks (`func NewLogger(logger *slog.Logger) health.CheckListener { return checkListener{logger: logger} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `logger.go` | Whole package: `checkListener` implementing OnCheckRegistered/OnCheckStarted/OnCheckCompleted, constructed via `NewLogger` | OnCheckRegistered and OnCheckCompleted both branch on result.Error; keep failure logs at Warn and routine lifecycle at Debug to avoid log noise |

## Anti-Patterns

- Logging healthy check lifecycle at Info/Warn level (floods logs)
- Falling back to slog.Default() instead of requiring the injected logger

## Decisions

- **Adapter pattern over go-sundheit's CheckListener** — Keeps the third-party health library's logging routed through the app's structured slog logger without coupling other packages to go-sundheit

<!-- archie:ai-end -->
