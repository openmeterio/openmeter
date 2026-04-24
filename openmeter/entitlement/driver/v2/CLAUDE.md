# v2

<!-- archie:ai-start -->

> V2 HTTP driver for the entitlement domain — adapts entitlement.Service and meteredentitlement.Connector to HTTP handlers using the httptransport.HandlerWithArgs pattern, always scoped to a resolved customer (ID or key). All endpoints include customerId/customerKey fields in responses, distinguishing this layer from the v1 subject-keyed driver.

## Patterns

**HandlerWithArgs typed triplet** — Every endpoint declares three type aliases — Request, Response, Params — then aliases the handler type as `httptransport.HandlerWithArgs[Req, Resp, Params]`. The method on entitlementHandler returns that alias type. (`type CreateCustomerEntitlementHandler httptransport.HandlerWithArgs[CreateCustomerEntitlementHandlerRequest, CreateCustomerEntitlementHandlerResponse, CreateCustomerEntitlementHandlerParams]`)
**Customer resolution before entitlement ops** — Every mutating or read handler resolves the customer via h.customerService.GetCustomer with CustomerIDOrKey first, then checks cus.IsDeleted() returning models.NewGenericPreConditionFailedError before any entitlement call. (`cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerIDOrKey: &customer.CustomerIDOrKey{Namespace: ns, IDOrKey: req.CustomerIDOrKey}}); if cus.IsDeleted() { return ..., models.NewGenericPreConditionFailedError(...) }`)
**Namespace resolved from context via namespaceDecoder** — All handlers call h.resolveNamespace(ctx) as the first step; it delegates to h.namespaceDecoder.GetNamespace(ctx). If not found, returns a 500 HTTP error. Never accept namespace from URL params. (`ns, err := h.resolveNamespace(ctx); if err != nil { return def, err }`)
**ParserV2.ToAPIGenericV2 for all entitlement responses** — All handlers returning an entitlement call ParserV2.ToAPIGenericV2(ent, cust.ID, cust.Key) — never inline the mapping. This dispatches by EntitlementType to ToMeteredV2/ToStaticV2/ToBooleanV2. (`return ParserV2.ToAPIGenericV2(ent, cust.ID, cust.Key)`)
**Error encoder chains v1 then generic** — getErrorEncoder() composes entitlementdriver.GetErrorEncoder() (v1 errors) with commonhttp.GenericErrorEncoder() so v2 error codes stay consistent with v1. (`httptransport.WithErrorEncoder(getErrorEncoder())`)
**ParseAPICreateInputV2 for create/override input parsing** — Mapping API create inputs uses ParseAPICreateInputV2(inp, ns, cus.GetUsageAttribution()) — never inline the discriminator logic. It handles metered/static/boolean variants and enforces mutual exclusion of issueAfterReset vs grants. (`createInp, grantsInp, err := ParseAPICreateInputV2(request.APIInput, request.Namespace, cus.GetUsageAttribution())`)
**Separate connector fields for base and metered ops** — entitlementHandler has two separate service fields: connector (entitlement.Service) for generic entitlement CRUD, and balanceConnector (meteredentitlement.Connector) for grant/history/reset ops. Do not collapse them. (`h.connector.CreateEntitlement(...) vs h.balanceConnector.CreateGrant(...)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines EntitlementHandler interface listing all V2 methods plus entitlementHandler struct and NewEntitlementHandler constructor. This is the sole DI entry point — all dependencies are injected here. | Adding new handler methods requires adding to EntitlementHandler interface; failing to do so breaks the compile-time contract. |
| `customer.go` | Handlers for customer-scoped CRUD entitlement endpoints (create, list, get, delete, override). All follow the resolve-customer-first pattern. | Must always check cus.IsDeleted() before proceeding; omitting this lets soft-deleted customer operations succeed. |
| `customer_metered.go` | Handlers for metered-specific endpoints (grants CRUD, balance history, usage reset) routed through h.balanceConnector. | Resolves entitlement via h.connector.GetEntitlementOfCustomerAt before using the returned ent.ID for balance calls — do not pass featureKey directly to balanceConnector. |
| `entitlement.go` | Namespace-level entitlement list and get-by-ID handlers (not customer-scoped). ListEntitlements uses ListEntitlementsWithCustomer to return customer context in results. | OrderBy values must go through strcase.CamelToSnake before comparison against ListEntitlementsOrderBy.StrValues(); skipping this causes silent invalid orderby acceptance. |
| `mapping.go` | All domain-to-API type conversions: parserV2 struct with ToAPIGenericV2/ToMeteredV2/ToStaticV2/ToBooleanV2; ParseAPICreateInputV2 for input parsing; MapEntitlementGrantToAPIV2 and MapAPIGrantV2ToCreateGrantInput. | EntitlementV2 is a union type — always use FromEntitlementMeteredV2/FromEntitlementStaticV2/FromEntitlementBooleanV2 to set the variant, not direct struct assignment. |
| `errors.go` | Single getErrorEncoder() function that composes v1 and generic error encoders. No other error handling logic lives here. | Do not define new error types here; use models.Generic* errors or extend the v1 entitlementdriver error encoder. |

## Anti-Patterns

- Calling entitlement domain methods with a subject key string instead of first resolving the customer to cus.ID — v2 is customer-ID-based, not subject-key-based.
- Inlining entitlement type discrimination (switching on EntitlementType) in handlers instead of calling ParserV2.ToAPIGenericV2.
- Accepting namespace as a URL/query parameter rather than resolving via h.resolveNamespace(ctx).
- Skipping the cus.IsDeleted() check before performing any entitlement mutation.
- Adding business logic (grant burn-down, balance calculation) in handler or mapping files — delegate entirely to connector/balanceConnector.

## Decisions

- **V2 driver is customer-ID-centric rather than subject-key-centric like v1.** — The v2 API surface is customer-first; entitlements are always queried and returned with customerId/customerKey to align with the customer-scoped billing model.
- **Error encoding delegates to v1 entitlementdriver.GetErrorEncoder() first.** — Reusing v1 error codes prevents behavior divergence between API versions for the same domain errors.
- **Customer resolution (including deleted check) happens in the request decoder closure, not the operation closure.** — Fail-fast on bad customer before hitting the entitlement service; also means the resolved cus is available in the decoder for building the domain input (e.g. GetUsageAttribution).

## Example: Adding a new customer-scoped entitlement endpoint (e.g. GetCustomerEntitlementValue)

```
// 1. Declare types in a new file or customer.go
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

// ...
```

<!-- archie:ai-end -->
