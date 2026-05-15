# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the custominvoicing app — exposes DraftSyncronized, IssuingSyncronized, and UpdatePaymentStatus endpoints by decoding API requests, delegating to appcustominvoicing.SyncService, and encoding billing.StandardInvoice responses via the shared httptransport pattern.

## Patterns

**httptransport.NewHandlerWithArgs per endpoint** — Each endpoint is a method returning a typed HandlerWithArgs[Request, Response, Params] with inline decoder, operation, and encoder functions. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("X"), httptransport.WithErrorEncoder(errorEncoder()))...)`)
**Four type aliases per endpoint** — Each endpoint block declares four type aliases: *Request (domain input), *Response (api type), *Params (path/query params), *Handler (typed handler alias). (`type DraftSyncronizedRequest = appcustominvoicing.SyncDraftInvoiceInput; type DraftSyncronizedResponse = api.Invoice; type DraftSyncronizedParams = struct{ InvoiceID string }; type DraftSyncronizedHandler httptransport.HandlerWithArgs[...]`)
**Namespace resolved from context in decoder** — Every decoder calls h.resolveNamespace(ctx) first; on failure it returns an empty request and wraps the error. (`namespace, err := h.resolveNamespace(ctx); if err != nil { return DraftSyncronizedRequest{}, fmt.Errorf("failed to resolve namespace: %w", err) }`)
**input.Validate() at operation func entry** — The operation function validates the decoded request before calling the service — not inside the decoder. (`func(ctx context.Context, request DraftSyncronizedRequest) (DraftSyncronizedResponse, error) { if err := request.Validate(); err != nil { return DraftSyncronizedResponse{}, err } ... }`)
**Shared errorEncoder() attached to every handler** — errors.go defines errorEncoder() with a chain of HandleErrorIfTypeMatches covering all billing error types; attached via httptransport.WithErrorEncoder(errorEncoder()). (`commonhttp.HandleErrorIfTypeMatches[billing.NotFoundError](ctx, http.StatusNotFound, err, w, billing.EncodeValidationIssues) || commonhttp.HandleErrorIfTypeMatches[billing.ValidationError](...)`)
**Response mapped via billinghttpdriver.MapStandardInvoiceToAPI** — All three endpoints return api.Invoice by delegating to billinghttpdriver.MapStandardInvoiceToAPI — never inline-constructing the API type. (`return billinghttpdriver.MapStandardInvoiceToAPI(invoice)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface definition, handler struct, New() constructor, and resolveNamespace helper. Entry point for wiring. | SyncService is the only service dependency — do not inject billing.Service directly here. |
| `custominvoicing.go` | Three endpoint handler methods: DraftSyncronized, IssuingSyncronized, UpdatePaymentStatus. | Each method returns a new handler instance on every call — they are not cached on the struct. |
| `errors.go` | Centralized error encoder shared by all three handlers. | New billing error types must be added here — missing an error type causes 500s instead of correct status codes. |
| `mapper.go` | API to domain type conversions: mapUpsertStandardInvoiceResultFromAPI, mapFinalizeStandardInvoiceResultFromAPI, mapPaymentTriggerFromAPI. | mapPaymentTriggerFromAPI uses a switch with a default returning GenericValidationError — new API trigger values must be added to this switch. |

## Anti-Patterns

- Injecting billing.Service directly into handler — use SyncService as the single dependency
- Inline-constructing api.Invoice in operation functions — always delegate to billinghttpdriver.MapStandardInvoiceToAPI
- Skipping request.Validate() in the operation function
- Adding new billing error types without updating errorEncoder() in errors.go
- Caching handler instances on the struct — each endpoint method must return a new handler

## Decisions

- **SyncService as the sole handler dependency** — Keeps the HTTP layer thin — all business logic lives in the service layer, not the handler.
- **Shared errorEncoder() instead of per-handler error mapping** — All three endpoints expose the same billing error surface; centralising avoids divergence.

## Example: Add a new endpoint GetInvoiceStatus that decodes an invoiceId path param and returns api.Invoice

```
type (
	GetInvoiceStatusRequest  = appcustominvoicing.GetInvoiceStatusInput
	GetInvoiceStatusResponse = api.Invoice
	GetInvoiceStatusParams   = struct{ InvoiceID string `json:"invoiceId"` }
	GetInvoiceStatusHandler  httptransport.HandlerWithArgs[GetInvoiceStatusRequest, GetInvoiceStatusResponse, GetInvoiceStatusParams]
)

func (h *handler) GetInvoiceStatus() GetInvoiceStatusHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetInvoiceStatusParams) (GetInvoiceStatusRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return GetInvoiceStatusRequest{}, fmt.Errorf("failed to resolve namespace: %w", err) }
			return GetInvoiceStatusRequest{InvoiceID: billing.InvoiceID{ID: params.InvoiceID, Namespace: ns}}, nil
		},
		func(ctx context.Context, request GetInvoiceStatusRequest) (GetInvoiceStatusResponse, error) {
// ...
```

<!-- archie:ai-end -->
