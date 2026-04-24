# service

<!-- archie:ai-start -->

> Concrete implementation of subscriptionworkflow.Service — the high-level orchestration layer that composes the core subscription.Service with addon, customer, locking, and transaction primitives to execute multi-step lifecycle operations (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity) atomically.

## Patterns

**Every mutating operation wraps in transaction.Run** — All methods that touch multiple domain objects call transaction.Run(ctx, s.TransactionManager, ...) so failures roll back atomically. Nested operations (e.g. syncWithAddons called from AddAddon) also wrap in their own transaction.Run for savepoint support. (`transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (T, error) { ... })`)
**lockCustomer before any customer-scoped write** — CreateFromPlan calls s.lockCustomer(ctx, inp.CustomerID) inside the transaction before any write to prevent concurrent subscription mutations for the same customer. Uses subscription.GetCustomerLock + lockr.LockForTX (pg_advisory_xact_lock). (`if err := s.lockCustomer(ctx, inp.CustomerID); err != nil { return def, err }`)
**Addon sync via restore+apply diff cycle** — syncWithAddons computes Diffable for before/after addon sets, calls spec.ApplyMany with GetRestores() to strip old addon contributions, then ApplyMany with GetApplies() to apply new ones, then Service.Update. Never directly patches the spec with addon data without restoring first. (`spec.ApplyMany(lo.Map(restores, func(d addondiff.Diffable, _ int) subscription.AppliesToSpec { return d.GetRestores() }), ...)
spec.ApplyMany(lo.Map(applies, ...), ...)
s.Service.Update(ctx, view.Subscription.NamespacedID, spec)`)
**Compile-time interface assertion** — var _ subscriptionworkflow.Service = &service{} in service.go enforces that the struct fully implements the interface at compile time. (`var _ subscriptionworkflow.Service = &service{}`)
**Domain errors via models.NewGeneric* constructors** — Validation failures return models.NewGenericValidationError, conflicts return models.NewGenericConflictError, forbidden state transitions return models.NewGenericForbiddenError, pre-condition failures return models.NewGenericPreConditionFailedError. Never return raw errors for these cases. (`return def1, def2, models.NewGenericConflictError(fmt.Errorf("subscription already has that addon purchased"))`)
**subscriptionworkflow.MapSubscriptionErrors for spec errors** — After calling NewSpecFromPlan or spec.ApplyMany, pipe the error through subscriptionworkflow.MapSubscriptionErrors(err) before returning — it translates domain-specific spec errors into the correct generic error types. (`if err := subscriptionworkflow.MapSubscriptionErrors(err); err != nil { return def, fmt.Errorf("failed to create spec from plan: %w", err) }`)
**Feature flag gate via ffx.Service** — Feature flags (e.g. subscription.MultiSubscriptionEnabledFF) are checked via s.FeatureFlags.IsFeatureEnabled(ctx, ...) and short-circuit the operation with a domain-specific error before entering the transaction. (`multiSubscriptionEnabled, err := s.FeatureFlags.IsFeatureEnabled(ctx, subscription.MultiSubscriptionEnabledFF)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines WorkflowServiceConfig (all injected deps) and the service struct. NewWorkflowService is the only constructor — wire it via app/common. lockCustomer helper is here. | Adding new dependencies must go into WorkflowServiceConfig; never store state outside that struct. |
| `subscription.go` | Implements CreateFromPlan, EditRunning, ChangeToPlan, Restore. All cross-subscription lifecycle ops. ChangeToPlan cancels old sub and creates new one in a single transaction, storing superseding/previous annotation cross-references. | EditRunning blocks if subscription has active addons (hasAddons check). Restore is gated behind MultiSubscriptionEnabledFF. BillingAnchor normalization and annotation injection (OwnerSubscriptionSubSystem, UniquePatchID) happen here. |
| `addon.go` | Implements AddAddon and ChangeAddonQuantity. Both follow the pattern: validate → transaction → fetch current state → mutate addon record → syncWithAddons → return updated view. syncWithAddons is the core diff-apply loop. | Duplicate addon purchase is detected by ID equality check before Create — guard is inside the transaction. asDiffs filters nil Diffable items; mismatch between len(diffs) and len(addons) is a hard error. |
| `addon_test.go` | Integration tests for AddAddon and ChangeAddonQuantity. Uses subscriptiontestutils.SetupDBDeps + NewService for full DB-backed testing with clock.FreezeTime. | clock.FreezeTime(now.Add(time.Millisecond)) pattern avoids Postgres timestamp truncation boundary issues — replicate this in new tests. |
| `subscription_test.go` | Integration tests for CreateFromPlan, EditRunning, TestEditingCurrentPhase. Also demonstrates manual WorkflowService construction with MockService for unit testing specific delegation behavior. | MockService injection pattern (workflowservice.NewWorkflowService with custom UpdateFn/GetViewFn) is the correct approach for unit-testing delegation without a DB. |

## Anti-Patterns

- Calling s.Service.Update directly without going through spec restore+apply cycle when addons are present — always use syncWithAddons for addon-aware updates
- Starting a new outer transaction.Run inside syncWithAddons when it is already called from an outer transaction.Run — transaction.Run is idempotent via savepoints but callers must not bypass the outer transaction
- Returning raw fmt.Errorf for validation, conflict, or forbidden errors — always use models.NewGenericValidationError / NewGenericConflictError / NewGenericForbiddenError
- Skipping lockCustomer in new mutating workflow methods that create or modify subscriptions for a customer
- Directly editing the subscription spec without calling spec.ValidateAlignment() after applying patches

## Decisions

- **Workflow service is a separate layer from subscription.Service rather than expanding the core service** — Core subscription.Service stays persistence-focused; workflow layer composes it with addon, customer, locking, and feature-flag concerns without polluting the domain interface or creating circular imports.
- **Addon sync uses a full restore-then-apply diff cycle instead of incremental patching** — Ensures idempotency: the spec always reflects exactly the addons currently attached, not accumulated patches. Prevents double-application on retry.
- **ChangeToPlan cancels the old subscription then creates a new one within a single transaction, storing cross-reference annotations** — Preserves a navigable subscription history (previousSubscriptionID / supersedingSubscriptionID annotations) while guaranteeing atomicity — no window where neither subscription is active.

## Example: Add an addon to a subscription: validate, lock in transaction, detect duplicate, create addon record, sync spec

```
func (s *service) AddAddon(ctx context.Context, subscriptionID models.NamespacedID, addonInp subscriptionworkflow.AddAddonWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error) {
	if err := addonInp.Validate(); err != nil {
		return def1, def2, models.NewGenericValidationError(err)
	}
	res, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (purchaseRes, error) {
		subView, _ := s.Service.GetView(ctx, subscriptionID)
		subsAdds, _ := s.AddonService.List(ctx, subscriptionID.Namespace, ...)
		if lo.SomeBy(subsAdds.Items, func(sa subscriptionaddon.SubscriptionAddon) bool { return sa.Addon.ID == addonInp.AddonID }) {
			return def, models.NewGenericConflictError(fmt.Errorf("subscription already has that addon purchased"))
		}
		subsAdd, _ := s.AddonService.Create(ctx, subscriptionID.Namespace, ...)
		subView, err = s.syncWithAddons(ctx, subView, subsAdds.Items, append(subsAdds.Items, *subsAdd), editTime)
		return purchaseRes{sub: subView, subAdd: *subsAdd}, err
	})
	return res.sub, res.subAdd, err
// ...
```

<!-- archie:ai-end -->
