# plan

<!-- archie:ai-start -->

> Domain package for plan lifecycle management: defines Plan (with Phases and RateCards), Phase/RateCard managed types, typed errors, domain events, a custom JSON serializer, and the Service + Repository interfaces; children split into adapter (Ent), httpdriver (v1 HTTP), and service (validation, feature/taxcode resolution, events). Primary constraint: EffectivePeriod changes are only allowed via Publish/Archive, never via UpdatePlan.

## Patterns

**Nested managed-type embedding** — Plan = NamespacedID + ManagedModel + PlanMeta + []Phase; Phase = PhaseManagedFields{ManagedModel, NamespacedID, PlanID} + productcatalog.Phase; RateCard = productcatalog.RateCard + RateCardManagedFields{PhaseID}. (`type Plan struct { models.NamespacedID; models.ManagedModel; productcatalog.PlanMeta; Phases []Phase }`)
**EffectivePeriod zeroed in UpdatePlan** — UpdatePlanInput embeds productcatalog.EffectivePeriod but the service must zero it before calling the adapter — status transitions only go through Publish/Archive. (`// service: input.EffectivePeriod = productcatalog.EffectivePeriod{} before adapter.UpdatePlan`)
**Custom Plan JSON serializer for polymorphic RateCards** — serializer.go implements Plan.Marshal/UnmarshalJSON using alias types to avoid recursion, dispatching RateCards via the RateCardSerde type discriminator (flat_fee / usage_based). Any new field on Plan/Phase/RateCard must be reflected here. (`json.Marshal(plan) // Plan.MarshalJSON dispatches through rateCardAlias`)
**ValidatorFunc[Plan] status/deletion guards** — validators.go provides IsPlanDeleted(at) and HasPlanStatus(statuses...) as composable models.ValidatorFunc[Plan] used via Plan.ValidateWith(). (`if err := p.ValidateWith(plan.HasPlanStatus(productcatalog.PlanStatusDraft)); err != nil { return nil, err }`)
**Typed NotFoundError + draft validation options** — Adapters return plan.NewNotFoundError(...) (never raw Ent not-found); CreatePlanInput/UpdatePlanInput embed inputOptions{IgnoreNonCriticalIssues} with AsValidationIssues+WithSeverityOrHigher for relaxed draft validation. (`return nil, plan.NewNotFoundError(plan.NotFoundErrorParams{Namespace: ns, Key: key, Version: v})`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface (List/Create/Delete/Get/Update/Publish/Archive/Next) and input types with Validate(). | PublishPlanInput.Validate() uses a 30s jitter via pkg/clock.Now() — not time.Now(). |
| `plan.go` | Plan aggregate with Validate(), ValidateWith(), AsProductCatalogPlan(). | Addons field is *[]Addon — only populated on expand; never assume non-nil. |
| `phase.go` | Phase and PhaseManagedFields types with their validator funcs. | PhaseManagedFields.Validate() asserts PlanID != '' — always populate PlanID when constructing a Phase in adapters. |
| `ratecard.go` | RateCard wrapping productcatalog.RateCard with PhaseID; custom Marshal/UnmarshalJSON dispatching on type. | New rate card types require a case in UnmarshalJSON and the serializer switch. |
| `serializer.go` | Custom JSON marshaler/unmarshaler for Plan using alias types to avoid recursion. | Any new field on Plan/Phase/RateCard must be added here; serializer_test.go covers round-trip and error paths. |
| `validators.go` | IsPlanDeleted and HasPlanStatus validator funcs. | IsPlanDeleted takes a time.Time check argument — use clock.Now() at the call site. |

## Anti-Patterns

- Setting EffectivePeriod via UpdatePlan — it must be zeroed in the service; only Publish/Archive alter it.
- Skipping resolveFeatures/resolveTaxCodes before persisting rate cards in the service.
- Returning raw entdb.IsNotFound — always wrap in plan.NewNotFoundError.
- Publishing events outside a transaction.Run closure — DB write and event publish must be atomic.
- Editing openmeter/ent/db/ generated files — always regenerate with make generate.

## Decisions

- **EffectivePeriod changes are strictly gated to Publish/Archive.** — UpdatePlan zeroes EffectivePeriod so callers cannot manipulate plan status via update; status transitions are explicit state-machine operations.
- **Custom JSON serializer on Plan instead of generated oapi-codegen types.** — productcatalog.RateCard is a polymorphic interface; the serializer dispatches on the type discriminator to pick the concrete struct, which generated decoders cannot handle.
- **Version auto-incremented in the service layer, not settable by callers.** — Prevents callers from forging version numbers; the service computes the next version from existing versions at create/next time.

## Example: Service mutation wrapping adapter call and event publish in transaction.Run

```
import (
    "github.com/openmeterio/openmeter/openmeter/productcatalog/plan"
    "github.com/openmeterio/openmeter/pkg/framework/transaction"
)

func (s *service) CreatePlan(ctx context.Context, params plan.CreatePlanInput) (*plan.Plan, error) {
    if err := params.Validate(); err != nil { return nil, err }
    return transaction.Run(ctx, s.adapter, func(ctx context.Context) (*plan.Plan, error) {
        created, err := s.adapter.CreatePlan(ctx, params)
        if err != nil { return nil, err }
        if err := s.publisher.Publish(ctx, plan.NewPlanCreateEvent(ctx, created)); err != nil {
            return nil, fmt.Errorf("publish plan created: %w", err)
        }
        return created, nil
    })
// ...
```

<!-- archie:ai-end -->
