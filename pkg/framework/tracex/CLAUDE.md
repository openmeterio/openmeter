# tracex

<!-- archie:ai-start -->

> Provides a generic, type-safe OTel span wrapper (Span[T] and SpanNoValue) that automatically records errors, sets span status, and recovers panics — eliminating span lifecycle boilerplate from every domain service.

## Patterns

**Start[T] + Wrap for value-returning operations** — Use tracex.Start[T](ctx, tracer, name) then span.Wrap(fn) for functions returning (T, error); status is set Ok/Error automatically and panics are recorded and re-panicked after span.End(). (`span := tracex.Start[*Invoice](ctx, s.tracer, "billing.GetInvoice"); return span.Wrap(func(ctx context.Context) (*Invoice, error) { return s.adapter.Get(ctx, id) })`)
**StartWithNoValue + Wrap for error-only operations** — Use tracex.StartWithNoValue(ctx, tracer, name) then span.Wrap(fn) when the function returns only error; delegates to Span[any].Wrap to share status logic. (`span := tracex.StartWithNoValue(ctx, s.tracer, "billing.DeleteInvoice"); return span.Wrap(func(ctx context.Context) error { return s.adapter.Delete(ctx, id) })`)
**Pass the span's ctx into the callback, never the outer ctx** — Span.Wrap passes the child span ctx into fn — always use the ctx argument inside Wrap callbacks, never capture the outer ctx, or OTel parent-child links break. (`span.Wrap(func(ctx context.Context) (*Invoice, error) { return s.adapter.Get(ctx, id) }) // ctx is the span child ctx`)
**WithOkStatusDescription for custom span descriptions** — Pass tracex.WithOkStatusDescription(desc) as a variadic opt to Wrap for a specific success description; default is 'success'. (`span.Wrap(fn, tracex.WithOkStatusDescription("invoice fetched"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `tracex.go` | Entire package: Span[T], SpanNoValue, Start, StartWithNoValue, Wrap, Options. Single file; all logic here. | Wrap always calls span.End() via both the panic-defer and normal paths — never call span.End() outside Wrap or the span is double-ended and OTel discards data. |

## Anti-Patterns

- Calling tracer.Start directly and managing span.End/RecordError manually — use tracex.Start + Wrap.
- Introducing context.Background() inside Wrap callbacks — use the ctx argument carrying the child span ctx.
- Adding new Span variants that don't delegate to Span[any].Wrap — SpanNoValue shows the correct delegation for error-only wrappers.
- Calling span.End() manually after span.Wrap — Wrap already ends; double-ending silently truncates the span.

## Decisions

- **Generic Span[T] instead of an untyped wrapper.** — Preserves compile-time type safety for the return value while sharing identical error-recording and panic-recovery logic across call sites.
- **Panic is re-panicked after span.End().** — The span must be closed before the stack unwinds, but the caller's defer/recover chain must still see the original panic.

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
