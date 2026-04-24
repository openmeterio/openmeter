# httpdriver

<!-- archie:ai-start -->

> v1 HTTP handler package for the billing domain — translates between api.* types and billing.* service inputs/outputs, registers the billing errorEncoder, and exposes the composite Handler interface consumed by openmeter/server/router.

## Patterns

**Handler method returns typed httptransport.HandlerWithArgs or httptransport.Handler** — Each HTTP operation is a method on *handler that returns an httptransport.Handler[Req,Resp] or httptransport.HandlerWithArgs[Req,Resp,Params]. The handler method only declares the type, never calls ServeHTTP. (`func (h *handler) GetProfile() GetProfileHandler { return httptransport.NewHandlerWithArgs(decode, operate, encode, opts...) }`)
**Type alias triple for every endpoint** — Each endpoint declares Request, Response, and Params type aliases at the top of its function block or as package-level types, then a Handler type alias for the httptransport.Handler generic. (`type ( GetProfileRequest = billing.GetProfileInput; GetProfileResponse = api.BillingProfile; GetProfileParams struct { ID string; Expand []api.BillingProfileExpand }; GetProfileHandler httptransport.HandlerWithArgs[...] )`)
**resolveNamespace helper for all decoders** — Every decode function calls h.resolveNamespace(ctx) first; an InternalServerError is returned if the namespace is missing. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., fmt.Errorf("failed to resolve namespace: %w", err) }`)
**errorEncoder() composed from billing domain errors** — errors.go defines errorEncoder() returning a chain: billing.NotFoundError→404, billing.ValidationError→400, billing.UpdateAfterDeleteError→409, billing.ValidationIssue→400, billing.AppError→400. (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(errorEncoder()))`)
**mapAndValidateInvoiceLineRateCardDeprecatedFields for dual-field parsing** — When an endpoint accepts both rateCard and deprecated top-level price/featureKey/taxConfig fields, use mapAndValidateInvoiceLineRateCardDeprecatedFields to resolve them consistently. (`rateCardParsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{ RateCard: line.RateCard, Price: line.Price, ... })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `openmeter/billing/httpdriver/handler.go` | Defines the composite Handler interface (ProfileHandler + InvoiceLineHandler + InvoiceHandler + CustomerOverrideHandler), the handler struct, and the New() constructor. | New methods on the Handler interface must also be implemented on *handler and registered in router.Config. |
| `openmeter/billing/httpdriver/errors.go` | Single errorEncoder() for all billing HTTP endpoints; maps billing-domain error types to HTTP status codes. | New billing domain error types must be added here; omitting them causes 500 responses. |
| `openmeter/billing/httpdriver/deprecations.go` | Handles backward-compatibility parsing of legacy price/featureKey/taxConfig fields alongside the newer rateCard field. | When new invoice line fields are deprecated, add their bridge logic here; never inline dual-field logic into endpoint decoders. |
| `openmeter/billing/httpdriver/invoice.go` | All invoice-level HTTP operations: List, Get, Delete, Update, InvoicePendingLines, Simulate, ProgressInvoice (approve/retry/advance/snapshot_quantities). | ProgressInvoice uses a ProgressAction discriminator — add new actions to both InvoiceProgressActions and invoiceProgressOperationNames. |
| `openmeter/billing/httpdriver/profile.go` | Profile CRUD and MapProfileToApi; also contains fromAPIBillingWorkflow, resolveProfileApps, app reference mapping. | MapProfileToApi has two code paths (p.Apps set vs p.AppReferences) — keep both in sync when adding profile fields. |
| `openmeter/billing/httpdriver/defaults.go` | Package-level pagination and display defaults (DefaultPageSize=100, DefaultPageNumber=1, etc.). | All list handlers must use these defaults via lo.FromPtrOr — never hard-code page size inline. |

## Anti-Patterns

- Calling billing.Service methods directly from a decode function — decode must only parse, operate must call the service
- Returning billing.ValidationError without wrapping in errorEncoder — will cause a 500
- Adding new Handler interface methods without adding them to handler.go's interface definition
- Inlining deprecated-field compatibility logic outside deprecations.go
- Using context.Background() instead of the caller's ctx in any handler closure

## Decisions

- **Two-closure httptransport.Handler pattern (decode + operate) rather than a single ServeHTTP method** — Separates namespace/input parsing from business logic; allows the httptransport framework to handle error encoding and OTel tracing uniformly.
- **errorEncoder() defined once in errors.go, appended to every handler's options** — Ensures consistent HTTP status codes for billing domain errors across all endpoints without repeating type-switch logic.

## Example: Adding a new billing HTTP endpoint

```
type (
	FooRequest  = billing.FooInput
	FooResponse = api.BillingFoo
	FooParams   struct{ ID string }
	FooHandler  httptransport.HandlerWithArgs[FooRequest, FooResponse, FooParams]
)

func (h *handler) Foo() FooHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, p FooParams) (FooRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return FooRequest{}, fmt.Errorf("namespace: %w", err) }
			return billing.FooInput{Namespace: ns, ID: p.ID}, nil
		},
		func(ctx context.Context, req FooRequest) (FooResponse, error) {
// ...
```

<!-- archie:ai-end -->
