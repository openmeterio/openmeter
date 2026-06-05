# httpdriver

<!-- archie:ai-start -->

> The v1 HTTP transport adapter for the ingest service: it parses the IngestEvents request body (single JSON event, JSON batch, single CloudEvent, or CloudEvent batch) and hands it to ingest.Service. It owns only HTTP concerns — content-type negotiation, namespace resolution, and error encoding — never business logic.

## Patterns

**httptransport.NewHandler decode/exec/encode triple** — Each endpoint method returns a typed handler built from a request decoder, a business-call closure, and a response encoder, plus options. No logic lives outside these three closures. (`func (h *handler) IngestEvents() IngestEventsHandler { return httptransport.NewHandler(decodeFn, func(ctx, params){ h.service.IngestEvents(ctx, params) }, commonhttp.EmptyResponseEncoder[...](http.StatusNoContent), ...) }`)
**Request type aliases to the service input** — Driver request type is a type alias of the service-layer request, not a separate struct, so the decoder fills the same type the service consumes. (`type IngestEventsRequest = ingest.IngestEventsRequest`)
**Content-Type switch over the API body unions** — The decoder switches on the Content-Type header and decodes into the matching generated api.* body type (api.IngestEventsBody, api.IngestEventsApplicationCloudeventsPlusJSONRequestBody, api.IngestEventsApplicationCloudeventsBatchPlusJSONBody), falling through to ErrorInvalidContentType on default. (`switch contentType { case "application/json": ...; case "application/cloudevents+json": ...; default: return req, ErrorInvalidContentType{ContentType: contentType} }`)
**Namespace from decoder, not request body** — Namespace is resolved from the context via h.namespaceDecoder.GetNamespace and assigned to req.Namespace; a missing namespace is a 500 internal error, never trusted from the client payload. (`ns, ok := h.namespaceDecoder.GetNamespace(ctx); if !ok { return "", commonhttp.NewHTTPError(http.StatusInternalServerError, ...) }`)
**Typed driver errors with errorEncoder** — Validation failures return ErrorInvalidEvent / ErrorInvalidContentType structs implementing Error()/Message()/Details(); errorEncoder() maps them to 400 via commonhttp.HandleErrorIfTypeMatches and is wired with httptransport.WithErrorEncoder. (`return commonhttp.HandleErrorIfTypeMatches[ErrorInvalidContentType](ctx, http.StatusBadRequest, err, w) || commonhttp.HandleErrorIfTypeMatches[ErrorInvalidEvent](...)`)
**Constructor injects collaborators, returns interface** — New(namespaceDecoder, service, options...) returns the Handler interface backed by the unexported handler struct; options are appended via httptransport.AppendOptions(h.options, ...). (`func New(namespaceDecoder namespacedriver.NamespaceDecoder, service ingest.Service, options ...httptransport.HandlerOption) Handler`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `ingest.go` | IngestEvents() handler: decodes the four supported content types into ingest.IngestEventsRequest and calls service.IngestEvents. | Single-vs-batch JSON parsing relies on AsEvent() then falling back to AsIngestEventsBody1(); if both yield zero events it must return ErrorInvalidEvent. Success returns 204 No Content via EmptyResponseEncoder. |
| `handler.go` | Handler/IngestHandler interfaces, handler struct, resolveNamespace, and the New constructor. | Do not read namespace from the request body — only resolveNamespace via namespaceDecoder is trusted; a missing namespace yields 500, not 400. |
| `errors.go` | Typed transport errors (ErrorInvalidContentType, ErrorInvalidEvent) and errorEncoder() mapping them to HTTP 400. | New error types must implement Message() (and optionally Details()) and be registered in errorEncoder() or they fall through to a generic 500. |
| `ingest_test.go` | httptest-based tests for single, invalid, and batch CloudEvent flows using ingest.NewInMemoryCollector and namespacedriver.StaticNamespaceDecoder. | Tests assert exact status codes (204 success, 400 invalid) and verify events landed in the in-memory collector keyed by namespace. |

## Anti-Patterns

- Putting dedupe/validation/persistence business logic in the handler instead of delegating to ingest.Service.
- Trusting a namespace value from the request payload instead of namespacedriver.GetNamespace.
- Returning raw errors from the decoder instead of ErrorInvalidEvent/ErrorInvalidContentType, which breaks the 400 mapping in errorEncoder.
- Decoding into hand-written structs instead of the generated api.* body union types.
- Writing a non-empty success body — ingest responds 204 No Content via EmptyResponseEncoder.

## Decisions

- **Driver request types are aliases of ingest.IngestEventsRequest rather than separate DTOs.** — Keeps the transport layer thin and avoids an extra mapping step between HTTP and the service input.
- **Content negotiation is explicit per CloudEvents media type.** — OpenMeter ingest accepts plain JSON, JSON batch, and the two CloudEvents content types; each maps to a distinct generated body type so the decoder stays unambiguous.

## Example: Adding/decoding an ingest content type and delegating to the service

```
func (h *handler) IngestEvents() IngestEventsHandler {
  return httptransport.NewHandler(
    func(ctx context.Context, r *http.Request) (IngestEventsRequest, error) {
      var req ingest.IngestEventsRequest
      ns, err := h.resolveNamespace(ctx)
      if err != nil { return req, err }
      req.Namespace = ns
      switch r.Header.Get("Content-Type") {
      case "application/cloudevents+json":
        var body api.IngestEventsApplicationCloudeventsPlusJSONRequestBody
        if err := json.NewDecoder(r.Body).Decode(&body); err != nil { return req, ErrorInvalidEvent{Err: err} }
        req.Events = []event.Event{body}
      default:
        return req, ErrorInvalidContentType{ContentType: r.Header.Get("Content-Type")}
      }
// ...
```

<!-- archie:ai-end -->
