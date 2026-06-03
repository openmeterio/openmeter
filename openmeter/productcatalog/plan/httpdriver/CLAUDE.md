# httpdriver

<!-- archie:ai-start -->

> v1 HTTP handler layer for plan lifecycle endpoints (List, Create, Update, Delete, Get, Publish, Archive, Next), translating between api.* types and plan.Service via httptransport.Handler. Primary constraint: never touch Ent or plan.Repository directly — all persistence goes through plan.Service.

## Patterns

**httptransport.NewHandler(WithArgs) with typed aliases** — Each endpoint declares XxxRequest/XxxResponse/XxxHandler type aliases, then returns a handler built from decoder, operation, encoder, and AppendOptions. (`type ListPlansHandler httptransport.HandlerWithArgs[ListPlansRequest, ListPlansResponse, ListPlansParams]`)
**Namespace resolved from context in every decoder** — h.resolveNamespace(ctx) is called in every request decoder; failure returns a 500 because the namespace is always set by upstream middleware. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Req{}, fmt.Errorf("failed to resolve namespace: %w", err) }`)
**credits.Enabled guard in the decoder** — CreatePlan/UpdatePlan decoders reject CreditOnlySettlementMode when h.credits.Enabled is false; this guard belongs in the decoder, not the operation/service. (`if !h.credits.Enabled && req.SettlementMode == productcatalog.CreditOnlySettlementMode { return CreatePlanRequest{}, models.NewGenericValidationError(...) }`)
**ValidationErrorEncoder on every handler** — Every endpoint appends productcataloghttp.ValidationErrorEncoder(ResourceKindPlan) so plan validation issues map to proper HTTP statuses. (`httptransport.AppendOptions(h.options, httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)))`)
**ref.ParseIDOrKey for dual ID/key routing** — GetPlan and NextPlan use ref.ParseIDOrKey to accept either ULID IDs or string keys on one endpoint. (`idOrKey := ref.ParseIDOrKey(params.IDOrKey); GetPlanRequest{NamespacedID: models.NamespacedID{Namespace: ns, ID: idOrKey.ID}, Key: idOrKey.Key}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Declares Handler/PlanHandler interfaces, the handler struct, and New constructor; injects credits appconfig.CreditsConfiguration. | credits must be passed to New; omitting it silently allows CreditOnly plans when credits are disabled. |
| `plan.go` | All endpoint implementations. Create/Update decoders set IgnoreNonCriticalIssues=true; PublishPlan sets EffectiveFrom=clock.Now(); ArchivePlan sets EffectiveTo=clock.Now(). | Do not remove IgnoreNonCriticalIssues=true — plans may carry non-critical validation issues during create/update. |
| `mapping.go` | FromPlan, FromPlanPhase, AsCreatePlanRequest, AsUpdatePlanRequest, AsPlanPhase — api↔domain conversion, delegating rate cards to productcatalog/http.FromRateCard / AsRateCards. | New plan fields must be mapped in both FromPlan and AsCreate/AsUpdate; missing fields are silently ignored at the API boundary. |
| `featuregate.go` | Feature-gating helpers used by the handlers to conditionally restrict behavior. | Keep gating logic in the decode phase consistent with the credits guard. |

## Anti-Patterns

- Importing entdb or calling plan.Repository directly — all access goes through plan.Service.
- Hardcoding namespace strings instead of h.resolveNamespace(ctx).
- Omitting productcataloghttp.ValidationErrorEncoder — validation errors won't map to HTTP status codes.
- Putting the credits.Enabled guard in the operation closure instead of the decoder.
- Setting EffectivePeriod directly in UpdatePlan decoder — status transitions go through PublishPlan/ArchivePlan.

## Decisions

- **Credits feature guard is duplicated in both CreatePlan and UpdatePlan decoders.** — credits.enabled must be enforced at the HTTP boundary; the service layer does not know about deployment configuration.
- **PublishPlan and ArchivePlan are separate explicit operations, not flags on UpdatePlan.** — Prevents callers manipulating plan status via UpdatePlan; EffectivePeriod is zeroed in UpdatePlan so transitions are gated through explicit publish/archive paths.

## Example: Adding a plan endpoint with the three-alias + handler pattern

```
type (
	MyPlanRequest  = plan.MyPlanInput
	MyPlanResponse = api.Plan
	MyPlanHandler  httptransport.HandlerWithArgs[MyPlanRequest, MyPlanResponse, string]
)

func (h *handler) MyPlan() MyPlanHandler {
	return httptransport.NewHandlerWithArgs(
		func(ctx context.Context, r *http.Request, planID string) (MyPlanRequest, error) {
			ns, err := h.resolveNamespace(ctx)
			if err != nil { return MyPlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err) }
			return MyPlanRequest{NamespacedID: models.NamespacedID{Namespace: ns, ID: planID}}, nil
		},
		func(ctx context.Context, req MyPlanRequest) (MyPlanResponse, error) { return h.service.MyPlan(ctx, req) },
		commonhttp.JSONResponseEncoderWithStatus[MyPlanResponse](http.StatusOK),
// ...
```

<!-- archie:ai-end -->
