# gosundheit

<!-- archie:ai-start -->

> Thin adapter bridging the go-sundheit health-check library to OpenMeter's structured slog logger. Implements health.CheckListener so health-check lifecycle events emit structured log lines at Warn on error and Debug on success.

## Patterns

**Return health.CheckListener interface from constructor** — checkListener is unexported; consumers receive a health.CheckListener from NewLogger — do not expose the concrete struct. (`listener := gosundheit.NewLogger(logger)
healthSvc.WithCheckListener(listener)`)
**Warn on error, Debug on success** — OnCheckRegistered and OnCheckCompleted log at Warn when result.Error != nil and Debug otherwise; keep this split for any new listener method. (`if result.Error != nil { c.logger.Warn("health check failed", slog.String("check", name), slog.Any("error", result.Error)); return }
c.logger.Debug("health check completed", slog.String("check", name))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `logger.go` | Sole file: unexported checkListener + exported NewLogger; implements only health.CheckListener. | go-sundheit's CheckListener may add methods in future versions — this file must be updated or builds break. |

## Anti-Patterns

- Logging at Error level for health-check failures — Warn is intentional; the library handles alerting.
- Storing state in checkListener — it is a pure pass-through to the logger.
- Exposing checkListener as a concrete type outside the package.

## Decisions

- **Package wraps go-sundheit's CheckListener rather than using it directly in startup code.** — Centralizes log severity and field naming for health events; app/common wires it once so all binaries get uniform health-check logging.

<!-- archie:ai-end -->
