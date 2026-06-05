# service

<!-- archie:ai-start -->

> Concrete implementation of subscriptionworkflow.Service — the orchestration layer that composes subscription.Service, subscriptionaddon.Service, and customer.Service into multi-step workflows (create-from-plan, edit, change-plan, restore, add/change addon). Every workflow wraps its steps in a single transaction.Run and operates on subscription specs/views rather than raw DB rows.

## Patterns

**Single transaction per workflow** — Every exported workflow method body is wrapped in transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (...) {...}) so all sub-service calls commit or roll back atomically. Nested helpers (syncWithAddons) also open their own transaction.Run. (`transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (subscription.SubscriptionView, error) { ... })`)
**Spec-then-update flow** — Workflows fetch a SubscriptionView via s.Service.GetView, derive a spec with view.AsSpec(), mutate the spec (NewSpecFromPlan, spec.ApplyMany, addon diffs), validate with spec.ValidateAlignment(), then persist via s.Service.Update / s.Service.Create, and re-fetch with GetView to return the fresh view. (`spec := curr.AsSpec(); spec.ApplyMany(...); s.Service.Update(ctx, id, spec); return s.Service.GetView(ctx, sub.NamespacedID)`)
**Timing validate + resolve before mutation** — Before applying changes, validate timing against the action (inp.Timing.ValidateForAction(subscription.SubscriptionActionCreate/Update/ChangeAddons, &view)) then resolve the edit time (Timing.Resolve / ResolveForSpec). Wrap timing failures in models.NewGenericValidationError. (`if err := inp.Timing.ValidateForAction(subscription.SubscriptionActionCreate, nil); err != nil { ... }; activeFrom, err := inp.Timing.Resolve()`)
**Domain typed errors via models constructors** — Return models.NewGenericValidationError, NewGenericConflictError, NewGenericForbiddenError, NewGenericPreConditionFailedError rather than bare errors for client-facing failures; wrap subscription-package errors with subscriptionworkflow.MapSubscriptionErrors. (`return def, models.NewGenericConflictError(fmt.Errorf("subscription already has that addon purchased"))`)
**Addon sync via diff apply/restore** — Addon mutations call syncWithAddons(view, before, after, time): build addondiff.Diffable for before/after with asDiffs (GetDiffableFromAddon), spec.ApplyMany(restores.GetRestores()) then spec.ApplyMany(applies.GetApplies()), then Update. Adding/editing addons goes through s.AddonService, never raw item patches. (`spec.ApplyMany(lo.Map(applies, func(d addondiff.Diffable, _ int) subscription.AppliesToSpec { return d.GetApplies() }), subscription.ApplyContext{CurrentTime: currentTime})`)
**Customer locking for create** — CreateFromPlan calls s.lockCustomer(ctx, inp.CustomerID) (subscription.GetCustomerLock + s.Lockr.LockForTX) as the first step inside the transaction to serialize concurrent subscription creation per customer. (`if err := s.lockCustomer(ctx, inp.CustomerID); err != nil { return def, err }`)
**Owner-subsystem annotation stamping** — EditRunning rewrites patch.PatchAddItem patches (both value and pointer forms) to set OwnerSubscriptionSubSystem via subscription.AnnotationParser.AddOwnerSubSystem and a unique patch ID via subscriptionworkflow.AnnotationParser.SetUniquePatchID before applying. (`subscription.AnnotationParser.AddOwnerSubSystem(ap.CreateInput.CreateSubscriptionItemInput.Annotations, subscription.OwnerSubscriptionSubSystem)`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `service.go` | Defines WorkflowServiceConfig (all injected deps: Service, AddonService, CustomerService, TransactionManager, Logger, Lockr, FeatureFlags), the service struct, NewWorkflowService constructor, the compile-time interface assertion, and lockCustomer helper. | service embeds WorkflowServiceConfig directly — add new dependencies as config fields, not as struct-only fields. Keep `var _ subscriptionworkflow.Service = &service{}` satisfied. |
| `subscription.go` | Core lifecycle workflows: CreateFromPlan, EditRunning, ChangeToPlan, Restore. Houses spec construction (NewSpecFromPlan), addon-guard (hasAddons blocks editing subs with addons), and previous/superseding subscription annotation wiring for plan changes. | ChangeToPlan is cancel-then-create: it pins the new sub's Timing to curr.ActiveTo (verbatim) so both resolve to the exact same timestamp. Restore is gated by the MultiSubscriptionEnabledFF feature flag and deletes scheduled subs before Continue. |
| `addon.go` | Addon workflows: AddAddon, ChangeAddonQuantity, and the shared syncWithAddons. Helpers asDiffs (subsAddon -> addondiff.Diffable) and hasAddons. purchaseRes is the transaction return struct. | syncWithAddons restores BEFORE applying (order matters). Conflict check uses lo.SomeBy over existing addons. Namespace cross-checks (subscriptionID.Namespace vs SubscriptionAddonID.Namespace) must precede the addon fetch. The logErrWithArgs JSON-dump block is a temporary debugging aid marked TODO. |

## Anti-Patterns

- Calling s.Service / s.AddonService adapters or persisting outside a transaction.Run wrapper — breaks atomicity of multi-step workflows.
- Mutating a SubscriptionView's items directly instead of going through view.AsSpec(), spec mutation, and s.Service.Update.
- Editing a subscription that has addons via EditRunning — hasAddons must reject it with NewGenericForbiddenError.
- Applying timing without ValidateForAction + Resolve/ResolveForSpec, or returning raw errors instead of models.NewGeneric*Error for client-facing failures.
- Bypassing s.AddonService for addon changes and hand-patching addon items, instead of building addondiff.Diffable via asDiffs/syncWithAddons.

## Decisions

- **Workflows are spec-centric: read view, derive spec, mutate, validate alignment, update, re-read view.** — The subscription spec is the single declarative source of truth; applying patches/diffs to a spec and re-validating alignment keeps phase/item invariants consistent before any DB write.
- **Plan changes are implemented as cancel-current + create-new with linked annotations rather than in-place mutation.** — Subscriptions are immutable over their cadence; superseding/previous-subscription annotations preserve the audit chain while the new sub starts exactly at the old sub's ActiveTo.
- **Addon effects are computed as reversible diffs (GetRestores/GetApplies) and re-synced from the full before/after addon set on every change.** — Recomputing the spec from the base view plus the current full addon set avoids drift and makes add/change/remove a single idempotent reconciliation path.

## Example: Transactional, timing-validated addon sync inside a workflow method

```
res, err := transaction.Run(ctx, s.TransactionManager, func(ctx context.Context) (purchaseRes, error) {
    subView, err := s.Service.GetView(ctx, subscriptionID)
    if err != nil { return purchaseRes{}, err }
    if err := addonInp.Timing.ValidateForAction(subscription.SubscriptionActionChangeAddons, &subView); err != nil {
        return purchaseRes{}, models.NewGenericValidationError(err)
    }
    editTime, err := addonInp.Timing.ResolveForSpec(subView.AsSpec())
    if err != nil { return purchaseRes{}, err }
    subsAdd, err := s.AddonService.Create(ctx, subscriptionID.Namespace, subscriptionaddon.CreateSubscriptionAddonInput{
        AddonID: addonInp.AddonID, SubscriptionID: subscriptionID.ID,
        InitialQuantity: subscriptionaddon.CreateSubscriptionAddonQuantityInput{ActiveFrom: editTime, Quantity: addonInp.InitialQuantity},
    })
    if err != nil { return purchaseRes{}, err }
    subView, err = s.syncWithAddons(ctx, subView, subsAdds.Items, append(subsAdds.Items, *subsAdd), editTime)
    return purchaseRes{sub: subView, subAdd: *subsAdd}, err
// ...
```

<!-- archie:ai-end -->
