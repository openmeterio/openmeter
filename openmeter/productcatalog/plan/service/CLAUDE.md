# service

<!-- archie:ai-start -->

> Business-logic layer implementing plan.Service: enforces plan versioning, lifecycle/status rules, feature and tax-code resolution, event publishing, and transactional orchestration around the adapter. This is where plan invariants live.

## Patterns

**Config + New(plan.Service) with required-dependency checks** — service.go validates Adapter, FeatureResolver, Logger, TaxCode, Publisher are non-nil before building the unexported service struct. var _ plan.Service = (*service)(nil) enforces the interface. (`func New(config Config) (plan.Service, error) { if config.Adapter == nil { return nil, errors.New("plan adapter is required") } ... }`)
**transaction.Run around mutating operations** — Create/Delete/Update/Publish wrap their fn in transaction.Run(ctx, s.adapter, fn); read-only ListPlans/GetPlan just call fn(ctx) directly. All adapter calls inside share the tx. (`return transaction.Run(ctx, s.adapter, fn)`)
**Validate -> resolve features/tax -> persist -> publish event** — Mutations call params.Validate(), then featureresolver.ResolveFeaturesForRateCards and s.resolveTaxCodes per phase, then the adapter, then s.publisher.Publish(ctx, plan.NewPlanXEvent(ctx, p)). A failed publish fails the whole tx. (`event := plan.NewPlanCreateEvent(ctx, p); if err := s.publisher.Publish(ctx, event); err != nil { return nil, err }`)
**Status-gated lifecycle transitions** — Delete allows Archived/Scheduled/Draft; Update/Publish allow only Draft/Scheduled; CreatePlan enforces a single Draft version per Key and auto-increments Version. Violations return models.NewGenericValidationError. (`if !lo.Contains(allowedPlanStatuses, planStatus) { return nil, models.NewGenericValidationError(...) }`)
**Field-prefixed validation errors for phase rate cards** — Feature resolution errors are wrapped with models.ErrorWithFieldPrefix using a phases[key=...] FieldSelectorGroup so API clients get precise field paths. (`models.ErrorWithFieldPrefix(phaseFieldSelector, fmt.Errorf("failed to expand features for ratecards...: %w", err))`)
**Publish performs deep cross-aggregate validation** — PublishPlan fetches with Expand{PlanAddons:true}, runs pp.Validate(), ValidatePlanWithFeatures, and per-addon PlanAddon.Validate (ErrPlanHasIncompatibleAddon), collecting errs via errors.Join, then archives the prior active version. (`err = pp.ValidateWith(productcatalog.ValidatePlanWithFeatures(ctx, s.featureResolver.WithNamespace(params.Namespace)))`)
**resolveTaxCodes ensures Stripe tax codes exist before persist** — For each rate card with a TaxConfig, productcatalog.ResolveTaxConfig creates/links a TaxCode entity and populates TaxConfig.TaxCodeID, then rc.Merge applies the resolved meta back onto the typed rate card. (`if err := productcatalog.ResolveTaxConfig(ctx, s.taxCode, namespace, meta.TaxConfig); err != nil { return err }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config, New constructor, service struct, dependency wiring | All five deps (Adapter, FeatureResolver, Logger, TaxCode, Publisher) are mandatory; no slog.Default fallback allowed. |
| `plan.go` | All Service methods: ListPlans/CreatePlan/DeletePlan/GetPlan/UpdatePlan/PublishPlan/ArchivePlan and resolveTaxCodes | UpdatePlan zeroes EffectivePeriod (lifecycle only via Publish/Archive); CreatePlan rejects a second Draft and auto-increments Version; events publish inside the tx. |
| `service_test.go` | Integration tests via pctestutils.NewTestEnv covering plan lifecycle, addons, filters | Builds real meters/features/addons; prefer service-backed fixtures over hand-assembled aggregates. |
| `taxcode_test.go` | Tests for resolveTaxCodes / tax code resolution on rate cards | Asserts TaxCodeID population and TaxCode entity creation per namespace. |

## Anti-Patterns

- Calling the adapter for a mutation outside transaction.Run, losing atomicity across phase/feature/tax steps and event publish
- Allowing updates or publishes on Active/Archived plans (status gate must reject)
- Skipping featureresolver.ResolveFeaturesForRateCards or resolveTaxCodes before persisting phases
- Permitting more than one Draft version per plan Key in CreatePlan
- Publishing without Expand{PlanAddons:true}, so add-on compatibility can't be validated

## Decisions

- **Versioning and lifecycle rules live in the service, not the adapter or handler** — Single-draft-per-key, version auto-increment, and status-gated transitions are domain invariants that span multiple reads/writes.
- **Event publish happens inside the same transaction as the write** — A failed Publish rolls back the mutation, keeping the event stream consistent with persisted state (outbox-style coupling).
- **EffectivePeriod is mutated only via Publish/Archive, zeroed on Update** — Forces scheduling through dedicated lifecycle operations rather than arbitrary field edits.

## Example: Mutation: validate, resolve features/tax per phase, persist, publish in one tx

```
func (s service) CreatePlan(ctx context.Context, params plan.CreatePlanInput) (*plan.Plan, error) {
  fn := func(ctx context.Context) (*plan.Plan, error) {
    if err := params.Validate(); err != nil { return nil, err }
    // enforce single-draft-per-key + version auto-increment via ListPlans
    for idx := range params.Phases {
      if err := featureresolver.ResolveFeaturesForRateCards(ctx, s.featureResolver, params.Namespace, &params.Phases[idx].RateCards); err != nil { return nil, err }
      if err := s.resolveTaxCodes(ctx, params.Namespace, &params.Phases[idx].RateCards); err != nil { return nil, err }
    }
    p, err := s.adapter.CreatePlan(ctx, params)
    if err != nil { return nil, err }
    if err := s.publisher.Publish(ctx, plan.NewPlanCreateEvent(ctx, p)); err != nil { return nil, err }
    return p, nil
  }
  return transaction.Run(ctx, s.adapter, fn)
}
```

<!-- archie:ai-end -->
