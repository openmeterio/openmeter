# server

<!-- archie:ai-start -->

> Shared HTTP server primitives — request attribute extraction, structured logging middleware, powered-by header middleware, and a MiddlewareFunc type alias — reused by both the v1 (openmeter/server) and v3 (api/v3/server) API servers with no business logic.

## Patterns

**OTel semconv attribute keys** — Use go.opentelemetry.io/otel/semconv/v1.27.0 key constants for all HTTP attribute names. Never hardcode string keys. (`string(semconv.HTTPRequestMethodKey): r.Method`)
**Chi middleware.LogFormatter / LogEntry interfaces** — RequestLogger implements middleware.LogFormatter and RequestLoggerEntry implements middleware.LogEntry. Pass RequestLogger to middleware.RequestLogger() to integrate with Chi's logging chain. (`middleware.RequestLogger(&RequestLogger{Logger: handler})`)
**Middleware factory pattern** — Every middleware is exposed as a NewXxxMiddleware() returning func(http.Handler) http.Handler, matching Chi's middleware signature. (`func NewPoweredByMiddleware() func(next http.Handler) http.Handler { ... }`)
**req_id depends on upstream RequestID middleware** — GetRequestAttributes populates req_id only when middleware.RequestID is upstream in the chain. Absent middleware → absent attribute with no error. (`if reqID := middleware.GetReqID(ctx); reqID != "" { attrs["req_id"] = reqID }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `attributes.go` | Returns an OTel-keyed attribute map for a request. Requires middleware.RequestID upstream for the req_id field. | req_id is silently absent if RequestID middleware is missing; this is not an error. |
| `logger.go` | Structured slog-based request logger satisfying Chi's middleware.LogFormatter interface. | Normal requests log at slog.LevelDebug; panics at slog.LevelError. Do not raise normal requests to LevelInfo without considering log volume. |
| `middleware.go` | Defines the shared MiddlewareFunc type alias used across v1 and v3 server wiring. | Must stay in sync with Chi's func(http.Handler) http.Handler signature. Never wrap in a struct. |
| `poweredby.go` | Injects the X-Powered-By header on every response. | Header value 'OpenMeter by Kong, Inc.' is asserted in poweredby_test.go — changing it breaks clients that parse this header. |

## Anti-Patterns

- Adding business logic or domain package imports — this package must remain a pure infrastructure utility.
- Hardcoding OTel attribute key strings instead of using semconv constants.
- Returning errors from middleware constructors — they should panic on bad config or return a no-op handler.
- Using GetRequestAttributes without middleware.RequestID in the chain — req_id will silently be missing.

## Decisions

- **Log at DEBUG for normal requests, ERROR for panics.** — High-volume request logs would flood INFO in production; DEBUG keeps them filterable without losing panic diagnostics.

## Example: Add the powered-by middleware to a Chi router

```
import "github.com/openmeterio/openmeter/pkg/server"

router.Use(server.NewPoweredByMiddleware())
```

<!-- archie:ai-end -->
