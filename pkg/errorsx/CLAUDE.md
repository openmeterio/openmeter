# errorsx

<!-- archie:ai-start -->

> Error utility package providing prefix propagation through joined error trees, a warn-severity error wrapper, and the Handler/SlogHandler/NopHandler trio that implement httptransport.ErrorHandler for consistent error logging across HTTP transports.

## Patterns

**WithPrefix for structured error wrapping** — Use WithPrefix(err, prefix) instead of fmt.Errorf("%s: %w") when the error may be a joined error — it recursively prefixes each leaf error rather than wrapping the join. (`return errorsx.WithPrefix(err, "reconcile subscription")`)
**NewWarnError for non-fatal errors** — Wrap errors that should log as warnings (not errors) with NewWarnError before returning. SlogHandler detects *warnError and downgrades the log level. (`return errorsx.NewWarnError(fmt.Errorf("skipping stale event: %w", err))`)
**SlogHandler as default HTTP error handler** — Inject SlogHandler (not NopHandler) into httptransport.Handler via WithErrorHandler — it maps context.Canceled to Warn and *warnError to Warn automatically. (`httptransport.NewHandler(op, dec, enc, httptransport.WithErrorHandler(errorsx.NewSlogHandler(logger)))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errorsx.go` | WithPrefix recursive join-aware prefixer; warnError type and NewWarnError constructor. | WithPrefix only recurses one level via the Unwrap() []error interface check — errors wrapped with fmt.Errorf("%w", joinedErr) lose the recursive prefix on the inner errors. |
| `handler.go` | Handler interface (mirrors httptransport.ErrorHandler), SlogHandler, NopHandler. Used by HTTP transport layer. | SlogHandler.Handle and HandleContext differ only in context-aware vs non-context logging — always prefer HandleContext when a ctx is available. |

## Anti-Patterns

- Using fmt.Errorf("%s: %w", prefix, joinedErr) instead of WithPrefix — loses per-leaf prefix propagation in joined errors
- Returning *warnError directly — always use NewWarnError which handles nil safely
- Using NopHandler in production HTTP handlers — silently swallows all transport errors

## Decisions

- **WithPrefix recurses through Unwrap() []error (joined errors) but stops at single-Unwrap wrappers** — Only top-level joins should be expanded; wrapping a join with fmt.Errorf is an intentional barrier that should not be silently penetrated.

<!-- archie:ai-end -->
