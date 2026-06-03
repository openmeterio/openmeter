# http

<!-- archie:ai-start -->

> v1 HTTP driver for plan-aware subscription lifecycle endpoints (create, get, list, edit, cancel, continue, restore, change, migrate, delete). Translates between api.* and domain types, delegating to PlanSubscriptionService, SubscriptionWorkflowService, and SubscriptionService via httptransport.HandlerWithArgs.

## Patterns

**HandlerWithArgs type alias pattern** — Each operation defines Request/Response/Params/Handler aliases then a method on *handler returning httptransport.NewHandlerWithArgs(decoder, operation, encoder, options...). (`type CancelSubscriptionHandler = httptransport.HandlerWithArgs[CancelSubscriptionRequest, CancelSubscriptionResponse, CancelSubscriptionParams]`)
**Namespace resolved first in every decoder** — Every decoder lambda calls h.resolveNamespace(ctx) first; failure returns 500. Namespace is never read from the request body. (`ns, err := h.resolveNamespace(ctx); if err != nil { return ..., err }`)
**Custom-plan discriminator sniff via marshal+unmarshal** — oapi-codegen As* helpers succeed on any structurally-serializable body, so create.go/change.go marshal then unmarshal checking for the customPlan field before calling As*. (`type testForCustomPlan struct { CustomPlan any `json:"customPlan"` }; json.Unmarshal(bodyBytes, &t); if t.CustomPlan != nil { body.AsCustomSubscriptionCreate() }`)
**Credits guard in decoder, not in operation** — If !h.Credits.Enabled && req.SettlementMode == CreditOnlySettlementMode the decoder returns models.NewGenericValidationError; domain services are deployment-agnostic. (`if !h.Credits.Enabled && req.SettlementMode == productcatalog.CreditOnlySettlementMode { return ..., models.NewGenericValidationError(...) }`)
**Centralized errorEncoder shared by all handlers** — errors.go's single errorEncoder() maps subscription patch errors, entitlement errors, feature.FeatureNotFoundError, pagination.InvalidError; appended via WithErrorEncoder on every handler. (`httptransport.AppendOptions(h.Options, httptransport.WithOperationName("cancelSubscription"), httptransport.WithErrorEncoder(errorEncoder()))...`)
**All domain-to-API conversions in mapping.go** — MapSubscriptionToAPI, MapSubscriptionViewToAPI, MapSubscriptionPhaseToAPI, MapAPITimingToTiming, MapAPISubscriptionEditOperationToPatch, CustomPlanToCreatePlanRequest are the only conversion points. (`return MapSubscriptionToAPI(sub), nil`)
**HandlerConfig struct injection** — *handler embeds HandlerConfig holding all services (workflow, subscription, customer, plansubscription, namespace decoder, logger, credits). New methods must not add deps outside HandlerConfig. (`type HandlerConfig struct { SubscriptionWorkflowService subscriptionworkflow.Service; PlanSubscriptionService plansubscription.PlanSubscriptionService; ... }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface listing all operation methods, HandlerConfig with all deps, NewHandler constructor. | Every new operation needs a method on Handler and its services must already be in HandlerConfig — do not inline services into operation files. |
| `mapping.go` | All bidirectional domain↔api.* conversions including phase item selection (current=active, future=first, past=last). | MapSubscriptionPhaseToAPI relativePhaseTime branching determines which items appear in responses — changes affect all GET subscription responses. |
| `errors.go` | Centralized error→HTTP-status mapping; ValidationIssues handled first via HandleIssueIfHTTPStatusKnown, then types via HandleErrorIfTypeMatches. | New domain error types needing non-500 status must be registered here; omitting silently yields 500. |
| `create.go` | CreateSubscription with custom-plan sniff; resolves customer via getCustomer (returns 412 GenericPreConditionFailedError for deleted customers). | getCustomer must be called and its error propagated before PlanSubscriptionService.Create. |
| `change.go` | ChangeSubscription with the same custom-plan sniff; credits guard only in the custom plan path. | Timing is mandatory in the change body; credits guard applies only on the AsCustomSubscriptionChange() path. |

## Anti-Patterns

- Calling SubscriptionService directly for workflow operations (create/change/edit/restore/migrate) — route through PlanSubscriptionService or SubscriptionWorkflowService.
- Putting validation/transformation in the operation lambda — it belongs in the decoder (first lambda).
- Using As* helpers without the marshal+unmarshal discriminator sniff — they succeed on wrong types and produce zero values.
- Registering a handler operation without adding its method to the Handler interface.
- Returning a domain error not covered by errorEncoder — it silently becomes a 500.

## Decisions

- **Custom-plan discriminator sniff via marshal+unmarshal instead of generated As* helpers.** — Union helpers succeed if the JSON is structurally serializable even for the wrong member; field presence is the only reliable discriminator.
- **Credits.Enabled guard in the HTTP decoder, not the domain service.** — The HTTP driver owns deployment config; domain services are deployment-agnostic so the flag belongs at the transport boundary.

## Example: Add a new subscription action endpoint (PauseSubscription)

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
            if err != nil { return PauseSubscriptionRequest{}, err }
// ...
```

<!-- archie:ai-end -->
