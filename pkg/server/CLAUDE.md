# server

<!-- archie:ai-start -->

> Shared HTTP server middleware and request-logging primitives reused across both api/v3/server and openmeter/server, built on go-chi middleware and OTel semconv.

## Patterns

**Middleware as func(http.Handler) http.Handler** — Each middleware is a constructor returning the standard chi/net-http middleware signature; MiddlewareFunc aliases it for cross-version reuse. (`type MiddlewareFunc func(http.Handler) http.Handler`)
**OTel semconv attribute keys** — Request attributes and log fields use go.opentelemetry.io/otel/semconv/v1.27.0 keys (HTTPRequestMethodKey, URLFullKey, HTTPResponseStatusCodeKey) rather than ad-hoc strings. (`attrs[string(semconv.HTTPRequestMethodKey)] = r.Method`)
**chi LogFormatter/LogEntry implementation** — RequestLogger implements middleware.LogFormatter and RequestLoggerEntry implements middleware.LogEntry (compile-time asserted via var _ ...), logging via slog at Debug on Write and Error on Panic. (`var ( _ middleware.LogFormatter = (*RequestLogger)(nil); _ middleware.LogEntry = (*RequestLoggerEntry)(nil) )`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `attributes.go` | GetRequestAttributes(r) builds an OTel-semconv attribute map for a request; adds req_id from middleware.GetReqID. | Requires middleware.RequestID to be installed upstream or req_id will be absent. There is no semconv key for request id, so a custom `req_id` key is used. |
| `logger.go` | NewRequestLoggerMiddleware(slog.Handler) plus RequestLogger/RequestLoggerEntry implementing chi's logging hooks. | Successful requests log at slog.LevelDebug — they are silent at higher log levels. Panics log stack + panic value at Error. |
| `poweredby.go` | NewPoweredByMiddleware sets the X-Powered-By: "OpenMeter by Kong, Inc." header on every response. | Header value is a constant; change it in one place. Covered by poweredby_test.go. |
| `middleware.go` | Declares the shared MiddlewareFunc type used across API versions. | Keep this signature identical to net/http middleware so chi .Use accepts it directly. |

## Anti-Patterns

- Using hardcoded attribute/log-field strings instead of semconv keys.
- Logging successful requests above Debug level (breaks the intended quiet default).
- Duplicating the X-Powered-By value instead of using the constant in poweredby.go.

## Decisions

- **Centralize cross-version HTTP middleware here rather than in api/v3 or openmeter/server.** — Both server stacks import pkg/server so request logging, attributes, and headers stay consistent across API versions.

## Example: Adding a response-header middleware in chi style

```
func NewPoweredByMiddleware() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set(poweredByHeader, poweredByValue)
			next.ServeHTTP(w, r)
		})
	}
}
```

<!-- archie:ai-end -->
