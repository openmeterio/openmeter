# contextx

<!-- archie:ai-start -->

> Context-keyed key-value store (via peterbourgon/ctxdata) and a slog.Handler adapter that auto-attaches stored attributes to every log record emitted within a request context. Used to propagate structured log fields through service layers without threading explicit logger parameters.

## Patterns

**WithAttr / WithAttrs to annotate context** — Call contextx.WithAttr(ctx, key, value) or contextx.WithAttrs(ctx, map) to attach structured data. A ctxdata bag is lazily created if absent. (`ctx = contextx.WithAttr(ctx, "customer_id", customerID)`)
**NewLogHandler wraps existing slog.Handler** — Wrap any slog.Handler with contextx.NewLogHandler to automatically add all context attributes to every log record. Install at logger construction time, not per-request. (`slog.New(contextx.NewLogHandler(slog.NewJSONHandler(os.Stdout, nil)))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `attr.go` | WithAttr and WithAttrs write to ctxdata bag; both silently ignore Set errors. Creates a new bag via ctxdata.New if none exists. | ctxdata.From returns nil if no bag exists — always handle the nil case (the helpers do this, but raw ctxdata callers must too). |
| `log.go` | Handler wraps slog.Handler; Handle reads ctxdata.GetAllMap and appends all entries as slog.Any attributes before delegating. | WithAttrs and WithGroup must re-wrap the inner handler in a new Handler struct to preserve the contextx behaviour on derived loggers. |

## Anti-Patterns

- Using context.WithValue directly for structured log fields instead of contextx.WithAttr
- Installing contextx.NewLogHandler more than once in a handler chain — double-attaches the same attributes to every record

## Decisions

- **peterbourgon/ctxdata as the context bag instead of custom context keys** — Provides a typed map API accessible from any layer without defining package-private key types everywhere.

<!-- archie:ai-end -->
