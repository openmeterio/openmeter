# entitlement

<!-- archie:ai-start -->

> Bridge adapter between the subscription and entitlement domains. Implements subscription.EntitlementAdapter (ScheduleEntitlement, GetForSubscriptionsAt, DeleteByItemID) so subscription items can schedule, query, and delete entitlements without subscription/service importing entitlement.Service directly.

## Patterns

**Interface compliance assertion** — Assert the target interface with a blank var _ at file top. (`var _ subscription.EntitlementAdapter = &EntitlementSubscriptionAdapter{}`)
**Transaction wrapping via transaction.Run** — Every write runs in transaction.Run(ctx, a.txCreator, ...) so it rebinds to any tx already in ctx. (`return transaction.Run(ctx, a.txCreator, func(ctx context.Context) (*subscription.SubscriptionEntitlement, error) { ... })`)
**SubscriptionManaged + annotation merge** — ScheduleEntitlement sets CreateEntitlementInputs.SubscriptionManaged = true and merges caller annotations before delegating. (`input.CreateEntitlementInputs.SubscriptionManaged = true`)
**Bulk fetch with zero-value pagination** — GetForSubscriptionsAt lists entitlements with pagination.Page{} (zero value) to fetch all; never pass a filtered page. (`Page: pagination.Page{}, // zero value so all entitlements are fetched`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Concrete subscription.EntitlementAdapter: ScheduleEntitlement, GetForSubscriptionsAt, DeleteByItemID. | DeleteByItemID reads itemRepo.GetByID first to find the EntitlementID — the item must exist and have a non-nil EntitlementID. |
| `errors.go` | NotFoundError for a missing entitlement on a subscription item. | At field is optional; check IsZero() before rendering in messages. |

## Anti-Patterns

- Calling entitlement.Service directly from subscription service code instead of via EntitlementSubscriptionAdapter.
- Omitting transaction.Run around entitlement writes — falls off the caller's tx.
- Setting SubscriptionManaged = false on subscription-owned entitlements.
- Passing a non-zero pagination.Page to GetForSubscriptionsAt.

## Decisions

- **Separate package for entitlement bridging rather than inline in subscription/service.** — Keeps subscription.Service free of direct entitlement.Service imports and makes the boundary independently testable/mockable.

## Example: Schedule an entitlement for a new subscription item inside a transaction

```
return transaction.Run(ctx, a.txCreator, func(ctx context.Context) (*subscription.SubscriptionEntitlement, error) {
  input.CreateEntitlementInputs.SubscriptionManaged = true
  if input.CreateEntitlementInputs.Annotations == nil {
    input.CreateEntitlementInputs.Annotations = models.Annotations{}
  }
  for k, v := range annotations { input.CreateEntitlementInputs.Annotations[k] = v }
  ent, err := a.entitlementConnector.ScheduleEntitlement(ctx, input.CreateEntitlementInputs)
  // ...
})
```

<!-- archie:ai-end -->
