# http

<!-- archie:ai-start -->

> v1 HTTP driver for plan-aware subscription lifecycle endpoints (create, get, list, edit, cancel, continue, restore, change, migrate, delete). Translates between api.* types and domain types, delegating to PlanSubscriptionService, SubscriptionWorkflowService, and SubscriptionService via httptransport.HandlerWithArgs.

## Patterns

**HandlerWithArgs type alias pattern** — Each operation defines four type aliases (Request, Response, Params, Handler) then implements a method on *handler returning httptransport.NewHandlerWithArgs(decoder, operation, encoder, options...). (`type CancelSubscriptionHandler = httptransport.HandlerWithArgs[CancelSubscriptionRequest, CancelSubscriptionResponse, CancelSubscriptionParams]`)
**Namespace resolved first in every decoder** — Every decoder lambda calls h.resolveNamespace(ctx) as its first step; failure returns a 500 via commonhttp.NewHTTPError. Namespace is never obtained from the request body. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**Custom-plan discriminator sniff via marshal+unmarshal** — oapi-codegen union helpers (AsCustom*, AsPlan*) succeed on any structurally-serializable body, so create.go and change.go marshal the body to JSON then unmarshal into a struct checking for the customPlan field before calling As* helpers. (`type testForCustomPlan struct { CustomPlan any `json:"customPlan"` }; json.Unmarshal(bodyBytes, &t); if t.CustomPlan != nil { body.AsCustomSubscriptionCreate() }`)
**Credits guard in decoder, not in operation func** — If !h.Credits.Enabled && req.SettlementMode == productcatalog.CreditOnlySettlementMode, the decoder returns models.NewGenericValidationError. The domain service is deployment-agnostic so the flag belongs at the HTTP boundary. (`if !h.Credits.Enabled && req.SettlementMode == productcatalog.CreditOnlySettlementMode { return ..., models.NewGenericValidationError(...) }`)
**Centralized errorEncoder shared by all handlers** — errors.go defines a single errorEncoder() that maps subscription.PatchConflictError, PatchForbiddenError, PatchValidationError, entitlement errors, feature.FeatureNotFoundError, and pagination.InvalidError to HTTP status codes. Every handler appends it via httptransport.WithErrorEncoder(errorEncoder()). (`httptransport.AppendOptions(h.Options, httptransport.WithOperationName("cancelSubscription"), httptransport.WithErrorEncoder(errorEncoder()))...`)
**All domain-to-API conversions live in mapping.go** — MapSubscriptionToAPI, MapSubscriptionViewToAPI, MapSubscriptionPhaseToAPI, MapSubscriptionItemToAPI, MapAPITimingToTiming, MapAPISubscriptionEditOperationToPatch, and CustomPlanToCreatePlanRequest are the only allowed conversion points. (`return MapSubscriptionToAPI(sub), nil`)
**HandlerConfig struct injection with all dependencies** — Handler is implemented by *handler embedding HandlerConfig (SubscriptionWorkflowService, SubscriptionService, CustomerService, PlanSubscriptionService, NamespaceDecoder, Logger, Credits). New methods must not add dependencies outside HandlerConfig. (`type HandlerConfig struct { SubscriptionWorkflowService subscriptionworkflow.Service; SubscriptionService subscription.Service; PlanSubscriptionService plansubscription.PlanSubscriptionService; ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Defines Handler interface listing all operation methods, HandlerConfig with all dependencies, and NewHandler constructor. | Every new operation requires a new method on the Handler interface and all needed services must already be in HandlerConfig — do not inline services into individual operation files. |
| `mapping.go` | All bidirectional conversions between domain and api.* types including phase item selection logic (current=active item, future=first, past=last). | MapSubscriptionPhaseToAPI's relativePhaseTime branching determines what items appear in the API response; changing this affects all GET subscription responses. |
| `errors.go` | Centralized error-to-HTTP-status mapping. ValidationIssues are handled first via commonhttp.HandleIssueIfHTTPStatusKnown, then specific domain error types via HandleErrorIfTypeMatches. | Any new domain error type that needs a non-500 HTTP status must be registered here; omitting it silently produces 500. |
| `create.go` | CreateSubscription handler with custom-plan discriminator sniff; resolves customer by ID or key via getCustomer helper which returns GenericPreConditionFailedError (412) for deleted customers. | getCustomer must be called and its error propagated before calling PlanSubscriptionService.Create. |
| `change.go` | ChangeSubscription handler; same custom-plan discriminator sniff as create.go; credits guard applies only in the custom plan path. | Timing is mandatory in the change body; credits guard applies only when AsCustomSubscriptionChange() path is taken. |

## Anti-Patterns

- Calling SubscriptionService directly for operations that have a workflow (create, change, edit, restore, migrate) — always route through PlanSubscriptionService or SubscriptionWorkflowService.
- Adding business logic to the operation func (second lambda in NewHandlerWithArgs); all validation and transformation belongs in the decoder (first lambda).
- Using oapi-codegen As* helpers (body.AsCustom*, body.AsPlan*) without the marshal+unmarshal discriminator sniff — they succeed on wrong types and silently produce zero values.
- Registering a new handler operation without adding its method to the Handler interface in handler.go.
- Returning a domain error type not covered by errorEncoder — it silently becomes a 500.

## Decisions

- **Custom-plan discriminator sniff via marshal+unmarshal instead of generated As* helpers** — TypeSpec-generated union helpers succeed if the JSON is structurally serializable even for the wrong union member; field presence check is the only reliable discriminator.
- **Credits.Enabled guard placed in the HTTP decoder, not in the domain service** — The HTTP driver owns deployment configuration; domain services are deployment-agnostic so the feature flag belongs at the transport boundary.

## Example: Add a new subscription action endpoint (e.g. PauseSubscription)

```
// handler.go: add to Handler interface
PauseSubscription() PauseSubscriptionHandler

// pause.go:
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
