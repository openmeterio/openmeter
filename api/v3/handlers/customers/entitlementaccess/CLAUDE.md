# entitlementaccess

<!-- archie:ai-start -->

> v3 HTTP handler for listing customer entitlement access across all entitlement types (metered, boolean, static). Translates entitlement.EntitlementValue domain types to api.BillingEntitlementAccessResult API types at the v3 API boundary.

## Patterns

**HandlerWithArgs constructor pattern** — Each operation method returns a typed httptransport.HandlerWithArgs[Request, Response, PathArg] alias. The method builds the handler inline via httptransport.NewHandlerWithArgs with three closures: decoder (resolves namespace + path arg to domain input), operation (calls domain services), and encoder. (`type ListCustomerEntitlementAccessHandler httptransport.HandlerWithArgs[ListCustomerEntitlementAccessRequest, ListCustomerEntitlementAccessResponse, CustomerID]`)
**Namespace resolution via injected resolver** — All decoders call h.resolveNamespace(ctx) as the first step. Never read namespace from path or query params directly. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**apierrors.GenericErrorEncoder in options** — Every httptransport.NewHandlerWithArgs call appends httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()) via httptransport.AppendOptions so domain errors map to correct HTTP status codes. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("list-customer-entitlement-access"), httptransport.WithErrorEncoder(apierrors.GenericErrorEncoder()))...`)
**Type-switch mapping in separate mapping.go** — Domain-to-API type translation lives exclusively in mapping.go using a type switch on entitlement.EntitlementValue. Operation closures in list.go call mappers and never do inline conversion. (`switch ent := entitlementValue.(type) { case *meteredentitlement.MeteredEntitlementValue: ... case *booleanentitlement.BooleanEntitlementValue: ... }`)
**Deleted-customer guard before service call** — After fetching the customer, check cus.IsDeleted() and return apierrors.NewPreconditionFailedError before invoking any entitlement service method. (`if cus != nil && cus.IsDeleted() { return ..., apierrors.NewPreconditionFailedError(ctx, fmt.Sprintf(...)) }`)
**Handler interface with one method per operation** — The exported Handler interface declares one method per HTTP operation returning the typed handler alias. The private handler struct implements it. New operations add a method to both the interface and the struct. (`type Handler interface { ListCustomerEntitlementAccess() ListCustomerEntitlementAccessHandler }`)
**Deterministic sort after map iteration** — entitlement.GetAccess returns a map; results must be sorted by FeatureKey before encoding to produce stable, client-visible ordering. (`sort.Slice(items, func(i, j int) bool { return items[i].FeatureKey < items[j].FeatureKey })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Declares Handler interface and private handler struct with constructor New(). Holds injected dependencies: resolveNamespace, customerService, entitlementService, and shared options slice. | All dependencies injected via constructor; never use package-level vars or init(). Options slice is passed through to each operation via httptransport.AppendOptions. |
| `list.go` | Implements ListCustomerEntitlementAccess operation: decoder resolves namespace+customerID, operation fetches customer, checks deletion, calls entitlementService.GetAccess, iterates entitlements calling mapEntitlementValueToAPI, sorts by feature key. | NoAccessValue entries are silently skipped (found=false check on mapEntitlementValueToAPI return); unknown entitlement types return an error. Result is sorted deterministically before encoding. |
| `mapping.go` | Contains mapEntitlementValueToAPI: pure type-switch function converting domain EntitlementValue subtypes to api.BillingEntitlementAccessResult. Returns (bool found, result, error). | New entitlement subtypes must be added to the switch; the default case returns an error. NoAccessValue returns (false, zero, nil) — caller must check the bool to skip the entry. |

## Anti-Patterns

- Inline domain-to-API type conversion inside list.go instead of delegating to mapping.go
- Calling entitlementService.GetAccess without first fetching and checking the customer for deletion
- Omitting apierrors.GenericErrorEncoder from handler options — domain errors will not map to correct HTTP status codes
- Reading namespace from HTTP request params directly instead of h.resolveNamespace(ctx)
- Handling NoAccessValue as an error instead of returning found=false and skipping the entry

## Decisions

- **bool return from mapEntitlementValueToAPI instead of filtering at the service layer** — entitlement.GetAccess can return NoAccessValue entries (inactive entitlements); filtering at the mapping boundary keeps the domain service contract clean and avoids leaking HTTP concerns into the service layer.
- **Deterministic sort by FeatureKey after collecting results** — entitlement.GetAccess returns a map; iteration order is non-deterministic. Client-visible ordering must be stable for reproducible API responses.

## Example: Add a new operation following the existing HandlerWithArgs pattern

```
// handler.go — add to Handler interface:
GetCustomerEntitlementFoo() GetCustomerEntitlementFooHandler

// foo.go:
type GetCustomerEntitlementFooHandler httptransport.HandlerWithArgs[GetCustomerEntitlementFooRequest, GetCustomerEntitlementFooResponse, CustomerID]

func (h *handler) GetCustomerEntitlementFoo() GetCustomerEntitlementFooHandler {
    return httptransport.NewHandlerWithArgs(
        func(ctx context.Context, r *http.Request, customerID CustomerID) (GetCustomerEntitlementFooRequest, error) {
            ns, err := h.resolveNamespace(ctx)
            if err != nil {
                return GetCustomerEntitlementFooRequest{}, err
            }
            return GetCustomerEntitlementFooRequest{CustomerID: customer.CustomerID{Namespace: ns, ID: customerID}}, nil
        },
// ...
```

<!-- archie:ai-end -->
