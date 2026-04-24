# logging

<!-- archie:ai-start -->

> Adapts Benthos' service.Logger to a logr.LogSink so controller-runtime and klog can route through a single Benthos logger instance. Single file, single type, no business logic.

## Patterns

**Benthos-to-logr bridge** — CtrlLogger wraps *service.Logger and implements logr.LogSink. Level 0 maps to Infof; any other level maps to Debugf. Error always calls Errorf. (`logr.New(&CtrlLogger{logger: logger})`)
**klog global setup via SetupKlog** — SetupKlog sets the global klog logger by calling klog.SetLogger with a logr.Logger that carries a 'component=kubernetes' key. Call once at startup. (`logging.SetupKlog(svc.Logger())`)
**No-op WithValues / WithName** — WithValues and WithName return the same receiver unchanged. Structured key-value context is intentionally dropped — Benthos format strings are unstructured. (`func (l *CtrlLogger) WithValues(keysAndValues ...any) logr.LogSink { return l }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `logging.go` | Entire package: CtrlLogger struct + NewLogrLogger + SetupKlog | WithValues/WithName are no-ops — structured kv pairs from controller-runtime are silently dropped; do not assume they propagate |

## Anti-Patterns

- Adding a second logger abstraction — this package is purely a bridge, not a logging framework
- Implementing WithValues/WithName with real state — Benthos logger is format-string based and cannot carry structured fields
- Calling SetupKlog more than once — klog global state is not re-entrant

## Decisions

- **Map logr level 0 → Info, everything else → Debug** — controller-runtime uses level 0 for normal operational messages; higher levels are verbose debug output that should not appear at Info in production
- **Prefix klog entries with component=kubernetes** — Distinguishes Kubernetes controller-runtime log lines from Benthos pipeline logs in aggregated log streams

<!-- archie:ai-end -->
