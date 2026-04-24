# gosundheit

<!-- archie:ai-start -->

> Thin adapter that bridges the go-sundheit health check library to OpenMeter's structured slog logger. It implements health.CheckListener so health check lifecycle events (register, start, complete) emit structured log lines at appropriate levels.

## Patterns

**Implement health.CheckListener via unexported struct** — checkListener is unexported; consumers receive a health.CheckListener interface from NewLogger. Do not expose the concrete type. (`listener := gosundheit.NewLogger(logger)
healthSvc.WithCheckListener(listener)`)
**Warn on error, Debug on success** — OnCheckRegistered and OnCheckCompleted log at slog.Warn when result.Error != nil and at slog.Debug otherwise. Maintain this severity split for any new listener methods. (`// Error path -> logger.Warn("health check failed", slog.String("check", name), slog.Any("error", result.Error))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `logger.go` | Sole file; declares checkListener and NewLogger. Only health.CheckListener interface is implemented. | go-sundheit's CheckListener interface may add methods in future versions — breaking the compile if not updated here. |

## Anti-Patterns

- Logging at Error level for health check failures — the library handles alerting; Warn is intentional to avoid log-level escalation
- Storing state in checkListener — it is a pure pass-through to the logger

<!-- archie:ai-end -->
