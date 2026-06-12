# entitlement

<!-- archie:ai-start -->

> Bridges subscription items to the entitlement domain. Implements subscription.EntitlementAdapter so the subscription service can schedule, fetch, and delete entitlements tied to subscription items without depending on entitlement internals.

## Patterns

**Implements subscription.EntitlementAdapter** — EntitlementSubscriptionAdapter satisfies the subscription.EntitlementAdapter interface; the compile-time assertion guards it. (`var _ subscription.EntitlementAdapter = &EntitlementSubscriptionAdapter{}`)
**Constructor-injected dependencies** — NewSubscriptionEntitlementAdapter takes entitlement.Service, subscription.SubscriptionItemRepository, and transaction.Creator. No slog.Default or hidden globals. (`NewSubscriptionEntitlementAdapter(entitlementConnector, itemRepo, txCreator)`)
**Wrap mutations in transaction.Run** — ScheduleEntitlement runs inside transaction.Run(ctx, a.txCreator, ...) so entitlement creation participates in the surrounding subscription transaction. (`return transaction.Run(ctx, a.txCreator, func(ctx context.Context) (*subscription.SubscriptionEntitlement, error) { ... })`)
**Stamp SubscriptionManaged + merged annotations** — On schedule, set CreateEntitlementInputs.SubscriptionManaged = true and copy the passed annotations (e.g. AnnotationSubscriptionID) into CreateEntitlementInputs.Annotations. (`input.CreateEntitlementInputs.SubscriptionManaged = true`)
**Derive cadence from entitlement ActiveFrom/ActiveTo** — Build SubscriptionEntitlement.Cadence from the entitlement's ActiveFrom/ActiveTo; error if ActiveFrom is nil (entitlement lacks cadence). (`Cadence: models.CadencedModel{ActiveFrom: *ent.ActiveFrom, ActiveTo: ent.ActiveTo}`)
**Typed NotFoundError** — DeleteByItemID returns *NotFoundError{ItemID} when the item has no EntitlementID; errors carry item/namespace/at context. (`return &NotFoundError{ItemID: id}`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | EntitlementSubscriptionAdapter implementing ScheduleEntitlement, GetForSubscriptionsAt, DeleteByItemID | GetForSubscriptionsAt filters items to those with non-nil EntitlementID and fetches via ListEntitlementsWithCustomer with a zero-value Page so all entitlements are returned; do not introduce pagination here. DeleteByItemID deletes the entitlement at clock.Now(). |
| `errors.go` | NotFoundError type for missing subscription entitlements | Error() conditionally appends item/namespace/at; keep the struct fields optional-safe (zero values produce a shorter message). |

## Anti-Patterns

- Calling entitlementConnector.ScheduleEntitlement outside transaction.Run, breaking transactional consistency with the item create.
- Forgetting to set SubscriptionManaged = true or to merge subscription annotations into CreateEntitlementInputs.Annotations.
- Returning a SubscriptionEntitlement when entitlement.ActiveFrom is nil instead of erroring (the cadence would be invalid).

## Decisions

- **Subscription depends on an EntitlementAdapter interface rather than entitlement.Service directly** — Keeps the subscription domain decoupled from entitlement internals and lets the adapter own the SubscriptionManaged/annotation stamping and cadence translation.

<!-- archie:ai-end -->
