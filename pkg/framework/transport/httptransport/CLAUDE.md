# httptransport

<!-- archie:ai-start -->

> Generic typed HTTP handler pipeline (decode → operation → encode) shared by every domain httpdriver and api/v3/handlers package — Handler[Request,Response] is the universal adapter between Chi routes and domain service calls. Its encoder/ child declares the pure ResponseEncoder/ErrorEncoder function types that break the import cycle with commonhttp.

## Patterns

**NewHandler triple constructor** — Always construct via NewHandler(requestDecoder, op, responseEncoder, ...options); never instantiate handler{} directly — defaultHandlerOptions (GenericErrorEncoder) are injected only by newHandler. (`httptransport.NewHandler(decodeListFoos, svc.ListFoos, encodeListFoos, httptransport.WithOperationName("foo.list"))`)
**HandlerWithArgs for path-param injection** — Bake a Chi URL param into the decoder via NewHandlerWithArgs + .With(arg) at mount time; With() clones the value-receiver handler binding the arg into decodeRequest. (`h := httptransport.NewHandlerWithArgs(decodeGetFoo, op, encodeFoo); h.With(chi.URLParam(r, "id")).ServeHTTP(w, r)`)
**ErrorEncoder chain, first match wins** — WithErrorEncoder appends to errorEncoders; ServeHTTP iterates in order and the first returning true short-circuits. GenericErrorEncoder is appended last by defaultHandlerOptions, so custom encoders must be passed as options to NewHandler. (`httptransport.NewHandler(dec, op, enc, httptransport.WithErrorEncoder(billingErrorEncoder))`)
**Chain for cross-cutting middleware** — handler.Chain(outer, ...others) wraps the operation with operation.Middleware and returns a new handler value; the original is unmodified. (`secured := h.Chain(authmiddleware.RequireScope("billing:write"))`)
**SelfEncodingError escape hatch** — Errors implementing SelfEncodingError (EncodeError(ctx, w) bool) bypass the ErrorEncoder chain; reserved for infrastructure error types like models.StatusProblem, not domain errors. (`// models.StatusProblem implements SelfEncodingError and calls Respond(w) directly`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Core Handler[Req,Resp] interface and internal handler struct; ServeHTTP orchestrates decode→operation→encode; encodeError iterates the chain, then SelfEncodingError, then falls back to a 500 problem+json. defaultHandlerOptions appends GenericErrorEncoder unconditionally. | Do not pass GenericErrorEncoder again via WithErrorEncoder — it is already appended; a second copy double-encodes. |
| `argshandler.go` | HandlerWithArgs variant for route-param injection; With() clones the value-receiver handler and swaps decodeRequest; Chain() rebuilds the operation. | Both With() and Chain() rely on value-receiver copy semantics — adding pointer fields to handler would share state across copies and require explicit deep copy. |
| `options.go` | HandlerOption interface and all WithX constructors; the errorEncoders slice is order-sensitive (append); resolveErrorHandler returns dummyErrorHandler when none set; AppendOptions helper. | Custom domain encoders passed after defaultHandlerOptions land after GenericErrorEncoder and never fire — always pass custom encoders as options to NewHandler. |
| `encoder/encoder.go` | Declares ResponseEncoder[T] and ErrorEncoder pure function types with no logic — imported by both handler.go and commonhttp without a cycle. | An ErrorEncoder must return false if it did not write to ResponseWriter; returning true after a partial write corrupts the response. |

## Anti-Patterns

- Instantiating handler{} struct directly — bypasses defaultHandlerOptions and loses GenericErrorEncoder
- Calling context.Background() inside a RequestDecoder — always use the ctx passed from ServeHTTP
- Implementing SelfEncodingError on domain-level errors — reserve it for infrastructure types; domain errors belong in the ErrorEncoder chain
- Adding state (mutexes, caches) to HandlerWithArgs or handler — value types cloned by With()/Chain() would race on shared pointer fields
- Writing to ResponseWriter inside a RequestDecoder on error — return the error and let the ErrorEncoder chain handle it

## Decisions

- **Value-receiver handler with copy-on-With/Chain semantics** — Enables safe per-request arg injection and middleware wrapping without allocation or locking; the registered handler is immutable after construction.
- **Last-resort fallback to 500 lives in handler.go, not in GenericErrorEncoder** — Keeps the fallback authoritative and unconfigurable — any error not handled by the chain or SelfEncodingError always yields a well-formed 500 problem+json.
- **Separate encoder sub-package for the encoder function types** — Breaks the import cycle between handler.go (which uses both types) and commonhttp (which defines GenericErrorEncoder); encoder has no dependencies so both can import it.

## Example: Minimal domain HTTP handler wired into a Chi router

```
import (
	"context"
	"net/http"

	"github.com/openmeterio/openmeter/pkg/framework/commonhttp"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
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
