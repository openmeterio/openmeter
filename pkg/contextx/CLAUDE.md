# contextx

<!-- archie:ai-start -->

> Context-keyed key-value store (via peterbourgon/ctxdata) plus a slog.Handler adapter that auto-attaches stored attributes to every log record emitted within a request context. Propagates structured log fields through service layers without threading explicit logger parameters.

## Patterns

**WithAttr / WithAttrs to annotate context** — Call contextx.WithAttr(ctx, key, value) or WithAttrs(ctx, map) to attach structured data; a ctxdata bag is lazily created if absent. (`ctx = contextx.WithAttr(ctx, "customer_id", customerID)`)
**NewLogHandler wraps existing slog.Handler** — Wrap any slog.Handler with contextx.NewLogHandler to add context attributes to every record. Install at logger construction time, not per-request. (`slog.New(contextx.NewLogHandler(slog.NewJSONHandler(os.Stdout, nil)))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `attr.go` | WithAttr/WithAttrs write to a ctxdata bag (created via ctxdata.New if none exists); both silently ignore Set errors. | ctxdata.From returns nil if no bag exists — the helpers handle nil, but raw ctxdata callers must too. |
| `log.go` | Handler wraps a slog.Handler; Handle reads ctxdata.GetAllMap and appends all entries as slog.Any attributes before delegating. | WithAttrs and WithGroup must re-wrap the inner handler in a new Handler struct to preserve contextx behaviour on derived loggers. |

## Anti-Patterns

- Using context.WithValue directly for structured log fields instead of contextx.WithAttr.
- Installing contextx.NewLogHandler more than once in a handler chain — double-attaches the same attributes.

## Decisions

- **peterbourgon/ctxdata as the context bag instead of custom context keys.** — Provides a typed map API accessible from any layer without defining package-private key types everywhere.

<!-- archie:ai-end -->
