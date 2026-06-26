# errorsx

<!-- archie:ai-start -->

> Cross-cutting error utilities for the HTTP/server layer: prefix annotation that preserves errors.Join trees, a warn-level error wrapper, generated-API-error detection, and slog-based error Handlers implementing httptransport.ErrorHandler.

## Patterns

**WithPrefix preserves joined-error structure** — WithPrefix recursively prefixes each branch of an errors.Join tree (only at the top level via the Unwrap() []error interface) instead of flattening, so multi-error messages keep per-error prefixes. (`errorsx.WithPrefix(errors.Join(e1, e2), "prefix")`)
**warnError downgrades log severity** — Wrap an error with NewWarnError to make SlogHandler log it at Warn instead of Error; detected via lo.ErrorsAs[*warnError]. context.Canceled and generated API errors are also logged as warnings. (`return errorsx.NewWarnError(err)`)
**Handler interface mirrors httptransport.ErrorHandler** — Handler { Handle(err); HandleContext(ctx, err) }; SlogHandler (requires injected *slog.Logger) and NopHandler are the impls. Compile-time assertions enforce the interface match. (`var _ httptransport.ErrorHandler = (errorsx.Handler)(nil)`)
**Generated API errors identified by package path** — isAPIError reflects on the error type and walks the unwrap chain (single and []error) comparing PkgPath against api and api/v3 packages, because codegen emits no common base error type. (`reflect.TypeOf(api.InvalidParamFormatError{}).PkgPath()`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `errorsx.go` | WithPrefix + warnError/NewWarnError | WithPrefix only recurses when the error implements Unwrap() []error at the top level (deliberate) — wrapping a join with fmt.Errorf hides the inner errors from per-branch prefixing |
| `handler.go` | Handler interface, SlogHandler, NopHandler | SlogHandler requires an injected logger (no slog.Default fallback per project convention); severity downgrade order is context.Canceled, then isAPIError, then warnError, else Error |
| `helpers.go` | isAPIError/isErrorFromPackages reflection-based detection | Detection is by reflect PkgPath, so it breaks if generated error types move packages; both api and api/v3 must be listed |
| `errorsx_test.go` | WithPrefix tree-preservation tests | Encodes the non-top-level flattening behavior as expected output |
| `helpers_test.go` | isAPIError detection tests for legacy/v3/wrapped/joined errors | Asserts wrapped and joined API errors are still detected |

## Anti-Patterns

- Using slog.Default() to build SlogHandler — the logger must be injected explicitly per project convention
- Flattening errors.Join before WithPrefix (e.g. via fmt.Errorf %w) — loses per-error prefixing
- Hardcoding new API error type detection instead of extending apiErrorPackages with the generating package's path

## Decisions

- **Detect generated API errors by reflecting on package path** — oapi-codegen output provides no shared base error or marker interface, so package matching is the most maintainable signal
- **Provide a warn-level error wrapper and treat API/cancellation errors as warnings** — Keeps expected client-side and lifecycle errors out of error-level logs and alerting

## Example: Build a server error handler that logs API/cancellation/warn errors at Warn level

```
import (
  "github.com/openmeterio/openmeter/pkg/errorsx"
)

h := errorsx.NewSlogHandler(logger) // injected *slog.Logger
h.HandleContext(ctx, errorsx.NewWarnError(err)) // logged at Warn
```

<!-- archie:ai-end -->
