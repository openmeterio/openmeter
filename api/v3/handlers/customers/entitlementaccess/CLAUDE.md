# entitlementaccess

<!-- archie:ai-start -->

> v3 HTTP handler for listing a customer's entitlement access across all entitlement types (metered, boolean, static), translating entitlement.EntitlementValue domain types to api.BillingEntitlementAccessResult at the v3 boundary.

## Patterns

**HandlerWithArgs constructor pattern** — Each operation returns a typed httptransport.HandlerWithArgs alias and builds it inline via NewHandlerWithArgs with decoder/operation/encoder closures. (`type ListCustomerEntitlementAccessHandler httptransport.HandlerWithArgs[ListCustomerEntitlementAccessRequest, ListCustomerEntitlementAccessResponse, CustomerID]`)
**Namespace resolution via injected resolver** — Every decoder calls h.resolveNamespace(ctx) first; never read namespace from path/query params. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**apierrors.GenericErrorEncoder in options** — Each NewHandlerWithArgs appends httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()) via AppendOptions so domain errors map to correct HTTP status codes. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("list-customer-entitlement-access"), httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))...`)
**Type-switch mapping isolated in mapping.go** — Domain-to-API translation lives only in mapping.go (type switch on entitlement.EntitlementValue); list.go operation closures call mappers and never convert inline. (`switch ent := entitlementValue.(type) { case *meteredentitlement.MeteredEntitlementValue: ...; case *booleanentitlement.BooleanEntitlementValue: ... }`)
**Deleted-customer guard before service call** — After fetching the customer, check cus.IsDeleted() and return apierrors.NewPreconditionFailedError before invoking entitlement services. (`if cus != nil && cus.IsDeleted() { return ..., apierrors.NewPreconditionFailedError(ctx, fmt.Sprintf(...)) }`)
**Deterministic sort after map iteration** — entitlement.GetAccess returns a map; sort results by FeatureKey before encoding for stable, client-visible ordering. (`sort.Slice(items, func(i, j int) bool { return items[i].FeatureKey < items[j].FeatureKey })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Declares Handler interface and private struct with New(); holds resolveNamespace, customerService, entitlementService, shared options. | All deps via constructor — no package-level vars or init(). Options are threaded to each operation via AppendOptions. |
| `list.go` | ListCustomerEntitlementAccess: resolves namespace+customerID, fetches customer, checks deletion, calls GetAccess, maps via mapEntitlementValueToAPI, sorts by feature key. | NoAccessValue entries are skipped (found=false); unknown entitlement types error. Result is sorted before encoding. |
| `mapping.go` | mapEntitlementValueToAPI: pure type-switch converting domain EntitlementValue subtypes to api.BillingEntitlementAccessResult, returning (found bool, result, error). | New subtypes must be added to the switch (default errors). NoAccessValue returns (false, zero, nil) — caller must check the bool. |

## Anti-Patterns

- Inline domain-to-API conversion in list.go instead of delegating to mapping.go
- Calling entitlementService.GetAccess without first fetching and deletion-checking the customer
- Omitting apierrors.GenericErrorEncoder from handler options
- Reading namespace from request params instead of h.resolveNamespace(ctx)
- Treating NoAccessValue as an error instead of returning found=false and skipping

## Decisions

- **bool return from mapEntitlementValueToAPI instead of filtering at the service layer** — GetAccess can return NoAccessValue (inactive) entries; filtering at the mapping boundary keeps the domain service contract clean and avoids leaking HTTP concerns.
- **Deterministic sort by FeatureKey after collecting results** — GetAccess returns a map with non-deterministic iteration; client-visible ordering must be stable.

## Example: Add a new operation following the HandlerWithArgs pattern

```
// handler.go — add to Handler interface:
GetCustomerEntitlementFoo() GetCustomerEntitlementFooHandler

// foo.go:
type GetCustomerEntitlementFooHandler httptransport.HandlerWithArgs[GetCustomerEntitlementFooRequest, GetCustomerEntitlementFooResponse, CustomerID]

func (h *handler) GetCustomerEntitlementFoo() GetCustomerEntitlementFooHandler {
    return httptransport.NewHandlerWithArgs(
        func(ctx context.Context, r *http.Request, customerID CustomerID) (GetCustomerEntitlementFooRequest, error) {
            ns, err := h.resolveNamespace(ctx)
            if err != nil { return GetCustomerEntitlementFooRequest{}, err }
            return GetCustomerEntitlementFooRequest{CustomerID: customer.CustomerID{Namespace: ns, ID: customerID}}, nil
        },
        // ... operation, encoder, options
    )
// ...
```

<!-- archie:ai-end -->
