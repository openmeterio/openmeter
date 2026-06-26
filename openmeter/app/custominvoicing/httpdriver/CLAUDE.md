# httpdriver

<!-- archie:ai-start -->

> HTTP transport layer exposing the custom-invoicing app's external sync webhooks (draft synchronized, issuing synchronized, update payment status). Decodes api.* request bodies into appcustominvoicing/billing domain inputs and re-encodes billing.StandardInvoice as api.Invoice.

## Patterns

**httptransport.NewHandlerWithArgs triple** — Each endpoint returns a typed Handler built from (decode req, business call, response encoder) plus AppendOptions with WithOperationName and WithErrorEncoder. Request/Response/Params are type aliases to the underlying appcustominvoicing/api types. (`DraftSyncronizedHandler httptransport.HandlerWithArgs[DraftSyncronizedRequest, DraftSyncronizedResponse, DraftSyncronizedParams]`)
**Namespace resolution from decoder** — Every decode step calls h.resolveNamespace(ctx) which wraps namespaceDecoder.GetNamespace; a missing namespace yields a 500 commonhttp.NewHTTPError. InvoiceID is assembled from URL param + resolved namespace. (`namespace, err := h.resolveNamespace(ctx); ... InvoiceID: billing.InvoiceID{ID: params.InvoiceID, Namespace: namespace}`)
**Validate after decode, before service** — The business step calls request.Validate() first and returns the validation error directly so the errorEncoder maps it to the right status. (`if err := request.Validate(); err != nil { return DraftSyncronizedResponse{}, err }`)
**Centralized typed errorEncoder** — errors.go errorEncoder() chains commonhttp.HandleErrorIfTypeMatches for billing.NotFoundError(404), ValidationError/ValidationIssue(400), UpdateAfterDeleteError(409), AppError(400), each using billing.EncodeValidationIssues. (`commonhttp.HandleErrorIfTypeMatches[billing.NotFoundError](ctx, http.StatusNotFound, err, w, billing.EncodeValidationIssues) || ...`)
**FromAPI mapping helpers + reuse billing ToAPI** — mapper.go has mapUpsertStandardInvoiceResultFromAPI / mapFinalizeStandardInvoiceResultFromAPI / mapPaymentTriggerFromAPI converting api.* into billing builders (NewUpsertStandardInvoiceResult, NewFinalizeStandardInvoiceResult). Responses reuse billinghttpdriver.MapStandardInvoiceToAPI rather than hand-mapping. (`return billinghttpdriver.MapStandardInvoiceToAPI(invoice)`)
**API->internal trigger name translation** — mapPaymentTriggerFromAPI explicitly switches API trigger enums to billing.Trigger* constants; note the API 'payment_failed' maps to internal billing.TriggerFailed. Empty/unknown returns models.NewGenericValidationError. (`case api.CustomInvoicingPaymentTriggerPaymentFailed: return billing.TriggerFailed, nil`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `custominvoicing.go` | DraftSyncronized/IssuingSyncronized/UpdatePaymentStatus handler factories | service is appcustominvoicing.SyncService; each handler builds InvoiceID from params+namespace and delegates to h.service.Sync*/HandlePaymentTrigger |
| `handler.go` | Handler/AppHandler interfaces, handler struct, New constructor, resolveNamespace | handler holds SyncService + NamespaceDecoder + options; New takes them positionally |
| `errors.go` | errorEncoder mapping billing error types to HTTP status | ordering matters (first matching type wins); AppError is the apps-dependency fallthrough with no encoder |
| `mapper.go` | FromAPI mappers + payment-trigger translation | API trigger name 'payment_failed' != internal 'failed'; required-trigger and unknown cases must return validation errors |

## Anti-Patterns

- Hand-encoding the StandardInvoice response instead of reusing billinghttpdriver.MapStandardInvoiceToAPI
- Skipping request.Validate() before invoking the service
- Reading the namespace from the request body instead of resolveNamespace(ctx)
- Adding a new billing error type without registering it in errorEncoder()
- Assuming API trigger enum names equal internal billing.Trigger* values

## Decisions

- **Handlers depend only on appcustominvoicing.SyncService, not the full Service** — HTTP layer needs only the sync/payment surface; narrows coupling
- **Reuse billing httpdriver encoder and error types** — Custom-invoicing returns billing.StandardInvoice, so it shares billing's API mapping and validation-issue encoding

## Example: Define a sync webhook handler with decode, validate, delegate, encode

```
func (h *handler) DraftSyncronized() DraftSyncronizedHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params DraftSyncronizedParams) (DraftSyncronizedRequest, error) {
			namespace, err := h.resolveNamespace(ctx)
			if err != nil { return DraftSyncronizedRequest{}, fmt.Errorf("failed to resolve namespace: %w", err) }
			var body api.CustomInvoicingDraftSynchronizedRequest
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil { return DraftSyncronizedRequest{}, err }
			return DraftSyncronizedRequest{InvoiceID: billing.InvoiceID{ID: params.InvoiceID, Namespace: namespace}, UpsertInvoiceResults: mapUpsertStandardInvoiceResultFromAPI(body.Invoicing)}, nil
		},
		func(ctx context.Context, request DraftSyncronizedRequest) (DraftSyncronizedResponse, error) {
			if err := request.Validate(); err != nil { return DraftSyncronizedResponse{}, err }
			invoice, err := h.service.SyncDraftInvoice(ctx, request)
			if err != nil { return DraftSyncronizedResponse{}, err }
			return billinghttpdriver.MapStandardInvoiceToAPI(invoice)
		},
// ...
```

<!-- archie:ai-end -->
