# service

<!-- archie:ai-start -->

> Business-logic service implementing plan.Service: validates inputs, resolves feature/taxcode cross-references, enforces status-based mutation guards, delegates persistence to plan.Repository, and publishes domain events. Primary constraint: all mutations wrap adapter calls and event publishing inside transaction.Run.

## Patterns

**transaction.Run wrapping for all mutations** — Create, Update, Delete, Publish, Archive, Next wrap adapter calls and publisher.Publish inside transaction.Run(ctx, s.adapter, fn) for atomicity. (`return transaction.Run(ctx, s.adapter, fn)`)
**resolveFeatures + resolveTaxCodes before adapter write** — Before CreatePlan/UpdatePlan persists, resolveFeatures populates FeatureKey↔FeatureID and resolveTaxCodes populates TaxCodeID; missing features become GenericValidationError. (`if err = s.resolveFeatures(ctx, params.Namespace, &phase.RateCards); err != nil { if models.IsGenericNotFoundError(err) { err = models.NewGenericValidationError(err) }; return nil, err }`)
**Event publishing inside the transaction closure** — After a successful adapter mutation, the method publishes plan.NewPlanXxxEvent inside the closure so a publish failure rolls back the DB write. (`if err := s.publisher.Publish(ctx, plan.NewPlanCreateEvent(ctx, p)); err != nil { return nil, fmt.Errorf("failed to publish plan created event: %w", err) }`)
**Status guard before mutations** — UpdatePlan/DeletePlan/PublishPlan/ArchivePlan check allowed statuses via lo.Contains before delegating; violations return GenericValidationError. (`if !lo.Contains([]productcatalog.PlanStatus{productcatalog.PlanStatusDraft, productcatalog.PlanStatusScheduled}, p.Status()) { return nil, models.NewGenericValidationError(...) }`)
**Config-based constructor with non-nil validation** — New(Config) validates Adapter, Feature, TaxCode, Logger, Publisher are non-nil; var _ plan.Service = (*service)(nil) enforces interface satisfaction. (`func New(config Config) (plan.Service, error) { if err := config.Validate(); err != nil { return nil, err }; return service{...}, nil }`)
**Version auto-increment in CreatePlan** — CreatePlan queries existing versions for the plan key and sets params.Version = max + 1; callers must not pass a version. (`if p.Version >= params.Version { params.Version = p.Version + 1 }`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Config struct, New constructor, service struct with private dependency fields (adapter, feature, taxCode, logger, publisher). | Adding a dependency requires updating Config validation, Config struct, and service struct together. |
| `plan.go` | All plan.Service methods plus resolveFeatures/resolveTaxCodes helpers. | EffectivePeriod is explicitly zeroed in UpdatePlan to prevent status manipulation. Default settlement mode CreditThenInvoice is applied here when not provided. |
| `service_test.go` | Integration tests using pctestutils.NewTestEnv, driving behavior through env.Plan (service), not env.PlanRepository (adapter). | Existing tests use context.Background(); new tests should prefer t.Context() per project convention. |

## Anti-Patterns

- Calling s.adapter directly for mutations without transaction.Run — breaks atomicity with event publishing.
- Allowing EffectivePeriod to be set via UpdatePlan — it is explicitly zeroed.
- Skipping resolveFeatures before CreatePlan/UpdatePlan — rate cards store incomplete feature cross-references.
- Publishing events outside the transaction closure — both DB write and publish must be inside transaction.Run.
- Returning entdb/adapter-level errors directly instead of domain errors (GenericNotFoundError, GenericValidationError).

## Decisions

- **Feature and TaxCode resolution lives in the service, not the adapter.** — Adapter handles only persistence; cross-entity lookups are orchestration concerns.
- **Version is auto-incremented from existing versions; callers cannot set it.** — Prevents version gaps and ensures monotonically increasing versions per plan key under concurrent creates.
- **Default settlement mode CreditThenInvoice is applied in the service, not the HTTP handler.** — Keeps the default a domain invariant so non-HTTP callers get the same default.

## Example: A mutating service method that calls the adapter and publishes an event

```
func (s service) MyMutation(ctx context.Context, params plan.MyInput) (*plan.Plan, error) {
	fn := func(ctx context.Context) (*plan.Plan, error) {
		if err := params.Validate(); err != nil { return nil, fmt.Errorf("invalid params: %w", err) }
		p, err := s.adapter.GetPlan(ctx, plan.GetPlanInput{NamespacedID: params.NamespacedID})
		if err != nil { return nil, fmt.Errorf("failed to get Plan: %w", err) }
		if !lo.Contains([]productcatalog.PlanStatus{productcatalog.PlanStatusDraft}, p.Status()) {
			return nil, models.NewGenericValidationError(fmt.Errorf("plan must be in draft status"))
		}
		p, err = s.adapter.UpdatePlan(ctx, plan.UpdatePlanInput{NamespacedID: params.NamespacedID})
		if err != nil { return nil, err }
		if err := s.publisher.Publish(ctx, plan.NewPlanUpdateEvent(ctx, p)); err != nil { return nil, err }
		return p, nil
	}
	return transaction.Run(ctx, s.adapter, fn)
}
```

<!-- archie:ai-end -->
