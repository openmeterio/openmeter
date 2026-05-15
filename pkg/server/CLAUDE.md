# server

<!-- archie:ai-start -->

> Shared HTTP server primitives — request attribute extraction, structured logging middleware, powered-by header middleware, and a MiddlewareFunc type alias — reused by both the v1 (openmeter/server) and v3 (api/v3/server) API servers with no business logic.

## Patterns

**OTel semconv attribute keys** — Use go.opentelemetry.io/otel/semconv/v1.27.0 key constants for all HTTP attribute names. Never hardcode string keys. (`string(semconv.HTTPRequestMethodKey): r.Method`)
**Chi middleware.LogFormatter / LogEntry interfaces** — RequestLogger implements middleware.LogFormatter and RequestLoggerEntry implements middleware.LogEntry. Pass RequestLogger to middleware.RequestLogger() to integrate with Chi's request logging chain. (`middleware.RequestLogger(&RequestLogger{Logger: handler})`)
**Middleware factory pattern** — Every middleware is exposed as a NewXxxMiddleware() function returning func(http.Handler) http.Handler, matching Chi's middleware signature. (`func NewPoweredByMiddleware() func(next http.Handler) http.Handler { ... }`)
**req_id depends on upstream RequestID middleware** — GetRequestAttributes populates the req_id field only when middleware.RequestID is in the middleware chain upstream. Absent middleware → absent attribute with no error. (`if reqID := middleware.GetReqID(ctx); reqID != "" { attrs["req_id"] = reqID }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `attributes.go` | Returns OTel-keyed attribute map for a request. Requires middleware.RequestID upstream for req_id field. | req_id is silently absent if RequestID middleware is missing; this is not an error. |
| `logger.go` | Structured slog-based request logger satisfying Chi's middleware.LogFormatter interface. | Normal requests log at slog.LevelDebug; panics log at slog.LevelError. Do not raise normal requests to LevelInfo without considering log volume. |
| `middleware.go` | Defines shared MiddlewareFunc type alias used across v1 and v3 server wiring. | Must stay in sync with Chi's func(http.Handler) http.Handler signature. Never wrap in a struct. |
| `poweredby.go` | Injects X-Powered-By header on every response. | Header value 'OpenMeter by Kong, Inc.' is asserted in poweredby_test.go — changing it is a breaking change for clients that parse this header. |

## Anti-Patterns

- Adding business logic or domain package imports — this package must remain a pure infrastructure utility.
- Hardcoding OTel attribute key strings instead of using semconv constants.
- Returning errors from middleware constructors — they should panic on bad config or return a no-op handler.
- Registering GetRequestAttributes usage without middleware.RequestID in the chain — req_id will silently be missing.

## Decisions

- **Log at DEBUG for normal requests, ERROR for panics** — High-volume request logs would flood INFO level in production; DEBUG keeps them filterable without losing panic diagnostics.

<!-- archie:ai-end -->
