# http

<!-- archie:ai-start -->

> HTTP driver (package httpdriver) for the plan-to-subscription bridge: decodes subscription create/change/migrate/edit/cancel/continue/restore/delete/get/list requests, maps API DTOs to plansubscription domain inputs and subscription.Patch values, and delegates to PlanSubscriptionService, SubscriptionService, and SubscriptionWorkflowService. Primary constraint: it owns all subscription API translation, so request shape ambiguity (custom plan vs plan ref) is disambiguated here.

## Patterns

**httptransport handler triple** — Every endpoint is a method on *handler returning a httptransport.Handler/HandlerWithArgs built from (decode func, business func, encoder, AppendOptions with WithOperationName + WithErrorEncoder(errorEncoder())). (`func (h *handler) CreateSubscription() CreateSubscriptionHandler { return httptransport.NewHandler(decode, exec, commonhttp.JSONResponseEncoderWithStatus[...](http.StatusCreated), httptransport.AppendOptions(h.Options, httptransport.WithOperationName("createSubscription"), httptransport.WithErrorEncoder(errorEncoder()))...) }`)
**Request/Response/Params type aliases per endpoint** — Each file declares a type ( ... ) block aliasing XRequest (often = plansubscription.X), XResponse (= api.Y), XParams, and XHandler = httptransport.HandlerWithArgs[...]. Reuse plansubscription request types via alias rather than redefining. (`type ( CreateSubscriptionRequest = plansubscription.CreateSubscriptionRequest; CreateSubscriptionResponse = api.Subscription )`)
**Namespace resolved via h.resolveNamespace** — Every decode func starts by calling h.resolveNamespace(ctx) and builds models.NamespacedID{Namespace: ns, ID: params.ID}; never read namespace any other way. (`ns, err := h.resolveNamespace(ctx); ... ID: models.NamespacedID{Namespace: ns, ID: params.ID}`)
**Custom-plan vs plan-ref disambiguation by re-marshal probe** — create.go and change.go marshal the body then unmarshal into a local testForCustomPlan{CustomPlan any} to detect which oneOf variant is present, then call body.AsCustomSubscriptionCreate()/AsPlanSubscriptionCreate(). Do not rely on generated As* succeeding to pick a branch. (`type testForCustomPlan struct { CustomPlan any `json:"customPlan"` }; if t.CustomPlan != nil { parsedBody, _ := body.AsCustomSubscriptionCreate() } else { body.AsPlanSubscriptionCreate() }`)
**API <-> domain mapping lives in mapping.go** — All conversions use Map* / From* helpers in mapping.go (MapSubscriptionToAPI, MapSubscriptionViewToAPI, MapAPISubscriptionEditOperationToPatch, MapAPITimingToTiming). Patch decoding switches on apiPatch.Discriminator() and builds patch.PatchAddItem/RemoveItem/AddPhase/etc. (`patches = append(patches, MapAPISubscriptionEditOperationToPatch(patch))`)
**Validation errors wrapped as GenericValidationError** — Decode-time semantic failures (missing customizations, bad timing, credits disabled) return models.NewGenericValidationError(...); errorEncoder() in errors.go then maps domain error types to HTTP status via commonhttp.HandleErrorIfTypeMatches. (`return EditSubscriptionRequest{}, models.NewGenericValidationError(fmt.Errorf("missing customizations"))`)
**Credits feature-gate guard before custom-plan create/change** — Custom plans with productcatalog.CreditOnlySettlementMode are rejected unless h.isCreditsEnabled(ns) (featuregate.go) is true, which checks Credits.Enabled, FeatureGate, and Credits.FeatureFlag. (`if !creditEnabled && req.SettlementMode == productcatalog.CreditOnlySettlementMode { return ..., models.NewGenericValidationError(...) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `handler.go` | Handler interface, HandlerConfig (services + Credits + FeatureGate), *handler struct, NewHandler, resolveNamespace. | Adding an endpoint requires adding its method to the Handler interface or wiring breaks. |
| `create.go` | CreateSubscription + getCustomer helper (by id or key, rejects deleted customers). | Custom-plan branch must run the credits guard; getCustomer returns NewGenericPreConditionFailedError for deleted customers. |
| `change.go` | ChangeSubscription; same custom/ref split as create, builds ChangeSubscriptionWorkflowInput. | Custom branch maps plan name/description/metadata onto the subscription; keep credits guard in sync with create.go. |
| `migrate.go` | MigrateSubscription; passes TargetVersion/StartingPhase/Timing/BillingAnchor to PlanSubscriptionService.Migrate. | WithOperationName is "MigrateSubscription" (capitalized) unlike others — preserve exact operation names. |
| `edit.go` | EditSubscription; maps body.Customizations to []subscription.Patch via MapAPISubscriptionEditOperationToPatch and calls SubscriptionWorkflowService.EditRunning. | Empty customizations is a validation error; default timing is TimingImmediate. |
| `cancel.go` | Cancel/Continue/Restore handlers; Restore goes through SubscriptionWorkflowService, Cancel/Continue through SubscriptionService. | Cancel defaults timing to TimingImmediate when body.Timing is nil. |
| `mapping.go` | All API<->domain mappers incl. patch discriminator switch and MapSubscriptionItemToAPI (feature/entitlement/price/tax). | Patch switch must cover every api.EditSubscription*Op; unknown discriminator returns an error, not a panic. |
| `errors.go` | errorEncoder() maps subscription/entitlement/feature error types to HTTP status; mapValidationIssueForAPI remaps spec field selectors. | New domain error types are invisible to clients until added to the HandleErrorIfTypeMatches chain. |

## Anti-Patterns

- Branching on body.As*() success instead of the testForCustomPlan probe to pick custom-plan vs plan-ref.
- Reading the namespace from anywhere other than h.resolveNamespace(ctx).
- Returning raw fmt.Errorf for client-facing validation failures instead of models.NewGenericValidationError (bypasses errorEncoder status mapping).
- Calling adapters/services directly in the encoder or skipping WithErrorEncoder(errorEncoder()), so domain errors fall through as 500s.
- Adding a handler method without registering it in the Handler interface.

## Decisions

- **Subscription create/change accept both a custom inline plan and a plan reference in one endpoint.** — The API uses a oneOf body; the marshal/unmarshal probe is the only reliable way to disambiguate because generated As* always succeed on serializable bodies.
- **Cancel/Continue/Restore/Edit are split between SubscriptionService and SubscriptionWorkflowService.** — Workflow-orchestrated operations (Restore, EditRunning, Change/Migrate via PlanSubscriptionService) go through the workflow layer; simple lifecycle ops stay on the base service.

## Example: Standard endpoint: resolve namespace, decode, call service, map to API, encode

```
func (h *handler) CancelSubscription() CancelSubscriptionHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, params CancelSubscriptionParams) (CancelSubscriptionRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return CancelSubscriptionRequest{}, err }
			var body api.CancelSubscriptionJSONRequestBody
			if err := commonhttp.JSONRequestBodyDecoder(r, &body); err != nil { return CancelSubscriptionRequest{}, err }
			timing, err := MapAPITimingToTiming(*body.Timing)
			if err != nil { return CancelSubscriptionRequest{}, models.NewGenericValidationError(err) }
			return CancelSubscriptionRequest{Timing: timing, ID: models.NamespacedID{Namespace: ns, ID: params.ID}}, nil
		},
		func(ctx context.Context, req CancelSubscriptionRequest) (CancelSubscriptionResponse, error) {
			sub, err := h.SubscriptionService.Cancel(ctx, req.ID, req.Timing)
			if err != nil { return CancelSubscriptionResponse{}, err }
			return MapSubscriptionToAPI(sub), nil
// ...
```

<!-- archie:ai-end -->
