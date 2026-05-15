# httptransport

<!-- archie:ai-start -->

> Generic typed HTTP handler pipeline (decode → operation → encode) shared by all domain httpdriver and api/v3/handlers packages. Handler[Request,Response] is the universal adapter between Chi routes and domain service calls; every v1 and v3 endpoint is built on it.

## Patterns

**NewHandler triple constructor** — Always construct via NewHandler(requestDecoder, op, responseEncoder, ...options). Never instantiate handler{} directly — defaultHandlerOptions (GenericErrorEncoder) are injected by newHandler and would be missing. (`httptransport.NewHandler(decodeListFoos, svc.ListFoos, encodeListFoos, httptransport.WithOperationName("foo.list"))`)
**HandlerWithArgs for path-param injection** — When a Chi URL param must be baked into the decoder, use NewHandlerWithArgs + .With(arg) at mount time. With() clones the value-receiver handler, binding the arg into decodeRequest without allocation on each request. (`h := httptransport.NewHandlerWithArgs(decodeGetFoo, op, encodeFoo); r.Get("/{id}", func(w http.ResponseWriter, r *http.Request) { h.With(chi.URLParam(r, "id")).ServeHTTP(w, r) })`)
**ErrorEncoder chain (first match wins)** — WithErrorEncoder appends to h.errorEncoders; handlers iterate in order; the first returning true short-circuits. GenericErrorEncoder is appended last by defaultHandlerOptions — custom domain encoders must be passed as options to NewHandler, not after Chain. (`httptransport.NewHandler(dec, op, enc, httptransport.WithErrorEncoder(billingErrorEncoder))`)
**Chain for cross-cutting middleware** — Use handler.Chain(outer, ...others) to wrap the operation with operation.Middleware. Chain returns a new handler value (value-receiver copy); the original is unmodified. (`secured := h.Chain(authmiddleware.RequireScope("billing:write"))`)
**SelfEncodingError escape hatch** — Errors implementing SelfEncodingError (EncodeError(ctx, w) bool) bypass the ErrorEncoder chain. Use only for infrastructure error types (e.g. models.StatusProblem). Do not implement on domain errors — use ErrorEncoder instead. (`// models.StatusProblem implements SelfEncodingError and calls Respond(w) directly`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Core Handler[Req,Resp] interface and internal handler struct. ServeHTTP orchestrates decode→operation→encode; encodeError iterates the chain then falls back to SelfEncodingError then 500. defaultHandlerOptions appends GenericErrorEncoder unconditionally. | Do not pass GenericErrorEncoder again via WithErrorEncoder — it is already appended by defaultHandlerOptions; a second copy double-encodes errors. |
| `argshandler.go` | HandlerWithArgs variant for route-param injection. With() clones the value-receiver handler and swaps decodeRequest; relies on handler being a non-pointer struct. | Both With() and Chain() rely on value-receiver copy semantics. If pointer fields are ever added to handler, these methods will share state across copies — explicit deep copy required. |
| `options.go` | HandlerOption interface and all WithX constructors. errorEncoders slice is order-sensitive (append). resolveErrorHandler returns dummyErrorHandler when none set. | errorEncoders are appended in order; custom domain encoders passed after defaultHandlerOptions are appended after GenericErrorEncoder and will never fire — always pass custom encoders as options to NewHandler. |
| `encoder/encoder.go` | Declares ResponseEncoder[T] and ErrorEncoder pure function types. No logic — pure type definitions that both handler.go and commonhttp import without cycle. | ErrorEncoder must return false if it did not write to ResponseWriter; returning true after a partial write leaves the response in an inconsistent state. |

## Anti-Patterns

- Instantiating handler{} struct directly — bypasses defaultHandlerOptions and loses GenericErrorEncoder
- Calling context.Background() inside a RequestDecoder — always use the ctx passed from ServeHTTP
- Implementing SelfEncodingError on domain-level errors — reserve it for infrastructure types; domain errors belong in ErrorEncoder chain
- Adding state (mutexes, caches) to HandlerWithArgs or handler — value types cloned by With() and Chain() would race on shared pointer fields
- Writing to ResponseWriter inside RequestDecoder on error — return the error and let the ErrorEncoder chain handle it

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
)

func NewListFoosHandler(svc FooService) httptransport.Handler[ListFoosRequest, ListFoosResponse] {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (ListFoosRequest, error) {
			return ListFoosRequest{Namespace: r.Header.Get("X-Namespace")}, nil
		},
		svc.ListFoos,
		func(ctx context.Context, w http.ResponseWriter, r *http.Request, resp ListFoosResponse) error {
			return commonhttp.JSONResponseEncoder(ctx, w, resp)
// ...
```

<!-- archie:ai-end -->
