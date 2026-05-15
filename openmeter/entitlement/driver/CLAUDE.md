# driver

<!-- archie:ai-start -->

> V1 HTTP driver for the entitlement domain: adapts entitlement.Service and meteredentitlement.Connector to Chi HTTP handlers using httptransport.HandlerWithArgs, with subject-key-based customer resolution. Also provides the shared Parser singleton and MapEntitlementValueToAPI used by both v1 and v2 drivers.

## Patterns

**HandlerWithArgs typed triplet** — Each handler is declared as a type alias of httptransport.HandlerWithArgs[Request, Response, Params]. Request/Response/Params are defined as named types or type aliases immediately above each handler type declaration. (`type CreateEntitlementHandler httptransport.HandlerWithArgs[CreateEntitlementHandlerRequest, CreateEntitlementHandlerResponse, CreateEntitlementHandlerParams]`)
**Namespace from context only** — All handlers call h.resolveNamespace(ctx) via namespacedriver.NamespaceDecoder — namespace is never accepted as a URL or query parameter in v1. (`ns, err := h.resolveNamespace(ctx); if err != nil { return request, err }`)
**Customer resolution in operation closure, not decoder** — The decoder reads only URL params (subject key, entitlement ID). Customer resolution via resolveCustomerFromSubject happens in the operation closure. UsageAttribution is populated after resolution before calling the service. (`cust, err := h.resolveCustomerFromSubject(ctx, request.Namespace, request.SubjectIdOrKey); request.Inputs.UsageAttribution = cust.GetUsageAttribution()`)
**Parser singleton for entitlement type dispatch** — Parser.ToAPIGeneric dispatches on EntitlementType to ToMetered/ToStatic/ToBoolean. Always use Parser.ToAPIGeneric rather than switching on EntitlementType inline in handlers. (`return Parser.ToAPIGeneric(&entitlement.EntitlementWithCustomer{Entitlement: lo.FromPtr(res), Customer: *cust})`)
**GetErrorEncoder() for domain error mapping** — All handlers pass httptransport.WithErrorEncoder(GetErrorEncoder()) in their options. GetErrorEncoder maps feature.FeatureNotFoundError, entitlement.NotFoundError, AlreadyExistsError (with conflictingEntityId), InvalidValueError, InvalidFeatureError, WrongTypeError to HTTP status codes. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("createEntitlement"), httptransport.WithErrorEncoder(GetErrorEncoder()))...`)
**Separate connector fields for base and metered ops** — meteredEntitlementHandler holds both entitlementConnector (entitlement.Service) for base entitlement queries and balanceConnector (meteredentitlement.Connector) for grant/balance ops. Never cross-use them. (`h.balanceConnector.CreateGrant(...) for grant ops; h.entitlementConnector.ListEntitlements(...) for base entitlement queries`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | EntitlementHandler interface + entitlementHandler struct: CreateEntitlement, OverrideEntitlement, GetEntitlement, GetEntitlementById, DeleteEntitlement, GetEntitlementValue, GetEntitlementsOfSubject, ListEntitlements. | CreateEntitlement passes empty streaming.CustomerUsageAttribution{} to ParseAPICreateInput then overwrites it in the operation closure after resolving the customer. ListEntitlements supports both page-based and limit/offset pagination via commonhttp.Union response type. |
| `metered.go` | MeteredEntitlementHandler: CreateGrant, ListEntitlementGrants, ResetEntitlementUsage, GetEntitlementBalanceHistory. resolveCustomerFromSubject bridges subject key to customer ID. | resolveCustomerFromSubject checks IsDeleted() before proceeding. GetEntitlementBalanceHistory returns both windowed and burndown history in one response mapped to GrantBurnDownHistorySegment. |
| `parser.go` | Parser package-level singleton with ToMetered/ToStatic/ToBoolean/ToAPIGeneric for domain→API mapping. ParseAPICreateInput for API→domain. MapEntitlementValueToAPI used by both v1 and v2 drivers and balanceworker. | ParseAPICreateInput explicitly prunes ActiveFrom/ActiveTo (request.ActiveFrom = nil) — lifecycle fields are not accepted via the create API. MapRecurrenceToAPI is best-effort for ISO durations that don't map to named enums. |
| `errors.go` | GetErrorEncoder() returns the domain-specific error encoder for this package. Called by all handlers in entitlement.go and metered.go. | AlreadyExistsError encoder adds conflictingEntityId to the response extensions map. New domain errors must be registered here — omitting causes fallthrough to generic 500. |

## Anti-Patterns

- Switching on EntitlementType inline in handler closures instead of calling Parser.ToAPIGeneric
- Accepting namespace as a query/path parameter — always resolve via h.namespaceDecoder.GetNamespace(ctx)
- Using balanceConnector methods for base entitlement operations or entitlementConnector for grant/balance ops
- Adding new domain error types to handler closures instead of GetErrorEncoder()
- Calling entitlement service methods with a raw subject key string instead of first resolving to customer.ID via resolveCustomerFromSubject

## Decisions

- **Subject-key-centric v1 driver with customer resolution inside the operation closure** — V1 API predates customer-ID-centric v2; subject key is the public identifier in v1 but customer.ID is required internally. Resolving in the operation closure keeps the decoder stateless and allows lazy resolution only when needed.
- **Parser as package-level singleton shared between v1 and v2 drivers** — Stateless type-dispatch logic is shared between v1 driver (this package) and v2 driver; singleton avoids repeated construction while keeping per-subtype mapping methods organized.

## Example: Adding a new v1 handler that reads an entitlement by subject key

```
type (
	GetMyHandlerRequest  = struct{ Namespace, SubjectKey, EntitlementID string }
	GetMyHandlerResponse = *api.Entitlement
	GetMyHandlerParams   = struct{ SubjectKey, EntitlementID string }
)
type GetMyHandler httptransport.HandlerWithArgs[GetMyHandlerRequest, GetMyHandlerResponse, GetMyHandlerParams]

func (h *entitlementHandler) GetMy() GetMyHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, p GetMyHandlerParams) (GetMyHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return GetMyHandlerRequest{}, err }
			return GetMyHandlerRequest{Namespace: ns, SubjectKey: p.SubjectKey, EntitlementID: p.EntitlementID}, nil
		},
		func(ctx context.Context, req GetMyHandlerRequest) (GetMyHandlerResponse, error) {
// ...
```

<!-- archie:ai-end -->
