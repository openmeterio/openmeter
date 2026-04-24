# httptransport

<!-- archie:ai-start -->

> Generic typed HTTP handler pipeline (decode → operation → encode) shared by all domain httpdriver packages. The single Handler[Request,Response] struct is the universal adapter between Chi routes and domain service calls; every v1 and v3 HTTP endpoint is built on it.

## Patterns

**NewHandler triple constructor** — Always construct via NewHandler(requestDecoder, op, responseEncoder, ...options). Never instantiate handler{} directly — defaultHandlerOptions (GenericErrorEncoder) are injected by newHandler and would be missing. (`httptransport.NewHandler(decodeListFoosRequest, operation.Operation[ListFoosRequest, ListFoosResponse](svc.List), encodeListFoosResponse, httptransport.WithOperationName("foo.list"))`)
**HandlerWithArgs for path-param injection** — When a route parameter (e.g. Chi URL param) must be baked into the decoder, use NewHandlerWithArgs + .With(arg) at mount time. The With call clones the value-receiver handler, binding the arg into decodeRequest without allocation on each request. (`h := httptransport.NewHandlerWithArgs(decodeGetFoo, op, encodeFoo); r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) { h.With(chi.URLParam(r, "id")).ServeHTTP(w, r) })`)
**ErrorEncoder chain (first match wins)** — WithErrorEncoder appends to h.errorEncoders; handlers iterate them in order; the first returning true short-circuits. GenericErrorEncoder is appended last by defaultHandlerOptions — custom domain encoders must be passed before it via options. (`httptransport.NewHandler(dec, op, enc, httptransport.WithErrorEncoder(billingErrorEncoder), /* GenericErrorEncoder appended automatically */)`)
**Chain for cross-cutting middleware** — Use handler.Chain(outer, ...others) to wrap the operation with operation.Middleware (e.g. auth checks, rate limits). Chain returns a new handler value (value-receiver copy); the original is unmodified. (`secured := h.Chain(authmiddleware.RequireScope("billing:write"))`)
**SelfEncodingError escape hatch** — Errors that implement SelfEncodingError (EncodeError(ctx, w) bool) bypass the ErrorEncoder chain. Use only for error types that own their own HTTP status mapping (e.g. models.StatusProblem). Do not implement this on domain errors — use ErrorEncoder instead. (`// models.StatusProblem implements SelfEncodingError and calls Respond(w) directly`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Core Handler[Req,Resp] interface and internal handler struct. ServeHTTP orchestrates decode→operation→encode; encodeError iterates the chain then falls back to SelfEncodingError then 500. | defaultHandlerOptions injects GenericErrorEncoder unconditionally — do not add a second GenericErrorEncoder via WithErrorEncoder or errors will be double-encoded. |
| `argshandler.go` | HandlerWithArgs variant for route-param injection. With() clones the value-receiver handler and swaps decodeRequest; relies on handler being a non-pointer struct — if handler ever becomes a pointer, With() needs an explicit deep copy. | Both With() and Chain() rely on value-receiver copy semantics. If you add pointer fields to handler, these methods will share state across copies. |
| `options.go` | HandlerOption interface + all WithX constructors. HandlerOption.apply mutates handlerOptions; resolveErrorHandler returns dummyErrorHandler when none set. | errorEncoders slice is order-sensitive (append). Passing domain-specific encoders after NewHandler has already appended GenericErrorEncoder (via defaultHandlerOptions) means the domain encoder never fires — pass custom encoders as options to NewHandler, not after Chain. |
| `encoder/encoder.go` | Defines ResponseEncoder[T] and ErrorEncoder function types. Pure type definitions — no logic. | ErrorEncoder must return false if it did not write to ResponseWriter; returning true after a partial write leaves the response in an inconsistent state. |

## Anti-Patterns

- Instantiating handler{} struct directly — bypasses defaultHandlerOptions and loses GenericErrorEncoder
- Calling context.Background() inside a RequestDecoder — always use the ctx passed from ServeHTTP (already carries request-scoped values and cancellation)
- Implementing SelfEncodingError on domain-level errors — reserve it for infrastructure error types; domain errors belong in ErrorEncoder chain
- Adding state (mutexes, caches) to HandlerWithArgs or handler — these are value types cloned by With() and Chain(); shared state would race
- Writing to ResponseWriter inside RequestDecoder on error — return the error and let the ErrorEncoder chain handle it; writing before encodeError breaks status code negotiation

## Decisions

- **Value-receiver handler with copy-on-With/Chain semantics** — Enables safe per-request arg injection and middleware wrapping without allocation or locking; the original registered handler is immutable after construction.
- **ErrorEncoder chain with last-resort fallback to 500 in handler.go, not in GenericErrorEncoder** — Keeps the fallback authoritative and unconfigurable; any error not handled by the chain or SelfEncodingError always produces a well-formed 500 problem+json response.
- **Separate encoder sub-package for ResponseEncoder/ErrorEncoder function types** — Breaks the import cycle between handler.go (which uses both) and commonhttp (which defines GenericErrorEncoder) — encoder has no dependencies so both can import it.

## Example: Minimal domain HTTP handler wired into a Chi router

```
import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

func NewListFoosHandler(svc FooService) httptransport.Handler[ListFoosRequest, ListFoosResponse] {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ListFoosRequest, error) {
			return ListFoosRequest{Namespace: r.Header.Get("X-Namespace")}, nil
		},
		svc.ListFoos,
		func(ctx context.Context, w http.ResponseWriter, r *http.Request, resp ListFoosResponse) error {
// ...
```

<!-- archie:ai-end -->
