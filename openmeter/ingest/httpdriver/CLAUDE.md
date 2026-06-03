# httpdriver

<!-- archie:ai-start -->

> HTTP transport layer for CloudEvent ingestion. Decodes incoming requests across three wire formats (application/json, application/cloudevents+json, application/cloudevents-batch+json) into ingest.IngestEventsRequest and delegates to ingest.Service. Mounted by openmeter/server/router; must stay DI-agnostic.

## Patterns

**httptransport.NewHandler triple** — Every endpoint is built with httptransport.NewHandler(decoderFn, operationFn, encoderFn, ...options). Decoder extracts namespace + body; operation calls the service; encoder writes the response. (`httptransport.NewHandler(decoder, func(ctx, req) (resp, error) { return h.service.IngestEvents(ctx, req) }, commonhttp.EmptyResponseEncoder[IngestEventsResponse](http.StatusNoContent), opts...)`)
**Handler interface composition** — handler.go defines a Handler interface composed of sub-interfaces (IngestHandler); each method returns a typed httptransport.Handler alias so callers depend on the narrowest surface. (`type Handler interface { IngestHandler }; type IngestHandler interface { IngestEvents() IngestEventsHandler }`)
**Namespace resolution via NamespaceDecoder** — Namespace is always resolved from context through h.resolveNamespace(ctx) which calls namespacedriver.NamespaceDecoder.GetNamespace(ctx). Never read namespace from URL path params. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx); if !ok { return commonhttp.NewHTTPError(http.StatusInternalServerError, ...) }`)
**Domain-local error types + errorEncoder chain** — errors.go defines package-local error structs (ErrorInvalidContentType, ErrorInvalidEvent) implementing Error()/Message()/Details(). errorEncoder() maps them to 400 via commonhttp.HandleErrorIfTypeMatches. (`commonhttp.HandleErrorIfTypeMatches[ErrorInvalidContentType](ctx, http.StatusBadRequest, err, w)`)
**Content-type switch for multi-format ingestion** — ingest.go switches on Content-Type to decode application/json (single or batch via AsEvent/AsIngestEventsBody1), cloudevents+json, and cloudevents-batch+json. Unknown types return ErrorInvalidContentType. (`switch contentType { case "application/cloudevents+json": ...; default: return req, ErrorInvalidContentType{ContentType: contentType} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Constructs the handler struct and exposes New(). All DI (NamespaceDecoder, ingest.Service, HandlerOptions) happens here. | Wiring only — no business logic. Always accept variadic httptransport.HandlerOption so callers can attach middleware. |
| `ingest.go` | Implements IngestEvents() endpoint: multi-format request decoding and service dispatch. Type aliases IngestEventsRequest/Response/Handler live here. | An application/json body can be a single event OR a batch — both AsEvent() and AsIngestEventsBody1() must be tried before returning ErrorInvalidEvent. |
| `errors.go` | Defines domain-local HTTP error types and the errorEncoder() factory passed to httptransport.NewHandler. | Error types must implement Error() and Message(). Register new errors in errorEncoder() via HandleErrorIfTypeMatches — raw errors bypass encoding and produce 500s. |
| `ingest_test.go` | Table-driven tests using httptest.NewServer + ingest.NewInMemoryCollector covering single event, batch, and invalid-body paths. | Tests use StaticNamespaceDecoder('test') — keep a deterministic namespace so collector.Events('test') is predictable. |

## Anti-Patterns

- Adding business logic (validation, transformation) inside the decoder — delegate to ingest.Service.
- Returning raw errors from the decoder without wrapping in ErrorInvalidEvent/ErrorInvalidContentType — they bypass the errorEncoder and produce 500s.
- Reading namespace from URL path params instead of h.resolveNamespace(ctx).
- Importing app/common or wire providers — this package must stay DI-agnostic.
- Defining a new endpoint without adding it to the Handler / IngestHandler interface composition in handler.go.

## Decisions

- **Content-type switch instead of a single decoder.** — CloudEvents spec defines three wire formats; dispatching by Content-Type is spec-correct and keeps each parse path isolated.
- **Domain-local error types instead of generic HTTP errors.** — ErrorInvalidContentType/ErrorInvalidEvent carry structured details (contentType) and are matched by type in errorEncoder to produce precise 400s.

## Example: Adding a new ingest endpoint following the existing pattern

```
type (
	MyNewRequest  = ingest.MyNewRequest
	MyNewResponse = struct{}
	MyNewHandler  httptransport.Handler[MyNewRequest, MyNewResponse]
)

func (h *handler) MyNew() MyNewHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (MyNewRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return MyNewRequest{}, err }
			return MyNewRequest{Namespace: ns}, nil
		},
		func(ctx context.Context, req MyNewRequest) (MyNewResponse, error) { /* call service */ return MyNewResponse{}, nil },
		commonhttp.EmptyResponseEncoder[MyNewResponse](http.StatusNoContent),
// ...
```

<!-- archie:ai-end -->
