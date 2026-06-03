# httpdriver

<!-- archie:ai-start -->

> v1 HTTP handler package for billing — translates between api.* types and billing.* service inputs/outputs, registers the billing errorEncoder, and exposes the composite Handler interface consumed by openmeter/server/router. Each endpoint is a method on *handler returning an httptransport handler, never ServeHTTP.

## Patterns

**Handler method returns typed httptransport handler** — Each operation is a method on *handler returning httptransport.Handler[Req,Resp] or HandlerWithArgs; it only constructs the handler, never calls ServeHTTP. (`func (h *handler) GetProfile() GetProfileHandler { return httptransport.NewHandlerWithArgs(decode, operate, encode, opts...) }`)
**Type alias triple per endpoint** — Each endpoint declares Request, Response, Params (and a Handler) type aliases at the top. (`type ( GetProfileRequest = billing.GetProfileInput; GetProfileResponse = api.BillingProfile; GetProfileParams struct { ID string }; GetProfileHandler httptransport.HandlerWithArgs[...] )`)
**resolveNamespace first in every decoder** — Every decode calls h.resolveNamespace(ctx) first; returns InternalServerError if missing. Namespace is never inlined. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., fmt.Errorf("failed to resolve namespace: %w", err) }`)
**errorEncoder() appended to every handler** — errors.go's errorEncoder() maps billing.NotFoundError→404, ValidationError→400, UpdateAfterDeleteError→409; appended via httptransport.AppendOptions on every handler. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("ListProfiles"), httptransport.WithErrorEncoder(errorEncoder()))`)
**Deprecated dual-field parsing centralized** — Endpoints accepting both rateCard and deprecated price/featureKey/taxConfig use mapAndValidateInvoiceLineRateCardDeprecatedFields from deprecations.go. (`rateCardParsed, err := mapAndValidateInvoiceLineRateCardDeprecatedFields(invoiceLineRateCardItems{ RateCard: line.RateCard, Price: line.Price })`)
**Pagination defaults from defaults.go** — List handlers use DefaultPageSize=100/DefaultPageNumber=1 via lo.FromPtrOr or defaultx.WithDefault, never hard-coded inline. (`Page: pagination.Page{ PageSize: defaultx.WithDefault(params.PageSize, DefaultPageSize), PageNumber: defaultx.WithDefault(params.Page, DefaultPageNumber) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Composite Handler interface (ProfileHandler + InvoiceLineHandler + InvoiceHandler + CustomerOverrideHandler), handler struct, New(). | New interface methods must be implemented on *handler and registered in router.Config or you get a compile error or 404. |
| `errors.go` | Single errorEncoder() mapping billing domain errors to HTTP status codes. | New billing error types must be added here; omitting them yields 500 instead of 4xx. |
| `deprecations.go` | Backward-compat parsing of legacy price/featureKey/taxConfig alongside rateCard. | Add deprecated bridge logic here; never inline dual-field compatibility in endpoint decoders. |
| `invoice.go` | Invoice-level operations: List/Get/Delete/Update/InvoicePendingLines/Simulate/ProgressInvoice. | ProgressInvoice uses a ProgressAction discriminator — add new actions to both InvoiceProgressActions and invoiceProgressOperationNames. |
| `profile.go` | Profile CRUD and MapProfileToApi (fromAPIBillingWorkflow, resolveProfileApps). | MapProfileToApi has two paths (p.Apps vs p.AppReferences) — keep both in sync when adding profile fields. |
| `defaults.go` | Package-level pagination/display defaults for list handlers. | Reference via lo.FromPtrOr/defaultx.WithDefault — never hard-code page size inline. |

## Anti-Patterns

- Calling billing.Service methods from a decode function — decode only parses; operate calls the service.
- Returning ValidationError/NotFoundError without errorEncoder() — yields 500 instead of 4xx.
- Adding Handler interface methods without implementing on *handler and updating handler.go's interface.
- Inlining deprecated-field compatibility logic outside deprecations.go.
- Using context.Background() instead of the caller's ctx in a handler closure.

## Decisions

- **Two-closure decode+operate httptransport pattern rather than a single ServeHTTP.** — Separates namespace/input parsing from business logic and lets the framework handle error encoding and OTel tracing uniformly.
- **errorEncoder() defined once and appended to every handler.** — Ensures consistent HTTP status codes for billing errors without repeating type-switch logic per endpoint.

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
        func(ctx context.Context, req FooRequest) (FooResponse, error) { /* call service */ },
        commonhttp.JSONResponseEncoderWithStatus[FooResponse](http.StatusOK),
// ...
```

<!-- archie:ai-end -->
