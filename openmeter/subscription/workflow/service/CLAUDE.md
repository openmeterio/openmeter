# service

<!-- archie:ai-start -->

> Concrete implementation of subscriptionworkflow.Service — the high-level orchestration layer that composes core subscription.Service with addon management, customer locking, transaction primitives, and feature flags to execute multi-step lifecycle operations (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity) atomically.

## Patterns

**All mutating operations wrap in transaction.Run** — Every method that touches multiple domain objects calls transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (T, error) { ... }). Nested helpers like syncWithAddons also wrap in their own transaction.Run for savepoint support. (`return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionView, error) { ... })`)
**lockCustomer before any customer-scoped write** — CreateFromPlan (and any new mutating workflow that creates subscriptions for a customer) calls s.lockCustomer(ctx, inp.CustomerID) inside the transaction before any write. Uses subscription.GetCustomerLock + lockr.LockForTX (pg_advisory_xact_lock). (`if err := s.lockCustomer(ctx, inp.CustomerID); err != nil { return def, err }`)
**Addon sync via restore+apply diff cycle** — syncWithAddons computes Diffable for before/after addon sets, calls spec.ApplyMany with GetRestores() to strip old addon contributions, then ApplyMany with GetApplies() to apply new ones, then Service.Update. Never directly patches the spec with addon data without restoring first. (`spec.ApplyMany(lo.Map(restores, func(d addondiff.Diffable, _ int) subscription.AppliesToSpec { return d.GetRestores() }), ...)
spec.ApplyMany(lo.Map(applies, ...), ...)
s.Service.Update(ctx, view.Subscription.NamespacedID, spec)`)
**MapSubscriptionErrors after spec operations** — After calling NewSpecFromPlan or spec.ApplyMany, always pipe errors through subscriptionworkflow.MapSubscriptionErrors(err) before returning — it translates domain-specific spec errors into the correct generic error types. (`if err := subscriptionworkflow.MapSubscriptionErrors(err); err != nil { return def, fmt.Errorf("failed to create spec from plan: %w", err) }`)
**Domain errors via models.NewGeneric* constructors** — Validation failures return models.NewGenericValidationError, conflicts return models.NewGenericConflictError, forbidden state transitions return models.NewGenericForbiddenError, pre-condition failures return models.NewGenericPreConditionFailedError. Never return raw fmt.Errorf for these cases. (`return def, models.NewGenericConflictError(fmt.Errorf("subscription already has that addon purchased"))`)
**Feature flag gate via ffx.Service before entering transaction** — Feature flags (e.g. subscription.MultiSubscriptionEnabledFF) are checked via s.FeatureFlags.IsFeatureEnabled(ctx, ...) and short-circuit with a domain error before entering the transaction to avoid unnecessary locking. (`multiSubscriptionEnabled, err := s.FeatureFlags.IsFeatureEnabled(ctx, subscription.MultiSubscriptionEnabledFF)`)
**Compile-time interface assertion in service.go** — var _ subscriptionworkflow.Service = &service{} in service.go enforces the struct fully implements the interface at compile time. Must be present for any new implementation. (`var _ subscriptionworkflow.Service = &service{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines WorkflowServiceConfig (all injected deps) and the service struct. NewWorkflowService is the only constructor — wire it via app/common. lockCustomer helper lives here. | New dependencies must go into WorkflowServiceConfig, never stored as separate struct fields. lockCustomer calls LockForTX which requires an active transaction already in ctx. |
| `subscription.go` | Implements CreateFromPlan, EditRunning, ChangeToPlan, Restore — all cross-subscription lifecycle ops. | EditRunning blocks if subscription has active addons (hasAddons check). ChangeToPlan cancels old sub then creates new one in a single transaction, injecting previousSubscriptionID/supersedingSubscriptionID annotations. Restore is gated behind MultiSubscriptionEnabledFF. spec.ValidateAlignment() must be called after every ApplyMany sequence. |
| `addon.go` | Implements AddAddon and ChangeAddonQuantity. Both follow: validate → transaction → fetch current state → mutate addon record → syncWithAddons → return updated view. syncWithAddons is the core diff-apply loop. | Duplicate addon purchase detected by ID equality inside the transaction — must remain inside the transaction. asDiffs filters nil Diffable items; mismatch between len(diffs) and len(addons) is a hard error that surfaces as a generic error. |
| `addon_test.go` | Integration tests for AddAddon and ChangeAddonQuantity using subscriptiontestutils.SetupDBDeps + NewService for full DB-backed testing. | clock.FreezeTime(now.Add(time.Millisecond)) pattern avoids Postgres timestamp truncation boundary issues — replicate this in new tests. |
| `subscription_test.go` | Integration tests for CreateFromPlan, EditRunning, TestEditingCurrentPhase. Also demonstrates MockService injection for unit-testing delegation without a DB. | MockService injection pattern (workflowservice.NewWorkflowService with custom UpdateFn/GetViewFn) is the correct approach for unit-testing delegation behavior in isolation. |

## Anti-Patterns

- Calling s.Service.Update directly without the restore+apply diff cycle when addons are present — always use syncWithAddons for addon-aware spec updates
- Starting a new outer transaction.Run inside syncWithAddons when it is already called from within an outer transaction.Run — transaction.Run is safe via savepoints but callers must not create an outer wrapper that bypasses the existing transaction
- Returning raw fmt.Errorf for validation, conflict, or forbidden errors — always use models.NewGenericValidationError / NewGenericConflictError / NewGenericForbiddenError
- Skipping lockCustomer in new mutating workflow methods that create or modify subscriptions for a customer — concurrent subscription mutations for the same customer produce race conditions
- Directly editing the subscription spec fields without calling spec.ValidateAlignment() after applying patches

## Decisions

- **Workflow service is a separate layer from subscription.Service rather than expanding the core service** — Core subscription.Service stays persistence-focused; workflow layer composes it with addon, customer locking, and feature-flag concerns without polluting the domain interface or creating circular imports back into app/common.
- **Addon sync uses a full restore-then-apply diff cycle instead of incremental patching** — Ensures idempotency: the spec always reflects exactly the addons currently attached, not accumulated patches. Prevents double-application on retry and keeps the spec consistent with the current addon state.
- **ChangeToPlan cancels the old subscription then creates a new one within a single transaction, storing cross-reference annotations** — Preserves navigable subscription history (previousSubscriptionID / supersedingSubscriptionID annotations) while guaranteeing atomicity — there is never a window where neither subscription is active.

## Example: Add an addon to a subscription: validate, lock in transaction, detect duplicate, create addon record, sync spec

```
func (s *service) AddAddon(ctx context.Context, subscriptionID models.NamespacedID, addonInp subscriptionworkflow.AddAddonWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error) {
	var def1 subscription.SubscriptionView
	var def2 subscriptionaddon.SubscriptionAddon

	if err := addonInp.Validate(); err != nil {
		return def1, def2, models.NewGenericValidationError(err)
	}

	res, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (purchaseRes, error) {
		var def purchaseRes
		subView, _ := s.Service.GetView(ctx, subscriptionID)
		subsAdds, _ := s.AddonService.List(ctx, subscriptionID.Namespace, ...)
		if lo.SomeBy(subsAdds.Items, func(sa subscriptionaddon.SubscriptionAddon) bool { return sa.Addon.ID == addonInp.AddonID }) {
			return def, models.NewGenericConflictError(fmt.Errorf("subscription already has that addon purchased"))
		}
// ...
```

<!-- archie:ai-end -->
