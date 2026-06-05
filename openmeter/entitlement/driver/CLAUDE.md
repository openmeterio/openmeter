# driver

<!-- archie:ai-start -->

> HTTP driver layer (package entitlementdriver) for the V1 subject-key-based entitlement and metered-grant API, mapping api.* request/response types onto entitlement.Service and meteredentitlement.Connector. Every handler resolves the namespace then a customer from the subject key before touching the connector.

## Patterns

**Per-handler interface + httptransport.HandlerWithArgs triple** — EntitlementHandler / MeteredEntitlementHandler expose one method per operation returning a typed HandlerWithArgs[Request, Response, Params], built with httptransport.NewHandlerWithArgs(decode, handle, encode, options...). (`type CreateGrantHandler httptransport.HandlerWithArgs[CreateGrantHandlerRequest, CreateGrantHandlerResponse, CreateGrantHandlerParams]`)
**Namespace then subject->customer resolution** — Decoders call h.resolveNamespace(ctx) (from namespaceDecoder); business funcs call h.resolveCustomerFromSubject(ctx, ns, subjectIdOrKey) which goes subject.GetByIdOrKey then customer.GetCustomerByUsageAttribution. Nil customer -> NewGenericPreConditionFailedError. (`cust, err := h.resolveCustomerFromSubject(ctx, request.Namespace, request.SubjectKey)`)
**Generic vs metered connector split** — entitlementHandler uses entitlement.Service (connector) for CRUD/value/list; meteredEntitlementHandler uses meteredentitlement.Connector (balanceConnector) for grants, reset, balance history. (`grant, err := h.balanceConnector.CreateGrant(ctx, request.Namespace, cust.ID, request.EntitlementIdOrFeatureKey, request.GrantInput)`)
**Centralized mapping via Parser and free Map*/Parse* functions** — parser.go holds the stateless Parser (ToMetered/ToStatic/ToBoolean/ToAPIGeneric over EntitlementWithCustomer) plus MapEntitlementValueToAPI, ParseAPICreateInput, MapAPIPeriodIntervalToRecurrence, MapRecurrenceToAPI. Handlers never inline domain<->API translation. (`return Parser.ToAPIGeneric(&entitlement.EntitlementWithCustomer{Entitlement: lo.FromPtr(res), Customer: *cust})`)
**Shared error encoder** — All handlers attach httptransport.WithErrorEncoder(GetErrorEncoder()); errors.go maps domain errors (FeatureNotFoundError, entitlement.NotFoundError/AlreadyExistsError/InvalidValueError/InvalidFeatureError/WrongTypeError, pagination.InvalidError) to HTTP status via commonhttp.HandleErrorIfTypeMatches. (`commonhttp.HandleErrorIfTypeMatches[*entitlement.AlreadyExistsError](ctx, http.StatusConflict, err, w, func(e *entitlement.AlreadyExistsError) map[string]interface{}{ return map[string]interface{}{"conflictingEntityId": e.EntitlementID} })`)
**WithOperationName matches OpenAPI operationId** — Handlers append httptransport.WithOperationName("createEntitlement"/"overrideEntitlement"/"getEntitlementValue"/...) so telemetry/routing align with the generated spec. (`httptransport.WithOperationName("getEntitlementsOfSubject")`)
**Type discriminator dispatch for create/value** — ParseAPICreateInput switches on inp.ValueByDiscriminator() across Metered/Static/Boolean create inputs; MapEntitlementValueToAPI switches on the concrete value type (MeteredEntitlementValue/StaticEntitlementValue/BooleanEntitlementValue/NoAccessValue). (`switch v := value.(type) { case api.EntitlementMeteredCreateInputs: ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `entitlement.go` | EntitlementHandler interface + entitlementHandler: Create/Override/Get/GetById/Delete/GetValue/GetEntitlementsOfSubject/ListEntitlements, OrderBy/EntitlementType validation in decoders. | UsageAttribution is populated in the business func from the resolved customer, not the decoder ('somewhat hacky'); GetEntitlementsOfSubject also rejects deleted customers via NewGenericPreConditionFailedError. |
| `metered.go` | MeteredEntitlementHandler: CreateGrant, ListEntitlementGrants, ResetEntitlementUsage, GetEntitlementBalanceHistory; resolveNamespace/resolveCustomerFromSubject; MapEntitlementGrantToAPI. | Grant recurrence anchor defaults to EffectiveAt; balance history window timezone parsed via time.LoadLocation and burndown segments built from BalanceConnector output; ListEntitlementGrants uses a hardcoded page size of 1000. |
| `parser.go` | Stateless Parser + mapping/parse helpers; ParseAPICreateInput prunes ActiveFrom/ActiveTo and builds timeutil.Recurrence from UsagePeriod; MapRecurrenceToAPI is best-effort ISO mapping. | MapRecurrenceToAPI is explicitly approximate (24h != 1d) and falls back to ISO string; ParseAPICreateInput requires a usage period when MeasureUsageFrom is an enum preset. |
| `errors.go` | GetErrorEncoder mapping domain error types to HTTP statuses. | New domain error types are invisible to clients unless added here; ordering uses short-circuit || so the first matching type wins. |

## Anti-Patterns

- Treating an EntitlementIdOrFeatureKey path segment as a literal ID without resolving the customer first via resolveCustomerFromSubject.
- Calling the generic entitlement.Service for grant/reset/balance-history operations instead of meteredentitlement.Connector (balanceConnector).
- Putting domain<->API translation inside a handler closure instead of in parser.go (Parser/Map*/Parse*).
- Returning a raw error from a handler instead of routing through httptransport.WithErrorEncoder(GetErrorEncoder()) so domain errors get correct HTTP status.
- Passing a nil/unguarded customer to the connector instead of failing with models.NewGenericPreConditionFailedError when the subject has no customer.

## Decisions

- **V1 driver is subject-key-centric: every request resolves a customer from a subject key before hitting the connector.** — Preserves the legacy subject-based public API surface while the domain layer is customer-centric; v2 (entitlementdriverv2) is the customer-centric counterpart.
- **Generic and metered/grant operations are split across entitlement.Service and meteredentitlement.Connector.** — Keeps credit/grant/reset balance logic in the metered connector and CRUD/value in the generic service, matching the domain boundary.
- **All domain<->API mapping is concentrated in a stateless Parser plus free functions in parser.go.** — Handlers stay thin (decode/resolve/call/encode) and v2 reuses these helpers rather than duplicating them.

## Example: A typical V1 handler: decode + namespace, then resolve customer and map result

```
func (h *entitlementHandler) GetEntitlementValue() GetEntitlementValueHandler {
  return httptransport.NewHandlerWithArgs(
    func(ctx context.Context, r *http.Request, params GetEntitlementValueHandlerParams) (GetEntitlementValueHandlerRequest, error) {
      ns, err := h.resolveNamespace(ctx)
      if err != nil { return GetEntitlementValueHandlerRequest{}, err }
      return GetEntitlementValueHandlerRequest{SubjectKey: params.SubjectKey, EntitlementIdOrFeatureKey: params.EntitlementIdOrFeatureKey, Namespace: ns, At: defaultx.WithDefault(params.Params.Time, clock.Now())}, nil
    },
    func(ctx context.Context, request GetEntitlementValueHandlerRequest) (api.EntitlementValue, error) {
      cust, err := h.resolveCustomerFromSubject(ctx, request.Namespace, request.SubjectKey)
      if err != nil { return api.EntitlementValue{}, err }
      v, err := h.connector.GetEntitlementValue(ctx, request.Namespace, cust.ID, request.EntitlementIdOrFeatureKey, request.At)
      if err != nil { return api.EntitlementValue{}, err }
      return MapEntitlementValueToAPI(v)
    },
    commonhttp.JSONResponseEncoder[api.EntitlementValue],
// ...
```

<!-- archie:ai-end -->
