# contextx

<!-- archie:ai-start -->

> Context-keyed key-value store (via peterbourgon/ctxdata) and a slog.Handler adapter that auto-attaches those stored attributes to every log record emitted within a request context.

## Patterns

**WithAttr / WithAttrs to annotate context** — Call contextx.WithAttr(ctx, key, value) or contextx.WithAttrs(ctx, map) to attach structured data to the context. A ctxdata bag is lazily created if absent. (`ctx = contextx.WithAttr(ctx, "customer_id", customerID)`)
**NewLogHandler wraps existing slog.Handler** — Wrap any slog.Handler with contextx.NewLogHandler to automatically add all context attributes to every log record. Install at logger construction time. (`slog.New(contextx.NewLogHandler(slog.NewJSONHandler(os.Stdout, nil)))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `attr.go` | WithAttr and WithAttrs write to ctxdata; both silently ignore Set errors. | ctxdata.From returns nil if no bag exists — always handle the nil case by calling ctxdata.New before Set. |
| `log.go` | Handler wraps slog.Handler; Handle reads ctxdata.GetAllMap and appends all entries as slog.Any attributes. | WithAttrs and WithGroup delegate to the inner handler and re-wrap — do not forget to preserve the Handler struct. |

## Anti-Patterns

- Using context.WithValue directly for structured log fields instead of contextx.WithAttr
- Installing contextx.NewLogHandler more than once in the handler chain (double-attaches the same attributes)

## Decisions

- **peterbourgon/ctxdata as the context bag instead of custom context keys** — Provides a typed map API accessible from any layer without defining package-private key types everywhere.

<!-- archie:ai-end -->
