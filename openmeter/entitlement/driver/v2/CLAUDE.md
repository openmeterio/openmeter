# v2

<!-- archie:ai-start -->

> HTTP driver layer for the V2 customer-centric entitlement API (package entitlementdriverv2). Exposes customer-scoped entitlement and grant endpoints plus namespace-wide entitlement list/get, mapping api.* request/response types onto entitlement.Service and meteredentitlement.Connector. Unlike v1 (subject-key based), every handler resolves a customer via customerService before touching the connector.

## Patterns

**httptransport.NewHandlerWithArgs three-stage handlers** — Every endpoint is a method on *entitlementHandler returning a typed Handler built from (request-decoder, business-fn, response-encoder, options). Decoder resolves namespace + customer and builds a typed Request struct; business-fn calls the connector and maps to api types. (`func (h *entitlementHandler) GetEntitlement() GetEntitlementHandler { return httptransport.NewHandlerWithArgs(decode, handle, commonhttp.JSONResponseEncoder[...], httptransport.AppendOptions(h.options, httptransport.WithOperationName("getEntitlementByIdV2"), httptransport.WithErrorEncoder(getErrorEncoder()))...) }`)
**Per-handler Request/Response/Params type triple** — Each endpoint declares a type block with HandlerRequest, HandlerResponse (usually an api.* alias), and HandlerParams (path/query args), then a named Handler alias of httptransport.HandlerWithArgs[Req,Resp,Params]. The EntitlementHandler interface in handler.go lists every endpoint constructor. (`type CreateCustomerEntitlementHandler httptransport.HandlerWithArgs[CreateCustomerEntitlementHandlerRequest, CreateCustomerEntitlementHandlerResponse, CreateCustomerEntitlementHandlerParams]`)
**Namespace then customer resolution in decoder** — Decoders call h.resolveNamespace(ctx) first, then h.customerService.GetCustomer with a customer.CustomerIDOrKey, and reject deleted customers with models.NewGenericPreConditionFailedError before proceeding. (`ns, err := h.resolveNamespace(ctx); cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerIDOrKey: &customer.CustomerIDOrKey{Namespace: ns, IDOrKey: params.CustomerIDOrKey}}); if cus.IsDeleted() { return models.NewGenericPreConditionFailedError(...) }`)
**Feature-key-or-id resolution via GetEntitlementOfCustomerAt** — Endpoints taking an EntitlementIdOrFeatureKey resolve it to a concrete entitlement for the customer at clock.Now() before acting (get/delete/override/history/reset), never trusting the raw path segment as an ID. (`ent, err := h.connector.GetEntitlementOfCustomerAt(ctx, ns, cus.ID, params.EntitlementIdOrFeatureKey, clock.Now())`)
**Stateless ParserV2 + Map* mapping functions** — All domain<->API translation lives in mapping.go: ParserV2 (empty struct, var ParserV2 = parserV2{}) does ToAPIGenericV2 type-switch dispatch to ToMeteredV2/ToStaticV2/ToBooleanV2; package funcs MapEntitlementGrantToAPIV2, ParseAPICreateInputV2, MapAPIGrantV2ToCreateGrantInput handle grants/inputs. (`v2, err := ParserV2.ToAPIGenericV2(ent, cus.ID, cus.Key); createInp, grantsInp, err := ParseAPICreateInputV2(request.APIInput, request.Namespace, cus.GetUsageAttribution())`)
**Reuse v1 driver helpers, never duplicate them** — v2 imports entitlementdriver (v1) for shared logic: getErrorEncoder() chains entitlementdriver.GetErrorEncoder() before the generic encoder; interval/recurrence mapping uses entitlementdriver.MapRecurrenceToAPI and entitlementdriver.MapAPIPeriodIntervalToRecurrence. (`func getErrorEncoder() encoder.ErrorEncoder { v1 := entitlementdriver.GetErrorEncoder(); generic := commonhttp.GenericErrorEncoder(); return func(...) bool { if v1(...) { return true }; return generic(...) } }`)
**WithOperationName matches the V2 OpenAPI operationId** — Each handler sets httptransport.WithOperationName to the camelCase V2 operation (e.g. createCustomerEntitlementV2, listEntitlementsV2). These must align with api/spec generated names. (`httptransport.WithOperationName("listCustomerEntitlementsV2")`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines the EntitlementHandler interface (11 endpoint constructors), the entitlementHandler struct, and NewEntitlementHandler. Dependencies: connector entitlement.Service, balanceConnector meteredentitlement.Connector, customerService customer.Service, namespaceDecoder namespacedriver.NamespaceDecoder. | Adding an endpoint requires adding it to the interface AND wiring it in the router; resolveNamespace lives here implicitly via namespaceDecoder. |
| `customer.go` | Customer-scoped generic entitlement CRUD: Create/List/Get/Delete/OverrideCustomerEntitlement, plus the resolveNamespace helper. Uses ParseAPICreateInputV2 and ParserV2.ToAPIGenericV2. | Create reuses ParseAPICreateInputV2 with cus.GetUsageAttribution(); Override resolves the old entitlement first then calls connector.OverrideEntitlement(ctx, cus.ID, oldEnt.ID, ...); deleted-customer guard is repeated in every handler. |
| `customer_metered.go` | Metered-specific customer endpoints: List/CreateCustomerEntitlementGrant, GetCustomerEntitlementHistory, ResetCustomerEntitlementUsage. Routes through h.balanceConnector (meteredentitlement.Connector), not the generic connector. | History builds api.WindowedBalanceHistory/burndown manually from windowedHistory and burndownHistory.Segments(); WindowTimeZone parsed via time.LoadLocation (400 on error); reset defaults At to clock.Now(). |
| `entitlement.go` | Namespace-wide (non-customer-scoped) ListEntitlements and GetEntitlement (by id). ListEntitlements validates OrderBy/EntitlementType against StrValues() and builds entitlement.ListEntitlementsParams. | OrderBy is converted with strcase.CamelToSnake then validated; there is duplicated OrderBy logic (an inline closure plus a later switch on params.OrderBy) — the switch wins. Get returns 404 via connector error path. |
| `mapping.go` | All domain<->API conversion. ParserV2 type-switches EntitlementType to ParseFromGenericEntitlement + ToMeteredV2/ToStaticV2/ToBooleanV2; ParseAPICreateInputV2 dispatches on inp.ValueByDiscriminator(); MapAPIGrantV2ToCreateGrantInput wraps credit.CreateGrantInput. | ParseAPICreateInputV2 enforces 'issueAfterReset and grants cannot be used together' and prunes ActiveFrom/ActiveTo to nil at the end; usage period built via timeutil.AsTimed/timeutil.Recurrence with Anchor defaulting to clock.Now(); follow /go-types-conversion naming (ToAPI../FromAPI..). |
| `errors.go` | getErrorEncoder() composes the v1 entitlement error encoder with commonhttp.GenericErrorEncoder so V2 error responses stay byte-compatible with V1. | Do not write a parallel error encoder; extend behavior in the v1 driver so both versions share it. |

## Anti-Patterns

- Trusting the EntitlementIdOrFeatureKey path segment as a literal entitlement ID instead of resolving via GetEntitlementOfCustomerAt — feature keys must be resolved per customer first.
- Skipping the customer-resolution + IsDeleted() precondition guard before calling the connector in customer-scoped handlers.
- Putting domain<->API translation logic in handler files; all mapping must stay in mapping.go via ParserV2 / Map* / Parse* functions.
- Calling the generic entitlement connector for grant/history/reset operations — those go through h.balanceConnector (meteredentitlement.Connector).
- Duplicating v1 logic (error encoding, recurrence/interval mapping) instead of importing entitlementdriver helpers.

## Decisions

- **V2 is customer-centric while V1 is subject-key-centric.** — Handlers resolve a customer.Customer (via customerService) and use cus.GetUsageAttribution()/cus.ID, replacing V1's subject-key parsing while reusing V1 parsers and error encoders for behavioral parity.
- **Generic entitlement operations and metered/grant operations are split across two connectors.** — entitlement.Service handles CRUD and access; meteredentitlement.Connector owns grants, balance history, and resets, keeping credit/grant concerns out of the generic service.
- **Mapping is centralized in a stateless ParserV2 plus free functions.** — Keeps handlers thin (decode + call + encode) and makes domain/API translation independently testable; mirrors the V1 parser pattern for consistency.

## Example: Customer-scoped handler: resolve namespace + customer, guard deleted, resolve entitlement, act, map to api type

```
func (h *entitlementHandler) GetCustomerEntitlement() GetCustomerEntitlementHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params GetCustomerEntitlementHandlerParams) (GetCustomerEntitlementHandlerRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return GetCustomerEntitlementHandlerRequest{}, err }
			return GetCustomerEntitlementHandlerRequest{CustomerIDOrKey: params.CustomerIDOrKey, EntitlementIdOrFeatureKey: params.EntitlementIdOrFeatureKey, Namespace: ns}, nil
		},
		func(ctx context.Context, request GetCustomerEntitlementHandlerRequest) (GetCustomerEntitlementHandlerResponse, error) {
			cus, err := h.customerService.GetCustomer(ctx, customer.GetCustomerInput{CustomerIDOrKey: &customer.CustomerIDOrKey{Namespace: request.Namespace, IDOrKey: request.CustomerIDOrKey}})
			if err != nil { return nil, err }
			if cus != nil && cus.IsDeleted() { return nil, models.NewGenericPreConditionFailedError(fmt.Errorf("customer is deleted [namespace=%s customer.id=%s]", cus.Namespace, cus.ID)) }
			ent, err := h.connector.GetEntitlementOfCustomerAt(ctx, request.Namespace, cus.ID, request.EntitlementIdOrFeatureKey, clock.Now())
			if err != nil { return nil, err }
			return ParserV2.ToAPIGenericV2(ent, ent.CustomerID, cus.Key)
		},
// ...
```

<!-- archie:ai-end -->
