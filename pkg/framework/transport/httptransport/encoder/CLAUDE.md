# encoder

<!-- archie:ai-start -->

> Defines the two core function types — ResponseEncoder and ErrorEncoder — that form the encode half of the httptransport decode/operation/encode pipeline. Every HTTP handler in the system depends on these signatures for serialising responses and mapping errors to HTTP status codes.

## Patterns

**ResponseEncoder generic function type** — ResponseEncoder[Response any] is a typed function that receives the strongly-typed response value and writes it to http.ResponseWriter. New response encoders must match this exact signature; they must not hold state or close over mutable shared objects. (`var encode encoder.ResponseEncoder[MyResponse] = func(ctx context.Context, w http.ResponseWriter, r *http.Request, resp MyResponse) error { return json.NewEncoder(w).Encode(resp) }`)
**ErrorEncoder chain convention** — ErrorEncoder returns bool — true means the error was handled and the chain stops, false passes to the next encoder. Callers (httptransport.Handler) iterate encoders in order; the first true short-circuits. Every ErrorEncoder must write a response before returning true. (`var myErr encoder.ErrorEncoder = func(ctx context.Context, err error, w http.ResponseWriter, r *http.Request) bool { var ve *ValidationError; if errors.As(err, &ve) { http.Error(w, ve.Error(), 400); return true }; return false }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `encoder.go` | Declares ResponseEncoder[Response] and ErrorEncoder — the only two exported symbols. All httptransport.Handler instances in the codebase accept these types. | ErrorEncoder returning true without writing a response body leaves the client hanging; ResponseEncoder must set Content-Type before writing or JSON decoders on the client side may break. |

## Anti-Patterns

- Adding stateful fields or methods to this package — both types are pure function types; keep it that way
- Returning false from an ErrorEncoder after already writing to ResponseWriter — downstream encoders will attempt a second write, corrupting the response
- Using ResponseEncoder to also handle errors — keep error path in ErrorEncoder chain so the caller (httptransport.Handler) can control ordering

## Decisions

- **Separate ResponseEncoder and ErrorEncoder into distinct function types rather than a single Encoder interface** — Allows httptransport.Handler to apply an ordered ErrorEncoder chain before falling back to a success ResponseEncoder, giving callers fine-grained control over error mapping priority (e.g. ValidationIssue → 400 before generic 500).

## Example: Implementing a JSON ResponseEncoder and a domain-error ErrorEncoder for a new handler

```
import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

var encodeResponse encoder.ResponseEncoder[MyResponse] = func(
	ctx context.Context, w http.ResponseWriter, r *http.Request, resp MyResponse,
) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(resp)
}

// ...
```

<!-- archie:ai-end -->
