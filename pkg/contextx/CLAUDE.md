# contextx

<!-- archie:ai-start -->

> Context-carried structured attributes (via peterbourgon/ctxdata) plus an slog.Handler that automatically emits those attributes on every log record. Bridges request-scoped context data into structured logs.

## Patterns

**Attach attrs via WithAttr/WithAttrs** — Add request-scoped key/values with WithAttr(ctx, key, value) or WithAttrs(ctx, map); both lazily initialize the ctxdata store if absent. (`ctx = contextx.WithAttr(ctx, "namespace", ns)`)
**Wrap slog handler to drain ctxdata** — Wrap the base slog.Handler with contextx.NewLogHandler so every Handle() pulls ctxdata.From(ctx).GetAllMap() and adds them as record attrs. (`logger := slog.New(contextx.NewLogHandler(baseHandler))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `attr.go` | WithAttr / WithAttrs store key-values into the ctxdata container on the context. | Set errors are intentionally ignored (_ = d.Set). Returns a new ctx when the store was just created — always use the returned context. |
| `log.go` | Handler implementing slog.Handler that injects all ctxdata entries into each log record. | Handle calls ctxdata.From(ctx) without a nil check before GetAllMap(); records logged with a context lacking a ctxdata store rely on ctxdata returning a usable value. |

## Anti-Patterns

- Logging with a raw slog handler instead of the contextx-wrapped one, losing context attributes.
- Discarding the context returned by WithAttr/WithAttrs (the store may have just been created on it).

## Decisions

- **Carry log attributes on the context and drain them in a custom slog.Handler.** — Lets deep call sites enrich logs (namespace, request ids) without threading a logger and without per-call WithAttrs at every log statement.

## Example: Enrich context then have logs auto-include the attribute

```
import "github.com/openmeterio/openmeter/pkg/contextx"

logger := slog.New(contextx.NewLogHandler(base))
ctx = contextx.WithAttr(ctx, "namespace", ns)
logger.InfoContext(ctx, "created") // record carries namespace attr
```

<!-- archie:ai-end -->
