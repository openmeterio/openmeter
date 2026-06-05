# log

<!-- archie:ai-start -->

> Logging support utilities (not a logger implementation): a testify-based mock slog.Handler and a deferrable PanicLogger that logs recovered panics with an OTel-compatible stack trace and then re-panics, exits, or continues.

## Patterns

**MockHandler implements slog.Handler** — `MockHandler` embeds testify `mock.Mock` and satisfies `slog.Handler` (Enabled/Handle/WithAttrs/WithGroup); a compile-time assertion `var _ slog.Handler = &MockHandler{}` enforces the contract. Construct via NewMockHandler(). (`var _ slog.Handler = &MockHandler{}`)
**PanicLogger with functional propagation strategy** — `PanicLogger(options ...func(*panicLoggerOptions))` is used as `defer log.PanicLogger(...)`; WithRePanic (default), WithExit, WithContinue select what happens after logging the recovered panic. (`defer log.PanicLogger(log.WithExit)`)
**Stack trace recorded under OTel semantic key** — Recovered panics are logged with `debug.Stack()` under the `code.stacktrace` attribute (OTelCodeStackTrace const) for OpenTelemetry semantic-convention compatibility. (`slog.Error(description, OTelCodeStackTrace, string(debug.Stack()))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `mock.go` | NewMockHandler / MockHandler: a testify mock slog.Handler for asserting log behavior in tests. | Test-only helper; the four interface methods must all be kept implemented or the slog.Handler assertion breaks the build. |
| `panic.go` | PanicLogger plus WithRePanic/WithExit/WithContinue options and the OTelCodeStackTrace constant. | Default strategy re-panics; WithExit calls os.Exit(1). This package is the sanctioned recover point — production code elsewhere must not panic. |

## Anti-Patterns

- Using MockHandler outside tests, or shipping log assertions into production wiring.
- Changing the default propagation strategy away from re-panic, silently swallowing panics callers expect to propagate.
- Logging stack traces under an ad-hoc attribute key instead of OTelCodeStackTrace.

## Decisions

- **PanicLogger centralizes panic recovery with a selectable propagation strategy.** — main entrypoints can `defer log.PanicLogger(log.WithExit)` to guarantee panics are logged with a stack trace before the process terminates.

## Example: Guard a main goroutine against unlogged panics

```
import "github.com/openmeterio/openmeter/pkg/log"

func main() {
	defer log.PanicLogger(log.WithExit)
	// ... startup ...
}
```

<!-- archie:ai-end -->
