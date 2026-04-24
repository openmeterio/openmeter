# plan

<!-- archie:ai-start -->

> Domain package for plan lifecycle management: defines Plan (with Phases and RateCards), Phase/RateCard managed types, typed errors, domain events, a custom JSON serializer, and the Service + Repository interfaces. Primary constraint: EffectivePeriod changes are only allowed via Publish/Archive, never via UpdatePlan.

## Patterns

**Plan embeds PlanMeta + []Phase; Phase embeds PhaseManagedFields + productcatalog.Phase** — Plan struct = NamespacedID + ManagedModel + PlanMeta + []Phase. Phase = PhaseManagedFields{ManagedModel, NamespacedID, PlanID} + productcatalog.Phase. RateCard = productcatalog.RateCard + RateCardManagedFields{PhaseID}. (`type Plan struct { models.NamespacedID; models.ManagedModel; productcatalog.PlanMeta; Phases []Phase }`)
**Typed NotFoundError wrapping models.NewGenericNotFoundError** — Same pattern as addon: NewNotFoundError(NotFoundErrorParams{...}), IsNotFound() for errors.As. Never surface raw Ent not-found. (`return nil, plan.NewNotFoundError(plan.NotFoundErrorParams{Namespace: ns, Key: key, Version: v})`)
**Custom MarshalJSON/UnmarshalJSON on Plan for polymorphic RateCards** — serializer.go implements Plan.MarshalJSON/UnmarshalJSON using alias types to avoid recursion. RateCards serialized with RateCardSerde type discriminator (flat_fee / usage_based). (`json.Marshal(plan) // uses Plan.MarshalJSON which dispatches through rateCardAlias`)
**EffectivePeriod zeroed in UpdatePlanInput to prevent direct status manipulation** — UpdatePlanInput embeds productcatalog.EffectivePeriod but service must zero it before calling adapter. Callers cannot set status via update; only Publish/Archive are valid paths. (`// service: input.EffectivePeriod = productcatalog.EffectivePeriod{} before calling adapter.UpdatePlan`)
**ValidatorFunc[Plan] for status/deletion guards** — validators.go provides IsPlanDeleted(at) and HasPlanStatus(statuses...) as composable models.ValidatorFunc[Plan] used with Plan.ValidateWith(). (`if err := p.ValidateWith(plan.HasPlanStatus(productcatalog.PlanStatusDraft)); err != nil { return nil, err }`)
**Input types embed inputOptions and call models.AsValidationIssues** — CreatePlanInput and UpdatePlanInput both embed inputOptions{IgnoreNonCriticalIssues} and use models.AsValidationIssues+WithSeverityOrHigher in Validate(). (`issues, err := models.AsValidationIssues(errors.Join(errs...)); if i.IgnoreNonCriticalIssues { issues = issues.WithSeverityOrHigher(models.ErrorSeverityCritical) }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Service interface (ListPlans, CreatePlan, DeletePlan, GetPlan, UpdatePlan, PublishPlan, ArchivePlan, NextPlan) and all input types with Validate(). | PublishPlanInput.Validate() uses a 30s jitter clock check for EffectiveFrom; use pkg/clock.Now() not time.Now(). |
| `plan.go` | Plan aggregate with Validate(), ValidateWith(), AsProductCatalogPlan(). | Addons field is *[]Addon — only populated on expand. Never assume it is non-nil. |
| `phase.go` | Phase and PhaseManagedFields types; ValidatePhaseManagedFields() and ValidatePhase() validator funcs. | PhaseManagedFields.Validate() asserts PlanID != '' — always populate PlanID when constructing a Phase in adapters. |
| `ratecard.go` | RateCard wrapping productcatalog.RateCard with PhaseID; custom MarshalJSON/UnmarshalJSON dispatches on type. | New rate card types require a case in UnmarshalJSON and the serializer switch. |
| `serializer.go` | Custom JSON marshaler/unmarshaler for Plan; uses alias types to avoid recursion, rateCardAlias for polymorphic cards. | serializer_test.go covers round-trip plus error paths; any new field on Plan/Phase/RateCard must be reflected here. |
| `validators.go` | IsPlanDeleted and HasPlanStatus validator funcs. | IsPlanDeleted takes a time.Time argument (the check time); use clock.Now() at the call site. |

## Anti-Patterns

- Setting EffectivePeriod via UpdatePlan — it must be zeroed in the service; only Publish/Archive alter it.
- Skipping resolveFeatures/resolveTaxCodes before persisting rate cards in the service.
- Returning raw entdb.IsNotFound — always wrap in plan.NewNotFoundError.
- Publishing events outside a transaction.Run closure — DB write and event publish must be atomic.
- Editing openmeter/ent/db/ generated files — always regenerate with make generate.

## Decisions

- **EffectivePeriod changes are strictly gated to Publish/Archive operations** — UpdatePlan explicitly zeroes EffectivePeriod to prevent callers from manipulating plan status through the update path; status transitions are explicit state-machine operations.
- **Custom JSON serializer on Plan instead of generated oapi-codegen types** — productcatalog.RateCard is a polymorphic interface; the serializer dispatches on the type discriminator to pick the correct concrete struct, which generated decoders cannot handle.
- **Version auto-incremented in service layer, not settable by callers** — Prevents callers from forging version numbers; the service computes the next version from existing versions at create/next time.

## Example: Implementing a service mutation with transaction and event publish

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
        event := plan.NewPlanCreateEvent(ctx, created)
        if err := s.publisher.Publish(ctx, eventbus.SystemTopic, event); err != nil {
            return nil, fmt.Errorf("publish plan created: %w", err)
        }
        return created, nil
// ...
```

<!-- archie:ai-end -->
