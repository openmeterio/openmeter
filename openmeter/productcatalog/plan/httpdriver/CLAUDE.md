# httpdriver

<!-- archie:ai-start -->

> v1 HTTP handler layer for plan lifecycle endpoints (List, Create, Update, Delete, Get, Publish, Archive, Next), translating between api.* types and plan.Service calls via httptransport.Handler. Primary constraint: never call Ent or plan.Repository directly — all persistence goes through plan.Service.

## Patterns

**httptransport.NewHandler / NewHandlerWithArgs with typed aliases** — Each endpoint declares three type aliases (XxxRequest, XxxResponse, XxxHandler) then returns a handler built with decoder, operation, encoder, and AppendOptions. This makes the interface method signature self-documenting. (`type (ListPlansRequest = plan.ListPlansInput; ListPlansResponse = api.PlanPaginatedResponse; ListPlansHandler httptransport.HandlerWithArgs[ListPlansRequest, ListPlansResponse, ListPlansParams])`)
**Namespace decoded from context in every decoder** — h.resolveNamespace(ctx) must be called in every request decoder closure. Failure returns commonhttp.NewHTTPError(500) because the namespace is always set by middleware upstream. (`ns, err := h.resolveNamespace(ctx); if err != nil { return Req{}, fmt.Errorf("failed to resolve namespace: %w", err) }`)
**credits.Enabled guard in decoder for settlement mode** — CreatePlan and UpdatePlan decoders reject CreditOnlySettlementMode when h.credits.Enabled is false, returning models.NewGenericValidationError. Guard belongs in the decoder, not in the operation or service. (`if !h.credits.Enabled && req.SettlementMode == productcatalog.CreditOnlySettlementMode { return CreatePlanRequest{}, models.NewGenericValidationError(...) }`)
**ValidationErrorEncoder on every handler** — Every endpoint appends httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)) so plan validation issues map to proper HTTP status codes. (`httptransport.AppendOptions(h.options, httptransport.WithOperationName("listPlans"), httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(productcataloghttp.ResourceKindPlan)))...`)
**ref.ParseIDOrKey for dual ID/key routing** — GetPlan and NextPlan use ref.ParseIDOrKey to distinguish ULID IDs from string keys without separate endpoints. (`idOrKey := ref.ParseIDOrKey(params.IDOrKey); return GetPlanRequest{NamespacedID: models.NamespacedID{Namespace: ns, ID: idOrKey.ID}, Key: idOrKey.Key}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `driver.go` | Declares Handler and PlanHandler interfaces, the handler struct, and New constructor. credits appconfig.CreditsConfiguration is injected here. | credits must be passed to New; omitting it silently allows CreditOnly plans when credits are disabled. |
| `plan.go` | All endpoint implementations. IgnoreNonCriticalIssues=true is set in Create and Update decoders; PublishPlan sets EffectiveFrom to clock.Now(); ArchivePlan sets EffectiveTo to clock.Now(). | Do not remove IgnoreNonCriticalIssues=true — plans may have non-critical validation issues during create/update. |
| `mapping.go` | FromPlan, FromPlanPhase, AsCreatePlanRequest, AsUpdatePlanRequest, AsPlanPhase — api↔domain conversion. Delegates rate card mapping to productcatalog/http.FromRateCard and AsRateCards. | New plan fields must be mapped in both FromPlan and AsCreatePlanRequest/AsUpdatePlanRequest; missing fields are silently ignored at the API boundary. |

## Anti-Patterns

- Importing entdb or calling plan.Repository directly — all access must go through plan.Service.
- Hardcoding namespace strings instead of calling h.resolveNamespace(ctx).
- Omitting httptransport.WithErrorEncoder(productcataloghttp.ValidationErrorEncoder(...)) — plan validation errors won't map to proper HTTP status codes.
- Putting the credits.Enabled guard in the operation closure instead of the decoder — decode-phase guards run before any service call.
- Setting EffectivePeriod directly in UpdatePlan decoder — status transitions must go through PublishPlan/ArchivePlan.

## Decisions

- **Credits feature guard is duplicated in both CreatePlan and UpdatePlan decoders.** — credits.enabled must be enforced at the HTTP boundary; the service layer does not know about deployment configuration.
- **PublishPlan and ArchivePlan are separate explicit operations, not flags on UpdatePlan.** — Prevents callers from manipulating plan status via UpdatePlan; EffectivePeriod is explicitly zeroed in UpdatePlan so status transitions are gated through explicit publish/archive paths.

## Example: Add a new plan endpoint following the standard three-alias + handler pattern

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
			if err != nil {
				return MyPlanRequest{}, fmt.Errorf("failed to resolve namespace: %w", err)
			}
			return MyPlanRequest{NamespacedID: models.NamespacedID{Namespace: ns, ID: planID}}, nil
		},
// ...
```

<!-- archie:ai-end -->
