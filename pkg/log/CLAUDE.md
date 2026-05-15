# log

<!-- archie:ai-start -->

> Provides two orthogonal logging utilities: a testify-based mock for slog.Handler used in unit tests, and a PanicLogger recover-and-log helper for goroutine entry points. This is infrastructure glue, not a logger factory — callers must use slog.Default() for actual logging.

## Patterns

**PanicLogger via defer with explicit propagation strategy** — Always invoke PanicLogger as a deferred call at the top of a goroutine or main, passing exactly one propagation option. The zero value (no option) re-panics — explicitly pass WithExit or WithContinue for workers and servers. (`defer log.PanicLogger(log.WithExit)   // server main
defer log.PanicLogger(log.WithContinue) // worker loop iteration`)
**OTelCodeStackTrace constant for stack trace attribute key** — Stack traces must be logged under the OTelCodeStackTrace constant ("code.stacktrace") per OpenTelemetry semantic conventions. Never use a raw string for this key. (`slog.Error(description, log.OTelCodeStackTrace, string(debug.Stack()))`)
**Compile-time slog.Handler interface assertion for MockHandler** — var _ slog.Handler = &MockHandler{} is declared to enforce interface compliance at compile time. Any new mock in this package must include the same assertion. (`var _ slog.Handler = &MockHandler{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `panic.go` | Recover-and-log wrapper; exposes PanicLogger and three propagation strategies (WithRePanic, WithExit, WithContinue). | Default strategy (zero value) is RePanic — omitting any option re-panics, which is rarely correct for worker goroutines. Always pass WithExit or WithContinue explicitly. |
| `mock.go` | testify mock for slog.Handler; use only in tests. | MockHandler.WithAttrs and WithGroup return values obtained via type assertion on mock.Called — if the mock expectation is not set up, the assertion panics at test runtime. |

## Anti-Patterns

- Adding a custom logger constructor here — callers should use slog.Default() or inject slog.Logger externally
- Using a raw string instead of OTelCodeStackTrace for stack trace attribute keys
- Invoking PanicLogger without defer — it must run inside a recover() scope
- Calling PanicLogger with no options in a worker loop — the default re-panic strategy will terminate the worker

## Decisions

- **Three named propagation strategies instead of a boolean re-panic flag** — Workers (WithContinue), long-running servers (WithExit), and test helpers (WithRePanic) have fundamentally different needs; a single boolean would conflate exit-process vs continue-loop semantics.

<!-- archie:ai-end -->
