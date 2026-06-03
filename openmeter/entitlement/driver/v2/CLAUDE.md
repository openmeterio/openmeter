# v2

<!-- archie:ai-start -->

> V2 HTTP driver for the entitlement domain — adapts entitlement.Service and meteredentitlement.Connector to customer-scoped HTTP handlers via httptransport.HandlerWithArgs, always resolving a customer (by ID or key) before any entitlement operation. Differs from v1 by returning customerId/customerKey in all responses.

## Patterns

**HandlerWithArgs typed triplet** — Every endpoint declares Request, Response, Params type aliases and aliases the handler type as httptransport.HandlerWithArgs[Req, Resp, Params]; the entitlementHandler method returns that alias. (`type CreateCustomerEntitlementHandler httptransport.HandlerWithArgs[CreateCustomerEntitlementHandlerRequest, CreateCustomerEntitlementHandlerResponse, CreateCustomerEntitlementHandlerParams]`)
**Customer resolution before entitlement ops** — Each handler resolves the customer via h.customerService.GetCustomer with CustomerIDOrKey, then checks cus.IsDeleted() returning models.NewGenericPreConditionFailedError before any entitlement call. (`cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerIDOrKey: &customer.CustomerIDOrKey{Namespace: ns, IDOrKey: req.CustomerIDOrKey}}); if cus != nil && cus.IsDeleted() { return ..., models.NewGenericPreConditionFailedError(...) }`)
**Namespace resolved from context via namespaceDecoder** — All handlers call h.resolveNamespace(ctx) first (delegating to h.namespaceDecoder.GetNamespace(ctx)); HTTP 500 if missing. Never accept namespace from URL params. (`ns, err := h.resolveNamespace(ctx); if err != nil { return def, err }`)
**ParserV2.ToAPIGenericV2 for all entitlement responses** — Handlers returning an entitlement call ParserV2.ToAPIGenericV2(ent, cust.ID, cust.Key), which dispatches by EntitlementType to ToMeteredV2/ToStaticV2/ToBooleanV2 and the corresponding FromEntitlement*V2 setter. Never inline the mapping. (`v2, err := ParserV2.ToAPIGenericV2(ent, cus.ID, cus.Key); if err != nil { return api.EntitlementV2{}, err }; return *v2, nil`)
**Error encoder chains v1 then generic** — getErrorEncoder() composes entitlementdriver.GetErrorEncoder() (v1) with commonhttp.GenericErrorEncoder() so v2 error codes stay consistent with v1. (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(getErrorEncoder()))`)
**ParseAPICreateInputV2 for create/override input parsing** — Mapping API create inputs uses ParseAPICreateInputV2(inp, ns, cus.GetUsageAttribution()), handling metered/static/boolean variants and enforcing issueAfterReset vs grants mutual exclusion. Never inline discriminator logic. (`createInp, grantsInp, err := ParseAPICreateInputV2(request.APIInput, request.Namespace, cus.GetUsageAttribution())`)
**Separate connector fields for base and metered ops** — entitlementHandler holds connector (entitlement.Service) for generic CRUD and balanceConnector (meteredentitlement.Connector) for grant/history/reset; do not collapse them. (`h.connector.CreateEntitlement(...) vs h.balanceConnector.CreateGrant(...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | EntitlementHandler interface listing all V2 methods plus entitlementHandler struct and NewEntitlementHandler constructor — sole DI entry point. | New handler methods must be added to the EntitlementHandler interface; a missing method breaks the compile-time contract. |
| `customer.go` | Customer-scoped CRUD entitlement handlers (create, list, get, delete, override), all using resolve-customer-first. | Must always check cus.IsDeleted() before proceeding; omitting it lets soft-deleted customer operations succeed. |
| `customer_metered.go` | Metered-specific handlers (grants CRUD, balance history, usage reset) routed through h.balanceConnector. | Resolves entitlement via h.connector.GetEntitlementOfCustomerAt before using ent.ID for balance calls — do not pass featureKey directly to balanceConnector. |
| `entitlement.go` | Namespace-level list and get-by-ID handlers (not customer-scoped); ListEntitlements uses ListEntitlementsWithCustomer to return customer context. | OrderBy values must pass through strcase.CamelToSnake before comparison against ListEntitlementsOrderBy.StrValues(); skipping causes silent invalid orderby acceptance. |
| `mapping.go` | Domain-to-API conversions: parserV2 (ToAPIGenericV2/ToMeteredV2/ToStaticV2/ToBooleanV2), ParseAPICreateInputV2, MapEntitlementGrantToAPIV2, MapAPIGrantV2ToCreateGrantInput. | EntitlementV2 is a union — always use FromEntitlementMeteredV2/StaticV2/BooleanV2 to set the variant, not direct struct assignment. |
| `errors.go` | Single getErrorEncoder() composing v1 and generic error encoders. | Do not define new error types here; use models.Generic* errors or extend the v1 entitlementdriver encoder. |

## Anti-Patterns

- Calling entitlement domain methods with a subject key string instead of first resolving the customer to cus.ID — v2 is customer-ID-based
- Inlining entitlement type discrimination (switching on EntitlementType) in handlers instead of calling ParserV2.ToAPIGenericV2
- Accepting namespace as a URL/query parameter rather than resolving via h.resolveNamespace(ctx)
- Skipping the cus.IsDeleted() check before any entitlement mutation
- Adding business logic (grant burn-down, balance calculation) in handler or mapping files — delegate to connector/balanceConnector

## Decisions

- **V2 driver is customer-ID-centric rather than subject-key-centric like v1.** — The v2 API surface is customer-first; entitlements are always queried and returned with customerId/customerKey to align with the customer-scoped billing model.
- **Error encoding delegates to v1 entitlementdriver.GetErrorEncoder() first.** — Reusing v1 error codes prevents behavior divergence between API versions for the same domain errors.
- **Customer resolution (including deleted check) happens in the operation closure, not deferred to the domain service.** — Fail-fast on bad customer before hitting the entitlement service; the resolved cus also provides GetUsageAttribution() for building domain input.

## Example: Adding a new customer-scoped entitlement endpoint following the v2 pattern

```
type (
    GetCustomerEntitlementValueHandlerParams struct {
        CustomerIDOrKey           string
        EntitlementIdOrFeatureKey string
    }
    GetCustomerEntitlementValueHandlerRequest struct {
        CustomerIDOrKey           string
        EntitlementIdOrFeatureKey string
        Namespace                 string
    }
    GetCustomerEntitlementValueHandlerResponse = *api.EntitlementValue
    GetCustomerEntitlementValueHandler         = httptransport.HandlerWithArgs[GetCustomerEntitlementValueHandlerRequest, GetCustomerEntitlementValueHandlerResponse, GetCustomerEntitlementValueHandlerParams]
)
```

<!-- archie:ai-end -->
