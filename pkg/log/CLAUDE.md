# log

<!-- archie:ai-start -->

> Provides two orthogonal logging utilities: a testify-based mock for slog.Handler used in unit tests, and a PanicLogger recover-and-log helper for use in goroutine entry points. Primary constraint: this is infrastructure glue, not a logger factory — callers get slog.Default() for actual logging.

## Patterns

**PanicLogger via defer** — Always invoke PanicLogger as a deferred call at the top of a goroutine or main, passing one propagation option (WithRePanic, WithExit, or WithContinue). (`defer log.PanicLogger(log.WithExit)`)
**OTel stack trace attribute** — Stack traces are logged under the OTelCodeStackTrace constant key ("code.stacktrace") per OpenTelemetry semantic conventions — never use a different key for stack traces. (`slog.Error(description, OTelCodeStackTrace, string(debug.Stack()))`)
**MockHandler satisfies slog.Handler compile check** — var _ slog.Handler = &MockHandler{} is declared to enforce interface compliance at compile time — do the same for any new mock in this package. (`var _ slog.Handler = &MockHandler{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `panic.go` | Recover-and-log wrapper for goroutine panic handling; exposes three propagation strategies. | Default propagation strategy (zero value) is RePanic — omitting any option re-panics, which is rarely what you want in a worker; pass WithExit or WithContinue explicitly. |
| `mock.go` | testify mock for slog.Handler; use only in tests. | MockHandler.WithAttrs and WithGroup return values are obtained via type assertion on mock.Called — if the mock is not set up, the assertion will panic. |

## Anti-Patterns

- Adding a custom logger constructor here — callers should use slog.Default() or inject slog.Logger externally.
- Using a raw string instead of OTelCodeStackTrace for stack trace attribute keys.
- Invoking PanicLogger without a defer — it must run in a recover() scope.

## Decisions

- **Three named propagation strategies instead of a boolean re-panic flag** — Workers (WithContinue), long-running servers (WithExit), and test helpers (WithRePanic) have different needs; a single boolean would conflate exit vs continue.

<!-- archie:ai-end -->
