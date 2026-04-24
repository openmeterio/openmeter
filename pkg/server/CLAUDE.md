# server

<!-- archie:ai-start -->

> Shared HTTP server primitives (request attribute extraction, structured logging middleware, powered-by header middleware) reused by both the v1 and v3 API servers; contains no business logic.

## Patterns

**OTel semconv attribute keys** — Use go.opentelemetry.io/otel/semconv/v1.27.0 key constants for all HTTP attribute names; never hardcode string keys like 'http.method'. (`string(semconv.HTTPRequestMethodKey): r.Method`)
**Chi middleware.LogFormatter / LogEntry interfaces** — RequestLogger implements middleware.LogFormatter and RequestLoggerEntry implements middleware.LogEntry; pass RequestLogger to middleware.RequestLogger() to integrate with Chi's request logging chain. (`middleware.RequestLogger(&RequestLogger{Logger: handler})`)
**Middleware factory pattern** — Every middleware is exposed as a NewXxxMiddleware() func returning func(http.Handler) http.Handler, matching Chi's middleware signature. (`func NewPoweredByMiddleware() func(next http.Handler) http.Handler { ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `attributes.go` | Returns OTel-keyed attribute map for a request; requires middleware.RequestID to be in the middleware chain for req_id to be populated. | req_id is absent silently if RequestID middleware is missing upstream. |
| `logger.go` | Structured slog-based request logger satisfying Chi's middleware.LogFormatter interface. | Log level is slog.LevelDebug for normal requests and slog.LevelError for panics; do not change to Info without considering log volume. |
| `middleware.go` | Defines the shared MiddlewareFunc type alias used across v1 and v3 server wiring. | Keep this type alias in sync with Chi's func(http.Handler) http.Handler; never wrap it in a struct. |
| `poweredby.go` | Injects X-Powered-By header on every response. | The header value 'OpenMeter by Kong, Inc.' is tested explicitly in poweredby_test.go; updating it is a breaking change for clients that parse this header. |

## Anti-Patterns

- Adding business logic or domain imports to this package — it must stay a pure infrastructure utility
- Hardcoding OTel attribute key strings instead of using semconv constants
- Returning errors from middleware constructors — they should panic on bad config or return a no-op handler

## Decisions

- **Log at DEBUG for normal requests, ERROR for panics** — High-volume request logs would flood INFO level in production; DEBUG keeps them filterable without losing panic diagnostics.

<!-- archie:ai-end -->
