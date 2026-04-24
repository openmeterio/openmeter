# httpdriver

<!-- archie:ai-start -->

> HTTP adapter layer for the v1 customer API — translates api.* request/response types to customer.Service calls using the httptransport.Handler pattern. No business logic; all domain decisions delegate to customer.Service, entitlement.Service, and subscription.Service.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs per endpoint** — Each endpoint is a method returning a typed handler (e.g. ListCustomersHandler). The method calls httptransport.NewHandler(decoder, operation, encoder, ...options) — decoder maps *http.Request to domain input, operation calls service, encoder writes JSON response. (`func (h *handler) ListCustomers() ListCustomersHandler { return httptransport.NewHandlerWithArgs(decoder, operation, commonhttp.JSONResponseEncoderWithStatus[ListCustomersResponse](http.StatusOK), httptransport.AppendOptions(h.options, httptransport.WithOperationName("listCustomers"))...) }`)
**Type aliases for handler request/response types** — Each handler block declares type aliases (ListCustomersRequest, ListCustomersResponse, ListCustomersHandler) before the handler method. This makes the handler signature self-documenting and consistent with generated server stubs. (`type (\n\tListCustomersResponse = pagination.Result[api.Customer]\n\tListCustomersRequest  = customer.ListCustomersInput\n\tListCustomersHandler  httptransport.HandlerWithArgs[ListCustomersRequest, ListCustomersResponse, ListCustomersParams]\n)`)
**Namespace resolved from context via namespaceDecoder** — Every handler calls h.resolveNamespace(ctx) which delegates to namespacedriver.NamespaceDecoder.GetNamespace — never reads namespace from URL params directly. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ListCustomersRequest{}, err }`)
**API-to-domain mapping in apimapping.go** — All conversions between api.* types and domain types live in apimapping.go (MapCustomerCreate, CustomerToAPI, MapAddress, etc.). Handler files import these helpers; they never inline conversion logic. (`req := CreateCustomerRequest{Namespace: ns, CustomerMutate: MapCustomerCreate(body)}`)
**compile-time Handler interface assertion** — handler.go declares var _ Handler = (*handler)(nil) to enforce the interface at compile time. (`var _ Handler = (*handler)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface declaration, handler struct with injected services, New() constructor, resolveNamespace helper, compile-time assertion. | Adding new endpoints: the method must be added to both the CustomerHandler interface and the var _ assertion will catch missing implementations. |
| `customer.go` | Handler method implementations for all CRUD and entitlement-value endpoints. Each method returns a typed handler value. | Deleted-customer guard (cus.IsDeleted() check) must precede mutations; missing this returns 412 PreConditionFailed correctly. |
| `apimapping.go` | All api.* <-> domain type conversions. Pure functions, no service calls. | MapCustomerCreate and MapCustomerReplaceUpdate are structurally identical — keep them in sync when customer fields are added. |

## Anti-Patterns

- Placing business logic (validation, conflict resolution) in decoder or encoder functions — all logic belongs in the domain service.
- Reading namespace from URL/query params directly instead of h.resolveNamespace(ctx).
- Inline API-to-domain conversion in handler methods instead of using apimapping.go helpers.
- Returning domain types (customer.Customer) directly from handlers instead of api.* types.
- Calling adapter methods directly from the handler — always go through the service interface.

## Decisions

- **Handler methods return typed handler values (ListCustomersHandler) rather than implementing http.Handler directly.** — httptransport.Handler provides uniform error encoding, OTel operation naming, and middleware chaining without duplicating that logic in each handler method.
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
