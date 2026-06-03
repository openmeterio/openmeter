# driver

<!-- archie:ai-start -->

> V1 HTTP driver for the entitlement domain: adapts entitlement.Service and meteredentitlement.Connector to Chi handlers via httptransport.HandlerWithArgs with subject-key-based customer resolution. Also owns the shared Parser singleton and MapEntitlementValueToAPI reused by the v2 driver and balanceworker.

## Patterns

**HandlerWithArgs typed triplet** — Each handler is a type alias of httptransport.HandlerWithArgs[Request, Response, Params] with Request/Response/Params named immediately above. (`type CreateEntitlementHandler httptransport.HandlerWithArgs[CreateEntitlementHandlerRequest, CreateEntitlementHandlerResponse, CreateEntitlementHandlerParams]`)
**Namespace from context only** — All decoders call h.resolveNamespace(ctx) via namespacedriver.NamespaceDecoder; namespace is never a URL/query param in v1. (`ns, err := h.resolveNamespace(ctx); if err != nil { return request, err }`)
**Customer resolution in the operation closure** — Decoder reads only URL params; resolveCustomerFromSubject runs in the operation, then UsageAttribution is set before calling the service. (`cust, err := h.resolveCustomerFromSubject(ctx, request.Namespace, request.SubjectIdOrKey); request.Inputs.UsageAttribution = cust.GetUsageAttribution()`)
**Parser singleton for type dispatch** — Parser.ToAPIGeneric dispatches on EntitlementType to ToMetered/ToStatic/ToBoolean; never switch on EntitlementType inline. (`return Parser.ToAPIGeneric(&entitlement.EntitlementWithCustomer{Entitlement: lo.FromPtr(res), Customer: *cust})`)
**GetErrorEncoder() for domain error mapping** — Every handler passes httptransport.WithErrorEncoder(GetErrorEncoder()); new domain errors must be registered there or they fall through to 500. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("createEntitlement"), httptransport.WithErrorEncoder(GetErrorEncoder()))`)
**Separate connector fields for base vs metered ops** — meteredEntitlementHandler holds entitlementConnector (base queries) and balanceConnector (grant/balance ops); never cross-use them. (`h.balanceConnector.CreateGrant(...) for grants; h.entitlementConnector.ListEntitlements(...) for base queries`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | EntitlementHandler: CreateEntitlement, OverrideEntitlement, GetEntitlement(ById), DeleteEntitlement, GetEntitlementValue, GetEntitlementsOfSubject, ListEntitlements. | CreateEntitlement passes empty streaming.CustomerUsageAttribution{} to ParseAPICreateInput then overwrites after resolving the customer. ListEntitlements supports both page-based and limit/offset pagination via commonhttp.Union. |
| `metered.go` | MeteredEntitlementHandler: CreateGrant, ListEntitlementGrants, ResetEntitlementUsage, GetEntitlementBalanceHistory; resolveCustomerFromSubject bridges subject key→customer ID. | resolveCustomerFromSubject checks IsDeleted(); GetEntitlementBalanceHistory returns windowed + burndown history mapped to GrantBurnDownHistorySegment. |
| `parser.go` | Parser singleton (ToMetered/ToStatic/ToBoolean/ToAPIGeneric) and ParseAPICreateInput; MapEntitlementValueToAPI shared by v1/v2 drivers and balanceworker. | ParseAPICreateInput prunes ActiveFrom/ActiveTo (lifecycle fields not accepted via create). MapRecurrenceToAPI is best-effort for non-named ISO durations. |
| `errors.go` | GetErrorEncoder() returns the domain error encoder used by all handlers. | AlreadyExistsError adds conflictingEntityId to response extensions; unregistered domain errors fall through to generic 500. |

## Anti-Patterns

- Switching on EntitlementType inline in handler closures instead of calling Parser.ToAPIGeneric
- Accepting namespace as a query/path parameter — always resolve via h.namespaceDecoder.GetNamespace(ctx)
- Using balanceConnector for base entitlement ops or entitlementConnector for grant/balance ops
- Adding new domain error types to handler closures instead of GetErrorEncoder()
- Calling entitlement service methods with a raw subject key instead of first resolving to customer.ID via resolveCustomerFromSubject

## Decisions

- **Subject-key-centric v1 driver with customer resolution inside the operation closure** — Subject key is the public v1 identifier but customer.ID is required internally; resolving lazily in the operation keeps the decoder stateless.
- **Parser as a package-level singleton shared by v1 and v2 drivers** — Stateless type-dispatch logic is identical across drivers; a singleton avoids repeated construction while organising per-subtype mapping.

<!-- archie:ai-end -->
