# encoder

<!-- archie:ai-start -->

> Declares the two core function types — ResponseEncoder[Response] and ErrorEncoder — that form the encode half of the httptransport decode/operation/encode pipeline. Every HTTP handler in the codebase depends on these signatures for serialising responses and mapping errors to HTTP status codes.

## Patterns

**ResponseEncoder generic function type** — ResponseEncoder[Response any] receives the strongly-typed response value and writes it to http.ResponseWriter. New encoders must match this exact signature and must be stateless — no mutable captured state, since they may run concurrently across goroutines. (`var encode encoder.ResponseEncoder[MyResponse] = func(ctx context.Context, w http.ResponseWriter, r *http.Request, resp MyResponse) error { w.Header().Set("Content-Type", "application/json"); return json.NewEncoder(w).Encode(resp) }`)
**ErrorEncoder bool-chain convention** — ErrorEncoder returns bool — true means the error was handled and the chain stops; false passes to the next encoder. httptransport.Handler iterates encoders in order and the first true short-circuits. An ErrorEncoder returning true MUST have written a response. (`var myErr encoder.ErrorEncoder = func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool { var ve *ValidationError; if errors.As(err, &ve) { http.Error(w, ve.Error(), 400); return true }; return false }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `encoder.go` | Declares ResponseEncoder[Response] and ErrorEncoder — the only two exported symbols. All httptransport.Handler instances accept these types. | An ErrorEncoder returning true without writing a body leaves the client hanging; a ResponseEncoder must set Content-Type before writing or client JSON decoders break. |

## Anti-Patterns

- Adding stateful fields or methods to this package — both types are pure function types
- Returning false from an ErrorEncoder after already writing to ResponseWriter — a downstream encoder will attempt a second write and corrupt the response
- Using a ResponseEncoder to also handle errors — keep the error path in the ErrorEncoder chain so the caller controls ordering
- Holding mutable shared state in a ResponseEncoder closure — encoders may run concurrently

## Decisions

- **Separate ResponseEncoder and ErrorEncoder into distinct function types rather than one Encoder interface** — Lets httptransport.Handler apply an ordered ErrorEncoder chain before the success ResponseEncoder, giving callers fine-grained error-mapping priority (e.g. ValidationIssue → 400 before generic 500).

## Example: Implementing a JSON ResponseEncoder and a domain-error ErrorEncoder for a new handler

```
import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

var encodeResponse encoder.ResponseEncoder[MyResponse] = func(
	ctx context.Context, w http.ResponseWriter, r *http.Request, resp MyResponse,
) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}
```

<!-- archie:ai-end -->
