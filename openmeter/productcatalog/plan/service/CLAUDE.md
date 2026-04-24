# service

<!-- archie:ai-start -->

> Business-logic service implementing plan.Service: validates inputs, resolves feature/taxcode references, enforces status-based mutation guards, delegates to plan.Repository, and publishes domain events. Primary constraint: state transitions (publish/archive/next) are separate operations; UpdatePlan only modifies metadata and phases on Draft/Scheduled plans.

## Patterns

**transaction.Run wrapping for mutations** — All mutating methods wrap their closure in transaction.Run(ctx, s.adapter, fn) to ensure atomicity across adapter calls and event publishing. (`return transaction.Run(ctx, s.adapter, fn)`)
**resolveFeatures before adapter calls** — Before CreatePlan/UpdatePlan hits the adapter, s.resolveFeatures populates FeatureKey↔FeatureID cross-references by querying feature.FeatureConnector; missing features are converted to GenericValidationError. (`if err = s.resolveFeatures(ctx, params.Namespace, &phase.RateCards); err != nil { if models.IsGenericNotFoundError(err) { err = models.NewGenericValidationError(err) }; return nil, ... }`)
**Publish domain events after adapter write** — Create/Update/Delete/Publish/Archive/Next all call s.publisher.Publish(ctx, plan.NewPlanXxxEvent(ctx, p)) after the adapter mutation succeeds, inside the transaction closure. (`event := plan.NewPlanCreateEvent(ctx, p); if err := s.publisher.Publish(ctx, event); err != nil { return nil, fmt.Errorf("failed to publish plan created event: %w", err) }`)
**Status-guard before mutations** — UpdatePlan, DeletePlan, PublishPlan, ArchivePlan each check allowed statuses via lo.Contains before delegating to adapter; violations return models.NewGenericValidationError. (`allowedPlanStatuses := []productcatalog.PlanStatus{productcatalog.PlanStatusDraft, productcatalog.PlanStatusScheduled}; if !lo.Contains(allowedPlanStatuses, planStatus) { return nil, models.NewGenericValidationError(...) }`)
**Config-based constructor with nil checks** — New(Config) validates all dependencies (Adapter, Feature, TaxCode, Logger, Publisher) are non-nil before returning the service. (`var _ plan.Service = (*service)(nil)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct, New constructor, service struct definition. All dependency fields (adapter, feature, taxCode, logger, publisher) are private. | Adding a new dependency requires updating Config, New validation, and service struct — all three must stay in sync. |
| `plan.go` | All plan.Service method implementations. resolveFeatures and resolveTaxCodes are private helpers called before adapter writes. | CreatePlan auto-increments version based on existing versions; callers should not pass a Version in CreatePlanInput. |
| `service_test.go` | Integration tests using pctestutils.NewTestEnv which wires the full stack (adapter + service). Tests drive behavior through env.Plan (service) not env.PlanRepository (adapter). | Tests use context.Background() — in new tests prefer t.Context() per project convention. |

## Anti-Patterns

- Calling s.adapter directly for mutations without wrapping in transaction.Run — breaks atomicity with event publishing.
- Allowing EffectivePeriod to be set via UpdatePlan — it is explicitly zeroed to prevent direct status manipulation.
- Skipping resolveFeatures before CreatePlan/UpdatePlan — rate cards with FeatureKey-only or FeatureID-only refs will have incomplete cross-references in the DB.
- Publishing events outside the transaction closure — if the DB write succeeds but the event publish fails, the transaction rolls back only if both are inside transaction.Run.

## Decisions

- **Feature and TaxCode resolution is done in service layer, not in the adapter.** — Adapter only handles persistence; cross-entity lookups (feature by key/ID, tax code by stripe code) are orchestration concerns belonging in the service.
- **Version is auto-incremented from existing versions; callers cannot set it directly.** — Prevents version gaps and ensures monotonically increasing versions per key across create calls.
- **Default settlement mode CreditThenInvoice is applied in service, not in the HTTP handler.** — Keeps the default a domain invariant independent of the transport layer.

## Example: Add a new mutating service method that calls the adapter and publishes an event

```
func (s service) MyMutation(ctx context.Context, params plan.MyInput) (*plan.Plan, error) {
	fn := func(ctx context.Context) (*plan.Plan, error) {
		if err := params.Validate(); err != nil {
			return nil, fmt.Errorf("invalid params: %w", err)
		}
		p, err := s.adapter.GetPlan(ctx, plan.GetPlanInput{NamespacedID: params.NamespacedID})
		if err != nil {
			return nil, fmt.Errorf("failed to get Plan: %w", err)
		}
		if !lo.Contains([]productcatalog.PlanStatus{productcatalog.PlanStatusDraft}, p.Status()) {
			return nil, models.NewGenericValidationError(fmt.Errorf("plan must be in draft status"))
		}
		p, err = s.adapter.UpdatePlan(ctx, plan.UpdatePlanInput{NamespacedID: params.NamespacedID})
		if err != nil {
			return nil, fmt.Errorf("failed to update Plan: %w", err)
// ...
```

<!-- archie:ai-end -->
