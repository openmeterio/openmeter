# encoder

<!-- archie:ai-start -->

> Defines the two generic function-type contracts the httptransport handler abstraction uses to serialize results and errors to the HTTP wire: ResponseEncoder[Response] and ErrorEncoder. It is a pure type-declaration package with no behavior, consumed by the parent httptransport .With().ServeHTTP handler used across v3 handlers.

## Patterns

**Encoders are bare function types, not interfaces** — Both contracts are declared as named function types so callers pass closures inline rather than implementing structs. ResponseEncoder is generic over the Response type; ErrorEncoder is non-generic. (`type ResponseEncoder[Response any] func(ctx context.Context, w http.ResponseWriter, r *http.Request, response Response) error`)
**ErrorEncoder returns bool to signal handled** — ErrorEncoder returns a bool (true = this encoder handled/wrote the error) so the parent transport can chain encoders and fall through to a default when one declines. (`type ErrorEncoder func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool`)
**Stdlib-only, zero dependencies** — Imports are limited to context and net/http. No project packages, no logging, no models. Keep this package dependency-free so the whole transport layer can import it without cycles. (`import (
	"context"
	"net/http"
)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `encoder.go` | Sole file; declares ResponseEncoder[Response] (returns error) and ErrorEncoder (returns bool). Both take (ctx, w, r) plus a payload. | Note the asymmetric return types: ResponseEncoder returns error, ErrorEncoder returns bool. Changing either signature breaks every callsite in pkg/framework/transport/httptransport and downstream v3 handlers. |

## Anti-Patterns

- Adding concrete encoder implementations or HTTP-writing logic here — this package only declares the contracts; implementations live in callers/httptransport.
- Importing project packages (models, commonhttp, log) — would introduce an import cycle into the foundational transport layer.
- Converting the function types to interfaces — callers rely on passing plain closures.

## Decisions

- **Split the encoder type contracts into their own leaf package separate from httptransport.** — Keeps the type definitions importable by anything in the transport stack without pulling in the full handler machinery, avoiding cycles.
- **ErrorEncoder returns bool rather than error.** — Enables a chain-of-responsibility where each error encoder can decline (false) and let a later/default encoder handle the error.

## Example: Implementing a ResponseEncoder and a fall-through ErrorEncoder

```
var jsonResp encoder.ResponseEncoder[MyDTO] = func(ctx context.Context, w http.ResponseWriter, r *http.Request, response MyDTO) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(response)
}

var notFound encoder.ErrorEncoder = func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool {
	if !errors.Is(err, ErrNotFound) {
		return false // decline; let the next encoder handle it
	}
	w.WriteHeader(http.StatusNotFound)
	return true
}
```

<!-- archie:ai-end -->
