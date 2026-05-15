# gosundheit

<!-- archie:ai-start -->

> Thin adapter bridging the go-sundheit health check library to OpenMeter's structured slog logger. Implements health.CheckListener so health check lifecycle events (register, start, complete) emit structured log lines at Warn on error and Debug on success.

## Patterns

**Return health.CheckListener interface from constructor** — checkListener is unexported; consumers receive a health.CheckListener interface from NewLogger. Do not expose the concrete struct type. (`listener := gosundheit.NewLogger(logger)
healthSvc.WithCheckListener(listener)`)
**Warn on error, Debug on success** — OnCheckRegistered and OnCheckCompleted log at slog.Warn when result.Error != nil and at slog.Debug otherwise. Maintain this severity split for any new listener methods. (`if result.Error != nil {
    c.logger.Warn("health check failed", slog.String("check", name), slog.Any("error", result.Error))
    return
}
c.logger.Debug("health check completed", slog.String("check", name))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `logger.go` | Sole file; declares unexported checkListener and exported NewLogger. Only health.CheckListener interface is implemented. | go-sundheit's CheckListener interface may add methods in future versions — this file must be updated to remain a valid implementation or builds will break. |

## Anti-Patterns

- Logging at Error level for health check failures — Warn is intentional to avoid log-level escalation; the library handles alerting.
- Storing state in checkListener — it is a pure pass-through to the logger.
- Exposing checkListener as a concrete type outside the package.

## Decisions

- **Package wraps go-sundheit's CheckListener rather than using it directly in app startup code.** — Centralises log severity and field naming for health events; app/common wires it once so all binaries get uniform health check logging without duplicating the slog field calls.

<!-- archie:ai-end -->
