# logging

<!-- archie:ai-start -->

> Single-file bridge package that adapts Benthos' service.Logger to logr.LogSink, enabling controller-runtime and klog to route through the same Benthos logger instance. No business logic — purely a logging adapter.

## Patterns

**Benthos-to-logr bridge via CtrlLogger** — CtrlLogger implements logr.LogSink by wrapping *service.Logger. Level 0 maps to Infof; any other level maps to Debugf. Error always calls Errorf. Construct via logr.New(&CtrlLogger{logger: logger}). (`logr.New(&CtrlLogger{logger: svc.Logger()})`)
**SetupKlog called once at startup** — SetupKlog sets the global klog logger via klog.SetLogger with a logr.Logger carrying component=kubernetes key. Must be called exactly once — klog global state is not re-entrant. (`logging.SetupKlog(svc.Logger())`)
**WithValues and WithName are intentional no-ops** — Both methods return the same receiver unchanged. Benthos uses format strings — structured kv pairs from controller-runtime are silently dropped by design. (`func (l *CtrlLogger) WithValues(keysAndValues ...any) logr.LogSink { return l }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `logging.go` | Entire package: CtrlLogger struct, NewLogrLogger factory, SetupKlog global initializer. | WithValues/WithName are no-ops — structured kv pairs from controller-runtime are silently dropped; do not add state to them. |

## Anti-Patterns

- Adding a second logger abstraction — this package is purely a bridge, not a logging framework
- Implementing WithValues/WithName with real state — Benthos logger is format-string based and cannot carry structured fields
- Calling SetupKlog more than once — klog global state is not re-entrant

## Decisions

- **Map logr level 0 to Info, all other levels to Debug** — controller-runtime uses level 0 for normal operational messages; higher levels are verbose debug output that should not appear at Info in production
- **Prefix klog entries with component=kubernetes** — Distinguishes Kubernetes controller-runtime log lines from Benthos pipeline logs in aggregated log streams

## Example: Wire Benthos logger into controller-runtime and klog

```
import "github.com/openmeterio/openmeter/collector/benthos/internal/logging"

logging.SetupKlog(svc.Logger())
logrLogger := logging.NewLogrLogger(svc.Logger())
```

<!-- archie:ai-end -->
