# http

<!-- archie:ai-start -->

> HTTP driver for subscription lifecycle endpoints (create, get, list, edit, cancel, continue, restore, change, migrate, delete). Adapts plan-aware subscription operations to the v1 API using httptransport.HandlerWithArgs, delegating to PlanSubscriptionService, SubscriptionWorkflowService, and SubscriptionService.

## Patterns

**HandlerWithArgs for path-param endpoints** — Each operation returning a typed handler alias: type Foo[Handler|Params|Request|Response] = ... then method on *handler returns httptransport.NewHandlerWithArgs(decoder, operation, encoder, options...) (`type CancelSubscriptionHandler = httptransport.HandlerWithArgs[CancelSubscriptionRequest, CancelSubscriptionResponse, CancelSubscriptionParams]`)
**Namespace always resolved from context** — Every decoder calls h.resolveNamespace(ctx) first; failure returns a 500 internal server error via commonhttp.NewHTTPError (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**Custom-plan discriminator sniff** — Because oapi-codegen union helpers succeed even on wrong type, create/change decoders marshal+unmarshal body to test for presence of customPlan field before calling AsCustom*/AsPlan* helpers (`type testForCustomPlan struct { CustomPlan any `json:"customPlan"` }; json.Unmarshal(bodyBytes, &t); if t.CustomPlan != nil { body.AsCustomSubscriptionCreate() }`)
**Credits guard in decoder** — If !h.Credits.Enabled && req.SettlementMode == productcatalog.CreditOnlySettlementMode return models.NewGenericValidationError; guard lives in the request decoder, not the operation func (`if !h.Credits.Enabled && req.SettlementMode == productcatalog.CreditOnlySettlementMode { return ..., models.NewGenericValidationError(...) }`)
**Domain-specific errorEncoder** — errors.go registers a single errorEncoder() used by every handler; maps subscription.PatchConflictError, PatchForbiddenError, PatchValidationError, entitlement.NotFoundError, feature.FeatureNotFoundError, pagination.InvalidError to HTTP status codes (`httptransport.WithErrorEncoder(errorEncoder())`)
**Mapping layer in mapping.go** — All domain-to-API conversions live in mapping.go: MapSubscriptionToAPI, MapSubscriptionViewToAPI, MapSubscriptionPhaseToAPI, MapSubscriptionItemToAPI, MapAPITimingToTiming, MapAPISubscriptionEditOperationToPatch, CustomPlanToCreatePlanRequest (`return MapSubscriptionToAPI(sub), nil`)
**HandlerConfig struct injection** — Handler interface is implemented by *handler which embeds HandlerConfig containing all service dependencies; constructed via NewHandler(config, options...) (`type HandlerConfig struct { SubscriptionWorkflowService subscriptionworkflow.Service; SubscriptionService subscription.Service; ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface listing all operations, HandlerConfig with all dependencies, and NewHandler constructor | Adding an operation: add the method signature to Handler interface and HandlerConfig must include all services the method needs |
| `mapping.go` | All bidirectional conversions between domain types and api.* types; imports productcataloghttp, plandriver, entitlementdriver for sub-type mapping | MapSubscriptionPhaseToAPI applies phase-relative item selection logic (current=active item, future=first, past=last); changing this affects what items appear in the API response |
| `errors.go` | Centralized error-to-HTTP-status mapping; must be updated whenever a new domain error type is introduced that needs non-500 status | ValidationIssues with httpStatusCodeErrorAttribute are handled first via commonhttp.HandleIssueIfHTTPStatusKnown; add new error types via commonhttp.HandleErrorIfTypeMatches |
| `create.go` | CreateSubscription handler; resolves customer by ID or key (getCustomer helper), builds PlanInput, calls PlanSubscriptionService.Create | getCustomer returns GenericPreConditionFailedError (412) if customer is deleted; must propagate before calling service |
| `change.go` | ChangeSubscription handler; same custom-plan discriminator sniff as create.go; delegates to PlanSubscriptionService.Change | timing is mandatory in change body; credits guard applies to custom plan path only |

## Anti-Patterns

- Calling SubscriptionService directly for operations that have a workflow (create, change, edit, restore, migrate) — always route through PlanSubscriptionService or SubscriptionWorkflowService
- Adding business logic to the operation func (second lambda); keep all validation/transformation in the decoder (first lambda)
- Using type-assert AS helpers (body.AsCustom*) without the discriminator sniff — they succeed on wrong type and silently produce zero values
- Registering a new handler variant without adding it to the Handler interface in handler.go
- Returning a non-domain error type that is not covered by errorEncoder — it will silently become a 500

## Decisions

- **Custom-plan discriminator sniff via marshal+unmarshal instead of generated As* helpers** — TypeSpec-generated union helpers succeed if the JSON is structurally serializable even when fields belong to the other union member; field presence check is the only reliable discriminator
- **Credits.Enabled guard placed in the HTTP decoder, not in the service layer** — The HTTP driver owns the OpenMeter deployment configuration; the domain service is deployment-agnostic, so the feature flag belongs at the boundary

## Example: Add a new subscription action endpoint

```
// In handler.go add to Handler interface:
PauseSubscription() PauseSubscriptionHandler

// New file pause.go:
type (
    PauseSubscriptionRequest  = struct{ ID models.NamespacedID }
    PauseSubscriptionResponse = api.Subscription
    PauseSubscriptionParams   = struct{ ID string }
    PauseSubscriptionHandler  = httptransport.HandlerWithArgs[PauseSubscriptionRequest, PauseSubscriptionResponse, PauseSubscriptionParams]
)

func (h *handler) PauseSubscription() PauseSubscriptionHandler {
    return httptransport.NewHandlerWithArgs(
        func(ctx context.Context, r *http.Request, params PauseSubscriptionParams) (PauseSubscriptionRequest, error) {
            ns, err := h.resolveNamespace(ctx)
// ...
```

<!-- archie:ai-end -->
