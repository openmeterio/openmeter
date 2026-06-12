# httptransport

<!-- archie:ai-start -->

> Generic, type-parameterized HTTP handler abstraction (Handler[Request,Response] and HandlerWithArgs) that wraps an operation.Operation in a fixed decode -> operate -> encode pipeline. This is the core handler primitive every v3 handler and most openmeter/*/httpdriver packages build on (47 in-edges); its primary constraint is that it stays decoupled from any concrete domain — it only knows operations, encoders, and error encoders.

## Patterns

**decode -> operate -> encode pipeline** — ServeHTTP runs exactly three stages: decodeRequest, operation, encodeResponse. A non-nil error from decode or operate is routed through encodeError; an encodeResponse error is treated as terminal and only passed to errorHandler.HandleContext. (`request, err := h.decodeRequest(ctx, r); response, err := h.operation(ctx, request); h.encodeResponse(ctx, w, r, response)`)
**Constructors return interfaces, struct stays unexported** — NewHandler/NewHandlerWithArgs return the Handler/HandlerWithArgs interface; the implementing handler[Request,Response] and handlerWithArgs structs are unexported. newHandler (lowercase) is the shared internal builder used by both public constructors. (`func NewHandler[Request any, Response any](...) Handler[Request, Response] { return newHandler(...) }`)
**Non-pointer receivers for cheap immutable clones** — handler and handlerWithArgs use value receivers so With() and Chain() copy the struct and mutate the copy. Comments explicitly warn: if the receiver becomes a pointer, an explicit clone is required in With/Chain. (`func (h handlerWithArgs[...]) With(arg ArgType) Handler[...] { res := h.handler; res.decodeRequest = func(...){...}; return res }`)
**Functional options via HandlerOption** — Configuration is applied through HandlerOption (an interface implemented by optionFunc). WithErrorHandler, WithErrorEncoder, WithOperationName, WithOperationNameFunc are the only knobs. defaultHandlerOptions (commonhttp.GenericErrorEncoder) is appended after caller options in newHandler. (`options = append(options, defaultHandlerOptions...); h.apply(options)`)
**Layered error encoding with bool-handled signaling** — encodeError iterates h.errorEncoders, then tries the SelfEncodingError interface on the error, falling back to a 500 StatusProblem. Each encoder returns bool; the first true wins. If nothing handles it, errorHandler.HandleContext is called for diagnostics. (`if errorEncoder(ctx, err, w, r) { return true } ... if encoder, ok := err.(SelfEncodingError); ok { ... }`)
**Operation middleware via Chain** — Both Handler and HandlerWithArgs expose Chain(outer, others...) that wraps h.operation through operation.Chain(...). Middleware composes around the operation, not the HTTP layer. (`h.operation = operation.Chain(outer, others...)(h.operation)`)
**Optional per-operation OTel span gated by global toggle** — operationNameFunc sets the route attr; when operationSpansEnabled (set once via EnableOperationSpans) is true, ServeHTTP starts a child span named after the operation. Off by default to avoid API-wide span volume. (`if operationSpansEnabled.Load() { ctx, span = tracer.Start(ctx, name); defer span.End() }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface, NewHandler, the unexported handler struct, RequestDecoder, ErrorHandler, SelfEncodingError, the ServeHTTP pipeline, encodeError, and the EnableOperationSpans/tracer globals. | ServeHTTP uses a value receiver; do not switch handler to pointer methods without adding explicit clones in With/Chain. defaultHandlerOptions wires GenericErrorEncoder — removing it loses the fallback error encoder. |
| `argshandler.go` | HandlerWithArgs variant: With(ArgType) binds a route arg (e.g. path param) into the request decoder, returning a plain Handler. RequestDecoderWithArgs adds the ArgType parameter to the decoder signature. | With() relies on value-receiver copy semantics of handler; the inline comment warns this breaks if handler becomes a pointer. The bound closure captures arg per-With call. |
| `options.go` | HandlerOption/optionFunc functional-option machinery, the With* option constructors, handlerOptions aggregation, apply(), and resolveErrorHandler (returns dummyErrorHandler when none set). | errorEncoders is append-only across WithErrorEncoder calls (order matters in encodeError). resolveErrorHandler never returns nil — a missing handler silently swallows diagnostic errors via dummyErrorHandler. |

## Anti-Patterns

- Importing domain/openmeter packages here — this is the foundational transport primitive; it may only depend on pkg/contextx, pkg/framework/operation, pkg/framework/commonhttp, pkg/models, and the encoder subpackage.
- Changing handler/handlerWithArgs to pointer receivers without adding explicit struct clones in With()/Chain() — the immutable-copy contract that With relies on would silently share mutable decoders.
- Writing to the ResponseWriter directly inside the operation — output must flow through encodeResponse or an ErrorEncoder/SelfEncodingError so error handling and content negotiation stay centralized.
- Enabling operation spans unconditionally instead of via EnableOperationSpans — adds a child span to every request API-wide and inflates trace volume.
- Returning concrete handler structs from constructors instead of the Handler/HandlerWithArgs interface — callers depend on the interface and on Chain composition.

## Decisions

- **Handlers are generic over Request/Response and wrap an operation.Operation rather than embedding business logic.** — Keeps the transport layer domain-agnostic so every v3 handler and httpdriver reuses one decode/operate/encode pipeline with consistent error encoding and tracing.
- **Error handling is a chain of ErrorEncoders plus a SelfEncodingError escape hatch, with a 500 fallback.** — Lets each error type decide its own wire representation while guaranteeing every unhandled error still produces a structured StatusProblem and a diagnostic callback.
- **Per-operation OTel spans are global-toggle gated, off by default.** — The operation child span is useful for handler-vs-middleware timing but multiplies span count across the whole API, so it is opted into once at startup via EnableOperationSpans.

## Example: Build a route-arg-bound HTTP handler from an operation and serve it

```
import (
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport"
	"github.com/openmeterio/openmeter/pkg/framework/transport/httptransport/encoder"
)

h := httptransport.NewHandlerWithArgs[Request, Response, RouteArg](
	func(ctx context.Context, r *http.Request, arg RouteArg) (Request, error) { /* decode */ },
	op, // operation.Operation[Request, Response]
	encoder.ResponseEncoder[Response](func(ctx context.Context, w http.ResponseWriter, r *http.Request, resp Response) error { /* encode */ }),
	httptransport.WithOperationName("my-operation"),
)
h.With(routeArg).ServeHTTP(w, r)
```

<!-- archie:ai-end -->
