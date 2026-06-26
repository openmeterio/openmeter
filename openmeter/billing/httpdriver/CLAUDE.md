# httpdriver

<!-- archie:ai-start -->

> Legacy v1 HTTP transport for billing: translates api.* request/response types to/from billing domain inputs and delegates to billing.Service. Pure adapter layer — no business logic, only mapping, namespace resolution, and error encoding.

## Patterns

**httptransport handler triple** — Each endpoint is a method on *handler returning a typed httptransport.Handler[WithArgs] built from (request decoder, business call, response encoder) + AppendOptions(h.options, WithOperationName, WithErrorEncoder). Request/Response/Params are declared as type aliases at the top of each file. (`return httptransport.NewHandlerWithArgs(decode, func(ctx, in) (Resp,error){ return h.service.ListCustomerOverrides(ctx, in) }, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusOK), httptransport.AppendOptions(h.options, ...))`)
**Interface-segmented Handler** — The public Handler interface composes ProfileHandler, InvoiceLineHandler, InvoiceHandler, CustomerOverrideHandler; each method returns a typed handler constructor. New endpoints must be added to the matching sub-interface and implemented on *handler. (`type Handler interface { ProfileHandler; InvoiceLineHandler; InvoiceHandler; CustomerOverrideHandler }`)
**Namespace from decoder** — Every decoder calls h.resolveNamespace(ctx) which reads namespacedriver.NamespaceDecoder; failure returns a 500 HTTPError. Inputs are always namespace-scoped. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Req{}, fmt.Errorf(...) }`)
**Centralized billing error encoder** — errorEncoder() chains commonhttp.HandleErrorIfTypeMatches for billing.NotFoundError(404), ValidationError(400), UpdateAfterDeleteError(409), ValidationIssue(400), all rendered via billing.EncodeValidationIssues. Domain errors must be one of these types to map to the right status. (`commonhttp.HandleErrorIfTypeMatches[billing.ValidationError](ctx, http.StatusBadRequest, err, w, billing.EncodeValidationIssues)`)
**API<->entity mapping via lo pointer helpers** — Decoders unwrap optional api fields with lo.FromPtr / lo.FromPtrOr and build billing input structs; per-item conversions use slicesx.MapWithErr and dedicated mapXToAPI / mapXToEntity funcs. (`OrderBy: billing.CustomerOverrideOrderBy(lo.FromPtrOr(input.OrderBy, ...)), CustomerIDs: lo.FromPtr(input.CustomerId)`)
**ValidationIssues surfaced as errors** — When a returned invoice carries ValidationIssues, the handler converts them to a ValidationError via invoice.ValidationIssues.AsError() rather than returning a 200 with issues. (`if len(invoice.ValidationIssues) > 0 { return Resp{}, billing.ValidationError{Err: invoice.ValidationIssues.AsError()} }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface composition, handler struct, New constructor, resolveNamespace | handler holds service, appService, namespaceDecoder, featureSwitches (config.BillingFeatureSwitchesConfiguration), options; New also takes stripeAppService |
| `errors.go` | errorEncoder() mapping billing domain errors to HTTP statuses | Order matters — first matching type wins; all use billing.EncodeValidationIssues |
| `invoice.go` | List/Get/Delete/Progress/Simulate invoice + InvoicePendingLinesAction handlers | ProgressInvoice is parameterized by ProgressAction; surfaces invoice ValidationIssues as ValidationError |
| `invoiceline.go` | CreatePendingLine handler + mapCreateGatheringLineToEntity | Empty req.Lines returns billing.ValidationError; lines mapped per-item with slicesx.MapWithErr |
| `profile.go` | Profile CRUD handlers + MapProfileToApi | MapProfileToApi resolves app references via appService; keep nil-profile guards |
| `customeroverride.go` | Customer override list/get/upsert/delete + expand mapping | mapCustomerOverrideExpandToEntity translates api expand enums; reuses customerhttpdriver for nested customer mapping |
| `deprecations.go` | Backward-compat shims for deprecated request shapes | Has deprecations_test.go — keep shim behavior covered when changing field handling |

## Anti-Patterns

- Putting business logic (state transitions, calculation) in handlers instead of delegating to billing.Service
- Returning bare errors instead of billing.NotFoundError/ValidationError/ValidationIssue (breaks HTTP status mapping)
- Reading the namespace manually instead of h.resolveNamespace(ctx)
- Returning a 200 response while invoice.ValidationIssues is non-empty
- Hand-writing pointer/optional unwrapping instead of lo.FromPtr/FromPtrOr and slicesx.MapWithErr

## Decisions

- **All handlers built through httptransport with a shared errorEncoder** — Uniform error-to-status mapping and operation naming/telemetry across every billing endpoint without per-handler boilerplate
- **Request/Response/Params declared as type aliases of api.* and billing.* inputs** — Keeps the transport layer a thin, statically-checked mapping between OpenAPI types and domain inputs

## Example: A billing HTTP handler delegating to the service

```
func (h *handler) ListCustomerOverrides() ListCustomerOverridesHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, input ListCustomerOverridesParams) (ListCustomerOverridesRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil {
				return ListCustomerOverridesRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}
			return ListCustomerOverridesRequest{Namespace: ns, CustomerIDs: lo.FromPtr(input.CustomerId)}, nil
		},
		func(ctx context.Context, input ListCustomerOverridesRequest) (ListCustomerOverridesResponse, error) {
			return h.service.ListCustomerOverrides(ctx, input) // + map to API
		},
		commonhttp.JSONResponseEncoderWithStatus[ListCustomerOverridesResponse](http.StatusOK),
		httptransport.AppendOptions(h.options, httptransport.WithOperationName("ListCustomerOverrides"), httptransport.WithErrorEncoder(errorEncoder()))...,
	)
// ...
```

<!-- archie:ai-end -->
