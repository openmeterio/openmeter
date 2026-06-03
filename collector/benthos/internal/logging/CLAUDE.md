# logging

<!-- archie:ai-start -->

> Single-file bridge package that adapts Benthos' *service.Logger to a logr.LogSink, letting controller-runtime and klog route through the same Benthos logger instance. No business logic — purely a logging adapter.

## Patterns

**Benthos-to-logr bridge via CtrlLogger** — CtrlLogger implements logr.LogSink by wrapping *service.Logger. Info level 0 maps to Infof; any other level maps to Debugf; Error always calls Errorf. Construct via NewLogrLogger / logr.New(&CtrlLogger{logger: ...}). (`logrLogger := logging.NewLogrLogger(svc.Logger())`)
**SetupKlog called exactly once at startup** — SetupKlog sets the global klog logger via klog.SetLogger with a logr.Logger carrying component=kubernetes. klog global state is not re-entrant, so call it once. (`logging.SetupKlog(svc.Logger())`)
**WithValues and WithName are intentional no-ops** — Both methods return the same receiver unchanged. Benthos uses format strings, so structured kv pairs from controller-runtime are silently dropped by design — do not add state to them. (`func (l *CtrlLogger) WithValues(keysAndValues ...any) logr.LogSink { return l }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `logging.go` | Entire package: CtrlLogger struct (Init/Enabled/Info/Error/WithValues/WithName), NewLogrLogger factory, SetupKlog global initializer. | WithValues/WithName are no-ops — structured kv pairs are silently dropped; Enabled always returns true so level filtering must happen upstream. |

## Anti-Patterns

- Adding a second logger abstraction — this package is purely a bridge, not a logging framework
- Implementing WithValues/WithName with real state — the Benthos logger is format-string based and cannot carry structured fields
- Calling SetupKlog more than once — klog global state is not re-entrant

## Decisions

- **Map logr level 0 to Info and all other levels to Debug** — controller-runtime uses level 0 for normal operational messages; higher levels are verbose debug output that should not appear at Info in production.
- **Prefix klog entries with component=kubernetes** — Distinguishes Kubernetes controller-runtime log lines from Benthos pipeline logs in aggregated log streams.

## Example: Wire the Benthos logger into controller-runtime and klog

```
import "github.com/openmeterio/openmeter/collector/benthos/internal/logging"

logging.SetupKlog(svc.Logger())
logrLogger := logging.NewLogrLogger(svc.Logger())
```

<!-- archie:ai-end -->
