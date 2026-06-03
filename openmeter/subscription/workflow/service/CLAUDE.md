# service

<!-- archie:ai-start -->

> Concrete implementation of subscriptionworkflow.Service — the high-level orchestration layer composing core subscription.Service with addon management, customer locking, transaction primitives, and feature flags to execute multi-step lifecycle operations (CreateFromPlan, EditRunning, ChangeToPlan, Restore, AddAddon, ChangeAddonQuantity) atomically.

## Patterns

**All mutating ops wrap in transaction.Run** — Every method touching multiple domain objects calls transaction.Run(ctx, s.TransactionManager, fn). Nested helpers like syncWithAddons also wrap in their own transaction.Run for savepoint support. (`return transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionView, error) { ... })`)
**lockCustomer before any customer-scoped write** — CreateFromPlan (and any new mutating workflow) calls s.lockCustomer(ctx, inp.CustomerID) inside the transaction before any write, using subscription.GetCustomerLock + lockr.LockForTX (pg_advisory_xact_lock). (`if err := s.lockCustomer(ctx, inp.CustomerID); err != nil { return def, err }`)
**Addon sync via restore+apply diff cycle** — syncWithAddons computes Diffable for before/after addon sets, calls spec.ApplyMany with GetRestores() to strip old contributions, then ApplyMany with GetApplies() to add new ones, then Service.Update. Never patch the spec with addon data without restoring first. (`spec.ApplyMany(lo.Map(restores, func(d addondiff.Diffable, _ int) subscription.AppliesToSpec { return d.GetRestores() }), ...); spec.ApplyMany(lo.Map(applies, ...), ...); s.Service.Update(ctx, view.Subscription.NamespacedID, spec)`)
**MapSubscriptionErrors after spec operations** — After NewSpecFromPlan or spec.ApplyMany, pipe errors through subscriptionworkflow.MapSubscriptionErrors(err) before returning to translate domain spec errors into correct generic types. (`if err := subscriptionworkflow.MapSubscriptionErrors(err); err != nil { return def, fmt.Errorf("failed to create spec from plan: %w", err) }`)
**Domain errors via models.NewGeneric* constructors** — Validation -> NewGenericValidationError; conflict -> NewGenericConflictError; forbidden transition -> NewGenericForbiddenError; pre-condition -> NewGenericPreConditionFailedError. Never raw fmt.Errorf for these. (`return def, models.NewGenericConflictError(fmt.Errorf("subscription already has that addon purchased"))`)
**Feature flag gate before entering transaction** — Flags (e.g. subscription.MultiSubscriptionEnabledFF) are checked via s.FeatureFlags.IsFeatureEnabled and short-circuit with a domain error before the transaction to avoid unnecessary locking. (`multiSubscriptionEnabled, err := s.FeatureFlags.IsFeatureEnabled(ctx, subscription.MultiSubscriptionEnabledFF)`)
**Compile-time interface assertion** — var _ subscriptionworkflow.Service = &service{} in service.go enforces full implementation. Must be present for any new implementation. (`var _ subscriptionworkflow.Service = &service{}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | WorkflowServiceConfig (all injected deps) and the service struct. NewWorkflowService is the only constructor — wire via app/common. lockCustomer helper lives here. | New deps go into WorkflowServiceConfig, never separate struct fields. lockCustomer calls LockForTX which requires an active transaction already in ctx. |
| `subscription.go` | CreateFromPlan, EditRunning, ChangeToPlan, Restore — cross-subscription lifecycle ops. | EditRunning blocks if the subscription has active addons. ChangeToPlan cancels old then creates new in one transaction with previous/superseding annotations. Restore is gated behind MultiSubscriptionEnabledFF. Call spec.ValidateAlignment() after every ApplyMany sequence. |
| `addon.go` | AddAddon and ChangeAddonQuantity: validate -> transaction -> fetch state -> mutate addon record -> syncWithAddons -> return view. syncWithAddons is the core diff-apply loop. | Duplicate addon purchase detected by ID equality inside the transaction — keep it there. asDiffs filters nil Diffable; len mismatch between diffs and addons is a hard error surfaced as a generic error. |
| `addon_test.go` | Integration tests for AddAddon/ChangeAddonQuantity via subscriptiontestutils.SetupDBDeps + NewService. | clock.FreezeTime(now.Add(time.Millisecond)) avoids Postgres timestamp truncation boundary issues — replicate in new tests. |
| `subscription_test.go` | Integration tests for CreateFromPlan/EditRunning/TestEditingCurrentPhase; also demonstrates MockService injection for unit-testing delegation. | Use the MockService injection pattern (NewWorkflowService with custom UpdateFn/GetViewFn) to unit-test delegation in isolation. |

## Anti-Patterns

- Calling s.Service.Update directly without the restore+apply diff cycle when addons are present — always use syncWithAddons.
- Starting a new outer transaction.Run inside syncWithAddons when already inside an outer transaction.Run — do not bypass the existing transaction.
- Returning raw fmt.Errorf for validation/conflict/forbidden — use models.NewGeneric* constructors.
- Skipping lockCustomer in new mutating workflows for a customer — concurrent mutations race.
- Editing subscription spec fields without calling spec.ValidateAlignment() after applying patches.

## Decisions

- **Workflow service is a separate layer from subscription.Service.** — Keeps the core service persistence-focused; the workflow layer composes addon, locking, and feature-flag concerns without polluting the domain interface or creating circular imports back into app/common.
- **Addon sync uses a full restore-then-apply diff cycle, not incremental patching.** — Ensures idempotency — the spec always reflects exactly the currently-attached addons, preventing double-application on retry.
- **ChangeToPlan cancels the old then creates a new subscription in one transaction with cross-reference annotations.** — Preserves navigable history (previousSubscriptionID/supersedingSubscriptionID) while guaranteeing atomicity — never a window where neither is active.

## Example: Add an addon: validate, lock in transaction, detect duplicate, create record, sync spec

```
func (s *service) AddAddon(ctx context.Context, subscriptionID models.NamespacedID, addonInp subscriptionworkflow.AddAddonWorkflowInput) (subscription.SubscriptionView, subscriptionaddon.SubscriptionAddon, error) {
  if err := addonInp.Validate(); err != nil { return def1, def2, models.NewGenericValidationError(err) }
  res, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (purchaseRes, error) {
    subView, _ := s.Service.GetView(ctx, subscriptionID)
    subsAdds, _ := s.AddonService.List(ctx, subscriptionID.Namespace, ...)
    if lo.SomeBy(subsAdds.Items, func(sa subscriptionaddon.SubscriptionAddon) bool { return sa.Addon.ID == addonInp.AddonID }) {
      return purchaseRes{}, models.NewGenericConflictError(fmt.Errorf("subscription already has that addon purchased"))
    }
    // ... create addon record, then syncWithAddons ...
  })
}
```

<!-- archie:ai-end -->
