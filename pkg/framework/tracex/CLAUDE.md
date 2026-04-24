# tracex

<!-- archie:ai-start -->

> Provides a generic, type-safe OTel span wrapper that automatically records errors, sets span status, and handles panics. Eliminates boilerplate span lifecycle management from all domain services.

## Patterns

**Span[T] for value-returning operations** — Use tracex.Start[T] + Span.Wrap when the wrapped function returns (T, error). Status is set to Ok or Error automatically; panic is recorded and re-panicked after span.End(). (`span := tracex.Start[*billing.Invoice](ctx, tracer, "billing.GetInvoice"); return span.Wrap(func(ctx context.Context) (*billing.Invoice, error) { return adapter.Get(ctx, id) })`)
**SpanNoValue for error-only operations** — Use tracex.StartWithNoValue + SpanNoValue.Wrap when the wrapped function returns only error. Internally delegates to Span[any].Wrap to share status-setting logic. (`span := tracex.StartWithNoValue(ctx, tracer, "billing.DeleteInvoice"); return span.Wrap(func(ctx context.Context) error { return adapter.Delete(ctx, id) })`)
**Options via functional option pattern** — Customise the OkStatusDescription via tracex.WithOkStatusDescription(desc) passed as variadic opts to Wrap. Default is "success". (`span.Wrap(fn, tracex.WithOkStatusDescription("invoice fetched"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `tracex.go` | Entire package — defines Span[T], SpanNoValue, Start, StartWithNoValue, Wrap, and Options. Only file in the package. | Span.Wrap always calls span.End() via defer-on-panic and explicit call after fn; do not call span.End() outside Wrap or the span will be double-ended. |

## Anti-Patterns

- Calling tracer.Start directly and managing span.End/RecordError manually — use tracex.Start + Wrap instead
- Introducing context.Background() inside Wrap callbacks — the Span carries the correct child ctx; always pass s.ctx through
- Adding new Span variants that don't delegate to Span[any].Wrap — SpanNoValue already shows the correct delegation pattern

## Decisions

- **Generic Span[T] instead of a single untyped wrapper** — Preserves compile-time type safety for the return value while sharing the identical error-recording and panic-recovery logic across all call sites.
- **Panic is re-panicked after span.End()** — The span must be closed (exported) before the stack unwinds further, but the caller's defer/recover chain must still see the original panic.

## Example: Wrap a DB read in a named span with automatic error recording

```
import (
	"context"
	"go.opentelemetry.io/otel/trace"
	"github.com/openmeterio/openmeter/pkg/framework/tracex"
)

func (s *service) GetInvoice(ctx context.Context, id string) (*Invoice, error) {
	span := tracex.Start[*Invoice](ctx, s.tracer, "billing.GetInvoice")
	return span.Wrap(func(ctx context.Context) (*Invoice, error) {
		return s.adapter.Get(ctx, id)
	})
}
```

<!-- archie:ai-end -->
