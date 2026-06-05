# tracex

<!-- archie:ai-start -->

> Generic OpenTelemetry span wrapper that runs a callback inside a started span, recording errors and panics and setting OK/Error status automatically. Used by worker/event-handler paths (subscriptionsync, notification eventhandler/webhook, watermill router) to instrument operations without repeating span boilerplate.

## Patterns

**Start-then-Wrap span lifecycle** — Create a span with Start[T] (or StartWithNoValue), then call .Wrap(fn) exactly once; Wrap calls span.End() itself, so callers must NOT defer span.End() separately. (`tracex.Start[Result](ctx, tracer, "my.op").Wrap(func(ctx context.Context) (Result, error) { return doWork(ctx) })`)
**Typed return via generics** — Span[T] is parameterized over the callback's return type T; the value/error pair from fn is passed through verbatim after status is set. (`func Start[T any](ctx, tracer, spanName, opts...) *Span[T]`)
**No-value variant delegates to Span[any]** — SpanNoValue.Wrap builds an internal Span[any] and adapts the error-only callback to (any,error), so there is one source of truth for status/panic handling. (`func (s *SpanNoValue) Wrap(fn func(ctx context.Context) error, opts ...Option) error`)
**Panic re-raised after recording** — On panic the deferred recover records the error with stacktrace attribute, sets Error status, ends the span, then re-panics — it never swallows the panic. (`panic(r) // after s.span.RecordError(...) and s.span.End()`)
**Functional options for status text** — OK status description is configurable only through Option (WithOkStatusDescription); defaultOptions sets it to "success". (`span.Wrap(fn, tracex.WithOkStatusDescription("created"))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `tracex.go` | Entire package: Span[T]/SpanNoValue types, Start/StartWithNoValue constructors, Wrap methods, Options + WithOkStatusDescription option. | Wrap ends the span — do not End() it again. The ctx returned by tracer.Start is stored on the Span and passed to fn as the span-scoped context; use that ctx inside fn, not the outer one, or child spans won't nest. |

## Anti-Patterns

- Calling span.End() or deferring it around Wrap — Wrap already ends the span, causing double-End.
- Swallowing panics inside the wrapped fn expecting tracex to convert them to errors — tracex re-panics.
- Using the pre-Start ctx inside fn instead of the ctx argument fn receives, breaking span parent/child linkage.

## Decisions

- **Wrap owns span.End() and panic recovery** — Centralizes the record-error/set-status/end sequence so instrumented call sites stay one-liners and cannot forget to end spans or record panics.
- **SpanNoValue reuses Span[any] internally** — Avoids duplicating panic/status logic for the common error-only operation shape.

## Example: Instrument an error-only operation

```
import "github.com/openmeterio/openmeter/pkg/framework/tracex"

func (s *svc) Sync(ctx context.Context) error {
    return tracex.StartWithNoValue(ctx, s.tracer, "subscriptionsync.Sync").
        Wrap(func(ctx context.Context) error {
            return s.reconcile(ctx)
        })
}
```

<!-- archie:ai-end -->
