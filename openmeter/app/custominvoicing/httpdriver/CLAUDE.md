# httpdriver

<!-- archie:ai-start -->

> HTTP handler layer for the custominvoicing app — exposes DraftSyncronized, IssuingSyncronized, and UpdatePaymentStatus endpoints by decoding API requests, delegating to appcustominvoicing.SyncService, and encoding billing.StandardInvoice responses via the shared httptransport pattern.

## Patterns

**httptransport.NewHandlerWithArgs per endpoint** — Each endpoint is a method returning a typed HandlerWithArgs[Request, Response, Params]. The method inlines the decoder func (namespace extraction + body decode), operation func (validate + delegate to service), and encoder. (`return httptransport.NewHandlerWithArgs(decoderFn, operationFn, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("X"), httptransport.WithErrorEncoder(errorEncoder()))...)`)
**Type aliases for Request/Response/Params/Handler** — Each endpoint declares four type aliases at the top of its block: *Request = domain input, *Response = api type, *Params = path/query params struct, *Handler = typed handler alias. (`type DraftSyncronizedRequest = appcustominvoicing.SyncDraftInvoiceInput; type DraftSyncronizedHandler httptransport.HandlerWithArgs[DraftSyncronizedRequest, DraftSyncronizedResponse, DraftSyncronizedParams]`)
**Namespace resolved from context in decoder** — Every decoder calls h.resolveNamespace(ctx) first; on failure it returns an empty request and wraps the error. (`namespace, err := h.resolveNamespace(ctx); if err != nil { return DraftSyncronizedRequest{}, fmt.Errorf("failed to resolve namespace: %w", err) }`)
**Input.Validate() called at the start of the operation func** — The operation function validates the decoded request before calling the service — not inside the decoder. (`func(ctx context.Context, request DraftSyncronizedRequest) (DraftSyncronizedResponse, error) { if err := request.Validate(); err != nil { return DraftSyncronizedResponse{}, err } ... }`)
**Domain-specific error encoder chain** — errors.go defines errorEncoder() returning a chain of commonhttp.HandleErrorIfTypeMatches calls covering billing.NotFoundError, ValidationError, UpdateAfterDeleteError, ValidationIssue, AppError — attached to every handler via httptransport.WithErrorEncoder. (`return commonhttp.HandleErrorIfTypeMatches[billing.NotFoundError](ctx, http.StatusNotFound, err, w, billing.EncodeValidationIssues) || ...`)
**Response mapped via billinghttpdriver.MapStandardInvoiceToAPI** — All three endpoints return api.Invoice by delegating to billinghttpdriver.MapStandardInvoiceToAPI — never inline-constructing the API type. (`return billinghttpdriver.MapStandardInvoiceToAPI(invoice)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface definition, handler struct, New() constructor, and resolveNamespace helper. Entry point for wiring. | SyncService is the only service dependency — do not inject billing.Service directly here; access billing through SyncService. |
| `custominvoicing.go` | Three endpoint handler methods: DraftSyncronized, IssuingSyncronized, UpdatePaymentStatus. | Each method returns a new handler instance on every call — they are not cached on the struct. |
| `errors.go` | Centralized error encoder shared by all three handlers. | New billing error types must be added here to get correct HTTP status codes — missing an error type causes 500s. |
| `mapper.go` | API ↔ domain type conversions: mapUpsertStandardInvoiceResultFromAPI, mapFinalizeStandardInvoiceResultFromAPI, mapPaymentTriggerFromAPI. | mapPaymentTriggerFromAPI uses a switch with a default that returns GenericValidationError — new API trigger values must be added here. |

## Anti-Patterns

- Injecting billing.Service directly into handler — use SyncService as the single dependency
- Inline-constructing api.Invoice in operation functions — always delegate to billinghttpdriver.MapStandardInvoiceToAPI
- Skipping request.Validate() in the operation function
- Adding new error types to the billing domain without updating errorEncoder() in errors.go

## Decisions

- **SyncService as the sole handler dependency** — Keeps the HTTP layer thin — all business logic (app type validation, billing triggers) lives in the service layer.
- **Shared errorEncoder() instead of per-handler error mapping** — All three endpoints expose the same billing error surface; centralising avoids divergence.

## Example: Add a new endpoint GetInvoiceStatus that decodes an invoiceId param and returns api.Invoice

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
