# httpdriver

<!-- archie:ai-start -->

> Legacy v1 HTTP layer for the customer API. Builds httptransport handlers (List/Create/Update/Delete/Get + entitlement value/access) and maps between api.* DTOs and customer domain types.

## Patterns

**httptransport three-function handler** — Each endpoint is a method on *handler returning a typed httptransport.Handler[...] built from a request-decoder, a business func, and a response encoder, plus AppendOptions with WithOperationName. (`return httptransport.NewHandler(decode, func(ctx, req)(Resp,error){...}, commonhttp.JSONResponseEncoderWithStatus[Resp](http.StatusCreated), httptransport.AppendOptions(h.options, httptransport.WithOperationName("createCustomer"))...)`)
**Namespace resolved from decoder** — Every decoder calls h.resolveNamespace(ctx), which reads namespaceDecoder.GetNamespace and returns a 500 HTTPError if absent. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Request{}, err }`)
**Map* / *ToAPI / FromAPI conversion functions** — Conversions live in apimapping.go: MapCustomerCreate/MapCustomerReplaceUpdate (api→domain CustomerMutate), CustomerToAPI/MapAccessToAPI(V2) (domain→api), MapAddress, FromMetadata/FromAnnotations. (`func MapCustomerCreate(body api.CustomerCreate) customer.CustomerMutate { ... }`)
**v1 filter semantics preserved via FilterString wrappers** — containsFilter wraps *string into &filter.FilterString{Contains} (partial match) and eqFilter into {Eq} (exact) so legacy query-param behavior is retained when delegating to the v3 filter-based service. (`Key: containsFilter(params.Key), PlanKey: eqFilter(params.PlanKey)`)
**Mutations guard against deleted customers** — Update/Delete/entitlement handlers first GetCustomer, then return models.NewGenericPreConditionFailedError if cus.IsDeleted() before proceeding. (`if cus != nil && cus.IsDeleted() { return ..., models.NewGenericPreConditionFailedError(...) }`)
**Subscriptions fetched and joined separately** — ListCustomers fetches subscriptions in a second subscriptionService.List call keyed by customer IDs, GroupBy CustomerId, then maps via CustomerToAPI; entitlement NotFound is mapped to NoAccessValue. (`customerSubscriptions = lo.GroupBy(subscriptions.Items, func(item subscription.Subscription) string { return item.CustomerId })`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler/CustomerHandler interfaces, *handler struct (service, entitlementService, subscriptionService, namespaceDecoder, options), New() constructor, resolveNamespace. | Constructor wires four services; missing namespace yields a 500, not a 400. |
| `customer.go` | All endpoint handlers and request/response type aliases; mapCustomerWithSubscriptionsToAPI joins customers with their subscriptions. | ListCustomers defaults Expand to [subscriptions] (TODO[v2] breaking change to remove); CurrentSubscriptionId assumes single subscription per customer (FIXME). |
| `apimapping.go` | DTO mapping incl. EntitlementValueV2/CustomerAccessV2 and MapEntitlementValueToAPIV2 (adds GrantBalances for MeteredEntitlementValue). | V2 entitlement mapping type-asserts *meteredentitlement.MeteredEntitlementValue to copy GrantBalances; currency/country mapped via lo.ToPtr(currencyx.Code/models.CountryCode). |

## Anti-Patterns

- Allowing mutations on a deleted customer without the IsDeleted precondition check
- Hand-decoding bodies instead of commonhttp.JSONRequestBodyDecoder, or returning bare errors instead of commonhttp/models typed errors
- Adding business logic to handlers instead of delegating to customer.Service / entitlement.Service
- Skipping resolveNamespace and trusting a namespace from the request body/path

## Decisions

- **Keep v1 contains/eq query-param semantics by wrapping params in filter.FilterString** — Lets the legacy v1 surface delegate to the shared v3 filter-based service without breaking existing partial/exact-match behavior.
- **Fetch subscriptions in a separate service call and join in the handler** — Customer service stays subscription-agnostic; the HTTP layer composes the expanded API representation.

## Example: A typed httptransport handler with namespace resolution and operation name

```
func (h *handler) CreateCustomer() CreateCustomerHandler {
	return httptransport.NewHandler(
		func(ctx context.Context, r *http.Request) (CreateCustomerRequest, error) {
			body := api.CustomerCreate{}
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil {
				return CreateCustomerRequest{}, fmt.Errorf("field to decode create customer request: %w", err)
			}
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return CreateCustomerRequest{}, err }
			return CreateCustomerRequest{Namespace: ns, CustomerMutate: MapCustomerCreate(body)}, nil
		},
		func(ctx context.Context, request CreateCustomerRequest) (CreateCustomerResponse, error) {
			customer, err := h.service.CreateCustomer(ctx, request)
			if err != nil { return CreateCustomerResponse{}, err }
			return h.mapCustomerWithSubscriptionsToAPI(ctx, *customer, nil)
// ...
```

<!-- archie:ai-end -->
