# httpdriver

<!-- archie:ai-start -->

> v1 HTTP adapter layer for the customer API — translates api.* request/response types to customer.Service calls using the httptransport.Handler pattern. Contains no business logic; all domain decisions delegate to customer.Service, entitlement.Service, and subscription.Service.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs per endpoint** — Each endpoint is a method returning a named handler type. The method calls httptransport.NewHandler(decoder, operation, encoder, ...options). Decoder maps *http.Request to domain input, operation calls service, encoder writes JSON response. (`func (h *handler) ListCustomers() ListCustomersHandler { return httptransport.NewHandlerWithArgs(decoder, operation, commonhttp.JSONResponseEncoderWithStatus[ListCustomersResponse](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listCustomers"))...) }`)
**Type aliases for handler request/response/handler types** — Each endpoint block declares a type alias group (e.g. ListCustomersRequest, ListCustomersResponse, ListCustomersHandler) before the method. This makes each handler self-documenting and consistent with generated server stubs. (`type (
	ListCustomersResponse = pagination.Result[api.Customer]
	ListCustomersRequest  = customer.ListCustomersInput
	ListCustomersHandler  httptransport.HandlerWithArgs[ListCustomersRequest, ListCustomersResponse, ListCustomersParams]
)`)
**Namespace resolved via resolveNamespace(ctx)** — Every handler calls h.resolveNamespace(ctx) which delegates to namespacedriver.NamespaceDecoder.GetNamespace. Namespace is never read from URL params directly. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ListCustomersRequest{}, err }`)
**API-to-domain mapping in apimapping.go** — All conversions between api.* types and domain types live in apimapping.go (MapCustomerCreate, CustomerToAPI, MapAddress, etc.). Handler files import these helpers and never inline conversion logic. (`req := CreateCustomerRequest{Namespace: ns, CustomerMutate: MapCustomerCreate(body)}`)
**Deleted-customer guard before mutations** — UpdateCustomer, DeleteCustomer, and entitlement-value handlers call cus.IsDeleted() after fetching the customer and return models.NewGenericPreConditionFailedError if true, before proceeding. (`if cus != nil && cus.IsDeleted() { return UpdateCustomerResponse{}, models.NewGenericPreConditionFailedError(fmt.Errorf("customer is deleted")) }`)
**Compile-time Handler interface assertion** — handler.go declares var _ Handler = (*handler)(nil) to enforce the interface at compile time. (`var _ Handler = (*handler)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface declaration, handler struct with injected services, New() constructor, resolveNamespace helper, and compile-time assertion. | New endpoints must be added to both the CustomerHandler interface and the implementation; the var _ assertion catches missing implementations at compile time. |
| `customer.go` | Handler method implementations for all CRUD and entitlement-value endpoints. | Deleted-customer guard must precede mutations; mapCustomerWithSubscriptionsToAPI is a shared helper used in Create, Get, and Update — keep it consistent. |
| `apimapping.go` | All api.* to domain type conversions. Pure functions, no service calls. | MapCustomerCreate and MapCustomerReplaceUpdate are structurally identical — keep them in sync when customer fields are added. |

## Anti-Patterns

- Placing business logic (validation, conflict resolution) in decoder or encoder functions — all logic belongs in the domain service.
- Reading namespace from URL/query params directly instead of h.resolveNamespace(ctx).
- Inline API-to-domain conversion in handler methods instead of using apimapping.go helpers.
- Returning domain types (customer.Customer) directly from handlers instead of api.* types.
- Calling adapter methods directly from the handler — always go through the service interface.

## Decisions

- **Handler methods return typed handler values rather than implementing http.Handler directly.** — httptransport.Handler provides uniform error encoding, OTel operation naming, and middleware chaining without duplicating that boilerplate in each handler method.
- **apimapping.go is a dedicated file for API type conversions.** — Keeps handler files focused on request routing; makes the API contract surface easy to audit and test in isolation.

## Example: Add a new CRUD endpoint following the existing pattern

```
// In handler.go — add to CustomerHandler interface:
GetCustomerHistory() GetCustomerHistoryHandler

// In customer.go:
type (
	GetCustomerHistoryRequest  = customer.GetCustomerHistoryInput
	GetCustomerHistoryResponse = api.CustomerHistory
	GetCustomerHistoryHandler  httptransport.HandlerWithArgs[GetCustomerHistoryRequest, GetCustomerHistoryResponse, string]
)

func (h *handler) GetCustomerHistory() GetCustomerHistoryHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, customerIDOrKey string) (GetCustomerHistoryRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return GetCustomerHistoryRequest{}, err }
// ...
```

<!-- archie:ai-end -->
