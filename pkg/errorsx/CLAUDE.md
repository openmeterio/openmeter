# errorsx

<!-- archie:ai-start -->

> Error utility package: prefix propagation through joined error trees, a warn-severity error wrapper, and the Handler/SlogHandler/NopHandler trio implementing httptransport.ErrorHandler for consistent error logging across HTTP transports.

## Patterns

**WithPrefix for structured error wrapping** — Use WithPrefix(err, prefix) instead of fmt.Errorf("%s: %w") when the error may be joined — it recursively prefixes each leaf rather than wrapping the join. (`return errorsx.WithPrefix(err, "reconcile subscription")`)
**NewWarnError for non-fatal errors** — Wrap errors that should log as warnings (not errors) with NewWarnError; SlogHandler detects *warnError and downgrades the level. (`return errorsx.NewWarnError(fmt.Errorf("skipping stale event: %w", err))`)
**SlogHandler as default HTTP error handler** — Inject SlogHandler (not NopHandler) via WithErrorHandler — it maps context.Canceled and *warnError to Warn automatically. (`httptransport.NewHandler(op, dec, enc, httptransport.WithErrorHandler(errorsx.NewSlogHandler(logger)))`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errorsx.go` | WithPrefix recursive join-aware prefixer; warnError type and NewWarnError constructor. | WithPrefix only recurses through the Unwrap() []error interface — errors wrapped with fmt.Errorf("%w", joinedErr) lose recursive prefixing on inner errors. |
| `handler.go` | Handler interface (mirrors httptransport.ErrorHandler), SlogHandler, NopHandler. | Handle and HandleContext differ only in context-aware logging — prefer HandleContext when a ctx is available. |

## Anti-Patterns

- Using fmt.Errorf("%s: %w", prefix, joinedErr) instead of WithPrefix — loses per-leaf prefix propagation.
- Returning *warnError directly — use NewWarnError which is nil-safe.
- Using NopHandler in production HTTP handlers — silently swallows transport errors.

## Decisions

- **WithPrefix recurses through Unwrap() []error but stops at single-Unwrap wrappers** — Only top-level joins should be expanded; wrapping a join with fmt.Errorf is an intentional barrier.

<!-- archie:ai-end -->
