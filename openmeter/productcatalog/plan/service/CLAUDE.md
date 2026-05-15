# service

<!-- archie:ai-start -->

> Business-logic service implementing plan.Service: validates inputs, resolves feature/taxcode cross-references, enforces status-based mutation guards, delegates persistence to plan.Repository (adapter), and publishes domain events via Watermill. Primary constraint: all mutations must wrap adapter calls and event publishing inside transaction.Run.

## Patterns

**transaction.Run wrapping for all mutations** — All mutating methods (Create, Update, Delete, Publish, Archive, Next) wrap their adapter calls and publisher.Publish inside transaction.Run(ctx, s.adapter, fn) to ensure atomicity. (`return transaction.Run(ctx, s.adapter, fn)`)
**resolveFeatures + resolveTaxCodes before adapter write** — Before CreatePlan/UpdatePlan hits the adapter, s.resolveFeatures populates FeatureKey↔FeatureID cross-references and s.resolveTaxCodes populates TaxCodeID. Missing features are converted to GenericValidationError. (`if err = s.resolveFeatures(ctx, params.Namespace, &phase.RateCards); err != nil { if models.IsGenericNotFoundError(err) { err = models.NewGenericValidationError(err) }; return nil, err }`)
**Event publishing inside transaction closure** — After adapter mutation succeeds, each mutating method calls s.publisher.Publish(ctx, plan.NewPlanXxxEvent(ctx, p)) inside the transaction closure. If publish fails the transaction rolls back. (`event := plan.NewPlanCreateEvent(ctx, p); if err := s.publisher.Publish(ctx, event); err != nil { return nil, fmt.Errorf("failed to publish plan created event: %w", err) }`)
**Status guard before mutations** — UpdatePlan, DeletePlan, PublishPlan, ArchivePlan each check allowed statuses via lo.Contains before delegating to adapter; violations return models.NewGenericValidationError. (`allowedPlanStatuses := []productcatalog.PlanStatus{productcatalog.PlanStatusDraft, productcatalog.PlanStatusScheduled}; if !lo.Contains(allowedPlanStatuses, p.Status()) { return nil, models.NewGenericValidationError(...) }`)
**Config-based constructor with non-nil dependency validation** — New(Config) validates that Adapter, Feature, TaxCode, Logger, and Publisher are all non-nil before returning service. var _ plan.Service = (*service)(nil) ensures compile-time interface satisfaction. (`func New(config Config) (plan.Service, error) { if err := config.Validate(); err != nil { return nil, err }; return service{...}, nil }`)
**Version auto-increment in CreatePlan** — CreatePlan queries all existing versions for the plan key and auto-sets params.Version = max(existing) + 1. Callers must not pass a version; the service enforces monotonically increasing versions. (`if p.Version >= params.Version { params.Version = p.Version + 1 }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct, New constructor, and service struct. All dependency fields (adapter, feature, taxCode, logger, publisher) are private. | Adding a new dependency requires updating Config validation, Config struct, and service struct — all three must stay in sync. |
| `plan.go` | All plan.Service method implementations plus resolveFeatures and resolveTaxCodes private helpers. | EffectivePeriod is explicitly zeroed in UpdatePlan to prevent direct status manipulation; do not allow callers to set it via this method. Default settlement mode CreditThenInvoice is applied here when not provided. |
| `service_test.go` | Integration tests using pctestutils.NewTestEnv which wires the full stack (adapter + service). Tests drive behavior through env.Plan (service) not env.PlanRepository (adapter). | Tests use context.Background(); in new tests prefer t.Context() per project convention. |

## Anti-Patterns

- Calling s.adapter directly for mutations without wrapping in transaction.Run — breaks atomicity with event publishing.
- Allowing EffectivePeriod to be set via UpdatePlan — it is explicitly zeroed to prevent direct status manipulation.
- Skipping resolveFeatures before CreatePlan/UpdatePlan — rate cards with FeatureKey-only or FeatureID-only refs will have incomplete cross-references stored in the DB.
- Publishing events outside the transaction closure — if DB write succeeds but publish fails the transaction only rolls back when both are inside transaction.Run.
- Returning entdb or adapter-level errors directly to callers — convert them to domain errors (models.GenericNotFoundError, models.GenericValidationError) at the service boundary.

## Decisions

- **Feature and TaxCode resolution is done in the service layer, not in the adapter.** — Adapter handles only persistence; cross-entity lookups (feature by key/ID, tax code by stripe code) are orchestration concerns that belong in the service.
- **Version is auto-incremented from existing versions; callers cannot set it directly.** — Prevents version gaps and ensures monotonically increasing versions per plan key across concurrent create calls.
- **Default settlement mode CreditThenInvoice is applied in the service, not in the HTTP handler.** — Keeps the default a domain invariant independent of the transport layer so non-HTTP callers get the same default.

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
