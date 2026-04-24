# entitlement

<!-- archie:ai-start -->

> Bridge adapter between the subscription domain and the entitlement domain. Implements subscription.EntitlementAdapter to schedule, query, and delete entitlements on behalf of subscription items, wrapping every write in a transaction.Creator.

## Patterns

**Interface compliance assertion** — Every exported struct asserts the target interface with a blank var _ check at file top. (`var _ subscription.EntitlementAdapter = &EntitlementSubscriptionAdapter{}`)
**Transaction wrapping via transaction.Run** — All writes call transaction.Run(ctx, a.txCreator, ...) so they rebind to any tx already in ctx. (`return transaction.Run(ctx, a.txCreator, func(ctx context.Context) (*subscription.SubscriptionEntitlement, error) { ... })`)
**Annotation merging on schedule** — ScheduleEntitlement merges caller-supplied annotations into CreateEntitlementInputs.Annotations before delegating to entitlement.Service.ScheduleEntitlement. (`for k, v := range annotations { input.CreateEntitlementInputs.Annotations[k] = v }`)
**SubscriptionManaged flag** — Sets CreateEntitlementInputs.SubscriptionManaged = true so the entitlement layer knows it is lifecycle-managed by the subscription. (`input.CreateEntitlementInputs.SubscriptionManaged = true`)

## Key Files

| File | Role | Watch For |
|------|------|-----------|
| `adapter.go` | Concrete implementation of subscription.EntitlementAdapter: ScheduleEntitlement, GetForSubscriptionsAt, DeleteByItemID. | GetForSubscriptionsAt issues a bulk entitlement list with zero-value pagination.Page to fetch all; callers must not pass filtered pages. |
| `errors.go` | Defines NotFoundError for missing entitlement on a subscription item. | At field on NotFoundError is optional; check for IsZero before rendering. |

## Anti-Patterns

- Calling entitlement.Service directly from subscription service code instead of going through EntitlementSubscriptionAdapter.
- Omitting transaction.Run around entitlement writes — falls off the caller's tx.
- Setting SubscriptionManaged = false on subscription-owned entitlements.

## Decisions

- **Separate package for entitlement bridging rather than inline in subscription/service.** — Keeps the subscription.Service free of direct entitlement.Service imports; the adapter interface boundary allows independent testing and mocking.

## Example: Schedule an entitlement for a new subscription item inside a transaction

```
import (
	"github.com/openmeterio/openmeter/openmeter/entitlement"
	"github.com/openmeterio/openmeter/openmeter/subscription"
	"github.com/openmeterio/openmeter/pkg/framework/transaction"
)

return transaction.Run(ctx, a.txCreator, func(ctx context.Context) (*subscription.SubscriptionEntitlement, error) {
	input.CreateEntitlementInputs.SubscriptionManaged = true
	if input.CreateEntitlementInputs.Annotations == nil {
		input.CreateEntitlementInputs.Annotations = models.Annotations{}
	}
	for k, v := range annotations {
		input.CreateEntitlementInputs.Annotations[k] = v
	}
	ent, err := a.entitlementConnector.ScheduleEntitlement(ctx, input.CreateEntitlementInputs)
// ...
```

<!-- archie:ai-end -->
